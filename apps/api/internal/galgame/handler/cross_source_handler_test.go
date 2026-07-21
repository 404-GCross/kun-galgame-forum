package handler

// Route-layer tests for the cross-source read face: they wire the real
// CrossSourceHandler over a bare Fiber app, backed by httptest fakes standing in
// for the wiki (scores/stats) and the catalog (credits/reverse-lookups). They
// assert the three things the BFF passthrough must guarantee:
//   - the wiki / catalog `data` reaches the browser verbatim (no reshape);
//   - the stats ETag + If-None-Match / 304 conditional cache survives the hop;
//   - bad ids are rejected locally and a down catalog degrades (503), not 500.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/pkg/catalogclient"

	"github.com/gofiber/fiber/v3"
)

const wikiScoresData = `{"vndb":{"rating":76,"vote_count":615,"url":"https://vndb.org/v1"},"bangumi":null,"eg":null,"synced_at":"2026-03-02T00:00:00Z"}`
const catalogCreditsData = `{"work_id":3853,"groups":[{"role_id":1,"role_key":"scenario","role_name":"剧本","credits":[{"credit_name_id":42,"name":"丸戸史明","lang":"ja"}]}]}`

const statsETag = `W/"gstats-1752200700"`

// newFakeWiki serves the two wiki read endpoints the handler proxies.
func newFakeWiki(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Empty API key ⇒ the client's read calls fall back to the legacy /api
		// face, so paths arrive prefixed with /api.
		switch r.URL.Path {
		case "/api/galgame/1/scores":
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":` + wikiScoresData + `}`))
		case "/api/galgame/stats":
			// Mirror the wiki's ETag / 304 behaviour exactly.
			w.Header().Set("ETag", statsETag)
			if r.Header.Get("If-None-Match") == statsETag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"release_years":{"payload":[],"built_at":"x"},"built_at":"x"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newFakeCatalog serves the catalog credits endpoint.
func newFakeCatalog(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/catalog/works/3853/credits" {
			if r.Header.Get("Authorization") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":` + catalogCreditsData + `}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// buildApp wires the handler exactly as router.go does, over the given upstreams.
func buildApp(t *testing.T, wikiURL, catalogURL string) *fiber.App {
	t.Helper()
	gc := galgameClient.NewGalgameClient(wikiURL, "") // empty CDN base = banner walker off
	cc := catalogclient.New(catalogclient.Config{BaseURL: catalogURL, ClientID: "cid", ClientSecret: "sec"})
	h := NewCrossSourceHandler(service.NewCrossSourceService(gc, cc))

	app := fiber.New()
	app.Get("/api/galgame/stats", h.Stats)
	app.Get("/api/galgame/:gid/scores", h.Scores)
	app.Get("/api/catalog/works/:wid/credits", h.WorkCredits)
	app.Get("/api/catalog/names/:nid/works", h.NameWorks)
	app.Get("/api/catalog/characters/:cid/works", h.CharacterWorks)
	return app
}

// envelope is the house response wrapper; Data stays raw so we can byte-compare.
type envelope struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

func doGet(t *testing.T, app *fiber.App, path string, headers map[string]string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test %s: %v", path, err)
	}
	return resp
}

func TestScores_PassthroughVerbatim(t *testing.T) {
	app := buildApp(t, newFakeWiki(t).URL, newFakeCatalog(t).URL)
	resp := doGet(t, app, "/api/galgame/1/scores", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("bad envelope: %v (%s)", err, body)
	}
	if env.Code != 0 {
		t.Fatalf("code = %d, want 0", env.Code)
	}
	if !jsonEqual(t, env.Data, wikiScoresData) {
		t.Fatalf("scores data not verbatim:\n got %s\nwant %s", env.Data, wikiScoresData)
	}
}

func TestScores_BadID(t *testing.T) {
	app := buildApp(t, newFakeWiki(t).URL, newFakeCatalog(t).URL)
	if resp := doGet(t, app, "/api/galgame/abc/scores", nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestStats_ETagAnd304Passthrough(t *testing.T) {
	app := buildApp(t, newFakeWiki(t).URL, newFakeCatalog(t).URL)

	// First hit: 200 with the wiki's ETag forwarded + Cache-Control set.
	resp := doGet(t, app, "/api/galgame/stats", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("ETag"); got != statsETag {
		t.Fatalf("ETag = %q, want %q", got, statsETag)
	}
	if resp.Header.Get("Cache-Control") == "" {
		t.Fatal("Cache-Control not set on 200")
	}

	// Conditional re-hit with the same ETag: the 304 must survive to the exit.
	resp2 := doGet(t, app, "/api/galgame/stats", map[string]string{"If-None-Match": statsETag})
	if resp2.StatusCode != http.StatusNotModified {
		t.Fatalf("conditional status = %d, want 304", resp2.StatusCode)
	}
	if got := resp2.Header.Get("ETag"); got != statsETag {
		t.Fatalf("304 ETag = %q, want %q", got, statsETag)
	}
	if body, _ := io.ReadAll(resp2.Body); len(body) != 0 {
		t.Fatalf("304 body must be empty, got %q", body)
	}
}

func TestWorkCredits_PassthroughAndAuth(t *testing.T) {
	app := buildApp(t, newFakeWiki(t).URL, newFakeCatalog(t).URL)
	resp := doGet(t, app, "/api/catalog/works/3853/credits", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var env envelope
	_ = json.Unmarshal(body, &env)
	if !jsonEqual(t, env.Data, catalogCreditsData) {
		t.Fatalf("credits data not verbatim:\n got %s\nwant %s", env.Data, catalogCreditsData)
	}
}

func TestWorkCredits_BadID(t *testing.T) {
	app := buildApp(t, newFakeWiki(t).URL, newFakeCatalog(t).URL)
	if resp := doGet(t, app, "/api/catalog/works/0/credits", nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

// TestCatalog_DegradesWhenDown proves an unreachable catalog degrades to 503
// (graceful) rather than 500-ing the whole page.
func TestCatalog_DegradesWhenDown(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	deadURL := dead.URL
	dead.Close()
	app := buildApp(t, newFakeWiki(t).URL, deadURL)
	if resp := doGet(t, app, "/api/catalog/works/3853/credits", nil); resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
}

func jsonEqual(t *testing.T, got json.RawMessage, want string) bool {
	t.Helper()
	var a, b any
	if err := json.Unmarshal(got, &a); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(want), &b); err != nil {
		t.Fatalf("want is not valid json: %v", err)
	}
	ga, _ := json.Marshal(a)
	gb, _ := json.Marshal(b)
	return string(ga) == string(gb)
}
