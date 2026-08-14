package repository

import (
	"strings"
	"time"

	"kun-galgame-api/internal/toolset/model"

	"gorm.io/gorm"
)

type ToolsetRepository struct {
	db *gorm.DB
}

func NewToolsetRepository(db *gorm.DB) *ToolsetRepository {
	return &ToolsetRepository{db: db}
}

func (r *ToolsetRepository) DB() *gorm.DB { return r.db }

type ListFilters struct {
	Type     string
	Language string
	Platform string
	Version  string
	UserID   int
	Query    string
}

type ListOptions struct {
	SortField string
	SortOrder string
	Offset    int
	Limit     int
}

func (r *ToolsetRepository) buildListQuery(f ListFilters) *gorm.DB {
	q := r.db.Model(&model.GalgameToolset{}).Where("status != 1")
	if f.Type != "" && f.Type != "all" {
		q = q.Where("type = ?", f.Type)
	}
	if f.Language != "" && f.Language != "all" {
		q = q.Where("language = ?", f.Language)
	}
	if f.Platform != "" && f.Platform != "all" {
		q = q.Where("platform = ?", f.Platform)
	}
	if f.Version != "" && f.Version != "all" {
		q = q.Where("version = ?", f.Version)
	}
	if f.UserID > 0 {
		q = q.Where("user_id = ?", f.UserID)
	}
	if kw := strings.TrimSpace(f.Query); kw != "" {
		q = q.Where("name ILIKE ?", "%"+escapeLike(kw)+"%")
	}
	return q
}

func escapeLike(s string) string {
	return strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	).Replace(s)
}

func (r *ToolsetRepository) CountFiltered(f ListFilters) int64 {
	var total int64
	r.buildListQuery(f).Count(&total)
	return total
}

func (r *ToolsetRepository) ListFiltered(f ListFilters, o ListOptions) []model.GalgameToolset {
	var toolsets []model.GalgameToolset
	r.buildListQuery(f).
		Order(o.SortField + " " + o.SortOrder).
		Offset(o.Offset).Limit(o.Limit).
		Find(&toolsets)
	return toolsets
}

func (r *ToolsetRepository) FindByID(id int) (*model.GalgameToolset, error) {
	var toolset model.GalgameToolset
	if err := r.db.First(&toolset, id).Error; err != nil {
		return nil, err
	}
	return &toolset, nil
}

func (r *ToolsetRepository) Create(tx *gorm.DB, toolset *model.GalgameToolset) error {
	return tx.Create(toolset).Error
}

func (r *ToolsetRepository) UpdateFields(tx *gorm.DB, id int, updates map[string]any) error {
	return tx.Model(&model.GalgameToolset{}).Where("id = ?", id).Updates(updates).Error
}

func (r *ToolsetRepository) IncrementView(id int) {
	r.db.Model(&model.GalgameToolset{}).Where("id = ?", id).
		Update("view", gorm.Expr("view + 1"))
}

func (r *ToolsetRepository) UpdateResourceTime(tx *gorm.DB, id int, now time.Time) error {
	return tx.Model(&model.GalgameToolset{}).Where("id = ?", id).
		Update("resource_update_time", now).Error
}

func (r *ToolsetRepository) DeleteByID(tx *gorm.DB, id int) error {
	return tx.Delete(&model.GalgameToolset{}, id).Error
}

func (r *ToolsetRepository) FindAliases(toolsetID int) []model.GalgameToolsetAlias {
	var aliases []model.GalgameToolsetAlias
	r.db.Where("toolset_id = ?", toolsetID).Find(&aliases)
	return aliases
}

func (r *ToolsetRepository) ReplaceAliases(tx *gorm.DB, toolsetID int, aliases []string) error {
	if err := tx.Where("toolset_id = ?", toolsetID).
		Delete(&model.GalgameToolsetAlias{}).Error; err != nil {
		return err
	}
	for _, name := range aliases {
		if name == "" {
			continue
		}
		if err := tx.Create(&model.GalgameToolsetAlias{
			Name:      name,
			ToolsetID: toolsetID,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *ToolsetRepository) FindContributorIDs(toolsetID int) []int {
	var ids []int
	r.db.Model(&model.GalgameToolsetContributor{}).
		Where("toolset_id = ?", toolsetID).
		Pluck("user_id", &ids)
	return ids
}

func (r *ToolsetRepository) AddContributor(tx *gorm.DB, toolsetID, userID int) error {
	var cnt int64
	tx.Model(&model.GalgameToolsetContributor{}).
		Where("toolset_id = ? AND user_id = ?", toolsetID, userID).
		Count(&cnt)
	if cnt > 0 {
		return nil
	}
	return tx.Create(&model.GalgameToolsetContributor{
		ToolsetID: toolsetID,
		UserID:    userID,
	}).Error
}

func (r *ToolsetRepository) DeleteAllRelated(tx *gorm.DB, toolsetID int) error {
	related := []any{
		&model.GalgameToolsetAlias{},
		&model.GalgameToolsetContributor{},
		&model.GalgameToolsetPracticality{},
		&model.GalgameToolsetResource{},
		&model.GalgameToolsetCategoryRelation{},
	}
	for _, m := range related {
		if err := tx.Where("toolset_id = ?", toolsetID).Delete(m).Error; err != nil {
			return err
		}
	}
	return nil
}
