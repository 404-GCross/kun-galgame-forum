package client

// The catalog public face (`{base}/v1/catalog/*`) — works browse / batch /
// detail / search / calendar. See catalog_wire.go for the wire shapes and the
// projections back onto kungal's own DTOs.
//
// The one structural cost of re-anchoring is the ID BRIDGE. kungal is keyed on
// the wiki gid end to end (local rows, resources, ratings, collections, the
// /galgame/{gid} URL) and doc 106 R3 rules that it stays that way — no dual-id
// model, no data migration. So every call that starts from a kungal gid takes
// two hops:
//
//	gid ──POST /v1/catalog/lookup/batch──▶ catalog work id
//	    ──GET  /v1/catalog/works?ids=───▶ the row
//
// and every row that comes back maps home through `claimed_by.work_id`. The
// first hop is memoized (identity mappings essentially never change), so the
// steady state is one extra round-trip per cold batch, not per request.

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/pkg/errors"
)

// catalogIDsChunk is the hard per-call ceiling shared by the batch lookup and
// the works ids= filter. Both 400 above it (the deprecated face used to
// truncate silently), so the client chunks rather than letting a caller
// discover the limit as a failure.
const catalogIDsChunk = 100

// Lookup memo TTLs. A HIT is a registry identity — stable for the lifetime of
// both rows — so it is cached long. A MISS is not stable: the nightly
// wikirescue pass registers newly created wiki entries (doc 106 R6), so a miss
// must expire quickly enough that a new galgame becomes visible the same day
// rather than at the next deploy.
const (
	gidLookupHitTTL  = 30 * time.Minute
	gidLookupMissTTL = 2 * time.Minute
)

type gidLookupEntry struct {
	catalogID int64
	found     bool
	expire    time.Time
}

// catalogInclude is the include= vocabulary the kungal brief needs: the four
// localized names, the two cover slots, and the identity anchors (refs powers
// the DLsite 补票 link). Lanes that additionally render an intro / maker list
// ask for catalogDetailBriefInclude.
const (
	catalogBriefInclude       = "names,covers,refs"
	catalogDetailBriefInclude = "names,intros,labels,covers,refs"
)

// nsfwParam translates the kungal content_limit convention to the catalog
// face's boolean gate. kungal's "" means "no filter" (the permissive default
// the deprecated bridge had), and both "" and the explicit opt-ins map to
// nsfw=1; only "sfw" leaves the gate closed.
//
// Filtering MUST stay on this side of the wire — the handbook forbids
// downstream post-filtering, and the catalog face is the only place that can
// drop an r18 row before it leaves the boundary.
func nsfwParam(contentLimit string) string {
	if contentLimit == "sfw" {
		return ""
	}
	return "1"
}

// applyNSFW sets nsfw=1 on q when the content limit opts into adult content.
func applyNSFW(q url.Values, contentLimit string) url.Values {
	if v := nsfwParam(contentLimit); v != "" {
		q.Set("nsfw", v)
	}
	return q
}

// contentLimitFor is the isSFW→content_limit convention shared by every public
// read path.
func contentLimitFor(isSFW bool) string {
	if isSFW {
		return "sfw"
	}
	return "all"
}

// ─── the id bridge ───────────────────────────────────────────────────────

// catalogIDsForGIDs resolves kungal gids to catalog work ids through the batch
// external-id lookup, memoized per gid. Unresolvable gids are simply absent
// from the result (a wiki entry the registrar has not reached yet, or a hard
// deleted one) — callers treat absence as "no such work", never as an error.
//
// The lookup always runs with nsfw=1: it resolves an IDENTITY, and gating it
// by the caller's content preference would make an r18 game unresolvable
// rather than merely invisible. The visibility gate belongs on the row fetch
// that follows, which is where it is applied.
func (c *GalgameClient) catalogIDsForGIDs(ctx context.Context, gids []int) (map[int]int64, *errors.AppError) {
	out := make(map[int]int64, len(gids))
	var missing []int
	now := time.Now()

	c.gidMu.RLock()
	for _, gid := range gids {
		if e, ok := c.gidCache[gid]; ok && now.Before(e.expire) {
			if e.found {
				out[gid] = e.catalogID
			}
		} else {
			missing = append(missing, gid)
		}
	}
	c.gidMu.RUnlock()
	if len(missing) == 0 {
		return out, nil
	}

	resolved := make(map[int]int64, len(missing))
	for start := 0; start < len(missing); start += catalogIDsChunk {
		end := min(start+catalogIDsChunk, len(missing))
		chunk := missing[start:end]

		items := make([]map[string]string, 0, len(chunk))
		for _, gid := range chunk {
			items = append(items, map[string]string{
				"source":      siteGalgameWiki,
				"external_id": strconv.Itoa(gid),
				"type":        "work",
			})
		}
		data, appErr := c.postFace(ctx, c.v1Base, "/catalog/lookup/batch",
			url.Values{"nsfw": {"1"}}, map[string]any{"items": items}, c.apiKey)
		if appErr != nil {
			return nil, appErr
		}
		var parsed struct {
			Items []struct {
				ExternalID string `json:"external_id"`
				Work       *struct {
					ID int64 `json:"id"`
				} `json:"work"`
			} `json:"items"`
		}
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, errors.ErrInternal("解析 Catalog 批量解析响应失败")
		}
		for _, it := range parsed.Items {
			if it.Work == nil {
				continue
			}
			gid, err := strconv.Atoi(it.ExternalID)
			if err != nil {
				continue
			}
			resolved[gid] = it.Work.ID
		}
	}

	c.gidMu.Lock()
	if len(c.gidCache) > batchCacheMaxEntries {
		clear(c.gidCache)
	}
	for _, gid := range missing {
		id, ok := resolved[gid]
		ttl := gidLookupMissTTL
		if ok {
			ttl = gidLookupHitTTL
			out[gid] = id
		}
		c.gidCache[gid] = gidLookupEntry{catalogID: id, found: ok, expire: now.Add(ttl)}
	}
	c.gidMu.Unlock()
	return out, nil
}

// worksByCatalogIDs fetches catalog rows by catalog id, chunked at the face's
// own ceiling. limit is pinned to the chunk size: the works lane defaults to 20
// and would otherwise silently return the first 20 of a 100-id request.
func (c *GalgameClient) worksByCatalogIDs(ctx context.Context, ids []int64, include, contentLimit string) ([]CatalogWorkListItem, *errors.AppError) {
	var out []CatalogWorkListItem
	for start := 0; start < len(ids); start += catalogIDsChunk {
		end := min(start+catalogIDsChunk, len(ids))
		chunk := ids[start:end]

		raw := make([]string, len(chunk))
		for i, id := range chunk {
			raw[i] = strconv.FormatInt(id, 10)
		}
		q := url.Values{
			"ids":   {strings.Join(raw, ",")},
			"limit": {strconv.Itoa(catalogIDsChunk)},
		}
		if include != "" {
			q.Set("include", include)
		}
		applyNSFW(q, contentLimit)

		data, appErr := c.GetV1(ctx, "/catalog/works", q)
		if appErr != nil {
			return nil, appErr
		}
		var parsed catWorksListData
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, errors.ErrInternal("解析 Catalog 作品列表响应失败")
		}
		out = append(out, parsed.Items...)
	}
	return out, nil
}

// CatalogRowsByGIDs is the whole bridge in one call: kungal gids in, catalog
// rows out, keyed back by gid. Rows whose claim the wiki has withdrawn
// (state=hidden) are dropped here — a single choke point, so no lane can
// forget it and resurrect a banned entry.
func (c *GalgameClient) CatalogRowsByGIDs(ctx context.Context, gids []int, include, contentLimit string) (map[int]CatalogWorkListItem, *errors.AppError) {
	if len(gids) == 0 {
		return map[int]CatalogWorkListItem{}, nil
	}
	idMap, appErr := c.catalogIDsForGIDs(ctx, gids)
	if appErr != nil {
		return nil, appErr
	}
	if len(idMap) == 0 {
		return map[int]CatalogWorkListItem{}, nil
	}
	ids := make([]int64, 0, len(idMap))
	for _, id := range idMap {
		ids = append(ids, id)
	}
	rows, appErr := c.worksByCatalogIDs(ctx, ids, include, contentLimit)
	if appErr != nil {
		return nil, appErr
	}
	out := make(map[int]CatalogWorkListItem, len(rows))
	for i := range rows {
		row := rows[i]
		if !row.isRenderable() {
			continue
		}
		if gid := row.gid(); gid > 0 {
			out[gid] = row
		}
	}
	return out, nil
}

// ─── works detail ────────────────────────────────────────────────────────

// catWorkDetail is the catalog aggregate work record (GET
// /v1/catalog/works/{id}). Only the blocks kungal renders are typed.
type catWorkDetail struct {
	ID            int64         `json:"id"`
	DisplayName   string        `json:"display_name"`
	OLang         string        `json:"olang"`
	ContentRating string        `json:"content_rating"`
	ReleaseDate   *string       `json:"release_date"`
	Updated       string        `json:"updated"`
	Created       string        `json:"created"`
	ClaimedBy     *catClaimedBy `json:"claimed_by"`

	Titles []struct {
		Lang  string `json:"lang"`
		Title string `json:"title"`
		Kind  string `json:"kind"`
	} `json:"titles"`
	Refs []catRef `json:"refs"`
	Tags []struct {
		Name        string `json:"name"`
		Source      string `json:"source"`
		CanonicalID int64  `json:"canonical_id"`
		Kind        string `json:"kind"`
		Spoiler     int    `json:"spoiler"`
		Sexual      bool   `json:"sexual"`
		// WorkCount is omitempty upstream — only a canonical-mapped row has a
		// count to report. An unmapped row therefore has NO key here, forever,
		// and decodes to 0; those rows are dropped from the chip strip anyway.
		WorkCount int `json:"work_count"`
	} `json:"tags"`
	Intro []struct {
		Lang    string `json:"lang"`
		Intro   string `json:"intro"`
		Machine bool   `json:"machine"`
	} `json:"intro"`
	Covers []struct {
		URL       string `json:"url"`
		Kind      string `json:"kind"`
		Sexual    int    `json:"sexual"`
		Violence  int    `json:"violence"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Thumbhash string `json:"thumbhash"`
	} `json:"covers"`
	Screenshots []struct {
		URL       string `json:"url"`
		Caption   string `json:"caption"`
		Sexual    int    `json:"sexual"`
		Violence  int    `json:"violence"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Thumbhash string `json:"thumbhash"`
	} `json:"screenshots"`
	Labels  []catWorkLabel  `json:"labels"`
	Engines []catWorkEngine `json:"engines"`
	Links   []catWorkLink   `json:"links"`
}

// CatalogWorkDetail fetches one work's full aggregate by kungal gid. found is
// false when the gid resolves to nothing, when the row is gated away by the
// caller's content preference, or when the wiki has withdrawn the claim.
func (c *GalgameClient) CatalogWorkDetail(ctx context.Context, gid int, contentLimit string) (*catWorkDetail, bool, *errors.AppError) {
	idMap, appErr := c.catalogIDsForGIDs(ctx, []int{gid})
	if appErr != nil {
		return nil, false, appErr
	}
	catalogID, ok := idMap[gid]
	if !ok {
		return nil, false, nil
	}
	// spoilers=0 keeps the character roster's spoiler traits closed; the tag
	// edges carry their own per-edge spoiler level for the FE to gate on.
	q := url.Values{"spoilers": {"0"}}
	applyNSFW(q, contentLimit)
	data, appErr := c.GetV1(ctx, "/catalog/works/"+strconv.FormatInt(catalogID, 10), q)
	if appErr != nil {
		if appErr.StatusCode == 404 {
			return nil, false, nil
		}
		return nil, false, appErr
	}
	var d catWorkDetail
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, false, errors.ErrInternal("解析 Catalog 作品详情响应失败")
	}
	if d.ClaimedBy != nil && d.ClaimedBy.State == claimStateHidden {
		return nil, false, nil
	}
	return &d, true, nil
}

// ─── browse / search / calendar ──────────────────────────────────────────

// CatalogWorksPage is one page of the browse or search lane, already projected.
type CatalogWorksPage struct {
	Items      []CatalogWorkListItem
	NextCursor string
	Total      int64
	Count      int64
	Month      string
	Year       string
	Meta       catCalendarMeta
}

// CatalogWorksList runs the keyset browse lane with an arbitrary filter set.
// Used by the taxonomy member reverse-lookup (works?tag_id= / label_id= /
// engine_id=), which replaced the deprecated face's {ent}/{id}/galgame-ids.
func (c *GalgameClient) CatalogWorksList(ctx context.Context, q url.Values) (*CatalogWorksPage, *errors.AppError) {
	data, appErr := c.GetV1(ctx, "/catalog/works", q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed catWorksListData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 作品列表响应失败")
	}
	page := &CatalogWorksPage{Items: parsed.Items}
	if parsed.NextCursor != nil {
		page.NextCursor = *parsed.NextCursor
	}
	return page, nil
}

// CatalogMemberGIDs walks the browse lane for every work under one taxonomy
// filter and returns the kungal gids among them, in catalog id order.
//
// The entity detail pages then run kungal's OWN local filter/sort/paginate over
// these ids, exactly as they did with the deprecated galgame-ids op — so the
// page keeps working with local resource filters the catalog knows nothing
// about. Only CLAIMED works contribute: an unclaimed catalog work has no
// kungal row to filter, and no gid to key one by.
//
// The population is PUBLISHED members only (doc 106 §37, extended to the entity
// pages by R4): `claimed=true` alone admits an entry the wiki has claimed but
// never published — state=draft — which then rendered on the 词条 page as a
// 未发布 card. That is the entity-page instance of the search leak.
//
// pageCap bounds the walk; a taxonomy term with more members than that is
// truncated rather than allowed to fan out unboundedly on a request path.
func (c *GalgameClient) CatalogMemberGIDs(ctx context.Context, filter url.Values, isSFW bool, pageCap int) ([]int, *errors.AppError) {
	gids := []int{}
	cursor := ""
	for page := 0; page < pageCap; page++ {
		q := url.Values{}
		for k, v := range filter {
			q[k] = v
		}
		q.Set("limit", strconv.Itoa(catalogIDsChunk))
		// Only claimed works can map to a kungal gid, so let the face do that
		// filtering instead of paging through works we would discard. The two
		// gates are NOT redundant while the supplying wave is undeployed: the
		// unknown claim_state is ignored then, and `claimed` is what still
		// bounds the walk to addressable rows.
		q.Set("claimed", "true")
		q.Set("claim_state", claimStateLive)
		applyNSFW(q, contentLimitFor(isSFW))
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		res, appErr := c.CatalogWorksList(ctx, q)
		if appErr != nil {
			return nil, appErr
		}
		for i := range res.Items {
			if !res.Items[i].isRenderable() {
				continue
			}
			if gid := res.Items[i].gid(); gid > 0 {
				gids = append(gids, gid)
			}
		}
		if res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}
	return gids, nil
}

// CatalogWorksSearch runs the product search lane (page-paginated, filters
// compiled into one upstream expression so total and items are gated
// identically — the deprecated face's content_limit trap by construction
// cannot recur here).
func (c *GalgameClient) CatalogWorksSearch(ctx context.Context, q url.Values) (*CatalogWorksPage, *errors.AppError) {
	data, appErr := c.GetV1(ctx, "/catalog/works/search", q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed catWorksSearchData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 搜索响应失败")
	}
	return &CatalogWorksPage{Items: parsed.Items, Total: parsed.Total}, nil
}

// CatalogCalendar fetches one calendar bucket. bucket is "" (month),
// "/pending" or "/tba".
func (c *GalgameClient) CatalogCalendar(ctx context.Context, bucket string, q url.Values) (*CatalogWorksPage, *errors.AppError) {
	data, appErr := c.GetV1(ctx, "/catalog/calendar"+bucket, q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed catWorksListData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 月历响应失败")
	}
	page := &CatalogWorksPage{
		Items: parsed.Items, Count: parsed.Count,
		Month: parsed.Month, Year: parsed.Year, Meta: parsed.Meta,
	}
	if parsed.NextCursor != nil {
		page.NextCursor = *parsed.NextCursor
	}
	return page, nil
}
