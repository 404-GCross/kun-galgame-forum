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

	"kun-galgame-api/internal/galgame/client"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// fakeNotifier records every notification the BFF emits (E3b decision
// notices) without touching a database.
type fakeNotifier struct {
	mu    sync.Mutex
	specs []msgService.Spec
}

func (f *fakeNotifier) Emit(_ *gorm.DB, spec msgService.Spec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.specs = append(f.specs, spec)
	return nil
}

func (f *fakeNotifier) EmitMany(tx *gorm.DB, specs []msgService.Spec) error {
	for _, s := range specs {
		_ = f.Emit(tx, s)
	}
	return nil
}

// fakeWiki serves the galgame batch read (owner lookup + naming): one game,
// id 1, created by uid 7.
func fakeWiki(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/galgame/batch" {
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":[{"id":1,"user_id":7,"name_zh_cn":"测试游戏"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":233,"message":"not found"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

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
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/proposals/7":
			// An open kungal proposal on game 1 by uid 9 — the review target.
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":7,"entity_type":"galgame.game","entity_id":1,"site":"kungal","status":"open","proposer_uid":9,"patch":{"galgame.game.name_zh_cn":"新标题"}}}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/catalog/edit/proposals/7/merge":
			// The merged revision carries an amender → the notice marks the correction.
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":100,"seq":2,"action":"merged","actor_uid":9,"amender_uid":42,"changed_fields":["galgame.game.name_zh_cn"],"snapshot":{}}}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/catalog/edit/proposals/7/decline":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":7,"entity_type":"galgame.game","entity_id":1,"site":"kungal","status":"declined","proposer_uid":9,"patch":{}}}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/catalog/edit/proposals/7/amendments":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":1,"seq":1,"amender_uid":7,"set":{"galgame.game.name_zh_cn":"修正"}}}`))
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
	return editTestAppFull(t, catalogURL, "", user, nil)
}

// editTestAppFull additionally wires a fake wiki (owner lookup / naming) and
// a notifier sink (decision notices). Empty wikiURL / nil notifier = off.
func editTestAppFull(t *testing.T, catalogURL, wikiURL string, user *middleware.UserInfo, notifier msgService.Notifier) *fiber.App {
	t.Helper()
	cc := catalogclient.New(catalogclient.Config{BaseURL: catalogURL, ClientID: "cid", ClientSecret: "sec"})
	var wiki *client.GalgameClient
	if wikiURL != "" {
		wiki = client.NewGalgameClient(wikiURL, "")
	}
	h := NewEditHandler(cc, wiki, nil, notifier, nil) // nil user client/repo = best-effort off

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
	api.Get("/galgame/:gid/edit/proposals", h.GameProposals)
	authed := api.Group("", authStub)
	authed.Get("/galgame/:gid/edit/bootstrap", h.Bootstrap)
	authed.Post("/galgame/:gid/edit/proposals", h.Submit)
	authed.Get("/galgame-edit/mine", h.Mine)
	authed.Post("/galgame-edit/proposals/:id/withdraw", h.Withdraw)
	authed.Get("/galgame-edit/queue", middleware.RequireModerator(), h.Queue)
	authed.Get("/galgame-edit/proposals/:id", h.ProposalDetail)
	authed.Post("/galgame-edit/proposals/:id/amend", h.Amend)
	authed.Post("/galgame-edit/proposals/:id/merge", h.Merge)
	authed.Post("/galgame-edit/proposals/:id/decline", h.Decline)
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

// TestEditReviewEntryGates: the queue 403s for a plain user at the route
// layer with zero S2S traffic; the proposal-directed review surfaces admit
// moderators and the game's creator (E3b) — a plain NON-owner user 403s in
// the handler after the entry check. Bootstrap stays open to any logged-in
// user and anonymous callers are rejected by auth.
func TestEditReviewEntryGates(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	status, _ := doJSON(t, app, "GET", "/api/galgame-edit/queue", "")
	if status != http.StatusForbidden {
		t.Fatalf("queue as plain user: status = %d, want 403", status)
	}
	if len(fake.requests) != 0 {
		t.Fatalf("the queue gate must not reach the S2S face, got %d calls", len(fake.requests))
	}

	// A plain user who does NOT own game 1 (fake wiki: creator uid 7, this
	// user is 8): every proposal-directed surface 403s after the entry check
	// and no write ever reaches the face.
	nonOwner := &middleware.UserInfo{ID: 8, Name: "bystander", Roles: nil}
	wiki := fakeWiki(t)
	app = editTestAppFull(t, fake.server(t).URL, wiki.URL, nonOwner, nil)
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/galgame-edit/proposals/7"},
		{"POST", "/api/galgame-edit/proposals/7/amend"},
		{"POST", "/api/galgame-edit/proposals/7/merge"},
		{"POST", "/api/galgame-edit/proposals/7/decline"},
	} {
		status, _ := doJSON(t, app, tc.method, tc.path, `{"note":"x","set":{"galgame.game.name_zh_cn":"y"}}`)
		if status != http.StatusForbidden {
			t.Fatalf("%s %s as non-owner: status = %d, want 403", tc.method, tc.path, status)
		}
	}
	for _, r := range fake.requests {
		if r.Method != "GET" {
			t.Fatalf("non-owner must never reach a write op, got %s %s", r.Method, r.Path)
		}
	}

	anon := editTestApp(t, fake.server(t).URL, nil)
	status, _ = doJSON(t, anon, "GET", "/api/galgame/1/edit/bootstrap", "")
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous bootstrap: status = %d, want 401", status)
	}
}

// TestEditOwnerReview (E3b): the game's creator — a plain user, no roles —
// passes the review entry, the owner assertion rides the S2S actor, the
// merge lands, and the proposer is notified with the correction marker
// (the merged revision carries an amender).
func TestEditOwnerReview(t *testing.T) {
	fake := &fakeEditFace{}
	wiki := fakeWiki(t)
	sink := &fakeNotifier{}
	app := editTestAppFull(t, fake.server(t).URL, wiki.URL, plainUser, sink) // uid 7 = the creator

	status, raw := doJSON(t, app, "GET", "/api/galgame-edit/proposals/7", "")
	if status != http.StatusOK {
		t.Fatalf("owner workbench read: status = %d body %s", status, raw)
	}
	var schemaQuery string
	for _, r := range fake.requests {
		if strings.HasPrefix(r.Path, "/api/v1/catalog/edit/schema/") {
			schemaQuery = r.Query
		}
	}
	if !strings.Contains(schemaQuery, "is_entity_owner=true") {
		t.Fatalf("owner assertion missing from the schema projection query: %q", schemaQuery)
	}

	status, raw = doJSON(t, app, "POST", "/api/galgame-edit/proposals/7/merge", `{"note":""}`)
	if status != http.StatusOK {
		t.Fatalf("owner merge: status = %d body %s", status, raw)
	}
	var mergeBody map[string]any
	for _, r := range fake.requests {
		if r.Method == "POST" && strings.HasSuffix(r.Path, "/merge") {
			mergeBody = r.Body
		}
	}
	actor, _ := mergeBody["actor"].(map[string]any)
	if actor == nil || actor["is_entity_owner"] != true {
		t.Fatalf("owner assertion missing from the merge actor: %v", mergeBody)
	}

	sink.mu.Lock()
	defer sink.mu.Unlock()
	if len(sink.specs) != 1 {
		t.Fatalf("want exactly one merged notice, got %d", len(sink.specs))
	}
	n := sink.specs[0]
	if n.Kind != msgService.NotifyMerged || n.ReceiverID != 9 || n.SenderID != 7 || n.GalgameID != 1 {
		t.Fatalf("merged notice shape: %+v", n)
	}
	if !strings.Contains(n.Content, "测试游戏") || !strings.Contains(n.Content, "修正") {
		t.Fatalf("merged notice content must name the game + mark the correction: %q", n.Content)
	}
}

// TestEditDeclineNotification: the decline reason travels to the proposer in
// full on the decline notice (E3b ruling 1).
func TestEditDeclineNotification(t *testing.T) {
	fake := &fakeEditFace{}
	wiki := fakeWiki(t)
	sink := &fakeNotifier{}
	app := editTestAppFull(t, fake.server(t).URL, wiki.URL, moderatorUser, sink)

	status, raw := doJSON(t, app, "POST", "/api/galgame-edit/proposals/7/decline", `{"note":"资料来源不可靠，请补充出处"}`)
	if status != http.StatusOK {
		t.Fatalf("decline: status = %d body %s", status, raw)
	}
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if len(sink.specs) != 1 {
		t.Fatalf("want exactly one declined notice, got %d", len(sink.specs))
	}
	n := sink.specs[0]
	if n.Kind != msgService.NotifyDeclined || n.ReceiverID != 9 || n.SenderID != 42 || n.GalgameID != 1 {
		t.Fatalf("declined notice shape: %+v", n)
	}
	if !strings.Contains(n.Content, "资料来源不可靠，请补充出处") {
		t.Fatalf("declined notice must carry the reason in full: %q", n.Content)
	}
}

// TestEditGameProposals: the per-game proposal list is a public read (the
// old wire's PR list parity).
func TestEditGameProposals(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, nil) // anonymous
	status, raw := doJSON(t, app, "GET", "/api/galgame/1/edit/proposals", "")
	if status != http.StatusOK {
		t.Fatalf("game proposals: status = %d body %s", status, raw)
	}
	if len(fake.requests) != 1 || !strings.Contains(fake.requests[0].Query, "entity_id=1") {
		t.Fatalf("list must filter to the game: %+v", fake.requests)
	}
	if !strings.Contains(fake.requests[0].Query, "status=0") && !strings.Contains(fake.requests[0].Query, "status=open") {
		t.Fatalf("list must default to open proposals: %q", fake.requests[0].Query)
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
