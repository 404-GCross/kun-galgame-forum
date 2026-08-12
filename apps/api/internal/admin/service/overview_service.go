package service

import (
	"context"
	"sort"
	"time"

	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/repository"
	"kun-galgame-api/pkg/errors"
)

type OverviewService struct {
	overviewRepo *repository.OverviewRepository
}

func NewOverviewService(
	overviewRepo *repository.OverviewRepository,
) *OverviewService {
	return &OverviewService{overviewRepo: overviewRepo}
}

type localModel struct {
	Name, Table, Label, FeedType string
}

func localModels() []localModel {
	return []localModel{
		{Name: "topic", Table: "topic", Label: "话题"},
		{Name: "topic_reply", Table: "topic_reply", Label: "话题回复"},
		{Name: "topic_comment", Table: "topic_comment", Label: "话题评论"},
		{Name: "galgame", Table: "galgame", Label: "Galgame"},
		{Name: "galgame_resource", Table: "galgame_resource", Label: "Galgame 资源"},
		{Name: "galgame_comment", Label: "Galgame 评论", FeedType: "GALGAME_COMMENT_CREATION"},
		{Name: "galgame_website", Table: "galgame_website", Label: "Galgame 网站"},
		{Name: "galgame_website_comment", Label: "Galgame 网站评论", FeedType: "GALGAME_WEBSITE_COMMENT_CREATION"},
		{Name: "chat_message", Table: "chat_message", Label: "聊天消息"},
	}
}

func (s *OverviewService) GetOverview(ctx context.Context) ([]dto.OverviewItem, *errors.AppError) {
	locals := localModels()

	items := make([]dto.OverviewItem, 0, len(locals))
	for _, m := range locals {
		var (
			count int64
			err   error
		)
		if m.FeedType != "" {
			count, err = s.overviewRepo.CountFeedType(m.FeedType)
		} else {
			count, err = s.overviewRepo.CountTable(m.Table)
		}
		if err != nil {
			return nil, errors.ErrInternal("获取统计概览失败")
		}
		items = append(items, dto.OverviewItem{
			Name:  m.Name,
			Label: m.Label,
			Count: count,
		})
	}

	return items, nil
}

func (s *OverviewService) GetStats(ctx context.Context, days int) ([]dto.DailyStatRow, *errors.AppError) {
	if days == 0 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		since = since.In(loc)
	}
	since = time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, since.Location())

	locals := localModels()

	dateMap := make(map[string]map[string]int64)

	for _, t := range locals {
		var (
			stats []repository.DailyStat
			err   error
		)
		if t.FeedType != "" {
			stats, err = s.overviewRepo.DailyFeedCountsSince(t.FeedType, since)
		} else {
			stats, err = s.overviewRepo.DailyCountsSince(t.Table, since)
		}
		if err != nil {
			return nil, errors.ErrInternal("获取统计数据失败")
		}
		for _, row := range stats {
			if dateMap[row.Date] == nil {
				dateMap[row.Date] = make(map[string]int64)
			}
			dateMap[row.Date][t.Name] = row.Count
		}
	}

	allKeys := make([]string, 0, len(locals))
	for _, t := range locals {
		allKeys = append(allKeys, t.Name)
	}

	dates := make([]string, 0, len(dateMap))
	for d := range dateMap {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	result := make([]dto.DailyStatRow, len(dates))
	for i, d := range dates {
		row := dto.DailyStatRow{"date": d}
		for _, key := range allKeys {
			row[key] = dateMap[d][key]
		}
		result[i] = row
	}

	return result, nil
}
