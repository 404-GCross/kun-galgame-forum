package model

import "time"

type GalgamePostLike struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	PostID    int64     `gorm:"column:post_id;not null;uniqueIndex:idx_galgame_post_like" json:"post_id"`
	UserID    int       `gorm:"column:user_id;not null;uniqueIndex:idx_galgame_post_like" json:"user_id"`
	CreatedAt time.Time `gorm:"column:created;autoCreateTime" json:"created"`
}

func (GalgamePostLike) TableName() string { return "galgame_post_like" }

type GalgameCommentCommunityMap struct {
	OldCommentID int   `gorm:"column:old_comment_id;primaryKey" json:"old_comment_id"`
	ThreadID     int64 `gorm:"column:thread_id;not null" json:"thread_id"`
	PostID       int64 `gorm:"column:post_id;not null" json:"post_id"`
	GalgameID    int   `gorm:"column:galgame_id;not null" json:"galgame_id"`
}

func (GalgameCommentCommunityMap) TableName() string { return "galgame_comment_community_map" }
