package dto

// ──────────────────────────────────────────
// Shared galgame card (used by series/official/engine/tag detail)
// ──────────────────────────────────────────

// GalgameCard is the enriched galgame card returned by entity detail pages.
// It fuses wiki metadata with local interaction counts.
type GalgameCard struct {
	ID                 int         `json:"id"`
	Name               KunLanguage `json:"name"`
	Banner             string      `json:"banner"`
	User               UserBrief   `json:"user"`
	ContentLimit       string      `json:"contentLimit"`
	View               int         `json:"view"`
	LikeCount          int         `json:"likeCount"`
	ResourceUpdateTime string      `json:"resourceUpdateTime"`
	Platform           []string    `json:"platform"`
	Language           []string    `json:"language"`
	// U1: nil = unknown release; see WikiGalgameDetailFull comment.
	ReleaseDate    *string `json:"releaseDate"`
	ReleaseDateTBA bool    `json:"releaseDateTBA"`
	// U2: same convention as GalgameListCard — card only carries the
	// derived banner; URL injected by rewriteBanners. banner_image_hash
	// retired in wiki PR5 (K-PR6).
	EffectiveBannerHash      string `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL       string `json:"effective_banner_url,omitempty"`
	EffectiveBannerWidth     int    `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight    int    `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash string `json:"effective_banner_thumbhash,omitempty"`
}

// GalgameSample is a minimal galgame sample (name + banner) used in list views.
//
// U2: see GalgameCard. The FE series carousel needs the same hash/URL
// pair to render `_mini` for newly-uploaded (covers-only) galgames —
// without it the carousel falls back to empty `banner` and shows nothing.
type GalgameSample struct {
	Name                     KunLanguage `json:"name"`
	Banner                   string      `json:"banner"`
	EffectiveBannerHash      string      `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL       string      `json:"effective_banner_url,omitempty"`
	EffectiveBannerWidth     int         `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight    int         `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash string      `json:"effective_banner_thumbhash,omitempty"`
}

// ──────────────────────────────────────────
// Series
// ──────────────────────────────────────────

type SeriesListRequest struct {
	Page  int `query:"page" validate:"min=1"`
	Limit int `query:"limit" validate:"min=1,max=50"`
}

type SeriesListItem struct {
	ID            int             `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	IsNSFW        bool            `json:"isNSFW"`
	SampleGalgame []GalgameSample `json:"sampleGalgame"`
	GalgameCount  int             `json:"galgameCount"`
	Created       string          `json:"created"`
	Updated       string          `json:"updated"`
}

type SeriesListPage struct {
	Series []SeriesListItem `json:"series"`
	Total  int64            `json:"total"`
}

type SeriesDetail struct {
	ID            int             `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	IsNSFW        bool            `json:"isNSFW"`
	SampleGalgame []GalgameSample `json:"sampleGalgame"`
	GalgameCount  int             `json:"galgameCount"`
	Galgame       []GalgameCard   `json:"galgame"`
	Created       string          `json:"created"`
	Updated       string          `json:"updated"`
}

// ──────────────────────────────────────────
// Official
// ──────────────────────────────────────────

type OfficialListItem struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Link         string   `json:"link"`
	Category     string   `json:"category"`
	Lang         string   `json:"lang"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgameCount"`
}

type OfficialListPage struct {
	Officials []OfficialListItem `json:"officials"`
	Total     int64              `json:"total"`
}

type OfficialDetail struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Original-language name (wiki PR4 sub-change, K-PR6). Passed through
	// from wiki so the FE edit modal can pre-fill the current value;
	// without it the modal opens with an empty input every time.
	Original     string        `json:"original"`
	Link         string        `json:"link"`
	Category     string        `json:"category"`
	Lang         string        `json:"lang"`
	Description  string        `json:"description"`
	Alias        []string      `json:"alias"`
	Galgame      []GalgameCard `json:"galgame"`
	GalgameCount int64         `json:"galgameCount"`
}

// ──────────────────────────────────────────
// Engine
// ──────────────────────────────────────────

type EngineListItem struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgameCount"`
}

type EngineDetail struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Alias        []string      `json:"alias"`
	Galgame      []GalgameCard `json:"galgame"`
	GalgameCount int64         `json:"galgameCount"`
}

// ──────────────────────────────────────────
// Tag
// ──────────────────────────────────────────

type TagListItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	GalgameCount int    `json:"galgameCount"`
}

type TagListPage struct {
	Tags  []TagListItem `json:"tags"`
	Total int64         `json:"total"`
}

type TagDetail struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Category     string        `json:"category"`
	Description  string        `json:"description"`
	Alias        []string      `json:"alias"`
	Galgame      []GalgameCard `json:"galgame"`
	GalgameCount int64         `json:"galgameCount"`
}
