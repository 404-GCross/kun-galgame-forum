package service

// GetByMultiTag builds an upstream query, and every field in it is load-bearing
// in a way that is invisible from this side: a dropped param does not fail, it
// silently returns plausible-but-wrong results. page/limit went unforwarded for
// a long time and the only symptom was that page 2 quietly re-served page 1, and
// `expand` is what separates 包含模式 from 精确模式 — drop it and contains mode
// degrades into exact mode with no error anywhere. So the query is pinned here.
//
// The upstream is stubbed with an empty item list on purpose: FilterSFW and
// ToCards both short-circuit before touching the DB or the user service, so
// these tests need no fixtures and assert exactly one thing — what we asked the
// galgame for.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
)

// queryRecorder captures the query the galgame was called with.
type queryRecorder struct {
	mu    sync.Mutex
	path  string
	query url.Values
}

func (r *queryRecorder) service(t *testing.T) *TagService {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.path = req.URL.Path
		r.query = req.URL.Query()
		r.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[],"total":0}}`))
	}))
	t.Cleanup(srv.Close)
	return NewTagService(client.New(srv.URL, "nm_test_key", ""), &GalgameEnricher{}, nil)
}

func (r *queryRecorder) get(key string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.query.Get(key)
}

func (r *queryRecorder) has(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.query[key]
	return ok
}

func TestGetByMultiTag_ForwardsPagination(t *testing.T) {
	rec := &queryRecorder{}
	svc := rec.service(t)

	_, appErr := svc.GetByMultiTag(context.Background(), url.Values{
		"tagIds": {"638,41"},
		"page":   {"3"},
		"limit":  {"24"},
	}, false)
	if appErr != nil {
		t.Fatalf("GetByMultiTag: %v", appErr)
	}

	if got := rec.get("ids"); got != "638,41" {
		t.Errorf("ids = %q, want %q (FE sends tagIds; upstream takes ids)", got, "638,41")
	}
	// The regression this pins: without these the galgame answers with its own
	// default first page for every request, so the pager moves and nothing changes.
	if got := rec.get("page"); got != "3" {
		t.Errorf("page = %q, want %q — pagination must reach the galgame", got, "3")
	}
	if got := rec.get("limit"); got != "24" {
		t.Errorf("limit = %q, want %q", got, "24")
	}
	if got := rec.get("include"); got != "meta" {
		t.Errorf("include = %q, want %q (thin items need content_limit + user_id)", got, "meta")
	}
}

func TestGetByMultiTag_MatchMode(t *testing.T) {
	cases := []struct {
		name       string
		mode       string
		wantExpand string
	}{
		// contains is the ONLY value that expands; everything else must stay a
		// flat AND rather than guessing at the user's intent.
		{name: "contains expands to descendants", mode: "contains", wantExpand: "descendants"},
		{name: "exact stays flat", mode: "exact", wantExpand: ""},
		{name: "absent stays flat", mode: "", wantExpand: ""},
		{name: "unknown value stays flat", mode: "fuzzy", wantExpand: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &queryRecorder{}
			svc := rec.service(t)

			q := url.Values{"tagIds": {"41"}}
			if tc.mode != "" {
				q.Set("mode", tc.mode)
			}
			if _, appErr := svc.GetByMultiTag(context.Background(), q, false); appErr != nil {
				t.Fatalf("GetByMultiTag: %v", appErr)
			}

			if got := rec.get("expand"); got != tc.wantExpand {
				t.Errorf("expand = %q, want %q", got, tc.wantExpand)
			}
			if tc.wantExpand == "" && rec.has("expand") {
				t.Error("expand must be absent, not empty — an empty value is still a value upstream")
			}
			// `mode` is the forum's own vocabulary; the galgame never sees it.
			if rec.has("mode") {
				t.Error("mode leaked upstream; it should be translated to expand")
			}
		})
	}
}
