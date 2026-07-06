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
	wikiClient       *client.GalgameClient
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
	wikiClient *client.GalgameClient,
	userClient *userclient.Client,
) *GalgameService {
	return &GalgameService{
		galgameRepo:      galgameRepo,
		interactionRepo:  interactionRepo,
		listRepo:         listRepo,
		resourceMetaRepo: resourceMetaRepo,
		detailRatingRepo: detailRatingRepo,
		stateRepo:        stateRepo,
		wikiClient:       wikiClient,
		userClient:       userClient,
	}
}

// ──────────────────────────────────────────
// Create — POST /galgame
// ──────────────────────────────────────────

// Create forwards the payload to wiki, then awards moemoepoint and creates
// the local stub row for the new galgame. Returns the raw wiki response body
// so the handler can forward it verbatim.
//
// Daily-limit policy (mirrors topic create, formerly nitro
// api/galgame/index.post.ts:43): a user can create at most
// `moemoepoint/10 + 1` galgames per 24h. The limit is checked BEFORE the
// wiki call so we don't pollute wiki with rejects, using wiki's own
// `galgame_created_today` stat as the canonical day-count (kungal has no
// local creation log post-migration). There's still a thin race window
// between the check and wiki accepting the create — acceptable because
// wiki rejects duplicate vndb_id, which is the main spam vector.
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
	if wikiStats, sErr := s.wikiClient.GetUserStats(ctx, userID); sErr == nil && wikiStats != nil {
		if wikiStats.GalgameCreatedToday >= dailyLimit {
			return nil, errors.ErrBadRequest("您今日发布的 Galgame 已达上限")
		}
	}
	// On wiki stats failure we choose to allow the create rather than
	// hard-fail — wiki itself remains the authority on VNDB-ID uniqueness.

	// Forward to wiki.
	data, appErr := s.wikiClient.PostWithToken(ctx, "/galgame", token, json.RawMessage(body), contentType)
	if appErr != nil {
		return nil, appErr
	}

	var created dto.WikiCreatedResp
	_ = json.Unmarshal(data, &created)

	if created.ID > 0 {
		// Local stub so the galgame appears in kungal's list query. Idempotent
		// (OnConflict DoNothing), so no row lock needed. Log-only on failure:
		// the wiki create already succeeded; a missing stub self-heals on the
		// first interaction (lazy stub) and the approved-cron stub seed.
		if err := s.galgameRepo.CreateLocalStub(s.galgameRepo.DB(), created.ID); err != nil {
			slog.Warn("创建本地 galgame stub 失败", "gid", created.ID, "error", err)
		}
		// NOTE: deliberately no moemoepoint award here. A fresh create lands as
		// status=pending on the wiki; +RewardCreateGalgame is granted exactly
		// once when it is actually approved, via the wiki "approved" message in
		// wiki_message_sync (Ref "galgame"). Awarding at create time too would
		// double-count (the same galgame paid twice) and would pay out for
		// content that may yet be declined.
	}
	return data, nil
}

// ──────────────────────────────────────────
// MergePR — PUT /galgame/:gid/prs/:id/merge
// ──────────────────────────────────────────

func (s *GalgameService) MergePR(
	ctx context.Context,
	mergerID int,
	gid, prID, token string,
) (json.RawMessage, *errors.AppError) {
	// Look up submitter before merging (wiki may purge pending info after merge)
	prData, appErr := s.wikiClient.Get(ctx, fmt.Sprintf("/galgame/%s/prs/%s", gid, prID), nil)
	if appErr != nil {
		return nil, appErr
	}
	var prInfo dto.WikiPRDetail
	_ = json.Unmarshal(prData, &prInfo)

	data, appErr := s.wikiClient.PutWithToken(
		ctx, fmt.Sprintf("/galgame/%s/prs/%s/merge", gid, prID), token, nil, "",
	)
	if appErr != nil {
		return nil, appErr
	}

	gidInt, _ := strconv.Atoi(gid)

	// A merged PR changed the galgame's content → bump local resource_update_time
	// so it rises in the kungal "sort by update time" list.
	if gidInt > 0 {
		if err := s.galgameRepo.Touch(s.galgameRepo.DB(), gidInt); err != nil {
			slog.Warn("mergePR: 刷新本地 galgame resource_update_time 失败", "gid", gidInt, "error", err)
		}
	}

	submitter := prInfo.PR.UserID
	if submitter > 0 && submitter != mergerID {
		// Resolve the galgame name up front (wiki call, kept out of the tx) so
		// the "已被 X 合并" notification can show WHICH game. This merge notice
		// was the one galgame notification still created with empty content —
		// surfacing as a blank preview + no game name in the message center.
		_, name := s.fetchOwnerAndName(ctx, gidInt)
		s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
			s.helpers.AdjustMoemoepoint(tx, submitter, constants.RewardPRMerge,
				moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame_pr", gidInt))
			s.helpers.CreateGalgameMessageWithContent(tx, mergerID, submitter, "merged", name, gidInt)
			return nil
		})
	}
	return data, nil
}

// ──────────────────────────────────────────
// SubmitPR — POST /galgame/:gid/prs
// ──────────────────────────────────────────

// SubmitPR proxies a PR submission to wiki, then records two local, best-effort
// side effects (the wiki emits NO message on PR submit — its only 7 message
// types are submission-review events, see docs/galgame_wiki/08-messages.md):
//
//  1. a "requested" notification to the galgame owner (mirror of MergePR's
//     "merged" notice), so the owner doesn't have to dig into the PR tab;
//  2. a GALGAME_PR_CREATION row in galgame_activity, so the submission shows on
//     the site-wide activity timeline — discoverable WITHOUT entering the game
//     page. This restores the timeline entry that was dropped when the galgame_pr
//     table moved to the wiki (the table can no longer be queried locally).
//
// body/contentType are forwarded byte-for-byte so the multipart (banner-in-PR)
// submit mode keeps working. Both side effects are best-effort and never fail
// the already-applied PR.
func (s *GalgameService) SubmitPR(
	ctx context.Context,
	submitterID int,
	gid, token string,
	body []byte,
	contentType string,
) (json.RawMessage, *errors.AppError) {
	data, appErr := s.wikiClient.PostWithToken(
		ctx, fmt.Sprintf("/galgame/%s/prs", gid), token, json.RawMessage(body), contentType,
	)
	if appErr != nil {
		return nil, appErr
	}

	gidInt, _ := strconv.Atoi(gid)
	if gidInt > 0 {
		ownerID, name := s.fetchOwnerAndName(ctx, gidInt)
		// Dedup on (sender, receiver, type, link) means repeat PRs from the same
		// user to the same galgame notify the owner once — intentional anti-spam
		// (the owner opens the PR tab to review them all). The helper no-ops when
		// ownerID == submitterID or ownerID <= 0.
		s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
			s.helpers.CreateGalgameMessageWithContent(tx, submitterID, ownerID, "requested", name, gidInt)
			return nil
		})

		// Mirror onto the activity timeline. The wiki PR id (from the create
		// response — shape is either {id} or {pr:{id}}) is the idempotency key; if
		// we can't read it we skip rather than risk a dupe. The content/name is
		// filled from the galgame brief at render time (enrichGalgameItems).
		var created struct {
			ID int `json:"id"`
			PR struct {
				ID int `json:"id"`
			} `json:"pr"`
		}
		_ = json.Unmarshal(data, &created)
		if prID := created.ID; prID > 0 || created.PR.ID > 0 {
			if prID == 0 {
				prID = created.PR.ID
			}
			if err := s.galgameRepo.DB().Exec(`
				INSERT INTO galgame_activity (wiki_pr_id, galgame_id, user_id, type, created)
				VALUES (?, ?, ?, 'GALGAME_PR_CREATION', now())
				ON CONFLICT (wiki_pr_id) DO NOTHING
			`, prID, gidInt, submitterID).Error; err != nil {
				slog.Warn("submitPR: 写入活动时间线失败", "gid", gidInt, "pr_id", prID, "error", err)
			}
		}
	}
	return data, nil
}

// ──────────────────────────────────────────
// DeclinePR — PUT /galgame/:gid/prs/:id/decline
// ──────────────────────────────────────────

// DeclinePR proxies a PR decline to wiki, then notifies the PR's submitter that
// their update request was rejected — symmetric with MergePR's "merged" notice.
// The submitter is read from the PR detail BEFORE the decline (wiki may purge
// pending info after). No moemoepoint change: only a *merged* contribution earns
// +RewardPRMerge. body carries the reviewer's optional {note}; the notification
// is best-effort.
func (s *GalgameService) DeclinePR(
	ctx context.Context,
	declinerID int,
	gid, prID, token string,
	body []byte,
	contentType string,
) (json.RawMessage, *errors.AppError) {
	prData, appErr := s.wikiClient.Get(ctx, fmt.Sprintf("/galgame/%s/prs/%s", gid, prID), nil)
	if appErr != nil {
		return nil, appErr
	}
	var prInfo dto.WikiPRDetail
	_ = json.Unmarshal(prData, &prInfo)

	data, appErr := s.wikiClient.PutWithToken(
		ctx, fmt.Sprintf("/galgame/%s/prs/%s/decline", gid, prID), token, json.RawMessage(body), contentType,
	)
	if appErr != nil {
		return nil, appErr
	}

	gidInt, _ := strconv.Atoi(gid)
	submitter := prInfo.PR.UserID
	if submitter > 0 && submitter != declinerID && gidInt > 0 {
		_, name := s.fetchOwnerAndName(ctx, gidInt)
		s.galgameRepo.DB().Transaction(func(tx *gorm.DB) error {
			s.helpers.CreateGalgameMessageWithContent(tx, declinerID, submitter, "declined", name, gidInt)
			return nil
		})
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
// wiki in ONE request (0 / "" on any failure). The name becomes the notification
// content preview so a favorite/like notice shows WHICH galgame instead of a
// blank line — see the CreateGalgameMessageWithContent callers below.
//
// Fallback order zh-CN → zh-TW → ja-JP → en-US mirrors the FE's
// getPreferredLanguageText zh-cn default. en-US (usually the VNDB romaji title)
// is LAST on purpose: a JP/CN-titled game must never surface its VNDB English
// name when a Chinese or Japanese name exists.
func (s *GalgameService) fetchOwnerAndName(ctx context.Context, galgameID int) (int, string) {
	data, err := s.wikiClient.Get(ctx, fmt.Sprintf("/galgame/%d", galgameID), nil)
	if err != nil {
		return 0, ""
	}
	var env struct {
		Galgame struct {
			UserID   int    `json:"user_id"`
			NameZhCn string `json:"name_zh_cn"`
			NameEnUs string `json:"name_en_us"`
			NameJaJp string `json:"name_ja_jp"`
			NameZhTw string `json:"name_zh_tw"`
		} `json:"galgame"`
	}
	_ = json.Unmarshal(data, &env)
	g := env.Galgame
	name := firstNonEmpty(g.NameZhCn, g.NameZhTw, g.NameJaJp, g.NameEnUs)
	return g.UserID, truncate(name, constants.TextPreviewLength)
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

// GetDetail aggregates wiki metadata + local interaction stats into the
// full detail payload.
//
// token (Bearer access token from session, may be empty) is forwarded to
// wiki so its visibility filter sees the caller's identity — the
// submitter of a pending draft can view their own row, an authenticated
// user can see VNDB-source drafts, etc. Anonymous viewers get the same
// behavior as before (status=0 only).
//
// NSFW is NOT gated here — wiki's /galgame/:gid default is "不过滤"
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
	wikiData, appErr := s.wikiClient.GetWithToken(
		ctx, fmt.Sprintf("/galgame/%d", galgameID), token, nil,
	)
	if appErr != nil {
		return nil, appErr
	}

	var parsed dto.WikiGalgameDetailFullResp
	if err := json.Unmarshal(wikiData, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}
	g := parsed.Galgame
	if g.Status == 1 {
		return nil, errors.ErrNotFound("该 Galgame 已被封禁")
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

	detail := galgameDetailFromWiki(g, parsed.Users)
	detail.View = local.View
	detail.LikeCount = local.LikeCount
	detail.FavoriteCount = local.FavoriteCount
	// No local row ⇒ a wiki-catalogue game the forum has never ingested. The FE
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

// fetchSeriesBrief loads a minimal series summary (used on galgame detail page).
//
// content_limit MUST be sent: /series/:id defaults to sfw when omitted, so an
// all-NSFW series reports 0 members + isNSFW=false on the detail page (the
// series block looks empty / mislabeled SFW even while viewing an NSFW title).
// Mirror series_service: SFW → sfw; NSFW → all.
func (s *GalgameService) fetchSeriesBrief(ctx context.Context, seriesID int, isSFW bool) *dto.GalgameDetailSeries {
	contentLimit := "all"
	if isSFW {
		contentLimit = "sfw"
	}
	data, err := s.wikiClient.Get(
		ctx, fmt.Sprintf("/series/%d", seriesID),
		url.Values{"content_limit": {contentLimit}},
	)
	if err != nil {
		return nil
	}
	var brief dto.WikiSeriesBrief
	if jsonErr := json.Unmarshal(data, &brief); jsonErr != nil {
		return nil
	}

	isNSFW := false
	samples := make([]dto.GalgameSample, 0, min(len(brief.Galgame), 5))
	for i, sg := range brief.Galgame {
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
		ID:            brief.ID,
		Name:          brief.Name,
		Description:   brief.Description,
		IsNSFW:        isNSFW,
		SampleGalgame: samples,
		GalgameCount:  len(brief.Galgame),
		Created:       brief.Created,
		Updated:       brief.Updated,
	}
}

// buildDetailRatings assembles the ratings list with user resolution and liked flag.
func (s *GalgameService) buildDetailRatings(
	ctx context.Context,
	galgameID, currentUserID int,
	g dto.WikiGalgameDetailFull,
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

	// Resolve the release-date filter (wiki §17 "YYYY"/"YYYY-MM") to
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
// BOTH the global /galgame list AND the wiki-entity detail pages (tag/official/
// engine, which set filter.RestrictIDs = the wiki member ids). All filtering /
// sorting / pagination is local (list_repo over galgame_resource); hydration
// pulls wiki metadata + OAuth users + local stats/ratings/resource-meta. Keeping
// this in one place is why the entity pages add zero duplicated filter logic.
func (s *GalgameService) hydrateListCards(
	ctx context.Context,
	filter model.GalgameListFilter,
	isSFW bool,
) (*dto.GalgameListPage, *errors.AppError) {
	// Note: `total` from listRepo is the count of kungal-known galgames (stats
	// rows) and can over-report when wiki drops NSFW briefs in SFW mode; an exact
	// total requires the public list to source from wiki's /galgame, not kungal's
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
// wiki metadata + OAuth users + local stats/ratings/resource-meta. The output
// preserves the input order and drops ids the wiki filtered out (NSFW miss /
// deleted). Shared by the global list, the wiki-entity detail pages, AND the
// collection detail (收藏夹) so none of them duplicate the hydration.
//
// SFW gating is delegated to wiki via content_limit per
// docs/galgame_wiki/00-handbook §16 — no service-layer post-filter (would
// violate "wiki is the only NSFW SoT"). A wiki-batch error is surfaced, not
// silently degraded to a blank list.
func (s *GalgameService) HydrateCardsByIDs(
	ctx context.Context,
	ids []int,
	isSFW bool,
) ([]dto.GalgameListCard, *errors.AppError) {
	if len(ids) == 0 {
		return []dto.GalgameListCard{}, nil
	}

	briefMap, appErr := s.wikiClient.GetBatchPublic(ctx, ids, isSFW)
	if appErr != nil {
		return nil, appErr
	}

	// Users (from wiki briefs) — hydrated from OAuth.
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
			// resource_update_time (the sort key), NOT the wiki's (which never
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
