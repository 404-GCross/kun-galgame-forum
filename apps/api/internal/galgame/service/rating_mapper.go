package service

import (
	"encoding/json"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/pkg/userclient"
)

func rawJSON(s string) json.RawMessage {
	if s == "" {
		return json.RawMessage("[]")
	}
	return json.RawMessage(s)
}

func rowToScores(r model.GalgameRatingRow) dto.RatingScores {
	return dto.RatingScores{
		Art: r.Art, Story: r.Story, Music: r.Music, Character: r.Character,
		Route: r.Route, System: r.System, Voice: r.Voice,
		ReplayValue: r.ReplayValue,
	}
}

func ratingRowToCard(
	r model.GalgameRatingRow,
	user userclient.User,
	brief client.GalgameBrief,
) dto.RatingCard {
	return dto.RatingCard{
		ID:           r.ID,
		User:         userBriefToDTO(user),
		Recommend:    r.Recommend,
		Overall:      r.Overall,
		View:         r.View,
		GalgameType:  rawJSON(r.GalgameType),
		PlayStatus:   r.PlayStatus,
		ShortSummary: r.ShortSummary,
		SpoilerLevel: r.SpoilerLevel,
		RatingScores: rowToScores(r),
		LikeCount:    r.LikeCount,
		Created:      r.Created,
		Updated:      r.Updated,
		Galgame: dto.RatingGalgameBrief{
			ID:           brief.ID,
			ContentLimit: brief.ContentLimit,
			Name:         brief.Name,
		},
	}
}

func nextMoeOfficialsToDTO(rels []dto.NextMoeOfficialRel) []dto.RatingOfficial {
	out := make([]dto.RatingOfficial, len(rels))
	for i, rel := range rels {
		alias := make([]string, len(rel.Official.Alias))
		for j, a := range rel.Official.Alias {
			alias[j] = a.Name
		}
		out[i] = dto.RatingOfficial{
			ID:           rel.Official.ID,
			Name:         rel.Official.Name,
			Link:         rel.Official.Link,
			Category:     rel.Official.Category,
			Lang:         rel.Official.Lang,
			Alias:        alias,
			GalgameCount: rel.Official.GalgameCount,
		}
	}
	return out
}

func containsInt(haystack []int, needle int) bool {
	if needle == 0 {
		return false
	}
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
