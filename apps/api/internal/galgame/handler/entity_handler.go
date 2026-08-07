package handler

import (
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

// EntityHandler groups the galgame-entity endpoints (official/engine/tag).
// All of these read the catalog taxonomy face, enrich with local data, and
// apply NSFW filtering.
//
// The `:id` path parameter is a CATALOG id on every one of these routes (doc
// 106 R1): the browse pages link into the new /galgame-{tag,official,engine}/c/
// URL space, and the legacy wiki-id URLs are redirect shells that never reach
// this handler.
//
// SERIES came back on a different population. P3 retired the public series
// pages because the WIKI's 146-entry vocabulary had no migration path (only 6
// of them corresponded to a catalog series) — half-migrating it would have been
// worse than dropping it. The catalog's own series facet is not that
// vocabulary: ~600 source-mirrored groupings that already back works?series_id=
// and the series block on every work record. Rendering those is a complete
// facet, not the half-migration that was refused.
type EntityHandler struct {
	officialService  *service.OfficialService
	engineService    *service.EngineService
	seriesService    *service.SeriesService
	tagService       *service.TagService
	staffService     *service.StaffService
	characterService *service.CharacterService
}

func NewEntityHandler(
	official *service.OfficialService,
	engine *service.EngineService,
	series *service.SeriesService,
	tag *service.TagService,
	staff *service.StaffService,
	character *service.CharacterService,
) *EntityHandler {
	return &EntityHandler{
		officialService:  official,
		engineService:    engine,
		seriesService:    series,
		tagService:       tag,
		staffService:     staff,
		characterService: character,
	}
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

// GetOfficialDetail — GET /galgame-official/:id (catalog label id)
func (h *EntityHandler) GetOfficialDetail(c fiber.Ctx) error {
	detail, appErr := h.officialService.GetDetail(
		c.Context(),
		c.Params("id"),
		collectQuery(c),
		utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// GetOfficialRelationGraph — GET /galgame-official/:id/relation-graph
//
// Separate from the detail route on purpose: the detail payload is refetched
// whenever the page's games grid is paged or filtered, and the corporate family
// is unaffected by either. The FE fetches this once, lazily.
func (h *EntityHandler) GetOfficialRelationGraph(c fiber.Ctx) error {
	graph, appErr := h.officialService.GetRelationGraph(c.Context(), c.Params("id"))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, graph)
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

// GetEngineDetail — GET /galgame-engine/:id (catalog engine id)
func (h *EntityHandler) GetEngineDetail(c fiber.Ctx) error {
	detail, appErr := h.engineService.GetDetail(
		c.Context(),
		c.Params("id"),
		collectQuery(c),
		utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// ──────────────────────────────────────────
// Series
// ──────────────────────────────────────────

// GetSeriesList — GET /galgame-series
func (h *EntityHandler) GetSeriesList(c fiber.Ctx) error {
	items, appErr := h.seriesService.GetList(c.Context())
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

// GetSeriesCards — GET /galgame-series/cards
//
// The index's rich cards, and the same cards on a game's detail page when
// `ids` names them. Must be registered BEFORE /galgame-series/:id, which would
// otherwise swallow "cards" as an id.
func (h *EntityHandler) GetSeriesCards(c fiber.Ctx) error {
	ids := []int{}
	for raw := range strings.SplitSeq(fiber.Query(c, "ids", ""), ",") {
		if id, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	page, appErr := h.seriesService.GetCards(
		c.Context(),
		ids,
		fiber.Query(c, "page", 1),
		fiber.Query(c, "limit", 12),
		utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// GetSeriesDetail — GET /galgame-series/:id (catalog series id)
func (h *EntityHandler) GetSeriesDetail(c fiber.Ctx) error {
	detail, appErr := h.seriesService.GetDetail(
		c.Context(),
		c.Params("id"),
		collectQuery(c),
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

// GetTagDetail — GET /galgame-tag/:id (canonical catalog tag id)
func (h *EntityHandler) GetTagDetail(c fiber.Ctx) error {
	detail, appErr := h.tagService.GetDetail(
		c.Context(),
		c.Params("id"),
		collectQuery(c),
		utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// ResolveLegacyOfficial — GET /galgame-official/legacy/:id
//
// Resolves a legacy wiki 会社 id to its catalog label id so the old URL can
// 301. Makers resolve at runtime through the registry (A2-0 covered 100% of
// them) rather than from a frozen map, so future merges keep redirecting
// correctly. 404 when the id was never registered.
func (h *EntityHandler) ResolveLegacyOfficial(c fiber.Ctx) error {
	wikiID, err := strconv.Atoi(c.Params("id"))
	if err != nil || wikiID <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的会社 ID"))
	}
	id, found, appErr := h.officialService.ResolveLegacyID(c.Context(), wikiID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	if !found {
		return response.Error(c, errors.ErrNotFound("未找到该会社"))
	}
	return response.OK(c, fiber.Map{"id": id})
}

// ──────────────────────────────────────────
// Query helpers
// ──────────────────────────────────────────

// collectQuery converts the Fiber request query args into url.Values.
func collectQuery(c fiber.Ctx) url.Values {
	q := make(url.Values)
	for k, v := range c.Queries() {
		q.Set(k, v)
	}
	return q
}

// ──────────────────────────────────────────
// Staff (credited name)
// ──────────────────────────────────────────

// GetStaffDetail — GET /galgame-staff/:id (catalog credit-name id)
//
// offset/limit page the filmography. The catalog publishes no total for it, so
// the response carries next_offset and nothing else — see dto.StaffDetail.
func (h *EntityHandler) GetStaffDetail(c fiber.Ctx) error {
	detail, appErr := h.staffService.GetDetail(
		c.Context(),
		c.Params("id"),
		fiber.Query(c, "offset", 0),
		fiber.Query(c, "limit", 50),
		utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// ──────────────────────────────────────────
// Character
// ──────────────────────────────────────────

// GetCharacterDetail — GET /galgame-character/:id (catalog character id)
//
// offset/limit page the appearance list. Like the filmography it has no total,
// so the response carries next_offset and nothing else.
//
// `works=0` asks for the identity alone — the game page's character modal,
// which shows who she is and never the list of games she is in.
func (h *EntityHandler) GetCharacterDetail(c fiber.Ctx) error {
	detail, appErr := h.characterService.GetDetail(
		c.Context(),
		c.Params("id"),
		fiber.Query(c, "offset", 0),
		fiber.Query(c, "limit", 50),
		utils.IsSFW(c),
		fiber.Query(c, "works", 1) != 0,
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}
