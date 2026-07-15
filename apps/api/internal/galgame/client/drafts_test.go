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
// content_limit, and pass the wiki's {items, total} data through verbatim.
func TestDraftsRequestAndPassthrough(t *testing.T) {
	cases := []struct {
		name             string
		isSFW            bool
		wantContentLimit string
	}{
		{name: "sfw drops nsfw", isSFW: true, wantContentLimit: "sfw"},
		{name: "nsfw includes all", isSFW: false, wantContentLimit: "all"},
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
			// bytes reach us untouched.
			c := NewGalgameClient(srv.URL, "")
			data, appErr := c.Drafts(context.Background(), 2, 24, tc.isSFW)
			if appErr != nil {
				t.Fatalf("Drafts returned error: %v", appErr)
			}

			if gotPath != "/galgame/drafts" {
				t.Errorf("path = %q, want /galgame/drafts", gotPath)
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
