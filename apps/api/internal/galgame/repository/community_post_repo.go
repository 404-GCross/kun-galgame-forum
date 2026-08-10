package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CommunityPostRepository struct {
	db *gorm.DB
}

func NewCommunityPostRepository(db *gorm.DB) *CommunityPostRepository {
	return &CommunityPostRepository{db: db}
}

func (r *CommunityPostRepository) DB() *gorm.DB { return r.db }

func (r *CommunityPostRepository) EnsureLike(postID int64, userID int) error {
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&model.GalgamePostLike{PostID: postID, UserID: userID}).Error
}

func (r *CommunityPostRepository) RemoveLike(postID int64, userID int) error {
	return r.db.Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&model.GalgamePostLike{}).Error
}

func (r *CommunityPostRepository) CountLikes(postIDs []int64) map[int64]int {
	out := make(map[int64]int, len(postIDs))
	if len(postIDs) == 0 {
		return out
	}
	var rows []struct {
		PostID int64 `gorm:"column:post_id"`
		N      int   `gorm:"column:n"`
	}
	r.db.Table("galgame_post_like").
		Select("post_id, COUNT(*) AS n").
		Where("post_id IN ?", postIDs).
		Group("post_id").
		Scan(&rows)
	for _, row := range rows {
		out[row.PostID] = row.N
	}
	return out
}

func (r *CommunityPostRepository) LikedSet(userID int, postIDs []int64) map[int64]bool {
	out := make(map[int64]bool, len(postIDs))
	if userID <= 0 || len(postIDs) == 0 {
		return out
	}
	var rows []struct {
		PostID int64 `gorm:"column:post_id"`
	}
	r.db.Table("galgame_post_like").
		Where("user_id = ? AND post_id IN ?", userID, postIDs).
		Select("post_id").
		Scan(&rows)
	for _, row := range rows {
		out[row.PostID] = true
	}
	return out
}

func (r *CommunityPostRepository) FindMapByLegacyID(legacyID int) (*model.GalgameCommentCommunityMap, error) {
	var row model.GalgameCommentCommunityMap
	err := r.db.Where("old_comment_id = ?", legacyID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}
