package dto

import (
	"time"

	userModel "kun-galgame-api/internal/user/model"
)

type CommentDetailItem struct {
	ID        int                 `json:"id"`
	Content   string              `json:"content"`
	UserID    int                 `json:"user_id"`
	ToolsetID int                 `json:"toolset_id"`
	ParentID  *int                `json:"parent_id"`
	Edited    *time.Time          `json:"edited"`
	Created   time.Time           `json:"created"`
	Updated   time.Time           `json:"updated"`
	User      userModel.UserBrief `json:"user"`
}
