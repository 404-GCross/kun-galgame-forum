package dto

// ──────────────────────────────────────────
// Shared galgame card (used by series/official/engine/tag detail)
// ──────────────────────────────────────────

// GalgameCard is the enriched galgame card returned by entity detail pages.
// It fuses galgame metadata with local interaction counts.
type GalgameCard struct {
	ID                 int         `json:"id"`
	Name               KunLanguage `json:"name"`
	Banner             string      `json:"banner"`
	User               UserBrief   `json:"user"`
	ContentLimit       string      `json:"content_limit"`
	View               int         `json:"view"`
	LikeCount          int         `json:"like_count"`
	ResourceUpdateTime string      `json:"resource_update_time"`
	Platform           []string    `json:"platform"`
	Language           []string    `json:"language"`
	// U1: nil = unknown release; see NextMoeGalgameDetailFull comment.
	ReleaseDate    *string `json:"release_date"`
	ReleaseDateTBA bool    `json:"release_date_tba"`
	// ReleasePrecision (day/month/year/tba/unknown) tells the calendar how to
	// read ReleaseDate (e.g. "2026-06-01" with precision=month = "本月内, 日未定").
	// Only populated on the calendar endpoints; omitempty → absent elsewhere.
	ReleasePrecision string `json:"release_precision,omitempty"`
	// U2: same convention as GalgameListCard — card only carries the
	// derived banner; URL injected by rewriteBanners. banner_image_hash
	// retired in galgame PR5 (K-PR6).
	EffectiveBannerHash      string `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL       string `json:"effective_banner_url,omitempty"`
	EffectiveBannerWidth     int    `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight    int    `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash string `json:"effective_banner_thumbhash,omitempty"`
	// IsOnForum is false for galgame-catalogue games the forum has never ingested
	// (no local row → no resources / ratings / views). Entity detail pages
	// (会社 / tag / engine / series) list the FULL galgame catalogue, so the
	// frontend uses this to hide the forum-only fields + show a "未收录" state
	// instead of misleading zeros.
	IsOnForum bool `json:"is_on_forum"`
	// Status = galgame 草稿状态 (calendar only): 2 = 未认领的 VNDB 草稿. The FE renders
	// status=2 as a "未发布" claim card (→ publish wizard) rather than a /galgame
	// link. omitempty so published (0) cards stay unchanged everywhere else.
	Status int `json:"status,omitempty"`
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
	GalgameCount int      `json:"galgame_count"`
}

type OfficialListPage struct {
	Officials []OfficialListItem `json:"officials"`
	Total     int64              `json:"total"`
}

// OfficialLink is one of a 会社's web presences, with the source key the FE
// names it by (official_site / twitter / cien).
type OfficialLink struct {
	Source string `json:"source"`
	URL    string `json:"url"`
}

type OfficialDetail struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Original-language name (galgame PR4 sub-change, K-PR6). Passed through
	// from galgame so the FE edit modal can pre-fill the current value;
	// without it the modal opens with an empty input every time.
	Original string `json:"original"`
	// Link is the official site alone. Links is every web presence the catalog
	// carries, each with the source key that names it — a 会社 whose only
	// presence is an X account has an empty Link and one entry here, which is
	// the honest reading of "no official site, but you can still find them".
	Links        []OfficialLink `json:"links"`
	Link         string         `json:"link"`
	Category     string         `json:"category"`
	Lang         string         `json:"lang"`
	Description  string         `json:"description"`
	Alias        []string       `json:"alias"`
	Galgame      []GalgameCard  `json:"galgame"`
	GalgameCount int64          `json:"galgame_count"`
	// MovedTo is the ONLY field set when this label id was merged away
	// upstream: the identity now lives on that catalog label id and the page
	// must 301 to it in a single hop. Everything else stays zero on purpose —
	// the survivor's content never travels under the dead id, or the same
	// entity would exist at two URLs. Omitted entirely on a live label.
	MovedTo int `json:"moved_to,omitempty"`
}

// ──────────────────────────────────────────
// Engine
// ──────────────────────────────────────────

type EngineListItem struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgame_count"`
}

type EngineDetail struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Alias        []string      `json:"alias"`
	Galgame      []GalgameCard `json:"galgame"`
	GalgameCount int64         `json:"galgame_count"`
}

// ──────────────────────────────────────────
// Series
// ──────────────────────────────────────────

// SeriesListItem is one row of the series index — identity plus the upstream
// member count, which is what a picker needs to disambiguate two similarly
// named series.
type SeriesListItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	GalgameCount int    `json:"galgame_count"`
}

// SeriesDetail is the series entity page: the series' identity plus the
// forum-local, filterable subset of its member works — the same shape the
// tag / official / engine pages carry, so the four render through one set of
// components.
type SeriesDetail struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Description is ONE intro, not the whole set: the catalog keeps every
	// source's text for a series, and stacking them under the title reads as a
	// bug. See seriesIntro for the preference order.
	Description  string        `json:"description"`
	Galgame      []GalgameCard `json:"galgame"`
	GalgameCount int64         `json:"galgame_count"`
}

// ──────────────────────────────────────────
// Tag
// ──────────────────────────────────────────

type TagListItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	GalgameCount int    `json:"galgame_count"`
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
	GalgameCount int64         `json:"galgame_count"`
}
