package catalogclient

// Contract tests for the user-token write plane (wave 176). The axes are the
// ones a wrong implementation would get wrong SILENTLY: which credential is
// attached (a Basic header here would defeat the whole posture), which path and
// method are used, and the error mapping the BFF branches on — above all the
// scope denial, which is the difference between telling a user to re-login and
// telling them they are not allowed.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVoteCover_ForwardsBearerAndParsesEnvelope(t *testing.T) {
	var gotAuth, gotPath, gotMethod string
	var gotBodyLen int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath, gotMethod, gotBodyLen = r.Header.Get("Authorization"), r.URL.Path, r.Method, r.ContentLength
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"cover_id":88,"vote_count":13,"voted":true}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	res, err := c.VoteCover(context.Background(), "user-jwt", 1000, 88)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Fatalf("method = %s, want PUT", gotMethod)
	}
	if gotPath != "/api/v1/user/catalog/works/1000/covers/88/vote" {
		t.Fatalf("bad path %q", gotPath)
	}
	// The user plane authenticates AS THE USER: a Basic header here would let
	// the service fall back to the S2S posture.
	if gotAuth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", gotAuth)
	}
	if gotBodyLen > 0 {
		t.Fatalf("vote must send no body, got %d bytes", gotBodyLen)
	}
	if res.CoverID != 88 || res.VoteCount != 13 || !res.Voted {
		t.Fatalf("envelope not parsed: %+v", res)
	}
}

func TestUnvoteCover_UsesDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"cover_id":88,"vote_count":12,"voted":false}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	res, err := c.UnvoteCover(context.Background(), "user-jwt", 1000, 88)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/v1/user/catalog/works/1000/covers/88/vote" {
		t.Fatalf("unvote hit %s %s", gotMethod, gotPath)
	}
	if res.Voted || res.VoteCount != 12 {
		t.Fatalf("withdrawal result wrong: %+v", res)
	}
}

// TestCoverVoteErrorMapping pins the branches the handler acts on. The two 403s
// are the point: a missing scope is the user's own stale grant (fixable by
// re-login), anything else is an operator problem the user cannot act on.
func TestCoverVoteErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"dead token", http.StatusUnauthorized, `{"code":205,"message":"invalid token"}`, ErrUnauthorized},
		{"missing scope", http.StatusForbidden, `{"code":233,"message":"missing required scope: catalog:edit"}`, ErrInsufficientScope},
		{"unknown cover", http.StatusNotFound, `{"code":233,"message":"cover not on work"}`, ErrNotFound},
		{"upstream down", http.StatusBadGateway, `nonsense`, ErrUpstream},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
			_, err := c.VoteCover(context.Background(), "user-jwt", 1000, 88)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

// A 403 that is NOT about scope must stay a generic denial: telling a user to
// re-login when the CLIENT is misconfigured sends them round a loop that cannot
// fix anything.
func TestCoverVoteForbiddenWithoutScopeIsGeneric(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":233,"message":"client is not bound to a catalog site"}`))
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	_, err := c.VoteCover(context.Background(), "user-jwt", 1000, 88)
	if errors.Is(err, ErrInsufficientScope) {
		t.Fatal("a site-binding 403 must not read as a scope problem")
	}
	var apiErr *UserAPIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden {
		t.Fatalf("err = %v, want a *UserAPIError 403", err)
	}
}

// No token, no call: an empty access token is the caller's own dead session,
// not something to ask the catalog about.
func TestCoverVoteWithoutTokenNeverCalls(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	if _, err := c.VoteCover(context.Background(), "", 1000, 88); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if called {
		t.Fatal("a tokenless vote must not reach the catalog")
	}
}

func TestCoverVoteUnconfigured(t *testing.T) {
	if _, err := New(Config{}).VoteCover(context.Background(), "user-jwt", 1, 2); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

// The READ half rides the S2S face: Basic, plus the viewer uid as a query
// parameter, and only the covers block is decoded.
func TestWorkCoverVotes(t *testing.T) {
	var gotAuth, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath, gotQuery = r.Header.Get("Authorization"), r.URL.Path, r.URL.RawQuery
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"work":{"id":1000},"covers":[` +
			`{"id":88,"image_hash":"aa","vote_count":3,"voted":true},` +
			`{"id":89,"image_hash":"bb","vote_count":0}]}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	tallies, err := c.WorkCoverVotes(context.Background(), 1000, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/api/v1/catalog/works/1000" || gotQuery != "uid=7" {
		t.Fatalf("read hit %s?%s", gotPath, gotQuery)
	}
	if len(gotAuth) < 6 || gotAuth[:6] != "Basic " {
		t.Fatalf("the tally read is S2S; auth = %q", gotAuth)
	}
	if len(tallies) != 2 || tallies[0].ID != 88 || tallies[0].ImageHash != "aa" ||
		tallies[0].VoteCount != 3 || !tallies[0].Voted || tallies[1].Voted {
		t.Fatalf("tallies decoded wrong: %+v", tallies)
	}
}

// An anonymous read still wants the counts, so it must not invent a uid=0
// viewer — upstream reads 0 as "nobody asking", and sending it is noise.
func TestWorkCoverVotesAnonymousSendsNoUID(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"covers":[]}}`))
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	if _, err := c.WorkCoverVotes(context.Background(), 1000, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery != "" {
		t.Fatalf("anonymous read sent %q", gotQuery)
	}
}
