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

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

// GetComments returns paginated comments for a galgame.
// GET /api/galgame/:gid/comment/all
func (h *CommentHandler) GetComments(c fiber.Ctx) error {
	gid, _ := strconv.Atoi(c.Params("gid"))

	var req struct {
		Page      int    `query:"page" validate:"min=1"`
		Limit     int    `query:"limit" validate:"min=1,max=50"`
		SortOrder string `query:"sort_order" validate:"omitempty,oneof=asc desc"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	// Default to newest-first; only "asc" flips it. Validated above.
	sortOrder := "desc"
	if req.SortOrder == "asc" {
		sortOrder = "asc"
	}

	result := h.commentService.GetComments(c.Context(), gid, req.Page, req.Limit, optionalUID(c), sortOrder)
	return response.Paginated(c, result.Items, result.Total)
}

// CreateComment creates a galgame comment.
// POST /api/galgame/:gid/comment
//
// parent_comment_id is optional: when present, the new comment becomes
// a reply nested under that comment (and inherits its thread root). The
// service validates the parent belongs to the same galgame.
func (h *CommentHandler) CreateComment(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	gid, _ := strconv.Atoi(c.Params("gid"))

	var req struct {
		Content         string `json:"content" validate:"required,min=1,max=5000"`
		ParentCommentID *int   `json:"parent_comment_id"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	resp, appErr := h.commentService.CreateComment(c.Context(), user.ID, gid, req.Content, req.ParentCommentID)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OK(c, resp)
}

// GetCommentThread returns the full nested thread for a root comment.
// GET /api/galgame/:gid/comment/thread/:rootId
//
// The drawer view uses this when the inline view caps recursion depth.
func (h *CommentHandler) GetCommentThread(c fiber.Ctx) error {
	rootID, _ := strconv.Atoi(c.Params("rootId"))
	if rootID <= 0 {
		return response.Error(c, errors.ErrBadRequest("非法的评论 ID"))
	}

	root, appErr := h.commentService.GetThread(c.Context(), rootID, optionalUID(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, root)
}

// UpdateComment rewrites the content of an existing comment.
// PUT /api/galgame/:gid/comment
//
// Author or moderator only. Stamps `edited` so the UI can flag the
// comment as having been changed.
func (h *CommentHandler) UpdateComment(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		CommentID int    `json:"comment_id" validate:"required,min=1"`
		Content   string `json:"content" validate:"required,min=1,max=5000"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	resp, appErr := h.commentService.UpdateComment(c.Context(), user.ID, role.CanModerate(user.Roles), req.CommentID, req.Content)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, resp)
}

// DeleteComment deletes a galgame comment.
// DELETE /api/galgame/:gid/comment
func (h *CommentHandler) DeleteComment(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		CommentID int `query:"comment_id" validate:"required,min=1"`
	}
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.commentService.DeleteComment(user.ID, role.CanModerate(user.Roles), req.CommentID); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "评论已删除")
}

// ToggleCommentLike toggles like on a galgame comment.
// PUT /api/galgame/:gid/comment/like
func (h *CommentHandler) ToggleCommentLike(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req struct {
		CommentID int `json:"comment_id" validate:"required,min=1"`
	}
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.commentService.ToggleCommentLike(user.ID, req.CommentID); appErr != nil {
		return response.Error(c, appErr)
	}

	return response.OKMessage(c, "操作成功")
}
