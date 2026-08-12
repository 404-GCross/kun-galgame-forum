package repository

import (
	userModel "kun-galgame-api/internal/user/model"

	"gorm.io/gorm"
)

type ImageRepository struct {
	db *gorm.DB
}

func NewImageRepository(db *gorm.DB) *ImageRepository {
	return &ImageRepository{db: db}
}

func (r *ImageRepository) GetDailyCount(userID int) (int, error) {
	var s userModel.KungalUserState
	err := r.db.Select("daily_image_count").
		Where("user_id = ?", userID).First(&s).Error
	return s.DailyImageCount, err
}

func (r *ImageRepository) IncrementDailyCount(userID int) {
	r.db.Model(&userModel.KungalUserState{}).Where("user_id = ?", userID).
		Update("daily_image_count", gorm.Expr("daily_image_count + 1"))
}
