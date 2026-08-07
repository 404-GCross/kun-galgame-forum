package handler

// galgame_upload_test.go — boundary tests for /image/galgame.
//
// Scope (handler-level, no image_service round-trip):
//   - preset allowlist enforcement (must be galgame_banner / galgame_screenshot)
//   - file presence / size limit
//
// Successful upload is exercised end-to-end against a real image_service
// in deployment smoke tests, not here — we don't mock the entire SDK
// just to assert it forwards. The contract that matters at this layer
// is "kungal rejects junk before we even talk to image_service".

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"kun-galgame-api/internal/image/repository"
	"kun-galgame-api/internal/image/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
)

// newTestApp wires a Fiber app with the upload handler + a session stub
// so MustGetUser succeeds for UID=1. ImageService is constructed without a
// galgame client (galgameClient=nil) — galgame uploads now proxy to the galgame's
// /galgame/image — the handler boundary checks (preset / file / size) run
// before the service touches the client, so the missing client only matters
// once those pass.
func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	// BodyLimit 20 MiB so the oversized test can reach our handler's
	// MaxImageSize check (Fiber's default 4 MiB would pre-reject before
	// the handler sees the body, which makes the test exercise the
	// framework instead of our contract).
	app := fiber.New(fiber.Config{BodyLimit: 20 * 1024 * 1024})
	// Stub auth middleware: every request is UID=1. Real middleware
	// stores a *UserInfo (pointer) — see middleware.MustGetUser's type
	// assertion — so we match that here.
	app.Use(func(c fiber.Ctx) error {
		c.Locals(string(middleware.UserInfoKey), &middleware.UserInfo{ID: 1})
		return c.Next()
	})
	svc := service.NewImageService(&repository.ImageRepository{}, nil, nil)
	h := NewImageHandler(svc)
	app.Post("/image/galgame", h.UploadGalgameImage)
	return app
}

// makeMultipart builds a multipart form body with the given fields.
func makeMultipart(t *testing.T, fields map[string]string, fileName string, fileBytes []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	if fileName != "" {
		fw, err := mw.CreateFormFile("file", fileName)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := io.Copy(fw, bytes.NewReader(fileBytes)); err != nil {
			t.Fatalf("copy file: %v", err)
		}
	}
	_ = mw.Close()
	return body, mw.FormDataContentType()
}

func TestUploadGalgameImage_RejectsBadPreset(t *testing.T) {
	app := newTestApp(t)
	body, ct := makeMultipart(t, map[string]string{"preset": "topic"}, "x.png", []byte("not-an-image"))
	req := httptest.NewRequest("POST", "/image/galgame", body)
	req.Header.Set("Content-Type", ct)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(raw), "preset") {
		t.Errorf("expected preset error message, got %s", raw)
	}
}

func TestUploadGalgameImage_RejectsMissingFile(t *testing.T) {
	app := newTestApp(t)
	body, ct := makeMultipart(t, map[string]string{"preset": "galgame_banner"}, "", nil)
	req := httptest.NewRequest("POST", "/image/galgame", body)
	req.Header.Set("Content-Type", ct)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(raw), "图片") {
		t.Errorf("expected missing-file error, got %s", raw)
	}
}

func TestUploadGalgameImage_AcceptsAllowedPresets(t *testing.T) {
	// Both gated presets must pass the boundary and reach the service. With
	// galgameClient=nil the service returns "未配置" — that's how we confirm the
	// boundary checks all passed (preset OK, file OK, size OK) without a real
	// galgame round-trip.
	for _, preset := range []string{"galgame_banner", "galgame_screenshot"} {
		t.Run(preset, func(t *testing.T) {
			app := newTestApp(t)
			body, ct := makeMultipart(t, map[string]string{"preset": preset}, "x.png", []byte("xxxxx"))
			req := httptest.NewRequest("POST", "/image/galgame", body)
			req.Header.Set("Content-Type", ct)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()
			raw, _ := io.ReadAll(resp.Body)
			if !strings.Contains(string(raw), "未配置") {
				t.Errorf("preset %s expected service-level未配置 (boundary passed), got %s",
					preset, raw)
			}
		})
	}
}

// MaxImageSize boundary — fabricate a body claiming a large file via
// Content-Length doesn't actually trigger the check (Fiber reads body).
// Instead build a body larger than the limit and confirm rejection.
func TestUploadGalgameImage_RejectsOversized(t *testing.T) {
	app := newTestApp(t)
	big := bytes.Repeat([]byte("a"), int(service.MaxImageSize)+1)
	body, ct := makeMultipart(t, map[string]string{"preset": "galgame_banner"}, "big.png", big)
	req := httptest.NewRequest("POST", "/image/galgame", body)
	req.Header.Set("Content-Type", ct)
	// Timeout: 0 disables the test client timeout (v2 passed -1); the oversized
	// body can take longer than the 1s default to stream in.
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(raw), "10MB") && !strings.Contains(string(raw), "大小") {
		t.Errorf("expected size-limit error, got status=%d body=%s", resp.StatusCode, raw)
	}
}

// TestUploadGalgameImage_ForwardsTheUploadersToken pins wave 180's half of the
// U2 lane: the upload was the last edit-face WRITE that named its actor in a
// form field, honoured upstream on the forum's word alone. It now travels on
// the uploader's own token, and the failure it replaces is silent — an
// `actor_uid` left in place keeps working while attributing the image to
// whoever the forum said, so the absence of the field is the assertion.
func TestUploadGalgameImage_ForwardsTheUploadersToken(t *testing.T) {
	var (
		gotAuth   string
		gotPath   string
		gotFields = map[string]string{}
	)
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Errorf("content type: %v", err)
			return
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			raw, _ := io.ReadAll(part)
			if part.FileName() == "" {
				gotFields[part.FormName()] = string(raw)
			}
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"hash":"h1","url":"https://cdn/image/h1"}}`))
	}))
	defer catalog.Close()

	app := fiber.New(fiber.Config{BodyLimit: 20 * 1024 * 1024})
	app.Use(func(c fiber.Ctx) error {
		c.Locals(string(middleware.UserInfoKey), &middleware.UserInfo{ID: 1})
		c.Locals(string(middleware.OAuthAccessTokenKey), "user-jwt")
		return c.Next()
	})
	svc := service.NewImageService(&repository.ImageRepository{}, nil,
		catalogclient.New(catalogclient.Config{BaseURL: catalog.URL, ClientID: "cid", ClientSecret: "sec"}))
	app.Post("/image/galgame", NewImageHandler(svc).UploadGalgameImage)

	body, ct := makeMultipart(t, map[string]string{"preset": "galgame_banner"}, "x.png", []byte("PNG!"))
	req := httptest.NewRequest("POST", "/image/galgame", body)
	req.Header.Set("Content-Type", ct)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload: status = %d body %s", resp.StatusCode, raw)
	}
	if gotPath != "/api/v1/user/catalog/edit/images" {
		t.Errorf("upload hit %q, want the user plane's image leg", gotPath)
	}
	if gotAuth != "Bearer user-jwt" {
		t.Errorf("auth = %q, want the uploader's own bearer", gotAuth)
	}
	if _, ok := gotFields["actor_uid"]; ok {
		t.Errorf("the upload must assert no actor: %v", gotFields)
	}
	if gotFields["preset"] != "galgame_banner" {
		t.Errorf("preset lost on the wire: %v", gotFields)
	}
}
