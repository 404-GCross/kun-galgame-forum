package service

import (
	"context"
	"encoding/json"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type TagService struct {
	wikiClient *client.GalgameClient
	enricher   *GalgameEnricher
}

func NewTagService(wikiClient *client.GalgameClient, enricher *GalgameEnricher) *TagService {
	return &TagService{wikiClient: wikiClient, enricher: enricher}
}

type wikiTagListItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	GalgameCount int    `json:"galgame_count"`
}

type wikiTagListResp struct {
	Items []wikiTagListItem `json:"items"`
	Total int64             `json:"total"`
}

type wikiTagDetail struct {
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	Category    string          `json:"category"`
	Description string          `json:"description"`
	Alias       []dto.WikiAlias `json:"alias"`
}

type wikiTagDetailResp struct {
	Tag      wikiTagDetail         `json:"tag"`
	Galgames []dto.WikiGalgameItem `json:"galgames"`
	Total    int64                 `json:"total"`
}

// wikiMultiTagResp is the {items, total} shape wiki returns for /tag/multi.
type wikiMultiTagResp struct {
	Items []dto.WikiGalgameItem `json:"items"`
	Total int64                 `json:"total"`
}

// TagMultiPage is the enriched response for GET /galgame-tag/multi.
type TagMultiPage struct {
	Galgames []dto.GalgameCard `json:"galgames"`
	Total    int64             `json:"total"`
}

// Search — GET /galgame-tag/search
//
// Wiki search is Meilisearch-backed; response shape is
// `{items, total, processing_time_ms}`. The frontend
// (galgame/tag/Container.vue) does `searchResult.value = res` expecting a
// bare TagListItem[]. We unwrap `items` here so the gateway response stays
// compatible without touching the frontend.
//
// SFW filter: drop tags with category="sexual" (matches the convention in
// GetList line 155). Forwarding content_limit=sfw to wiki additionally
// hides tags whose galgame attachments are NSFW-only.
func (s *TagService) Search(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) ([]dto.TagListItem, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/tag/search", withSFWFilter(rawQuery, isSFW))
	if appErr != nil {
		return nil, appErr
	}
	var resp struct {
		Items []wikiTagListItem `json:"items"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}
	items := make([]dto.TagListItem, 0, len(resp.Items))
	for _, t := range resp.Items {
		if isSFW && t.Category == "sexual" {
			continue
		}
		items = append(items, dto.TagListItem{
			ID: t.ID, Name: t.Name, Category: t.Category,
			GalgameCount: t.GalgameCount,
		})
	}
	return items, nil
}

// GetByMultiTag — GET /galgame-tag/multi
//
// Proxies to the wiki /tag/multi endpoint with param renamed to the
// snake_case the wiki expects and content_limit forwarded in SFW mode,
// then enriches the resulting galgames with local like counts etc.
func (s *TagService) GetByMultiTag(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*TagMultiPage, *errors.AppError) {
	q := withSFWFilter(rawQuery, isSFW)
	// tagIds → tag_ids (wiki uses snake_case)
	if v := q.Get("tagIds"); v != "" {
		q.Del("tagIds")
		q.Set("tag_ids", v)
	}

	data, appErr := s.wikiClient.Get(ctx, "/tag/multi", q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed wikiMultiTagResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	filtered := s.enricher.FilterSFW(parsed.Items, isSFW)
	return &TagMultiPage{
		Galgames: s.enricher.ToCards(ctx, filtered),
		Total:    parsed.Total,
	}, nil
}

// GetList — GET /galgame-tag
//
// The wiki gates NSFW + paginates server-side: GET /tag accepts content_limit
// (sfw hides the sexual/NSFW tag category, all returns everything; safe default
// sfw — docs 04-taxonomy). We forward page + limit + content_limit and proxy
// the wiki's total for a true server-side paged list.
//
// (Was: limit=5000 fetch-all + client-side sexual filter — silently truncated
// by the wiki's 100/page cap, so most tags were unlisted and the pager never
// showed; client-side gating is also forbidden by handbook §16.)
func (s *TagService) GetList(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*dto.TagListPage, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/tag", withSFWFilter(rawQuery, isSFW))
	if appErr != nil {
		return nil, appErr
	}
	var parsed wikiTagListResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	tags := make([]dto.TagListItem, 0, len(parsed.Items))
	for _, t := range parsed.Items {
		tags = append(tags, dto.TagListItem{
			ID: t.ID, Name: t.Name, Category: t.Category,
			GalgameCount: t.GalgameCount,
		})
	}
	return &dto.TagListPage{Tags: tags, Total: parsed.Total}, nil
}

// GetDetail — GET /galgame-tag/:name
//
// In SFW mode we forward content_limit=sfw to the wiki so it filters the
// galgame list server-side and returns a matching total; the local
// FilterSFW call is kept as a defensive net.
func (s *TagService) GetDetail(
	ctx context.Context,
	name string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.TagDetail, *errors.AppError) {
	// Full wiki member catalogue (paginated, content_limit-aware, matching total),
	// NOT the forum-local subset. The wiki /tag/:id response already carries the
	// galgame page; the enricher overlays forum-local data + IsOnForum per card
	// (wiki-only games → IsOnForum=false). See OfficialService.GetDetail.
	q := withSFWFilter(rawQuery, isSFW)
	data, appErr := s.wikiClient.Get(ctx, "/tag/"+name, q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed wikiTagDetailResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	t := parsed.Tag

	return &dto.TagDetail{
		ID:           t.ID,
		Name:         t.Name,
		Category:     t.Category,
		Description:  t.Description,
		Alias:        aliasesToNames(t.Alias),
		Galgame:      s.enricher.ToCards(ctx, parsed.Galgames),
		GalgameCount: parsed.Total,
	}, nil
}
