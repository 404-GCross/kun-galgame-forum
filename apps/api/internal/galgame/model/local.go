package model

import "time"

// GalgameLocal represents the stripped-down local galgame row.
// After wiki migration, only interaction counts + view remain locally.
//
// CreatedAt / UpdatedAt are needed so the lazy-create stub
// (`Create(&GalgameLocal{ID: ...}).OnConflict(DoNothing)`) emits valid
// timestamps for a possibly-new row. The `galgame.updated` column is
// NOT NULL with no DB-level default — left over from the Prisma
// `@updatedAt` directive which Prisma fills application-side. PG
// evaluates NOT NULL BEFORE resolving ON CONFLICT, so omitting the
// column trips a constraint violation even when the row already
// exists. GORM auto-populates these by field-name convention.
type GalgameLocal struct {
	ID               int `gorm:"primaryKey" json:"id"`
	View             int `gorm:"default:0" json:"view"`
	LikeCount        int `gorm:"column:like_count;default:0" json:"like_count"`
	FavoriteCount    int `gorm:"column:favorite_count;default:0" json:"favorite_count"`
	ResourceCount    int `gorm:"column:resource_count;default:0" json:"resource_count"`
	CommentCount     int `gorm:"column:comment_count;default:0" json:"comment_count"`
	ContributorCount int `gorm:"column:contributor_count;default:0" json:"contributor_count"`
	RatingCount      int `gorm:"column:rating_count;default:0" json:"rating_count"`
	// Mirror of wiki's release_date (migration 013), so the local browse
	// list can filter/sort by release year/month — kungal's /galgame
	// doesn't proxy wiki's list. Nullable: NULL = unknown / not yet
	// backfilled. Populated by cmd/backfill-release-date (idempotent).
	// The lazy-create stub leaves it nil → NULL until a backfill run.
	ReleaseDate *time.Time `gorm:"column:release_date" json:"release_date"`
	CreatedAt   time.Time  `gorm:"column:created" json:"created"`
	UpdatedAt   time.Time  `gorm:"column:updated" json:"updated"`
	// ResourceUpdateTime is the dedicated content-update sort key (migration
	// 018), separate from the generic audit `updated`. autoCreateTime means GORM
	// sets it once on the lazy-create stub and NEVER bumps it on a plain
	// Save/Update — only the explicit Touch / TouchGalgameUpdated content paths
	// move it, so engagement (like / favorite / comment / view) can't reorder
	// the list.
	ResourceUpdateTime time.Time `gorm:"column:resource_update_time;autoCreateTime" json:"resource_update_time"`
}

func (GalgameLocal) TableName() string { return "galgame" }

// ──────────────────────────────────────────
// Interactions (local to each site)
// ──────────────────────────────────────────

type GalgameLike struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	GalgameID int `gorm:"column:galgame_id;not null;uniqueIndex:idx_galgame_like" json:"galgame_id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_galgame_like" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameLike) TableName() string { return "galgame_like" }

type GalgameFavorite struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	GalgameID int `gorm:"column:galgame_id;not null;uniqueIndex:idx_galgame_favorite" json:"galgame_id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_galgame_favorite" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameFavorite) TableName() string { return "galgame_favorite" }

// ──────────────────────────────────────────
// Comment (local)
// ──────────────────────────────────────────

type GalgameComment struct {
	ID        int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Content   string `gorm:"type:varchar(5000);not null" json:"content"`
	GalgameID int    `gorm:"column:galgame_id;not null" json:"galgame_id"`
	UserID    int    `gorm:"column:user_id;not null" json:"user_id"`
	// ParentCommentID: direct reply parent. NULL = top-level (root)
	// comment. Self-referencing FK with ON DELETE CASCADE; deleting a
	// parent removes all descendants.
	ParentCommentID *int `gorm:"column:parent_comment_id" json:"parent_comment_id"`
	// RootCommentID: thread anchor. NULL for roots; for replies always
	// equals the top-most ancestor's ID. Lets us fetch a full thread
	// with a single indexed WHERE root_comment_id = ?.
	RootCommentID *int `gorm:"column:root_comment_id" json:"root_comment_id"`
	LikeCount     int  `gorm:"column:like_count;default:0" json:"like_count"`

	// Edited: set on every author-driven content rewrite via the PUT
	// endpoint. Nil = never edited. Distinct from `updated`, which
	// ticks on any row-level write (e.g. like_count bumps).
	Edited *time.Time `gorm:"column:edited" json:"edited"`

	// Status: 0=normal, 1=hidden by a T&S `hide` enforcement (migration 055).
	Status int `gorm:"column:status;default:0" json:"-"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameComment) TableName() string { return "galgame_comment" }

type GalgameCommentLike struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_comment_like" json:"user_id"`
	CommentID int `gorm:"column:galgame_comment_id;not null;uniqueIndex:idx_comment_like" json:"galgame_comment_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameCommentLike) TableName() string { return "galgame_comment_like" }
