package dto

import (
	"time"

	"kun-galgame-api/internal/toolset/model"
	userModel "kun-galgame-api/internal/user/model"
)

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type CommentListRequest struct {
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=100"`
	SortOrder string `query:"sortOrder" validate:"omitempty,oneof=asc desc"`
}

// ToolsetCommentItem is the camelCase shape returned by GET
// /toolset/:id/comment/all. Matches the galgame comment view: GET returns
// ROOTS only (paginated), each carrying its FULL set of descendants flattened
// into `reply` (oldest-first, any DB depth → one visual tier), plus
// `replyCount`. `targetUser` is the replied-to comment's author ("A → B").
// Unlike galgame the replies aren't capped/lazy — the frontend's "展开更多"
// reveals them inline from this payload (no thread endpoint).
type ToolsetCommentItem struct {
	ID         int                  `json:"id"`
	ToolsetID  int                  `json:"toolsetId"`
	Content    string               `json:"content"`
	Created    time.Time            `json:"created"`
	Edited     *time.Time           `json:"edited"`
	ParentID   *int                 `json:"parentId"`
	UserID     int                  `json:"userId"`
	Reply      []ToolsetCommentItem `json:"reply"`
	ReplyCount int                  `json:"replyCount"`
	User       userModel.UserBrief  `json:"user"`
	TargetUser *userModel.UserBrief `json:"targetUser"`
}

// ToolsetCommentListResponse is the wrapper for GET /toolset/:id/comment/all.
// Uses `commentData` + `total` to match the frontend Container.vue contract.
type ToolsetCommentListResponse struct {
	CommentData []ToolsetCommentItem `json:"commentData"`
	Total       int64                `json:"total"`
}

type CreateCommentRequest struct {
	Content  string `json:"content" validate:"required,min=1,max=1007"`
	ParentID *int   `json:"parentId"`
}

type UpdateCommentRequest struct {
	CommentID int    `json:"commentId" validate:"required,min=1"`
	Content   string `json:"content" validate:"required,min=1,max=1007"`
}

type DeleteCommentRequest struct {
	CommentID int `query:"commentId" validate:"required,min=1"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// CommentItem is the shape returned by GET /toolset/:id/comment.
// It embeds the raw GalgameToolsetComment model so the wire format is
// unchanged from the pre-refactor response.
type CommentItem struct {
	model.GalgameToolsetComment
	User       userModel.UserBrief  `json:"user"`
	ParentUser *userModel.UserBrief `json:"parent_user,omitempty"`
}

// CommentDetailItem is a slim comment + user projection used by the toolset
// detail response (commentPreview field).
//
// Explicit camelCase fields instead of embedding model.GalgameToolsetComment
// — the model's json tags are snake_case (user_id / toolset_id / parent_id)
// which silently broke the FE ToolsetComment type contract on /toolset/:id
// while /toolset/:id/comment/all (which uses ToolsetCommentItem) returned
// the same logical row in camelCase.
type CommentDetailItem struct {
	ID        int                 `json:"id"`
	Content   string              `json:"content"`
	UserID    int                 `json:"userId"`
	ToolsetID int                 `json:"toolsetId"`
	ParentID  *int                `json:"parentId"`
	Edited    *time.Time          `json:"edited"`
	Created   time.Time           `json:"created"`
	Updated   time.Time           `json:"updated"`
	User      userModel.UserBrief `json:"user"`
}

// NewCommentDetailItem projects a row + hydrated user into the
// camelCase wire shape.
func NewCommentDetailItem(c model.GalgameToolsetComment, user userModel.UserBrief) CommentDetailItem {
	return CommentDetailItem{
		ID:        c.ID,
		Content:   c.Content,
		UserID:    c.UserID,
		ToolsetID: c.ToolsetID,
		ParentID:  c.ParentID,
		Edited:    c.Edited,
		Created:   c.CreatedAt,
		Updated:   c.UpdatedAt,
		User:      user,
	}
}

// UpdatedCommentResponse carries the fields the UpdateComment service modifies.
type UpdatedCommentResponse struct {
	Content string     `json:"content"`
	Edited  *time.Time `json:"edited"`
}
