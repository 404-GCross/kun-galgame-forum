package service

// The best-cover facet was the last read where the forum could ask the catalog
// "has THIS person voted" about a uid it picked itself (wave 176's ?uid=).
// Wave 180 replaced that with the viewer's own token — but only for viewers who
// have one: the detail page is optionally authed and the tallies are public, so
// the S2S read survives for logged-out visitors, with nobody named.
//
// Both halves are pinned here because either one degrades silently. A logged-in
// viewer served by the anonymous path sees vote counts with their own ballot
// missing (the star never lights up, and the bug reads as "voting is broken"),
// while an anonymous viewer routed through the user face would lose the counts
// entirely on a 401.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/catalogclient"
)

type coverPlaneRecorder struct {
	mu    sync.Mutex
	path  string
	query string
	auth  string
	// staleToken makes the user face answer the pre-scope 403 (code 233),
	// the way the catalog denies a session minted before catalog:edit existed.
	staleToken bool
}

// server answers the gid→work bridge (gid 1 = work 1000) and both cover faces.
func (r *coverPlaneRecorder) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		raw, _ := io.ReadAll(req.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(req.URL.Path, "/catalog/lookup/batch") {
			var in struct {
				Items []struct {
					ExternalID string `json:"external_id"`
				} `json:"items"`
			}
			_ = json.Unmarshal(raw, &in)
			out := make([]string, 0, len(in.Items))
			for _, it := range in.Items {
				out = append(out, `{"external_id":"`+it.ExternalID+`","work":{"id":1000}}`)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[` +
				strings.Join(out, ",") + `]}}`))
			return
		}

		r.mu.Lock()
		r.path, r.query, r.auth = req.URL.Path, req.URL.RawQuery, req.Header.Get("Authorization")
		stale := r.staleToken
		r.mu.Unlock()

		if stale && strings.Contains(req.URL.Path, "/user/catalog/") {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"code":233,"message":"missing required scope: catalog:edit"}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"covers":[` +
			`{"id":88,"image_hash":"abc","vote_count":3,"voted":true}]}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (r *coverPlaneRecorder) service(t *testing.T) *GalgameService {
	srv := r.server(t)
	return &GalgameService{
		galgameClient: client.New(srv.URL, "nm_test_key", ""),
		catalog:       catalogclient.New(catalogclient.Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"}),
	}
}

func TestCoverVotesReadAsTheViewerWhenTheyHaveAToken(t *testing.T) {
	rec := &coverPlaneRecorder{}
	covers := []dto.GalgameCover{{ImageHash: "abc"}}

	rec.service(t).hydrateCoverVotes(t.Context(), 1, "user-jwt", covers)

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.path != "/api/v1/user/catalog/works/1000/covers" {
		t.Errorf("covers read hit %q, want the user plane's covers face", rec.path)
	}
	if rec.auth != "Bearer user-jwt" {
		t.Errorf("auth = %q, want the viewer's own bearer", rec.auth)
	}
	// A uid here would mean the ballot was still being asked for by name.
	if strings.Contains(rec.query, "uid=") {
		t.Errorf("the viewer is the token, not a query parameter: %q", rec.query)
	}
	if covers[0].ID != 88 || covers[0].VoteCount != 3 || !covers[0].Voted {
		t.Errorf("hydration lost the tally: %+v", covers[0])
	}
}

func TestCoverVotesFallBackToPublicCountsOnAStaleToken(t *testing.T) {
	rec := &coverPlaneRecorder{staleToken: true}
	covers := []dto.GalgameCover{{ImageHash: "abc"}}

	rec.service(t).hydrateCoverVotes(t.Context(), 1, "pre-scope-jwt", covers)

	rec.mu.Lock()
	defer rec.mu.Unlock()
	// The scope denial is the one failure the viewer can't do anything about
	// mid-session, so it must not cost them the public half of the facet: the
	// read retries on the S2S face and the counts still render (ballot unlit).
	if rec.path != "/api/v1/catalog/works/1000" {
		t.Errorf("stale-token covers read landed on %q, want the S2S fallback", rec.path)
	}
	if !strings.HasPrefix(rec.auth, "Basic ") {
		t.Errorf("auth = %q, want the S2S credential after the fallback", rec.auth)
	}
	if covers[0].ID != 88 || covers[0].VoteCount != 3 {
		t.Errorf("fallback lost the tally: %+v", covers[0])
	}
}

func TestCoverVotesStayAnonymousWithoutAToken(t *testing.T) {
	rec := &coverPlaneRecorder{}
	covers := []dto.GalgameCover{{ImageHash: "abc"}}

	rec.service(t).hydrateCoverVotes(t.Context(), 1, "", covers)

	rec.mu.Lock()
	defer rec.mu.Unlock()
	// The public tallies are worth rendering for a logged-out visitor, so this
	// half deliberately did NOT move: there is no token, and inventing a uid is
	// the thing wave 180 removed.
	if rec.path != "/api/v1/catalog/works/1000" {
		t.Errorf("anonymous covers read hit %q, want the S2S face", rec.path)
	}
	if !strings.HasPrefix(rec.auth, "Basic ") {
		t.Errorf("auth = %q, want the S2S credential", rec.auth)
	}
	if rec.query != "" {
		t.Errorf("an anonymous read names nobody: %q", rec.query)
	}
	if covers[0].ID != 88 || covers[0].VoteCount != 3 {
		t.Errorf("hydration lost the tally: %+v", covers[0])
	}
}
