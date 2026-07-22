package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type SeriesService struct {
	galgameClient *client.GalgameClient
	enricher      *GalgameEnricher
}

func NewSeriesService(galgameClient *client.GalgameClient, enricher *GalgameEnricher) *SeriesService {
	return &SeriesService{galgameClient: galgameClient, enricher: enricher}
}

// ──────────────────────────────────────────
// Galgame response shapes (parsing-only)
// ──────────────────────────────────────────

type nextMoeSeriesListItem struct {
	ID           int                      `json:"id"`
	Name         string                   `json:"name"`
	Description  string                   `json:"description"`
	Galgame      []dto.NextMoeGalgameItem `json:"galgame"`
	GalgameCount int                      `json:"galgame_count"`
	Created      string                   `json:"created"`
	Updated      string                   `json:"updated"`
}

type nextMoeSeriesListResp struct {
	Items []nextMoeSeriesListItem `json:"items"`
	Total int64                   `json:"total"`
}

type nextMoeSeriesDetail struct {
	ID          int                      `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Galgame     []dto.NextMoeGalgameItem `json:"galgame"`
	Created     string                   `json:"created"`
	Updated     string                   `json:"updated"`
}

// ──────────────────────────────────────────
// GetList — GET /galgame-series
// ──────────────────────────────────────────
//
// We pull the whole series set (typically < 200) so we can apply NSFW
// filtering in Go before paginating, which avoids sparse pages when many
// series contain only NSFW titles.

func (s *SeriesService) GetList(
	ctx context.Context,
	req *dto.SeriesListRequest,
	isSFW bool,
) (*dto.SeriesListPage, *errors.AppError) {
	// NSFW gating MUST be reflected as content_limit to the galgame (single source
	// of truth; §16) — it defaults to sfw when omitted. On the LIST this makes
	// galgame_count reflect the right rating: without it an all-NSFW series
	// reports count 0 in NSFW mode, so the count-gated backfill below never runs
	// and the series shows empty / looks SFW. SFW → sfw; NSFW → all. The
	// per-series backfill + GetDetail carry the SAME content_limit.
	contentLimit := "all"
	if isSFW {
		contentLimit = "sfw"
	}
	query := url.Values{
		"page": {"1"}, "limit": {"500"}, "include": {"galgame"},
		"content_limit": {contentLimit},
	}
	data, appErr := s.galgameClient.Get(ctx, "/series", query)
	if appErr != nil {
		return nil, appErr
	}

	var parsed nextMoeSeriesListResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}

	// Backfill galgame arrays where the list endpoint omitted them — with the
	// SAME content_limit, or the per-series /series/:id fetch defaults to sfw and
	// an all-NSFW series backfills to ZERO games (the NSFW-mode bug).
	s.backfillGalgames(ctx, parsed.Items, contentLimit)

	items := make([]dto.SeriesListItem, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		filtered := s.enricher.FilterSFW(item.Galgame, isSFW)
		if isSFW && len(filtered) == 0 {
			continue
		}

		count := item.GalgameCount
		if isSFW {
			count = len(filtered)
		}

		items = append(items, dto.SeriesListItem{
			ID:            item.ID,
			Name:          item.Name,
			Description:   item.Description,
			IsNSFW:        s.enricher.HasNSFW(filtered),
			SampleGalgame: s.enricher.Samples(filtered, 5),
			GalgameCount:  count,
			Created:       item.Created,
			Updated:       item.Updated,
		})
	}

	total := int64(len(items))
	items = paginate(items, req.Page, req.Limit)
	return &dto.SeriesListPage{Series: items, Total: total}, nil
}

// backfillGalgames fetches the full galgame list for series whose included
// galgame slice is INCOMPLETE (fewer than galgame_count). The galgame list endpoint
// returns sparse galgame arrays — usually empty, but sometimes PARTIAL (e.g. 2
// of 14, as for series 51). The old "only when empty" check left those partial
// series stuck at their handful of games, so the card's 5-cover montage showed
// far fewer covers than the series has. Backfilling whenever len < count fixes
// both the empty and the partial case.
func (s *SeriesService) backfillGalgames(ctx context.Context, items []nextMoeSeriesListItem, contentLimit string) {
	q := url.Values{"content_limit": {contentLimit}}
	for i := range items {
		if items[i].GalgameCount == 0 || len(items[i].Galgame) >= items[i].GalgameCount {
			continue
		}
		data, err := s.galgameClient.Get(ctx, fmt.Sprintf("/series/%d", items[i].ID), q)
		if err != nil {
			continue
		}
		var parsed struct {
			Galgame []dto.NextMoeGalgameItem `json:"galgame"`
		}
		if json.Unmarshal(data, &parsed) == nil {
			items[i].Galgame = parsed.Galgame
		}
	}
}

// ──────────────────────────────────────────
// GetDetail — GET /galgame-series/:id
// ──────────────────────────────────────────

func (s *SeriesService) GetDetail(
	ctx context.Context,
	seriesID string,
	isSFW bool,
) (*dto.SeriesDetail, *errors.AppError) {
	// Same content_limit gating — without it /series/:id defaults to sfw, so the
	// detail page of an NSFW series shows no games even in NSFW mode.
	contentLimit := "all"
	if isSFW {
		contentLimit = "sfw"
	}
	data, appErr := s.galgameClient.Get(ctx, "/series/"+seriesID, url.Values{"content_limit": {contentLimit}})
	if appErr != nil {
		return nil, appErr
	}

	var parsed nextMoeSeriesDetail
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}

	filtered := s.enricher.FilterSFW(parsed.Galgame, isSFW)

	return &dto.SeriesDetail{
		ID:            parsed.ID,
		Name:          parsed.Name,
		Description:   parsed.Description,
		IsNSFW:        s.enricher.HasNSFW(filtered),
		SampleGalgame: s.enricher.Samples(filtered, 5),
		GalgameCount:  len(filtered),
		Galgame:       s.enricher.ToCards(ctx, filtered),
		Created:       parsed.Created,
		Updated:       parsed.Updated,
	}, nil
}

// paginate slices `items` according to page/limit, clamping out-of-range.
func paginate[T any](items []T, page, limit int) []T {
	start := (page - 1) * limit
	if start >= len(items) {
		return []T{}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}
