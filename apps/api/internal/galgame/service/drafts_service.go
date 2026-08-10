package service

import (
	"context"
	"net/url"
	"strconv"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type DraftsService struct {
	galgameClient *client.GalgameClient
	enricher      *GalgameEnricher
}

func NewDraftsService(galgameClient *client.GalgameClient, enricher *GalgameEnricher) *DraftsService {
	return &DraftsService{galgameClient: galgameClient, enricher: enricher}
}

type DraftFilters struct {
	LabelID           int
	TagID             int
	EngineID          int
	SeriesID          int
	OriginalLanguages string
}

func (s *DraftsService) GetDrafts(
	ctx context.Context,
	page, limit int,
	f DraftFilters,
) (*dto.DraftsPage, *errors.AppError) {
	q := url.Values{
		"claimed": {"false"},
		"page":    {strconv.Itoa(page)},
		"limit":   {strconv.Itoa(limit)},
		"include": {CatalogCardInclude},
		"sort":    {"released_desc"},
	}
	if f.LabelID > 0 {
		q.Set("label_id", strconv.Itoa(f.LabelID))
	}
	if f.TagID > 0 {
		q.Set("tag_id", strconv.Itoa(f.TagID))
	}
	if f.EngineID > 0 {
		q.Set("engine_id", strconv.Itoa(f.EngineID))
	}
	if f.SeriesID > 0 {
		q.Set("series_id", strconv.Itoa(f.SeriesID))
	}
	if f.OriginalLanguages != "" {
		q.Set("olang", f.OriginalLanguages)
	}
	client.OpenPopulation(q)

	res, appErr := s.galgameClient.CatalogWorksSearch(ctx, q)
	if appErr != nil {
		return nil, appErr
	}
	return &dto.DraftsPage{
		Items: s.enricher.ToCards(ctx, catalogItemsToNextMoe(res.Items)),
		Total: res.Total,
	}, nil
}
