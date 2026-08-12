package service

import (
	"log/slog"
	"math"
	"time"

	"gorm.io/gorm"
)

const feedTypeGalgameComment = "GALGAME_COMMENT_CREATION"

func feedParityUpsert(db *gorm.DB, feedType string, postID int64, userID, gid int, content, link string, nsfw bool, createdAt string) {
	if postID > math.MaxInt32 {
		slog.Warn("feed_activity source_id overflow — community post id exceeds int32; feed parity row skipped", "post_id", postID, "type", feedType)
		return
	}
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		created = time.Now()
	}
	if err := db.Exec("SELECT feed_upsert(?, ?, ?, ?, ?, ?, ?, ?)",
		feedType, postID, userID, gid, truncate(content, 100), link, nsfw, created).Error; err != nil {
		slog.Warn("feed_activity parity upsert failed (best-effort)", "post_id", postID, "type", feedType, "error", err)
	}
}

func feedParityDelete(db *gorm.DB, feedType string, postID int64) {
	if postID > math.MaxInt32 {
		return
	}
	if err := db.Exec("SELECT feed_delete(?, ?)", feedType, postID).Error; err != nil {
		slog.Warn("feed_activity parity delete failed (best-effort)", "post_id", postID, "type", feedType, "error", err)
	}
}

func feedParityDeleteLegacyGalgame(db *gorm.DB, postID int64) {
	feedParityDeleteLegacy(db, feedTypeGalgameComment,
		"SELECT old_comment_id FROM galgame_comment_community_map WHERE post_id = ?", postID)
}

func feedParityDeleteLegacyResource(db *gorm.DB, src CommentSource, postID int64) {
	feedParityDeleteLegacy(db, src.feedType,
		"SELECT old_id FROM resource_comment_community_map WHERE source = ? AND post_id = ?", src.key, postID)
}

func feedParityDeleteLegacy(db *gorm.DB, feedType, lookup string, args ...any) {
	var oldID int
	if err := db.Raw(lookup, args...).Scan(&oldID).Error; err != nil || oldID == 0 {
		return
	}
	if err := db.Exec("SELECT feed_delete(?, ?)", feedType, oldID).Error; err != nil {
		slog.Warn("feed_activity legacy-row delete failed (best-effort)", "old_id", oldID, "type", feedType, "error", err)
	}
}
