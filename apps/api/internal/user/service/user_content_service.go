package service

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"strconv"

	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
)

type UserContentService struct {
	userContentRepo *repository.UserContentRepository
	wikiClient      *galgameClient.GalgameClient
	userClient      *userclient.Client
	community       *communityclient.Client
}

func NewUserContentService(
	userContentRepo *repository.UserContentRepository,
	wikiClient *galgameClient.GalgameClient,
	userClient *userclient.Client,
	community *communityclient.Client,
) *UserContentService {
	return &UserContentService{
		userContentRepo: userContentRepo,
		wikiClient:      wikiClient,
		userClient:      userClient,
		community:       community,
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

// GetUserGalgameComments returns the comment-card data for the "评论 / 点赞评论"
// tabs under /user/:id/galgame/, now sourced from the community primitive
// (charter step 06a). Keyset paginated (opaque `after` cursor); the envelope
// carries the next cursor ("" = last page). Content is rendered via the forum's
// own goldmark pipeline (charter ruling 7) so the frontend drops it into
// <KunContent> consistently with the rest of the site. A down/unconfigured
// community degrades to an empty page (fail-closed).
//
// isSFW is intentionally NOT applied here: keyset pagination cannot afford the
// per-galgame wiki brief filter the old offset path used (it would yield ragged
// pages), and both tabs only expose galgame_id + comment text — the click-
// through /galgame/:id target is itself SFW-gated.
func (s *UserContentService) GetUserGalgameComments(
	ctx context.Context,
	userID int,
	req *dto.UserGalgameCommentsRequest,
	_ bool, // isSFW: intentionally unused — see the note above
) ([]dto.UserGalgameComment, string, *errors.AppError) {
	if s.hideTarget(ctx, userID) {
		return []dto.UserGalgameComment{}, "", nil
	}
	if req.Type == "galgame_comment_like" {
		return s.likedGalgameComments(ctx, userID, req.After, req.Limit)
	}
	return s.authoredGalgameComments(ctx, userID, req.After, req.Limit)
}

// authoredGalgameComments serves the 评论 tab: the profile owner's own visible
// posts across every galgame comments thread, via the community by-author read.
// The author of every row is the profile owner (one Hydrate call).
func (s *UserContentService) authoredGalgameComments(ctx context.Context, userID int, after string, limit int) ([]dto.UserGalgameComment, string, *errors.AppError) {
	resp, err := s.community.AuthorPosts(ctx, int64(userID), after, limit, communityclient.AnchorSiteGame)
	if err != nil {
		if communityDown(err) {
			return []dto.UserGalgameComment{}, "", nil
		}
		return nil, "", errors.ErrInternal("获取用户 Galgame 评论列表失败")
	}
	owner := s.userClient.Hydrate(ctx, []int{userID})[userID]
	items := make([]dto.UserGalgameComment, 0, len(resp.Posts))
	for _, av := range resp.Posts {
		items = append(items, dto.UserGalgameComment{
			ID:          av.Post.ID,
			GalgameID:   anchorGalgameID(av.Thread),
			Content:     av.Post.ContentRaw,
			ContentHtml: markdown.Render(av.Post.ContentRaw),
			User:        dto.UserBrief{ID: owner.ID, Name: owner.Name, Avatar: owner.Avatar},
			Created:     av.Post.CreatedAt,
			Deleted:     false, // by-author returns visible posts only
		})
	}
	return items, resp.NextCursor, nil
}

// likedGalgameComments serves the 点赞评论 tab: the community posts this user
// liked, in local like-recency order. It keyset-pages the LIVE local
// galgame_post_like table, batch-hydrates the post content via ResolvePosts, and
// emits a placeholder (deleted:true) for any liked post the primitive no longer
// returns (hidden/deleted) so a keyset page never silently shrinks.
func (s *UserContentService) likedGalgameComments(ctx context.Context, userID int, after string, limit int) ([]dto.UserGalgameComment, string, *errors.AppError) {
	rows, err := s.userContentRepo.FindUserLikedPostIDs(userID, after, limit)
	if err != nil {
		return nil, "", errors.ErrInternal("获取用户 Galgame 评论列表失败")
	}
	nextCursor := ""
	if len(rows) > limit {
		nextCursor = strconv.FormatInt(rows[limit-1].ID, 10)
		rows = rows[:limit]
	}
	if len(rows) == 0 {
		return []dto.UserGalgameComment{}, "", nil
	}

	postIDs := make([]int64, len(rows))
	for i, r := range rows {
		postIDs[i] = r.PostID
	}
	resolved, err := s.community.ResolvePosts(ctx, postIDs)
	if err != nil {
		if communityDown(err) {
			return []dto.UserGalgameComment{}, "", nil
		}
		return nil, "", errors.ErrInternal("获取用户 Galgame 评论列表失败")
	}
	byID := make(map[int64]communityclient.AuthorPostView, len(resolved.Posts))
	authorIDs := make([]int, 0, len(resolved.Posts))
	for _, av := range resolved.Posts {
		byID[av.Post.ID] = av
		authorIDs = append(authorIDs, int(av.Post.AuthorID))
	}
	userMap := s.userClient.Hydrate(ctx, authorIDs)

	items := make([]dto.UserGalgameComment, 0, len(rows))
	for _, r := range rows {
		av, ok := byID[r.PostID]
		author := userMap[int(av.Post.AuthorID)]
		// Absent from resolve (hidden/deleted/unknown) OR authored by a banned
		// user → a "已被删除" placeholder that keeps the page size stable.
		if !ok || !userclient.IsRenderable(author) {
			items = append(items, dto.UserGalgameComment{ID: r.PostID, Deleted: true, User: dto.UserBrief{}})
			continue
		}
		items = append(items, dto.UserGalgameComment{
			ID:          av.Post.ID,
			GalgameID:   anchorGalgameID(av.Thread),
			Content:     av.Post.ContentRaw,
			ContentHtml: markdown.Render(av.Post.ContentRaw),
			User:        dto.UserBrief{ID: author.ID, Name: author.Name, Avatar: author.Avatar},
			Created:     av.Post.CreatedAt,
			Deleted:     false,
		})
	}
	return items, nextCursor, nil
}

// communityDown reports whether a community error means the service is
// unreachable / unconfigured / forbidden — the cases where a profile comment
// read degrades to an empty page rather than a 500 (mirrors the galgame comment
// BFF's isCommunityDown; a tiny local copy keeps the user package decoupled).
func communityDown(err error) bool {
	if stderrors.Is(err, communityclient.ErrNotConfigured) || stderrors.Is(err, communityclient.ErrForbidden) {
		return true
	}
	var apiErr *communityclient.APIError
	return err != nil && !stderrors.As(err, &apiErr) && !stderrors.Is(err, communityclient.ErrRateLimited)
}

// anchorGalgameID extracts the galgame id from a thread's anchor. It is only
// meaningful for a site_game anchor (kind=1, anchor_id = galgame id text); any
// other anchor (a resource comment the user liked — galgame_post_like is region-
// agnostic, charter ruling 20) yields 0.
func anchorGalgameID(thread communityclient.PostThreadContext) int {
	if thread.AnchorKind != communityclient.AnchorSiteGame {
		return 0
	}
	gid, _ := strconv.Atoi(thread.AnchorID)
	return gid
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
