package handler

import (
	"encoding/json"

	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// GalgameProxyHandler groups galgame pass-through endpoints and the galgame sub-routes
// that proxy to galgame + enrich with local user data.
type GalgameProxyHandler struct {
	galgameProxyService *service.GalgameProxyService
}

func NewGalgameProxyHandler(galgameProxyService *service.GalgameProxyService) *GalgameProxyHandler {
	return &GalgameProxyHandler{galgameProxyService: galgameProxyService}
}

// ──────────────────────────────────────────
// Generic proxy
// ──────────────────────────────────────────

// ProxyGetWithToken forwards a GET to galgame WITH the caller's session OAuth
// token (attached by OptionalAuth — empty for anonymous callers). Needed for
// endpoints the galgame gates behind auth even on read (taxonomy revision
// history), where the plain token-less ProxyGet 401s.
func (h *GalgameProxyHandler) ProxyGetWithToken(c fiber.Ctx) error {
	token := middleware.GetAccessToken(c)
	data, appErr := h.galgameProxyService.ProxyGetWithToken(c.Context(), c.Path(), token, collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": data})
}

// GetTaxonomyRevisions returns a handler that lists a taxonomy entity's
// revision history, hydrated (real user name/avatar). `entity` is the galgame
// resource name (tag/official/engine), bound per-route. Token forwarded via
// OptionalAuth (the galgame gates these GETs behind auth).
func (h *GalgameProxyHandler) GetTaxonomyRevisions(entity string) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := middleware.GetAccessToken(c)
		page, appErr := h.galgameProxyService.GetTaxonomyRevisions(
			c.Context(), entity, c.Params("id"), token, collectQuery(c),
		)
		if appErr != nil {
			return response.Error(c, appErr)
		}
		return response.Paginated(c, page.Items, page.Total)
	}
}

// ProxyWriteWithToken returns a Fiber handler that forwards a POST/PUT/DELETE
// to galgame with the session-stored OAuth access token. The token is taken
// from the Redis session (attached by middleware.Auth) — NOT from a
// client-supplied header — so galgame always sees the authenticated kungal
// user's identity rather than whatever bearer the client felt like sending.
func (h *GalgameProxyHandler) ProxyWriteWithToken(method string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if _, appErr := middleware.MustGetUser(c); appErr != nil {
			return response.Error(c, appErr)
		}

		token := middleware.GetAccessToken(c)
		if token == "" {
			return response.Error(c, errors.ErrAuthExpired())
		}

		data, appErr := h.galgameProxyService.ProxyWrite(
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
func (h *GalgameProxyHandler) GetGalgameLinks(c fiber.Ctx) error {
	links, appErr := h.galgameProxyService.GetGalgameLinks(c.Context(), c.Params("gid"))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, links)
}

// The old-wire revision/PR reads (GetGalgameHistory / GetGalgamePRs and the
// bare ProxyGet they rode with) retired in E3b — kungal's history, diff and
// proposal reads all flow through the editing-engine BFF (edit_handler.go).
