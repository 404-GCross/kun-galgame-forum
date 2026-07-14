package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"kun-galgame-api/internal/galgame/handler"

	"github.com/gofiber/fiber/v3"
)

// routeExists reports whether the app has a route with the given method + path.
func routeExists(app *fiber.App, method, path string) bool {
	for _, r := range app.GetRoutes() {
		if r.Method == method && r.Path == path {
			return true
		}
	}
	return false
}

var communityRoutes = []struct{ method, path string }{
	{http.MethodGet, "/galgame/:gid/comments"},
	{http.MethodGet, "/galgame/:gid/comments/locate"},
	{http.MethodPost, "/galgame/:gid/comments"},
	{http.MethodPut, "/galgame/comments/:postId"},
	{http.MethodDelete, "/galgame/comments/:postId"},
	{http.MethodPut, "/galgame/comments/:postId/like"},
	{http.MethodPost, "/galgame/comments/:postId/flag"},
}

// TestCommunityRoutesGatedOffByFlag is the flag-off invariant probe: with
// enabled=false NONE of the new routes are mounted, so a request to one 404s —
// byte-identical to a build without the migration.
func TestCommunityRoutesGatedOffByFlag(t *testing.T) {
	app := fiber.New()
	handler.NewCommunityCommentHandler(nil).Register(app, app, false)

	for _, rt := range communityRoutes {
		if routeExists(app, rt.method, rt.path) {
			t.Errorf("route %s %s registered when flag OFF (must be absent)", rt.method, rt.path)
		}
	}
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/galgame/1/comments", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /galgame/1/comments = %d, want 404 when flag off", resp.StatusCode)
	}
}

// TestCommunityRoutesRegisteredWhenEnabled proves all six+locate routes mount
// when the flag is on (introspected, not invoked — the nil service is never hit).
func TestCommunityRoutesRegisteredWhenEnabled(t *testing.T) {
	app := fiber.New()
	handler.NewCommunityCommentHandler(nil).Register(app, app, true)

	for _, rt := range communityRoutes {
		if !routeExists(app, rt.method, rt.path) {
			t.Errorf("route %s %s NOT registered when flag on", rt.method, rt.path)
		}
	}
}
