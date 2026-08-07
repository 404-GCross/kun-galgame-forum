package repository

import (
	"time"

	"gorm.io/gorm"
)

// GalgameContributorRepository owns galgame_contributor (migration 069) — the
// forum's authoritative answer to "who has edited this galgame".
type GalgameContributorRepository struct {
	db *gorm.DB
}

func NewGalgameContributorRepository(db *gorm.DB) *GalgameContributorRepository {
	return &GalgameContributorRepository{db: db}
}

func (r *GalgameContributorRepository) DB() *gorm.DB { return r.db }

// ContributorTouch is one revision's contribution by one person, in the shape
// the upsert takes: one row per (galgame, user) PAIR seen in a batch, already
// folded so the same person editing twice in a page costs one statement.
type ContributorTouch struct {
	GalgameID int64
	UserID    int64
	Count     int
	FirstAt   time.Time
	LastAt    time.Time
}

// UpsertRevisionTouches records revision-sourced contributions.
//
// On conflict the counts accumulate and the window widens (LEAST/GREATEST, not
// assignment: a full replay walks ids in order but not necessarily timestamps,
// and a re-run of an already-ingested page must not narrow what is stored).
// `source` is left alone — a wiki-seeded pair that revisions later touch keeps
// saying where it was first learned, which is what makes revision_count 0 on a
// seeded row explicable rather than suspicious.
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

// RefreshContributorCounts recomputes galgame.contributor_count for the given
// gids from the table itself, rather than incrementing alongside the upserts: a
// count derived from the rows cannot drift away from them, and a replay that
// re-applies a page recomputes the same number.
//
// A gid with no local row updates nothing, which is correct — a draft work's
// contributions are held here until the entry has a row to summarise them on.
func (r *GalgameContributorRepository) RefreshContributorCounts(gids []int64) error {
	if len(gids) == 0 {
		return nil
	}
	return r.db.Exec(`
		UPDATE galgame SET contributor_count = (
			SELECT COUNT(*) FROM galgame_contributor c WHERE c.galgame_id = galgame.id
		) WHERE id IN ?`, gids).Error
}

// ContributorBrief is one contributor row of the detail page's strip.
type ContributorBrief struct {
	UserID        int64 `gorm:"column:user_id"`
	RevisionCount int   `gorm:"column:revision_count"`
}

// FindContributors returns a galgame's contributors, busiest first and oldest
// first among equals — so the wiki-seeded rows (revision_count 0, no forward
// activity yet) sort by when the person arrived rather than arbitrarily.
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
