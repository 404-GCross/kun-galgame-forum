package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GalgameInteractionRepository struct {
	db *gorm.DB
}

func NewGalgameInteractionRepository(db *gorm.DB) *GalgameInteractionRepository {
	return &GalgameInteractionRepository{db: db}
}

func (r *GalgameInteractionRepository) UserInteraction(userID, galgameID int) (liked, favorited bool) {
	if userID <= 0 {
		return false, false
	}
	var lc, fc int64
	r.db.Model(&model.GalgameLike{}).
		Where("user_id = ? AND galgame_id = ?", userID, galgameID).Count(&lc)
	r.db.Model(&model.GalgameCollectionItem{}).
		Where("user_id = ? AND galgame_id = ?", userID, galgameID).Count(&fc)
	return lc > 0, fc > 0
}

func (r *GalgameInteractionRepository) UserGalgameInteractions(userID int) (liked, favorited []int) {
	liked = []int{}
	favorited = []int{}
	if userID <= 0 {
		return
	}
	r.db.Model(&model.GalgameLike{}).
		Where("user_id = ?", userID).Pluck("galgame_id", &liked)
	r.db.Model(&model.GalgameCollectionItem{}).
		Distinct("galgame_id").
		Where("user_id = ?", userID).Pluck("galgame_id", &favorited)
	return
}

func (r *GalgameInteractionRepository) ToggleLike(tx *gorm.DB, userID, galgameID int) (bool, error) {
	var existing model.GalgameLike
	result := tx.Where("user_id = ? AND galgame_id = ?", userID, galgameID).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&model.GalgameLocal{ID: galgameID}).Error; err != nil {
			return false, err
		}
		if err := tx.Create(&model.GalgameLike{UserID: userID, GalgameID: galgameID}).Error; err != nil {
			return false, err
		}
		return true, tx.Model(&model.GalgameLocal{}).Where("id = ?", galgameID).
			Update("like_count", gorm.Expr("like_count + 1")).Error
	}
	if result.Error != nil {
		return false, result.Error
	}

	if err := tx.Delete(&existing).Error; err != nil {
		return false, err
	}
	return false, tx.Model(&model.GalgameLocal{}).Where("id = ?", galgameID).
		Update("like_count", gorm.Expr("like_count - 1")).Error
}
