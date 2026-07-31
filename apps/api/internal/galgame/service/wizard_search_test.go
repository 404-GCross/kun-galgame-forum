package service

// The publish wizard is the site's only defence against duplicate submissions,
// so both halves of its query are pinned here:
//
//   - the ITEMS half must hit the catalog search with claim_state=live,draft.
//     Narrowing that to `live` hides every unpublished entry, which is the
//     shape of the 52k incident: what the wizard cannot see gets submitted
//     again.
//   - the PENDING half must still hit the WIKI face with include_pending=true.
//     The catalog has no per-user read face for the pre-N5 backlog, so
//     re-pointing this half would silently empty "您的待审 / 已拒草稿".

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
)

type wizardRecorder struct {
	mu       sync.Mutex
	catalogQ url.Values
	wikiQ    url.Values
	wikiHits int
}

func (r *wizardRecorder) service(t *testing.T) *SubmissionService {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		body := `{"code":0,"message":"ok","data":{"items":[],"total":0}}`
		switch {
		case strings.HasSuffix(req.URL.Path, "/catalog/works/search"):
			r.catalogQ = req.URL.Query()
			body = `{"code":0,"message":"ok","data":{"total":2,"items":[
			  {"id":11,"display_name":"A","cover":"https://img/aa/bb/hash1.webp",
			   "claimed_by":{"site":"galgame_wiki","work_id":292,"state":"live"},
			   "names":{"ja-jp":"白恋サクラ"},"refs":[{"source":"vndb","external_id":"v22610"}]},
			  {"id":12,"display_name":"B","cover":"",
			   "claimed_by":{"site":"galgame_wiki","work_id":9978,"state":"draft"}},
			  {"id":13,"display_name":"withdrawn","cover":"",
			   "claimed_by":{"site":"galgame_wiki","work_id":404,"state":"hidden"}},
			  {"id":14,"display_name":"unclaimed","cover":"","claimed_by":null}
			]}}`
		case strings.HasSuffix(req.URL.Path, "/galgame/search"):
			r.wikiQ = req.URL.Query()
			r.wikiHits++
			body = `{"code":0,"message":"ok","data":{"items":[{"id":1}],"total":1,
			  "pending":[{"id":64689,"status":3,"name_ja_jp":"曇った瞳に恋してる"}]}}`
		}
		r.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewSubmissionService(client.New(srv.URL, "nm_test_key", ""), nil)
}

func wizardSearch(t *testing.T, svc *SubmissionService) *WizardSearchPage {
	t.Helper()
	page, appErr := svc.SearchWithPending(context.Background(), "tok",
		url.Values{"q": {"sakura"}, "limit": {"12"}})
	if appErr != nil {
		t.Fatalf("SearchWithPending: %v", appErr)
	}
	return page
}

func TestWizard_ItemsComeFromTheCatalogSearch(t *testing.T) {
	rec := &wizardRecorder{}
	page := wizardSearch(t, rec.service(t))

	if got := rec.catalogQ.Get("claim_state"); got != "live,draft" {
		t.Errorf("claim_state = %q, want live,draft — `live` alone hides every unpublished entry", got)
	}
	if got := rec.catalogQ.Get("claimed"); got != "true" {
		t.Errorf("claimed = %q, want true — an unclaimed work has no gid to act on", got)
	}
	if got := rec.catalogQ.Get("q"); got != "sakura" {
		t.Errorf("q = %q, want sakura", got)
	}
	if got := rec.catalogQ.Get("limit"); got != "12" {
		t.Errorf("limit = %q, want 12", got)
	}
	// The age gate is open and the EDITORIAL gate is absent: the wizard is a
	// dedup tool for a submitter, not a browse lane.
	if got := rec.catalogQ.Get("nsfw"); got != "1" {
		t.Errorf("nsfw = %q, want 1", got)
	}
	if got := rec.catalogQ.Get("content_limit"); got != "" {
		t.Errorf("content_limit = %q, want it absent on the wizard lane", got)
	}
	if !strings.Contains(rec.catalogQ.Get("include"), "refs") {
		t.Errorf("include = %q, want refs (the row prints the VNDB id)", rec.catalogQ.Get("include"))
	}
	if page.Total != 2 {
		t.Errorf("total = %d, want the catalog total 2", page.Total)
	}
}

func TestWizard_ItemsAreKeyedByGIDAndDropWithdrawnRows(t *testing.T) {
	rec := &wizardRecorder{}
	page := wizardSearch(t, rec.service(t))

	if len(page.Items) != 2 {
		t.Fatalf("items = %d, want 2 (hidden claim and unclaimed row are not actionable)", len(page.Items))
	}
	// Never the catalog id: the two key spaces overlap and every wizard action
	// (claim POST, /galgame/:gid link, draft link) is keyed by gid.
	if page.Items[0].ID != 292 || page.Items[1].ID != 9978 {
		t.Errorf("ids = %d,%d, want the gids 292,9978", page.Items[0].ID, page.Items[1].ID)
	}
	if page.Items[0].Status != 0 || page.Items[1].Status != 2 {
		t.Errorf("statuses = %d,%d, want live→0 and draft→2",
			page.Items[0].Status, page.Items[1].Status)
	}
	if page.Items[0].VndbID != "v22610" {
		t.Errorf("vndb_id = %q, want v22610", page.Items[0].VndbID)
	}
	// The card reads `banner`; the catalog delivers the art as the derived
	// effective banner, so the field must be filled from it or every row on the
	// wizard loses its cover.
	if page.Items[0].Banner == "" || page.Items[0].Banner != page.Items[0].EffectiveBannerURL {
		t.Errorf("banner = %q, want it mirrored from effective_banner_url %q",
			page.Items[0].Banner, page.Items[0].EffectiveBannerURL)
	}
}

func TestWizard_PendingStaysOnTheWikiFace(t *testing.T) {
	rec := &wizardRecorder{}
	page := wizardSearch(t, rec.service(t))

	if rec.wikiHits != 1 {
		t.Fatalf("wiki face hits = %d, want exactly 1 — the pending half has no catalog counterpart", rec.wikiHits)
	}
	if got := rec.wikiQ.Get("include_pending"); got != "true" {
		t.Errorf("include_pending = %q, want true", got)
	}
	if got := rec.wikiQ.Get("status"); got != "0,2" {
		t.Errorf("status = %q, want the pre-switchover 0,2 verbatim", got)
	}
	var pending []struct {
		ID     int `json:"id"`
		Status int `json:"status"`
	}
	if err := json.Unmarshal(page.Pending, &pending); err != nil {
		t.Fatalf("pending is not an array: %v", err)
	}
	if len(pending) != 1 || pending[0].ID != 64689 || pending[0].Status != 3 {
		t.Errorf("pending = %+v, want the caller's own status-3 row forwarded verbatim", pending)
	}
}

func TestWizard_PendingIsAnEmptyArrayWhenTheFaceOmitsIt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[],"total":0}}`))
	}))
	t.Cleanup(srv.Close)
	svc := NewSubmissionService(client.New(srv.URL, "nm_test_key", ""), nil)

	page := wizardSearch(t, svc)
	// `null` would make the FE's `pending.length` read throw on some paths; an
	// empty array is the shape the component already handles.
	if string(page.Pending) != "[]" {
		t.Errorf("pending = %s, want []", page.Pending)
	}
}
