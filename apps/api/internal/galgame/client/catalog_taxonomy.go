package client

// The catalog taxonomy read lanes — labels (会社/品牌), canonical tags and
// engines — plus the entity search that backs the pickers.
//
// Vocabulary note: the deprecated face called a maker an "official"; the
// catalog calls the same thing a LABEL. The kungal product wording (会社) is
// unchanged, so the rename lives only in this package and the URL space.
//
// All three list lanes are keyset (cursor) rather than page/offset, and every
// envelope now carries `total` (A2-1e), which is what lets the kungal list
// endpoints keep their {items, total} contract and their pager.

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"kun-galgame-api/pkg/errors"
)

// CatalogTaxonomyItem is one row of any of the three browse lanes, flattened to
// the union of their fields. Kind/Tier are tag- and label-specific; Description
// and Aliases are engine-specific on the LIST lane (the engine facet is a few
// hundred hand-curated rows, so the face ships them inline).
type CatalogTaxonomyItem struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Kind        string   `json:"kind"`
	Tier        string   `json:"tier"`
	WorkCount   int      `json:"work_count"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
}

// Label returns the row's display label, papering over the two naming
// conventions (labels use display_name, tags/engines use name).
func (t CatalogTaxonomyItem) Label() string {
	if t.DisplayName != "" {
		return t.DisplayName
	}
	return t.Name
}

// CatalogTaxonomyPage is one page of a browse lane.
type CatalogTaxonomyPage struct {
	Items      []CatalogTaxonomyItem `json:"items"`
	NextCursor *string               `json:"next_cursor"`
	Total      int64                 `json:"total"`
}

// CatalogIntro is one language's description of a taxonomy entity.
type CatalogIntro struct {
	Lang   string `json:"lang"`
	Intro  string `json:"intro"`
	Source string `json:"source"`
}

// CatalogLabelDetail is the full label record.
type CatalogLabelDetail struct {
	ID          int64          `json:"id"`
	DisplayName string         `json:"display_name"`
	Kind        string         `json:"kind"`
	Lang        string         `json:"lang"`
	Aliases     []string       `json:"aliases"`
	WorkCount   int            `json:"work_count"`
	Intros      []CatalogIntro `json:"intros"`
	Links       []struct {
		Source string `json:"source"`
		URL    string `json:"url"`
	} `json:"links"`
}

// CatalogTagDetail is the full canonical-tag record.
type CatalogTagDetail struct {
	ID        int64          `json:"id"`
	Name      string         `json:"name"`
	Tier      string         `json:"tier"`
	Kind      string         `json:"kind"`
	WorkCount int            `json:"work_count"`
	Intros    []CatalogIntro `json:"intros"`
}

// CatalogEngineDetail is the full engine record.
type CatalogEngineDetail struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
	WorkCount   int      `json:"work_count"`
}

// CatalogTaxonomyList pages one browse lane. entity is "labels" | "tags" |
// "engines".
func (c *GalgameClient) CatalogTaxonomyList(ctx context.Context, entity string, q url.Values) (*CatalogTaxonomyPage, *errors.AppError) {
	data, appErr := c.GetV1(ctx, "/catalog/"+entity, q)
	if appErr != nil {
		return nil, appErr
	}
	var page CatalogTaxonomyPage
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 词表响应失败")
	}
	return &page, nil
}

// CatalogTaxonomyPageAt walks the keyset lane to the requested 1-based page.
//
// The kungal list endpoints publish {items, total} with page/limit, and the
// catalog lane is cursor-based, so the offset has to be walked. That is cheap
// where it matters — these pages are browsed from the front, and the walk uses
// the face's maximum page size, so reaching kungal page N costs
// ceil(N*limit/100) upstream calls. A caller that jumps deep pays for it once;
// nothing else does.
func (c *GalgameClient) CatalogTaxonomyPageAt(ctx context.Context, entity string, base url.Values, page, limit int) ([]CatalogTaxonomyItem, int64, *errors.AppError) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	skip := (page - 1) * limit
	cursor := ""
	var total int64
	collected := make([]CatalogTaxonomyItem, 0, limit)

	for {
		q := url.Values{}
		for k, v := range base {
			q[k] = v
		}
		q.Set("limit", strconv.Itoa(catalogIDsChunk))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		res, appErr := c.CatalogTaxonomyList(ctx, entity, q)
		if appErr != nil {
			return nil, 0, appErr
		}
		total = res.Total
		for i := range res.Items {
			if skip > 0 {
				skip--
				continue
			}
			if len(collected) < limit {
				collected = append(collected, res.Items[i])
			}
		}
		if len(collected) >= limit || res.NextCursor == nil || *res.NextCursor == "" {
			break
		}
		cursor = *res.NextCursor
	}
	return collected, total, nil
}

// CatalogLabel / CatalogTag / CatalogEngine fetch one full taxonomy record.
// found=false on an unknown id (a 404, which is not an error for a page that
// wants to render its own not-found state).
func (c *GalgameClient) CatalogLabel(ctx context.Context, id string, isSFW bool) (*CatalogLabelDetail, bool, *errors.AppError) {
	var rec CatalogLabelDetail
	found, appErr := c.catalogTaxonomyDetail(ctx, "labels", id, isSFW, &rec)
	return &rec, found, appErr
}

func (c *GalgameClient) CatalogTag(ctx context.Context, id string, isSFW bool) (*CatalogTagDetail, bool, *errors.AppError) {
	var rec CatalogTagDetail
	found, appErr := c.catalogTaxonomyDetail(ctx, "tags", id, isSFW, &rec)
	return &rec, found, appErr
}

func (c *GalgameClient) CatalogEngine(ctx context.Context, id string, isSFW bool) (*CatalogEngineDetail, bool, *errors.AppError) {
	var rec CatalogEngineDetail
	found, appErr := c.catalogTaxonomyDetail(ctx, "engines", id, isSFW, &rec)
	return &rec, found, appErr
}

// catalogTaxonomyDetail is the shared fetch+decode for the three detail lanes.
// The nsfw gate rides along because work_count is nsfw-aware: a count that
// disagreed with the member list the same caller can page is exactly the defect
// the deprecated face's permanently-zero galgame_count was.
func (c *GalgameClient) catalogTaxonomyDetail(ctx context.Context, entity, id string, isSFW bool, out any) (bool, *errors.AppError) {
	q := url.Values{}
	applyNSFW(q, contentLimitFor(isSFW))
	data, appErr := c.GetV1(ctx, "/catalog/"+entity+"/"+id, q)
	if appErr != nil {
		if appErr.StatusCode == 404 {
			return false, nil
		}
		return false, appErr
	}
	if err := json.Unmarshal(data, out); err != nil {
		return false, errors.ErrInternal("解析 Catalog 词表详情响应失败")
	}
	return true, nil
}

// CatalogEntityHit is one entity-search hit.
type CatalogEntityHit struct {
	ID         int64  `json:"id"`
	EntityType string `json:"entity_type"`
	Name       string `json:"name"`
}

// CatalogEntitySearch runs the shared entity search. searchType is the face's
// own vocabulary: "labels" | "tags" | "names" | "characters" | "works".
//
// The hit shape is identity-only (id + name) by design — it is a picker feed.
// Consumers that need counts or metadata follow the id to the detail lane.
func (c *GalgameClient) CatalogEntitySearch(ctx context.Context, searchType, keywords string, limit int, isSFW bool) ([]CatalogEntityHit, *errors.AppError) {
	q := url.Values{
		"type":  {searchType},
		"q":     {keywords},
		"limit": {strconv.Itoa(limit)},
	}
	applyNSFW(q, contentLimitFor(isSFW))
	data, appErr := c.GetV1(ctx, "/catalog/search", q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed struct {
		Items []CatalogEntityHit `json:"items"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 实体搜索响应失败")
	}
	if parsed.Items == nil {
		parsed.Items = []CatalogEntityHit{}
	}
	return parsed.Items, nil
}

// ─── the surviving wiki faces ────────────────────────────────────────────

// GalgameMetaRow is one row of GET /internal/galgame/meta — the ownership +
// lifecycle projection of a galgame, STATUS-BLIND (unlike every public read).
//
// This is the honest supply for the two questions the edit lane asks about an
// entry kungal holds no copy of: may this user edit it, and who gets notified.
// Reading them off a published-only endpoint silently locked the true owner out
// of their own unpublished entry. The four names are here because the same
// notifications carry the entry's title (doc 106 §26 R2 ①).
type GalgameMetaRow struct {
	GID      int    `json:"gid"`
	UserID   int    `json:"user_id"`
	Status   int    `json:"status"`
	NameZhCN string `json:"name_zh_cn"`
	NameZhTW string `json:"name_zh_tw"`
	NameJaJP string `json:"name_ja_jp"`
	NameEnUS string `json:"name_en_us"`
}

// Name resolves the row's display title with the kungal fallback chain
// zh-CN → zh-TW → ja-JP → en-US. en-US is last on purpose: it is usually the
// VNDB romaji title, which must never win over a real Chinese or Japanese name.
func (m GalgameMetaRow) Name() string {
	return firstNonEmptyStr(m.NameZhCN, m.NameZhTW, m.NameJaJP, m.NameEnUS)
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// GalgameMeta reads ownership meta for up to 100 gids from the surviving
// /internal face. Ids that do not resolve are absent from the map — the caller
// distinguishes "no such entry" from "not the owner" by presence.
func (c *GalgameClient) GalgameMeta(ctx context.Context, gids []int) (map[int]GalgameMetaRow, *errors.AppError) {
	out := make(map[int]GalgameMetaRow, len(gids))
	if len(gids) == 0 {
		return out, nil
	}
	for start := 0; start < len(gids); start += catalogIDsChunk {
		end := min(start+catalogIDsChunk, len(gids))
		raw := make([]string, 0, end-start)
		for _, gid := range gids[start:end] {
			raw = append(raw, strconv.Itoa(gid))
		}
		data, appErr := c.getFace(ctx, c.internalBase, "/galgame/meta", "",
			url.Values{"ids": {joinCSV(raw)}}, c.apiKey)
		if appErr != nil {
			return nil, appErr
		}
		var parsed struct {
			Items []GalgameMetaRow `json:"items"`
		}
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, errors.ErrInternal("解析 Galgame 元信息响应失败")
		}
		for _, row := range parsed.Items {
			out[row.GID] = row
		}
	}
	return out, nil
}

// StaffTaxonomyRecord is the /api staff read-back for a taxonomy edit form —
// exactly the matching update payload's editable set, so the console can no
// longer silently erase a field it could not read (the defect this op exists
// for). Ids here are WIKI ids end to end: the staff edit lane keeps the wiki
// key space until the editing engine retires this face (doc 106 R11).
type StaffTaxonomyRecord struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Original    string   `json:"original"`
	Link        string   `json:"link"`
	Lang        string   `json:"lang"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Alias       []string `json:"alias"`
	GalgameIDs  []int    `json:"galgame_ids"`
}

// StaffTaxonomyDetail reads one taxonomy record back off the staff face.
// family is "tag" | "official" | "engine" | "series". The caller's Bearer is
// forwarded: the face is gated by jwtAuth + the taxonomy edit permission.
func (c *GalgameClient) StaffTaxonomyDetail(ctx context.Context, family, id, token string) (*StaffTaxonomyRecord, *errors.AppError) {
	data, appErr := c.getFace(ctx, c.legacyBase, "/"+family+"/"+id, token, nil, "")
	if appErr != nil {
		return nil, appErr
	}
	var rec StaffTaxonomyRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 词表编辑数据失败")
	}
	return &rec, nil
}

// TaxonomyPickerSearch runs the taxonomy picker on the surviving /internal
// face. family is "tag" | "official" | "engine" | "series".
//
// The /internal door rather than the /api one is the whole point (A2-1g): both
// answer the same query with the same {id, name} rows, but this one is gated on
// being SIGNED IN rather than on taxonomy.edit_any — so an ordinary contributor
// filling in the submission form can resolve a tag to the wiki id their write
// payload has to carry. Nothing here is more sensitive than what the same user
// is about to write: these are the names of public taxonomy rows.
//
// Dual-credential transport: the service X-API-Key plus the caller's Bearer,
// which is what the face reads the signed-in identity from.
//
// Blank-query behaviour is a property of the FAMILY, not of this call: tag and
// official return an empty list (open-ended vocabularies — type something),
// engine and series return their whole small curated set.
func (c *GalgameClient) TaxonomyPickerSearch(ctx context.Context, family, keywords, token string) ([]StaffTaxonomyRecord, *errors.AppError) {
	data, appErr := c.getFace(ctx, c.internalBase, "/galgame/taxonomy/"+family+"/search", token,
		url.Values{"q": {keywords}}, c.apiKey)
	if appErr != nil {
		return nil, appErr
	}
	var parsed struct {
		Items []StaffTaxonomyRecord `json:"items"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 词表搜索响应失败")
	}
	if parsed.Items == nil {
		parsed.Items = []StaffTaxonomyRecord{}
	}
	return parsed.Items, nil
}

// joinCSV joins ids without pulling in strings.Join's slice copy at every call
// site; kept local so the client has no import churn.
func joinCSV(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return out
}

// LookupWikiLabel resolves a legacy wiki 会社 id (galgame_official PK) to its
// catalog label id, so the old /galgame-official/{wikiId} URLs can 301 onto the
// new catalog-id space.
//
// Unlike tags and engines — whose migrations were frozen into a static map —
// makers resolve at RUNTIME through the registry's own external-ref index. The
// A2-0 registrar covered 100% of them, and the registry keeps that mapping
// correct across future merges, which a vendored file could not.
//
// found=false on an unregistered id (the caller 404s).
func (c *GalgameClient) LookupWikiLabel(ctx context.Context, wikiID int) (int64, bool, *errors.AppError) {
	q := url.Values{
		"source":      {siteGalgameWiki},
		"external_id": {strconv.Itoa(wikiID)},
		"type":        {"label"},
		// Identity resolution is not content: gating it would make an r18
		// maker's old URL 404 rather than redirect.
		"nsfw": {"1"},
	}
	data, appErr := c.GetV1(ctx, "/catalog/lookup", q)
	if appErr != nil {
		if appErr.StatusCode == 404 {
			return 0, false, nil
		}
		return 0, false, appErr
	}
	var parsed struct {
		Label *struct {
			ID int64 `json:"id"`
		} `json:"label"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return 0, false, errors.ErrInternal("解析 Catalog 反查响应失败")
	}
	if parsed.Label == nil {
		return 0, false, nil
	}
	return parsed.Label.ID, true, nil
}
