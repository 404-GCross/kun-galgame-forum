package dto

// This file holds parsing structs for Wiki Service responses.
// These types mirror the wire format produced by the wiki service;
// they are used by services when decoding `json.RawMessage` payloads.
//
// The fields are a superset — consumers ignore what they don't need.

// WikiAlias is a named alias entry used by Official/Tag/Galgame.
type WikiAlias struct {
	Name string `json:"name"`
}

// WikiGalgameItem is the shape returned inside list/detail responses of
// series/official/engine/tag endpoints (a "lite" galgame summary).
//
// Note: wiki returns `view` here too but kungal owns view as a per-site
// stat — see WikiGalgameDetailFull comment. We don't parse it; enricher
// reads view from the local stats row.
type WikiGalgameItem struct {
	ID                 int     `json:"id"`
	NameEnUs           string  `json:"name_en_us"`
	NameJaJp           string  `json:"name_ja_jp"`
	NameZhCn           string  `json:"name_zh_cn"`
	NameZhTw           string  `json:"name_zh_tw"`
	Banner             string  `json:"banner"`
	ContentLimit       string  `json:"content_limit"`
	ResourceUpdateTime string  `json:"resource_update_time"`
	ReleaseDate        *string `json:"release_date"`
	ReleaseDateTBA     bool    `json:"release_date_tba"`
	// U2: derived effective banner hash on list rows (wiki computes it
	// from covers[sort_order=0]). EffectiveBannerURL is injected by
	// client.rewriteBanners BEFORE we unmarshal — capture it
	// explicitly or Go's unmarshal drops the walker's work.
	// banner_image_hash was retired in wiki PR5 (K-PR6) — no field.
	EffectiveBannerHash      string `json:"effective_banner_hash"`
	EffectiveBannerURL       string `json:"effective_banner_url"`
	EffectiveBannerWidth     int    `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight    int    `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash string `json:"effective_banner_thumbhash,omitempty"`
	UserID                   int    `json:"user_id"`
}

// U2 cover/screenshot row shapes (snake_case, matches wiki wire). Both
// share the scalar fields; screenshot additionally has Caption.
//
// CDNURL: like EffectiveBannerURL above, client.rewriteBanners injects a
// per-row `cdn_url` (image_hash → CDN URL) into the wiki bytes BEFORE we
// unmarshal — this field MUST be declared or Go's unmarshal silently drops
// the walker's work and the gallery renders no images. (This was the bug:
// the field was missing here, so screenshots/covers reached the FE without a
// cdn_url and the gallery fell back to a /image/<hash> redirect per image.)
type WikiGalgameCover struct {
	ImageHash string `json:"image_hash"`
	SortOrder int    `json:"sort_order"`
	Sexual    int    `json:"sexual"`
	Violence  int    `json:"violence"`
	Source    string `json:"source"`
	SourceKey string `json:"source_key"`
	// Kind labels the VNDB cover type (main/pkgfront/dig/pkgback/…); "" for user
	// uploads. Declared so it survives unmarshal — the "查看所有封面" modal groups
	// covers by it (without this every cover fell into 其它).
	Kind      string `json:"kind,omitempty"`
	CDNURL    string `json:"cdn_url"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Thumbhash string `json:"thumbhash,omitempty"`
}

type WikiGalgameScreenshot struct {
	ImageHash string `json:"image_hash"`
	SortOrder int    `json:"sort_order"`
	Caption   string `json:"caption"`
	Sexual    int    `json:"sexual"`
	Violence  int    `json:"violence"`
	Source    string `json:"source"`
	SourceKey string `json:"source_key"`
	CDNURL    string `json:"cdn_url"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Thumbhash string `json:"thumbhash,omitempty"`
}

// WikiOfficial is a company/publisher/developer entity from the wiki.
//
// `galgame_count` is published by wiki since K-PR (2026-05-22): the
// detail endpoint's Preload now injects the same `COUNT(*) AS cnt`
// subquery the list endpoint uses, so the "会社名 +N" chip on the
// galgame detail page can be filled in a single round-trip. Field
// defaults to 0 (omitempty intentionally not used — 0 is the correct
// answer for an official that exists but currently has zero published
// galgames).
type WikiOfficial struct {
	ID           int         `json:"id"`
	Name         string      `json:"name"`
	Link         string      `json:"link"`
	Category     string      `json:"category"`
	Lang         string      `json:"lang"`
	Alias        []WikiAlias `json:"alias"`
	GalgameCount int         `json:"galgame_count"`
}

// WikiOfficialRel is the wrapper used when an official is attached to a galgame.
type WikiOfficialRel struct {
	Official WikiOfficial `json:"official"`
}

// WikiEngine is a game engine entity.
type WikiEngine struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Link  string `json:"link"`
	Intro string `json:"intro"`
}

// WikiEngineRel wraps an engine attached to a galgame.
type WikiEngineRel struct {
	Engine WikiEngine `json:"engine"`
}

// WikiTag is a tag entity with optional aliases.
//
// `galgame_count` is published since K-PR (2026-05-22) — same source
// as WikiOfficial.GalgameCount (wiki's detail Preload injects a COUNT
// subquery filtered on galgame.status = 0). Used to render the
// "+N" chip on tag bubbles in the galgame detail page.
type WikiTag struct {
	ID           int         `json:"id"`
	Name         string      `json:"name"`
	Category     string      `json:"category"`
	Alias        []WikiAlias `json:"alias"`
	GalgameCount int         `json:"galgame_count"`
}

// WikiTagRel wraps a tag attached to a galgame.
type WikiTagRel struct {
	Tag WikiTag `json:"tag"`
}

// WikiContributor represents a user who contributed to a galgame.
type WikiContributor struct {
	UserID int `json:"user_id"`
}

// WikiGalgameDetail is the core galgame payload returned by /galgame/:id.
// It contains the fields commonly consumed by the gateway service.
type WikiGalgameDetail struct {
	ID       int    `json:"id"`
	NameEnUs string `json:"name_en_us"`
	NameJaJp string `json:"name_ja_jp"`
	NameZhCn string `json:"name_zh_cn"`
	NameZhTw string `json:"name_zh_tw"`
	Banner   string `json:"banner"`
	// U2 derived banner pair (wiki PR5). Forwarded so consumers like the
	// rating detail service can ship them downstream — without these,
	// covers-only galgames render an empty hero on the FE.
	EffectiveBannerHash      string `json:"effective_banner_hash"`
	EffectiveBannerURL       string `json:"effective_banner_url"`
	EffectiveBannerWidth     int    `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight    int    `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash string `json:"effective_banner_thumbhash,omitempty"`
	ContentLimit             string `json:"content_limit"`
	AgeLimit                 string `json:"age_limit"`
	OriginalLanguage         string `json:"original_language"`
	// nil when the galgame isn't part of any series. Used by the rating
	// detail service to attach the minimal series brief for JSON-LD.
	SeriesID     *int              `json:"series_id"`
	Official     []WikiOfficialRel `json:"official"`
	Engine       []WikiEngineRel   `json:"engine"`
	Tag          []WikiTagRel      `json:"tag"`
	Contributors []WikiContributor `json:"contributors"`
}

// WikiGalgameDetailResponse is the envelope: {galgame: {...}}.
type WikiGalgameDetailResponse struct {
	Galgame WikiGalgameDetail `json:"galgame"`
}

// WikiUser mirrors the user shape returned by wiki inside the `users` map of
// detail responses.
type WikiUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// WikiTagWithSpoiler extends WikiTag with a spoiler_level annotation used
// only in galgame detail responses.
type WikiTagWithSpoiler struct {
	SpoilerLevel int     `json:"spoiler_level"`
	Tag          WikiTag `json:"tag"`
}

// WikiEngineAlias is a flat alias slice (used in list/detail shape).
type WikiEngineAlias []string

// WikiGalgameDetailFull is the superset of fields returned by GET /galgame/:id.
// It includes nested alias/official/engine/tag/contributor/intro/user_id.
type WikiGalgameDetailFull struct {
	ID           int    `json:"id"`
	VndbID       string `json:"vndb_id"`
	NameEnUs     string `json:"name_en_us"`
	NameJaJp     string `json:"name_ja_jp"`
	NameZhCn     string `json:"name_zh_cn"`
	NameZhTw     string `json:"name_zh_tw"`
	Banner       string `json:"banner"`
	IntroEnUs    string `json:"intro_en_us"`
	IntroJaJp    string `json:"intro_ja_jp"`
	IntroZhCn    string `json:"intro_zh_cn"`
	IntroZhTw    string `json:"intro_zh_tw"`
	ContentLimit string `json:"content_limit"`
	// Wiki also returns `view`, but kungal owns view as a per-site stat
	// (each site has its own user base, so wiki's cross-site view doesn't
	// fit the on-page "this many people viewed on kungal" semantics).
	// Intentionally not parsed — see GetDetail in galgame_service.go which
	// reads view from the local galgame stats row instead.
	ResourceUpdateTime string `json:"resource_update_time"`
	// Wiki upgrade U1: `released` (free-form string) was replaced by a
	// proper date column + a TBA flag. `*string "YYYY-MM-DD"` keeps tz/
	// precision out of revision diffs (wire is plain string, not Time).
	ReleaseDate      *string `json:"release_date"`
	ReleaseDateTBA   bool    `json:"release_date_tba"`
	OriginalLanguage string  `json:"original_language"`
	AgeLimit         string  `json:"age_limit"`
	UserID           int     `json:"user_id"`
	SeriesID         *int    `json:"series_id"`
	Status           int     `json:"status"`
	// U2: cover candidate set + screenshot gallery + derived effective
	// banner. wiki PR5 (K-PR6) dropped the legacy banner_image_hash
	// top-level field — banner is now expressed solely through
	// covers[sort_order=0] and its derived effective_banner_hash.
	EffectiveBannerHash      string                  `json:"effective_banner_hash"`
	EffectiveBannerURL       string                  `json:"effective_banner_url"`
	EffectiveBannerWidth     int                     `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight    int                     `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash string                  `json:"effective_banner_thumbhash,omitempty"`
	Covers                   []WikiGalgameCover      `json:"covers"`
	Screenshots              []WikiGalgameScreenshot `json:"screenshots"`
	Alias                    []WikiAlias             `json:"alias"`
	Official                 []WikiOfficialRel       `json:"official"`
	Engine                   []WikiEngineWithAlias   `json:"engine"`
	Tag                      []WikiTagWithSpoiler    `json:"tag"`
	Contributor              []WikiContributor       `json:"contributor"`
	Created                  string                  `json:"created"`
	Updated                  string                  `json:"updated"`
}

// WikiEngineWithAlias matches the engine-embedded-in-galgame shape (alias is []string).
//
// `galgame_count` source matches WikiOfficial — wiki injects the count
// subquery into the detail-time Preload so the "+N" chip can be filled
// without a follow-up request.
type WikiEngineWithAlias struct {
	Engine struct {
		ID           int      `json:"id"`
		Name         string   `json:"name"`
		Alias        []string `json:"alias"`
		GalgameCount int      `json:"galgame_count"`
	} `json:"engine"`
}

// WikiGalgameDetailFullResp is the envelope with galgame + users map.
type WikiGalgameDetailFullResp struct {
	Galgame WikiGalgameDetailFull `json:"galgame"`
	Users   map[string]WikiUser   `json:"users"`
}

// WikiSeriesSample is a sample galgame inside a series detail response.
//
// EffectiveBannerHash/URL must be declared here even though wiki's series
// payload nests these objects — rewriteBanners walks the JSON and injects
// `effective_banner_url`, but Go's json.Unmarshal silently drops any field
// not declared on the target struct, leaving the FE carousel without the
// U2 hash/URL pair for newly-uploaded (covers-only) galgames.
type WikiSeriesSample struct {
	NameEnUs                 string `json:"name_en_us"`
	NameJaJp                 string `json:"name_ja_jp"`
	NameZhCn                 string `json:"name_zh_cn"`
	NameZhTw                 string `json:"name_zh_tw"`
	Banner                   string `json:"banner"`
	ContentLimit             string `json:"content_limit"`
	EffectiveBannerHash      string `json:"effective_banner_hash"`
	EffectiveBannerURL       string `json:"effective_banner_url"`
	EffectiveBannerWidth     int    `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight    int    `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash string `json:"effective_banner_thumbhash,omitempty"`
}

// WikiSeriesBrief is the shape of /series/:id used inside GalgameDetail.
type WikiSeriesBrief struct {
	ID          int                `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Galgame     []WikiSeriesSample `json:"galgame"`
	Created     string             `json:"created"`
	Updated     string             `json:"updated"`
}

// WikiPRDetail is the shape returned by /galgame/:gid/prs/:id (inside "pr").
type WikiPRDetail struct {
	PR struct {
		UserID int `json:"user_id"`
	} `json:"pr"`
}

// WikiCreatedResp is the shape returned by POST /galgame (just the ID).
type WikiCreatedResp struct {
	ID int `json:"id"`
}
