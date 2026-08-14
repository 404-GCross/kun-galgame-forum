package repository

import (
	"kun-galgame-api/internal/friendlink/model"

	"gorm.io/gorm"
)

type FriendLinkRepository struct {
	db *gorm.DB
}

func NewFriendLinkRepository(db *gorm.DB) *FriendLinkRepository {
	return &FriendLinkRepository{db: db}
}

func (r *FriendLinkRepository) FindAllOrdered() []model.FriendLink {
	var links []model.FriendLink
	r.db.Order("category ASC, sort_order ASC, id ASC").Find(&links)
	return links
}

func (r *FriendLinkRepository) Create(fl *model.FriendLink) error {
	var maxOrder int
	r.db.Model(&model.FriendLink{}).
		Where("category = ?", fl.Category).
		Select("COALESCE(MAX(sort_order), -1)").Scan(&maxOrder)
	fl.SortOrder = maxOrder + 1
	return r.db.Create(fl).Error
}

func (r *FriendLinkRepository) Update(id int, fields map[string]any) error {
	return r.db.Model(&model.FriendLink{}).Where("id = ?", id).Updates(fields).Error
}

func (r *FriendLinkRepository) Delete(id int) error {
	return r.db.Delete(&model.FriendLink{}, id).Error
}

func (r *FriendLinkRepository) Reorder(category string, ids []int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			if err := tx.Model(&model.FriendLink{}).
				Where("id = ? AND category = ?", id, category).
				Update("sort_order", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
