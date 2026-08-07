package app

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
	"unsafe"

	"kun-galgame-api/pkg/config"

	"github.com/gofiber/fiber/v3"
)

// The route table is the ONLY place where a middleware's blast radius is
// visible. Reading router.go top-to-bottom does not show it: Fiber resolves a
// group's middleware against its PATH, so `parent.Group("", mw)` — an empty
// prefix — becomes `Use("/api", mw)` and silently attaches mw to every route
// registered after it, group handle notwithstanding. That is how a
// RequireAdmin gate meant for five routes ended up on ninety.
//
// So we build the real router and read the real chains back out. Handler
// identity comes from the function pointer, which for a closure returned by a
// constructor is named after the constructor ("middleware.Auth.func1") — i.e.
// the chain prints as the middleware names actually applied.
//
// Every handler field on App is a POINTER, and route registration only takes
// method VALUES (never calls them), so a zero App registers the whole table
// without a database, Redis or OAuth.

var updateGolden = flag.Bool("update-routes", false, "rewrite testdata/routes.golden")

const goldenPath = "testdata/routes.golden"

func buildManifest(t *testing.T) []string {
	t.Helper()

	lines := make([]string, 0, 256)
	for _, r := range resolveRoutes(t) {
		lines = append(lines, fmt.Sprintf("%-6s %-46s %s",
			r.Method, r.Path, strings.Join(r.chain, " → ")))
	}
	sort.Strings(lines)
	return lines
}

// route pairs a registered route with the chain a request to it ACTUALLY
// runs. Fiber does not store the inherited half on the route — it walks the
// per-method stack at match time and runs every Use() entry it passes whose
// prefix matches — which is precisely why a leak is invisible in both the
// source and a naive dump of Route.Handlers.
type route struct {
	fiber.Route
	chain []string
}

// resolveRoutes replays Fiber's own matching to fold each route's inherited
// middleware into it. GetRoutes walks app.stack method by method in
// REGISTRATION order, so a Use entry affects exactly the routes that come
// after it in the same method's stack — the ordering the whole hazard turns on.
func resolveRoutes(t *testing.T) []route {
	t.Helper()

	a := &App{Fiber: fiber.New(), Config: testConfig()}
	a.setupRoutes()

	var (
		out     []route
		pending []fiber.Route // Use entries seen so far, this method's stack
		method  string
	)
	for _, r := range a.Fiber.GetRoutes() {
		if r.Method != method { // GetRoutes emits one method's stack at a time
			method, pending = r.Method, nil
		}
		if isUse(r) {
			pending = append(pending, r)
			continue
		}
		// HEAD is Fiber's automatic mirror of GET; one entry per real route.
		if r.Method == fiber.MethodHead {
			continue
		}
		chain := make([]string, 0, len(r.Handlers)+2)
		for _, u := range pending {
			if u.Path == "/" || r.Path == u.Path || strings.HasPrefix(r.Path, u.Path+"/") {
				for _, h := range u.Handlers {
					chain = append(chain, handlerName(h))
				}
			}
		}
		for _, h := range r.Handlers {
			chain = append(chain, handlerName(h))
		}
		out = append(out, route{Route: r, chain: chain})
	}
	if len(out) == 0 {
		t.Fatal("no routes registered")
	}
	return out
}

// isUse reports whether a route is a mounted middleware rather than an
// endpoint. Fiber keeps the flag unexported and offers no accessor (GetRoutes'
// filterUse option can only drop them, and we need to know WHERE they sit), so
// the test reads the field directly. Pinned to fiber v3.3.0; if the field is
// ever renamed this fails loudly instead of silently reporting empty chains.
func isUse(r fiber.Route) bool {
	v := reflect.ValueOf(&r).Elem().FieldByName("use")
	if !v.IsValid() {
		panic("fiber.Route has no `use` field — update this test for the new Fiber")
	}
	return *(*bool)(unsafe.Pointer(v.UnsafeAddr()))
}

// handlerName renders a handler as "Auth", "RequirePermission" or
// "UpdateHandler.CreateTodo" — the package path and Go's closure/method-value
// suffixes carry no information here and only make the golden file noisy.
func handlerName(h fiber.Handler) string {
	full := runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	if i := strings.LastIndex(full, "/"); i >= 0 {
		full = full[i+1:]
	}
	full = strings.TrimSuffix(full, "-fm")
	// "middleware.RequireAdmin.func1" → "RequireAdmin"
	if i := strings.Index(full, ".func"); i >= 0 {
		full = full[:i]
	}
	full = strings.NewReplacer("(", "", ")", "", "*", "").Replace(full)
	parts := strings.Split(full, ".")
	last := parts[len(parts)-1]
	// A middleware constructor whose closure got inlined into its caller is
	// named after the CALL SITE ("app.App.setupRoutes.RequirePermission"), not
	// the defining package — so key off the constructor name, which is the last
	// segment either way.
	if strings.HasPrefix(last, "Require") || last == "Auth" || last == "OptionalAuth" {
		return last
	}
	if len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}
	return strings.Join(parts, ".")
}

// TestRouteManifest pins the full route table WITH its middleware chains. A
// new route adds one line; a middleware that escapes its group rewrites
// dozens — which is exactly the diff that must not pass review unnoticed.
// Regenerate deliberately: `go test ./internal/app/ -run TestRouteManifest -update-routes`.
func TestRouteManifest(t *testing.T) {
	got := strings.Join(buildManifest(t), "\n") + "\n"

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Log("golden rewritten — review the diff before committing")
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read %s: %v (regenerate with -update-routes)", goldenPath, err)
	}
	if string(want) == got {
		return
	}

	wantLines, gotLines := strings.Split(string(want), "\n"), strings.Split(got, "\n")
	t.Errorf("route table changed (%d → %d routes). If intended, regenerate:\n"+
		"  go test ./internal/app/ -run TestRouteManifest -update-routes\n"+
		"and READ the diff — a middleware leaking out of its group shows up as a\n"+
		"changed chain on routes you never touched.", len(wantLines)-1, len(gotLines)-1)
	for _, d := range diffLines(wantLines, gotLines) {
		t.Error(d)
	}
}

func diffLines(want, got []string) []string {
	inWant := make(map[string]bool, len(want))
	for _, l := range want {
		inWant[l] = true
	}
	inGot := make(map[string]bool, len(got))
	for _, l := range got {
		inGot[l] = true
	}
	var out []string
	for _, l := range want {
		if l != "" && !inGot[l] {
			out = append(out, "  - "+l)
		}
	}
	for _, l := range got {
		if l != "" && !inWant[l] {
			out = append(out, "  + "+l)
		}
	}
	if len(out) > 40 {
		out = append(out[:40], fmt.Sprintf("  … and %d more", len(out)-40))
	}
	return out
}

// TestNoRouteCarriesTwoAuthorizationGates is the churn-free half of the guard:
// it needs no golden file and fails on the exact shape of the 2026-07 outage.
// A gate that escapes its group lands on routes that already carry their own,
// so a second Require* in one chain means one of them was never meant to be
// there. (A route legitimately needing two different capabilities should say
// so in ONE gate, not stack two — stacked gates AND together silently.)
func TestNoRouteCarriesTwoAuthorizationGates(t *testing.T) {
	for _, r := range resolveRoutes(t) {
		var gates []string
		for _, n := range r.chain {
			if strings.HasPrefix(n, "Require") {
				gates = append(gates, n)
			}
		}
		if len(gates) > 1 {
			t.Errorf("%s %s carries %d authorization gates (%s) — one of them "+
				"leaked from an empty-prefix group above it in router.go",
				r.Method, r.Path, len(gates), strings.Join(gates, ", "))
		}
	}
}

// TestAuthorizationSitsBehindAuthentication: a Require* gate reads the caller
// from the request locals, so a gate in front of an unauthenticated route
// judges a caller nobody established. Every gate must sit behind Auth.
func TestAuthorizationSitsBehindAuthentication(t *testing.T) {
	for _, r := range resolveRoutes(t) {
		gated, authed := false, false
		for _, n := range r.chain {
			switch {
			case strings.HasPrefix(n, "Require"):
				gated = true
			case n == "Auth":
				authed = true
			}
		}
		if gated && !authed {
			t.Errorf("%s %s is authorization-gated but not authenticated: %s",
				r.Method, r.Path, strings.Join(r.chain, " → "))
		}
	}
}

// publicWrites are the only mutating endpoints allowed to run without a
// session, each authenticated by something other than the cookie.
var publicWrites = map[string]string{
	"POST /api/trust/callback":      "HMAC X-Trust-Signature over the raw body",
	"POST /api/auth/oauth/callback": "the OAuth authorization code itself",
	"POST /api/auth/logout":         "destroys a session; nothing to protect",
}

// TestEveryWriteIsAuthenticated is the invariant the boundary exists to
// uphold, checked against the resolved table rather than against where the
// line happens to sit in the file. A write registered above the boundary —
// the mirror-image of the 2026-07 leak, and the dangerous direction — fails
// here even though it looks perfectly ordinary in router.go.
func TestEveryWriteIsAuthenticated(t *testing.T) {
	for _, r := range resolveRoutes(t) {
		if r.Method == fiber.MethodGet || !strings.HasPrefix(r.Path, "/api") {
			continue
		}
		if _, ok := publicWrites[r.Method+" "+r.Path]; ok {
			continue
		}
		if !slices.Contains(r.chain, "Auth") {
			t.Errorf("%s %s mutates without authentication: %s\n"+
				"If that is deliberate, add it to publicWrites with the reason.",
				r.Method, r.Path, strings.Join(r.chain, " → "))
		}
	}
}

// testConfig supplies the only two settings setupRoutes reads: the CORS
// origin list (the middleware panics on an empty one) and the server mode.
func testConfig() *config.Config {
	cfg := &config.Config{}
	cfg.CORS.AllowOrigins = "https://www.kungal.com"
	cfg.Server.Mode = "prod"
	return cfg
}

// TestNoRouteIsShadowed guards the OTHER order-sensitive thing about Fiber:
// it matches routes in REGISTRATION order, not by specificity. A param route
// registered first swallows every later static route it can match, so
// `/topic/:tid` before `/topic/draft` makes the draft endpoint unreachable —
// it answers with the topic handler and a tid of "draft". router.go is full of
// comments pleading with future readers to preserve the order; this checks it.
func TestNoRouteIsShadowed(t *testing.T) {
	byMethod := map[string][]route{}
	for _, r := range resolveRoutes(t) {
		byMethod[r.Method] = append(byMethod[r.Method], r)
	}
	for method, routes := range byMethod {
		for i, later := range routes {
			for _, earlier := range routes[:i] {
				if earlier.Path == later.Path {
					t.Errorf("%s %s is registered twice", method, later.Path)
					continue
				}
				if patternMatches(t, earlier.Path, later.Path) {
					t.Errorf("%s %s is unreachable: %s was registered first and "+
						"matches it (Fiber resolves in registration order, not by "+
						"specificity). Move the static route above the param one.",
						method, later.Path, earlier.Path)
				}
			}
		}
	}
}

// patternMatches reports whether the route pattern `pat` would capture a
// request for the pattern `path`. A param segment in `path` is a literal that
// varies, so it can only be captured by a param segment in `pat`.
func patternMatches(t *testing.T, pat, path string) bool {
	p, q := strings.Split(pat, "/"), strings.Split(path, "/")
	if len(p) != len(q) {
		return false // no wildcards in this router, so length must agree
	}
	for i := range p {
		if strings.HasSuffix(p[i], "?") || strings.Contains(p[i], "*") {
			t.Fatalf("route %q uses an optional/wildcard segment; this check "+
				"assumes neither exists in the router", pat)
		}
		if strings.HasPrefix(p[i], ":") {
			continue // a param segment captures anything, literal or not
		}
		if p[i] != q[i] {
			return false
		}
	}
	return true
}
