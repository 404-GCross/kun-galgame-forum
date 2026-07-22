package service

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

// enginePageLimit is the /v1 engines list page size (the curated engine list is
// page/limit paginated, capped at 100 — the bridge's unpaginated bare array does
// not carry over). GetList walks every page to reproduce the full list the FE
// engine index expects.
const enginePageLimit = 100

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

// GetList — GET /galgame-engine
func (s *EngineService) GetList(ctx context.Context) ([]dto.EngineListItem, *errors.AppError) {
	// /v1 engines is page/limit paginated ({items,total}); walk every page and
	// concatenate to reproduce the full bare list the FE engine index consumes.
	var engines []nextMoeEngineListItem
	for page := 1; ; page++ {
		q := url.Values{
			"page":  {strconv.Itoa(page)},
			"limit": {strconv.Itoa(enginePageLimit)},
		}
		data, appErr := s.galgameClient.GetV1(ctx, "/galgame/engines", q)
		if appErr != nil {
			return nil, appErr
		}
		var resp struct {
			Items []nextMoeEngineListItem `json:"items"`
			Total int64                   `json:"total"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, errors.ErrInternal("解析 Galgame 响应失败")
		}
		engines = append(engines, resp.Items...)
		if len(resp.Items) == 0 || int64(len(engines)) >= resp.Total {
			break
		}
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
	// /v1 engine entity is by-id (the :name segment carries the numeric id) and
	// needs no query params — the curated record has name/description/alias/
	// galgame_count. The kungal member list is recomputed locally below.
	data, appErr := s.galgameClient.GetV1(ctx, "/galgame/engines/"+name, nil)
	if appErr != nil {
		return nil, appErr
	}
	var e nextMoeEngineDetail
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}

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
