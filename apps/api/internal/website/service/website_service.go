package service

import (
	"context"
	"encoding/json"
	"net/url"

	"kun-galgame-api/internal/website/dto"
	"kun-galgame-api/internal/website/model"
	"kun-galgame-api/internal/website/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

// marshalDomain produces a jsonb-compatible payload for the website's
// alternate-domain list. The column is a jsonb default '[]', so we
// always emit a JSON array (never null) — nil input becomes "[]" so
// downstream readers don't crash on json.Unmarshal.
func marshalDomain(domains []string) json.RawMessage {
	if len(domains) == 0 {
		return json.RawMessage("[]")
	}
	b, err := json.Marshal(domains)
	if err != nil {
		return json.RawMessage("[]")
	}
	return json.RawMessage(b)
}

type WebsiteService struct {
	websiteRepo  *repository.WebsiteRepository
	categoryRepo *repository.CategoryRepository
	tagRepo      *repository.TagRepository
	commentRepo  *repository.CommentRepository
	userClient   *userclient.Client
	// cdnBase resolves a stored icon_image_hash into a full CDN URL on read.
	cdnBase string
}

func NewWebsiteService(
	websiteRepo *repository.WebsiteRepository,
	categoryRepo *repository.CategoryRepository,
	tagRepo *repository.TagRepository,
	commentRepo *repository.CommentRepository,
	userClient *userclient.Client,
	cdnBase string,
) *WebsiteService {
	return &WebsiteService{
		websiteRepo:  websiteRepo,
		categoryRepo: categoryRepo,
		tagRepo:      tagRepo,
		commentRepo:  commentRepo,
		userClient:   userClient,
		cdnBase:      cdnBase,
	}
}

// ──────────────────────────────────────────
// GetList — GET /website
// ──────────────────────────────────────────

func (s *WebsiteService) GetList(isSFW bool) []dto.WebsiteCard {
	rows := s.websiteRepo.FindAll(isSFW)
	catMap := s.categoryRepo.FindNamesByIDs(collectCategoryIDs(rows))
	levelMap := s.tagRepo.LevelSumsAll()
	return websiteCardsFromRows(rows, catMap, levelMap, s.cdnBase)
}

// ──────────────────────────────────────────
// Create — POST /website
// ──────────────────────────────────────────

func (s *WebsiteService) Create(userID int, req *dto.CreateWebsiteRequest) *errors.AppError {
	// Domain parse is left in place for parity with the old handler (unused).
	_, _ = url.Parse(req.URL)

	txErr := s.websiteRepo.DB().Transaction(func(tx *gorm.DB) error {
		website := model.GalgameWebsite{
			Name:          req.Name,
			URL:           req.URL,
			Description:   req.Description,
			Icon:          req.Icon,
			IconImageHash: req.IconImageHash,
			Language:      req.Language,
			AgeLimit:      req.AgeLimit,
			CategoryID:    req.CategoryID,
			UserID:        userID,
			CreateTime:    req.CreateTime,
			Domain:        marshalDomain(req.Domain),
		}
		if err := s.websiteRepo.Create(tx, &website); err != nil {
			return err
		}
		s.tagRepo.InsertWebsiteTagRelations(tx, website.ID, req.TagIDs)
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("创建网站失败")
	}
	return nil
}

// ──────────────────────────────────────────
// GetDetail — GET /website/:domain
// ──────────────────────────────────────────

func (s *WebsiteService) GetDetail(
	ctx context.Context,
	domain string,
	currentUserID int,
) (*dto.WebsiteDetailResponse, *errors.AppError) {
	website, err := s.websiteRepo.FindByDomain(domain)
	if err != nil {
		return nil, errors.ErrNotFound("未找到该网站")
	}

	go s.websiteRepo.IncrementView(website.ID)

	category, _ := s.categoryRepo.FindByID(website.CategoryID)
	catBrief := dto.WebsiteCategoryBrief{}
	if category != nil {
		catBrief = dto.WebsiteCategoryBrief{
			ID:          category.ID,
			Name:        category.Name,
			Label:       category.Label,
			Description: category.Description,
		}
	}

	rels := s.tagRepo.FindRelationsByWebsiteWithTag(website.ID)
	tags := make([]dto.WebsiteTagBrief, len(rels))
	for i, tr := range rels {
		tags[i] = dto.WebsiteTagBrief{
			ID:          tr.Tag.ID,
			Name:        tr.Tag.Name,
			Description: tr.Tag.Description,
			Label:       tr.Tag.Label,
			Level:       tr.Tag.Level,
		}
	}

	detailComments := s.commentRepo.FindByWebsiteForDetail(website.ID)
	uids := userclient.CollectIDs(detailComments, func(r repository.DetailCommentRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)
	commentList := make([]dto.WebsiteDetailComment, 0, len(detailComments))
	for _, cm := range detailComments {
		u := userMap[cm.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		commentList = append(commentList, dto.WebsiteDetailComment{
			ID:      cm.ID,
			Content: cm.Content,
			User: dto.UserBriefCompact{
				ID: u.ID, Name: u.Name, Avatar: u.Avatar,
			},
			Created: cm.Created,
			Updated: cm.Updated,
		})
	}

	isLiked, isFavorited := false, false
	if currentUserID > 0 {
		isLiked = s.websiteRepo.HasLike(currentUserID, website.ID)
		isFavorited = s.websiteRepo.HasFavorite(currentUserID, website.ID)
	}

	return &dto.WebsiteDetailResponse{
		ID:            website.ID,
		Name:          website.Name,
		URL:           website.URL,
		Description:   website.Description,
		Icon:          website.Icon,
		IconImageHash: website.IconImageHash,
		IconURL:       imageclient.ResolveURL(s.cdnBase, website.IconImageHash, website.Icon),
		View:          website.View,
		Language:      website.Language,
		AgeLimit:      website.AgeLimit,
		Category:      catBrief,
		Tags:          tags,
		LikeCount:     website.LikeCount,
		IsLiked:       isLiked,
		FavoriteCount: website.FavoriteCount,
		IsFavorited:   isFavorited,
		Domain:        website.Domain,
		CreateTime:    website.CreateTime,
		Comment:       commentList,
		Created:       website.CreatedAt,
		Updated:       website.UpdatedAt,
	}, nil
}

// ──────────────────────────────────────────
// Update — PUT /website/:domain
// ──────────────────────────────────────────

func (s *WebsiteService) Update(req *dto.UpdateWebsiteRequest) *errors.AppError {
	txErr := s.websiteRepo.DB().Transaction(func(tx *gorm.DB) error {
		s.websiteRepo.UpdateFields(tx, req.WebsiteID, map[string]any{
			"name":            req.Name,
			"url":             req.URL,
			"description":     req.Description,
			"icon":            req.Icon,
			"icon_image_hash": req.IconImageHash,
			"category_id":     req.CategoryID,
			"age_limit":       req.AgeLimit,
			"language":        req.Language,
			"create_time":     req.CreateTime,
			"domain":          marshalDomain(req.Domain),
		})
		s.tagRepo.ReplaceWebsiteTagRelations(tx, req.WebsiteID, req.TagIDs)
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("更新网站失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Delete — DELETE /website/:domain
// ──────────────────────────────────────────

func (s *WebsiteService) Delete(websiteID int) *errors.AppError {
	if err := s.websiteRepo.DeleteByID(websiteID); err != nil {
		return errors.ErrInternal("删除网站失败")
	}
	return nil
}

// ──────────────────────────────────────────
// Interactions — PUT /website/:domain/{like,favorite}
// ──────────────────────────────────────────

func (s *WebsiteService) ToggleLike(userID, websiteID int) *errors.AppError {
	s.websiteRepo.DB().Transaction(func(tx *gorm.DB) error {
		existing, err := s.websiteRepo.FindLike(tx, userID, websiteID)
		if err == gorm.ErrRecordNotFound {
			s.websiteRepo.CreateLike(tx, userID, websiteID)
			s.websiteRepo.AdjustLikeCount(tx, websiteID, 1)
		} else if err == nil && existing != nil {
			s.websiteRepo.DeleteLike(tx, existing)
			s.websiteRepo.AdjustLikeCount(tx, websiteID, -1)
		}
		return nil
	})
	return nil
}

func (s *WebsiteService) ToggleFavorite(userID, websiteID int) *errors.AppError {
	s.websiteRepo.DB().Transaction(func(tx *gorm.DB) error {
		existing, err := s.websiteRepo.FindFavorite(tx, userID, websiteID)
		if err == gorm.ErrRecordNotFound {
			s.websiteRepo.CreateFavorite(tx, userID, websiteID)
			s.websiteRepo.AdjustFavoriteCount(tx, websiteID, 1)
		} else if err == nil && existing != nil {
			s.websiteRepo.DeleteFavorite(tx, existing)
			s.websiteRepo.AdjustFavoriteCount(tx, websiteID, -1)
		}
		return nil
	})
	return nil
}
