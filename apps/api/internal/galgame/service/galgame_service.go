package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/moemoepoint"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
	"kun-galgame-api/pkg/utils"

	"gorm.io/gorm"
)

// GalgameService handles the "core" galgame lifecycle: create, merge PR,
// detail aggregation, list with filters, and local interaction toggles.
type GalgameService struct {
	galgameRepo      *repository.GalgameRepository
	interactionRepo  *repository.GalgameInteractionRepository
	listRepo         *repository.GalgameListRepository
	resourceMetaRepo *repository.GalgameResourceMetaRepository
	detailRatingRepo *repository.GalgameDetailRatingRepository
	stateRepo        *userRepo.StateRepository
	galgameClient    *client.GalgameClient
	userClient       *userclient.Client
	helpers          InteractionHelpers
}

func NewGalgameService(
	galgameRepo *repository.GalgameRepository,
	interactionRepo *repository.GalgameInteractionRepository,
	listRepo *repository.GalgameListRepository,
	resourceMetaRepo *repository.GalgameResourceMetaRepository,
	detailRatingRepo *repository.GalgameDetailRatingRepository,
	stateRepo *userRepo.StateRepository,
	galgameClient *client.GalgameClient,
	userClient *userclient.Client,
) *GalgameService {
	return &GalgameService{
		galgameRepo:      galgameRepo,
		interactionRepo:  interactionRepo,
		listRepo:         listRepo,
		resourceMetaRepo: resourceMetaRepo,
		detailRatingRepo: detailRatingRepo,
		stateRepo:        stateRepo,
		galgameClient:    galgameClient,
		userClient:       userClient,
	}
}

// ──────────────────────────────────────────
// Create — POST /galgame
// ──────────────────────────────────────────

// Create forwards the payload to galgame, then awards moemoepoint and creates
// the local stub row for the new galgame. Returns the raw galgame response body
// so the handler can forward it verbatim.
//
// Daily-limit policy (mirrors topic create, formerly nitro
// api/galgame/index.post.ts:43): a user can create at most
// `moemoepoint/10 + 1` galgames per 24h. The limit is checked BEFORE the
// galgame call so we don't pollute galgame with rejects, using galgame's own
// `galgame_created_today` stat as the canonical day-count (kungal has no
// local creation log post-migration). There's still a thin race window
// between the check and galgame accepting the create — acceptable because
// galgame rejects duplicate vndb_id, which is the main spam vector.
//
// Post-success local side effects (stub row + moemoepoint +3) run inside
// a single transaction with SELECT … FOR UPDATE on kungal_user_state, so
// concurrent self-double-submits can't double-reward.
func (s *GalgameService) Create(
	ctx context.Context,
	userID int,
	token string,
	body []byte,
	contentType string,
) (json.RawMessage, *errors.AppError) {
	// Daily-limit gate.
	state, err := s.stateRepo.FindByID(userID)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该用户")
	}
	dailyLimit := int64(state.Moemoepoint/10 + 1)
	if galgameStats, sErr := s.galgameClient.GetUserStats(ctx, userID); sErr == nil && galgameStats != nil {
		if galgameStats.GalgameCreatedToday >= dailyLimit {
			return nil, errors.ErrBadRequest("您今日发布的 Galgame 已达上限")
		}
	}
	// On galgame stats failure we choose to allow the create rather than
	// hard-fail — galgame itself remains the authority on VNDB-ID uniqueness.

	// Forward to galgame.
	data, appErr := s.galgameClient.PostWithToken(ctx, "/galgame", token, json.RawMessage(body), contentType)
	if appErr != nil {
		return nil, appErr
	}

	var created dto.NextMoeCreatedResp
	_ = json.Unmarshal(data, &created)

	if created.ID > 0 {
		// Local stub so the galgame appears in kungal's list query. Idempotent
		// (OnConflict DoNothing), so no row lock needed. Log-only on failure:
		// the galgame create already succeeded; a missing stub self-heals on the
		// first interaction (lazy stub) and the approved-cron stub seed.
		if err := s.galgameRepo.CreateLocalStub(s.galgameRepo.DB(), created.ID); err != nil {
			slog.Warn("创建本地 galgame stub 失败", "gid", created.ID, "error", err)
		}
		// NOTE: deliberately no moemoepoint award here. A fresh create lands as
		// status=pending on the galgame; +RewardCreateGalgame is granted exactly
		// once when it is actually approved, via the galgame "approved" message in
		// wiki_message_sync (Ref "galgame"). Awarding at create time too would
		// double-count (the same galgame paid twice) and would pay out for
		// content that may yet be declined.
	}
	return data, nil
}

// ──────────────────────────────────────────
// Interactions — PUT /galgame/:gid/like|favorite
// ──────────────────────────────────────────

// ToggleLike reports an error when the user tries to self-like, otherwise
// atomically flips the like and adjusts owner moemoepoint + notification.
func (s *GalgameService) ToggleLike(
	ctx context.Context,
	userID, galgameID int,
) *errors.AppError {
	ownerID, name := s.fetchOwnerAndName(ctx, galgameID)
	if ownerID == userID {
		return errors.ErrBadRequest("您不能给自己点赞")
	}

	s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
		liked := s.interactionRepo.ToggleLike(tx, userID, galgameID)
		if liked {
			s.helpers.AdjustMoemoepoint(tx, ownerID, 1,
				moemoepoint.ReasonLiked, moemoepoint.Ref("galgame", galgameID))
			s.helpers.CreateGalgameMessageWithContent(tx, userID, ownerID, "liked", name, galgameID)
		} else {
			s.helpers.AdjustMoemoepoint(tx, ownerID, -1,
				moemoepoint.ReasonLiked, moemoepoint.Ref("galgame", galgameID))
		}
		return nil
	})
	return nil
}

// NOTE: favorite is no longer a simple per-galgame toggle — it is now membership
// in one or more collections (收藏夹). The write path + owner moemoe/notification
// (first-add / last-remove semantics) live in CollectionService.SetMembership.

// GetMyInteractions returns the current user's liked + favorited galgame ids,
// for hydrating feed-card like/favorite state.
func (s *GalgameService) GetMyInteractions(userID int) dto.MyGalgameInteractions {
	liked, favorited := s.interactionRepo.UserGalgameInteractions(userID)
	return dto.MyGalgameInteractions{Liked: liked, Favorited: favorited}
}

// fetchOwnerAndName reads the galgame's owner user_id AND a display name from
// galgame in ONE request (0 / "" on any failure). The name becomes the notification
// content preview so a favorite/like notice shows WHICH galgame instead of a
// blank line — see the CreateGalgameMessageWithContent callers below.
//
// Fallback order zh-CN → zh-TW → ja-JP → en-US mirrors the FE's
// getPreferredLanguageText zh-cn default. en-US (usually the VNDB romaji title)
// is LAST on purpose: a JP/CN-titled game must never surface its VNDB English
// name when a Chinese or Japanese name exists.
func (s *GalgameService) fetchOwnerAndName(ctx context.Context, galgameID int) (int, string) {
	// /v1 aggregate detail: names are always present; user_id lives in the meta
	// block (include=meta). content_limit=all keeps an NSFW title resolvable (the
	// default sfw would 404 it, and a like/favorite notification can target any
	// game); the key's galgame:nsfw scope makes `all` effective.
	data, err := s.galgameClient.GetV1(
		ctx, fmt.Sprintf("/galgame/%d", galgameID),
		url.Values{"include": {"meta"}, "content_limit": {"all"}},
	)
	if err != nil {
		return 0, ""
	}
	var g struct {
		Names struct {
			EnUS *string `json:"en-us"`
			JaJP *string `json:"ja-jp"`
			ZhCN *string `json:"zh-cn"`
			ZhTW *string `json:"zh-tw"`
		} `json:"names"`
		Meta struct {
			UserID int `json:"user_id"`
		} `json:"meta"`
	}
	_ = json.Unmarshal(data, &g)
	deref := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	name := firstNonEmpty(deref(g.Names.ZhCN), deref(g.Names.ZhTW), deref(g.Names.JaJP), deref(g.Names.EnUS))
	return g.Meta.UserID, truncate(name, constants.TextPreviewLength)
}

// firstNonEmpty returns the first non-blank argument, or "".
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ──────────────────────────────────────────
// GetDetail — GET /galgame/:gid
// ──────────────────────────────────────────

// GetDetail aggregates galgame metadata + local interaction stats into the
// full detail payload.
//
// token (Bearer access token from session, may be empty) is forwarded to
// galgame so its visibility filter sees the caller's identity — the
// submitter of a pending draft can view their own row, an authenticated
// user can see VNDB-source drafts, etc. Anonymous viewers get the same
// behavior as before (status=0 only).
//
// NSFW is NOT gated here — galgame's /galgame/:gid default is "不过滤"
// (docs/galgame_wiki/00-handbook §16.2: direct URL access is "有意为之").
// We deliberately let the response carry contentLimit through to the FE
// and let the FE decide UX: anonymous + SFW-cookie users see a "click
// to confirm" interstitial, logged-in OR NSFW-cookie-on users see the
// page directly. This trades a tiny SSR leak (mitigated by FE
// `useKunDisableSeo`) for a much better UX on shared NSFW links.
func (s *GalgameService) GetDetail(
	ctx context.Context,
	galgameID, currentUserID int,
	token string,
	isSFW bool,
) (*dto.GalgameDetail, *errors.AppError) {
	// content_limit=all (permissive): /galgame/:gid is "不过滤" (§16.2 direct URL
	// access is 有意为之) — the FE decides the NSFW interstitial. /v1 serves only
	// published rows, so a banned (status=1) or unknown id comes back as a
	// not-found AppError (the bridge's "该 Galgame 已被封禁" message is subsumed by
	// the standard not-found — /v1 never returns a banned row).
	g, appErr := s.galgameClient.GalgameDetailV1(ctx, galgameID, token, "all")
	if appErr != nil {
		return nil, appErr
	}

	// Async view bump (don't block the response).
	go s.galgameRepo.IncrementView(galgameID)

	local := s.galgameRepo.FindLocal(galgameID)
	isLiked, isFavorited := s.interactionRepo.UserInteraction(currentUserID, galgameID)

	platforms, languages, types := s.resourceMetaRepo.FindResourceMetaByGalgame(galgameID)

	var series *dto.GalgameDetailSeries
	if g.SeriesID != nil {
		series = s.fetchSeriesBrief(ctx, *g.SeriesID, isSFW)
	}

	ratings := s.buildDetailRatings(ctx, galgameID, currentUserID, g)

	// /v1 carries no users map — resolve the author + contributors from OAuth so
	// galgameDetailFromNextMoe can attach their briefs (same source the bridge's
	// users map came from).
	users := s.hydrateDetailUsers(ctx, g)
	detail := galgameDetailFromNextMoe(g, users)
	detail.View = local.View
	detail.LikeCount = local.LikeCount
	detail.FavoriteCount = local.FavoriteCount
	detail.ResourcePublishBanned = local.ResourcePublishBanned
	// No local row ⇒ a galgame-catalogue game the forum has never ingested. The FE
	// then shows a 未收录 notice + hides the (always-0) view count, but keeps the
	// upload/rate/comment CTAs that create the local row on first use.
	detail.IsOnForum = local.ID != 0
	detail.IsLiked = isLiked
	detail.IsFavorited = isFavorited
	detail.Platform = platforms
	detail.Language = languages
	detail.Type = types
	detail.Series = series
	detail.Ratings = ratings
	return &detail, nil
}

// hydrateDetailUsers resolves the author + every contributor from OAuth into the
// uid-keyed users map galgameDetailFromNextMoe consumes (the /v1 detail carries
// no users map — the bridge's map was itself an OAuth resolution, so this is the
// same source).
func (s *GalgameService) hydrateDetailUsers(ctx context.Context, g dto.NextMoeGalgameDetailFull) map[string]dto.NextMoeUser {
	uids := make([]int, 0, len(g.Contributor)+1)
	uids = append(uids, g.UserID)
	for _, c := range g.Contributor {
		uids = append(uids, c.UserID)
	}
	umap := s.userClient.Hydrate(ctx, uids)
	users := make(map[string]dto.NextMoeUser, len(umap))
	for id, u := range umap {
		users[strconv.Itoa(id)] = dto.NextMoeUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
	}
	return users
}

// fetchSeriesBrief loads a minimal series summary (used on galgame detail page).
//
// content_limit MUST be sent: /series/:id defaults to sfw when omitted, so an
// all-NSFW series reports 0 members + isNSFW=false on the detail page (the
// series block looks empty / mislabeled SFW even while viewing an NSFW title).
// Mirror series_service: SFW → sfw; NSFW → all. The /v1 series entity carries
// created/updated (W1d) + the gated member previews (with meta) the samples need.
func (s *GalgameService) fetchSeriesBrief(ctx context.Context, seriesID int, isSFW bool) *dto.GalgameDetailSeries {
	contentLimit := "all"
	if isSFW {
		contentLimit = "sfw"
	}
	entry, found, appErr := s.galgameClient.SeriesEntryV1(ctx, strconv.Itoa(seriesID), contentLimit)
	if appErr != nil || !found {
		return nil
	}

	isNSFW := false
	samples := make([]dto.GalgameSample, 0, min(len(entry.Galgame), 5))
	for i, sg := range entry.Galgame {
		if sg.ContentLimit == "nsfw" {
			isNSFW = true
		}
		if i < 5 {
			samples = append(samples, dto.GalgameSample{
				Name: dto.KunLanguage{
					EnUs: sg.NameEnUs, JaJp: sg.NameJaJp,
					ZhCn: sg.NameZhCn, ZhTw: sg.NameZhTw,
				},
				Banner:                   sg.Banner,
				EffectiveBannerHash:      sg.EffectiveBannerHash,
				EffectiveBannerURL:       sg.EffectiveBannerURL,
				EffectiveBannerWidth:     sg.EffectiveBannerWidth,
				EffectiveBannerHeight:    sg.EffectiveBannerHeight,
				EffectiveBannerThumbhash: sg.EffectiveBannerThumbhash,
			})
		}
	}
	return &dto.GalgameDetailSeries{
		ID:            entry.ID,
		Name:          entry.Name,
		Description:   entry.Description,
		IsNSFW:        isNSFW,
		SampleGalgame: samples,
		GalgameCount:  len(entry.Galgame),
		Created:       entry.Created,
		Updated:       entry.Updated,
	}
}

// buildDetailRatings assembles the ratings list with user resolution and liked flag.
func (s *GalgameService) buildDetailRatings(
	ctx context.Context,
	galgameID, currentUserID int,
	g dto.NextMoeGalgameDetailFull,
) []dto.GalgameDetailRating {
	rows := s.detailRatingRepo.FindRatingsByGalgame(galgameID)
	if len(rows) == 0 {
		return []dto.GalgameDetailRating{}
	}

	userIDs := make([]int, len(rows))
	ratingIDs := make([]int, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
		ratingIDs[i] = r.ID
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)
	likedSet := s.detailRatingRepo.FindLikedRatingIDs(currentUserID, ratingIDs)

	out := make([]dto.GalgameDetailRating, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		out = append(out, detailRatingFromRow(r, u, likedSet[r.ID], galgameID, g))
	}
	return out
}

// ──────────────────────────────────────────
// GetList — GET /galgame
// ──────────────────────────────────────────

func (s *GalgameService) GetList(
	ctx context.Context,
	req *dto.GalgameListRequest,
	isSFW bool,
) (*dto.GalgameListPage, *errors.AppError) {
	sortOrder := req.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	// Resolve the release-date filter (galgame §17 "YYYY"/"YYYY-MM") to
	// inclusive date boundaries. Malformed input is a client error, not
	// a silently-ignored param.
	releasedFrom, err := utils.ParseReleaseLowerBound(req.ReleasedFrom)
	if err != nil {
		return nil, errors.ErrBadRequest(err.Error())
	}
	releasedTo, err := utils.ParseReleaseUpperBound(req.ReleasedTo)
	if err != nil {
		return nil, errors.ErrBadRequest(err.Error())
	}
	releasedMonths, err := utils.ParseMonthSet(req.ReleasedMonths)
	if err != nil {
		return nil, errors.ErrBadRequest(err.Error())
	}

	filter := model.GalgameListFilter{
		Type:                 req.Type,
		Language:             req.Language,
		Platform:             req.Platform,
		GameType:             req.GameType,
		SortField:            req.SortField,
		SortOrder:            sortOrder,
		IncludeProviders:     splitCSV(req.IncludeProviders),
		ExcludeOnlyProviders: splitCSV(req.ExcludeOnlyProviders),
		ReleasedFrom:         releasedFrom,
		ReleasedTo:           releasedTo,
		ReleasedMonths:       releasedMonths,
		MinRatingCount:       req.MinRatingCount,
		MinRating:            req.MinRating,
		ShowNoResource:       req.ShowNoResource,
		Page:                 req.Page,
		Limit:                req.Limit,
	}

	return s.hydrateListCards(ctx, filter, isSFW)
}

// hydrateListCards runs the shared "filter → ids → hydrate → cards" flow used by
// BOTH the global /galgame list AND the galgame-entity detail pages (tag/official/
// engine, which set filter.RestrictIDs = the galgame member ids). All filtering /
// sorting / pagination is local (list_repo over galgame_resource); hydration
// pulls galgame metadata + OAuth users + local stats/ratings/resource-meta. Keeping
// this in one place is why the entity pages add zero duplicated filter logic.
func (s *GalgameService) hydrateListCards(
	ctx context.Context,
	filter model.GalgameListFilter,
	isSFW bool,
) (*dto.GalgameListPage, *errors.AppError) {
	// Note: `total` from listRepo is the count of kungal-known galgames (stats
	// rows) and can over-report when galgame drops NSFW briefs in SFW mode; an exact
	// total requires the public list to source from galgame's /galgame, not kungal's
	// local stats — out of scope here.
	ids, total := s.listRepo.ListIDs(filter)
	if len(ids) == 0 {
		return &dto.GalgameListPage{Galgames: []dto.GalgameListCard{}, Total: total}, nil
	}
	cards, appErr := s.HydrateCardsByIDs(ctx, ids, isSFW)
	if appErr != nil {
		return nil, appErr
	}
	return &dto.GalgameListPage{Galgames: cards, Total: total}, nil
}

// HydrateCardsByIDs turns an ORDERED galgame id list into list cards, fusing
// galgame metadata + OAuth users + local stats/ratings/resource-meta. The output
// preserves the input order and drops ids the galgame filtered out (NSFW miss /
// deleted). Shared by the global list, the galgame-entity detail pages, AND the
// collection detail (收藏夹) so none of them duplicate the hydration.
//
// SFW gating is delegated to galgame via content_limit per
// docs/galgame_wiki/00-handbook §16 — no service-layer post-filter (would
// violate "galgame is the only NSFW SoT"). A galgame-batch error is surfaced, not
// silently degraded to a blank list.
func (s *GalgameService) HydrateCardsByIDs(
	ctx context.Context,
	ids []int,
	isSFW bool,
) ([]dto.GalgameListCard, *errors.AppError) {
	if len(ids) == 0 {
		return []dto.GalgameListCard{}, nil
	}

	briefMap, appErr := s.galgameClient.GetBatchPublic(ctx, ids, isSFW)
	if appErr != nil {
		return nil, appErr
	}

	// Users (from galgame briefs) — hydrated from OAuth.
	userIDs := make([]int, 0, len(briefMap))
	for _, b := range briefMap {
		userIDs = append(userIDs, b.UserID)
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)

	// Local stats batch
	localMap := s.galgameRepo.FindLocalBatch(ids)

	// Bayesian display rating per card (same formula as the rating sort).
	ratingMap := s.listRepo.BayesianRatings(ids)

	// Platform/language aggregation
	metaRows := s.resourceMetaRepo.FindResourceMetaBatch(ids)
	platformMap, languageMap := groupResourceMeta(metaRows)

	cards := make([]dto.GalgameListCard, 0, len(ids))
	for _, id := range ids {
		b, ok := briefMap[id]
		if !ok {
			continue
		}
		cards = append(cards, dto.GalgameListCard{
			ID: id,
			Name: dto.KunLanguage{
				EnUs: b.NameEnUs, JaJp: b.NameJaJp,
				ZhCn: b.NameZhCn, ZhTw: b.NameZhTw,
			},
			Banner:       b.Banner,
			User:         userBriefToDTO(userMap[b.UserID]),
			ContentLimit: b.ContentLimit,
			View:         localMap[id].View,
			LikeCount:    localMap[id].LikeCount,
			Rating:       ratingMap[id].Score,
			RatingCount:  ratingMap[id].Count,
			// kungal's own list: the displayed "最近更新" comes from the LOCAL
			// resource_update_time (the sort key), NOT the galgame's (which never
			// tracks kungal resource activity) — so order and label agree.
			ResourceUpdateTime:       localMap[id].ResourceUpdateTime.Format(time.RFC3339),
			ReleaseDate:              b.ReleaseDate,
			ReleaseDateTBA:           b.ReleaseDateTBA,
			EffectiveBannerHash:      b.EffectiveBannerHash,
			EffectiveBannerURL:       b.EffectiveBannerURL,
			EffectiveBannerWidth:     b.EffectiveBannerWidth,
			EffectiveBannerHeight:    b.EffectiveBannerHeight,
			EffectiveBannerThumbhash: b.EffectiveBannerThumbhash,
			Platform:                 emptyStrSliceIfNil(platformMap[id]),
			Language:                 emptyStrSliceIfNil(languageMap[id]),
		})
	}
	return cards, nil
}
