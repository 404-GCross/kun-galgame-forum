package client

// Contract tests for the catalog re-anchoring (A2-3). Hermetic: every test
// drives a httptest server, so nothing here needs a running catalog service.
//
// What they pin, and why each one is worth a test rather than a comment:
//
//   - the TWO-HOP id bridge actually happens, in order, with the right bodies
//     (a regression here silently returns an empty batch, which every caller
//     renders as "this game vanished");
//   - a withdrawn claim (state=hidden) NEVER reaches a caller — the single
//     sharpest finding of the A2-2 report was that re-anchoring on claimed_by
//     without this check republishes banned entries;
//   - a catalog id is never mistaken for a gid (the two key spaces overlap, so
//     this failure attaches another game's stats rather than erroring);
//   - the NSFW gate travels as a request parameter, never as a post-filter.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// catalogRecorder captures every request a fake catalog service received.
type catalogRecorder struct {
	mu    sync.Mutex
	paths []string
	query []url.Values
	body  []string
}

func (r *catalogRecorder) record(req *http.Request) {
	body := ""
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		body = string(b)
	}
	r.mu.Lock()
	r.paths = append(r.paths, req.URL.Path)
	r.query = append(r.query, req.URL.Query())
	r.body = append(r.body, body)
	r.mu.Unlock()
}

func (r *catalogRecorder) pathAt(i int) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if i >= len(r.paths) {
		return ""
	}
	return r.paths[i]
}

func (r *catalogRecorder) queryAt(i int) url.Values {
	r.mu.Lock()
	defer r.mu.Unlock()
	if i >= len(r.query) {
		return url.Values{}
	}
	return r.query[i]
}

func (r *catalogRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.paths)
}

// catalogStub serves the two bridge endpoints from canned data. lookup maps a
// gid to a catalog id; works maps a catalog id to a list-item JSON fragment.
func catalogStub(t *testing.T, rec *catalogRecorder, lookup map[string]int64, works map[int64]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rec.record(req)
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasSuffix(req.URL.Path, "/catalog/lookup/batch"):
			var body struct {
				Items []struct {
					Source     string `json:"source"`
					ExternalID string `json:"external_id"`
					Type       string `json:"type"`
				} `json:"items"`
			}
			raw := rec.body[len(rec.body)-1]
			_ = json.Unmarshal([]byte(raw), &body)
			items := make([]string, 0, len(body.Items))
			for _, it := range body.Items {
				if id, ok := lookup[it.ExternalID]; ok {
					items = append(items, `{"source":"galgame_wiki","external_id":"`+it.ExternalID+
						`","type":"work","work":{"id":`+itoa(id)+`}}`)
					continue
				}
				items = append(items, `{"source":"galgame_wiki","external_id":"`+it.ExternalID+
					`","type":"work","work":null}`)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[` + strings.Join(items, ",") + `]}}`))

		case strings.HasSuffix(req.URL.Path, "/catalog/works"):
			var rows []string
			for _, raw := range strings.Split(req.URL.Query().Get("ids"), ",") {
				id := atoi64(raw)
				if frag, ok := works[id]; ok {
					rows = append(rows, frag)
				}
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[` + strings.Join(rows, ",") + `],"next_cursor":null}}`))

		default:
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{}}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func itoa(v int64) string {
	out := ""
	if v == 0 {
		return "0"
	}
	for v > 0 {
		out = string(rune('0'+v%10)) + out
		v /= 10
	}
	return out
}

func atoi64(s string) int64 {
	var out int64
	for _, r := range strings.TrimSpace(s) {
		if r < '0' || r > '9' {
			return 0
		}
		out = out*10 + int64(r-'0')
	}
	return out
}

// liveRow builds a claimed, live catalog row whose gid differs from its catalog
// id — the shape that catches a catalog-id/gid mix-up.
func liveRow(catalogID int64, gid int, name string) string {
	return `{"id":` + itoa(catalogID) + `,"medium":"galgame","display_name":"` + name +
		`","content_rating":"all_ages","olang":"ja","release_date":"2024-06-14",` +
		`"claimed_by":{"site":"galgame_wiki","work_id":` + itoa(int64(gid)) + `,"state":"live"},` +
		`"updated":"2026-01-01T00:00:00Z","names":{"ja-jp":"` + name + `","zh-cn":"` + name + `CN"},` +
		`"covers":{"portrait":{"url":"https://cdn.example/ab/cd/abcdef.webp","width":600,"height":800,"thumbhash":"TH"},"banner":null},` +
		`"refs":[{"source":"dlsite","external_id":"RJ01"},{"source":"vndb","external_id":"v19658"}]}`
}

func TestCatalogBridge_TwoHopAndGIDKeying(t *testing.T) {
	rec := &catalogRecorder{}
	srv := catalogStub(t, rec,
		map[string]int64{"777": 4242},
		map[int64]string{4242: liveRow(4242, 777, "Kun")},
	)
	c := New(srv.URL, "nm_test_key", "")

	got, err := c.GetBatch(context.Background(), []int{777})
	if err != nil {
		t.Fatalf("GetBatch: %v", err)
	}

	// Hop 1 is the batch lookup, hop 2 is the works fetch — in that order.
	if p := rec.pathAt(0); p != "/v1/catalog/lookup/batch" {
		t.Errorf("first call = %q, want /v1/catalog/lookup/batch", p)
	}
	if p := rec.pathAt(1); p != "/v1/catalog/works" {
		t.Errorf("second call = %q, want /v1/catalog/works", p)
	}
	// The works fetch must ask for the CATALOG id, not the gid.
	if ids := rec.queryAt(1).Get("ids"); ids != "4242" {
		t.Errorf("works ids = %q, want 4242 (the catalog id, not the gid)", ids)
	}
	// Default limit is 20; a 100-id request would silently truncate without this.
	if lim := rec.queryAt(1).Get("limit"); lim != "100" {
		t.Errorf("works limit = %q, want 100", lim)
	}

	// The result is keyed by the GID, and the brief's own ID is the gid too.
	b, ok := got[777]
	if !ok {
		t.Fatalf("result not keyed by gid 777: %#v", got)
	}
	if b.ID != 777 {
		t.Errorf("brief.ID = %d, want 777 (the gid, never the catalog id)", b.ID)
	}
	if _, leaked := got[4242]; leaked {
		t.Error("result is keyed by the catalog id — the two id spaces overlap, so this attaches another game's local stats")
	}
	if b.NameJaJp != "Kun" || b.NameZhCn != "KunCN" {
		t.Errorf("names not projected: %+v", b)
	}
	if b.EffectiveBannerURL != "https://cdn.example/ab/cd/abcdef.webp" || b.EffectiveBannerThumbhash != "TH" {
		t.Errorf("portrait cover slot not projected: %+v", b)
	}
	if b.Refs["dlsite"] != "RJ01" {
		t.Errorf("refs not projected (the DLsite purchase link reads this): %+v", b.Refs)
	}
	if b.VndbID != "v19658" {
		t.Errorf("vndb_id = %q, want it derived from refs", b.VndbID)
	}
	if b.ContentLimit != "sfw" || b.AgeLimit != "all" {
		t.Errorf("content rating projection wrong: %+v", b)
	}
	if b.OriginalLanguage != "ja-jp" {
		t.Errorf("olang = %q, want the ja-jp product key", b.OriginalLanguage)
	}
	if b.Status != galgameStatusPublished {
		t.Errorf("status = %d, want published for a live claim", b.Status)
	}
}

func TestCatalogBridge_HiddenClaimNeverRenders(t *testing.T) {
	hidden := strings.Replace(liveRow(4242, 777, "Banned"), `"state":"live"`, `"state":"hidden"`, 1)
	rec := &catalogRecorder{}
	srv := catalogStub(t, rec,
		map[string]int64{"777": 4242},
		map[int64]string{4242: hidden},
	)
	c := New(srv.URL, "nm_test_key", "")

	got, err := c.GetBatch(context.Background(), []int{777})
	if err != nil {
		t.Fatalf("GetBatch: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("a withdrawn (state=hidden) claim reached the caller: %#v — this republishes banned entries", got)
	}
}

func TestCatalogBridge_UnresolvedGIDIsAbsentNotAnError(t *testing.T) {
	rec := &catalogRecorder{}
	srv := catalogStub(t, rec, map[string]int64{}, map[int64]string{})
	c := New(srv.URL, "nm_test_key", "")

	got, err := c.GetBatch(context.Background(), []int{999})
	if err != nil {
		t.Fatalf("an unregistered gid must not be an error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %#v, want empty", got)
	}
	// Nothing to fetch ⇒ the second hop must not fire at all.
	if rec.count() != 1 {
		t.Errorf("made %d calls, want 1 (no works fetch when the lookup resolved nothing)", rec.count())
	}
}

func TestCatalogBridge_NSFWGateIsAParameterNotAPostFilter(t *testing.T) {
	rec := &catalogRecorder{}
	srv := catalogStub(t, rec,
		map[string]int64{"777": 4242},
		map[int64]string{4242: liveRow(4242, 777, "Kun")},
	)
	c := New(srv.URL, "nm_test_key", "")

	// SFW caller: the works fetch must leave the gate closed…
	if _, err := c.GetBatchPublic(context.Background(), []int{777}, true); err != nil {
		t.Fatalf("GetBatchPublic sfw: %v", err)
	}
	if v := rec.queryAt(1).Get("nsfw"); v != "" {
		t.Errorf("sfw caller sent nsfw=%q, want it absent", v)
	}
	// …while the IDENTITY lookup always runs with nsfw=1, or an r18 game
	// becomes unresolvable rather than merely invisible.
	if v := rec.queryAt(0).Get("nsfw"); v != "1" {
		t.Errorf("lookup sent nsfw=%q, want 1 (identity resolution is not content)", v)
	}

	// NSFW caller: a different cache key, so a fresh pair of calls with the gate open.
	before := rec.count()
	if _, err := c.GetBatchPublic(context.Background(), []int{777}, false); err != nil {
		t.Fatalf("GetBatchPublic nsfw: %v", err)
	}
	if v := rec.queryAt(before).Get("nsfw"); v != "1" {
		t.Errorf("nsfw caller's works fetch sent nsfw=%q, want 1", v)
	}
}

func TestCatalogBridge_LookupIsMemoized(t *testing.T) {
	rec := &catalogRecorder{}
	srv := catalogStub(t, rec,
		map[string]int64{"777": 4242},
		map[int64]string{4242: liveRow(4242, 777, "Kun")},
	)
	c := New(srv.URL, "nm_test_key", "")
	ctx := context.Background()

	if _, err := c.GetBatch(ctx, []int{777}); err != nil {
		t.Fatalf("first GetBatch: %v", err)
	}
	// GetBatch does not use the brief cache, so a second call re-fetches the
	// ROW — but the identity lookup must be served from the memo.
	if _, err := c.GetBatch(ctx, []int{777}); err != nil {
		t.Fatalf("second GetBatch: %v", err)
	}
	lookups := 0
	for i := range rec.count() {
		if rec.pathAt(i) == "/v1/catalog/lookup/batch" {
			lookups++
		}
	}
	if lookups != 1 {
		t.Errorf("made %d lookup calls, want 1 (the gid→catalog id memo is what keeps the second hop cheap)", lookups)
	}
}

// TestCatalogFace_PathsAndCredentials pins that every re-anchored lane talks to
// the catalog face under /v1 with the service key, and that the two SURVIVING
// wiki faces keep their own bases — the /internal ownership op and the /api
// staff read-back are not part of the deprecated face and must not move.
func TestCatalogFace_PathsAndCredentials(t *testing.T) {
	rec := &faceRecorder{}
	srv := rec.server(t)
	c := New(srv.URL, "nm_test_key", "")
	ctx := context.Background()

	t.Run("taxonomy list → /v1/catalog + key", func(t *testing.T) {
		if _, err := c.CatalogTaxonomyList(ctx, "tags", nil); err != nil {
			t.Fatalf("CatalogTaxonomyList: %v", err)
		}
		if rec.path != "/v1/catalog/tags" {
			t.Errorf("path = %q, want /v1/catalog/tags", rec.path)
		}
		if rec.apiKey != "nm_test_key" {
			t.Errorf("X-API-Key = %q, want nm_test_key", rec.apiKey)
		}
	})

	t.Run("entity search → /v1/catalog/search", func(t *testing.T) {
		if _, err := c.CatalogEntitySearch(ctx, "labels", "kun", 10, true); err != nil {
			t.Fatalf("CatalogEntitySearch: %v", err)
		}
		if rec.path != "/v1/catalog/search" {
			t.Errorf("path = %q, want /v1/catalog/search", rec.path)
		}
	})

	t.Run("works search → /v1/catalog/works/search", func(t *testing.T) {
		if _, err := c.CatalogWorksSearch(ctx, url.Values{"q": {"kun"}}); err != nil {
			t.Fatalf("CatalogWorksSearch: %v", err)
		}
		if rec.path != "/v1/catalog/works/search" {
			t.Errorf("path = %q, want /v1/catalog/works/search", rec.path)
		}
	})

	t.Run("calendar buckets → /v1/catalog/calendar*", func(t *testing.T) {
		for bucket, want := range map[string]string{
			"":         "/v1/catalog/calendar",
			"/pending": "/v1/catalog/calendar/pending",
			"/tba":     "/v1/catalog/calendar/tba",
		} {
			if _, err := c.CatalogCalendar(ctx, bucket, nil); err != nil {
				t.Fatalf("CatalogCalendar(%q): %v", bucket, err)
			}
			if rec.path != want {
				t.Errorf("bucket %q → path %q, want %q", bucket, rec.path, want)
			}
		}
	})

	t.Run("ownership meta stays on the surviving /internal face", func(t *testing.T) {
		if _, err := c.GalgameMeta(ctx, []int{1, 2}); err != nil {
			t.Fatalf("GalgameMeta: %v", err)
		}
		if rec.path != "/internal/galgame/meta" {
			t.Errorf("path = %q, want /internal/galgame/meta", rec.path)
		}
		if rec.apiKey != "nm_test_key" {
			t.Errorf("X-API-Key = %q, want nm_test_key on the internal face", rec.apiKey)
		}
	})

	t.Run("taxonomy picker → surviving /internal face, dual credential", func(t *testing.T) {
		// A2-1g: the picker's door is the CONTRIBUTOR one. Pointing it at the
		// staff /api door instead is not a cosmetic difference — that door is
		// gated on taxonomy.edit_any, so an ordinary contributor filling in the
		// submission form would get a 403 from the only lane that can turn a
		// tag name into the wiki id their write payload must carry.
		if _, err := c.TaxonomyPickerSearch(ctx, "tag", "恋爱", "user-jwt"); err != nil {
			t.Fatalf("TaxonomyPickerSearch: %v", err)
		}
		if rec.path != "/internal/galgame/taxonomy/tag/search" {
			t.Errorf("path = %q, want /internal/galgame/taxonomy/tag/search", rec.path)
		}
		// Both credentials: the service key admits the call to the internal
		// face, the Bearer is what the signed-in gate reads.
		if rec.apiKey != "nm_test_key" {
			t.Errorf("X-API-Key = %q, want nm_test_key", rec.apiKey)
		}
		if rec.auth != "Bearer user-jwt" {
			t.Errorf("Authorization = %q, want the caller's Bearer", rec.auth)
		}
	})

	t.Run("staff read-back stays on the surviving /api face, no key", func(t *testing.T) {
		if _, err := c.StaffTaxonomyDetail(ctx, "official", "12", "staff-jwt"); err != nil {
			t.Fatalf("StaffTaxonomyDetail: %v", err)
		}
		if rec.path != "/api/official/12" {
			t.Errorf("path = %q, want /api/official/12", rec.path)
		}
		if rec.apiKey != "" {
			t.Errorf("X-API-Key = %q, want empty on the legacy staff face", rec.apiKey)
		}
		if rec.auth != "Bearer staff-jwt" {
			t.Errorf("Authorization = %q, want the staff Bearer", rec.auth)
		}
	})
}

// TestCatalogMemberGIDs_PublishedMembersOnly pins the entity-page member walk's
// outgoing query, one pin per taxonomy family. The regression it guards is
// invisible from this side — the 词条 page's list quietly WIDENS to include
// entries the wiki claimed but never published (state=draft), which then render
// as 未发布 cards on a public browse page (doc 106 §37, extended to the entity
// pages by R4). `claimed=true` alone does not exclude them.
func TestCatalogMemberGIDs_PublishedMembersOnly(t *testing.T) {
	for family, filter := range map[string]string{
		"tag":    "tag_id",
		"label":  "label_id",
		"engine": "engine_id",
	} {
		t.Run(family, func(t *testing.T) {
			rec := &catalogRecorder{}
			srv := catalogStub(t, rec, map[string]int64{}, map[int64]string{})
			c := New(srv.URL, "nm_test_key", "")

			if _, err := c.CatalogMemberGIDs(context.Background(),
				url.Values{filter: {"5"}}, true, 200); err != nil {
				t.Fatalf("CatalogMemberGIDs: %v", err)
			}
			if p := rec.pathAt(0); p != "/v1/catalog/works" {
				t.Fatalf("path = %q, want /v1/catalog/works", p)
			}
			q := rec.queryAt(0)
			if got := q.Get("claim_state"); got != "live" {
				t.Errorf("claim_state = %q, want live — without it the %s page lists unpublished works", got, family)
			}
			// Still bounded to addressable rows: until the supplying wave ships,
			// `claimed` is the only gate the face actually understands.
			if got := q.Get("claimed"); got != "true" {
				t.Errorf("claimed = %q, want true", got)
			}
			if got := q.Get(filter); got != "5" {
				t.Errorf("%s = %q, want 5 — an unscoped walk lists the whole registry", filter, got)
			}
			if q.Get("nsfw") != "" {
				t.Error("an SFW caller must not open the nsfw gate")
			}
		})
	}
}

// TestProductLocaleProjection pins the D7 language table — the mapping a card's
// `original_language` label rides on.
func TestProductLocaleProjection(t *testing.T) {
	cases := map[string]string{
		"ja": "ja-jp", "ja-JP": "ja-jp",
		"zh": "zh-cn", "zh-Hans": "zh-cn",
		"zh-Hant": "zh-tw", "zh-TW": "zh-tw", "zh-HK": "zh-tw",
		"en": "en-us", "en-GB": "en-us",
		// Outside the four product locales the tag passes through verbatim:
		// showing "ko" is honest, showing "" is a silent loss.
		"ko": "ko", "": "",
	}
	for in, want := range cases {
		if got := productLocale(in); got != want {
			t.Errorf("productLocale(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestReleasePrecisionFromPartialISO pins D7 table ②: the catalog's date is
// partial ISO and its LENGTH is the precision. Reading it with a date parser
// would turn "2021" into January 1st, which is exactly the failure the table
// exists to prevent.
func TestReleasePrecisionFromPartialISO(t *testing.T) {
	day, month, year := "2021-06-04", "2021-06", "2021"
	for date, want := range map[*string]string{
		&day: "day", &month: "month", &year: "year", nil: "tba",
	} {
		if got := releasePrecisionOf(date); got != want {
			t.Errorf("releasePrecisionOf(%v) = %q, want %q", date, got, want)
		}
	}
}
