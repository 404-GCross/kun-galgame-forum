package service

import (
	"context"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/ranking/dto"
	"kun-galgame-api/internal/ranking/repository"
	"kun-galgame-api/pkg/userclient"
)

type RankingService struct {
	repo          *repository.RankingRepository
	galgameClient *client.GalgameClient
	userClient    *userclient.Client
}

func NewRankingService(
	repo *repository.RankingRepository,
	gc *client.GalgameClient,
	userClient *userclient.Client,
) *RankingService {
	return &RankingService{repo: repo, galgameClient: gc, userClient: userClient}
}

func (s *RankingService) GetGalgameRanking(
	ctx context.Context, req *dto.GalgameRankingRequest, isSFW bool,
) []dto.GalgameRankingItem {
	rows := s.repo.FindGalgameLocal(req.SortField, req.SortOrder, req.Page, req.Limit, req.ShowNoResource)
	if len(rows) == 0 {
		return []dto.GalgameRankingItem{}
	}

	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	briefMap, appErr := s.galgameClient.GetBatchPublic(ctx, ids, isSFW)
	if appErr != nil {
		return []dto.GalgameRankingItem{}
	}

	userMap := s.userClient.Hydrate(ctx, userclient.CollectIDs(rows,
		func(r repository.GalgameLocalRow) int { return userclient.DerefID(r.CreatorUserID) }))

	items := make([]dto.GalgameRankingItem, 0, len(rows))
	for _, r := range rows {
		b, ok := briefMap[r.ID]
		if !ok {
			continue
		}
		u := userMap[userclient.DerefID(r.CreatorUserID)]
		items = append(items, dto.GalgameRankingItem{
			ID: r.ID,
			Name: dto.LocaleName{
				EnUS: b.NameEnUs, JaJP: b.NameJaJp,
				ZhCN: b.NameZhCn, ZhTW: b.NameZhTw,
			},
			User:                dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Banner:              b.Banner,
			Value:               r.Value,
			SortField:           req.SortField,
			EffectiveBannerHash: b.EffectiveBannerHash,
			EffectiveBannerURL:  b.EffectiveBannerURL,
		})
	}
	return items
}

func (s *RankingService) GetTopicRanking(ctx context.Context, req *dto.TopicRankingRequest, isSFW bool) []dto.TopicRankingItem {
	rows := s.repo.FindTopicRanking(req.SortField, req.SortOrder, req.Page, req.Limit, isSFW)
	uids := userclient.CollectIDs(rows, func(r repository.TopicRankingRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	items := make([]dto.TopicRankingItem, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		items = append(items, dto.TopicRankingItem{
			ID:        r.ID,
			Title:     r.Title,
			User:      dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Value:     r.Value,
			SortField: req.SortField,
		})
	}
	return items
}

func (s *RankingService) GetUserRanking(ctx context.Context, req *dto.UserRankingRequest) []dto.UserRankingItem {
	rows := s.repo.FindUserRanking(req.SortField, req.SortOrder, req.Page, req.Limit)
	uids := userclient.CollectIDs(rows, func(r repository.UserRankingRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	items := make([]dto.UserRankingItem, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		items = append(items, dto.UserRankingItem{
			ID: u.ID, Name: u.Name, Avatar: u.Avatar,
			Bio: u.Bio, Value: r.Value,
			SortField: req.SortField,
		})
	}
	return items
}
