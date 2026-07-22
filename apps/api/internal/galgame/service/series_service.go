package service

import (
	"context"
	"strconv"

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
	// NSFW gating MUST be reflected as content_limit (single source of truth; §16).
	// On the LIST this makes galgame_count reflect the right rating: without it an
	// all-NSFW series reports count 0 in NSFW mode, so the count-gated backfill
	// below never runs and the series shows empty / looks SFW. SFW → sfw; NSFW →
	// all. The per-series backfill + GetDetail carry the SAME content_limit.
	contentLimit := "all"
	if isSFW {
		contentLimit = "sfw"
	}
	// /v1 series entities carry the gated galgame_count + created/updated + a
	// capped member preview (with meta). SeriesListV1 pages the whole catalogue.
	entries, appErr := s.galgameClient.SeriesListV1(ctx, contentLimit)
	if appErr != nil {
		return nil, appErr
	}

	// Backfill full members where the capped preview is short of galgame_count, so
	// the sfw-mode count (len(filtered)) + isNSFW reflect every member.
	s.backfillGalgames(ctx, entries, contentLimit)

	items := make([]dto.SeriesListItem, 0, len(entries))
	for i := range entries {
		item := &entries[i]
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

// backfillGalgames fetches the full member list for series whose /v1 list preview
// is INCOMPLETE (capped below galgame_count) — the /v1 series list caps the
// embedded preview (Preload Limit), so any series with more than the cap needs
// the /series/:id fetch to compute an accurate sfw count + isNSFW + 5-cover
// montage. Same content_limit so an all-NSFW series doesn't backfill to zero.
func (s *SeriesService) backfillGalgames(ctx context.Context, entries []client.NextMoeSeriesEntry, contentLimit string) {
	for i := range entries {
		if entries[i].GalgameCount == 0 || len(entries[i].Galgame) >= entries[i].GalgameCount {
			continue
		}
		full, found, appErr := s.galgameClient.SeriesEntryV1(ctx, strconv.Itoa(entries[i].ID), contentLimit)
		if appErr != nil || !found {
			continue
		}
		entries[i].Galgame = full.Galgame
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
	entry, found, appErr := s.galgameClient.SeriesEntryV1(ctx, seriesID, contentLimit)
	if appErr != nil {
		return nil, appErr
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该系列")
	}

	filtered := s.enricher.FilterSFW(entry.Galgame, isSFW)

	return &dto.SeriesDetail{
		ID:            entry.ID,
		Name:          entry.Name,
		Description:   entry.Description,
		IsNSFW:        s.enricher.HasNSFW(filtered),
		SampleGalgame: s.enricher.Samples(filtered, 5),
		GalgameCount:  len(filtered),
		Galgame:       s.enricher.ToCards(ctx, filtered),
		Created:       entry.Created,
		Updated:       entry.Updated,
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
