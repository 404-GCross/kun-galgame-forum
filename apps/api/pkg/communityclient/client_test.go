package communityclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kun-galgame-api/pkg/communityclient"
)

func newTestClient(baseURL string) *communityclient.Client {
	return communityclient.New(communityclient.Config{BaseURL: baseURL, ClientID: "cid", ClientSecret: "sec"})
}

// TestNotConfigured proves an unconfigured client degrades: Configured() is
// false and every call short-circuits to ErrNotConfigured with no network hit.
func TestNotConfigured(t *testing.T) {
	c := communityclient.New(communityclient.Config{BaseURL: "http://x"}) // no creds
	if c.Configured() {
		t.Fatal("Configured() true without creds")
	}
	if _, err := c.ResolveComments(context.Background(), communityclient.ResolveCommentsRequest{}); !errors.Is(err, communityclient.ErrNotConfigured) {
		t.Errorf("err = %v, want ErrNotConfigured", err)
	}
}

// TestResolveEnvelopeAndAuth checks the Basic auth header, the request path, and
// that the {code,message,data} envelope decodes into the typed result.
func TestResolveEnvelopeAndAuth(t *testing.T) {
	var gotAuth, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "message": "成功",
			"data": map[string]any{
				"thread": map[string]any{"id": 7, "site": "kungal", "kind": 1, "anchor_kind": 1, "anchor_id": "42", "content_rating": 0, "status": 0, "posts_count": 2, "participants_count": 1, "highest_post_number": 2, "created_by": 1, "created_at": "2026-07-13T00:00:00Z"},
				"posts": []any{
					map[string]any{"id": 100, "thread_id": 7, "post_number": 1, "author_id": 1, "content_raw": "hi", "content_html": "<p>hi</p>", "content_rating": 0, "status": 0, "created_at": "2026-07-13T00:00:00Z"},
				},
				"next_cursor": "",
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).ResolveComments(context.Background(), communityclient.ResolveCommentsRequest{
		AnchorKind: communityclient.AnchorSiteGame, AnchorID: "42", ContentRating: communityclient.RatingAll,
	})
	if err != nil {
		t.Fatalf("ResolveComments: %v", err)
	}
	if gotAuth != "Basic Y2lkOnNlYw==" { // base64("cid:sec")
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotPath != "/comments/resolve" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"anchor_kind":1`) {
		t.Errorf("body %q missing anchor_kind", gotBody)
	}
	if out.Thread.ID != 7 || len(out.Posts) != 1 || out.Posts[0].ID != 100 {
		t.Errorf("decoded = %+v", out)
	}
}

// TestErrorMapping proves 403→ErrForbidden, 429→ErrRateLimited, and a non-zero
// business code (HTTP 200) surfaces as *APIError.
func TestErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   map[string]any
		want   error // sentinel, or nil to expect an *APIError
	}{
		{"forbidden site binding", http.StatusForbidden, nil, communityclient.ErrForbidden},
		{"tl0 rate limit", http.StatusTooManyRequests, nil, communityclient.ErrRateLimited},
		{"business code", http.StatusOK, map[string]any{"code": 40001, "message": "bad"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				if tc.body != nil {
					_ = json.NewEncoder(w).Encode(tc.body)
				}
			}))
			defer srv.Close()

			_, err := newTestClient(srv.URL).ListPosts(context.Background(), 7, "", "")
			if tc.want != nil {
				if !errors.Is(err, tc.want) {
					t.Errorf("err = %v, want %v", err, tc.want)
				}
				return
			}
			var apiErr *communityclient.APIError
			if !errors.As(err, &apiErr) || apiErr.Code != 40001 {
				t.Errorf("err = %v, want *APIError code=40001", err)
			}
		})
	}
}

// TestToggleReactionResult proves the reaction result (added + the post's
// author/anchor context) decodes so the BFF can drive the like triple-write.
func TestToggleReactionResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/posts/100/reaction" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"added": true, "author_id": 55, "thread_id": 7, "anchor_kind": 1, "anchor_id": "42"},
		})
	}))
	defer srv.Close()

	res, err := newTestClient(srv.URL).ToggleReaction(context.Background(), 100, communityclient.ReactionToggleRequest{UserID: 9, Kind: communityclient.ReactionLike})
	if err != nil {
		t.Fatalf("ToggleReaction: %v", err)
	}
	if !res.Added || res.AuthorID != 55 || res.AnchorID != "42" {
		t.Errorf("result = %+v", res)
	}
}

// TestListPostsQuery proves after/limit ride as query params and empty values
// are omitted (community defaults then apply).
func TestListPostsQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"posts": []any{}}})
	}))
	defer srv.Close()

	if _, err := newTestClient(srv.URL).ListPosts(context.Background(), 7, "50", "30"); err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	for _, want := range []string{"after=50", "limit=30"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
}
