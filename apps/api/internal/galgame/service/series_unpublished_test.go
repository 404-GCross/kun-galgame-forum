package service

// The series page's unpublished bucket rides one upstream query, and that
// query is load-bearing the same way the drafts funnel's is: widen it to
// `hidden` and the page relists entries the site took DOWN; narrow it to
// `none` and a draft-claimed member vanishes from both halves of the page,
// which is the "one-work series" reading the bucket exists to kill. So the
// query is pinned here, drafts_query_test.go-style — the upstream answers
// empty, ToCards short-circuits, no fixtures needed.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
)

type unpublishedRecorder struct {
	mu    sync.Mutex
	path  string
	query url.Values
}

func (r *unpublishedRecorder) service(t *testing.T) *SeriesService {
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
	return NewSeriesService(client.New(srv.URL, "nm_test_key", ""), &GalgameEnricher{}, nil)
}

func (r *unpublishedRecorder) get(key string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.query.Get(key)
}

func TestSeriesUnpublished_AsksForTheUnpublishedStatesOnly(t *testing.T) {
	rec := &unpublishedRecorder{}
	svc := rec.service(t)

	cards := svc.unpublishedMembers(context.Background(), "594", false)
	if len(cards) != 0 {
		t.Fatalf("cards = %d, want 0 from an empty upstream", len(cards))
	}
	if rec.path != "/v1/catalog/works/search" {
		t.Errorf("path = %q, want /v1/catalog/works/search", rec.path)
	}
	if got := rec.get("series_id"); got != "594" {
		t.Errorf("series_id = %q, want 594 — without the scope every series lists the global pool", got)
	}
	// draft + pending + none and NOTHING else: `live` would double-list the
	// published half, `hidden` would republish a takedown.
	if got := rec.get("claim_state"); got != seriesUnpublishedStates {
		t.Errorf("claim_state = %q, want %q", got, seriesUnpublishedStates)
	}
	// The age gate is open like every lane; the reader's editorial gate came in
	// as isSFW=false here, and the gate's "all" convention is to send nothing.
	if got := rec.get("nsfw"); got != "1" {
		t.Errorf("nsfw = %q, want 1 — the age gate is never a population cut", got)
	}
	if got := rec.get("content_limit"); got != "" {
		t.Errorf("content_limit = %q, want it absent for the NSFW reader", got)
	}
}

// The SFW reader's bucket is gated on the SAME axis as the page above it —
// content_limit, which every works doc carries (an unclaimed row's value is
// projected off its age rating). An ungated bucket would render the covers the
// list half just hid.
func TestSeriesUnpublished_SFWReaderGetsTheGatedBucket(t *testing.T) {
	rec := &unpublishedRecorder{}
	svc := rec.service(t)

	svc.unpublishedMembers(context.Background(), "594", true)
	if got := rec.get("content_limit"); got != "sfw" {
		t.Errorf("content_limit = %q, want sfw for the SFW reader", got)
	}
}

// No enricher wired (the taxonomy-only tests): the bucket degrades to empty
// instead of dereferencing its way down.
func TestSeriesUnpublished_NilEnricherMeansEmptyBucket(t *testing.T) {
	svc := NewSeriesService(client.New("http://127.0.0.1:1", "nm_test_key", ""), nil, nil)
	if cards := svc.unpublishedMembers(context.Background(), "1", false); len(cards) != 0 {
		t.Errorf("cards = %d, want 0 with no enricher", len(cards))
	}
}
