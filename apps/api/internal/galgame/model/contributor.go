package model

import "time"

// Contributor sources (galgame_contributor.source, migration 069). The value
// records WHICH writer first learned about a (galgame, user) pair and is never
// rewritten afterwards: a wiki-seeded row that later collects revisions keeps
// saying 0, which is how a pair with revision_count 0 stays explicable.
const (
	ContributorSourceWikiSeed int16 = 0
	ContributorSourceRevision int16 = 1
)

// GalgameContributor is one person's editing footprint on one galgame.
//
// The pair (GalgameID, UserID) is unique; RevisionCount counts only what the
// forward sync has seen, so a seeded row that nobody has edited since the
// re-anchoring legitimately sits at 0.
type GalgameContributor struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	GalgameID     int64     `gorm:"column:galgame_id;not null" json:"galgame_id"`
	UserID        int64     `gorm:"column:user_id;not null" json:"user_id"`
	FirstAt       time.Time `gorm:"column:first_at;not null" json:"first_at"`
	LastAt        time.Time `gorm:"column:last_at;not null" json:"last_at"`
	RevisionCount int       `gorm:"column:revision_count;not null" json:"revision_count"`
	Source        int16     `gorm:"column:source;not null" json:"source"`
}

func (GalgameContributor) TableName() string { return "galgame_contributor" }
