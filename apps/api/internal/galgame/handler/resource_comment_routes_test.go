package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"kun-galgame-api/internal/galgame/handler"

	"github.com/gofiber/fiber/v3"
)

var resourceCommentRoutes = []struct{ method, path string }{
	{http.MethodGet, "/galgame-rating/:id/comments"},
	{http.MethodGet, "/website/:domain/comments"},
	{http.MethodGet, "/toolset/:id/comments"},
	{http.MethodPost, "/galgame-rating/:id/comments"},
	{http.MethodDelete, "/galgame-rating/:id/comments/:postId"},
	{http.MethodPost, "/website/:domain/comments"},
	{http.MethodDelete, "/website/:domain/comments/:postId"},
	{http.MethodPost, "/toolset/:id/comments"},
	{http.MethodDelete, "/toolset/:id/comments/:postId"},
}

// TestResourceCommentRoutesGatedOffByFlag is the flag-off invariant probe: with
// enabled=false NONE of the new resource comment routes are mounted, so a request
// to one 404s — byte-identical to a build without the migration.
func TestResourceCommentRoutesGatedOffByFlag(t *testing.T) {
	app := fiber.New()
	h := handler.NewResourceCommentHandler(nil)
	h.RegisterReads(app, false)
	h.RegisterWrites(app, false)

	for _, rt := range resourceCommentRoutes {
		if routeExists(app, rt.method, rt.path) {
			t.Errorf("route %s %s registered when flag OFF (must be absent)", rt.method, rt.path)
		}
	}
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/galgame-rating/1/comments", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /galgame-rating/1/comments = %d, want 404 when flag off", resp.StatusCode)
	}
}

// TestResourceCommentRoutesRegisteredWhenEnabled proves all nine routes mount
// when the flag is on (introspected, not invoked — the nil service is never hit).
func TestResourceCommentRoutesRegisteredWhenEnabled(t *testing.T) {
	app := fiber.New()
	h := handler.NewResourceCommentHandler(nil)
	h.RegisterReads(app, true)
	h.RegisterWrites(app, true)

	for _, rt := range resourceCommentRoutes {
		if !routeExists(app, rt.method, rt.path) {
			t.Errorf("route %s %s NOT registered when flag on", rt.method, rt.path)
		}
	}
}

// TestResourceCommentReadsAnonymous pins the router-ordering rule that keeps the
// public reads anonymous: mounted like the real router (reads BEFORE a
// mandatory-auth Use, writes after), an anonymous GET must reach the read handler
// instead of being rejected by the auth middleware; an anonymous POST must be
// rejected by the boundary.
func TestResourceCommentReadsAnonymous(t *testing.T) {
	app := fiber.New()
	h := handler.NewResourceCommentHandler(nil)

	h.RegisterReads(app, true)
	// The mandatory-auth boundary, as in router.go — everything after is gated.
	app.Use(func(c fiber.Ctx) error { return c.SendStatus(http.StatusUnauthorized) })
	h.RegisterWrites(app, true)

	// Anonymous read reaches the handler: a non-numeric id fails parsing and 400s
	// (proving the handler ran) rather than being intercepted as the boundary 401.
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/galgame-rating/abc/comments", nil))
	if err != nil {
		t.Fatalf("app.Test read: %v", err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatal("anonymous GET /galgame-rating/:id/comments was rejected by the mandatory-auth boundary — reads mounted on the wrong side")
	}

	// Writes stay behind the boundary: anonymous POST is rejected by it.
	resp, err = app.Test(httptest.NewRequest(http.MethodPost, "/toolset/1/comments", nil))
	if err != nil {
		t.Fatalf("app.Test write: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous POST = %d, want the boundary's 401", resp.StatusCode)
	}
}
