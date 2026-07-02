package dto

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type CommentListRequest struct {
	WebsiteID int `query:"website_id" validate:"required,min=1"`
}

type CreateCommentRequest struct {
	Content   string `json:"content" validate:"required,min=1,max=1007"`
	WebsiteID int    `json:"website_id" validate:"required,min=1"`
	ParentID  *int   `json:"parent_id"`
}

type DeleteCommentRequest struct {
	CommentID int `query:"comment_id" validate:"required,min=1"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// CommentUser is the user projection embedded in a comment.
type CommentUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// CommentItem matches the galgame comment view: GET returns ROOTS only, each
// carrying its FULL set of descendants flattened into `reply` (oldest-first, any
// DB depth → one visual tier) plus `replyCount`. `targetUser` is the replied-to
// comment's author ("A → B"). The frontend's "展开更多" reveals the replies inline
// from this payload (no lazy thread fetch).
type CommentItem struct {
	ID         int            `json:"id"`
	Content    string         `json:"content"`
	ParentID   *int           `json:"parent_id"`
	UserID     int            `json:"user_id"`
	WebsiteID  int            `json:"website_id"`
	Created    string         `json:"created"`
	Edited     *string        `json:"edited"`
	Reply      []*CommentItem `json:"reply"`
	ReplyCount int            `json:"reply_count"`
	User       CommentUser    `json:"user"`
	TargetUser any            `json:"target_user"`
}
