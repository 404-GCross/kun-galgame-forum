package client

// Contract tests for the character face. Hermetic: a httptest catalog serves
// the record, so the query the client BUILDS is under test alongside the decode
// — and the query is where this face's two policy decisions live.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// characterStub serves GET /v1/catalog/characters/{id} and records the query it
// was called with. status/body let a test drive the 301 and 404 branches.
func characterStub(t *testing.T, id int64, status int, body string) (*httptest.Server, *url.Values) {
	t.Helper()
	var seen url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/catalog/characters/"+itoa(id) {
			t.Errorf("unexpected upstream call: %s", req.URL.Path)
		}
		seen = req.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &seen
}

// The popup and the page ask for different amounts of the same record, and the
// difference has to reach the wire: the popup never renders the appearance
// list, and asking for it anyway would cost a works-lane hydration per open.
func TestCatalogCharacter_WorksAreIncludeGatedOnTheWire(t *testing.T) {
	srv, seen := characterStub(t, 5, http.StatusOK,
		`{"code":0,"message":"ok","data":{"id":5,"name":{"ja":"朝倉"},"traits":[],"intros":[],"refs":[]}}`)
	c := New(srv.URL, "nm_test_key", "")

	if _, _, _, appErr := c.CatalogCharacterDetail(context.Background(), 5, 50, 0, false); appErr != nil {
		t.Fatalf("CatalogCharacterDetail: %v", appErr)
	}
	if got := seen.Get("include"); got != "" {
		t.Errorf("include = %q with withWorks=false, want it absent", got)
	}
	// Spoiler traits must ride along even on the popup's light fetch: the
	// reveal is a click, not a second request.
	if got := seen.Get("spoilers"); got != "2" {
		t.Errorf("spoilers = %q, want the full ceiling 2", got)
	}
	// r18 is 94.5% of the registry — a closed gate would delete most of a
	// character's appearances rather than filter them.
	if got := seen.Get("nsfw"); got != "1" {
		t.Errorf("nsfw = %q, want the population open", got)
	}

	if _, _, _, appErr := c.CatalogCharacterDetail(context.Background(), 5, 50, 0, true); appErr != nil {
		t.Fatalf("CatalogCharacterDetail(withWorks): %v", appErr)
	}
	if got := seen.Get("include"); got != "works" {
		t.Errorf("include = %q with withWorks=true, want works", got)
	}
}

// Bust and 立绘 are different assets, not two sizes of one, and each has to
// arrive in its own field for the renderer to give it the right fit. Collapsing
// them is the obvious simplification and also the bug: a figure cropped into a
// portrait box is a picture of someone's midriff.
func TestCatalogCharacter_BothArtsSurviveInTheirOwnFields(t *testing.T) {
	srv, _ := characterStub(t, 7, http.StatusOK, `{"code":0,"message":"ok","data":{
		"id":7,"name":{"ja":"雪村杏"},"latin":"Yukimura Anzu",
		"image":"https://cdn.test/aa/bb/bust.webp",
		"figure":"https://cdn.test/cc/dd/figure.webp",
		"traits":[{"id":1,"name":"Blonde","group":"Hair","spoiler":0,"sexual":false,"lie":false},
		          {"id":2,"name":"Dead","group":"Role","spoiler":2,"sexual":false,"lie":true}],
		"intros":[{"lang":"zh-Hans","intro":"主人公的青梅竹马","source":"bangumi","machine":true}],
		"refs":[{"source":"vndb","external_id":"c1234"}],
		"works":[{"work":{"id":900,"display_name":"テスト","medium":"game","content_rating":"r18"},
		          "voices":[{"id":11,"name":"茶木ひかる","lang":"ja"}]}],
		"next_offset":50}}`)
	c := New(srv.URL, "nm_test_key", "")

	ch, found, movedTo, appErr := c.CatalogCharacterDetail(context.Background(), 7, 50, 0, true)
	if appErr != nil || !found || movedTo != 0 {
		t.Fatalf("CatalogCharacterDetail = (%v, %v, %d)", appErr, found, movedTo)
	}
	if ch.Image != "https://cdn.test/aa/bb/bust.webp" {
		t.Errorf("Image = %q", ch.Image)
	}
	if ch.Figure != "https://cdn.test/cc/dd/figure.webp" {
		t.Errorf("Figure = %q", ch.Figure)
	}
	// A major-spoiler trait decodes rather than being dropped upstream — the
	// whole point of asking for the ceiling.
	if len(ch.Traits) != 2 || ch.Traits[1].Spoiler != 2 || !ch.Traits[1].Lie {
		t.Errorf("Traits = %+v, want the spoiler+lie row intact", ch.Traits)
	}
	if len(ch.Intros) != 1 || !ch.Intros[0].Machine {
		t.Errorf("Intros = %+v, want the machine flag carried", ch.Intros)
	}
	if len(ch.Works) != 1 || len(ch.Works[0].Voices) != 1 || ch.Works[0].Voices[0].ID != 11 {
		t.Errorf("Works = %+v, want the per-work voice name", ch.Works)
	}
	if ch.NextOffset == nil || *ch.NextOffset != 50 {
		t.Errorf("NextOffset = %v, want 50", ch.NextOffset)
	}
}

// A merged character keeps her old id addressable, but only as a 301: the
// survivor's record must never be painted under the dead id, so the caller gets
// an id to redirect to and nothing else.
func TestCatalogCharacter_MergedIDRedirectsRatherThanRenders(t *testing.T) {
	srv, _ := characterStub(t, 3, http.StatusMovedPermanently,
		`{"code":12,"message":"moved","data":{"entity_type":"character","id":3,"current_id":91}}`)
	c := New(srv.URL, "nm_test_key", "")

	ch, found, movedTo, appErr := c.CatalogCharacterDetail(context.Background(), 3, 50, 0, true)
	if appErr != nil {
		t.Fatalf("CatalogCharacterDetail: %v", appErr)
	}
	if found || ch != nil {
		t.Errorf("found = %v, record = %+v, want neither under a dead id", found, ch)
	}
	if movedTo != 91 {
		t.Errorf("movedTo = %d, want 91", movedTo)
	}
}

// A 404 is a miss, not an error: the page renders 未找到该角色 rather than a 500.
func TestCatalogCharacter_UnknownIDIsAMiss(t *testing.T) {
	srv, _ := characterStub(t, 4, http.StatusNotFound, `{"code":404,"message":"not found","data":null}`)
	c := New(srv.URL, "nm_test_key", "")

	_, found, movedTo, appErr := c.CatalogCharacterDetail(context.Background(), 4, 50, 0, true)
	if appErr != nil {
		t.Fatalf("CatalogCharacterDetail: %v", appErr)
	}
	if found || movedTo != 0 {
		t.Errorf("found = %v, movedTo = %d, want a clean miss", found, movedTo)
	}
}
