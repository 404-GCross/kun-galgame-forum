package service

import (
	"context"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/userclient"
)

// GalgameEnricher turns wiki galgame items into the enriched GalgameCard
// shape the frontend consumes, applying NSFW filtering and fusing in local
// interaction counts + user info.
//
// This is the single source of truth for "wiki galgame + local enrichment"
// across series / official / engine / tag detail endpoints.
type GalgameEnricher struct {
	galgameRepo *repository.GalgameRepository
	metaRepo    *repository.GalgameResourceMetaRepository
	userClient  *userclient.Client
}

func NewGalgameEnricher(
	galgameRepo *repository.GalgameRepository,
	metaRepo *repository.GalgameResourceMetaRepository,
	userClient *userclient.Client,
) *GalgameEnricher {
	return &GalgameEnricher{galgameRepo: galgameRepo, metaRepo: metaRepo, userClient: userClient}
}

// FilterSFW removes NSFW items when the caller requests SFW-only content.
func (e *GalgameEnricher) FilterSFW(items []dto.WikiGalgameItem, isSFW bool) []dto.WikiGalgameItem {
	if !isSFW {
		return items
	}
	out := make([]dto.WikiGalgameItem, 0, len(items))
	for _, g := range items {
		if g.ContentLimit == "sfw" {
			out = append(out, g)
		}
	}
	return out
}

// HasNSFW reports whether any item in the list is nsfw.
func (e *GalgameEnricher) HasNSFW(items []dto.WikiGalgameItem) bool {
	for _, g := range items {
		if g.ContentLimit == "nsfw" {
			return true
		}
	}
	return false
}

// Samples returns up to `n` minimal samples (name + banner).
func (e *GalgameEnricher) Samples(items []dto.WikiGalgameItem, n int) []dto.GalgameSample {
	if n > len(items) {
		n = len(items)
	}
	out := make([]dto.GalgameSample, 0, n)
	for i := 0; i < n; i++ {
		g := items[i]
		out = append(out, dto.GalgameSample{
			Name: dto.KunLanguage{
				EnUs: g.NameEnUs, JaJp: g.NameJaJp,
				ZhCn: g.NameZhCn, ZhTw: g.NameZhTw,
			},
			Banner:                   g.Banner,
			EffectiveBannerHash:      g.EffectiveBannerHash,
			EffectiveBannerURL:       g.EffectiveBannerURL,
			EffectiveBannerWidth:     g.EffectiveBannerWidth,
			EffectiveBannerHeight:    g.EffectiveBannerHeight,
			EffectiveBannerThumbhash: g.EffectiveBannerThumbhash,
		})
	}
	return out
}

// ToCards converts wiki galgame items into enriched GalgameCard DTOs, batch-
// loading users (from OAuth) and local stats once.
func (e *GalgameEnricher) ToCards(ctx context.Context, items []dto.WikiGalgameItem) []dto.GalgameCard {
	if len(items) == 0 {
		return []dto.GalgameCard{}
	}

	galgameIDs := make([]int, len(items))
	userIDs := make([]int, len(items))
	for i, g := range items {
		galgameIDs[i] = g.ID
		userIDs[i] = g.UserID
	}

	userMap := e.userClient.Hydrate(ctx, userIDs)
	localMap := e.galgameRepo.FindLocalBatch(galgameIDs)
	// Platform/language badges come from the LOCAL galgame_resource rows — the
	// wiki item (incl. Meilisearch search hits) carries neither — so the search /
	// series / official / engine / tag cards match the /galgame list card's
	// top-left platform badges. Cheap now that galgame_resource(galgame_id) is indexed.
	platformMap, languageMap := groupResourceMeta(e.metaRepo.FindResourceMetaBatch(galgameIDs))

	cards := make([]dto.GalgameCard, len(items))
	for i, g := range items {
		// Present in FindLocalBatch ⇒ the forum has a local row for this game
		// (created/claimed/approved or has activity). Wiki-only games are absent
		// ⇒ IsOnForum=false and their view/like/platform/language stay zero/empty
		// (the frontend hides those rather than showing misleading zeros).
		_, onForum := localMap[g.ID]
		cards[i] = dto.GalgameCard{
			ID: g.ID,
			Name: dto.KunLanguage{
				EnUs: g.NameEnUs, JaJp: g.NameJaJp,
				ZhCn: g.NameZhCn, ZhTw: g.NameZhTw,
			},
			Banner:       g.Banner,
			User:         userBriefToDTO(userMap[g.UserID]),
			ContentLimit: g.ContentLimit,
			// View is a kungal-local stat (each site has its own audience),
			// not metadata; pull from the local stats row instead of wiki.
			View:               localMap[g.ID].View,
			LikeCount:          localMap[g.ID].LikeCount,
			ResourceUpdateTime: g.ResourceUpdateTime,
			ReleaseDate:        g.ReleaseDate,
			ReleaseDateTBA:     g.ReleaseDateTBA,
			// U2: card carries only the derived banner; cdn_url/
			// effective_banner_url is injected by client.rewriteBanners
			// walker. banner_image_hash retired in wiki PR5 (K-PR6).
			EffectiveBannerHash:      g.EffectiveBannerHash,
			EffectiveBannerURL:       g.EffectiveBannerURL,
			EffectiveBannerWidth:     g.EffectiveBannerWidth,
			EffectiveBannerHeight:    g.EffectiveBannerHeight,
			EffectiveBannerThumbhash: g.EffectiveBannerThumbhash,
			Platform:                 emptyStrSliceIfNil(platformMap[g.ID]),
			Language:                 emptyStrSliceIfNil(languageMap[g.ID]),
			IsOnForum:                onForum,
		}
	}
	return cards
}
