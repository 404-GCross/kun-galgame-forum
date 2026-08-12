package service

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
	if got := rec.get("claim_state"); got != seriesUnpublishedStates {
		t.Errorf("claim_state = %q, want %q", got, seriesUnpublishedStates)
	}
	if got := rec.get("nsfw"); got != "1" {
		t.Errorf("nsfw = %q, want 1 — the age gate is never a population cut", got)
	}
	if got := rec.get("content_limit"); got != "" {
		t.Errorf("content_limit = %q, want it absent for the NSFW reader", got)
	}
}

func TestSeriesUnpublished_SFWReaderGetsTheGatedBucket(t *testing.T) {
	rec := &unpublishedRecorder{}
	svc := rec.service(t)

	svc.unpublishedMembers(context.Background(), "594", true)
	if got := rec.get("content_limit"); got != "sfw" {
		t.Errorf("content_limit = %q, want sfw for the SFW reader", got)
	}
}

func TestSeriesUnpublished_NilEnricherMeansEmptyBucket(t *testing.T) {
	svc := NewSeriesService(client.New("http://127.0.0.1:1", "nm_test_key", ""), nil, nil)
	if cards := svc.unpublishedMembers(context.Background(), "1", false); len(cards) != 0 {
		t.Errorf("cards = %d, want 0 with no enricher", len(cards))
	}
}
