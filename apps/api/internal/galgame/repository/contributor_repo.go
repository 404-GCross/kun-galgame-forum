package repository

import (
	"time"

	"gorm.io/gorm"
)

type GalgameContributorRepository struct {
	db *gorm.DB
}

func NewGalgameContributorRepository(db *gorm.DB) *GalgameContributorRepository {
	return &GalgameContributorRepository{db: db}
}

type ContributorTouch struct {
	GalgameID int64
	UserID    int64
	Count     int
	FirstAt   time.Time
	LastAt    time.Time
}

func (r *GalgameContributorRepository) UpsertRevisionTouches(touches []ContributorTouch) error {
	if len(touches) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, t := range touches {
			if err := tx.Exec(`
				INSERT INTO galgame_contributor
					(galgame_id, user_id, first_at, last_at, revision_count, source)
				VALUES (?, ?, ?, ?, ?, 1)
				ON CONFLICT (galgame_id, user_id) DO UPDATE SET
					revision_count = galgame_contributor.revision_count + excluded.revision_count,
					first_at = LEAST(galgame_contributor.first_at, excluded.first_at),
					last_at = GREATEST(galgame_contributor.last_at, excluded.last_at)
			`, t.GalgameID, t.UserID, t.FirstAt, t.LastAt, t.Count).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GalgameContributorRepository) RefreshContributorCounts(gids []int64) error {
	if len(gids) == 0 {
		return nil
	}
	return r.db.Exec(`
		UPDATE galgame SET contributor_count = (
			SELECT COUNT(*) FROM galgame_contributor c WHERE c.galgame_id = galgame.id
		) WHERE id IN ?`, gids).Error
}

type ContributorBrief struct {
	UserID        int64 `gorm:"column:user_id"`
	RevisionCount int   `gorm:"column:revision_count"`
}

func (r *GalgameContributorRepository) FindContributors(galgameID, limit int) []ContributorBrief {
	var rows []ContributorBrief
	r.db.Table("galgame_contributor").
		Select("user_id, revision_count").
		Where("galgame_id = ?", galgameID).
		Order("revision_count DESC, first_at ASC").
		Limit(limit).
		Scan(&rows)
	return rows
}
