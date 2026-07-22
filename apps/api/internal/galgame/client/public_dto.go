package client

// NextMoe /v1 public-contract wire DTOs + mappers (open-API phase 2 wave 07, W4).
//
// Background: the kungal galgame READ set migrates off the internal bridge face
// (raw-model shapes = legacy) onto the frozen /v1 public contract (curated
// shapes). This file holds the /v1 wire structs this client parses and the
// mappers that project them back onto the kungal-internal GalgameBrief the
// enrichers already consume — so kungal's OWN API output stays byte-stable.
//
// A handful of raw-model-only fields the /v1 curation deliberately drops
// (per-cover source/source_key provenance, taxonomy alias-row metadata) have no
// /v1 source and fall to their zero value — the FE does not consume them (W4
// census). Reads whose FE-consumed fields have NO /v1 source (galgame detail
// nested taxonomy galgame_count / official link+alias, series-detail
// created/updated, batch-detail intro) are NOT migrated here — they stay on the
// internal bridge pending an infra enrichment (see the W4 report §6f).
//
// See refs/plans/09-open-api-phase2/07-route-b-endgame.md.

import (
	"strings"

	"kun-galgame-api/internal/galgame/dto"
)

// ─── /v1 wire structs (only the fields this client reads) ─────────────

// v1Names is the /v1 localized-names object: every key present, empty → null.
type v1Names struct {
	JaJP *string `json:"ja-jp"`
	ZhCN *string `json:"zh-cn"`
	ZhTW *string `json:"zh-tw"`
	EnUS *string `json:"en-us"`
}

// v1Image is one rendered /v1 image: a COMPLETE CDN URL (never a bare hash) +
// intrinsic dims + ThumbHash. Only the banner pin is read by the thin-item
// mapper below.
type v1Image struct {
	URL       string `json:"url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Thumbhash string `json:"thumbhash"`
	Kind      string `json:"kind,omitempty"`
}

// v1Meta is the /v1 include=meta block — its scalars mirror the internal bridge
// face's values (migration-parity contract). resource_update_time / created are
// null on the zero timestamp (the internal Timestamp discipline).
type v1Meta struct {
	OriginalLanguage   string  `json:"original_language"`
	VNDBID             string  `json:"vndb_id"`
	Status             int     `json:"status"`
	ContentLimit       string  `json:"content_limit"`
	ReleasePrecision   string  `json:"release_precision"`
	SeriesID           *int    `json:"series_id"`
	CatalogWorkID      *int64  `json:"catalog_work_id"`
	UserID             int     `json:"user_id"`
	ResourceUpdateTime *string `json:"resource_update_time"`
	View               int     `json:"view"`
	Created            *string `json:"created"`
}

// V1Item is the /v1 thin list/batch/search item (with include=meta expansion).
type V1Item struct {
	ID          int      `json:"id"`
	Names       v1Names  `json:"names"`
	ReleaseDate *string  `json:"release_date"`
	AgeLimit    string   `json:"age_limit"`
	Portrait    *v1Image `json:"portrait"`
	Banner      *v1Image `json:"banner"`
	Updated     string   `json:"updated"`
	Meta        *v1Meta  `json:"meta"`
}

// v1BatchData is the batch envelope ({items}, no total).
type v1BatchData struct {
	Items []V1Item `json:"items"`
}

// v1SearchData is the /v1 search envelope (+ optional pending under a user JWT).
type v1SearchData struct {
	Items   []V1Item  `json:"items"`
	Total   int64     `json:"total"`
	Pending *[]V1Item `json:"pending"`
}

// ─── mappers: /v1 → kungal-internal DTOs ──────────────────────────────

// hashFromURL extracts the content-addressed image hash from a /v1 sharded CDN
// URL ({base}/aa/bb/<hash>.webp): the basename minus the .webp extension. The
// bare-hash form GalgameBrief.EffectiveBannerHash carries. Returns "" for an
// empty URL (no pinned image).
func hashFromURL(u string) string {
	if u == "" {
		return ""
	}
	base := u
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	return strings.TrimSuffix(base, ".webp")
}

// v1ContentLimit translates the kungal content_limit convention to the /v1 wire.
// kungal's "" means "no filter — return the row regardless of grading" (the
// internal bridge's permissive default); on /v1 the absent param defaults to sfw
// (which also NSFW-strips a game's nsfw cover pins), so "" maps to "all" to
// preserve the permissive semantics. "all"/"nsfw" require the key's galgame:nsfw
// scope (kungal's internal key carries it since W1a P5) — without it /v1 silently
// falls back to sfw. sfw / nsfw / all pass through unchanged.
func v1ContentLimit(cl string) string {
	if cl == "" {
		return "all"
	}
	return cl
}

// str derefs a *string to its value ("" on nil) — the /v1 empty→null locale
// discipline collapses back to the kungal "" convention.
func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// zeroTS is the JSON rendering of a Go zero time.Time — the value the internal
// bridge brief emits for an un-set resource_update_time. The /v1 meta drops it to
// null on zero, so this restores the bridge-parity value.
const zeroTS = "0001-01-01T00:00:00Z"

// resourceUpdateTime maps the /v1 meta.resource_update_time (*string, null on
// zero) to the kungal string field, restoring the bridge's zero-time literal.
func resourceUpdateTime(m *v1Meta) string {
	if m == nil || m.ResourceUpdateTime == nil {
		return zeroTS
	}
	return *m.ResourceUpdateTime
}

// bannerFields projects a /v1 banner image pin onto the derived effective-banner
// quintet the kungal briefs/items carry. The /v1 url IS the CDN url the internal
// bridge's rewriteBanners would build (both are imageclient.MainURL over the same
// KUN_IMAGE_PUBLIC_BASE_URL), so hash + url + dims + thumbhash are byte-parity.
func bannerFields(b *v1Image) (hash, url string, w, h int, thumb string) {
	if b == nil {
		return "", "", 0, 0, ""
	}
	return hashFromURL(b.URL), b.URL, b.Width, b.Height, b.Thumbhash
}

// V1ItemToBrief projects a /v1 thin item (with include=meta) onto the
// GalgameBrief the enrichers consume. Fields the internal batch brief does not
// carry (the legacy `banner` hash string) stay at their zero value, matching the
// bridge brief.
func V1ItemToBrief(it *V1Item) GalgameBrief {
	b := GalgameBrief{
		ID:                 it.ID,
		NameEnUs:           str(it.Names.EnUS),
		NameZhCn:           str(it.Names.ZhCN),
		NameJaJp:           str(it.Names.JaJP),
		NameZhTw:           str(it.Names.ZhTW),
		AgeLimit:           it.AgeLimit,
		ReleaseDate:        it.ReleaseDate,
		ResourceUpdateTime: resourceUpdateTime(it.Meta),
	}
	if it.Meta != nil {
		b.VndbID = it.Meta.VNDBID
		b.Status = it.Meta.Status
		b.ContentLimit = it.Meta.ContentLimit
		b.OriginalLanguage = it.Meta.OriginalLanguage
		b.UserID = it.Meta.UserID
	}
	b.EffectiveBannerHash, b.EffectiveBannerURL,
		b.EffectiveBannerWidth, b.EffectiveBannerHeight,
		b.EffectiveBannerThumbhash = bannerFields(it.Banner)
	return b
}

// V1ItemToNextMoeItem projects a /v1 thin item (with include=meta) onto the
// dto.NextMoeGalgameItem the GalgameEnricher / calendar / search / series
// adapters consume. release_date_tba has no thin-item source (the bridge's plain
// list item omits it too) → false; release_precision is meta-carried (calendar
// endpoints populate it, other list reads leave it "").
func V1ItemToNextMoeItem(it *V1Item) dto.NextMoeGalgameItem {
	m := dto.NextMoeGalgameItem{
		ID:          it.ID,
		NameEnUs:    str(it.Names.EnUS),
		NameZhCn:    str(it.Names.ZhCN),
		NameJaJp:    str(it.Names.JaJP),
		NameZhTw:    str(it.Names.ZhTW),
		ReleaseDate: it.ReleaseDate,
	}
	if it.Meta != nil {
		m.ContentLimit = it.Meta.ContentLimit
		m.ResourceUpdateTime = resourceUpdateTime(it.Meta)
		m.UserID = it.Meta.UserID
		m.ReleasePrecision = it.Meta.ReleasePrecision
		m.Status = it.Meta.Status
	}
	m.EffectiveBannerHash, m.EffectiveBannerURL,
		m.EffectiveBannerWidth, m.EffectiveBannerHeight,
		m.EffectiveBannerThumbhash = bannerFields(it.Banner)
	return m
}
