package service

import (
	"context"
	"log/slog"
	"strconv"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

const MaxCollectionsPerUser = 100

const previewCoversPerCollection = 4

type CollectionService struct {
	collectionRepo *repository.GalgameCollectionRepository
	galgameService *GalgameService
	galgameClient  *client.GalgameClient
	userClient     *userclient.Client
	check          *gate.CheckService
	scan           *gate.ScanService
	helpers        InteractionHelpers
}

func NewCollectionService(
	collectionRepo *repository.GalgameCollectionRepository,
	galgameService *GalgameService,
	galgameClient *client.GalgameClient,
	userClient *userclient.Client,
	check *gate.CheckService,
	scan *gate.ScanService,
) *CollectionService {
	return &CollectionService{
		collectionRepo: collectionRepo,
		galgameService: galgameService,
		galgameClient:  galgameClient,
		userClient:     userClient,
		check:          check,
		scan:           scan,
	}
}

func (s *CollectionService) Create(ctx context.Context, userID int, req *dto.CreateCollectionRequest) (int, *errors.AppError) {
	count, err := s.collectionRepo.CountByUser(userID)
	if err != nil {
		return 0, errors.ErrInternal("读取收藏夹数量失败")
	}
	if count >= MaxCollectionsPerUser {
		return 0, errors.ErrBadRequest("收藏夹数量已达上限")
	}

	moderationText := gate.ComposeText(req.Name, req.Description)
	authorID := int64(userID)
	decision, matched := s.check.Decision(ctx, moderationText, &authorID)
	if decision == gate.DecisionDeny {
		return 0, gate.ErrContentBlocked()
	}

	c := &model.GalgameCollection{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
	}
	txErr := s.collectionRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.collectionRepo.Create(tx, c); err != nil {
			return err
		}
		if c.Visibility == model.CollectionRestricted {
			return s.collectionRepo.ReplaceViewers(tx, c.ID, sanitizeViewerIDs(req.ViewerIDs, userID))
		}
		return nil
	})
	if txErr != nil {
		return 0, errors.ErrInternal("创建收藏夹失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameCollection, "subject_id", c.ID, "author_id", userID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindGalgameCollection, strconv.Itoa(c.ID), moderationText, int64(userID))
	return c.ID, nil
}

func (s *CollectionService) Update(ctx context.Context, userID int, canEditAny bool, cid int, req *dto.UpdateCollectionRequest) *errors.AppError {
	c, err := s.loadForMutation(cid, userID, canEditAny)
	if err != nil {
		return err
	}
	ownerID := c.UserID

	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.Description != nil {
		c.Description = *req.Description
	}
	if req.Visibility != nil {
		c.Visibility = *req.Visibility
	}

	moderationText := gate.ComposeText(c.Name, c.Description)
	authorID := int64(ownerID)
	decision, matched := s.check.Decision(ctx, moderationText, &authorID)
	if decision == gate.DecisionDeny {
		return gate.ErrContentBlocked()
	}

	txErr := s.collectionRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.collectionRepo.Save(tx, c); err != nil {
			return err
		}
		switch {
		case c.Visibility != model.CollectionRestricted:
			return s.collectionRepo.ReplaceViewers(tx, cid, nil)
		case req.ViewerIDs != nil:
			return s.collectionRepo.ReplaceViewers(tx, cid, sanitizeViewerIDs(*req.ViewerIDs, ownerID))
		default:
			return nil
		}
	})
	if txErr != nil {
		return errors.ErrInternal("更新收藏夹失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameCollection, "subject_id", cid, "author_id", ownerID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindGalgameCollection, strconv.Itoa(cid), moderationText, int64(ownerID))
	return nil
}

func (s *CollectionService) loadForMutation(cid, userID int, canMutateAny bool) (*model.GalgameCollection, *errors.AppError) {
	var (
		c   *model.GalgameCollection
		err error
	)
	if canMutateAny {
		c, err = s.collectionRepo.GetByID(cid)
	} else {
		c, err = s.collectionRepo.GetByIDForUser(cid, userID)
	}
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrNotFound("收藏夹不存在")
		}
		return nil, errors.ErrInternal("读取收藏夹失败")
	}
	return c, nil
}

func (s *CollectionService) Delete(userID int, canDeleteAny bool, cid int) *errors.AppError {
	c, appErr := s.loadForMutation(cid, userID, canDeleteAny)
	if appErr != nil {
		return appErr
	}
	if c.IsDefault {
		return errors.ErrForbidden("默认收藏夹不能删除")
	}
	ownerID := c.UserID

	txErr := s.collectionRepo.DB().Transaction(func(tx *gorm.DB) error {
		affected, err := s.collectionRepo.GalgamesOnlyInCollection(tx, cid, ownerID)
		if err != nil {
			return err
		}
		if err := s.collectionRepo.DecrementFavoriteCounts(tx, affected); err != nil {
			return err
		}
		n, err := s.collectionRepo.DeleteForUser(tx, cid, ownerID)
		if err != nil {
			return err
		}
		if n == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if txErr != nil {
		if txErr == gorm.ErrRecordNotFound {
			return errors.ErrNotFound("收藏夹不存在")
		}
		return errors.ErrInternal("删除收藏夹失败")
	}
	return nil
}

func (s *CollectionService) SetMembership(ctx context.Context, userID, galgameID int, targetIDs []int) *errors.AppError {
	targetIDs = dedupInts(targetIDs)
	if !s.collectionRepo.OwnsAll(userID, targetIDs) {
		return errors.ErrForbidden("收藏夹不存在或不属于您")
	}

	ownerID, name := s.galgameService.fetchOwnerAndName(ctx, galgameID)

	txErr := s.collectionRepo.DB().Transaction(func(tx *gorm.DB) error {
		current, err := s.collectionRepo.UserCollectionIDsForGalgame(tx, userID, galgameID)
		if err != nil {
			return err
		}
		toAdd, toRemove := diffIntSets(targetIDs, current)

		for _, cid := range toAdd {
			if err := s.collectionRepo.EnsureGalgameLocal(tx, galgameID); err != nil {
				return err
			}
			if err := s.collectionRepo.AddItem(tx, cid, galgameID, userID); err != nil {
				return err
			}
		}
		for _, cid := range toRemove {
			if err := s.collectionRepo.RemoveItem(tx, cid, galgameID); err != nil {
				return err
			}
		}

		firstAdd := len(current) == 0 && len(targetIDs) > 0
		lastRemove := len(current) > 0 && len(targetIDs) == 0

		if firstAdd {
			if err := s.collectionRepo.AdjustGalgameFavoriteCount(tx, galgameID, 1); err != nil {
				return err
			}
		} else if lastRemove {
			if err := s.collectionRepo.AdjustGalgameFavoriteCount(tx, galgameID, -1); err != nil {
				return err
			}
		}

		if ownerID != 0 && ownerID != userID {
			if firstAdd {
				s.helpers.AdjustMoemoepoint(tx, ownerID, 1,
					moemoepoint.ReasonLiked, moemoepoint.Ref("galgame", galgameID))
				if err := s.helpers.CreateGalgameMessageWithContent(tx, userID, ownerID, "favorite", name, galgameID); err != nil {
					return err
				}
			} else if lastRemove {
				s.helpers.AdjustMoemoepoint(tx, ownerID, -1,
					moemoepoint.ReasonLiked, moemoepoint.Ref("galgame", galgameID))
			}
		}
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("更新收藏失败")
	}
	return nil
}

func (s *CollectionService) GetMyCollectionsForGalgame(userID, galgameID int) ([]dto.MyCollectionForGalgame, *errors.AppError) {
	count, err := s.collectionRepo.CountByUser(userID)
	if err != nil {
		return nil, errors.ErrInternal("读取收藏夹失败")
	}
	if count == 0 {
		s.ensureDefault(userID)
	}

	cols, err := s.collectionRepo.ListAllByUser(userID)
	if err != nil {
		return nil, errors.ErrInternal("读取收藏夹失败")
	}
	containing, err := s.collectionRepo.UserCollectionIDsForGalgame(s.collectionRepo.DB(), userID, galgameID)
	if err != nil {
		return nil, errors.ErrInternal("读取收藏状态失败")
	}
	inSet := toIntSet(containing)

	out := make([]dto.MyCollectionForGalgame, 0, len(cols))
	for _, c := range cols {
		out = append(out, dto.MyCollectionForGalgame{
			ID:         c.ID,
			Name:       c.Name,
			Visibility: c.Visibility,
			IsDefault:  c.IsDefault,
			ItemCount:  c.ItemCount,
			Contains:   inSet[c.ID],
		})
	}
	return out, nil
}

func (s *CollectionService) ensureDefault(userID int) {
	c := &model.GalgameCollection{
		UserID:      userID,
		Name:        "",
		Description: "",
		Visibility:  model.CollectionPublic,
		IsDefault:   true,
	}
	_ = s.collectionRepo.DB().Transaction(func(tx *gorm.DB) error {
		return s.collectionRepo.Create(tx, c)
	})
}

func (s *CollectionService) GetDetail(ctx context.Context, viewerID, cid, page, limit int, isSFW bool) (*dto.CollectionDetail, *errors.AppError) {
	c, err := s.collectionRepo.GetByID(cid)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrNotFound("收藏夹不存在")
		}
		return nil, errors.ErrInternal("读取收藏夹失败")
	}
	if !s.canView(c, viewerID) {
		return nil, errors.ErrNotFound("收藏夹不存在")
	}

	owner, _, _ := s.userClient.User(ctx, c.UserID)
	if !userclient.IsRenderable(owner) && viewerID != c.UserID {
		return nil, errors.ErrNotFound("收藏夹不存在")
	}

	ids, total, err := s.collectionRepo.ListItemGalgameIDs(cid, page, limit)
	if err != nil {
		return nil, errors.ErrInternal("读取收藏夹内容失败")
	}
	cards, appErr := s.galgameService.HydrateCardsByIDs(ctx, ids, isSFW)
	if appErr != nil {
		cards = []dto.GalgameListCard{}
	}

	isOwner := viewerID == c.UserID
	viewers := []dto.UserBrief{}
	if isOwner && c.Visibility == model.CollectionRestricted {
		viewerIDs, _ := s.collectionRepo.ListViewerIDs(cid)
		if len(viewerIDs) > 0 {
			m := s.userClient.Hydrate(ctx, viewerIDs)
			for _, vid := range viewerIDs {
				u := m[vid]
				viewers = append(viewers, dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar})
			}
		}
	}

	return &dto.CollectionDetail{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		Visibility:  c.Visibility,
		IsDefault:   c.IsDefault,
		ItemCount:   c.ItemCount,
		IsOwner:     isOwner,
		Owner:       dto.UserBrief{ID: owner.ID, Name: owner.Name, Avatar: owner.Avatar},
		Viewers:     viewers,
		Galgames:    cards,
		Total:       total,
		Created:     c.CreatedAt,
		Updated:     c.UpdatedAt,
	}, nil
}

func (s *CollectionService) ListForUser(ctx context.Context, ownerID, viewerID, page, limit int, isSFW bool) ([]dto.CollectionSummary, int64, *errors.AppError) {
	owner, _, _ := s.userClient.User(ctx, ownerID)
	if !userclient.IsRenderable(owner) && viewerID != ownerID {
		return []dto.CollectionSummary{}, 0, nil
	}

	cols, total, err := s.collectionRepo.ListForOwnerVisible(ownerID, viewerID, page, limit)
	if err != nil {
		return nil, 0, errors.ErrInternal("读取收藏夹列表失败")
	}
	if len(cols) == 0 {
		return []dto.CollectionSummary{}, total, nil
	}

	coverByGid := s.resolvePreviewCovers(ctx, cols, isSFW)

	out := make([]dto.CollectionSummary, 0, len(cols))
	for _, c := range cols {
		out = append(out, dto.CollectionSummary{
			ID:            c.ID,
			Name:          c.Name,
			Description:   c.Description,
			Visibility:    c.Visibility,
			IsDefault:     c.IsDefault,
			ItemCount:     c.ItemCount,
			PreviewCovers: coverByGid[c.ID],
			Created:       c.CreatedAt,
			Updated:       c.UpdatedAt,
		})
	}
	return out, total, nil
}

func (s *CollectionService) resolvePreviewCovers(ctx context.Context, cols []model.GalgameCollection, isSFW bool) map[int][]string {
	result := make(map[int][]string, len(cols))
	colIDs := make([]int, len(cols))
	for i, c := range cols {
		colIDs[i] = c.ID
		result[c.ID] = []string{}
	}

	previewMap, err := s.collectionRepo.PreviewGalgameIDs(colIDs, previewCoversPerCollection)
	if err != nil || len(previewMap) == 0 {
		return result
	}

	allGids := []int{}
	for _, gids := range previewMap {
		allGids = append(allGids, gids...)
	}
	briefMap, appErr := s.galgameClient.GetBatchPublic(ctx, dedupInts(allGids), isSFW)
	if appErr != nil {
		return result
	}

	for cid, gids := range previewMap {
		covers := make([]string, 0, len(gids))
		for _, gid := range gids {
			b, ok := briefMap[gid]
			if !ok {
				continue
			}
			if b.EffectiveBannerURL != "" {
				covers = append(covers, b.EffectiveBannerURL)
			}
		}
		result[cid] = covers
	}
	return result
}

func (s *CollectionService) canView(c *model.GalgameCollection, viewerID int) bool {
	if viewerID == c.UserID {
		return true
	}
	switch c.Visibility {
	case model.CollectionPublic:
		return true
	case model.CollectionRestricted:
		return s.collectionRepo.IsViewer(c.ID, viewerID)
	default:
		return false
	}
}

func dedupInts(in []int) []int {
	seen := make(map[int]struct{}, len(in))
	out := make([]int, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func toIntSet(in []int) map[int]bool {
	m := make(map[int]bool, len(in))
	for _, v := range in {
		m[v] = true
	}
	return m
}

func diffIntSets(target, current []int) (toAdd, toRemove []int) {
	cur := toIntSet(current)
	tgt := toIntSet(target)
	for _, v := range target {
		if !cur[v] {
			toAdd = append(toAdd, v)
		}
	}
	for _, v := range current {
		if !tgt[v] {
			toRemove = append(toRemove, v)
		}
	}
	return
}

func sanitizeViewerIDs(ids []int, ownerID int) []int {
	out := make([]int, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id == ownerID || id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
