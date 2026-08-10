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

func (h *EntityHandler) GetOfficialList(c fiber.Ctx) error {
	page, appErr := h.officialService.GetList(c.Context(), collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *EntityHandler) SearchOfficials(c fiber.Ctx) error {
	items, appErr := h.officialService.Search(c.Context(), collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

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

func (h *EntityHandler) GetOfficialRelationGraph(c fiber.Ctx) error {
	graph, appErr := h.officialService.GetRelationGraph(c.Context(), c.Params("id"))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, graph)
}

func (h *EntityHandler) GetEngineList(c fiber.Ctx) error {
	items, appErr := h.engineService.GetList(c.Context())
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

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

func (h *EntityHandler) GetSeriesList(c fiber.Ctx) error {
	items, appErr := h.seriesService.GetList(c.Context())
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

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

func (h *EntityHandler) GetTagList(c fiber.Ctx) error {
	page, appErr := h.tagService.GetList(c.Context(), collectQuery(c), utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *EntityHandler) SearchTags(c fiber.Ctx) error {
	items, appErr := h.tagService.Search(c.Context(), collectQuery(c), utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

func (h *EntityHandler) GetMultiTagGalgames(c fiber.Ctx) error {
	page, appErr := h.tagService.GetByMultiTag(c.Context(), collectQuery(c), utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

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

func collectQuery(c fiber.Ctx) url.Values {
	q := make(url.Values)
	for k, v := range c.Queries() {
		q.Set(k, v)
	}
	return q
}

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
