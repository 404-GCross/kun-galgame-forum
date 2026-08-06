package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/catalogclient"
)

// Filing a submission is the one flow with no id to start from, and how that id
// is resolved is the whole design: kungal names none, the registry mints the
// work and the claim ADOPTS that work's own key, and the response carries it
// back as the gid.
//
// The alternative this replaces — a local sequence allocating ids alongside the
// registry's — is correct only while somebody keeps reseeding the follower, and
// its failure is silent: a collision surfaces as "you already submitted this".
// So both halves are pinned here.

type submitRecorder struct {
	mu   sync.Mutex
	body map[string]any
	// submitPath / submitAuth record WHICH plane the MINT took. Until wave 179
	// it was the asserted-actor S2S face sitting next to a Bearer banner edit —
	// two planes for one gesture, and the only one that let kungal name a
	// submitter who was not the one logged in. Both halves are the user's token
	// now, and a silent fall back to Basic would still work while undoing that.
	submitPath string
	submitAuth string
	editBody   map[string]any
	editPath   string
	editAuth   string
}

func (r *submitRecorder) service(t *testing.T) *SubmissionService {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		raw, _ := io.ReadAll(req.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)

		r.mu.Lock()
		switch {
		case strings.HasSuffix(req.URL.Path, "/catalog/works/submit"):
			r.body = body
			r.submitPath = req.URL.Path
			r.submitAuth = req.Header.Get("Authorization")
		case strings.Contains(req.URL.Path, "/catalog/edit/proposals"):
			r.editBody = body
			r.editPath = req.URL.Path
			r.editAuth = req.Header.Get("Authorization")
		}
		r.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(req.URL.Path, "/catalog/works/submit"):
			// The registry-issued identity: product_work_id ADOPTS work_id.
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{` +
				`"work_id":90210,"product_work_id":90210,` +
				`"claim_state":"pending","event_id":5}}`))
		case strings.HasSuffix(req.URL.Path, "/catalog/lookup/batch"):
			// A freshly minted work has NO external_ref anchor.
			var body struct {
				Items []struct {
					ExternalID string `json:"external_id"`
				} `json:"items"`
			}
			_ = json.Unmarshal(raw, &body)
			out := make([]string, 0, len(body.Items))
			for _, it := range body.Items {
				out = append(out, `{"external_id":"`+it.ExternalID+`","work":null}`)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[` +
				strings.Join(out, ",") + `]}}`))
		case strings.HasSuffix(req.URL.Path, "/catalog/works"):
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[` +
				`{"id":90210,"claimed_by":{"site":"kungal","work_id":90210,"state":"pending"}}` +
				`],"next_cursor":null}}`))
		default:
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"merged":false,"proposal":{"id":1}}}`))
		}
	}))
	t.Cleanup(srv.Close)
	return NewSubmissionService(
		client.New(srv.URL, "nm_test_key", ""),
		catalogclient.New(catalogclient.Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"}),
		nil,
	)
}

func TestSubmitAdoptsTheRegistryIssuedID(t *testing.T) {
	rec := &submitRecorder{}
	svc := rec.service(t)

	res, appErr := svc.Submit(t.Context(), "user-jwt",
		&SubmissionForm{NameJaJP: "白恋サクラ", AgeLimit: "r18", ContentLimit: "nsfw"})
	if appErr != nil {
		t.Fatalf("Submit: %v", appErr)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	// Sending an id here would reintroduce the second allocator. `omitempty` is
	// what makes its absence expressible at all.
	if _, present := rec.body["product_work_id"]; present {
		t.Errorf("request carried product_work_id %v — kungal must name no id", rec.body["product_work_id"])
	}
	// The mint speaks as the user: the tenant and the actor are the token's, so
	// neither may appear in the body.
	if rec.submitPath != "/api/v1/user/catalog/works/submit" {
		t.Errorf("mint hit %q, want the user plane", rec.submitPath)
	}
	if rec.submitAuth != "Bearer user-jwt" {
		t.Errorf("mint auth = %q, want the submitter's bearer and no Basic credential", rec.submitAuth)
	}
	if _, ok := rec.body["site"]; ok {
		t.Errorf("the mint must assert no site: %v", rec.body)
	}
	if _, ok := rec.body["actor"]; ok {
		t.Errorf("the mint must assert no actor: %v", rec.body)
	}
	// The gid is the ADOPTED id off the response, not anything local.
	if res.GID != 90210 {
		t.Errorf("gid = %d, want the registry-issued 90210", res.GID)
	}
	if res.WorkID != 90210 || res.ClaimState != "pending" {
		t.Errorf("result = %+v, want the minted work in pending", res)
	}
}

// The banner cannot ride the mint (a cover REFERENCES bytes that must already
// exist), so it becomes the submission's first edit — against the registry work
// id, which is what the editing engine keys on.
func TestSubmitAttachesTheBannerAsAFollowUpEdit(t *testing.T) {
	rec := &submitRecorder{}
	svc := rec.service(t)

	if _, appErr := svc.Submit(t.Context(), "user-jwt",
		&SubmissionForm{NameJaJP: "x", AgeLimit: "all", ContentLimit: "sfw", BannerHash: "abc123"}); appErr != nil {
		t.Fatalf("Submit: %v", appErr)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.editBody == nil {
		t.Fatal("no follow-up edit filed for the submitted banner")
	}
	if rec.editBody["entity_type"] != "catalog.work" {
		t.Errorf("entity_type = %v, want catalog.work", rec.editBody["entity_type"])
	}
	if rec.editBody["entity_id"] != float64(90210) {
		t.Errorf("entity_id = %v, want the registry work id 90210", rec.editBody["entity_id"])
	}
	// The banner is an ordinary edit, so it takes the ordinary edit plane: the
	// submitter's own token, no asserted actor and no site — the same plane the
	// mint above now takes.
	if rec.editPath != "/api/v1/user/catalog/edit/proposals" {
		t.Errorf("banner edit hit %q, want the user plane", rec.editPath)
	}
	if rec.editAuth != "Bearer user-jwt" {
		t.Errorf("banner edit auth = %q, want the submitter's bearer", rec.editAuth)
	}
	if _, ok := rec.editBody["actor"]; ok {
		t.Errorf("the banner edit must assert no actor: %v", rec.editBody)
	}
	if _, ok := rec.editBody["site"]; ok {
		t.Errorf("the banner edit must assert no site: %v", rec.editBody)
	}
}

// The round trip that the whole switchover rests on: file a submission, take
// the id the registry issued, and reach the same work by it.
//
// It is a separate test because the two halves are answered by DIFFERENT
// mechanisms — the mint adopts the work's own key, and the read has to fall
// through to identity because no anchor was ever written. Either half working
// alone still leaves every new entry 404ing on its own page, silently.
func TestSubmittedEntryIsReachableByItsOwnID(t *testing.T) {
	rec := &submitRecorder{}
	svc := rec.service(t)

	res, appErr := svc.Submit(t.Context(), "user-jwt",
		&SubmissionForm{NameJaJP: "白恋サクラ", AgeLimit: "all", ContentLimit: "sfw"})
	if appErr != nil {
		t.Fatalf("Submit: %v", appErr)
	}

	workID, appErr := svc.workIDOf(t.Context(), res.GID)
	if appErr != nil {
		t.Fatalf("resolving the freshly issued gid %d: %v", res.GID, appErr)
	}
	if workID != res.WorkID {
		t.Errorf("gid %d resolved to work %d, want %d", res.GID, workID, res.WorkID)
	}
}

// A submission with no title is refused before any registry call: the mint
// requires a display name and a 422 round-trip would say the same thing later.
func TestSubmitRefusesATitlelessForm(t *testing.T) {
	rec := &submitRecorder{}
	svc := rec.service(t)

	if _, appErr := svc.Submit(t.Context(), "user-jwt",
		&SubmissionForm{AgeLimit: "all"}); appErr == nil {
		t.Fatal("want a refusal for a form with no title")
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.body != nil {
		t.Error("local validation must not reach the registry")
	}
}
