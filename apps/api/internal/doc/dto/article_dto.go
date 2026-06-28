package dto

import (
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
)

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

// GetArticlesRequest is the query for GET /doc/article.
type GetArticlesRequest struct {
	Page       int    `query:"page" validate:"min=1"`
	Limit      int    `query:"limit" validate:"min=1,max=100"`
	CategoryID *int   `query:"categoryId"`
	TagID      *int   `query:"tagId"`
	Status     *int   `query:"status"`
	IsPin      *bool  `query:"isPin"`
	Keyword    string `query:"keyword"`
	OrderBy    string `query:"orderBy" validate:"omitempty,oneof=publishedTime created view updated order"`
	SortOrder  string `query:"sortOrder" validate:"omitempty,oneof=asc desc"`
	// AllStatuses returns articles of every status (incl. draft/archived) for the
	// admin manager. NO `query` tag on purpose — it is set ONLY by the
	// moderator-gated admin handler, so a public caller can never request drafts.
	AllStatuses bool `json:"-"`
}

// ReorderArticlesRequest carries the full new ordering as a flat list of article
// ids (top → bottom); sort_order is rewritten to each id's index. Mirrors the
// friend-link reorder contract.
type ReorderArticlesRequest struct {
	IDs []int `json:"ids" validate:"required,min=1,dive,min=1"`
}

// SetArticlePinRequest toggles a single article's first-page pin from the admin
// doc manager, without a full article update.
type SetArticlePinRequest struct {
	ArticleID int  `json:"articleId" validate:"required,min=1"`
	IsPin     bool `json:"isPin"`
}

// CreateArticleRequest is the payload for POST /doc/article.
type CreateArticleRequest struct {
	Title           string `json:"title" validate:"required,max=233"`
	Slug            string `json:"slug" validate:"required,max=233"`
	Description     string `json:"description" validate:"max=1000"`
	Banner          string `json:"banner" validate:"max=500"`
	Status          int    `json:"status" validate:"oneof=0 1 2"`
	IsPin           bool   `json:"isPin"`
	ContentMarkdown string `json:"contentMarkdown" validate:"required"`
	CategoryID      int    `json:"categoryId" validate:"required,min=1"`
	TagIDs          []int  `json:"tagIds"`
}

// UpdateArticleRequest is the payload for PUT /doc/article.
type UpdateArticleRequest struct {
	ArticleID       int    `json:"articleId" validate:"required,min=1"`
	Title           string `json:"title" validate:"required,max=233"`
	Slug            string `json:"slug" validate:"required,max=233"`
	Description     string `json:"description" validate:"max=1000"`
	Banner          string `json:"banner" validate:"max=500"`
	Status          int    `json:"status" validate:"oneof=0 1 2"`
	IsPin           bool   `json:"isPin"`
	ContentMarkdown string `json:"contentMarkdown" validate:"required"`
	CategoryID      int    `json:"categoryId" validate:"required,min=1"`
	TagIDs          []int  `json:"tagIds"`
}

// DeleteArticleRequest is the query for DELETE /doc/article.
type DeleteArticleRequest struct {
	ArticleID int `query:"articleId" validate:"required,min=1"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// ArticleCategoryBrief is the nested category shape embedded in list/detail
// responses so the frontend can render category pills without a separate
// fetch. Matches the legacy Nitro relation output.
type ArticleCategoryBrief struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

// ArticleSummary is the list-row shape of GET /doc/article. All fields
// are camelCase to match the rest of the kungal API surface (post-doc
// casing cleanup). The pre-refactor mixed-case response is gone.
type ArticleSummary struct {
	ID            int                  `json:"id"`
	Title         string               `json:"title"`
	Slug          string               `json:"slug"`
	Path          string               `json:"path"`
	Description   string               `json:"description"`
	Banner        string               `json:"banner"`
	Status        int                  `json:"status"`
	IsPin         bool                 `json:"isPin"`
	View          int                  `json:"view"`
	SortOrder     int                  `json:"sortOrder"`
	PublishedTime time.Time            `json:"publishedTime"`
	EditedTime    *time.Time           `json:"editedTime"`
	CategoryID    int                  `json:"categoryId"`
	AuthorID      int                  `json:"authorId"`
	Category      ArticleCategoryBrief `json:"category"`
	Created       time.Time            `json:"created"`
	Updated       time.Time            `json:"updated"`
}

// ArticleDetailResponse is the shape of GET /doc/article/:slug.
type ArticleDetailResponse struct {
	ID              int                  `json:"id"`
	Title           string               `json:"title"`
	Slug            string               `json:"slug"`
	Path            string               `json:"path"`
	Description     string               `json:"description"`
	Banner          string               `json:"banner"`
	Status          int                  `json:"status"`
	IsPin           bool                 `json:"isPin"`
	View            int                  `json:"view"`
	PublishedTime   time.Time            `json:"publishedTime"`
	EditedTime      *time.Time           `json:"editedTime"`
	ContentMarkdown string               `json:"contentMarkdown"`
	ContentHTML     string               `json:"contentHtml"`
	Toc             []markdown.TocLink   `json:"toc"`
	CategoryID      int                  `json:"categoryId"`
	AuthorID        int                  `json:"authorId"`
	Category        ArticleCategoryBrief `json:"category"`
	// Tag IDs attached to this article. Embedded so the rewrite flow on
	// the FE can pre-fill the tag picker without a second round-trip.
	// Empty array when no tags are set.
	TagIDs  []int     `json:"tagIds"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}
