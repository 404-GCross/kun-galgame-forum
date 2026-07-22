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
	galgameClient *client.GalgameClient
	// enricher hydrates the /tag list + /tag/multi results (galgame catalogue).
	enricher *GalgameEnricher
	// galgameSvc runs the shared local filter/sort/paginate + hydration flow
	// over a tag's member ids for the detail page. See GetDetail.
	galgameSvc *GalgameService
}

func NewTagService(galgameClient *client.GalgameClient, enricher *GalgameEnricher, galgameSvc *GalgameService) *TagService {
	return &TagService{galgameClient: galgameClient, enricher: enricher, galgameSvc: galgameSvc}
}

type nextMoeTagListItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	GalgameCount int    `json:"galgame_count"`
}

type nextMoeTagListResp struct {
	Items []nextMoeTagListItem `json:"items"`
	Total int64                `json:"total"`
}

// TagMultiPage is the enriched response for GET /galgame-tag/multi.
type TagMultiPage struct {
	Galgames []dto.GalgameCard `json:"galgames"`
	Total    int64             `json:"total"`
}

// Search — GET /galgame-tag/search
//
// Galgame search is Meilisearch-backed; response shape is
// `{items, total, processing_time_ms}`. The frontend
// (galgame/tag/Container.vue) does `searchResult.value = res` expecting a
// bare TagListItem[]. We unwrap `items` here so the gateway response stays
// compatible without touching the frontend.
//
// SFW filter: drop tags with category="sexual" (matches the convention in
// GetList line 155). Forwarding content_limit=sfw to galgame additionally
// hides tags whose galgame attachments are NSFW-only.
func (s *TagService) Search(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) ([]dto.TagListItem, *errors.AppError) {
	data, appErr := s.galgameClient.GetV1(ctx, "/galgame/tags/search", withSFWFilter(rawQuery, isSFW))
	if appErr != nil {
		return nil, appErr
	}
	var resp struct {
		Items []nextMoeTagListItem `json:"items"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
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
// Proxies to the galgame /tag/multi endpoint with param renamed to the
// snake_case the galgame expects and content_limit forwarded in SFW mode,
// then enriches the resulting galgames with local like counts etc.
func (s *TagService) GetByMultiTag(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*TagMultiPage, *errors.AppError) {
	src := withSFWFilter(rawQuery, isSFW)
	// FE sends tagIds (camelCase); /v1 tags/multi takes `ids`. include=meta (W1d)
	// so the embedded thin items carry content_limit for FilterSFW + user_id for
	// the enricher.
	ids := src.Get("tagIds")
	if ids == "" {
		ids = src.Get("tag_ids")
	}
	q := url.Values{"content_limit": {src.Get("content_limit")}, "include": {"meta"}}
	if ids != "" {
		q.Set("ids", ids)
	}

	data, appErr := s.galgameClient.GetV1(ctx, "/galgame/tags/multi", q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed struct {
		Items []client.V1Item `json:"items"`
		Total int64           `json:"total"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}
	items := make([]dto.NextMoeGalgameItem, 0, len(parsed.Items))
	for i := range parsed.Items {
		items = append(items, client.V1ItemToNextMoeItem(&parsed.Items[i]))
	}

	filtered := s.enricher.FilterSFW(items, isSFW)
	return &TagMultiPage{
		Galgames: s.enricher.ToCards(ctx, filtered),
		Total:    parsed.Total,
	}, nil
}

// GetList — GET /galgame-tag
//
// The galgame gates NSFW + paginates server-side: GET /tag accepts content_limit
// (sfw hides the sexual/NSFW tag category, all returns everything; safe default
// sfw — docs 04-taxonomy). We forward page + limit + content_limit and proxy
// the galgame's total for a true server-side paged list.
//
// (Was: limit=5000 fetch-all + client-side sexual filter — silently truncated
// by the galgame's 100/page cap, so most tags were unlisted and the pager never
// showed; client-side gating is also forbidden by handbook §16.)
func (s *TagService) GetList(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*dto.TagListPage, *errors.AppError) {
	data, appErr := s.galgameClient.GetV1(ctx, "/galgame/tags", withSFWFilter(rawQuery, isSFW))
	if appErr != nil {
		return nil, appErr
	}
	var parsed nextMoeTagListResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
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
// Entity detail lists the forum-LOCAL subset of the tag's catalogue, so the
// kungal filters (类型/语言/平台/作品类型) + every sort work. Only the tag's
// metadata is used from the galgame here (cheapest page); the galgame list is
// recomputed locally from the member ids below.
func (s *TagService) GetDetail(
	ctx context.Context,
	name string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.TagDetail, *errors.AppError) {
	// /v1 tag entity is by-id (the :name segment carries the numeric id) and needs
	// no query params — the curated record has the full metadata + aliases (as a
	// plain []string). The kungal member list is recomputed locally below.
	data, appErr := s.galgameClient.GetV1(ctx, "/galgame/tags/"+name, nil)
	if appErr != nil {
		return nil, appErr
	}
	var t v1TagEntity
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}

	// Member ids from the galgame, then the SAME local filter/sort/paginate as
	// /galgame over them (RestrictIDs). Un-ingested members drop out naturally.
	memberIDs, appErr := s.galgameClient.EntityGalgameIDs(ctx, "tag", t.ID)
	if appErr != nil {
		return nil, appErr
	}
	page, appErr := s.galgameSvc.hydrateListCards(ctx, buildEntityFilter(rawQuery, memberIDs), isSFW)
	if appErr != nil {
		return nil, appErr
	}

	return &dto.TagDetail{
		ID:           t.ID,
		Name:         t.Name,
		Category:     t.Category,
		Description:  t.Description,
		Alias:        emptyStrSliceIfNil(t.Aliases),
		Galgame:      listCardsToEntityCards(page.Galgames),
		GalgameCount: page.Total,
	}, nil
}

// v1TagEntity is the /v1 curated tag record (GET /v1/galgame/tags/{id}); only the
// metadata the entity page renders is typed (the member list is recomputed
// locally). Aliases is a plain name slice ([]string), unlike the bridge's
// alias-row objects.
type v1TagEntity struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
}
