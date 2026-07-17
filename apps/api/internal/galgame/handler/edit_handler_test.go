package handler

// Route-layer tests for the editing-engine BFF (E3a): the real EditHandler
// over a bare Fiber app wired exactly like router.go (auth stub + moderator
// gates), backed by an httptest fake standing in for the catalog edit face.
// They pin the four things the BFF must guarantee:
//   - an unconfigured catalog client degrades every endpoint to 503 with
//     ZERO S2S traffic (never a local fallback);
//   - the asserted actor shape: roles pass through VERBATIM, trust tier is
//     the conservative staff mapping (staff → 3, everyone else → 0);
//   - local validation rejects foreign field keys before the S2S hop;
//   - reads/writes are pinned to kungal's own tenant (a foreign-site
//     proposal reads as 404) and the moderator entry gates hold.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"

	"github.com/gofiber/fiber/v3"
)

// fakeEditFace records every edit-face request and serves canned replies.
type fakeEditFace struct {
	mu       sync.Mutex
	requests []recordedRequest
}

type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Body   map[string]any
}

func (f *fakeEditFace) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		f.mu.Lock()
		f.requests = append(f.requests, recordedRequest{
			Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: body,
		})
		f.mu.Unlock()
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/catalog/edit/proposals":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"merged":false,"proposal":{"id":7,"entity_type":"galgame.game","entity_id":1,"site":"kungal","status":"open","patch":{}}}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/proposals/55":
			// A proposal OUTSIDE kungal's tenant — the BFF must 404 it.
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":55,"entity_type":"galgame.game","entity_id":1,"site":"galgame_wiki","status":"open","patch":{}}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/snapshot":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"entity_type":"galgame.game","entity_id":1,"values":{"galgame.game.name_zh_cn":"现值"}}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/schema/galgame.game":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"entity_type":"galgame.game","fields":[{"key":"galgame.game.name_zh_cn","kind":"text","diff_hint":"inline","locked":false,"can_propose":true,"can_review":false,"would_automerge":false}]}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/proposals":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":233,"message":"not found"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// editTestApp wires the edit routes exactly like router.go, with a stubbed
// session user (nil = anonymous → the auth stub rejects like middleware.Auth).
func editTestApp(t *testing.T, catalogURL string, user *middleware.UserInfo) *fiber.App {
	t.Helper()
	cc := catalogclient.New(catalogclient.Config{BaseURL: catalogURL, ClientID: "cid", ClientSecret: "sec"})
	h := NewEditHandler(cc, nil) // nil wiki client = enrichment off (best-effort)

	app := fiber.New()
	authStub := func(c fiber.Ctx) error {
		if user == nil {
			return c.Status(401).JSON(fiber.Map{"code": 205, "message": "用户登录失效"})
		}
		c.Locals(string(middleware.UserInfoKey), user)
		return c.Next()
	}
	api := app.Group("/api")
	api.Get("/galgame/:gid/edit/revisions", h.Revisions)
	api.Get("/galgame/:gid/edit/diff", h.Diff)
	authed := api.Group("", authStub)
	authed.Get("/galgame/:gid/edit/bootstrap", h.Bootstrap)
	authed.Post("/galgame/:gid/edit/proposals", h.Submit)
	authed.Get("/galgame-edit/mine", h.Mine)
	authed.Post("/galgame-edit/proposals/:id/withdraw", h.Withdraw)
	authed.Get("/galgame-edit/queue", middleware.RequireModerator(), h.Queue)
	authed.Get("/galgame-edit/proposals/:id", middleware.RequireModerator(), h.ProposalDetail)
	authed.Post("/galgame-edit/proposals/:id/amend", middleware.RequireModerator(), h.Amend)
	authed.Post("/galgame-edit/proposals/:id/merge", middleware.RequireModerator(), h.Merge)
	authed.Post("/galgame-edit/proposals/:id/decline", middleware.RequireModerator(), h.Decline)
	return app
}

func doJSON(t *testing.T, app *fiber.App, method, path, body string) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, path, err)
	}
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, raw
}

var moderatorUser = &middleware.UserInfo{ID: 42, Name: "mod", Roles: []string{"moderator"}}
var plainUser = &middleware.UserInfo{ID: 7, Name: "user", Roles: nil}

// TestEditDegradesWhenUnconfigured: every endpoint 503s on an unconfigured
// catalog client, with zero S2S traffic (there is no server to hit).
func TestEditDegradesWhenUnconfigured(t *testing.T) {
	app := editTestApp(t, "", moderatorUser) // empty base URL = not configured
	for _, tc := range []struct{ method, path, body string }{
		{"GET", "/api/galgame/1/edit/bootstrap", ""},
		{"POST", "/api/galgame/1/edit/proposals", `{"patch":{"galgame.game.name_zh_cn":"x"}}`},
		{"GET", "/api/galgame/1/edit/revisions", ""},
		{"GET", "/api/galgame-edit/queue", ""},
		{"GET", "/api/galgame-edit/mine", ""},
	} {
		status, raw := doJSON(t, app, tc.method, tc.path, tc.body)
		if status != http.StatusServiceUnavailable {
			t.Fatalf("%s %s: status = %d body %s, want 503", tc.method, tc.path, status, raw)
		}
	}
}

// TestEditActorAssertionShape pins the S2S actor: a moderator's roles pass
// through verbatim with the staff trust tier, a plain user asserts an empty
// role set at tier 0, and the dirty-field patch reaches the face untouched.
func TestEditActorAssertionShape(t *testing.T) {
	for _, tc := range []struct {
		name      string
		user      *middleware.UserInfo
		wantRoles []any
		wantTier  float64
	}{
		{"moderator", moderatorUser, []any{"moderator"}, 3},
		{"plain user", plainUser, nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeEditFace{}
			app := editTestApp(t, fake.server(t).URL, tc.user)
			status, raw := doJSON(t, app, "POST", "/api/galgame/1/edit/proposals",
				`{"patch":{"galgame.game.name_zh_cn":"新标题"},"note":"typo"}`)
			if status != http.StatusOK {
				t.Fatalf("submit: status = %d body %s", status, raw)
			}
			if len(fake.requests) != 1 {
				t.Fatalf("want exactly one S2S call, got %d", len(fake.requests))
			}
			body := fake.requests[0].Body
			if body["entity_type"] != "galgame.game" || body["site"] != "kungal" {
				t.Fatalf("entity/site assertion wrong: %v", body)
			}
			patch := body["patch"].(map[string]any)
			if patch["galgame.game.name_zh_cn"] != "新标题" {
				t.Fatalf("patch not passed through verbatim: %v", patch)
			}
			actor := body["actor"].(map[string]any)
			if actor["user_id"] != float64(tc.user.ID) {
				t.Fatalf("actor user_id = %v", actor["user_id"])
			}
			tier, _ := actor["trust_tier"].(float64)
			if tier != tc.wantTier {
				t.Fatalf("trust_tier = %v, want %v", actor["trust_tier"], tc.wantTier)
			}
			roles, _ := actor["roles"].([]any)
			if len(roles) != len(tc.wantRoles) {
				t.Fatalf("roles = %v, want %v", roles, tc.wantRoles)
			}
			for i := range roles {
				if roles[i] != tc.wantRoles[i] {
					t.Fatalf("roles = %v, want %v", roles, tc.wantRoles)
				}
			}
		})
	}
}

// TestEditSubmitLocalValidation: foreign field keys and empty patches are
// rejected locally — the S2S face is never touched.
func TestEditSubmitLocalValidation(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	for _, body := range []string{
		`{"patch":{}}`,
		`{"patch":{"catalog.work.display_name":"x"}}`,
		`{"patch":{"gid":1}}`,
	} {
		status, _ := doJSON(t, app, "POST", "/api/galgame/1/edit/proposals", body)
		if status != http.StatusBadRequest {
			t.Fatalf("body %s: status = %d, want 400", body, status)
		}
	}
	if len(fake.requests) != 0 {
		t.Fatalf("local validation must not reach the S2S face, got %d calls", len(fake.requests))
	}
}

// TestEditModeratorGates: the review surface 403s for a plain user at the
// route layer; bootstrap stays open to any logged-in user and anonymous
// callers are rejected by auth.
func TestEditModeratorGates(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/galgame-edit/queue"},
		{"GET", "/api/galgame-edit/proposals/7"},
		{"POST", "/api/galgame-edit/proposals/7/amend"},
		{"POST", "/api/galgame-edit/proposals/7/merge"},
		{"POST", "/api/galgame-edit/proposals/7/decline"},
	} {
		status, _ := doJSON(t, app, tc.method, tc.path, `{"note":"x"}`)
		if status != http.StatusForbidden {
			t.Fatalf("%s %s as plain user: status = %d, want 403", tc.method, tc.path, status)
		}
	}
	if len(fake.requests) != 0 {
		t.Fatalf("gated routes must not reach the S2S face, got %d calls", len(fake.requests))
	}

	anon := editTestApp(t, fake.server(t).URL, nil)
	status, _ := doJSON(t, anon, "GET", "/api/galgame/1/edit/bootstrap", "")
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous bootstrap: status = %d, want 401", status)
	}
}

// TestEditTenantPin: a proposal on a foreign tenant (site=galgame_wiki, the
// old wire) reads as 404 through this BFF — kungal adjudicates only its own.
func TestEditTenantPin(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, moderatorUser)
	status, raw := doJSON(t, app, "GET", "/api/galgame-edit/proposals/55", "")
	if status != http.StatusNotFound {
		t.Fatalf("foreign-tenant proposal: status = %d body %s, want 404", status, raw)
	}
	status, _ = doJSON(t, app, "POST", "/api/galgame-edit/proposals/55/merge", `{"note":""}`)
	if status != http.StatusNotFound {
		t.Fatalf("foreign-tenant merge: status = %d, want 404", status)
	}
}

// TestEditBootstrapShape: bootstrap bundles the snapshot values + the
// capability projection; can_review reflects the projection.
func TestEditBootstrapShape(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	status, raw := doJSON(t, app, "GET", "/api/galgame/1/edit/bootstrap", "")
	if status != http.StatusOK {
		t.Fatalf("bootstrap: status = %d body %s", status, raw)
	}
	var out struct {
		Data struct {
			Gid       int64          `json:"gid"`
			Values    map[string]any `json:"values"`
			Fields    []any          `json:"fields"`
			CanReview bool           `json:"can_review"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Data.Gid != 1 || out.Data.Values["galgame.game.name_zh_cn"] != "现值" {
		t.Fatalf("bootstrap shape wrong: %s", raw)
	}
	if len(out.Data.Fields) != 1 || out.Data.CanReview {
		t.Fatalf("projection shape wrong: %s", raw)
	}
}
