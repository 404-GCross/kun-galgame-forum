package repository

import (
	"kun-galgame-api/internal/doc/model"

	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) DB() *gorm.DB { return r.db }

func (r *CategoryRepository) FindPaginated(keyword string, page, limit int) ([]model.DocCategory, int64) {
	query := r.db.Model(&model.DocCategory{})
	if keyword != "" {
		query = query.Where(
			"title ILIKE ? OR slug ILIKE ?",
			"%"+keyword+"%", "%"+keyword+"%",
		)
	}

	var total int64
	query.Count(&total)

	var categories []model.DocCategory
	query.Order("sort_order ASC, id ASC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&categories)

	return categories, total
}

func (r *CategoryRepository) Create(category *model.DocCategory) error {
	return r.db.Create(category).Error
}

func (r *CategoryRepository) UpdateFields(id int, updates map[string]any) error {
	return r.db.Model(&model.DocCategory{}).Where("id = ?", id).Updates(updates).Error
}

func (r *CategoryRepository) CountArticles(categoryID int) int64 {
	var count int64
	r.db.Model(&model.DocArticle{}).
		Where("category_id = ?", categoryID).
		Count(&count)
	return count
}

func (r *CategoryRepository) DeleteByID(id int) error {
	return r.db.Delete(&model.DocCategory{}, id).Error
}
