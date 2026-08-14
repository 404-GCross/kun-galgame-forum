package repository

import (
	"kun-galgame-api/internal/website/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) FindAll() []model.GalgameWebsiteTag {
	var tags []model.GalgameWebsiteTag
	r.db.Order("id ASC").Find(&tags)
	return tags
}

func (r *TagRepository) FindByName(name string) (*model.GalgameWebsiteTag, error) {
	var tag model.GalgameWebsiteTag
	if err := r.db.Where("name = ?", name).First(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *TagRepository) Create(tag *model.GalgameWebsiteTag) error {
	return r.db.Create(tag).Error
}

func (r *TagRepository) UpdateFields(id int, updates map[string]any) error {
	return r.db.Model(&model.GalgameWebsiteTag{}).Where("id = ?", id).Updates(updates).Error
}

// The relation's column is galgame_website_tag_id. `tag_id` named nothing, so
// every tag deletion left its relation rows behind — and the delete returned
// no error because the result was discarded.
func (r *TagRepository) DeleteByID(id int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("galgame_website_tag_id = ?", id).
			Delete(&model.GalgameWebsiteTagRelation{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.GalgameWebsiteTag{}, id).Error
	})
}

func (r *TagRepository) FindRelationsByTagID(tagID int) []model.GalgameWebsiteTagRelation {
	var rels []model.GalgameWebsiteTagRelation
	r.db.Where("galgame_website_tag_id = ?", tagID).Find(&rels)
	return rels
}

func (r *TagRepository) FindRelationsByWebsiteWithTag(websiteID int) []model.GalgameWebsiteTagRelation {
	var rels []model.GalgameWebsiteTagRelation
	r.db.Where("galgame_website_id = ?", websiteID).
		Preload("Tag").Find(&rels)
	return rels
}

func (r *TagRepository) LevelSumsByWebsiteIDs(websiteIDs []int) map[int]int {
	if len(websiteIDs) == 0 {
		return map[int]int{}
	}
	type tagSum struct {
		WebsiteID int
		Total     int
	}
	var rows []tagSum
	r.db.Table("galgame_website_tag_relation r").
		Select("r.galgame_website_id AS website_id, COALESCE(SUM(t.level), 0) AS total").
		Joins("JOIN galgame_website_tag t ON t.id = r.galgame_website_tag_id").
		Where("r.galgame_website_id IN ?", websiteIDs).
		Group("r.galgame_website_id").Scan(&rows)
	out := make(map[int]int, len(rows))
	for _, r := range rows {
		out[r.WebsiteID] = r.Total
	}
	return out
}

func (r *TagRepository) LevelSumsAll() map[int]int {
	type tagSum struct {
		WebsiteID int
		Total     int
	}
	var rows []tagSum
	r.db.Table("galgame_website_tag_relation r").
		Select("r.galgame_website_id AS website_id, COALESCE(SUM(t.level), 0) AS total").
		Joins("JOIN galgame_website_tag t ON t.id = r.galgame_website_tag_id").
		Group("r.galgame_website_id").Scan(&rows)
	out := make(map[int]int, len(rows))
	for _, r := range rows {
		out[r.WebsiteID] = r.Total
	}
	return out
}

func (r *TagRepository) ReplaceWebsiteTagRelations(tx *gorm.DB, websiteID int, tagIDs []int) error {
	if err := tx.Where("galgame_website_id = ?", websiteID).
		Delete(&model.GalgameWebsiteTagRelation{}).Error; err != nil {
		return err
	}
	return r.InsertWebsiteTagRelations(tx, websiteID, tagIDs)
}

func (r *TagRepository) InsertWebsiteTagRelations(tx *gorm.DB, websiteID int, tagIDs []int) error {
	for _, tagID := range tagIDs {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.GalgameWebsiteTagRelation{
			GalgameWebsiteID: websiteID, GalgameWebsiteTagID: tagID,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}
