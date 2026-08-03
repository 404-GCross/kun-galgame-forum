package service

import (
	"context"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

// SeriesService serves the series detail lane off the catalog series facet
// (catalog_series + catalog_series_member).
//
// A series is the one grouping entity a reader arrives at from a game rather
// than from an index — "这游戏属于哪个系列, 还有哪几部" — so the page it lands on
// is the same shape as the other three entity pages: the forum-LOCAL subset of
// the series' member works, filterable and sortable like /galgame itself.
type SeriesService struct {
	galgameClient *client.GalgameClient
	// galgameSvc runs the shared local filter/sort/paginate + hydration flow
	// over the series' member ids (the catalog cannot filter by kungal-local
	// resource data). Same arrangement as EngineService.
	galgameSvc *GalgameService
}

func NewSeriesService(galgameClient *client.GalgameClient, galgameSvc *GalgameService) *SeriesService {
	return &SeriesService{galgameClient: galgameClient, galgameSvc: galgameSvc}
}

// GetDetail — GET /galgame-series/:id (id = a catalog SERIES id)
func (s *SeriesService) GetDetail(
	ctx context.Context,
	id string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.SeriesDetail, *errors.AppError) {
	rec, found, appErr := s.galgameClient.CatalogSeries(ctx, id)
	if appErr != nil {
		return nil, appErr
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该系列")
	}

	memberIDs, appErr := s.galgameClient.CatalogMemberGIDs(ctx,
		url.Values{"series_id": {id}}, isSFW, taxonomyMemberPageCap)
	if appErr != nil {
		return nil, appErr
	}
	page, appErr := s.galgameSvc.hydrateListCards(ctx, buildEntityFilter(rawQuery, memberIDs), isSFW)
	if appErr != nil {
		return nil, appErr
	}

	return &dto.SeriesDetail{
		ID:          int(rec.ID),
		Name:        rec.DisplayName,
		Description: seriesIntro(rec),
		Galgame:     listCardsToEntityCards(page.Galgames),
		// The gated page's own total, never the upstream member count: that one
		// counts the series' whole catalogue, published here or not.
		GalgameCount: page.Total,
	}, nil
}

// seriesIntro picks the blurb to render under the title.
//
// The catalog does NOT merge series intros to one row per language — a
// hand-written rescue and a source's own text both survive — so this takes the
// first row of the reader's preferred language rather than concatenating them:
// two descriptions of the same series stacked on one page reads as a bug.
// Chinese first, then Japanese, then whatever exists; empty is fine, the
// header renders without a description.
func seriesIntro(rec *client.CatalogSeriesDetail) string {
	for _, lang := range []string{"zh-Hans", "zh-Hant", "ja", "en"} {
		for _, in := range rec.Intros {
			if in.Lang == lang && in.Intro != "" {
				return in.Intro
			}
		}
	}
	for _, in := range rec.Intros {
		if in.Intro != "" {
			return in.Intro
		}
	}
	return ""
}
