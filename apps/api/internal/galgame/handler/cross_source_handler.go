package handler

import (
	"strconv"

	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// CrossSourceHandler serves the galgame detail page's cross-source read face
// (step 36): the three-source scores + site stats (wiki passthrough) and the
// catalog credits / entity reverse-lookups (catalog S2S passthrough). All
// endpoints are public GET proxies — the wiki / catalog response shape is
// forwarded verbatim, so the 37 frontend reads the infra contract directly
// (docs/integration/galgame_wiki/10 + docs/catalog).
type CrossSourceHandler struct {
	service *service.CrossSourceService
}

func NewCrossSourceHandler(svc *service.CrossSourceService) *CrossSourceHandler {
	return &CrossSourceHandler{service: svc}
}

// ──────────────────────────────────────────
// Scores / stats (wiki passthrough)
// ──────────────────────────────────────────

// Scores — GET /galgame/:gid/scores
func (h *CrossSourceHandler) Scores(c fiber.Ctx) error {
	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil || gid <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}
	data, appErr := h.service.Scores(c.Context(), gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, data)
}

// Stats — GET /galgame/stats. Forwards the wiki's ETag and honours
// If-None-Match so the browser's conditional cache (304) survives the BFF hop.
func (h *CrossSourceHandler) Stats(c fiber.Ctx) error {
	res, appErr := h.service.Stats(c.Context(), c.Get("If-None-Match"))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	if res.ETag != "" {
		c.Set("ETag", res.ETag)
	}
	c.Set("Cache-Control", "public, max-age=0, s-maxage=3600, stale-while-revalidate=300")
	if res.NotModified {
		return c.SendStatus(fiber.StatusNotModified)
	}
	return response.OK(c, res.Data)
}

// ──────────────────────────────────────────
// Catalog read face (S2S passthrough)
// ──────────────────────────────────────────

// WorkCredits — GET /catalog/works/:wid/credits. :wid comes from the detail
// response's catalog_work_id (the frontend passes it back); the backend only
// validates the shape, never resolves gid→wid (job stays minimal).
func (h *CrossSourceHandler) WorkCredits(c fiber.Ctx) error {
	wid, err := strconv.ParseInt(c.Params("wid"), 10, 64)
	if err != nil || wid <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的作品 ID"))
	}
	data, appErr := h.service.WorkCredits(c.Context(), wid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, data)
}

// NameWorks — GET /catalog/names/:nid/works
func (h *CrossSourceHandler) NameWorks(c fiber.Ctx) error {
	nid, err := strconv.ParseInt(c.Params("nid"), 10, 64)
	if err != nil || nid <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的人物名 ID"))
	}
	data, appErr := h.service.NameWorks(c.Context(), nid, queryInt(c, "limit"), queryInt(c, "offset"))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, data)
}

// CharacterWorks — GET /catalog/characters/:cid/works
func (h *CrossSourceHandler) CharacterWorks(c fiber.Ctx) error {
	cid, err := strconv.ParseInt(c.Params("cid"), 10, 64)
	if err != nil || cid <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的角色 ID"))
	}
	data, appErr := h.service.CharacterWorks(c.Context(), cid, queryInt(c, "limit"), queryInt(c, "offset"))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, data)
}

// queryInt reads a non-negative int query param, returning 0 when absent /
// invalid (the catalog then applies its own default page size).
func queryInt(c fiber.Ctx, key string) int {
	n, err := strconv.Atoi(c.Query(key))
	if err != nil || n < 0 {
		return 0
	}
	return n
}
