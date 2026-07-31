package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

// Two identity keys move during the W1 window — the source-12 key that anchors
// gid lookups (galgame_wiki -> curated) and the claim site (galgame_wiki ->
// kungal). Both are read here, both are written elsewhere, and both fail
// SILENTLY when reader and data disagree: an unmatched source resolves no gid
// at all, and an unmatched site yields gid 0, which strips a card's link and
// its local stats without raising anything.
//
// So the reader accepts both spellings rather than being flipped in lockstep
// with the data. These tests pin that tolerance in both directions; when the
// legacy halves are eventually removed, they are what should fail first.

func TestClaimSiteAcceptedOnBothSpellings(t *testing.T) {
	cases := []struct {
		site string
		want int
	}{
		{"kungal", 4321},
		{"galgame_wiki", 4321},
		{"moyu", 0},
		{"", 0},
	}
	for _, tc := range cases {
		t.Run(tc.site, func(t *testing.T) {
			it := &CatalogWorkListItem{ClaimedBy: &catClaimedBy{Site: tc.site, WorkID: 4321}}
			if got := it.gid(); got != tc.want {
				t.Errorf("gid() on site %q = %d, want %d", tc.site, got, tc.want)
			}
		})
	}
	if (&CatalogWorkListItem{}).gid() != 0 {
		t.Error("an unclaimed row has no gid")
	}
}

// The batch lookup must ask for every source key in flight, so it resolves on
// either side of the rename without a coordinated deploy.
func TestAnchorLookupAsksForEverySourceKey(t *testing.T) {
	// Only the LEGACY key answers, i.e. the state before the infra rename.
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Items []struct {
				Source     string `json:"source"`
				ExternalID string `json:"external_id"`
			} `json:"items"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)
		out := make([]string, 0, len(body.Items))
		for _, it := range body.Items {
			asked = append(asked, it.Source)
			work := "null"
			if it.Source == "galgame_wiki" && it.ExternalID == "7" {
				work = `{"id":9001}`
			}
			out = append(out, `{"external_id":"`+it.ExternalID+`","work":`+work+`}`)
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[` +
			strings.Join(out, ",") + `]}}`))
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "nm_test_key", "")
	ids, appErr := c.catalogIDsForGIDs(t.Context(), []int{7})
	if appErr != nil {
		t.Fatalf("catalogIDsForGIDs: %v", appErr)
	}
	if ids[7] != 9001 {
		t.Errorf("gid 7 resolved to %d, want 9001 via the pre-rename source key", ids[7])
	}
	for _, key := range anchorSourceKeys {
		if !slices.Contains(asked, key) {
			t.Errorf("source key %q was never asked for; the lookup only resolves "+
				"on the side of the rename it happens to name", key)
		}
	}
}

// And the post-rename side resolves too, from a cold cache.
func TestAnchorLookupResolvesAfterTheRename(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Items []struct {
				Source     string `json:"source"`
				ExternalID string `json:"external_id"`
			} `json:"items"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)
		out := make([]string, 0, len(body.Items))
		for _, it := range body.Items {
			work := "null"
			if it.Source == "curated" && it.ExternalID == "8" {
				work = `{"id":9002}`
			}
			out = append(out, `{"external_id":"`+it.ExternalID+`","work":`+work+`}`)
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[` +
			strings.Join(out, ",") + `]}}`))
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "nm_test_key", "")
	ids, appErr := c.catalogIDsForGIDs(t.Context(), []int{8})
	if appErr != nil {
		t.Fatalf("catalogIDsForGIDs: %v", appErr)
	}
	if ids[8] != 9002 {
		t.Errorf("gid 8 resolved to %d, want 9002 via the post-rename source key", ids[8])
	}
}
