package repository

import (
	"time"

	"kun-galgame-api/internal/toolset/model"

	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) DB() *gorm.DB { return r.db }

// ──────────────────────────────────────────
// Reads
// ──────────────────────────────────────────

// NOTE: CountByToolset / CountsForToolsets were retired in charter step 06a —
// the toolset comment counts now come from the LIVE galgame_toolset.comment_count
// column (migration 059), not a count(*) over this frozen table.

// FindPaginated returns the paginated comments for a toolset ordered by
// created in the requested direction. `sortOrder` accepts "asc" / "desc";
// any other value falls back to "desc".
func (r *CommentRepository) FindPaginated(toolsetID, page, limit int, sortOrder string) []model.GalgameToolsetComment {
	var comments []model.GalgameToolsetComment
	offset := (page - 1) * limit
	dir := "DESC"
	if sortOrder == "asc" {
		dir = "ASC"
	}
	r.db.Where("toolset_id = ?", toolsetID).
		Order("created " + dir).
		Offset(offset).Limit(limit).
		Find(&comments)
	return comments
}

// FindAllByToolset returns every comment for a toolset, oldest-first, so the
// service can group roots + flattened descendants for the 2-tier view. Comment
// volumes per toolset are small, so loading the whole set in one query (then
// grouping in memory) is simpler and cheaper than recursive root resolution.
func (r *CommentRepository) FindAllByToolset(toolsetID int) []model.GalgameToolsetComment {
	var comments []model.GalgameToolsetComment
	r.db.Where("toolset_id = ?", toolsetID).
		Order("created ASC").
		Find(&comments)
	return comments
}

// FindByID loads a single comment.
func (r *CommentRepository) FindByID(id int) (*model.GalgameToolsetComment, error) {
	var comment model.GalgameToolsetComment
	if err := r.db.First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

// ──────────────────────────────────────────
// Writes
// ──────────────────────────────────────────

// Create inserts a new comment and returns the created row.
func (r *CommentRepository) Create(comment *model.GalgameToolsetComment) error {
	return r.db.Create(comment).Error
}

// UpdateContent sets the content and `edited` timestamp on a comment.
func (r *CommentRepository) UpdateContent(comment *model.GalgameToolsetComment, content string, editedAt time.Time) {
	r.db.Model(comment).Updates(map[string]any{
		"content": content,
		"edited":  editedAt,
	})
}

// Delete removes a comment by reference.
func (r *CommentRepository) Delete(comment *model.GalgameToolsetComment) {
	r.db.Delete(comment)
}
