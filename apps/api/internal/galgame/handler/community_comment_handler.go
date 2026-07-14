package handler

import (
	"strconv"

	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/role"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

// CommunityCommentHandler serves the community-backed galgame comment routes
// (the `/comments` plural prefix, mounted only when KUN_GALGAME_COMMENTS_COMMUNITY
// is on). It coexists with the OLD `/comment` routes until step 04/05 retire
// them. Post-addressed writes take the community post id in the path; a `?gid`
// hint carries the galgame id for the LOCAL display counter + mention deep-links
// (the frontend always has it in context).
type CommunityCommentHandler struct {
	service *service.CommunityCommentService
}

func NewCommunityCommentHandler(svc *service.CommunityCommentService) *CommunityCommentHandler {
	return &CommunityCommentHandler{service: svc}
}

// Register mounts the community comment routes when enabled; a no-op when not,
// so with the flag off none of these paths exist (404) — the byte-identical
// deployment invariant. optAuth is the OptionalAuth group (public reads),
// authed the session-required group.
func (h *CommunityCommentHandler) Register(optAuth, authed fiber.Router, enabled bool) {
	if !enabled {
		return
	}
	optAuth.Get("/galgame/:gid/comments", h.List)
	optAuth.Get("/galgame/:gid/comments/locate", h.Locate)
	authed.Post("/galgame/:gid/comments", h.Create)
	authed.Put("/galgame/comments/:postId", h.Update)
	authed.Delete("/galgame/comments/:postId", h.Delete)
	authed.Put("/galgame/comments/:postId/like", h.ToggleLike)
	authed.Post("/galgame/comments/:postId/flag", h.Flag)
}

// List returns a flat keyset page of a galgame's comments (optional auth).
// GET /api/galgame/:gid/comments?cursor=<post_number>&limit=<=50
func (h *CommunityCommentHandler) List(c fiber.Ctx) error {
	gid, ok := parsePositive(c.Params("gid"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的 galgame ID"))
	}
	var req struct {
		Cursor string `query:"cursor"`
		Limit  int    `query:"limit" validate:"omitempty,min=1,max=50"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.service.GetComments(c.Context(), gid, optionalUID(c), req.Cursor, req.Limit)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

// Create posts a comment/reply on a galgame (authenticated).
// POST /api/galgame/:gid/comments   body { content, reply_to_post_id? }
func (h *CommunityCommentHandler) Create(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	gid, ok := parsePositive(c.Params("gid"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的 galgame ID"))
	}
	var req struct {
		Content       string `json:"content" validate:"required,min=1,max=5000"`
		ReplyToPostID *int64 `json:"reply_to_post_id"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	item, appErr := h.service.CreateComment(c.Context(), user.ID, gid, req.Content, req.ReplyToPostID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, item)
}

// Update edits a comment's body (author, or moderator via the mod-actor variant).
// PUT /api/galgame/comments/:postId?gid=<id>   body { content }
func (h *CommunityCommentHandler) Update(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	postID, ok := parsePositive64(c.Params("postId"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的评论 ID"))
	}
	var req struct {
		Content string `json:"content" validate:"required,min=1,max=5000"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	item, appErr := h.service.UpdateComment(c.Context(), user.ID, role.CanModerate(user.Roles), postID, optionalGid(c), req.Content)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, item)
}

// Delete tombstones a comment (author or moderator).
// DELETE /api/galgame/comments/:postId?gid=<id>
func (h *CommunityCommentHandler) Delete(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	postID, ok := parsePositive64(c.Params("postId"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的评论 ID"))
	}
	if appErr := h.service.DeleteComment(c.Context(), user.ID, role.CanModerate(user.Roles), postID, optionalGid(c)); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "评论已删除")
}

// ToggleLike flips the like on a comment (the triple write).
// PUT /api/galgame/comments/:postId/like
func (h *CommunityCommentHandler) ToggleLike(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	postID, ok := parsePositive64(c.Params("postId"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的评论 ID"))
	}
	result, appErr := h.service.ToggleLike(c.Context(), user.ID, postID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, result)
}

// Flag reports a comment to the community moderation queue.
// POST /api/galgame/comments/:postId/flag   body { reason 0-4, note? }
func (h *CommunityCommentHandler) Flag(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	postID, ok := parsePositive64(c.Params("postId"))
	if !ok {
		return response.Error(c, errors.ErrBadRequest("非法的评论 ID"))
	}
	var req struct {
		Reason int    `json:"reason" validate:"min=0,max=4"`
		Note   string `json:"note" validate:"omitempty,max=500"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.service.FlagComment(c.Context(), user.ID, postID, req.Reason, req.Note); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "举报已提交")
}

// Locate resolves an old galgame_comment id to its migrated community post.
// GET /api/galgame/:gid/comments/locate?legacy_id=<int>
func (h *CommunityCommentHandler) Locate(c fiber.Ctx) error {
	var req struct {
		LegacyID int `query:"legacy_id" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	result, appErr := h.service.Locate(req.LegacyID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, result)
}

// optionalGid reads the ?gid hint (the galgame id the frontend renders the
// comment under) for local counter + mention-deeplink bookkeeping on the
// post-addressed write routes; nil when absent/invalid (the enrichment is then
// skipped and the drift tolerated).
func optionalGid(c fiber.Ctx) *int {
	if gid, ok := parsePositive(c.Query("gid")); ok {
		return &gid
	}
	return nil
}

func parsePositive(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func parsePositive64(s string) (int64, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}
