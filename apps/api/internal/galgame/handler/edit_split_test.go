package handler

// P6 split-routing tests (09-open-api-phase2 06b W2): the editing-engine BFF
// fans the galgame proposal chain across two faces —
//   - the platform propose face (/internal/edit/*, dual-credential X-API-Key +
//     Bearer, server-derived actor) for the engine-policy-irrelevant subset
//     (mine / withdraw / snapshot) for EVERYONE, and submit / schema for PLAIN
//     actors; and
//   - the S2S actor-assertion face (/api/v1/catalog/edit/*) for staff/owner
//     writes and the whole review chain.
// These pin the routing predicate, that no assertion field ever rides the
// platform face (G7), and that withdraw drops its S2S pre-flight.

import (
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/internal/middleware"
)

// plainNonOwner is a logged-in user who created nothing — the plain-actor path.
var plainNonOwner = &middleware.UserInfo{ID: 8, Name: "bystander", Roles: nil}

// platformReqs / s2sReqs partition the recorded calls by face.
func (f *fakeEditFace) byFace(face string) []recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []recordedRequest
	for _, r := range f.requests {
		if r.Face == face {
			out = append(out, r)
		}
	}
	return out
}

// assertNoAssertionFields is the G7 gate: a platform-face request must carry no
// asserted actor/site anywhere — not in the JSON body, not in the query string.
func assertNoAssertionFields(t *testing.T, r recordedRequest) {
	t.Helper()
	if r.Face != "platform" {
		t.Fatalf("expected a platform request, got %s %s (%s)", r.Method, r.Path, r.Face)
	}
	if r.APIKey == "" || !strings.HasPrefix(r.Auth, "Bearer ") {
		t.Fatalf("platform request missing dual credentials: key=%q auth=%q", r.APIKey, r.Auth)
	}
	for _, k := range []string{"actor", "site", "proposer_uid", "roles", "trust_tier", "is_entity_owner", "user_id"} {
		if _, ok := r.Body[k]; ok {
			t.Fatalf("platform body carries asserted field %q: %v", k, r.Body)
		}
	}
	for _, k := range []string{"proposer_uid", "site=", "roles=", "trust_tier=", "is_entity_owner=", "user_id="} {
		if strings.Contains(r.Query, k) {
			t.Fatalf("platform query carries asserted field %q: %q", k, r.Query)
		}
	}
}

// TestEditSubmitSplit: a plain proposer files through the platform face (no actor
// asserted); the game's owner and staff stay on the S2S actor-assertion face so
// their direct-edit automerge is preserved.
func TestEditSubmitSplit(t *testing.T) {
	const body = `{"patch":{"galgame.game.name_zh_cn":"新标题"},"note":"typo"}`

	// Plain actor (no galgame wired → owner lookup is false) → platform.
	t.Run("plain→platform", func(t *testing.T) {
		fake := &fakeEditFace{}
		app := editTestApp(t, fake.server(t).URL, plainNonOwner)
		if status, raw := doJSON(t, app, "POST", "/api/galgame/1/edit/proposals", body); status != http.StatusOK {
			t.Fatalf("plain submit: status = %d body %s", status, raw)
		}
		if len(fake.byFace("s2s")) != 0 {
			t.Fatalf("plain submit must not touch the S2S face: %+v", fake.requests)
		}
		plat := fake.byFace("platform")
		if len(plat) != 1 || plat[0].Method != "POST" || plat[0].Path != "/internal/edit/proposals" {
			t.Fatalf("plain submit must be one platform create, got %+v", plat)
		}
		assertNoAssertionFields(t, plat[0])
		if plat[0].Body["entity_type"] != "galgame.game" || plat[0].Body["patch"] == nil {
			t.Fatalf("platform create body missing entity_type/patch: %v", plat[0].Body)
		}
	})

	// The game's owner (uid 7 = the fake galgame's creator) → S2S with ownership.
	t.Run("owner→S2S", func(t *testing.T) {
		fake := &fakeEditFace{}
		nm := fakeGalgame(t)
		app := editTestAppFull(t, fake.server(t).URL, nm.URL, plainUser, nil) // uid 7
		if status, raw := doJSON(t, app, "POST", "/api/galgame/1/edit/proposals", body); status != http.StatusOK {
			t.Fatalf("owner submit: status = %d body %s", status, raw)
		}
		if len(fake.byFace("platform")) != 0 {
			t.Fatalf("owner submit must not touch the platform face: %+v", fake.requests)
		}
		s2s := fake.byFace("s2s")
		if len(s2s) != 1 || s2s[0].Path != "/api/v1/catalog/edit/proposals" {
			t.Fatalf("owner submit must be one S2S create, got %+v", s2s)
		}
		actor, _ := s2s[0].Body["actor"].(map[string]any)
		if actor == nil || actor["is_entity_owner"] != true {
			t.Fatalf("owner submit must assert is_entity_owner: %v", s2s[0].Body)
		}
	})

	// Staff (moderator, not the owner) → S2S with the verbatim role set.
	t.Run("staff→S2S", func(t *testing.T) {
		fake := &fakeEditFace{}
		nm := fakeGalgame(t)
		app := editTestAppFull(t, fake.server(t).URL, nm.URL, moderatorUser, nil) // uid 42
		if status, raw := doJSON(t, app, "POST", "/api/galgame/1/edit/proposals", body); status != http.StatusOK {
			t.Fatalf("staff submit: status = %d body %s", status, raw)
		}
		if len(fake.byFace("platform")) != 0 {
			t.Fatalf("staff submit must not touch the platform face: %+v", fake.requests)
		}
		s2s := fake.byFace("s2s")
		if len(s2s) != 1 {
			t.Fatalf("staff submit must be one S2S create, got %+v", s2s)
		}
		actor, _ := s2s[0].Body["actor"].(map[string]any)
		roles, _ := actor["roles"].([]any)
		if len(roles) != 1 || roles[0] != "moderator" {
			t.Fatalf("staff submit must assert roles verbatim: %v", s2s[0].Body)
		}
	})
}

// TestEditMineViaPlatform: "my proposals" routes to the platform for everyone,
// and the BFF forwards neither proposer_uid nor site (the platform forces both).
func TestEditMineViaPlatform(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	if status, raw := doJSON(t, app, "GET", "/api/galgame-edit/mine?gid=1&status=open", ""); status != http.StatusOK {
		t.Fatalf("mine: status = %d body %s", status, raw)
	}
	if len(fake.byFace("s2s")) != 0 {
		t.Fatalf("mine must not touch the S2S face: %+v", fake.requests)
	}
	plat := fake.byFace("platform")
	if len(plat) != 1 || plat[0].Method != "GET" || plat[0].Path != "/internal/edit/proposals" {
		t.Fatalf("mine must be one platform list, got %+v", plat)
	}
	assertNoAssertionFields(t, plat[0])
	if !strings.Contains(plat[0].Query, "entity_type=galgame.game") ||
		!strings.Contains(plat[0].Query, "entity_id=1") ||
		!strings.Contains(plat[0].Query, "status=open") {
		t.Fatalf("mine query must forward entity/status narrowing only: %q", plat[0].Query)
	}
}

// TestEditWithdrawViaPlatform: withdraw routes to the platform for everyone and
// makes exactly ONE call — the platform withdraw op subsumes the old S2S
// tenant/family pre-flight read (no S2S residual, G7).
func TestEditWithdrawViaPlatform(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	status, raw := doJSON(t, app, "POST", "/api/galgame-edit/proposals/7/withdraw", "")
	if status != http.StatusOK {
		t.Fatalf("withdraw: status = %d body %s", status, raw)
	}
	if len(fake.byFace("s2s")) != 0 {
		t.Fatalf("withdraw must make no S2S pre-flight, got %+v", fake.byFace("s2s"))
	}
	plat := fake.byFace("platform")
	if len(plat) != 1 || plat[0].Method != "POST" || plat[0].Path != "/internal/edit/proposals/7/withdraw" {
		t.Fatalf("withdraw must be one platform call, got %+v", plat)
	}
	assertNoAssertionFields(t, plat[0])
}

// TestEditBootstrapSchemaSplit: bootstrap's snapshot is always the platform face;
// its schema follows the submit split — plain → platform, staff → S2S (so the
// reviewer's asserted capability projection survives).
func TestEditBootstrapSchemaSplit(t *testing.T) {
	// Plain (no galgame wired → non-owner): snapshot + schema both platform.
	t.Run("plain→platform schema", func(t *testing.T) {
		fake := &fakeEditFace{}
		app := editTestApp(t, fake.server(t).URL, plainNonOwner)
		if status, raw := doJSON(t, app, "GET", "/api/galgame/1/edit/bootstrap", ""); status != http.StatusOK {
			t.Fatalf("plain bootstrap: status = %d body %s", status, raw)
		}
		if len(fake.byFace("s2s")) != 0 {
			t.Fatalf("plain bootstrap must not touch the S2S face: %+v", fake.requests)
		}
		var sawSnapshot, sawSchema bool
		for _, r := range fake.byFace("platform") {
			assertNoAssertionFields(t, r)
			switch r.Path {
			case "/internal/edit/snapshot":
				sawSnapshot = true
			case "/internal/edit/schema/galgame.game":
				sawSchema = true
			}
		}
		if !sawSnapshot || !sawSchema {
			t.Fatalf("plain bootstrap must read snapshot+schema on the platform: %+v", fake.requests)
		}
	})

	// Staff: snapshot on the platform, schema on S2S with the asserted overlay.
	t.Run("staff→S2S schema", func(t *testing.T) {
		fake := &fakeEditFace{}
		nm := fakeGalgame(t)
		app := editTestAppFull(t, fake.server(t).URL, nm.URL, moderatorUser, nil)
		if status, raw := doJSON(t, app, "GET", "/api/galgame/1/edit/bootstrap", ""); status != http.StatusOK {
			t.Fatalf("staff bootstrap: status = %d body %s", status, raw)
		}
		snap := fake.byFace("platform")
		if len(snap) != 1 || snap[0].Path != "/internal/edit/snapshot" {
			t.Fatalf("staff bootstrap snapshot must ride the platform: %+v", fake.requests)
		}
		var schemaQuery string
		for _, r := range fake.byFace("s2s") {
			if strings.HasPrefix(r.Path, "/api/v1/catalog/edit/schema/") {
				schemaQuery = r.Query
			}
		}
		if schemaQuery == "" || !strings.Contains(schemaQuery, "trust_tier=3") {
			t.Fatalf("staff bootstrap schema must ride S2S with the staff assertion: %q", schemaQuery)
		}
	})
}
