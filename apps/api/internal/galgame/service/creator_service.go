package service

import (
	"context"
	"encoding/json"

	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/role"
	"kun-galgame-api/pkg/userclient"
)

const (
	creatorMinMergedPRs   = 5
	creatorMinGalgames    = 10
	creatorMinReviews     = 5
	creatorReviewMinLen   = 100
	creatorMinMoemoepoint = 2000
	creatorSource         = "forum"
)

type CreatorEligibility struct {
	Eligible          bool  `json:"eligible"`
	MergedPRs         int64 `json:"merged_prs"`
	GalgamesPublished int64 `json:"galgames_published"`
	Reviews100        int64 `json:"reviews_100"`
	Moemoepoint       int64 `json:"moemoepoint"`
	NeedMergedPRs     int   `json:"need_merged_prs"`
	NeedGalgames      int   `json:"need_galgames"`
	NeedReviews       int   `json:"need_reviews"`
	NeedMoemoepoint   int   `json:"need_moemoepoint"`
}

type CreatorService struct {
	ratingRepo *repository.RatingRepository
	stats      *GalgameUserStatsService
	userClient *userclient.Client
}

func NewCreatorService(ratingRepo *repository.RatingRepository, stats *GalgameUserStatsService, userClient *userclient.Client) *CreatorService {
	return &CreatorService{ratingRepo: ratingRepo, stats: stats, userClient: userClient}
}

func (s *CreatorService) eligibility(ctx context.Context, userID int) (*CreatorEligibility, *errors.AppError) {
	stats := s.stats.Stats(ctx, int64(userID))
	reviews, rErr := s.ratingRepo.CountReviewsWithMinLength(userID, creatorReviewMinLen)
	if rErr != nil {
		return nil, errors.ErrInternal("统计简评失败")
	}
	moe, _ := s.userClient.GetMoemoepoint(ctx, userID)
	e := &CreatorEligibility{
		MergedPRs:         stats.MergedEdits,
		GalgamesPublished: stats.Published,
		Reviews100:        reviews,
		Moemoepoint:       int64(moe),
		NeedMergedPRs:     creatorMinMergedPRs,
		NeedGalgames:      creatorMinGalgames,
		NeedReviews:       creatorMinReviews,
		NeedMoemoepoint:   creatorMinMoemoepoint,
	}
	e.Eligible = e.MergedPRs >= creatorMinMergedPRs ||
		e.GalgamesPublished >= creatorMinGalgames ||
		e.Reviews100 >= creatorMinReviews ||
		e.Moemoepoint >= creatorMinMoemoepoint
	return e, nil
}

func (s *CreatorService) Status(ctx context.Context, userID int, token string) (*CreatorEligibility, *userclient.CreatorApplication, bool, *errors.AppError) {
	e, appErr := s.eligibility(ctx, userID)
	if appErr != nil {
		return nil, nil, false, appErr
	}
	app, err := s.userClient.GetMyCreatorApplication(ctx, token)
	if err != nil {
		return nil, nil, false, errors.ErrInternal("获取申请状态失败")
	}
	isCreator := false
	if u, ok, uErr := s.userClient.User(ctx, userID); ok && uErr == nil {
		isCreator = role.IsCreator(u.Roles)
	}
	return e, app, isCreator, nil
}

func (s *CreatorService) Apply(ctx context.Context, userID int, token, message string) (*userclient.CreatorApplication, *errors.AppError) {
	e, appErr := s.eligibility(ctx, userID)
	if appErr != nil {
		return nil, appErr
	}
	if !e.Eligible {
		return nil, errors.ErrForbidden("尚不满足创作者申请条件")
	}
	evidence, _ := json.Marshal(map[string]any{
		"merged_prs":         e.MergedPRs,
		"galgames_published": e.GalgamesPublished,
		"reviews_100":        e.Reviews100,
		"moemoepoint":        e.Moemoepoint,
	})
	app, err := s.userClient.CreateCreatorApplication(ctx, token, creatorSource, evidence, message)
	if err != nil {
		if oe, ok := err.(*userclient.OAuthError); ok {
			return nil, errors.ErrBadRequest(oe.Message)
		}
		return nil, errors.ErrInternal("提交申请失败")
	}
	return app, nil
}
