package service

import (
	"fmt"
	"strings"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/userclient"
)

// ──────────────────────────────────────────
// Shared slice/CSV utilities
// ──────────────────────────────────────────

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// groupResourceMeta bucketises rows from galgame_resource into per-galgame
// platform/language sets (preserving insertion order + dedup).
func groupResourceMeta(rows []model.GalgameResourceMeta) (platforms, languages map[int][]string) {
	platforms = make(map[int][]string)
	languages = make(map[int][]string)
	for _, r := range rows {
		if r.Platform != "" {
			platforms[r.GalgameID] = appendUniqueStr(platforms[r.GalgameID], r.Platform)
		}
		if r.Language != "" {
			languages[r.GalgameID] = appendUniqueStr(languages[r.GalgameID], r.Language)
		}
	}
	return
}

// ──────────────────────────────────────────
// Galgame → Detail DTO
// ──────────────────────────────────────────

// galgameDetailFromNextMoe maps a galgame galgame payload into the response DTO,
// resolving author/contributor users from the galgame-returned users map.
func galgameDetailFromNextMoe(g dto.NextMoeGalgameDetailFull, users map[string]dto.NextMoeUser) dto.GalgameDetail {
	return dto.GalgameDetail{
		ID:     g.ID,
		VndbID: g.VndbID,
		User:   lookupNextMoeUser(users, g.UserID),
		Name: dto.KunLanguage{
			EnUs: g.NameEnUs, JaJp: g.NameJaJp,
			ZhCn: g.NameZhCn, ZhTw: g.NameZhTw,
		},
		Banner: g.Banner,
		Introduction: dto.KunLanguage{
			EnUs: markdown.Render(g.IntroEnUs),
			JaJp: markdown.Render(g.IntroJaJp),
			ZhCn: markdown.Render(g.IntroZhCn),
			ZhTw: markdown.Render(g.IntroZhTw),
		},
		Markdown: dto.KunLanguage{
			EnUs: g.IntroEnUs, JaJp: g.IntroJaJp,
			ZhCn: g.IntroZhCn, ZhTw: g.IntroZhTw,
		},
		ContentLimit:       g.ContentLimit,
		ResourceUpdateTime: g.ResourceUpdateTime,
		Status:             g.Status,
		OriginalLanguage:   g.OriginalLanguage,
		AgeLimit:           g.AgeLimit,
		ReleaseDate:        g.ReleaseDate,
		ReleaseDateTBA:     g.ReleaseDateTBA,
		// U2: effective_banner_hash + Covers/Screenshots are the
		// canonical banner/gallery sources. CDN URLs (effective_banner_url
		// + per-row cdn_url) are injected by client.rewriteBanners over
		// the galgame response BEFORE we unmarshal — and we explicitly
		// declare the fields on NextMoeGalgameDetailFull so they survive,
		// then pipe through here. banner_image_hash retired in galgame
		// PR5 (K-PR6).
		EffectiveBannerHash:      g.EffectiveBannerHash,
		EffectiveBannerURL:       g.EffectiveBannerURL,
		EffectiveBannerWidth:     g.EffectiveBannerWidth,
		EffectiveBannerHeight:    g.EffectiveBannerHeight,
		EffectiveBannerThumbhash: g.EffectiveBannerThumbhash,
		Covers:                   coversFromNextMoe(g.Covers),
		Screenshots:              screenshotsFromNextMoe(g.Screenshots),
		Contributor:              contributorsFromNextMoe(g.Contributor, users),
		Alias:                    nextMoeAliasesToNames(g.Alias),
		Engine:                   enginesFromNextMoe(g.Engine),
		Official:                 officialsFromNextMoe(g.Official),
		Tag:                      tagsFromNextMoe(g.Tag),
		Created:                  g.Created,
		Updated:                  g.Updated,
	}
}

func lookupNextMoeUser(users map[string]dto.NextMoeUser, userID int) dto.UserBrief {
	if u, ok := users[fmt.Sprintf("%d", userID)]; ok {
		return dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
	}
	return dto.UserBrief{ID: userID}
}

// U2: cover/screenshot row mappers. Plain field-by-field copy — wire
// shape is identical (snake_case JSON tags); the wrappers exist so the
// frontend-exposed types can later diverge (e.g. omit Source/SourceKey
// from public responses) without rewriting the mapper site.
func coversFromNextMoe(rows []dto.NextMoeGalgameCover) []dto.GalgameCover {
	out := make([]dto.GalgameCover, len(rows))
	for i, r := range rows {
		out[i] = dto.GalgameCover{
			ImageHash: r.ImageHash, SortOrder: r.SortOrder,
			Sexual: r.Sexual, Violence: r.Violence,
			Source: r.Source, SourceKey: r.SourceKey,
			Kind:      r.Kind,
			CDNURL:    r.CDNURL,
			Width:     r.Width,
			Height:    r.Height,
			Thumbhash: r.Thumbhash,
		}
	}
	return out
}

func screenshotsFromNextMoe(rows []dto.NextMoeGalgameScreenshot) []dto.GalgameScreenshot {
	out := make([]dto.GalgameScreenshot, len(rows))
	for i, r := range rows {
		out[i] = dto.GalgameScreenshot{
			ImageHash: r.ImageHash, SortOrder: r.SortOrder, Caption: r.Caption,
			Sexual: r.Sexual, Violence: r.Violence,
			Source: r.Source, SourceKey: r.SourceKey,
			CDNURL:    r.CDNURL,
			Width:     r.Width,
			Height:    r.Height,
			Thumbhash: r.Thumbhash,
		}
	}
	return out
}

func contributorsFromNextMoe(contribs []dto.NextMoeContributor, users map[string]dto.NextMoeUser) []dto.UserBrief {
	out := make([]dto.UserBrief, len(contribs))
	for i, c := range contribs {
		out[i] = lookupNextMoeUser(users, c.UserID)
	}
	return out
}

func nextMoeAliasesToNames(aliases []dto.NextMoeAlias) []string {
	out := make([]string, len(aliases))
	for i, a := range aliases {
		out[i] = a.Name
	}
	return out
}

func enginesFromNextMoe(engines []dto.NextMoeEngineWithAlias) []dto.GalgameDetailEngine {
	out := make([]dto.GalgameDetailEngine, len(engines))
	for i, e := range engines {
		alias := e.Engine.Alias
		if alias == nil {
			alias = []string{}
		}
		out[i] = dto.GalgameDetailEngine{
			ID:           e.Engine.ID,
			Name:         e.Engine.Name,
			Alias:        alias,
			GalgameCount: e.Engine.GalgameCount,
		}
	}
	return out
}

func officialsFromNextMoe(rels []dto.NextMoeOfficialRel) []dto.GalgameDetailOfficial {
	out := make([]dto.GalgameDetailOfficial, len(rels))
	for i, rel := range rels {
		out[i] = dto.GalgameDetailOfficial{
			ID:           rel.Official.ID,
			Name:         rel.Official.Name,
			Link:         rel.Official.Link,
			Category:     rel.Official.Category,
			Lang:         rel.Official.Lang,
			Alias:        nextMoeAliasesToNames(rel.Official.Alias),
			GalgameCount: rel.Official.GalgameCount,
		}
	}
	return out
}

func tagsFromNextMoe(tags []dto.NextMoeTagWithSpoiler) []dto.GalgameDetailTag {
	out := make([]dto.GalgameDetailTag, len(tags))
	for i, t := range tags {
		out[i] = dto.GalgameDetailTag{
			ID:           t.Tag.ID,
			Name:         t.Tag.Name,
			Category:     t.Tag.Category,
			SpoilerLevel: t.SpoilerLevel,
			GalgameCount: t.Tag.GalgameCount,
		}
	}
	return out
}

// detailRatingFromRow maps a DB rating row into the detail-page rating card.
func detailRatingFromRow(
	r repository.GalgameDetailRatingRow,
	user userclient.User,
	isLiked bool,
	galgameID int,
	g dto.NextMoeGalgameDetailFull,
) dto.GalgameDetailRating {
	return dto.GalgameDetailRating{
		ID:           r.ID,
		User:         userBriefToDTO(user),
		Recommend:    r.Recommend,
		Overall:      r.Overall,
		View:         r.View,
		GalgameType:  rawJSON(r.GalgameType),
		PlayStatus:   r.PlayStatus,
		ShortSummary: r.ShortSummary,
		SpoilerLevel: r.SpoilerLevel,
		Art:          r.Art,
		Story:        r.Story,
		Music:        r.Music,
		Character:    r.Character,
		Route:        r.Route,
		System:       r.System,
		Voice:        r.Voice,
		ReplayValue:  r.ReplayValue,
		LikeCount:    r.LikeCount,
		IsLiked:      isLiked,
		GalgameID:    galgameID,
		Created:      r.Created,
		Updated:      r.Updated,
		Galgame: dto.GalgameDetailRatingGalgame{
			ID:           g.ID,
			ContentLimit: g.ContentLimit,
			Name: dto.KunLanguage{
				EnUs: g.NameEnUs, JaJp: g.NameJaJp,
				ZhCn: g.NameZhCn, ZhTw: g.NameZhTw,
			},
		},
	}
}
