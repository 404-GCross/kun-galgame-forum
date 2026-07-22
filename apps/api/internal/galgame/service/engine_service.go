package service

import (
	"context"
	"encoding/json"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type EngineService struct {
	galgameClient *client.GalgameClient
	// galgameSvc runs the shared local filter/sort/paginate + hydration flow
	// over the engine's member ids (the galgame can't filter by kungal-local
	// resource data). See GetDetail.
	galgameSvc *GalgameService
}

func NewEngineService(galgameClient *client.GalgameClient, galgameSvc *GalgameService) *EngineService {
	return &EngineService{galgameClient: galgameClient, galgameSvc: galgameSvc}
}

type nextMoeEngineListItem struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgame_count"`
	Created      string   `json:"created"`
	Updated      string   `json:"updated"`
}

type nextMoeEngineDetail struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Alias       []string `json:"alias"`
}

type nextMoeEngineDetailResp struct {
	Engine   nextMoeEngineDetail      `json:"engine"`
	Galgames []dto.NextMoeGalgameItem `json:"galgames"`
	Total    int64                    `json:"total"`
}

// GetList — GET /galgame-engine
func (s *EngineService) GetList(ctx context.Context) ([]dto.EngineListItem, *errors.AppError) {
	data, appErr := s.galgameClient.Get(ctx, "/engine", nil)
	if appErr != nil {
		return nil, appErr
	}
	var engines []nextMoeEngineListItem
	if err := json.Unmarshal(data, &engines); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}

	items := make([]dto.EngineListItem, len(engines))
	for i, e := range engines {
		items[i] = dto.EngineListItem{
			ID:           e.ID,
			Name:         e.Name,
			Description:  e.Description,
			Alias:        emptyStrSliceIfNil(e.Alias),
			GalgameCount: e.GalgameCount,
		}
	}
	return items, nil
}

// GetDetail — GET /galgame-engine/:name
func (s *EngineService) GetDetail(
	ctx context.Context,
	name string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.EngineDetail, *errors.AppError) {
	// Entity detail lists the forum-LOCAL subset of the engine's catalogue, so
	// the kungal filters (类型/语言/平台/作品类型) + every sort work. Only the
	// engine's metadata is used from the galgame here (cheapest page); the galgame
	// list is recomputed locally from the member ids below.
	q := withSFWFilter(rawQuery, isSFW)
	// Resolve by the engine_id QUERY param (the :name path is cosmetic — 04-taxonomy),
	// sourced from the path so the lookup never depends on the FE echoing it.
	q.Set("engine_id", name)
	q.Set("page", "1")
	q.Set("limit", "1")
	data, appErr := s.galgameClient.Get(ctx, "/engine/"+name, q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed nextMoeEngineDetailResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}

	e := parsed.Engine

	// Member ids from the galgame, then the SAME local filter/sort/paginate as
	// /galgame over them (RestrictIDs). Un-ingested members drop out naturally.
	memberIDs, appErr := s.galgameClient.EntityGalgameIDs(ctx, "engine", e.ID)
	if appErr != nil {
		return nil, appErr
	}
	page, appErr := s.galgameSvc.hydrateListCards(ctx, buildEntityFilter(rawQuery, memberIDs), isSFW)
	if appErr != nil {
		return nil, appErr
	}

	return &dto.EngineDetail{
		ID:           e.ID,
		Name:         e.Name,
		Description:  e.Description,
		Alias:        emptyStrSliceIfNil(e.Alias),
		Galgame:      listCardsToEntityCards(page.Galgames),
		GalgameCount: page.Total,
	}, nil
}

func emptyStrSliceIfNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
