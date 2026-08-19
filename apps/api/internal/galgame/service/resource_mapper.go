package service

import (
	"encoding/json"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/userclient"
)

func decodeProviderNames(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return []string{}
	}
	return out
}

func collectIDs(rows []model.GalgameResourceRow) (galgameIDs, userIDs []int) {
	galgameIDs = make([]int, 0, len(rows))
	userIDs = make([]int, 0, len(rows))
	for _, r := range rows {
		galgameIDs = append(galgameIDs, r.GalgameID)
		userIDs = append(userIDs, r.UserID)
	}
	return
}

func collectAggregate(aggs []model.ResourceAggregate) (platforms, languages, types []string) {
	platforms, languages, types = []string{}, []string{}, []string{}
	for _, a := range aggs {
		if a.Platform != "" {
			platforms = appendUniqueStr(platforms, a.Platform)
		}
		if a.Language != "" {
			languages = appendUniqueStr(languages, a.Language)
		}
		if a.Type != "" {
			types = appendUniqueStr(types, a.Type)
		}
	}
	return
}

func appendUniqueStr(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

func userBriefToDTO(u userclient.User) dto.UserBrief {
	return dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
}

func rowToCard(r model.GalgameResourceRow, u userclient.User, isLiked bool) dto.ResourceCard {
	return dto.ResourceCard{
		ID:            r.ID,
		View:          r.View,
		GalgameID:     r.GalgameID,
		User:          userBriefToDTO(u),
		Type:          r.Type,
		Language:      r.Language,
		Platform:      r.Platform,
		Size:          r.Size,
		Status:        r.Status,
		Download:      r.Download,
		LikeCount:     r.LikeCount,
		IsLiked:       isLiked,
		CommentCount:  r.CommentCount,
		LinkDomain:    "",
		ProviderNames: decodeProviderNames(r.ProviderName),
		Note:          r.Note,
		NoteHtml:      markdown.RenderHardWrap(r.Note),
		Created:       r.Created,
		Edited:        r.Edited,
	}
}

func rowToMeta(
	r model.GalgameResourceRow,
	links []string,
	isLiked bool,
	owner userclient.User,
) dto.ResourceMeta {
	linkDomain := ""
	if len(links) > 0 {
		linkDomain = links[0]
	}
	return dto.ResourceMeta{
		ID:            r.ID,
		View:          r.View,
		GalgameID:     r.GalgameID,
		User:          userBriefToDTO(owner),
		Type:          r.Type,
		Language:      r.Language,
		Platform:      r.Platform,
		Size:          r.Size,
		Status:        r.Status,
		Download:      r.Download,
		LikeCount:     r.LikeCount,
		IsLiked:       isLiked,
		CommentCount:  r.CommentCount,
		LinkDomain:    linkDomain,
		ProviderNames: decodeProviderNames(r.ProviderName),
		Note:          r.Note,
		NoteHtml:      markdown.RenderHardWrap(r.Note),
		Created:       r.Created,
		Edited:        r.Edited,
	}
}

func rowToDownloadDetail(
	r model.GalgameResourceRow,
	links []string,
	isLiked bool,
	owner userclient.User,
) dto.ResourceDownloadDetail {
	return dto.ResourceDownloadDetail{
		ResourceMeta: rowToMeta(r, links, isLiked, owner),
		Link:         links,
		Code:         r.Code,
		Password:     r.Password,
	}
}
