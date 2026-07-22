package client

// Face-selection tests: prove the galgame client routes each call to the right
// face by ROUTE membership, not HTTP method — reads, the two cron feeds, and
// (since Phase-2 06a) the user write set (galgame-content mutations under
// /galgame) go to the internal face with X-API-Key, while taxonomy writes
// (/tag /official /engine /series) and /admin/* reads+writes stay on the legacy
// /api staff face. There is no keyless-fallback valve any more.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// faceRecorder captures the last request a fake service received.
type faceRecorder struct {
	mu     sync.Mutex
	path   string
	apiKey string
	auth   string
}

func (r *faceRecorder) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.path = req.URL.Path
		r.apiKey = req.Header.Get("X-API-Key")
		r.auth = req.Header.Get("Authorization")
		r.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestFaceSelection_WithKey proves that, with an internal-tier key configured,
// reads hit {base}/internal + X-API-Key, admin reads stay on {base}/api without
// the key, writes stay on {base}/api, and both cron feeds hit {base}/internal +
// X-API-Key — and that a user JWT rides Authorization alongside X-API-Key.
func TestFaceSelection_WithKey(t *testing.T) {
	rec := &faceRecorder{}
	srv := rec.server(t)
	c := New(srv.URL, "nm_test_key", "")
	ctx := context.Background()

	t.Run("anonymous read → internal + key", func(t *testing.T) {
		if _, err := c.Get(ctx, "/galgame/123", nil); err != nil {
			t.Fatalf("Get: %v", err)
		}
		if rec.path != "/internal/galgame/123" {
			t.Errorf("path = %q, want /internal/galgame/123", rec.path)
		}
		if rec.apiKey != "nm_test_key" {
			t.Errorf("X-API-Key = %q, want nm_test_key", rec.apiKey)
		}
	})

	t.Run("personalized read → internal + key + user JWT (dual credential)", func(t *testing.T) {
		if _, err := c.GetWithToken(ctx, "/galgame/mine", "user-jwt", nil); err != nil {
			t.Fatalf("GetWithToken: %v", err)
		}
		if rec.path != "/internal/galgame/mine" {
			t.Errorf("path = %q, want /internal/galgame/mine", rec.path)
		}
		if rec.apiKey != "nm_test_key" {
			t.Errorf("X-API-Key = %q, want nm_test_key", rec.apiKey)
		}
		if rec.auth != "Bearer user-jwt" {
			t.Errorf("Authorization = %q, want Bearer user-jwt", rec.auth)
		}
	})

	t.Run("admin stats read → legacy, no key", func(t *testing.T) {
		if _, err := c.GetAdminStats(ctx, 1, "admin-jwt"); err != nil {
			t.Fatalf("GetAdminStats: %v", err)
		}
		if rec.path != "/api/admin/stats" {
			t.Errorf("path = %q, want /api/admin/stats", rec.path)
		}
		if rec.apiKey != "" {
			t.Errorf("X-API-Key = %q, want empty on legacy admin face", rec.apiKey)
		}
	})

	t.Run("admin messages read → legacy, no key", func(t *testing.T) {
		if _, err := c.GetWithToken(ctx, "/admin/galgame/messages", "admin-jwt", nil); err != nil {
			t.Fatalf("GetWithToken(/admin/...): %v", err)
		}
		if rec.path != "/api/admin/galgame/messages" {
			t.Errorf("path = %q, want /api/admin/galgame/messages", rec.path)
		}
		if rec.apiKey != "" {
			t.Errorf("X-API-Key = %q, want empty on legacy admin face", rec.apiKey)
		}
	})

	t.Run("user write (create) → internal + key + user JWT (dual credential)", func(t *testing.T) {
		if _, err := c.PostWithToken(ctx, "/galgame", "user-jwt", map[string]any{"x": 1}, ""); err != nil {
			t.Fatalf("PostWithToken: %v", err)
		}
		if rec.path != "/internal/galgame" {
			t.Errorf("path = %q, want /internal/galgame", rec.path)
		}
		if rec.apiKey != "nm_test_key" {
			t.Errorf("X-API-Key = %q, want nm_test_key on internal write face", rec.apiKey)
		}
		if rec.auth != "Bearer user-jwt" {
			t.Errorf("Authorization = %q, want Bearer user-jwt", rec.auth)
		}
	})

	t.Run("user write (draft delete /galgame/:gid) → internal + key", func(t *testing.T) {
		if _, err := c.DeleteWithToken(ctx, "/galgame/321", "user-jwt", nil, ""); err != nil {
			t.Fatalf("DeleteWithToken: %v", err)
		}
		if rec.path != "/internal/galgame/321" {
			t.Errorf("path = %q, want /internal/galgame/321", rec.path)
		}
		if rec.apiKey != "nm_test_key" {
			t.Errorf("X-API-Key = %q, want nm_test_key on internal write face", rec.apiKey)
		}
	})

	t.Run("taxonomy write (PUT /tag) → legacy, no key", func(t *testing.T) {
		if _, err := c.PutWithToken(ctx, "/tag", "user-jwt", map[string]any{"tag_id": 1}, ""); err != nil {
			t.Fatalf("PutWithToken(/tag): %v", err)
		}
		if rec.path != "/api/tag" {
			t.Errorf("path = %q, want /api/tag on legacy staff face", rec.path)
		}
		if rec.apiKey != "" {
			t.Errorf("X-API-Key = %q, want empty on legacy taxonomy-write face", rec.apiKey)
		}
	})

	t.Run("taxonomy write (POST /series) → legacy, no key", func(t *testing.T) {
		if _, err := c.PostWithToken(ctx, "/series", "user-jwt", map[string]any{"name": "x"}, ""); err != nil {
			t.Fatalf("PostWithToken(/series): %v", err)
		}
		if rec.path != "/api/series" {
			t.Errorf("path = %q, want /api/series on legacy staff face", rec.path)
		}
		if rec.apiKey != "" {
			t.Errorf("X-API-Key = %q, want empty on legacy taxonomy-write face", rec.apiKey)
		}
	})

	t.Run("admin write (PUT /admin/galgame/:gid/status) → legacy, no key", func(t *testing.T) {
		if _, err := c.PutWithToken(ctx, "/admin/galgame/5/status", "admin-jwt", map[string]any{"status": 0}, ""); err != nil {
			t.Fatalf("PutWithToken(/admin/...): %v", err)
		}
		if rec.path != "/api/admin/galgame/5/status" {
			t.Errorf("path = %q, want /api/admin/galgame/5/status on legacy staff face", rec.path)
		}
		if rec.apiKey != "" {
			t.Errorf("X-API-Key = %q, want empty on legacy admin-write face", rec.apiKey)
		}
	})

	t.Run("messages feed → internal + key", func(t *testing.T) {
		if _, err := c.MessagesFeed(ctx, 0, 10); err != nil {
			t.Fatalf("MessagesFeed: %v", err)
		}
		if rec.path != "/internal/galgame/messages/feed" {
			t.Errorf("path = %q, want /internal/galgame/messages/feed", rec.path)
		}
		if rec.apiKey != "nm_test_key" {
			t.Errorf("X-API-Key = %q, want nm_test_key on internal feed face", rec.apiKey)
		}
		if rec.auth != "" {
			t.Errorf("Authorization = %q, want empty (no Basic auth on feeds)", rec.auth)
		}
	})

	t.Run("revisions feed → internal + key", func(t *testing.T) {
		if _, err := c.RecentRevisions(ctx, 0, 10); err != nil {
			t.Fatalf("RecentRevisions: %v", err)
		}
		if rec.path != "/internal/galgame/revisions/recent" {
			t.Errorf("path = %q, want /internal/galgame/revisions/recent", rec.path)
		}
		if rec.apiKey != "nm_test_key" {
			t.Errorf("X-API-Key = %q, want nm_test_key on internal feed face", rec.apiKey)
		}
		if rec.auth != "" {
			t.Errorf("Authorization = %q, want empty (no Basic auth on feeds)", rec.auth)
		}
	})
}
