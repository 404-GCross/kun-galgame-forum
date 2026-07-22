package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/client"
	galgameDto "kun-galgame-api/internal/galgame/dto"
	galgameService "kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/search/dto"
	"kun-galgame-api/internal/search/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
)

type SearchService struct {
	repo          *repository.SearchRepository
	galgameClient *client.GalgameClient
	enricher      *galgameService.GalgameEnricher
	userClient    *userclient.Client
}

func NewSearchService(
	repo *repository.SearchRepository,
	galgameClient *client.GalgameClient,
	enricher *galgameService.GalgameEnricher,
	userClient *userclient.Client,
) *SearchService {
	return &SearchService{repo: repo, galgameClient: galgameClient, enricher: enricher, userClient: userClient}
}

// tokenize splits a keyword string into trimmed non-empty tokens.
// Returns an error if the result is empty.
func tokenize(raw string) ([]string, *errors.AppError) {
	keywords := strings.Fields(strings.TrimSpace(raw))
	if len(keywords) == 0 {
		return nil, errors.ErrBadRequest("搜索关键词不能为空")
	}
	return keywords, nil
}

// SearchTopics returns topic search results. Identity is hydrated from OAuth
// via userclient; rows authored by banned users are dropped.
func (s *SearchService) SearchTopics(ctx context.Context, raw string, page, limit int) (*dto.PaginatedResult[dto.TopicItem], *errors.AppError) {
	keywords, appErr := tokenize(raw)
	if appErr != nil {
		return nil, appErr
	}
	rows, total := s.repo.SearchTopics(keywords, page, limit)

	uids := userclient.CollectIDs(rows, func(r repository.TopicRow) int { return r.UserID })
	userMap := s.userClient.Hydrate(ctx, uids)

	topicIDs := make([]int, len(rows))
	for i, r := range rows {
		topicIDs[i] = r.ID
	}
	sectionMap := map[int][]string{}
	for _, sct := range s.repo.FindTopicSections(topicIDs) {
		sectionMap[sct.TopicID] = append(sectionMap[sct.TopicID], sct.SectionName)
	}
	pollSet := s.repo.FindTopicIDsWithPoll(topicIDs)

	items := make([]dto.TopicItem, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		sections := sectionMap[r.ID]
		if sections == nil {
			sections = []string{}
		}
		items = append(items, dto.TopicItem{
			ID: r.ID, Title: r.Title, View: r.View, Status: r.Status,
			LikeCount: r.LikeCount, ReplyCount: r.ReplyCount,
			CommentCount:     r.CommentCount,
			HasBestAnswer:    r.BestAnswerID != nil,
			IsPollTopic:      pollSet[r.ID],
			IsNSFWTopic:      r.IsNSFW,
			Section:          sections,
			UpvoteTime:       r.UpvoteTime,
			StatusUpdateTime: r.StatusUpdateTime,
			User:             dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
		})
	}
	return &dto.PaginatedResult[dto.TopicItem]{Items: items, Total: total}, nil
}

// SearchUsers returns user search results from the OAuth /users/search
// endpoint. Identity is OAuth-owned, so we don't run any local DB query
// here. moemoepoint and created are populated from kungal_user_state
// keyed by the OAuth user ids that come back.
//
// Banned users (status != 0) are filtered out at the gateway: per the
// agreed policy, kungal hides their content rather than rendering a
// "this user is banned" placeholder in search results.
func (s *SearchService) SearchUsers(
	ctx context.Context,
	raw string,
	page, limit int,
) (*dto.PaginatedResult[dto.UserItem], *errors.AppError) {
	if _, appErr := tokenize(raw); appErr != nil {
		return nil, appErr
	}
	if s.userClient == nil {
		return nil, errors.ErrInternal("用户搜索未启用")
	}

	// OAuth /users/search is a top-N suggestion endpoint with no total or
	// real pagination; we ignore `page` and use len(result) as the total.
	_ = page
	users, err := s.userClient.SearchUsers(ctx, raw, limit)
	if err != nil {
		return nil, errors.ErrInternal(fmt.Sprintf("用户搜索失败: %v", err))
	}

	items := make([]dto.UserItem, 0, len(users))
	for _, u := range users {
		if u.Status != 0 {
			continue // hide banned users from search results
		}
		items = append(items, dto.UserItem{
			ID:     u.ID,
			Name:   u.Name,
			Avatar: u.Avatar,
			Bio:    u.Bio,
			// Moemoepoint / Created come from kungal_user_state if you
			// want them — left zero for now since /search?type=user is a
			// list view that just renders name+avatar+bio cards.
		})
	}
	return &dto.PaginatedResult[dto.UserItem]{Items: items, Total: int64(len(items))}, nil
}

// SearchReplies returns reply search results. Identity hydrated from OAuth;
// rows authored by banned users are dropped.
func (s *SearchService) SearchReplies(ctx context.Context, raw string, page, limit int) (*dto.PaginatedResult[dto.ReplyItem], *errors.AppError) {
	keywords, appErr := tokenize(raw)
	if appErr != nil {
		return nil, appErr
	}
	rows, total := s.repo.SearchReplies(keywords, page, limit)

	uids := userclient.CollectIDs(rows, func(r repository.ReplyRow) int { return r.UserID })
	// Also hydrate the parent-topic authors so a reply under a banned author's
	// (now-hidden) topic can be dropped too.
	for _, r := range rows {
		if r.TopicUserID > 0 {
			uids = append(uids, r.TopicUserID)
		}
	}
	userMap := s.userClient.Hydrate(ctx, uids)

	items := make([]dto.ReplyItem, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		// Drop when the PARENT topic's author is banned: that topic is hidden
		// (its detail 404s), so this hit is a dead link that would still leak
		// the banned author's title as context.
		if r.TopicUserID > 0 && !userclient.IsRenderable(userMap[r.TopicUserID]) {
			continue
		}
		items = append(items, dto.ReplyItem{
			ID: r.ID, TopicID: r.TopicID, TopicTitle: r.TopicTitle,
			Content: r.Content, Floor: r.Floor,
			User:    dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Created: r.Created,
		})
	}
	return &dto.PaginatedResult[dto.ReplyItem]{Items: items, Total: total}, nil
}

// SearchGalgames returns galgame search results from the galgame Meilisearch
// index, enriched with local interaction counts. Uses the galgame `fields`
// parameter so the heavy `intro_*` markdown isn't sent over the wire on
// list pages — saves hundreds of KB per request.
func (s *SearchService) SearchGalgames(
	ctx context.Context,
	raw string,
	page, limit int,
	isSFW bool,
) (*dto.PaginatedResult[galgameDto.GalgameCard], *errors.AppError) {
	if _, appErr := tokenize(raw); appErr != nil {
		return nil, appErr
	}
	if s.galgameClient == nil || s.enricher == nil {
		return nil, errors.ErrInternal("Galgame 搜索未启用")
	}

	q := url.Values{}
	q.Set("q", raw)
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	// /v1 thin item + include=meta carries every scalar the enricher reads (the
	// heavy intro_* / taxonomy name arrays are never on the thin item, so the old
	// `fields=` bandwidth trim is now inherent to the curated shape). Content
	// limit MUST be explicit on BOTH branches: /v1 search defaults to sfw when
	// omitted, so NSFW mode has to say `all` aloud (needs the key's galgame:nsfw
	// scope) or NSFW hits silently disappear.
	q.Set("include", "meta")
	if isSFW {
		q.Set("content_limit", "sfw")
	} else {
		q.Set("content_limit", "all")
	}

	data, appErr := s.galgameClient.GetV1(ctx, "/galgame/search", q)
	if appErr != nil {
		return nil, appErr
	}

	var resp struct {
		Items []client.V1Item `json:"items"`
		Total int64           `json:"total"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, errors.ErrInternal(fmt.Sprintf("解析 Galgame 搜索响应失败: %v", err))
	}
	items := make([]galgameDto.NextMoeGalgameItem, len(resp.Items))
	for i := range resp.Items {
		items[i] = client.V1ItemToNextMoeItem(&resp.Items[i])
	}

	// Defensive: if galgame ignores the SFW filter we still strip NSFW here.
	filtered := s.enricher.FilterSFW(items, isSFW)
	cards := s.enricher.ToCards(ctx, filtered)

	return &dto.PaginatedResult[galgameDto.GalgameCard]{
		Items: cards,
		Total: resp.Total,
	}, nil
}

// SearchComments returns comment search results. Identity hydrated from OAuth;
// rows authored by banned users are dropped.
func (s *SearchService) SearchComments(ctx context.Context, raw string, page, limit int) (*dto.PaginatedResult[dto.CommentItem], *errors.AppError) {
	keywords, appErr := tokenize(raw)
	if appErr != nil {
		return nil, appErr
	}
	rows, total := s.repo.SearchComments(keywords, page, limit)

	uids := userclient.CollectIDs(rows, func(r repository.CommentRow) int { return r.UserID })
	// Also hydrate the parent-topic authors so a comment under a banned
	// author's (now-hidden) topic can be dropped too.
	for _, r := range rows {
		if r.TopicUserID > 0 {
			uids = append(uids, r.TopicUserID)
		}
	}
	userMap := s.userClient.Hydrate(ctx, uids)

	items := make([]dto.CommentItem, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		// Drop when the PARENT topic's author is banned: that topic is hidden
		// (its detail 404s), so this hit is a dead link that would still leak
		// the banned author's title as context.
		if r.TopicUserID > 0 && !userclient.IsRenderable(userMap[r.TopicUserID]) {
			continue
		}
		items = append(items, dto.CommentItem{
			ID: r.ID, TopicID: r.TopicID, TopicTitle: r.TopicTitle,
			Content: r.Content,
			User:    dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
			Created: r.Created,
		})
	}
	return &dto.PaginatedResult[dto.CommentItem]{Items: items, Total: total}, nil
}
