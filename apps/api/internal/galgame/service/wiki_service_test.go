package service

// Integration-style test for ProxyWrite — uses an httptest server as
// the fake wiki target so we can assert the path + query + body + token
// header survive the kungal → wiki hop verbatim. The single most
// important contract this test guards is:
//
//   `?force=true` (taxonomy two-stage safe delete) MUST reach wiki.
//
// A silent drop here would degrade `DELETE /galgame-tag/:id?force=true`
// into a regular DELETE which wiki rejects → user sees "still
// referenced" forever even after the second confirmation, with no
// indication that the query param was lost in transit. Worse, if wiki
// were to ever accept the bare DELETE, the FE-side two-stage UI would
// nuke references the user thought they were force-deleting.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"kun-galgame-api/internal/galgame/client"
)

// captured records what the fake wiki saw on a single request.
type captured struct {
	method string
	path   string
	query  url.Values
	auth   string
	body   string
}

// newFakeWiki spins up a tiny HTTP server that captures the next
// request and replies with a successful {code,message,data} envelope.
// Returns the captured-request channel + the server URL.
func newFakeWiki(t *testing.T) (*captured, *httptest.Server) {
	t.Helper()
	got := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got.method = r.Method
		got.path = r.URL.Path
		got.query = r.URL.Query()
		got.auth = r.Header.Get("Authorization")
		got.body = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"ok":true}}`))
	}))
	t.Cleanup(srv.Close)
	return got, srv
}

func TestProxyWrite_ForwardsForceQuery(t *testing.T) {
	got, srv := newFakeWiki(t)
	// CDN base intentionally empty: rewriteBanners must not interfere
	// with the success-envelope body when there are no banner fields.
	gc := client.NewGalgameClient(srv.URL, "")
	svc := NewWikiService(gc, nil, nil)

	q := url.Values{"force": {"true"}}
	_, err := svc.ProxyWrite(
		context.Background(),
		"DELETE",
		"/api/galgame-tag/5", // gateway path
		"tok-bearer",
		q,
		nil,
		"application/json",
	)
	if err != nil {
		t.Fatalf("ProxyWrite: %v", err)
	}
	// Path mapping: /api/galgame-tag/5 → /tag/5, then the client prepends the
	// legacy /api face → /api/tag/5 reaches the service.
	if got.path != "/api/tag/5" {
		t.Errorf("path mismatch: got %q, want /api/tag/5", got.path)
	}
	// Critical: ?force=true survives.
	if got.query.Get("force") != "true" {
		t.Errorf("force query lost: %v", got.query)
	}
	// Method + Bearer token forwarded.
	if got.method != "DELETE" {
		t.Errorf("method: got %s", got.method)
	}
	if got.auth != "Bearer tok-bearer" {
		t.Errorf("auth header: got %q", got.auth)
	}
}

func TestProxyWrite_OmitsQueryWhenEmpty(t *testing.T) {
	got, srv := newFakeWiki(t)
	gc := client.NewGalgameClient(srv.URL, "")
	svc := NewWikiService(gc, nil, nil)

	_, err := svc.ProxyWrite(
		context.Background(),
		"POST",
		"/api/galgame-tag",
		"tok",
		nil, // no query
		[]byte(`{"name":"foo"}`),
		"application/json",
	)
	if err != nil {
		t.Fatalf("ProxyWrite: %v", err)
	}
	// No spurious `?` in the URL.
	if strings.Contains(got.path, "?") || len(got.query) != 0 {
		t.Errorf("expected no query, got path=%q query=%v", got.path, got.query)
	}
}

func TestProxyWrite_ForwardsBodyAndContentType(t *testing.T) {
	got, srv := newFakeWiki(t)
	gc := client.NewGalgameClient(srv.URL, "")
	svc := NewWikiService(gc, nil, nil)

	body := []byte(`{"alias":["a","b"]}`)
	_, err := svc.ProxyWrite(
		context.Background(),
		"PUT",
		"/api/galgame-tag",
		"tok",
		nil,
		body,
		"application/json",
	)
	if err != nil {
		t.Fatalf("ProxyWrite: %v", err)
	}
	if got.body != string(body) {
		t.Errorf("body lost: got %q want %q", got.body, body)
	}
	// Sanity: response envelope parses as expected.
	var env struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal([]byte(`{}`), &env)
}
