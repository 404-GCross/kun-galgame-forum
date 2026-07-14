package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"kun-galgame-api/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

// mountGuarded wires a dummy write handler behind the read-only guard, matching
// how router.go prepends it to the old comment write routes.
func mountGuarded(enabled bool) *fiber.App {
	app := fiber.New()
	app.Post("/galgame/1/comment", middleware.GalgameCommentReadonly(enabled), func(c fiber.Ctx) error {
		return c.SendString("wrote")
	})
	return app
}

func doPost(t *testing.T, app *fiber.App) *http.Response {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/galgame/1/comment", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return resp
}

// TestReadonlyOffIsPassthrough proves the guard is transparent when the flag is
// off (the default) — the underlying handler runs, byte-identical to today.
func TestReadonlyOffIsPassthrough(t *testing.T) {
	resp := doPost(t, mountGuarded(false))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if body, _ := io.ReadAll(resp.Body); string(body) != "wrote" {
		t.Fatalf("body = %q, want the handler to have run", body)
	}
}

// TestReadonlyOnBlocksWrites proves the guard answers 503 when the freeze flag
// is on, without reaching the underlying handler.
func TestReadonlyOnBlocksWrites(t *testing.T) {
	resp := doPost(t, mountGuarded(true))
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
	if body, _ := io.ReadAll(resp.Body); string(body) == "wrote" {
		t.Fatal("the underlying write handler must NOT run when read-only")
	}
}
