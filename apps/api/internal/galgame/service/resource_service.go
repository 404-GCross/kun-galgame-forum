package service

import (
	"context"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/linkcheck"
	"kun-galgame-api/pkg/userclient"
	"kun-galgame-api/pkg/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ResourceService struct {
	resourceRepo *repository.ResourceRepository
	wikiClient   *client.GalgameClient
	userClient   *userclient.Client
	helpers      InteractionHelpers
	// linkChecker gates "report expired" on an objective netdisk-API verdict.
	// nil when unconfigured → MarkExpired falls back to the legacy flow.
	linkChecker *linkcheck.Client
}

func NewResourceService(
	resourceRepo *repository.ResourceRepository,
	wikiClient *client.GalgameClient,
	userClient *userclient.Client,
	linkChecker *linkcheck.Client,
) *ResourceService {
	return &ResourceService{
		resourceRepo: resourceRepo,
		wikiClient:   wikiClient,
		userClient:   userClient,
		linkChecker:  linkChecker,
	}
}

// ──────────────────────────────────────────
// GetResourceList — GET /galgame-resource
// ──────────────────────────────────────────

// GetResourceList returns the public resource list. SFW gating is
// delegated to wiki via content_limit per docs/galgame_wiki/00-handbook
// §16 — wiki only returns briefs for galgames matching the requested
// content_limit, so any row whose galgame is filtered shows up as
// "no brief returned" below. `total` over-reports in SFW mode (it's the
// count of kungal-side rows, not the post-filter remainder).
func (s *ResourceService) GetResourceList(
	ctx context.Context,
	req *dto.ResourceListRequest,
	currentUserID int,
	isSFW bool,
) (*dto.ResourceListPage, *errors.AppError) {
	total := s.resourceRepo.CountAll()
	rows := s.resourceRepo.ListPaginated(req.Page, req.Limit)

	galgameIDs, userIDs := collectIDs(rows)
	briefMap := s.fetchWikiBriefsPublic(ctx, galgameIDs, isSFW)
	userMap := s.userClient.Hydrate(ctx, userIDs)

	// Batch lookup of "did current user already like" so the FE can
	// hydrate the heart icon correctly on first render. Anonymous /
	// userID=0 yields an empty set without hitting the DB.
	resourceIDs := make([]int, len(rows))
	for i, r := range rows {
		resourceIDs[i] = r.ID
	}
	likedSet := s.resourceRepo.FindLikedSet(currentUserID, resourceIDs)

	cards := make([]dto.ResourceCard, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		b, hasBrief := briefMap[r.GalgameID]
		// Wiki dropped this row's galgame (didn't match content_limit
		// or doesn't exist) → resource is unrenderable in this mode.
		if !hasBrief {
			continue
		}
		card := rowToCard(r, u, likedSet[r.ID])
		card.GalgameName = briefToName(b)
		cards = append(cards, card)
	}

	return &dto.ResourceListPage{Resources: cards, Total: total}, nil
}

// ──────────────────────────────────────────
// GetResourceDetail — GET /galgame-resource/:id
// Returns (detail, nil) on success or (nil, nil) for "not found" (legacy format).
// ──────────────────────────────────────────

// NotFoundSentinel is returned by GetResourceDetail when the resource doesn't exist.
// The handler serialises it to the JSON string "not found" for backwards compat.
type ResourceNotFound struct{}

// GetResourceDetail returns the resource detail. NSFW is NOT gated here —
// the parent galgame's content_limit reaches the FE in the page payload,
// and the FE decides whether to show an "click to confirm" interstitial
// (same UX as galgame detail). Cross-detail behaviour stays consistent
// rather than 404-ing only on certain paths.
func (s *ResourceService) GetResourceDetail(
	ctx context.Context,
	resourceID, currentUserID int,
) (*dto.ResourceDetailPage, *ResourceNotFound, *errors.AppError) {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return nil, &ResourceNotFound{}, nil
	}

	// Fire-and-forget view increment. We add 1 to the returned value too so
	// the client sees the freshly-incremented count without re-fetching.
	s.resourceRepo.IncrementView(resourceID)
	row.View++

	links := s.resourceRepo.FindLinks(resourceID)
	isLiked := s.resourceRepo.IsLikedBy(resourceID, currentUserID)

	ownerUser, _, _ := s.userClient.User(ctx, row.UserID)

	resource := rowToMeta(row, links, isLiked, ownerUser)

	// Galgame summary
	galgameSummary := s.buildGalgameSummary(ctx, row.GalgameID)

	// Recommendations (max 6)
	recRows := s.resourceRepo.FindRecommendations(row.GalgameID, resourceID, 6)
	recommendations := s.buildRecommendations(ctx, recRows, row.GalgameID, currentUserID)

	return &dto.ResourceDetailPage{
		Galgame:         galgameSummary,
		Resource:        resource,
		Recommendations: recommendations,
	}, nil, nil
}

// ──────────────────────────────────────────
// GetResourceDownloadDetail — GET /galgame-resource/:id/detail
// Bumps the download counter and returns links/code/password.
// ──────────────────────────────────────────

func (s *ResourceService) GetResourceDownloadDetail(
	ctx context.Context,
	resourceID, currentUserID int,
) (*dto.ResourceDownloadDetail, *errors.AppError) {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return nil, errors.ErrNotFound("未找到该资源")
	}

	s.resourceRepo.IncrementDownload(resourceID)
	row.Download++

	links := s.resourceRepo.FindLinks(resourceID)
	isLiked := s.resourceRepo.IsLikedBy(resourceID, currentUserID)
	owner, _, _ := s.userClient.User(ctx, row.UserID)

	detail := rowToDownloadDetail(row, links, isLiked, owner)
	return &detail, nil
}

// ──────────────────────────────────────────
// GetGalgameResources — GET /galgame/:gid/resource/all
// ──────────────────────────────────────────

func (s *ResourceService) GetGalgameResources(
	ctx context.Context,
	req *dto.GalgameResourcesRequest,
	currentUserID int,
) ([]dto.ResourceCard, *errors.AppError) {
	rows := s.resourceRepo.FindByGalgameID(req.GalgameID)

	userIDs := make([]int, len(rows))
	resourceIDs := make([]int, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
		resourceIDs[i] = r.ID
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)
	// Per-row like state for the logged-in viewer; anonymous → empty set.
	likedSet := s.resourceRepo.FindLikedSet(currentUserID, resourceIDs)

	cards := make([]dto.ResourceCard, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		cards = append(cards, rowToCard(r, u, likedSet[r.ID]))
	}
	return cards, nil
}

// ──────────────────────────────────────────
// CreateResource — POST /galgame/:gid/resource
// Creates the resource row + links + provider tags in a single transaction,
// awarding +3 moemoepoint and bumping local resource_count.
// ──────────────────────────────────────────

func (s *ResourceService) CreateResource(
	userID int,
	req *dto.CreateGalgameResourceRequest,
) *errors.AppError {
	providers := utils.DetectProvidersFromURLs(req.Link)
	providerNames := utils.DetectProviderNamesFromURLs(req.Link)
	res := &model.GalgameResource{
		Type:      req.Type,
		Language:  req.Language,
		Platform:  req.Platform,
		Size:      req.Size,
		Code:      req.Code,
		Password:  req.Password,
		Note:      req.Note,
		GalgameID: req.GalgameID,
		UserID:    userID,
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		// Lazy-create stub before incrementing — see decision 2 in the
		// kungal/wiki integration plan (pending submissions don't get
		// a kungal stub, so the first interaction must INSERT one).
		tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&model.GalgameLocal{ID: req.GalgameID})
		if err := s.resourceRepo.Create(tx, res); err != nil {
			return err
		}
		if err := s.resourceRepo.ReplaceProviders(tx, res.ID, providers); err != nil {
			return err
		}
		if err := s.resourceRepo.ReplaceProviderNames(tx, res.ID, providerNames); err != nil {
			return err
		}
		if err := s.resourceRepo.CreateLinks(tx, res.ID, req.Link); err != nil {
			return err
		}
		if err := s.resourceRepo.AdjustLocalResourceCount(tx, req.GalgameID, 1); err != nil {
			return err
		}
		// Publishing a resource is a content update → bump
		// galgame.resource_update_time so it rises in the "sort by update time"
		// list (the count adjust above does not, by design — see TouchGalgameUpdated).
		if err := s.resourceRepo.TouchGalgameUpdated(tx, req.GalgameID); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, userID, constants.RewardCreateResource,
			moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame_resource", req.GalgameID))
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("创建 Galgame 资源失败")
	}
	return nil
}

// ──────────────────────────────────────────
// UpdateResource — PUT /galgame/:gid/resource
// Replaces all links, recomputes providers, patches scalar fields.
// ──────────────────────────────────────────

func (s *ResourceService) UpdateResource(
	userID int, canModerate bool,
	req *dto.UpdateGalgameResourceRequest,
) *errors.AppError {
	row, ok := s.resourceRepo.FindByID(req.GalgameResourceID)
	if !ok {
		return errors.ErrNotFound("未找到这个 Galgame 资源")
	}
	if row.UserID != userID && !canModerate {
		return errors.ErrForbidden("您没有权限更新这个 Galgame 资源")
	}

	providers := utils.DetectProvidersFromURLs(req.Link)
	providerNames := utils.DetectProviderNamesFromURLs(req.Link)
	fields := map[string]any{
		"type":     req.Type,
		"language": req.Language,
		"platform": req.Platform,
		"size":     req.Size,
		"code":     req.Code,
		"password": req.Password,
		"note":     req.Note,
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.resourceRepo.UpdateFields(tx, req.GalgameResourceID, fields); err != nil {
			return err
		}
		if err := s.resourceRepo.DeleteLinks(tx, req.GalgameResourceID); err != nil {
			return err
		}
		if err := s.resourceRepo.CreateLinks(tx, req.GalgameResourceID, req.Link); err != nil {
			return err
		}
		if err := s.resourceRepo.ReplaceProviders(tx, req.GalgameResourceID, providers); err != nil {
			return err
		}
		if err := s.resourceRepo.ReplaceProviderNames(tx, req.GalgameResourceID, providerNames); err != nil {
			return err
		}
		// Editing a published resource is a content update → bump galgame.resource_update_time.
		return s.resourceRepo.TouchGalgameUpdated(tx, row.GalgameID)
	})
	if txErr != nil {
		return errors.ErrInternal("更新 Galgame 资源失败")
	}
	return nil
}

// ──────────────────────────────────────────
// DeleteResource — DELETE /galgame/:gid/resource
// Deducts 5 + like_count moemoepoint from the original uploader and decrements
// the galgame resource_count. Cascade deletes links/likes via FK.
// ──────────────────────────────────────────

func (s *ResourceService) DeleteResource(
	userID int, canModerate bool, resourceID int,
) *errors.AppError {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return errors.ErrNotFound("未找到该 Galgame 资源")
	}
	if row.UserID != userID && !canModerate {
		return errors.ErrForbidden("您没有权限删除这个 Galgame 资源")
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		s.helpers.AdjustMoemoepoint(tx, row.UserID, -(row.LikeCount + 5),
			moemoepoint.ReasonContentRemoved, moemoepoint.Ref("galgame_resource", resourceID))
		if err := s.resourceRepo.DeleteByID(tx, resourceID); err != nil {
			return err
		}
		return s.resourceRepo.AdjustLocalResourceCount(tx, row.GalgameID, -1)
	})
	if txErr != nil {
		return errors.ErrInternal("删除 Galgame 资源失败")
	}
	return nil
}

// ──────────────────────────────────────────
// ToggleLike — PUT /galgame/:gid/resource/like
// Self-like is rejected. Toggles the like row + maintains like_count and a +/-1
// moemoepoint swing on the liker (matches existing nitro behaviour).
// ──────────────────────────────────────────

func (s *ResourceService) ToggleLike(
	userID int,
	req *dto.ToggleResourceLikeRequest,
) *errors.AppError {
	row, ok := s.resourceRepo.FindByID(req.GalgameResourceID)
	if !ok {
		return errors.ErrNotFound("未找到该资源")
	}
	if row.UserID == userID {
		return errors.ErrBadRequest("您不能给自己的资源点赞")
	}

	links := s.resourceRepo.FindLinks(req.GalgameResourceID)
	preview := ""
	if len(links) > 0 {
		preview = truncate(links[0], constants.TextPreviewLength)
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		existing, has := s.resourceRepo.FindLike(tx, req.GalgameResourceID, userID)
		var delta int
		if has {
			if err := s.resourceRepo.DeleteLike(tx, existing); err != nil {
				return err
			}
			delta = -1
		} else {
			if err := s.resourceRepo.CreateLike(tx, req.GalgameResourceID, userID); err != nil {
				return err
			}
			delta = 1
		}
		if err := s.resourceRepo.AdjustLikeCount(tx, req.GalgameResourceID, delta); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, userID, delta,
			moemoepoint.ReasonLiked, moemoepoint.Ref("galgame_resource", req.GalgameResourceID))
		s.helpers.CreateGalgameMessageWithContent(
			tx, userID, row.UserID, "liked", preview, row.GalgameID,
		)
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

// ──────────────────────────────────────────
// MarkValid / MarkExpired — PUT valid|expired
// MarkValid is restricted to the original uploader; MarkExpired is open.
// MarkExpired sends a non-deduped notification to the uploader.
// ──────────────────────────────────────────

func (s *ResourceService) MarkValid(userID int, resourceID int) *errors.AppError {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok || row.UserID != userID {
		return errors.ErrNotFound("未找到这个 Galgame 资源")
	}
	if err := s.resourceRepo.UpdateStatus(s.resourceRepo.DB(), resourceID, 0); err != nil {
		return errors.ErrInternal("更新失败")
	}
	return nil
}

func (s *ResourceService) MarkExpired(ctx context.Context, userID int, resourceID int) (*dto.ReportExpireResult, *errors.AppError) {
	row, ok := s.resourceRepo.FindByID(resourceID)
	if !ok {
		return nil, errors.ErrNotFound("未找到该 Galgame 资源")
	}
	if row.Status == 1 {
		return nil, errors.ErrBadRequest("该资源已经被标记为失效")
	}

	links := s.resourceRepo.FindLinks(resourceID)

	// Objective gate: don't let one subjective click expire a resource. Ask the
	// link-live-checker (when configured AND the resource has links) using the
	// share passcode (Code = 提取码; Password = 解压码 is irrelevant to access).
	//   alive   → verifiably reachable → reject (NOT an error: marked=false).
	//   dead    → every link verified gone → expire (one report suffices).
	//   unknown → unsupported netdisk / no passcode / rate-limited / checker
	//             down / mixed → fall through to the legacy single-report flow.
	verdict := ""
	if s.linkChecker != nil && len(links) > 0 {
		verdict = string(s.linkChecker.CheckShare(ctx, links, row.Code))
		if verdict == string(linkcheck.StatusAlive) {
			return &dto.ReportExpireResult{Verdict: verdict, Marked: false}, nil
		}
	}

	preview := ""
	if len(links) > 0 {
		preview = truncate(links[0], constants.TextPreviewLength)
	}

	txErr := s.resourceRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.resourceRepo.UpdateStatus(tx, resourceID, 1); err != nil {
			return err
		}
		s.helpers.CreateGalgameMessageWithContent(
			tx, userID, row.UserID, "expired", preview, row.GalgameID,
		)
		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("更新失败")
	}
	return &dto.ReportExpireResult{Verdict: verdict, Marked: true}, nil
}

// ──────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────

func (s *ResourceService) fetchWikiBriefs(
	ctx context.Context,
	galgameIDs []int,
) map[int]client.GalgameBrief {
	if len(galgameIDs) == 0 {
		return map[int]client.GalgameBrief{}
	}
	briefMap, _ := s.wikiClient.GetBatch(ctx, galgameIDs)
	if briefMap == nil {
		return map[int]client.GalgameBrief{}
	}
	return briefMap
}

// fetchWikiBriefsPublic is the SFW-aware variant — for public list paths
// that must honour content_limit per docs/galgame_wiki/00-handbook §16.
// The unfiltered fetchWikiBriefs above is kept for internal call sites
// where the caller already knows the IDs are safe to show (e.g.,
// detail-page-internal lookups by ID the user already navigated to).
func (s *ResourceService) fetchWikiBriefsPublic(
	ctx context.Context,
	galgameIDs []int,
	isSFW bool,
) map[int]client.GalgameBrief {
	if len(galgameIDs) == 0 {
		return map[int]client.GalgameBrief{}
	}
	briefMap, _ := s.wikiClient.GetBatchPublic(ctx, galgameIDs, isSFW)
	if briefMap == nil {
		return map[int]client.GalgameBrief{}
	}
	return briefMap
}

func (s *ResourceService) buildGalgameSummary(
	ctx context.Context,
	galgameID int,
) dto.ResourceGalgameSummary {
	summary := dto.ResourceGalgameSummary{
		ID:       galgameID,
		Platform: []string{}, Language: []string{}, Type: []string{},
	}

	briefMap := s.fetchWikiBriefs(ctx, []int{galgameID})
	b, ok := briefMap[galgameID]
	if !ok {
		return summary
	}

	aggs := s.resourceRepo.AggregateByGalgame(galgameID)
	platforms, languages, types := collectAggregate(aggs)
	localView := s.resourceRepo.FindGalgameView(galgameID)

	return dto.ResourceGalgameSummary{
		ID:                       b.ID,
		Name:                     briefToName(b),
		Banner:                   b.Banner,
		EffectiveBannerHash:      b.EffectiveBannerHash,
		EffectiveBannerURL:       b.EffectiveBannerURL,
		EffectiveBannerWidth:     b.EffectiveBannerWidth,
		EffectiveBannerHeight:    b.EffectiveBannerHeight,
		EffectiveBannerThumbhash: b.EffectiveBannerThumbhash,
		ContentLimit:             b.ContentLimit,
		View:                     localView,
		ResourceUpdateTime:       b.ResourceUpdateTime,
		OriginalLanguage:         b.OriginalLanguage,
		AgeLimit:                 b.AgeLimit,
		Platform:                 platforms,
		Language:                 languages,
		Type:                     types,
	}
}

func (s *ResourceService) buildRecommendations(
	ctx context.Context,
	rows []model.GalgameResourceRow,
	galgameID int,
	currentUserID int,
) []dto.ResourceCard {
	userIDs := make([]int, len(rows))
	resourceIDs := make([]int, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
		resourceIDs[i] = r.ID
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)
	briefMap := s.fetchWikiBriefs(ctx, []int{galgameID})
	likedSet := s.resourceRepo.FindLikedSet(currentUserID, resourceIDs)

	cards := make([]dto.ResourceCard, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		card := rowToCard(r, u, likedSet[r.ID])
		if b, ok := briefMap[galgameID]; ok {
			card.GalgameName = briefToName(b)
		}
		cards = append(cards, card)
	}
	return cards
}
