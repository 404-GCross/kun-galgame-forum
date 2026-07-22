package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// TestDraftsRequestAndPassthrough pins the /galgame/drafts client method: it
// must hit the right path, forward page/limit, map the SFW flag to
// content_limit, forward any entity scope (official_id / tag_id / engine_id)
// only when non-zero, and pass the wiki's {items, total} data through verbatim.
func TestDraftsRequestAndPassthrough(t *testing.T) {
	cases := []struct {
		name             string
		isSFW            bool
		filters          DraftFilters
		wantContentLimit string
		// wantParams: query params that must be present with the given value.
		wantParams map[string]string
		// absentParams: scope params that must NOT appear (0 = global).
		absentParams []string
	}{
		{
			name:             "sfw global drops nsfw + no scope",
			isSFW:            true,
			wantContentLimit: "sfw",
			absentParams:     []string{"official_id", "tag_id", "engine_id"},
		},
		{
			name:             "nsfw global includes all + no scope",
			isSFW:            false,
			wantContentLimit: "all",
			absentParams:     []string{"official_id", "tag_id", "engine_id"},
		},
		{
			name:             "scoped to official",
			isSFW:            true,
			filters:          DraftFilters{OfficialID: 88},
			wantContentLimit: "sfw",
			wantParams:       map[string]string{"official_id": "88"},
			absentParams:     []string{"tag_id", "engine_id"},
		},
		{
			name:             "scoped to tag",
			isSFW:            false,
			filters:          DraftFilters{TagID: 12},
			wantContentLimit: "all",
			wantParams:       map[string]string{"tag_id": "12"},
			absentParams:     []string{"official_id", "engine_id"},
		},
		{
			name:             "scoped to engine",
			isSFW:            true,
			filters:          DraftFilters{EngineID: 5},
			wantContentLimit: "sfw",
			wantParams:       map[string]string{"engine_id": "5"},
			absentParams:     []string{"official_id", "tag_id"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			var gotQuery url.Values
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotQuery = r.URL.Query()
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":7,"status":2}],"total":42}}`))
			}))
			defer srv.Close()

			// Empty imageCDNBase makes rewriteBanners a no-op, so the {items,total}
			// bytes reach us untouched. Drafts is a read → the internal face +
			// X-API-Key, so the derived path is /internal/galgame/drafts.
			c := New(srv.URL, "nm_test", "")
			data, appErr := c.Drafts(context.Background(), 2, 24, tc.isSFW, tc.filters)
			if appErr != nil {
				t.Fatalf("Drafts returned error: %v", appErr)
			}

			if gotPath != "/internal/galgame/drafts" {
				t.Errorf("path = %q, want /internal/galgame/drafts", gotPath)
			}
			if got := gotQuery.Get("page"); got != "2" {
				t.Errorf("page = %q, want 2", got)
			}
			if got := gotQuery.Get("limit"); got != "24" {
				t.Errorf("limit = %q, want 24", got)
			}
			if got := gotQuery.Get("content_limit"); got != tc.wantContentLimit {
				t.Errorf("content_limit = %q, want %q", got, tc.wantContentLimit)
			}
			for key, want := range tc.wantParams {
				if got := gotQuery.Get(key); got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
			}
			for _, key := range tc.absentParams {
				if gotQuery.Has(key) {
					t.Errorf("%s = %q, want absent (0 = global)", key, gotQuery.Get(key))
				}
			}

			var parsed struct {
				Items []struct {
					ID     int `json:"id"`
					Status int `json:"status"`
				} `json:"items"`
				Total int64 `json:"total"`
			}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("passthrough data not parseable: %v", err)
			}
			if parsed.Total != 42 {
				t.Errorf("total = %d, want 42", parsed.Total)
			}
			if len(parsed.Items) != 1 || parsed.Items[0].ID != 7 || parsed.Items[0].Status != 2 {
				t.Errorf("items = %+v, want one status=2 draft id=7", parsed.Items)
			}
		})
	}
}
