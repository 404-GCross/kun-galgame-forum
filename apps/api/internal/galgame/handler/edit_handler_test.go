package handler

// Route-layer tests for the editing-engine BFF: the real EditHandler over a
// bare Fiber app wired exactly like router.go (auth stub + moderator gates),
// backed by an httptest fake standing in for both catalog faces.
// They pin the things the BFF must guarantee:
//   - an unconfigured catalog client degrades every endpoint to 503 with
//     ZERO catalog traffic (never a local fallback);
//   - no write asserts an actor: every act rides the caller's own token, and
//     what the caller may do is infra's answer, not a mirrored local gate;
//   - can_decide / can_revert are COMPUTED from the caller's own capability
//     projection, so a control only appears when the write behind it works;
//   - local validation rejects foreign field keys before any hop;
//   - reads are pinned to kungal's own tenant (a foreign-site proposal reads
//     as 404) and the two remaining VIEW gates hold;
//   - the decision side effects (notices, points, timestamp bumps) survive the
//     migration and fire only on a landed decision.

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

// fakeGalgame serves the ID BRIDGE both directions for one entry: gid 1 ⇄
// registry work 1000. The two are deliberately different numbers — an id that
// happened to match on both sides would hide exactly the bug the bridge exists
// to prevent.
func fakeGalgame(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/catalog/lookup/batch":
			raw, _ := io.ReadAll(r.Body)
			if strings.Contains(string(raw), `"external_id":"1"`) {
				_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"external_id":"1","work":{"id":1000}}]}}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[]}}`))
		case "/v1/catalog/works":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":1000,` +
				`"claimed_by":{"site":"kungal","work_id":1,"state":"live"},` +
				`"names":{"zh-cn":"测试游戏"}}],"next_cursor":null}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":233,"message":"not found"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// fakeOwners is the submitter snapshot: entry gid 1 was created by uid 7.
type fakeOwners map[int]int

func (f fakeOwners) OwnerOf(gid int) int { return f[gid] }

// fakeEditFace records every edit-face request and serves canned replies, on
// BOTH planes: the Basic-authed S2S face and the Bearer user face (wave 177).
type fakeEditFace struct {
	mu       sync.Mutex
	requests []recordedRequest
	// userStatus/userBody override the reply on the USER plane only. Scoped to
	// that plane deliberately: a lane that mixes the two (bootstrap reads the
	// snapshot S2S, the projection as the user) must be able to fail exactly the
	// half under test.
	userStatus int
	userBody   string
	// userReviewable flips can_review in the USER-plane schema projection. It is
	// the only input to can_decide / can_revert now that the forum computes both
	// from the caller's own projection instead of from a local role test, so it
	// is how those two are driven.
	userReviewable bool
}

type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Body   map[string]any
	// Face is "s2s" for the Basic-authed /api/v1/catalog/edit/* face, "user" for
	// the Bearer /api/v1/user/catalog/edit/* face, and "other" for anything else
	// — "other" in a recorded request is itself a failure. WHICH of the two
	// planes a lane took is the whole subject of wave 177's routing tests.
	Face string
	Auth string // Authorization header (Basic on S2S, Bearer on the user plane)
}

// s2sFace reports whether a path targets the S2S /api/v1/catalog/edit/* face —
// the asserted-actor channel (owner + moderation lanes).
func s2sFace(path string) bool { return strings.HasPrefix(path, "/api/v1/catalog/edit/") }

// userFace reports whether a path targets the user-token editing face.
func userFace(path string) bool { return strings.HasPrefix(path, "/api/v1/user/catalog/edit/") }

func (f *fakeEditFace) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		face := "other"
		switch {
		case s2sFace(r.URL.Path):
			face = "s2s"
		case userFace(r.URL.Path):
			face = "user"
		}
		f.mu.Lock()
		f.requests = append(f.requests, recordedRequest{
			Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: body,
			Face: face, Auth: r.Header.Get("Authorization"),
		})
		f.mu.Unlock()
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if face == "user" && f.userStatus != 0 {
			w.WriteHeader(f.userStatus)
			_, _ = w.Write([]byte(f.userBody))
			return
		}
		switch {
		// ── the user-token plane (wave 177) ──
		case r.Method == "POST" && r.URL.Path == "/api/v1/user/catalog/edit/proposals":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"merged":false,"proposal":{"id":7,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"open","proposer_uid":9,"patch":{}}}}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/user/catalog/edit/proposals/7/withdraw":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":7,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"withdrawn","patch":{}}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/user/catalog/edit/schema/catalog.work":
			review := "false"
			if f.userReviewable {
				review = "true"
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"entity_type":"catalog.work","fields":[` +
				`{"key":"catalog.work.name_zh_cn","kind":"text","diff_hint":"inline","locked":false,"can_propose":true,"can_review":` + review + `,"would_automerge":false},` +
				// A locked and a deprecated field, neither of which anybody may
				// review: can_revert must ignore both or it could never be true.
				`{"key":"catalog.work.vndb_id","kind":"text","diff_hint":"inline","locked":true,"can_propose":false,"can_review":false,"would_automerge":false},` +
				`{"key":"catalog.work.legacy_alias","kind":"text","diff_hint":"inline","deprecated":true,"locked":false,"can_propose":false,"can_review":false,"would_automerge":false}` +
				`]}}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/user/catalog/edit/proposals/7/amendments":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":1,"seq":1,"amender_uid":7,"set":{"catalog.work.name_zh_cn":"修正"}}}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/user/catalog/edit/proposals/7/merge":
			// The merged revision carries an amender → the notice marks the correction.
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":100,"seq":2,"action":"merged","actor_uid":9,"amender_uid":42,"changed_fields":["catalog.work.name_zh_cn"],"snapshot":{}}}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/user/catalog/edit/proposals/7/decline":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":7,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"declined","proposer_uid":9,"patch":{}}}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/user/catalog/edit/revert":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"proposal":{"id":8,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"merged","patch":{}},"revision":{"id":101,"seq":3,"action":"reverted","actor_uid":7,"changed_fields":["catalog.work.name_zh_cn"],"snapshot":{}}}}`))
		// ── the asserted-actor S2S plane ──
		case r.Method == "POST" && r.URL.Path == "/api/v1/catalog/edit/proposals/7/withdraw":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":7,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"withdrawn","patch":{}}}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/catalog/edit/proposals":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"merged":false,"proposal":{"id":7,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"open","patch":{}}}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/proposals/7":
			// An open kungal proposal on game 1 by uid 9 — the review target.
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":7,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"open","proposer_uid":9,"patch":{"catalog.work.name_zh_cn":"新标题"}}}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/catalog/edit/proposals/7/merge":
			// The merged revision carries an amender → the notice marks the correction.
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":100,"seq":2,"action":"merged","actor_uid":9,"amender_uid":42,"changed_fields":["catalog.work.name_zh_cn"],"snapshot":{}}}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/catalog/edit/proposals/7/decline":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":7,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"declined","proposer_uid":9,"patch":{}}}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/catalog/edit/proposals/7/amendments":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":1,"seq":1,"amender_uid":7,"set":{"catalog.work.name_zh_cn":"修正"}}}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/catalog/edit/revert":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"proposal":{"id":8,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"merged","patch":{}},"revision":{"id":101,"seq":3,"action":"reverted","actor_uid":7,"changed_fields":["catalog.work.name_zh_cn"],"snapshot":{}}}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/proposals/8":
			// An open kungal proposal whose patch is EMPTY — nothing to decide.
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":8,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"open","proposer_uid":9,"patch":{}}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/proposals/9":
			// Two keys, and the EFFECTIVE patch is what counts: the amendments
			// dropped the locked one, so a reviewer of name_zh_cn may decide it.
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":9,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"open","proposer_uid":9,` +
				`"patch":{"catalog.work.name_zh_cn":"新标题","catalog.work.vndb_id":"v1"},` +
				`"effective_patch":{"catalog.work.name_zh_cn":"新标题"}}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/proposals/55":
			// A proposal OUTSIDE kungal's tenant — the BFF must 404 it.
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":55,"entity_type":"catalog.work","entity_id":1000,"site":"letmoe","status":"open","patch":{}}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/snapshot":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"entity_type":"catalog.work","entity_id":1000,"values":{"catalog.work.name_zh_cn":"现值"}}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/schema/catalog.work":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"entity_type":"catalog.work","fields":[{"key":"catalog.work.name_zh_cn","kind":"text","diff_hint":"inline","locked":false,"can_propose":true,"can_review":false,"would_automerge":false}]}}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/catalog/edit/revisions":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":100,"seq":1,"action":"created","actor_uid":7,"changed_fields":["catalog.work.name_zh_cn"],"snapshot":{}}]}}`))
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
	// The id bridge is wired for every app: it is not optional infrastructure
	// any more, it is how a kungal route reaches an entity at all. The
	// degradation test passes an empty catalog URL, which leaves the EDIT face
	// unconfigured — the thing it is actually asserting.
	return editTestAppFull(t, catalogURL, fakeGalgame(t).URL, user, nil)
}

// editTestAppFull additionally wires a fake galgame (owner lookup / naming) and
// a notifier sink (decision notices). Empty galgameURL / nil notifier = off.
func editTestAppFull(t *testing.T, catalogURL, galgameURL string, user *middleware.UserInfo, notifier msgService.Notifier) *fiber.App {
	t.Helper()
	cc := catalogclient.New(catalogclient.Config{BaseURL: catalogURL, ClientID: "cid", ClientSecret: "sec"})
	var galgameClient *client.GalgameClient
	if galgameURL != "" {
		galgameClient = client.New(galgameURL, "nm_test", "")
	}
	// nil user client / repo = best-effort enrichment off. The submitter lookup
	// is a narrow port so the owner-review gates are testable without a database.
	h := NewEditHandler(cc, galgameClient, nil, notifier, nil).
		WithOwners(fakeOwners{1: 7})

	app := fiber.New()
	authStub := func(c fiber.Ctx) error {
		if user == nil {
			return c.Status(401).JSON(fiber.Map{"code": 205, "message": "用户登录失效"})
		}
		c.Locals(string(middleware.UserInfoKey), user)
		// Mirror middleware.Auth: the session's OAuth access token is exposed for
		// the handlers that forward it (the edit chain does not — it is S2S).
		c.Locals(string(middleware.OAuthAccessTokenKey), "user-jwt")
		return c.Next()
	}
	// optAuthStub mirrors middleware.OptionalAuth on the public reads: a session,
	// when there is one, is exposed exactly as the authed group exposes it, and
	// an anonymous caller passes through untouched. can_revert is projected from
	// the token, so a public route that silently dropped it would make that
	// projection untestable.
	optAuthStub := func(c fiber.Ctx) error {
		if user != nil {
			c.Locals(string(middleware.UserInfoKey), user)
			c.Locals(string(middleware.OAuthAccessTokenKey), "user-jwt")
		}
		return c.Next()
	}
	api := app.Group("/api")
	api.Get("/galgame/:gid/edit/revisions", optAuthStub, h.Revisions)
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
	authed.Post("/galgame/:gid/edit/revert", h.Revert)
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
var adminUser = &middleware.UserInfo{ID: 42, Name: "admin", Roles: []string{"admin"}}
var plainUser = &middleware.UserInfo{ID: 7, Name: "user", Roles: nil}

// TestEditDegradesWhenUnconfigured: every endpoint 503s on an unconfigured
// catalog client, with zero S2S traffic (there is no server to hit).
func TestEditDegradesWhenUnconfigured(t *testing.T) {
	app := editTestApp(t, "", moderatorUser) // empty base URL = not configured
	for _, tc := range []struct{ method, path, body string }{
		{"GET", "/api/galgame/1/edit/bootstrap", ""},
		{"POST", "/api/galgame/1/edit/proposals", `{"patch":{"catalog.work.name_zh_cn":"x"}}`},
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

// No edit-face write asserts an actor any more (wave 178). This is the
// whole-chain version of that claim: whatever the lane and whoever the caller,
// a request that reaches the edit face with an `actor` in it — or on the
// Basic-authed plane at all — is a lane that did not migrate.
//
// It is asserted here rather than lane by lane because the failure it guards is
// the silent one: an un-migrated lane still works, and only the recorded request
// says who the catalog thought was acting.
func TestEditNoWriteAssertsAnActor(t *testing.T) {
	// gid 1's creator is uid 7 — the caller who used to have a lane of their own.
	owner := &middleware.UserInfo{ID: 7, Name: "mod-owner", Roles: []string{"moderator"}}
	for _, user := range []*middleware.UserInfo{owner, adminUser, bystander} {
		fake := &fakeEditFace{}
		nm := fakeGalgame(t)
		app := editTestAppFull(t, fake.server(t).URL, nm.URL, user, nil)
		for _, tc := range []struct{ method, path, body string }{
			{"POST", "/api/galgame/1/edit/proposals", `{"patch":{"catalog.work.name_zh_cn":"新标题"},"note":"typo"}`},
			{"POST", "/api/galgame-edit/proposals/7/amend", `{"set":{"catalog.work.name_zh_cn":"修正"}}`},
			{"POST", "/api/galgame-edit/proposals/7/merge", `{"note":""}`},
			{"POST", "/api/galgame-edit/proposals/7/decline", `{"note":"理由"}`},
			{"POST", "/api/galgame/1/edit/revert", `{"to_seq":3}`},
		} {
			if status, raw := doJSON(t, app, tc.method, tc.path, tc.body); status != http.StatusOK {
				t.Fatalf("%s as %s: status = %d body %s", tc.path, user.Name, status, raw)
			}
		}
		for _, r := range fake.requests {
			if r.Method != "POST" {
				continue
			}
			if r.Face != "user" {
				t.Fatalf("%s as %s took the %s plane", r.Path, user.Name, r.Face)
			}
			if r.Auth != "Bearer user-jwt" {
				t.Fatalf("%s as %s used auth %q", r.Path, user.Name, r.Auth)
			}
			if _, ok := r.Body["actor"]; ok {
				t.Fatalf("%s as %s asserted an actor: %v", r.Path, user.Name, r.Body)
			}
			if _, ok := r.Body["site"]; ok {
				t.Fatalf("%s as %s asserted a site: %v", r.Path, user.Name, r.Body)
			}
		}
	}
}

// The patch still reaches the face untouched, and the entity id is the
// TRANSLATED registry work id — the two things the BFF is actually responsible
// for on a submit.
func TestEditSubmitPassesThePatchThrough(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	if status, raw := doJSON(t, app, "POST", "/api/galgame/1/edit/proposals",
		`{"patch":{"catalog.work.name_zh_cn":"新标题"},"note":"typo"}`); status != http.StatusOK {
		t.Fatalf("submit: status = %d body %s", status, raw)
	}
	if len(fake.requests) != 1 || fake.requests[0].Path != "/api/v1/user/catalog/edit/proposals" {
		t.Fatalf("submit must be a single user-plane call, got %+v", fake.requests)
	}
	body := fake.requests[0].Body
	if body["entity_type"] != "catalog.work" || body["entity_id"] != float64(1000) {
		t.Fatalf("entity assertion wrong: %v", body)
	}
	patch := body["patch"].(map[string]any)
	if patch["catalog.work.name_zh_cn"] != "新标题" || body["note"] != "typo" {
		t.Fatalf("patch/note not passed through verbatim: %v", body)
	}
}

// Revert's body: the registry work id, the target seq, no site (the acting
// tenant comes off the token now).
func TestEditRevertRidesTheUserToken(t *testing.T) {
	fake := &fakeEditFace{}
	nm := fakeGalgame(t)
	app := editTestAppFull(t, fake.server(t).URL, nm.URL, plainUser, nil)
	status, raw := doJSON(t, app, "POST", "/api/galgame/1/edit/revert", `{"to_seq":3,"note":"回滚测试"}`)
	if status != http.StatusOK {
		t.Fatalf("revert: status = %d body %s", status, raw)
	}
	req := fake.callTo("/api/v1/user/catalog/edit/revert")
	if req == nil {
		t.Fatalf("revert must ride the user plane, got %+v", fake.requests)
	}
	if req.Body["to_seq"] != float64(3) || req.Body["entity_type"] != "catalog.work" ||
		req.Body["entity_id"] != float64(1000) {
		t.Fatalf("revert body: %v", req.Body)
	}
}

// Infra owns the denial now, so a revert the caller may not perform comes back
// as infra's 403 — the forum runs no gate that could disagree with it.
func TestEditRevertDeniedByInfra(t *testing.T) {
	fake := &fakeEditFace{userStatus: http.StatusForbidden,
		userBody: `{"code":233,"message":"field review denied"}`}
	nm := fakeGalgame(t)
	app := editTestAppFull(t, fake.server(t).URL, nm.URL, bystander, nil)
	status, raw := doJSON(t, app, "POST", "/api/galgame/1/edit/revert", `{"to_seq":3}`)
	if status != http.StatusForbidden {
		t.Fatalf("denied revert: status = %d body %s, want 403", status, raw)
	}
	if code := envelopeCode(t, raw); code != 233 {
		t.Fatalf("code = %d, want a plain permission denial", code)
	}
}

// TestEditSubmitLocalValidation: foreign field keys and empty patches are
// rejected locally — the S2S face is never touched.
func TestEditSubmitLocalValidation(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	for _, body := range []string{
		`{"patch":{}}`,
		`{"patch":{"galgame.game.name_zh_cn":"x"}}`,
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

// TestEditViewGates: what is LEFT of local gating after wave 178 — two entry
// checks onto read surfaces, and nothing in front of a write.
//
//   - the queue 403s for a plain user at the route layer, with zero catalog
//     traffic (a moderator entry, not a policy);
//   - the proposal workbench admits moderators and the entry's creator, and a
//     bystander 403s on the READ without any adjudication surface being
//     consulted;
//   - the adjudication writes themselves are ungated locally: a bystander's
//     amend/merge/decline goes out to infra, which is the only thing entitled
//     to refuse it.
func TestEditViewGates(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	status, _ := doJSON(t, app, "GET", "/api/galgame-edit/queue", "")
	if status != http.StatusForbidden {
		t.Fatalf("queue as plain user: status = %d, want 403", status)
	}
	if len(fake.requests) != 0 {
		t.Fatalf("the queue gate must not reach the catalog, got %d calls", len(fake.requests))
	}

	// A plain user who does NOT own game 1 (creator uid 7, this user is 8).
	viewFake := &fakeEditFace{}
	nm := fakeGalgame(t)
	app = editTestAppFull(t, viewFake.server(t).URL, nm.URL, bystander, nil)
	status, raw := doJSON(t, app, "GET", "/api/galgame-edit/proposals/7", "")
	if status != http.StatusForbidden {
		t.Fatalf("workbench read as bystander: status = %d body %s, want 403", status, raw)
	}
	for _, r := range viewFake.requests {
		if r.Method != "GET" || strings.Contains(r.Path, "schema") {
			t.Fatalf("a refused workbench read must stop at the proposal read, got %s %s", r.Method, r.Path)
		}
	}

	// The writes behind it carry no local gate at all — they reach infra, and
	// infra's own 403 is what the bystander sees.
	writeFake := &fakeEditFace{userStatus: http.StatusForbidden,
		userBody: `{"code":233,"message":"review denied"}`}
	app = editTestAppFull(t, writeFake.server(t).URL, nm.URL, bystander, nil)
	for _, tc := range []struct{ path, body string }{
		{"/api/galgame-edit/proposals/7/amend", `{"set":{"catalog.work.name_zh_cn":"y"}}`},
		{"/api/galgame-edit/proposals/7/merge", `{"note":""}`},
		{"/api/galgame-edit/proposals/7/decline", `{"note":"x"}`},
	} {
		status, raw := doJSON(t, app, "POST", tc.path, tc.body)
		if status != http.StatusForbidden {
			t.Fatalf("%s as bystander: status = %d body %s, want infra's 403", tc.path, status, raw)
		}
	}
	if writeFake.callTo("/api/v1/user/catalog/edit/proposals/7/amendments") == nil {
		t.Fatalf("the amend must actually reach infra: %+v", writeFake.requests)
	}

	anon := editTestApp(t, fake.server(t).URL, nil)
	status, _ = doJSON(t, anon, "GET", "/api/galgame/1/edit/bootstrap", "")
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous bootstrap: status = %d, want 401", status)
	}
}

// can_decide is COMPUTED from the caller's own projection against the
// proposal's own keys — no role test anywhere. It is what makes the workbench's
// buttons a prediction of the write rather than a second opinion about it.
func TestEditProposalDetailCanDecideFromProjection(t *testing.T) {
	for _, tc := range []struct {
		name       string
		id         string
		reviewable bool
		want       bool
	}{
		{"every key reviewable", "7", true, true},
		{"a key the caller may not review", "7", false, false},
		// The effective patch is what would land; its one key is reviewable even
		// though the ORIGINAL patch also names a locked field.
		{"effective patch wins over the original", "9", true, true},
		{"nothing to decide", "8", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeEditFace{userReviewable: tc.reviewable}
			nm := fakeGalgame(t)
			app := editTestAppFull(t, fake.server(t).URL, nm.URL, moderatorUser, nil)
			status, raw := doJSON(t, app, "GET", "/api/galgame-edit/proposals/"+tc.id, "")
			if status != http.StatusOK {
				t.Fatalf("workbench read: status = %d body %s", status, raw)
			}
			var out struct {
				Data struct {
					CanDecide bool `json:"can_decide"`
				} `json:"data"`
			}
			if err := json.Unmarshal(raw, &out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if out.Data.CanDecide != tc.want {
				t.Fatalf("can_decide = %v, want %v (body %s)", out.Data.CanDecide, tc.want, raw)
			}
			// The projection must be the VIEWER's own, or the prediction is about
			// somebody else.
			if req := fake.callTo("/api/v1/user/catalog/edit/schema/catalog.work"); req == nil {
				t.Fatalf("the workbench projection must ride the user plane: %+v", fake.requests)
			}
		})
	}
}

// can_revert is the same idea on the history page: projected, not role-tested.
// Locked and deprecated fields are excluded — nobody may review those, so
// counting them would make the control unreachable for everyone.
func TestEditRevisionsCanRevertFromProjection(t *testing.T) {
	for _, tc := range []struct {
		name       string
		reviewable bool
		token      bool
		want       bool
	}{
		{"reviews every editable field", true, true, true},
		{"cannot review an editable field", false, true, false},
		{"anonymous reader", true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeEditFace{userReviewable: tc.reviewable}
			var user *middleware.UserInfo
			if tc.token {
				user = moderatorUser
			}
			app := editTestApp(t, fake.server(t).URL, user)
			// The route is optionally authed, so it is served either way; the
			// anonymous case simply carries no token to project with.
			status, raw := doJSON(t, app, "GET", "/api/galgame/1/edit/revisions", "")
			if status != http.StatusOK {
				t.Fatalf("revisions: status = %d body %s", status, raw)
			}
			var out struct {
				Data struct {
					CanRevert bool `json:"can_revert"`
				} `json:"data"`
			}
			if err := json.Unmarshal(raw, &out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if out.Data.CanRevert != tc.want {
				t.Fatalf("can_revert = %v, want %v (body %s)", out.Data.CanRevert, tc.want, raw)
			}
		})
	}
}

// TestEditOwnerReview: the game's creator — a plain user, no roles — opens the
// workbench, and their merge goes out on their OWN token with nothing asserted.
// The capability that used to require asserting is_entity_owner now arrives in
// the projection instead (userReviewable stands for infra having derived it),
// and the merge's side effects are unchanged: the proposer is notified with the
// correction marker, because the merged revision carries an amender.
func TestEditOwnerReview(t *testing.T) {
	fake := &fakeEditFace{userReviewable: true}
	nm := fakeGalgame(t)
	sink := &fakeNotifier{}
	app := editTestAppFull(t, fake.server(t).URL, nm.URL, plainUser, sink) // uid 7 = the creator

	status, raw := doJSON(t, app, "GET", "/api/galgame-edit/proposals/7", "")
	if status != http.StatusOK {
		t.Fatalf("owner workbench read: status = %d body %s", status, raw)
	}
	var detail struct {
		Data struct {
			CanDecide bool `json:"can_decide"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &detail)
	if !detail.Data.CanDecide {
		t.Fatalf("an owner whose projection reviews the patch must get can_decide: %s", raw)
	}
	if fake.callTo("/api/v1/catalog/edit/schema/catalog.work") != nil {
		t.Fatalf("no projection may still be asked for on the S2S face: %+v", fake.requests)
	}

	status, raw = doJSON(t, app, "POST", "/api/galgame-edit/proposals/7/merge", `{"note":""}`)
	if status != http.StatusOK {
		t.Fatalf("owner merge: status = %d body %s", status, raw)
	}
	req := fake.callTo("/api/v1/user/catalog/edit/proposals/7/merge")
	if req == nil || req.Auth != "Bearer user-jwt" {
		t.Fatalf("the owner's merge must ride their own token: %+v", fake.requests)
	}
	if _, ok := req.Body["actor"]; ok {
		t.Fatalf("the merge must assert no actor: %v", req.Body)
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

// TestEditDeclineNotification: who may decline is infra's call now (the forum
// used to run an admin-or-owner mirror of it, and mirrors drift). What the forum
// still owns is the NOTICE — the decline reason travels to the proposer in full
// (E3b ruling 1), addressed by kungal id, and only once the decline landed.
func TestEditDeclineNotification(t *testing.T) {
	fake := &fakeEditFace{}
	nm := fakeGalgame(t)
	sink := &fakeNotifier{}

	// A refused decline notifies nobody: the notice hangs off the landed
	// decision, not off the attempt.
	deniedFake := &fakeEditFace{userStatus: http.StatusForbidden,
		userBody: `{"code":233,"message":"review denied"}`}
	deniedSink := &fakeNotifier{}
	deniedApp := editTestAppFull(t, deniedFake.server(t).URL, nm.URL, moderatorUser, deniedSink)
	if status, raw := doJSON(t, deniedApp, "POST", "/api/galgame-edit/proposals/7/decline", `{"note":"x"}`); status != http.StatusForbidden {
		t.Fatalf("denied decline: status = %d body %s, want 403", status, raw)
	}
	deniedSink.mu.Lock()
	if len(deniedSink.specs) != 0 {
		t.Fatalf("a refused decline must notify nobody, got %+v", deniedSink.specs)
	}
	deniedSink.mu.Unlock()

	app := editTestAppFull(t, fake.server(t).URL, nm.URL, adminUser, sink)
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
	if out.Data.Gid != 1 || out.Data.Values["catalog.work.name_zh_cn"] != "现值" {
		t.Fatalf("bootstrap shape wrong: %s", raw)
	}
	if len(out.Data.Fields) != 3 || out.Data.CanReview {
		t.Fatalf("projection shape wrong: %s", raw)
	}
}
