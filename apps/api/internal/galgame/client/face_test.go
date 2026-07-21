package client

// Face-selection tests: prove the galgame client routes each call to the right
// face (internal read face + X-API-Key vs legacy /api) by ROUTE membership, not
// HTTP method, and that an empty API key falls back to legacy (rollback valve).

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
// the key, writes stay on {base}/api, and the Basic-Auth feed stays on
// {base}/api — and that a user JWT rides Authorization alongside X-API-Key.
func TestFaceSelection_WithKey(t *testing.T) {
	rec := &faceRecorder{}
	srv := rec.server(t)
	c := NewGalgameClientWithBasicAuth(srv.URL, "nm_test_key", "", "cid", "sec")
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

	t.Run("write → legacy, no key", func(t *testing.T) {
		if _, err := c.PostWithToken(ctx, "/galgame", "user-jwt", map[string]any{"x": 1}, ""); err != nil {
			t.Fatalf("PostWithToken: %v", err)
		}
		if rec.path != "/api/galgame" {
			t.Errorf("path = %q, want /api/galgame", rec.path)
		}
		if rec.apiKey != "" {
			t.Errorf("X-API-Key = %q, want empty on legacy write face", rec.apiKey)
		}
	})

	t.Run("basic-auth messages feed → legacy", func(t *testing.T) {
		if _, err := c.MessagesFeed(ctx, 0, 10); err != nil {
			t.Fatalf("MessagesFeed: %v", err)
		}
		if rec.path != "/api/galgame/messages/feed" {
			t.Errorf("path = %q, want /api/galgame/messages/feed", rec.path)
		}
		if rec.apiKey != "" {
			t.Errorf("X-API-Key = %q, want empty on legacy feed face", rec.apiKey)
		}
	})

	t.Run("basic-auth revisions feed → legacy", func(t *testing.T) {
		if _, err := c.RecentRevisions(ctx, 0, 10); err != nil {
			t.Fatalf("RecentRevisions: %v", err)
		}
		if rec.path != "/api/galgame/revisions/recent" {
			t.Errorf("path = %q, want /api/galgame/revisions/recent", rec.path)
		}
		if rec.apiKey != "" {
			t.Errorf("X-API-Key = %q, want empty on legacy feed face", rec.apiKey)
		}
	})
}

// TestFaceSelection_EmptyKey proves the rollback valve: with no API key, reads
// fall back to the legacy /api face and never send X-API-Key.
func TestFaceSelection_EmptyKey(t *testing.T) {
	rec := &faceRecorder{}
	srv := rec.server(t)
	c := NewGalgameClient(srv.URL, "") // no API key
	ctx := context.Background()

	if _, err := c.Get(ctx, "/galgame/123", nil); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rec.path != "/api/galgame/123" {
		t.Errorf("path = %q, want /api/galgame/123 (legacy fallback)", rec.path)
	}
	if rec.apiKey != "" {
		t.Errorf("X-API-Key = %q, want empty when no key configured", rec.apiKey)
	}
}
