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
// intrinsic dims + ThumbHash. Kind is meaningful on covers[]; Sexual/Violence
// are the W1c scope-gated per-image rating levels (present on covers/screenshots
// only when the credential is nsfw-capable — kungal's internal key carries
// galgame:nsfw since W1a P5); Caption is the per-screenshot gallery text (W1c).
// sort_order is NOT on the wire (the arrays are server-ordered) — the detail
// mapper synthesizes it from the array index.
type v1Image struct {
	URL       string `json:"url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Thumbhash string `json:"thumbhash"`
	Kind      string `json:"kind,omitempty"`
	Sexual    *int   `json:"sexual"`
	Violence  *int   `json:"violence"`
	Caption   string `json:"caption"`
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
	UserID             int     `json:"user_id"`
	ResourceUpdateTime *string `json:"resource_update_time"`
	View               int     `json:"view"`
	Created            *string `json:"created"`
}

// v1OfficialBrief is one include=officials thin-item entry ({id,name}).
type v1OfficialBrief struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// V1Item is the /v1 thin list/batch/search item. Meta (include=meta), Intro
// (include=intro, W1d) and Officials (include=officials) are the optional
// expansions — nil unless requested.
type V1Item struct {
	ID          int                `json:"id"`
	Names       v1Names            `json:"names"`
	ReleaseDate *string            `json:"release_date"`
	AgeLimit    string             `json:"age_limit"`
	Portrait    *v1Image           `json:"portrait"`
	Banner      *v1Image           `json:"banner"`
	Updated     string             `json:"updated"`
	Meta        *v1Meta            `json:"meta"`
	Intro       *v1Intro           `json:"intro"`
	Officials   *[]v1OfficialBrief `json:"officials"`
	// Refs are the work's external identities keyed by catalog source
	// ("dlsite" → the DLsite workno verbatim). Open map, not named fields: the
	// source registry grows and an unknown key must be carried, not rejected.
	Refs map[string]string `json:"refs"`
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
	// External identities (kungal reads refs.dlsite for the 补票 purchase link).
	b.Refs = it.Refs
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

// ─── /v1 detail-level wire (W1d enrichment: refs galgame_count + official
//     link/aliases, contributors, links user_id, series created/updated) ──────

// v1Intro is the /v1 include=intro block (empty → null per locale).
type v1Intro struct {
	ZhCN *string `json:"zh-cn"`
	EnUS *string `json:"en-us"`
	JaJP *string `json:"ja-jp"`
	ZhTW *string `json:"zh-tw"`
}

// v1Images is the /v1 detail images block (covers/screenshots present under their
// include tokens; all entries NSFW-filtered on the sfw face).
type v1Images struct {
	Banner      *v1Image  `json:"banner"`
	Portrait    *v1Image  `json:"portrait"`
	Covers      []v1Image `json:"covers"`
	Screenshots []v1Image `json:"screenshots"`
}

// v1TagRef / v1OfficialRef / v1EngineRef are the include=*_refs entries (W1d:
// galgame_count on all three; official also carries link + aliases).
type v1TagRef struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	SpoilerLevel int    `json:"spoiler_level"`
	GalgameCount int    `json:"galgame_count"`
}

type v1OfficialRef struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Lang         string   `json:"lang"`
	Link         string   `json:"link"`
	Aliases      []string `json:"aliases"`
	GalgameCount int      `json:"galgame_count"`
}

type v1EngineRef struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	GalgameCount int    `json:"galgame_count"`
}

// v1Taxonomy is the /v1 include=taxonomy block + the rich ref sub-keys.
type v1Taxonomy struct {
	SeriesID     *int             `json:"series_id"`
	TagRefs      *[]v1TagRef      `json:"tag_refs"`
	OfficialRefs *[]v1OfficialRef `json:"official_refs"`
	EngineRefs   *[]v1EngineRef   `json:"engine_refs"`
}

// v1Link is one include=links entry (W1d added user_id for the banned-author
// filter; source_key stays unsurfaced).
type v1Link struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Link   string `json:"link"`
	Source string `json:"source"`
	UserID int    `json:"user_id"`
}

// v1Contributor is one include=contributors entry (W1d): the user id only.
type v1Contributor struct {
	UserID int `json:"user_id"`
}

// v1Detail is the /v1 aggregate detail record (GET /v1/galgame/{id} data).
type v1Detail struct {
	ID               int              `json:"id"`
	Names            v1Names          `json:"names"`
	Aliases          []string         `json:"aliases"`
	Intro            *v1Intro         `json:"intro"`
	ReleaseDate      *string          `json:"release_date"`
	ReleaseDateTBA   bool             `json:"release_date_tba"`
	OriginalLanguage string           `json:"original_language"`
	AgeLimit         string           `json:"age_limit"`
	Images           v1Images         `json:"images"`
	Taxonomy         *v1Taxonomy      `json:"taxonomy"`
	Links            *[]v1Link        `json:"links"`
	Contributors     *[]v1Contributor `json:"contributors"`
	Meta             *v1Meta          `json:"meta"`
	Updated          string           `json:"updated"`
	// Refs are the work's external identities keyed by catalog source
	// ("dlsite" → the DLsite workno verbatim). See GalgameBrief.Refs.
	Refs map[string]string `json:"refs"`
}

// derefInt returns *p, or 0 when p is nil — the /v1 rating levels are pointer
// ints (absent when the credential is not nsfw-capable); the kungal row fields
// are plain ints.
func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// aliasStringsToNextMoe wraps a /v1 curated []string alias set into the
// []NextMoeAlias shape the detail struct declares (the kungal mapper reads the
// names back out via nextMoeAliasesToNames).
func aliasStringsToNextMoe(names []string) []dto.NextMoeAlias {
	out := make([]dto.NextMoeAlias, 0, len(names))
	for _, n := range names {
		out = append(out, dto.NextMoeAlias{Name: n})
	}
	return out
}

// v1CoversToNextMoe / v1ScreenshotsToNextMoe map /v1 detail images to the kungal
// cover/screenshot rows: image_hash from the sharded url, sort_order from the
// (server-ordered) index, sexual/violence from the W1c scope-gated levels,
// caption (screenshots), cdn_url = the /v1 url, dims/thumbhash/kind pass through.
// source/source_key have no /v1 source and stay "" (FE-unconsumed, W4 census).
func v1CoversToNextMoe(covers []v1Image) []dto.NextMoeGalgameCover {
	out := make([]dto.NextMoeGalgameCover, 0, len(covers))
	for i := range covers {
		out = append(out, dto.NextMoeGalgameCover{
			ImageHash: hashFromURL(covers[i].URL),
			SortOrder: i,
			Sexual:    derefInt(covers[i].Sexual),
			Violence:  derefInt(covers[i].Violence),
			Kind:      covers[i].Kind,
			CDNURL:    covers[i].URL,
			Width:     covers[i].Width,
			Height:    covers[i].Height,
			Thumbhash: covers[i].Thumbhash,
		})
	}
	return out
}

func v1ScreenshotsToNextMoe(shots []v1Image) []dto.NextMoeGalgameScreenshot {
	out := make([]dto.NextMoeGalgameScreenshot, 0, len(shots))
	for i := range shots {
		out = append(out, dto.NextMoeGalgameScreenshot{
			ImageHash: hashFromURL(shots[i].URL),
			SortOrder: i,
			Caption:   shots[i].Caption,
			Sexual:    derefInt(shots[i].Sexual),
			Violence:  derefInt(shots[i].Violence),
			CDNURL:    shots[i].URL,
			Width:     shots[i].Width,
			Height:    shots[i].Height,
			Thumbhash: shots[i].Thumbhash,
		})
	}
	return out
}

// v1DetailToFull projects the /v1 aggregate detail onto dto.NextMoeGalgameDetailFull
// (the shape galgameDetailFromNextMoe consumes). The users map is NOT carried by
// /v1 — the service hydrates author + contributors from OAuth itself.
func v1DetailToFull(d *v1Detail) dto.NextMoeGalgameDetailFull {
	f := dto.NextMoeGalgameDetailFull{
		ID:               d.ID,
		Refs:             d.Refs,
		NameEnUs:         str(d.Names.EnUS),
		NameJaJp:         str(d.Names.JaJP),
		NameZhCn:         str(d.Names.ZhCN),
		NameZhTw:         str(d.Names.ZhTW),
		ReleaseDate:      d.ReleaseDate,
		ReleaseDateTBA:   d.ReleaseDateTBA,
		OriginalLanguage: d.OriginalLanguage,
		AgeLimit:         d.AgeLimit,
		Updated:          d.Updated,
		Alias:            aliasStringsToNextMoe(d.Aliases),
		Covers:           v1CoversToNextMoe(d.Images.Covers),
		Screenshots:      v1ScreenshotsToNextMoe(d.Images.Screenshots),
	}
	if d.Intro != nil {
		f.IntroEnUs = str(d.Intro.EnUS)
		f.IntroJaJp = str(d.Intro.JaJP)
		f.IntroZhCn = str(d.Intro.ZhCN)
		f.IntroZhTw = str(d.Intro.ZhTW)
	}
	if d.Meta != nil {
		f.VndbID = d.Meta.VNDBID
		f.ContentLimit = d.Meta.ContentLimit
		f.ResourceUpdateTime = resourceUpdateTime(d.Meta)
		f.UserID = d.Meta.UserID
		f.SeriesID = d.Meta.SeriesID
		f.Status = d.Meta.Status
		f.Created = str(d.Meta.Created)
	}
	f.EffectiveBannerHash, f.EffectiveBannerURL,
		f.EffectiveBannerWidth, f.EffectiveBannerHeight,
		f.EffectiveBannerThumbhash = bannerFields(d.Images.Banner)
	if d.Taxonomy != nil {
		if d.Taxonomy.TagRefs != nil {
			for _, t := range *d.Taxonomy.TagRefs {
				f.Tag = append(f.Tag, dto.NextMoeTagWithSpoiler{
					SpoilerLevel: t.SpoilerLevel,
					Tag:          dto.NextMoeTag{ID: t.ID, Name: t.Name, Category: t.Category, GalgameCount: t.GalgameCount},
				})
			}
		}
		if d.Taxonomy.OfficialRefs != nil {
			for _, o := range *d.Taxonomy.OfficialRefs {
				f.Official = append(f.Official, dto.NextMoeOfficialRel{Official: dto.NextMoeOfficial{
					ID: o.ID, Name: o.Name, Link: o.Link, Category: o.Category, Lang: o.Lang,
					Alias: aliasStringsToNextMoe(o.Aliases), GalgameCount: o.GalgameCount,
				}})
			}
		}
		if d.Taxonomy.EngineRefs != nil {
			for _, e := range *d.Taxonomy.EngineRefs {
				var ew dto.NextMoeEngineWithAlias
				ew.Engine.ID = e.ID
				ew.Engine.Name = e.Name
				ew.Engine.Alias = []string{} // engine alias has no /v1 ref source (FE-unconsumed on detail)
				ew.Engine.GalgameCount = e.GalgameCount
				f.Engine = append(f.Engine, ew)
			}
		}
	}
	if d.Contributors != nil {
		for _, c := range *d.Contributors {
			f.Contributor = append(f.Contributor, dto.NextMoeContributor{UserID: c.UserID})
		}
	}
	return f
}

// ─── /v1 series wire + entry (metadata + timestamps + thin member previews) ───

// v1SeriesEntity is one /v1 series record (list item or /series/{id} detail):
// curated metadata + gated galgame_count + created/updated (W1d) + thin member
// previews (carrying meta under include=meta).
type v1SeriesEntity struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	GalgameCount int      `json:"galgame_count"`
	Galgames     []V1Item `json:"galgames"`
	Created      string   `json:"created"`
	Updated      string   `json:"updated"`
}

// v1SeriesListData is the /v1 series list envelope.
type v1SeriesListData struct {
	Items []v1SeriesEntity `json:"items"`
	Total int64            `json:"total"`
}

// NextMoeSeriesEntry is the client-mapped /v1 series record the SeriesService /
// galgame-detail / rating-detail series consumers share: metadata + timestamps +
// member previews projected to NextMoeGalgameItem (so their FilterSFW / Samples /
// ToCards logic works unchanged).
type NextMoeSeriesEntry struct {
	ID           int
	Name         string
	Description  string
	GalgameCount int
	Galgame      []dto.NextMoeGalgameItem
	Created      string
	Updated      string
}

func v1SeriesEntityToEntry(e *v1SeriesEntity) NextMoeSeriesEntry {
	entry := NextMoeSeriesEntry{
		ID: e.ID, Name: e.Name, Description: e.Description,
		GalgameCount: e.GalgameCount, Created: e.Created, Updated: e.Updated,
	}
	for i := range e.Galgames {
		entry.Galgame = append(entry.Galgame, V1ItemToNextMoeItem(&e.Galgames[i]))
	}
	return entry
}

// V1ItemToDetailBrief projects a /v1 thin item carrying include=intro,officials,
// meta onto the GalgameDetailBrief the activity-feed / quiz detail cards consume.
// The embedded brief's release_date is the thin-item value (null on the batch
// face — a pre-existing /v1 limitation; the plain brief already omits it too).
func V1ItemToDetailBrief(it *V1Item) GalgameDetailBrief {
	b := GalgameDetailBrief{GalgameBrief: V1ItemToBrief(it)}
	if it.Intro != nil {
		b.IntroEnUS = str(it.Intro.EnUS)
		b.IntroJaJP = str(it.Intro.JaJP)
		b.IntroZhCN = str(it.Intro.ZhCN)
		b.IntroZhTW = str(it.Intro.ZhTW)
	}
	if it.Officials != nil {
		for _, o := range *it.Officials {
			b.Officials = append(b.Officials, o.Name)
		}
	}
	return b
}
