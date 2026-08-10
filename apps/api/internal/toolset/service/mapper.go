package service

import (
	"math"

	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/pkg/userclient"
)

func toolsetCardFromRow(
	t model.GalgameToolset,
	userMap map[int]userclient.User,
	avgMap map[int]float64,
	dlMap map[int]int,
	ccMap map[int]int,
) dto.ToolsetCard {
	var practicalityAvg any
	if avg, ok := avgMap[t.ID]; ok && avg != 0 {
		practicalityAvg = math.Round(avg*100) / 100
	} else {
		practicalityAvg = nil
	}

	return dto.ToolsetCard{
		ID:                 t.ID,
		Name:               t.Name,
		User:               userBriefFromClient(userMap[t.UserID]),
		Type:               t.Type,
		Platform:           t.Platform,
		Language:           t.Language,
		Version:            t.Version,
		View:               t.View,
		Download:           dlMap[t.ID],
		CommentCount:       ccMap[t.ID],
		PracticalityAvg:    practicalityAvg,
		ResourceUpdateTime: t.ResourceUpdateTime,
	}
}

func allowedSortField(raw string) string {
	allowed := map[string]bool{
		"created":              true,
		"view":                 true,
		"name":                 true,
		"resource_update_time": true,
	}
	if raw != "" && allowed[raw] {
		return raw
	}
	return "created"
}

func sortOrder(raw string) string {
	if raw == "asc" {
		return "ASC"
	}
	return "DESC"
}
