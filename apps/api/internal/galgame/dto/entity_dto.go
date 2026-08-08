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
// Taxonomy search
// ──────────────────────────────────────────

// TaxonomySearchItem is one picker row — identity and nothing else.
//
// The catalog's entity search is identity-only by design (a picker feed; a
// consumer that needs metadata follows the id to the detail lane). Search used
// to answer in the BROWSE row's shape, which meant every field the search does
// not know shipped as its zero value and the shared card rendered them: every
// hit claimed "+ 0" games and a blank category, no matter how many games it
// actually had. A shape that cannot say more than it knows cannot lie.
type TaxonomySearchItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Logo is the LABEL hit's ready-made 会社 logo URL — the one exception to
	// "identity and nothing else", because the brand mark IS identity for a
	// maker and the catalog ships its hash inline on the hit. omitempty: a tag
	// hit has no logo and must not claim an empty one.
	Logo string `json:"logo,omitempty"`
}

// ──────────────────────────────────────────
// Official
// ──────────────────────────────────────────

type OfficialListItem struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Link     string `json:"link"`
	Category string `json:"category"`
	// Logo is the 会社 logo as a READY-MADE absolute CDN URL (original size —
	// the FE derives the `_mini` variant with withImageVariant, the same way it
	// does for galgame banners), "" when the maker has no logo. A URL rather
	// than a hash so no consumer has to know the CDN layout, matching how every
	// other catalog-derived image reaches this face.
	Logo         string   `json:"logo"`
	Lang         string   `json:"lang"`
	Alias        []string `json:"alias"`
	GalgameCount int      `json:"galgame_count"`
}

type OfficialListPage struct {
	Officials []OfficialListItem `json:"officials"`
	Total     int64              `json:"total"`
}

// OfficialLink is one of a 会社's web presences.
//
// Name is resolved server-side (client.LinkDisplayName) rather than by a table
// in the frontend: most of these arrive under the catch-all source `web`, whose
// site identity is in the URL, so naming them needs the URL and a host table —
// and there is no reason for that table to exist twice, in two languages. The
// source key still travels, for a consumer that wants to group or icon by it.
type OfficialLink struct {
	Source string `json:"source"`
	Name   string `json:"name"`
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
	Links []OfficialLink `json:"links"`
	Link  string         `json:"link"`
	// Logo — see OfficialListItem.Logo. "" when the maker has no logo, which is
	// the page's cue to render the name alone rather than a broken frame.
	Logo         string        `json:"logo"`
	Category     string        `json:"category"`
	Lang         string        `json:"lang"`
	Description  string        `json:"description"`
	Alias        []string      `json:"alias"`
	Galgame      []GalgameCard `json:"galgame"`
	GalgameCount int64         `json:"galgame_count"`
	// MovedTo is the ONLY field set when this label id was merged away
	// upstream: the identity now lives on that catalog label id and the page
	// must 301 to it in a single hop. Everything else stays zero on purpose —
	// the survivor's content never travels under the dead id, or the same
	// entity would exist at two URLs. Omitted entirely on a live label.
	MovedTo int `json:"moved_to,omitempty"`
}

// OfficialRelationNode is one 会社 in the corporate family graph.
type OfficialRelationNode struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Logo — a ready-made absolute CDN URL like OfficialDetail.Logo (the FE
	// derives the `_mini` variant), "" when the maker has no logo.
	Logo string `json:"logo"`
	// WorkCount is the CATALOG-wide count, deliberately not the forum-local
	// `galgame_count` the detail header shows: the family tree describes the
	// corporate structure, and a local count per sibling would cost one members
	// query per node. Named differently so the two can never be confused.
	WorkCount int `json:"work_count"`
}

// OfficialRelationEdge reads "To is the Relation of From" — e.g.
// {from: Key, to: VisualArt's, relation: parent}. Only the canonical
// orientations arrive (parent / imprint / spawned / succeeded_by); each also
// implies its mirror read backwards, so the FE must not look for the inverse
// row.
type OfficialRelationEdge struct {
	From     int    `json:"from"`
	To       int    `json:"to"`
	Relation string `json:"relation"`
}

// OfficialRelationGraph is the connected component around one 会社 — capped
// upstream, cycle-safe, and always including the requested label itself, so a
// one-node graph means "this maker has no recorded relations" and the FE draws
// nothing.
type OfficialRelationGraph struct {
	Nodes []OfficialRelationNode `json:"nodes"`
	Edges []OfficialRelationEdge `json:"edges"`
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

// SeriesSample is one member work as it appears in a series card's cover
// montage: the name to list and the head image to fan out, nothing else. The
// card shows at most five, so this is deliberately not a GalgameCard — the
// montage needs no views, no likes and no local enrichment.
type SeriesSample struct {
	Name                     KunLanguage `json:"name"`
	EffectiveBannerHash      string      `json:"effective_banner_hash,omitempty"`
	EffectiveBannerURL       string      `json:"effective_banner_url,omitempty"`
	EffectiveBannerThumbhash string      `json:"effective_banner_thumbhash,omitempty"`
}

// SeriesCard is the rich index/panel card: identity, member count, and a
// five-work sample to render the montage and the "包含 …" line.
type SeriesCard struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// IsNSFW is read off the SAMPLE, not off the series: the catalog has no
	// series-level content verdict, so this says "at least one of the works
	// shown here is r18" — which is exactly what the chip sits next to.
	IsNSFW bool `json:"is_nsfw"`
	// GalgameCount is upstream's member count (the whole catalogue), while the
	// page behind the card lists the forum-local subset. Same caveat the other
	// three indexes carry.
	GalgameCount  int            `json:"galgame_count"`
	SampleGalgame []SeriesSample `json:"sample_galgame"`
}

// SeriesCardPage is the paged index of series cards.
type SeriesCardPage struct {
	Series []SeriesCard `json:"series"`
	Total  int64        `json:"total"`
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
	// UnpublishedGalgame is the REST of the series' catalogue: members this
	// site has no published entry for (a draft claim, or no claim at all).
	// Without it a two-work series with one published member reads as a
	// one-work "series" — the grouping only makes sense shown whole. Each row
	// is a status-2 claim card, so its link leads to the publish wizard, not to
	// a local page. Unpaged: the whole bucket rides one detail response.
	UnpublishedGalgame []GalgameCard `json:"unpublished_galgame"`
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
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	// Hidden mirrors the canonical vocabulary's do-not-display tier. Such a tag
	// is absent from every list, search and picker; its page still renders for
	// a direct link, but the FE gives it noindex on this flag.
	Hidden       bool          `json:"hidden"`
	Description  string        `json:"description"`
	Alias        []string      `json:"alias"`
	Galgame      []GalgameCard `json:"galgame"`
	GalgameCount int64         `json:"galgame_count"`
}

// ──────────────────────────────────────────
// Staff (credited name)
// ──────────────────────────────────────────

// StaffWork is one entry in a credited name's filmography: the SHARED galgame
// card, plus what this person did on it.
//
// The card is embedded rather than reimplemented so the filmography renders
// through the same component every other galgame grid on the site uses — and
// so it carries the same local enrichment (views, likes, platform badges) for
// the works the forum has ingested. GalgameCard.ID is the forum gid when the
// entry lives here and 0 otherwise: most of a working career is games this
// forum has never ingested, and those rows render as 未收录 cards with nowhere
// to click rather than being dropped.
type StaffWork struct {
	GalgameCard
	// CatalogID is the registry id, and the only stable key on this list: every
	// work the forum has not ingested shares gid 0.
	CatalogID int `json:"catalog_id"`
	// Roles are this person's credits ON THIS WORK, localized — the reason a
	// filmography beats a plain game grid.
	Roles []string `json:"roles"`
	// Characters is the voice-acting case: the catalog files one credit per
	// character voiced, so a VA arrives on the same work several times. Folded
	// onto the card, because for a voice actor the cast IS the credit.
	Characters []string `json:"characters,omitempty"`
}

// StaffLink is one external identity page for the person.
type StaffLink struct {
	Source string `json:"source"`
	Name   string `json:"name"`
	// URL is empty for a source kungal has no verified person-page template
	// for; the row then renders as text. Better a missing link than a wrong one.
	URL string `json:"url,omitempty"`
}

// StaffSibling is another name the same person signs work under.
type StaffSibling struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// StaffDetail is the 制作人员 page. It describes a credited NAME, which is not
// the same thing as a human: the registry links names to a person only where
// the evidence supports it AND the link is public, so Siblings may be empty for
// someone who demonstrably has other pen names.
type StaffDetail struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	NameJa string `json:"name_ja,omitempty"`
	NameZh string `json:"name_zh,omitempty"`
	Latin  string `json:"latin,omitempty"`
	Intro  string `json:"intro"`
	// Photo is the PERSON's portrait as a READY-MADE absolute CDN URL (original
	// size — the FE derives the variant it needs), "" for none. A URL rather
	// than a hash so no consumer has to know the CDN layout, matching how the
	// 会社 logo and every cover reach this face.
	//
	// Photo/Gender/Birth* all describe the person behind the name, so the
	// registry publishes them only where the name→person link is public. A
	// hidden link arrives zeroed here, and the page then renders no portrait
	// and no meta — which is the same thing it renders for a name the registry
	// has no person for. Both are honest; inferring the difference is not.
	Photo string `json:"photo"`
	// Gender: 1 = male, 2 = female, null = unknown or unpublished.
	Gender *int `json:"gender"`
	// The birthday is FUZZY: which parts are present IS the precision claim.
	// Year alone, year+month, and month+day with no year are all valid, so the
	// three parts travel separately rather than as a formatted date — the
	// renderer decides how to say what it was given.
	BirthY   *int           `json:"birth_y"`
	BirthM   *int           `json:"birth_m"`
	BirthD   *int           `json:"birth_d"`
	Links    []StaffLink    `json:"links"`
	Siblings []StaffSibling `json:"siblings"`
	// Roles is the set of positions seen on the LOADED works, deliberately
	// without counts: the credits list is offset-paged and publishes no total,
	// so a count could only ever describe this page. A role, once seen, is true
	// of the person no matter how many pages follow.
	Roles []string    `json:"roles"`
	Works []StaffWork `json:"works"`
	// NextOffset is null on the last page — the only end-of-list signal the
	// catalog gives, and the reason this page counts nothing.
	NextOffset *int `json:"next_offset"`
	// MovedTo is the ONLY field set when this name id was merged away (wave
	// 171's fold): the page 301s to the survivor instead of rendering anything
	// under the dead id.
	MovedTo int `json:"moved_to,omitempty"`
}
