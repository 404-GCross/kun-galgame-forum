package handler

// Wave 180's half of the migration: the human-serving READS.
//
// A read that kept the S2S credential is the hardest kind of miss to notice —
// it returns the right rows, the page renders, nothing 500s. What it quietly
// does is answer "who is asking" with "the forum", which is exactly what the
// user plane exists to stop. So these tests assert the REQUEST rather than the
// response: every edit-face call a logged-in reader triggers must be the Bearer
// face carrying that reader's own token, with no `site` and no `proposer_uid`
// anywhere on the wire.

import (
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/internal/middleware"
)

// editFaceCalls returns the recorded requests that hit either catalog edit
// face, ignoring the id-bridge traffic (a different server, a different
// question).
func editFaceCalls(f *fakeEditFace) []recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]recordedRequest, 0, len(f.requests))
	for i := range f.requests {
		if f.requests[i].Face != "other" {
			out = append(out, f.requests[i])
		}
	}
	return out
}

// TestEditReadsRideTheUserPlane: bootstrap, the review queue, "my proposals"
// and the proposal workbench each reach the catalog as the caller — no lane
// left on the Basic-authed face, and every one of them carrying the session's
// own token.
func TestEditReadsRideTheUserPlane(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		user       *middleware.UserInfo
	}{
		{"bootstrap", "/api/galgame/1/edit/bootstrap", moderatorUser},
		{"queue", "/api/galgame-edit/queue", moderatorUser},
		{"mine", "/api/galgame-edit/mine", plainUser},
		{"workbench", "/api/galgame-edit/proposals/7", moderatorUser},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeEditFace{}
			app := editTestApp(t, fake.server(t).URL, tc.user)
			if status, raw := doJSON(t, app, "GET", tc.path, ""); status != http.StatusOK {
				t.Fatalf("%s: status = %d body %s", tc.name, status, raw)
			}
			calls := editFaceCalls(fake)
			if len(calls) == 0 {
				t.Fatalf("%s reached no catalog face at all", tc.name)
			}
			for _, r := range calls {
				if r.Face != "user" {
					t.Fatalf("%s still speaks S2S: %s %s", tc.name, r.Method, r.Path)
				}
				if r.Auth != "Bearer user-jwt" {
					t.Fatalf("%s auth = %q, want the session's own bearer", tc.name, r.Auth)
				}
				if strings.Contains(r.Query, "site=") || strings.Contains(r.Query, "proposer_uid=") {
					t.Fatalf("%s names a site or a proposer: %q", tc.name, r.Query)
				}
			}
		})
	}
}

// "Mine" is the token's subject, spelled `mine=true`. The uid this handler used
// to write into the query was its last assertion, and a filter that silently
// went missing would hand every user everybody else's drafts.
func TestEditMineAsksForItsOwn(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, plainUser)
	if status, raw := doJSON(t, app, "GET", "/api/galgame-edit/mine", ""); status != http.StatusOK {
		t.Fatalf("mine: status = %d body %s", status, raw)
	}
	req := fake.callTo("/api/v1/user/catalog/edit/proposals")
	if req == nil {
		t.Fatalf("mine did not reach the user-plane list: %+v", fake.requests)
	}
	if !strings.Contains(req.Query, "mine=true") {
		t.Fatalf("mine query = %q, want mine=true", req.Query)
	}
}

// The queue is the same face WITHOUT the flag: it asks for everybody's
// proposals, and the catalog refuses unless the token carries the review
// permission. A `mine` here would silently turn the moderator queue into the
// moderator's own drafts.
func TestEditQueueAsksForEverybodys(t *testing.T) {
	fake := &fakeEditFace{}
	app := editTestApp(t, fake.server(t).URL, moderatorUser)
	if status, raw := doJSON(t, app, "GET", "/api/galgame-edit/queue", ""); status != http.StatusOK {
		t.Fatalf("queue: status = %d body %s", status, raw)
	}
	req := fake.callTo("/api/v1/user/catalog/edit/proposals")
	if req == nil {
		t.Fatalf("queue did not reach the user-plane list: %+v", fake.requests)
	}
	if strings.Contains(req.Query, "mine") {
		t.Fatalf("queue query = %q, want no mine flag", req.Query)
	}
	if !strings.Contains(req.Query, "status=open") {
		t.Fatalf("queue query = %q, want the status filter", req.Query)
	}
}

// A queue read the catalog refuses because the token lacks the review
// permission must reach the browser as a plain 403 — the local RequireModerator
// is only which nav entry opens, and infra's answer is the one that counts.
func TestEditQueueRelaysTheInfraDenial(t *testing.T) {
	fake := &fakeEditFace{userStatus: http.StatusForbidden,
		userBody: `{"code":233,"message":"permission denied: catalog.edit.review"}`}
	app := editTestApp(t, fake.server(t).URL, moderatorUser)
	status, raw := doJSON(t, app, "GET", "/api/galgame-edit/queue", "")
	if status != http.StatusForbidden {
		t.Fatalf("refused queue: status = %d body %s, want 403", status, raw)
	}
}
