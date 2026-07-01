package handler

import (
	"strconv"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

// GalgameHandler groups the "core" galgame endpoints: create, merge PR,
// detail aggregation, list, and local interactions.
type GalgameHandler struct {
	galgameService *service.GalgameService
}

func NewGalgameHandler(galgameService *service.GalgameService) *GalgameHandler {
	return &GalgameHandler{galgameService: galgameService}
}

// ──────────────────────────────────────────
// Create / MergePR (proxy to wiki with local side effects)
// ──────────────────────────────────────────

// Create — POST /api/galgame
func (h *GalgameHandler) Create(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	// Session-stored token; see middleware.GetAccessToken for rationale.
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	data, appErr := h.galgameService.Create(c.Context(), user.ID, token, c.Body(), c.Get("Content-Type"))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": data})
}

// MergePR — PUT /api/galgame/:gid/prs/:id/merge
func (h *GalgameHandler) MergePR(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	if _, appErr := h.galgameService.MergePR(
		c.Context(), user.ID, c.Params("gid"), c.Params("id"), token,
	); appErr != nil {
		return response.Error(c, appErr)
	}
	// The wiki merge payload is empty and unused by clients. Emit a message-only
	// success so the FE's `if (result)` gate stays truthy — returning `data: null`
	// (what c.JSON produced from the empty wiki body) read as a FAILURE there, so
	// the merge silently gave no toast / no refresh and the PR sat on 进行中 until
	// a hard reload. See kunFetch's transform (message fallback).
	return response.OKMessage(c, "已合并更新请求")
}

// SubmitPR — POST /api/galgame/:gid/prs
//
// Thin wrapper over the generic write-proxy that additionally notifies the
// galgame owner ("requested"). See GalgameService.SubmitPR.
func (h *GalgameHandler) SubmitPR(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	data, appErr := h.galgameService.SubmitPR(
		c.Context(), user.ID, c.Params("gid"), token, c.Body(), c.Get("Content-Type"),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": data})
}

// DeclinePR — PUT /api/galgame/:gid/prs/:id/decline
//
// Thin wrapper over the generic write-proxy that additionally notifies the PR
// submitter ("declined"). See GalgameService.DeclinePR.
func (h *GalgameHandler) DeclinePR(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	if _, appErr := h.galgameService.DeclinePR(
		c.Context(), user.ID, c.Params("gid"), c.Params("id"), token, c.Body(), c.Get("Content-Type"),
	); appErr != nil {
		return response.Error(c, appErr)
	}
	// Message-only success — same rationale as MergePR (empty wiki body must not
	// surface as `data: null`, which the FE reads as failure).
	return response.OKMessage(c, "已拒绝更新请求")
}

// ──────────────────────────────────────────
// GetDetail / GetList
// ──────────────────────────────────────────

// GetDetail — GET /api/galgame/:gid
//
// Bearer-aware: forwards the caller's OAuth access token (when present
// via OptionalAuth) so the wiki returns the caller's own pending /
// declined drafts in addition to public status=0 rows. Without the
// token, wiki applies its default visibility filter and the call
// behaves identically to the legacy anonymous path.
//
// This is what makes /edit/galgame/draft/:gid (owner viewing own
// pending) and the publish wizard's VNDB-id lookup (claimable VNDB
// draft, status=2) work without dedicated owner-only endpoints.
func (h *GalgameHandler) GetDetail(c fiber.Ctx) error {
	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}

	detail, appErr := h.galgameService.GetDetail(
		c.Context(), gid, optionalUID(c), middleware.GetAccessToken(c), utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

// GetList — GET /api/galgame
//
// SFW-default. Crawlers and cookie-less visitors get content_limit=sfw
// only; logged-in users with the NSFW switch on see everything. The
// filter happens in the service layer because kungal's galgame table
// has no content_limit field (see service.GetList for the trade-off).
func (h *GalgameHandler) GetList(c fiber.Ctx) error {
	var req dto.GalgameListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	page, appErr := h.galgameService.GetList(c.Context(), &req, utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// ──────────────────────────────────────────
// Interactions
// ──────────────────────────────────────────

// ToggleLike — PUT /api/galgame/:gid/like
func (h *GalgameHandler) ToggleLike(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}

	if appErr := h.galgameService.ToggleLike(c.Context(), user.ID, gid); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}

// ToggleFavorite — PUT /api/galgame/:gid/favorite
func (h *GalgameHandler) ToggleFavorite(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}

	if appErr := h.galgameService.ToggleFavorite(c.Context(), user.ID, gid); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}

// MyInteractions — GET /api/galgame/interactions/mine
// The current user's liked + favorited galgame ids, used to hydrate feed-card
// like/favorite state (the shared feed cache can't carry per-user state).
func (h *GalgameHandler) MyInteractions(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, h.galgameService.GetMyInteractions(user.ID))
}
