package service

import (
	"context"
	"encoding/json"

	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
)

type UserContentService struct {
	userContentRepo *repository.UserContentRepository
	wikiClient      *galgameClient.GalgameClient
	userClient      *userclient.Client
}

func NewUserContentService(
	userContentRepo *repository.UserContentRepository,
	wikiClient *galgameClient.GalgameClient,
	userClient *userclient.Client,
) *UserContentService {
	return &UserContentService{
		userContentRepo: userContentRepo,
		wikiClient:      wikiClient,
		userClient:      userClient,
	}
}

// hideTarget reports whether the profile owner (userID) is banned. When true,
// every content tab must render nothing: the profile page itself already
// degrades to a stub (user_service.GetUserProfile), but the content-list
// endpoints are separate routes with no shared gate — and topics/replies/
// comments don't hydrate identity, so a per-row filter can't cover them.
// Fails open (a transient OAuth error yields a zero-Status renderable user)
// so an OAuth outage doesn't blank every profile.
func (s *UserContentService) hideTarget(ctx context.Context, userID int) bool {
	u, _, _ := s.userClient.User(ctx, userID)
	return !userclient.IsRenderable(u)
}

// ──────────────────────────────────────────
// User galgame list — GET /user/:userID/galgames
// ──────────────────────────────────────────

// GetUserGalgameCards returns enriched galgame cards for the user's list
// (created / liked / favorited / commented depending on req.Type).
//
// SFW gating delegated to wiki via content_limit per
// docs/galgame_wiki/00-handbook §16; rows whose galgame is filtered come
// back as "no brief returned" and get dropped below. `total` over-reports
// in SFW mode (it counts kungal-side relation rows pre-filter).
func (s *UserContentService) GetUserGalgameCards(
	ctx context.Context,
	userID int,
	req *dto.UserGalgamesRequest,
	isSFW bool,
) ([]dto.UserGalgameCard, int64, *errors.AppError) {
	if s.hideTarget(ctx, userID) {
		return []dto.UserGalgameCard{}, 0, nil
	}
	// "已发布" (galgame_publish): ownership lives in the wiki — kungal's local
	// galgame mirror has no user_id after the OAuth migration — so the list
	// comes straight from the wiki endpoint (already ordered, paginated and
	// NSFW-filtered there). Other types (like / favorite / comment) still join
	// local relation tables for the IDs, then enrich via the wiki batch.
	if req.Type == "galgame_publish" {
		briefs, total, wikiErr := s.wikiClient.GetUserGalgames(ctx, userID, req.Page, req.Limit, isSFW)
		if wikiErr != nil {
			return []dto.UserGalgameCard{}, 0, nil
		}
		return s.buildGalgameCards(ctx, briefs), total, nil
	}

	// "贡献的" (galgame_contributed): created ∪ edited — also wiki-owned, same
	// wiki-paginated/NSFW-filtered list shape as 已发布, just a different
	// endpoint. Superset of galgame_publish.
	if req.Type == "galgame_contributed" {
		briefs, total, wikiErr := s.wikiClient.GetUserContributedGalgames(ctx, userID, req.Page, req.Limit, isSFW)
		if wikiErr != nil {
			return []dto.UserGalgameCard{}, 0, nil
		}
		return s.buildGalgameCards(ctx, briefs), total, nil
	}

	ids, total, err := s.userContentRepo.FindUserGalgameIDs(userID, req.Type, req.Page, req.Limit, req.ShowNoResource)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取用户 Galgame 列表失败")
	}
	if len(ids) == 0 {
		return []dto.UserGalgameCard{}, total, nil
	}

	briefMap, wikiErr := s.wikiClient.GetBatchPublic(ctx, ids, isSFW)
	if wikiErr != nil {
		// Wiki failure → return empty list but preserve total count.
		return []dto.UserGalgameCard{}, total, nil
	}

	// Preserve the local ordering (FindUserGalgameIDs returns newest-first);
	// drop IDs the wiki filtered out (NSFW miss / deleted).
	briefs := make([]galgameClient.GalgameBrief, 0, len(ids))
	for _, id := range ids {
		if b, ok := briefMap[id]; ok {
			briefs = append(briefs, b)
		}
	}
	return s.buildGalgameCards(ctx, briefs), total, nil
}

// buildGalgameCards turns an ORDERED slice of wiki briefs into profile cards,
// fusing in kungal-local stats (view / like), resource meta (platform /
// language) and author identity. Shared by every galgame tab so the card shape
// stays identical across 已发布 / 点赞 / 收藏 / 评论.
func (s *UserContentService) buildGalgameCards(
	ctx context.Context,
	briefs []galgameClient.GalgameBrief,
) []dto.UserGalgameCard {
	if len(briefs) == 0 {
		return []dto.UserGalgameCard{}
	}

	ids := make([]int, len(briefs))
	for i, b := range briefs {
		ids[i] = b.ID
	}
	localMap := s.userContentRepo.FindGalgameLocalStats(ids)
	metaRows := s.userContentRepo.FindResourceMetaByGalgameIDs(ids)
	platformMap, languageMap := groupResourceMeta(metaRows)

	userIDs := collectUniqueIDs(briefs, func(b galgameClient.GalgameBrief) int { return b.UserID })
	userMap := s.userClient.Hydrate(ctx, userIDs)

	cards := make([]dto.UserGalgameCard, 0, len(briefs))
	for _, b := range briefs {
		l := localMap[b.ID]
		u := userMap[b.UserID]
		cards = append(cards, dto.UserGalgameCard{
			ID:                 b.ID,
			Name:               briefToLocale(b),
			Banner:             b.Banner,
			User:               dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			ContentLimit:       b.ContentLimit,
			View:               l.View,
			LikeCount:          l.LikeCount,
			ResourceUpdateTime: b.ResourceUpdateTime,
			Platform:           emptyStrSlice(platformMap[b.ID]),
			Language:           emptyStrSlice(languageMap[b.ID]),
			ReleaseDate:        b.ReleaseDate,
			ReleaseDateTBA:     b.ReleaseDateTBA,
			// U2: pass through the wiki-derived banner so the FE card
			// can pick `_mini` instead of falling back to empty legacy
			// `banner` for newly-uploaded galgames.
			EffectiveBannerHash: b.EffectiveBannerHash,
			EffectiveBannerURL:  b.EffectiveBannerURL,
		})
	}
	return cards
}

// ──────────────────────────────────────────
// Topics / replies / comments (already thin)
// ──────────────────────────────────────────

func (s *UserContentService) GetUserTopics(ctx context.Context, userID int, req *dto.UserTopicsRequest, isSFW bool) ([]dto.UserTopic, int64, *errors.AppError) {
	if s.hideTarget(ctx, userID) {
		return []dto.UserTopic{}, 0, nil
	}
	items, total, err := s.userContentRepo.FindUserTopics(userID, req.Type, req.Page, req.Limit, isSFW)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取用户话题列表失败")
	}
	return items, total, nil
}

func (s *UserContentService) GetUserReplies(ctx context.Context, userID int, req *dto.UserRepliesRequest, isSFW bool) ([]repository.UserReply, int64, *errors.AppError) {
	if s.hideTarget(ctx, userID) {
		return []repository.UserReply{}, 0, nil
	}
	items, total, err := s.userContentRepo.FindUserReplies(userID, req.Type, req.Page, req.Limit, isSFW)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取用户回复列表失败")
	}
	return items, total, nil
}

func (s *UserContentService) GetUserComments(ctx context.Context, userID int, req *dto.UserCommentsRequest, isSFW bool) ([]repository.UserComment, int64, *errors.AppError) {
	if s.hideTarget(ctx, userID) {
		return []repository.UserComment{}, 0, nil
	}
	items, total, err := s.userContentRepo.FindUserComments(userID, req.Type, req.Page, req.Limit, isSFW)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取用户评论列表失败")
	}
	return items, total, nil
}

// GetUserGalgameComments returns the comment-card data for the
// "评论 / 被评论 / 点赞评论" tabs under /user/:id/galgame/.
// Author identity comes from userclient; content is rendered via the
// project goldmark pipeline so the frontend can drop it into
// <KunContent> consistently with the rest of the site.
func (s *UserContentService) GetUserGalgameComments(
	ctx context.Context,
	userID int,
	req *dto.UserGalgameCommentsRequest,
	isSFW bool,
) ([]dto.UserGalgameComment, int64, *errors.AppError) {
	if s.hideTarget(ctx, userID) {
		return []dto.UserGalgameComment{}, 0, nil
	}
	rows, total, err := s.userContentRepo.FindUserGalgameComments(userID, req.Type, req.Page, req.Limit)
	if err != nil {
		return nil, 0, errors.ErrInternal("获取用户 Galgame 评论列表失败")
	}
	if len(rows) == 0 {
		return []dto.UserGalgameComment{}, total, nil
	}

	uidSet := make(map[int]struct{}, len(rows))
	gidSet := make(map[int]struct{}, len(rows))
	for _, r := range rows {
		uidSet[r.UserID] = struct{}{}
		gidSet[r.GalgameID] = struct{}{}
	}
	uids := make([]int, 0, len(uidSet))
	for id := range uidSet {
		uids = append(uids, id)
	}
	gids := make([]int, 0, len(gidSet))
	for id := range gidSet {
		gids = append(gids, id)
	}
	userMap := s.userClient.Hydrate(ctx, uids)
	// SFW gate via wiki content_limit
	// (docs/galgame_wiki/00-handbook §16). Comments whose galgame is
	// filtered out won't have a brief returned and are dropped here.
	briefMap, _ := s.wikiClient.GetBatchPublic(ctx, gids, isSFW)

	items := make([]dto.UserGalgameComment, 0, len(rows))
	for _, r := range rows {
		if _, ok := briefMap[r.GalgameID]; !ok {
			continue
		}
		u := userMap[r.UserID]
		// Drop comments authored by a banned user — the 被评论 / 点赞评论 tabs
		// list third-party comments (flat list, no nesting).
		if !userclient.IsRenderable(u) {
			continue
		}
		items = append(items, dto.UserGalgameComment{
			ID:          r.ID,
			GalgameID:   r.GalgameID,
			Content:     r.Content,
			ContentHtml: markdown.Render(r.Content),
			User:        dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Created:     r.CreatedAt,
		})
	}
	return items, total, nil
}

// ──────────────────────────────────────────
// Resources — GET /user/:userID/resources
// ──────────────────────────────────────────

func (s *UserContentService) GetUserResources(
	ctx context.Context,
	userID int,
	req *dto.UserResourcesRequest,
	isSFW bool,
) (*dto.UserResourcesResponse, *errors.AppError) {
	if s.hideTarget(ctx, userID) {
		return &dto.UserResourcesResponse{Resources: []dto.UserResourceItem{}, Total: 0}, nil
	}
	rows, total, err := s.userContentRepo.FindUserResources(userID, req.Type, req.Page, req.Limit)
	if err != nil {
		return nil, errors.ErrInternal("获取用户资源列表失败")
	}

	resourceIDs := make([]int, len(rows))
	galgameIDs := collectUniqueIDs(rows, func(r repository.UserResource) int { return r.GalgameID })
	for i, r := range rows {
		resourceIDs[i] = r.ID
	}

	var linkMap map[int][]string
	if len(resourceIDs) > 0 {
		linkMap, _ = s.userContentRepo.FindResourceLinks(resourceIDs)
	}

	// SFW gating via wiki content_limit per
	// docs/galgame_wiki/00-handbook §16. Rows whose galgame is filtered
	// come back as "no brief returned" and get dropped below.
	var briefMap map[int]galgameClient.GalgameBrief
	if len(galgameIDs) > 0 {
		briefMap, _ = s.wikiClient.GetBatchPublic(ctx, galgameIDs, isSFW)
	}

	items := make([]dto.UserResourceItem, 0, len(rows))
	for _, r := range rows {
		b, hasBrief := briefMap[r.GalgameID]
		if !hasBrief {
			continue
		}
		links := linkMap[r.ID]
		if links == nil {
			links = []string{}
		}
		name := briefToLocale(b)
		items = append(items, dto.UserResourceItem{
			ID:          r.ID,
			GalgameID:   r.GalgameID,
			GalgameName: name,
			Type:        r.Type,
			Language:    r.Language,
			Platform:    r.Platform,
			Size:        r.Size,
			Link:        links,
			Code:        r.Code,
			Password:    r.Password,
			Note:        r.Note,
			Status:      r.Status,
			Created:     r.Created,
		})
	}

	return &dto.UserResourcesResponse{Resources: items, Total: total}, nil
}

// ──────────────────────────────────────────
// Ratings — GET /user/:userID/ratings
// ──────────────────────────────────────────

func (s *UserContentService) GetUserRatings(
	ctx context.Context,
	userID int,
	req *dto.UserRatingsRequest,
	isSFW bool,
) (*dto.UserRatingsResponse, *errors.AppError) {
	if s.hideTarget(ctx, userID) {
		return &dto.UserRatingsResponse{RatingData: []dto.UserRatingItem{}, Total: 0}, nil
	}
	rows, total, err := s.userContentRepo.FindUserRatings(userID, req.Page, req.Limit)
	if err != nil {
		return nil, errors.ErrInternal("获取用户评分列表失败")
	}

	galgameIDs := collectUniqueIDs(rows, func(r repository.UserRating) int { return r.GalgameID })
	// SFW gating via wiki content_limit per
	// docs/galgame_wiki/00-handbook §16.
	var briefMap map[int]galgameClient.GalgameBrief
	if len(galgameIDs) > 0 {
		briefMap, _ = s.wikiClient.GetBatchPublic(ctx, galgameIDs, isSFW)
	}

	uids := collectUniqueIDs(rows, func(r repository.UserRating) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	items := make([]dto.UserRatingItem, 0, len(rows))
	for _, r := range rows {
		b, hasBrief := briefMap[r.GalgameID]
		if !hasBrief {
			continue
		}
		var galgameType []string
		if r.GalgameType != "" {
			_ = json.Unmarshal([]byte(r.GalgameType), &galgameType)
		}

		galgame := dto.UserRatingGalgame{ID: r.GalgameID}
		if hasBrief {
			galgame = dto.UserRatingGalgame{
				ID:           b.ID,
				Name:         briefToLocale(b),
				ContentLimit: b.ContentLimit,
			}
		}

		u := userMap[r.UserID]
		items = append(items, dto.UserRatingItem{
			ID:           r.ID,
			User:         dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Recommend:    r.Recommend,
			Overall:      r.Overall,
			View:         r.View,
			GalgameType:  galgameType,
			PlayStatus:   r.PlayStatus,
			ShortSummary: r.ShortSummary,
			Art:          r.Art,
			Story:        r.Story,
			Music:        r.Music,
			Character:    r.Character,
			Route:        r.Route,
			System:       r.System,
			Voice:        r.Voice,
			ReplayValue:  r.ReplayValue,
			SpoilerLevel: r.SpoilerLevel,
			LikeCount:    r.LikeCount,
			Created:      r.Created,
			Updated:      r.Updated,
			Galgame:      galgame,
		})
	}

	return &dto.UserRatingsResponse{RatingData: items, Total: total}, nil
}
