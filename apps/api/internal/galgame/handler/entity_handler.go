package handler

import (
	"net/url"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

// EntityHandler groups the wiki-entity endpoints (series/official/engine/tag).
// All of these proxy to wiki, enrich with local data, and apply NSFW filtering.
type EntityHandler struct {
	seriesService   *service.SeriesService
	officialService *service.OfficialService
	engineService   *service.EngineService
	tagService      *service.TagService
}

func NewEntityHandler(
	series *service.SeriesService,
	official *service.OfficialService,
	engine *service.EngineService,
	tag *service.TagService,
) *EntityHandler {
	return &EntityHandler{
		seriesService:   series,
		officialService: official,
		engineService:   engine,
		tagService:      tag,
	}
}

// ──────────────────────────────────────────
// Series
// ──────────────────────────────────────────

// GetSeriesList — GET /galgame-series
func (h *EntityHandler) GetSeriesList(c fiber.Ctx) error {
	var req dto.SeriesListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.seriesService.GetList(c.Context(), &req, utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// GetSeriesDetail — GET /galgame-series/:id
func (h *EntityHandler) GetSeriesDetail(c fiber.Ctx) error {
	detail, appErr := h.seriesService.GetDetail(c.Context(), c.Params("id"), utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// ──────────────────────────────────────────
// Official
// ──────────────────────────────────────────

// GetOfficialList — GET /galgame-official
func (h *EntityHandler) GetOfficialList(c fiber.Ctx) error {
	page, appErr := h.officialService.GetList(c.Context(), collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// SearchOfficials — GET /galgame-official/search
func (h *EntityHandler) SearchOfficials(c fiber.Ctx) error {
	items, appErr := h.officialService.Search(c.Context(), collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

// GetOfficialDetail — GET /galgame-official/:name
func (h *EntityHandler) GetOfficialDetail(c fiber.Ctx) error {
	detail, appErr := h.officialService.GetDetail(
		c.Context(),
		c.Params("name"),
		collectQueryWithRenames(c, entityDetailRenames("officialId", "official_id")),
		utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// ──────────────────────────────────────────
// Engine
// ──────────────────────────────────────────

// GetEngineList — GET /galgame-engine
func (h *EntityHandler) GetEngineList(c fiber.Ctx) error {
	items, appErr := h.engineService.GetList(c.Context())
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

// GetEngineDetail — GET /galgame-engine/:name
func (h *EntityHandler) GetEngineDetail(c fiber.Ctx) error {
	detail, appErr := h.engineService.GetDetail(
		c.Context(),
		c.Params("name"),
		collectQueryWithRenames(c, entityDetailRenames("engineId", "engine_id")),
		utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// ──────────────────────────────────────────
// Tag
// ──────────────────────────────────────────

// GetTagList — GET /galgame-tag
func (h *EntityHandler) GetTagList(c fiber.Ctx) error {
	page, appErr := h.tagService.GetList(c.Context(), collectQuery(c), utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// SearchTags — GET /galgame-tag/search
func (h *EntityHandler) SearchTags(c fiber.Ctx) error {
	items, appErr := h.tagService.Search(c.Context(), collectQuery(c), utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

// GetMultiTagGalgames — GET /galgame-tag/multi
func (h *EntityHandler) GetMultiTagGalgames(c fiber.Ctx) error {
	page, appErr := h.tagService.GetByMultiTag(c.Context(), collectQuery(c), utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// GetTagDetail — GET /galgame-tag/:name
func (h *EntityHandler) GetTagDetail(c fiber.Ctx) error {
	detail, appErr := h.tagService.GetDetail(
		c.Context(),
		c.Params("name"),
		collectQueryWithRenames(c, entityDetailRenames("tagId", "tag_id")),
		utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// ──────────────────────────────────────────
// Query helpers
// ──────────────────────────────────────────

// collectQuery converts the Fiber request query args into url.Values.
func collectQuery(c fiber.Ctx) url.Values {
	q := make(url.Values)
	c.RequestCtx().QueryArgs().VisitAll(func(key, value []byte) {
		q.Set(string(key), string(value))
	})
	return q
}

// collectQueryWithRenames is like collectQuery but rewrites keys per the
// supplied translation map on the way through. Used to bridge the
// camelCase params the FE sends to the snake_case the wiki expects.
//
// Any key absent from `renames` passes through unchanged.
func collectQueryWithRenames(c fiber.Ctx, renames map[string]string) url.Values {
	q := make(url.Values)
	c.RequestCtx().QueryArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		if to, ok := renames[k]; ok {
			q.Set(to, string(value))
		} else {
			q.Set(k, string(value))
		}
	})
	return q
}

// entityDetailRenames returns the rename map for a wiki-backed entity detail
// handler (tag / official / engine): only the FE's per-entity ID key →
// snake_case so the wiki metadata lookup resolves the entity.
//
// sortField/sortOrder are deliberately NOT renamed/normalized: these pages now
// filter+sort LOCALLY (the wiki's galgames are discarded), and list_repo speaks
// the FE sort vocabulary (time/view/rating/…). The raw FE sortField must reach
// buildEntityFilter unchanged — translating it to the wiki's vocabulary here
// would make list_repo fall back to its default sort.
func entityDetailRenames(idFrom, idTo string) map[string]string {
	return map[string]string{idFrom: idTo}
}
