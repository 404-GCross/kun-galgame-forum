package handler

import (
	"encoding/json"

	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// WikiHandler groups wiki pass-through endpoints and the galgame sub-routes
// that proxy to wiki + enrich with local user data.
type WikiHandler struct {
	wikiService *service.WikiService
}

func NewWikiHandler(wikiService *service.WikiService) *WikiHandler {
	return &WikiHandler{wikiService: wikiService}
}

// ──────────────────────────────────────────
// Generic proxy
// ──────────────────────────────────────────

// ProxyGet forwards a GET request to wiki service.
func (h *WikiHandler) ProxyGet(c fiber.Ctx) error {
	data, appErr := h.wikiService.ProxyGet(c.Context(), c.Path(), collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": data})
}

// ProxyGetWithToken forwards a GET to wiki WITH the caller's session OAuth
// token (attached by OptionalAuth — empty for anonymous callers). Needed for
// endpoints the wiki gates behind auth even on read (taxonomy revision
// history), where the plain token-less ProxyGet 401s.
func (h *WikiHandler) ProxyGetWithToken(c fiber.Ctx) error {
	token := middleware.GetAccessToken(c)
	data, appErr := h.wikiService.ProxyGetWithToken(c.Context(), c.Path(), token, collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": data})
}

// GetTaxonomyRevisions returns a handler that lists a taxonomy entity's
// revision history, hydrated (real user name/avatar). `entity` is the wiki
// resource name (tag/official/engine), bound per-route. Token forwarded via
// OptionalAuth (the wiki gates these GETs behind auth).
func (h *WikiHandler) GetTaxonomyRevisions(entity string) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := middleware.GetAccessToken(c)
		page, appErr := h.wikiService.GetTaxonomyRevisions(
			c.Context(), entity, c.Params("id"), token, collectQuery(c),
		)
		if appErr != nil {
			return response.Error(c, appErr)
		}
		return response.Paginated(c, page.Items, page.Total)
	}
}

// ProxyWriteWithToken returns a Fiber handler that forwards a POST/PUT/DELETE
// to wiki with the session-stored OAuth access token. The token is taken
// from the Redis session (attached by middleware.Auth) — NOT from a
// client-supplied header — so wiki always sees the authenticated kungal
// user's identity rather than whatever bearer the client felt like sending.
func (h *WikiHandler) ProxyWriteWithToken(method string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if _, appErr := middleware.MustGetUser(c); appErr != nil {
			return response.Error(c, appErr)
		}

		token := middleware.GetAccessToken(c)
		if token == "" {
			return response.Error(c, errors.ErrAuthExpired())
		}

		data, appErr := h.wikiService.ProxyWrite(
			c.Context(), method, c.Path(), token,
			collectQuery(c), c.Body(), c.Get("Content-Type"),
		)
		if appErr != nil {
			return response.Error(c, appErr)
		}
		return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": json.RawMessage(data)})
	}
}

// ──────────────────────────────────────────
// Galgame sub-routes
// ──────────────────────────────────────────

// GetGalgameLinks — GET /galgame/:gid/link/all
func (h *WikiHandler) GetGalgameLinks(c fiber.Ctx) error {
	links, appErr := h.wikiService.GetGalgameLinks(c.Context(), c.Params("gid"))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, links)
}

// GetGalgameHistory — GET /galgame/:gid/history/all
func (h *WikiHandler) GetGalgameHistory(c fiber.Ctx) error {
	page, appErr := h.wikiService.GetGalgameHistory(
		c.Context(), c.Params("gid"), collectQuery(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.Paginated(c, page.Items, page.Total)
}

// GetGalgamePRs — GET /galgame/:gid/pr/all
func (h *WikiHandler) GetGalgamePRs(c fiber.Ctx) error {
	page, appErr := h.wikiService.GetGalgamePRs(
		c.Context(), c.Params("gid"), collectQuery(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.Paginated(c, page.Items, page.Total)
}
