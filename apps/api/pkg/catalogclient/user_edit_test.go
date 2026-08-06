package catalogclient

// Contract tests for the editing engine on the user-token plane (wave 177).
// The axes are the ones a wrong implementation gets wrong SILENTLY: which
// credential travels (a Basic header here would restore the S2S posture and
// void the whole point), which fields do NOT travel (an actor or a site in the
// body is a claim this plane does not make), and the error mapping — above all
// the scope denial, the difference between "log out and back in" and "you are
// not allowed".

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// captured is what a recording fake saw on the one request it served.
type captured struct {
	method string
	path   string
	query  string
	auth   string
	body   map[string]any
}

func recordingServer(t *testing.T, status int, reply string) (*httptest.Server, *captured) {
	t.Helper()
	var got captured
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got.method, got.path, got.query = r.Method, r.URL.Path, r.URL.RawQuery
		got.auth = r.Header.Get("Authorization")
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &got.body)
		}
		if status != 0 {
			w.WriteHeader(status)
		}
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

func userClient(baseURL string) *Client {
	return New(Config{BaseURL: baseURL, ClientID: "cid", ClientSecret: "sec"})
}

// TestCreateEditProposalUser_TravelsAsTheUser: the bearer is the user's, the
// path is the user plane, and the body carries NEITHER actor NOR site — the
// catalog derives both from the token, and sending them would be a claim.
func TestCreateEditProposalUser_TravelsAsTheUser(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"merged":false,`+
		`"proposal":{"id":7,"entity_type":"catalog.work","entity_id":1000,"site":"kungal",`+
		`"status":"open","proposer_uid":9,"patch":{"catalog.work.name_zh_cn":"新标题"}}}}`)

	res, err := userClient(srv.URL).CreateEditProposalUser(context.Background(), "user-jwt",
		UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000,
			Patch: map[string]any{"catalog.work.name_zh_cn": "新标题"}, Note: "typo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodPost || got.path != "/api/v1/user/catalog/edit/proposals" {
		t.Fatalf("create hit %s %s", got.method, got.path)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	if strings.HasPrefix(got.auth, "Basic ") {
		t.Fatal("the user plane must never attach the client credential")
	}
	if _, ok := got.body["actor"]; ok {
		t.Fatalf("the user plane asserts no actor: %v", got.body)
	}
	if _, ok := got.body["site"]; ok {
		t.Fatalf("the user plane asserts no site: %v", got.body)
	}
	if got.body["entity_type"] != "catalog.work" || got.body["note"] != "typo" {
		t.Fatalf("create body wrong: %v", got.body)
	}
	patch, _ := got.body["patch"].(map[string]any)
	if patch["catalog.work.name_zh_cn"] != "新标题" {
		t.Fatalf("patch not passed through verbatim: %v", got.body)
	}
	if res.Merged || res.Proposal.ID != 7 || res.Proposal.Status != "open" {
		t.Fatalf("pending result decoded wrong: %+v", res)
	}
}

// An admin's token still automerges upstream — the roles ride IN the token, so
// the plane does not decide the outcome and the merged shape must decode whole
// (the revision included, or the caller cannot report what landed).
func TestCreateEditProposalUser_DecodesAutomerge(t *testing.T) {
	srv, _ := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"merged":true,`+
		`"proposal":{"id":8,"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"merged","patch":{}},`+
		`"revision":{"id":100,"seq":4,"action":"merged","actor_uid":42,"changed_fields":["catalog.work.name_zh_cn"],"snapshot":{}}}}`)

	res, err := userClient(srv.URL).CreateEditProposalUser(context.Background(), "admin-jwt",
		UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000,
			Patch: map[string]any{"catalog.work.name_zh_cn": "x"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Merged || res.Revision == nil || res.Revision.Seq != 4 {
		t.Fatalf("automerge result decoded wrong: %+v", res)
	}
}

func TestWithdrawEditProposalUser(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"id":7,`+
		`"entity_type":"catalog.work","entity_id":1000,"site":"kungal","status":"withdrawn","patch":{}}}`)

	prop, err := userClient(srv.URL).WithdrawEditProposalUser(context.Background(), "user-jwt", 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodPost || got.path != "/api/v1/user/catalog/edit/proposals/7/withdraw" {
		t.Fatalf("withdraw hit %s %s", got.method, got.path)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	// The proposer is the token subject; a withdraw body that named an actor
	// would be asking to close somebody else's proposal.
	if _, ok := got.body["actor"]; ok {
		t.Fatalf("withdraw must assert no actor: %v", got.body)
	}
	if prop.Status != "withdrawn" || prop.ID != 7 {
		t.Fatalf("withdraw result decoded wrong: %+v", prop)
	}
}

// The schema projection is entity-aware but actor-FREE: uid, roles and site all
// come off the token, so entity_id is the only query parameter there is.
func TestGetEditSchemaUser(t *testing.T) {
	srv, got := recordingServer(t, 0, `{"code":0,"message":"ok","data":{"entity_type":"catalog.work",`+
		`"fields":[{"key":"catalog.work.name_zh_cn","kind":"text","diff_hint":"inline",`+
		`"locked":false,"can_propose":true,"can_review":false,"would_automerge":false}]}}`)

	schema, err := userClient(srv.URL).GetEditSchemaUser(context.Background(), "user-jwt", "catalog.work", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != http.MethodGet || got.path != "/api/v1/user/catalog/edit/schema/catalog.work" {
		t.Fatalf("schema hit %s %s", got.method, got.path)
	}
	if got.query != "entity_id=1000" {
		t.Fatalf("schema query = %q, want only the entity id", got.query)
	}
	if got.auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the user's bearer", got.auth)
	}
	if len(schema.Fields) != 1 || schema.Fields[0].Key != "catalog.work.name_zh_cn" ||
		!schema.Fields[0].CanPropose {
		t.Fatalf("projection decoded wrong: %+v", schema)
	}
}

// TestUserEditErrorMapping pins the branches the handler branches on.
func TestUserEditErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"dead token", http.StatusUnauthorized, `{"code":205,"message":"invalid token"}`, ErrUnauthorized},
		{"missing scope", http.StatusForbidden, `{"code":233,"message":"missing required scope: catalog:edit"}`, ErrInsufficientScope},
		{"unknown entity", http.StatusNotFound, `{"code":233,"message":"entity not found"}`, ErrNotFound},
		{"upstream down", http.StatusBadGateway, `nonsense`, ErrUpstream},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := recordingServer(t, tc.status, tc.body)
			_, err := userClient(srv.URL).CreateEditProposalUser(context.Background(), "user-jwt",
				UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000,
					Patch: map[string]any{"catalog.work.name_zh_cn": "x"}})
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

// A policy denial and a validation failure must keep the upstream's own wording:
// they are the reasons a contributor can act on, and collapsing them into a
// sentinel is how an editor stops being able to say what was wrong.
func TestUserEditKeepsActionableReplies(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"policy denial", http.StatusForbidden, `{"code":233,"message":"field is locked by policy"}`},
		{"validation", http.StatusUnprocessableEntity, `{"code":233,"message":"name_zh_cn 不能为空"}`},
		{"rebase conflict", http.StatusConflict, `{"code":233,"message":"proposal is closed"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := recordingServer(t, tc.status, tc.body)
			_, err := userClient(srv.URL).CreateEditProposalUser(context.Background(), "user-jwt",
				UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1000,
					Patch: map[string]any{"catalog.work.name_zh_cn": "x"}})
			if errors.Is(err, ErrInsufficientScope) {
				t.Fatalf("%s must not read as a scope problem", tc.name)
			}
			var apiErr *UserAPIError
			if !errors.As(err, &apiErr) || apiErr.Status != tc.status {
				t.Fatalf("err = %v, want a *UserAPIError %d", err, tc.status)
			}
			if !strings.Contains(apiErr.Message, "policy") && !strings.Contains(apiErr.Message, "不能为空") &&
				!strings.Contains(apiErr.Message, "closed") {
				t.Fatalf("upstream wording lost: %q", apiErr.Message)
			}
		})
	}
}

// No token, no call: an empty access token is the caller's own dead session.
func TestUserEditWithoutTokenNeverCalls(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()
	c := userClient(srv.URL)
	if _, err := c.WithdrawEditProposalUser(context.Background(), "", 7); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if _, err := c.GetEditSchemaUser(context.Background(), "", "catalog.work", 1000); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if called {
		t.Fatal("a tokenless user-plane call must not reach the catalog")
	}
}

func TestUserEditUnconfigured(t *testing.T) {
	c := New(Config{})
	if _, err := c.CreateEditProposalUser(context.Background(), "user-jwt",
		UserEditCreateRequest{EntityType: "catalog.work", EntityID: 1}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
	if _, err := c.GetEditSchemaUser(context.Background(), "user-jwt", "catalog.work", 1); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}
