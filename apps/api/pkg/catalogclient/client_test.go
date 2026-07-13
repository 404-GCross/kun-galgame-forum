package catalogclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfigured(t *testing.T) {
	if New(Config{BaseURL: "http://x", ClientID: "", ClientSecret: ""}).Configured() {
		t.Fatal("expected not configured without credentials")
	}
	if New(Config{BaseURL: "", ClientID: "a", ClientSecret: "b"}).Configured() {
		t.Fatal("expected not configured without base URL")
	}
	if !New(Config{BaseURL: "http://x", ClientID: "a", ClientSecret: "b"}).Configured() {
		t.Fatal("expected configured with base URL + credentials")
	}
}

func TestNotConfigured(t *testing.T) {
	if _, err := New(Config{}).WorkCredits(context.Background(), 1); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("want ErrNotConfigured, got %v", err)
	}
}

// TestWorkCredits_ForwardsBasicAndPassesThrough is the core contract: the Basic
// header is attached, the path is right, and the envelope's data field is
// returned byte-for-byte (no reshape).
func TestWorkCredits_ForwardsBasicAndPassesThrough(t *testing.T) {
	var gotAuth, gotPath string
	body := `{"work_id":3853,"groups":[{"role_id":1,"role_key":"scenario","role_name":"剧本","credits":[{"credit_name_id":42,"name":"丸戸史明"}]}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":` + body + `}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	data, err := c.WorkCredits(context.Background(), 3853)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/api/v1/catalog/works/3853/credits" {
		t.Fatalf("bad path %q", gotPath)
	}
	if len(gotAuth) < 6 || gotAuth[:6] != "Basic " {
		t.Fatalf("expected Basic auth, got %q", gotAuth)
	}
	// The returned data must be the exact `data` sub-object, verbatim.
	var want, got any
	_ = json.Unmarshal([]byte(body), &want)
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("returned data is not valid json: %v", err)
	}
	if string(data) != body {
		t.Fatalf("data not passed through verbatim:\n got %s\nwant %s", data, body)
	}
}

// TestReverseLookups_PathAndPagination checks the two reverse-lookup paths and
// that limit/offset are forwarded only when non-zero.
func TestReverseLookups_PathAndPagination(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"items":[]}}`))
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})

	if _, err := c.NameWorks(context.Background(), 45121, 10, 20); err != nil {
		t.Fatalf("NameWorks error: %v", err)
	}
	if gotPath != "/api/v1/catalog/names/45121/works" {
		t.Fatalf("bad name path %q", gotPath)
	}
	if gotQuery != "limit=10&offset=20" {
		t.Fatalf("bad name query %q", gotQuery)
	}

	if _, err := c.CharacterWorks(context.Background(), 1335, 0, 0); err != nil {
		t.Fatalf("CharacterWorks error: %v", err)
	}
	if gotPath != "/api/v1/catalog/characters/1335/works" {
		t.Fatalf("bad character path %q", gotPath)
	}
	if gotQuery != "" {
		t.Fatalf("expected no query when limit/offset are zero, got %q", gotQuery)
	}
}

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   error
	}{
		{http.StatusUnauthorized, `{"code":10001,"message":"bad creds"}`, ErrUnauthorized},
		{http.StatusNotFound, `{"code":4,"message":"not found"}`, ErrNotFound},
		{http.StatusInternalServerError, `{"code":1,"message":"boom"}`, ErrUpstream},
		{http.StatusOK, `{"code":1,"message":"logical failure"}`, ErrUpstream},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		}))
		c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
		_, err := c.WorkCredits(context.Background(), 1)
		if !errors.Is(err, tc.want) {
			t.Fatalf("status %d: want %v, got %v", tc.status, tc.want, err)
		}
		srv.Close()
	}
}

// TestUpstreamUnreachable proves a dead catalog degrades to ErrUpstream (never a
// panic / hang) so the BFF can return a graceful 503 instead of 500-ing the page.
func TestUpstreamUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // now nothing is listening
	c := New(Config{BaseURL: url, ClientID: "cid", ClientSecret: "sec"})
	if _, err := c.WorkCredits(context.Background(), 1); !errors.Is(err, ErrUpstream) {
		t.Fatalf("want ErrUpstream, got %v", err)
	}
}
