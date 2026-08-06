package service

// Wave 179 moved the whole claim lifecycle off the asserted-actor S2S face and
// onto the submitter's / moderator's own Bearer token. What that buys is only
// real if the requests actually LOOK like it, and the failure mode is silent in
// both directions: a lane that quietly kept the Basic credential still works
// (as the forum, on behalf of whoever the forum named), and a moderator whose
// console grant has not reached the token would be approved by a stale local
// mirror. So the plane itself is pinned here — path, Authorization header, and
// the absence of the two fields the token now answers — for one owner action,
// one review action, and the "my claims" read.
//
// The error taxonomy is pinned alongside it, because the Bearer plane can
// produce a refusal the S2S face never could: a session minted before the
// `catalog:edit` scope existed. That MUST reach the browser as 235 ("log out
// and back in"), not as 403 "你没有权限" — the user holds the permission; it is
// the token that is too old, and no refresh can widen a grant.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
)

type claimPlaneRecorder struct {
	mu     sync.Mutex
	path   string
	auth   string
	body   map[string]any
	claimQ url.Values

	// status / message, when set, make the claim-action face refuse.
	status  int
	message string
}

// server answers the gid→work bridge and the claim face, recording whichever
// claim call arrives.
func (r *claimPlaneRecorder) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		raw, _ := io.ReadAll(req.Body)

		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(req.URL.Path, "/catalog/lookup/batch"):
			var in struct {
				Items []struct {
					ExternalID string `json:"external_id"`
				} `json:"items"`
			}
			_ = json.Unmarshal(raw, &in)
			out := make([]string, 0, len(in.Items))
			for _, it := range in.Items {
				out = append(out, `{"external_id":"`+it.ExternalID+`","work":null}`)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[` +
				strings.Join(out, ",") + `]}}`))
			return
		case strings.HasSuffix(req.URL.Path, "/catalog/works"):
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[` +
				`{"id":90210,"claimed_by":{"site":"kungal","work_id":90210,"state":"draft"}}` +
				`],"next_cursor":null}}`))
			return
		}

		r.mu.Lock()
		r.path = req.URL.Path
		r.auth = req.Header.Get("Authorization")
		if strings.Contains(req.URL.Path, "/claims/mine") {
			r.claimQ = req.URL.Query()
		} else {
			_ = json.Unmarshal(raw, &r.body)
		}
		status, message := r.status, r.message
		r.mu.Unlock()

		if status != 0 {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"code":233,"message":"` + message + `","data":null}`))
			return
		}
		if strings.Contains(req.URL.Path, "/claims/mine") {
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":null,"next_before":0,"total":0}}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{` +
			`"work_id":90210,"from_state":"draft","to_state":"pending","event_id":7}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (r *claimPlaneRecorder) submissionService(t *testing.T) *SubmissionService {
	srv := r.server(t)
	return NewSubmissionService(
		client.New(srv.URL, "nm_test_key", ""),
		catalogclient.New(catalogclient.Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"}),
		nil,
	)
}

func (r *claimPlaneRecorder) reviewService(t *testing.T) *ClaimReviewService {
	srv := r.server(t)
	return NewClaimReviewService(
		client.New(srv.URL, "nm_test_key", ""),
		catalogclient.New(catalogclient.Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"}),
	)
}

func TestOwnerActionSpeaksAsTheSubmitter(t *testing.T) {
	rec := &claimPlaneRecorder{}
	svc := rec.submissionService(t)

	if _, appErr := svc.Withdraw(t.Context(), "user-jwt", 90210); appErr != nil {
		t.Fatalf("Withdraw: %v", appErr)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.path != "/api/v1/user/catalog/works/90210/claim-actions/withdraw" {
		t.Errorf("withdraw hit %q, want the user plane's claim-action face", rec.path)
	}
	// A Basic credential here would let the service fall back to the S2S
	// posture — the request would succeed and the ownership check would be
	// against nobody.
	if rec.auth != "Bearer user-jwt" {
		t.Errorf("auth = %q, want the submitter's bearer alone", rec.auth)
	}
	if _, ok := rec.body["actor"]; ok {
		t.Errorf("an owner action must assert no actor: %v", rec.body)
	}
	if _, ok := rec.body["site"]; ok {
		t.Errorf("an owner action must assert no site: %v", rec.body)
	}
}

func TestReviewVerdictSpeaksAsTheModerator(t *testing.T) {
	rec := &claimPlaneRecorder{}
	svc := rec.reviewService(t)

	if _, appErr := svc.Review(t.Context(), "mod-jwt", 90210,
		catalogclient.ClaimActionDecline, "资料不足"); appErr != nil {
		t.Fatalf("Review: %v", appErr)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	// The verdict is authorized against catalog.claim.review over THIS token's
	// roles. RequireModerator on the route is only which page opens.
	if rec.path != "/api/v1/user/catalog/works/90210/claim-actions/decline" {
		t.Errorf("decline hit %q, want the user plane's claim-action face", rec.path)
	}
	if rec.auth != "Bearer mod-jwt" {
		t.Errorf("auth = %q, want the moderator's own bearer", rec.auth)
	}
	if _, ok := rec.body["actor"]; ok {
		t.Errorf("a verdict must assert no actor: %v", rec.body)
	}
	// The reason still travels: it is what reaches the submitter.
	if rec.body["reason"] != "资料不足" {
		t.Errorf("reason = %v, want it recorded on the event", rec.body["reason"])
	}
}

func TestListMineIsTheTokensOwnClaims(t *testing.T) {
	rec := &claimPlaneRecorder{}
	svc := rec.submissionService(t)

	page, appErr := svc.ListMine(t.Context(), "user-jwt", url.Values{})
	if appErr != nil {
		t.Fatalf("ListMine: %v", appErr)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.path != "/api/v1/user/catalog/claims/mine" {
		t.Errorf("ListMine hit %q, want the own-claims face", rec.path)
	}
	if rec.auth != "Bearer user-jwt" {
		t.Errorf("auth = %q, want the caller's bearer", rec.auth)
	}
	// No uid in the path and no site in the query: there is nothing left for a
	// caller to get wrong and end up reading somebody else's list.
	if got := rec.claimQ.Get("site"); got != "" {
		t.Errorf("site = %q, want it absent — the tenant rides the token", got)
	}
	// The default filter (everything not yet published) is unchanged by the
	// plane switch.
	if got := rec.claimQ.Get("claim_state"); got != "pending,declined,draft" {
		t.Errorf("claim_state = %q, want the 我的提交 default", got)
	}
	// nil items must still serialize as [] — the page reads data.items.length.
	if page.Items == nil || len(page.Items) != 0 {
		t.Errorf("items = %v, want an empty array", page.Items)
	}
}

// The two refusals only a forwarded token can produce. Both used to be
// impossible on this face, and both would be actively misleading if folded into
// the generic 403 the S2S mapping had.
func TestClaimErrorsCarryTheTokenTaxonomy(t *testing.T) {
	t.Run("a token minted before catalog:edit asks for a re-login", func(t *testing.T) {
		rec := &claimPlaneRecorder{status: http.StatusForbidden, message: "missing required scope: catalog:edit"}
		svc := rec.submissionService(t)

		_, appErr := svc.Resubmit(t.Context(), "old-jwt", 90210)
		if appErr == nil {
			t.Fatal("want a refusal")
		}
		// 233 here would tell a user they may not submit when in fact they may;
		// 205 would log out a perfectly live session.
		if appErr.Code != errors.CodeReauthRequired {
			t.Errorf("code = %d, want %d (re-login prompt)", appErr.Code, errors.CodeReauthRequired)
		}
	})

	t.Run("a dead session is auth-expired, not an upstream fault", func(t *testing.T) {
		rec := &claimPlaneRecorder{status: http.StatusUnauthorized, message: "invalid token"}
		svc := rec.submissionService(t)

		_, appErr := svc.Resubmit(t.Context(), "dead-jwt", 90210)
		if appErr == nil {
			t.Fatal("want a refusal")
		}
		if appErr.Code != errors.CodeAuth || appErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("error = %+v, want the auth-expired envelope", appErr)
		}
	})

	t.Run("a real denial stays a denial", func(t *testing.T) {
		rec := &claimPlaneRecorder{status: http.StatusForbidden, message: "not the owner of this work"}
		svc := rec.reviewService(t)

		_, appErr := svc.Review(t.Context(), "mod-jwt", 90210, catalogclient.ClaimActionApprove, "")
		if appErr == nil {
			t.Fatal("want a refusal")
		}
		if appErr.StatusCode != http.StatusForbidden || appErr.Code == errors.CodeReauthRequired {
			t.Errorf("error = %+v, want a plain 403 — re-logging in would not help", appErr)
		}
	})
}
