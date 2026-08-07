package app

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// A Fiber group created with an EMPTY prefix registers its middleware as Use()
// on the PARENT path, so the middleware runs for every route declared below it
// — not just the group's own. Two authorization gates were written that way
// (`authed.Group("", middleware.RequireAdmin())` and its RequireModerator
// twin), and between 2026-07-21 and 2026-08-07 they swallowed every staff route
// declared after them: the trust inbox, the galgame submission queue, doc,
// website, update-log and friend-link. Moderators holding the permission were
// refused with 403, and no permission-console change could repair it.
//
// The two auth-boundary groups (OptionalAuth / Auth) rely on that same spread
// deliberately — everything below them is meant to carry it — so only the
// Require* gates are forbidden here.
func TestNoAuthorizationGateOnEmptyPrefixGroup(t *testing.T) {
	src, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}

	offender := regexp.MustCompile(`Group\(\s*""\s*,\s*middleware\.Require`)
	for i, line := range strings.Split(string(src), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue // the comments above explain the hazard by name
		}
		if offender.MatchString(line) {
			t.Errorf("router.go:%d puts an authorization gate on an empty-prefix "+
				"group, which leaks it onto every route registered below:\n\t%s\n"+
				"Pass the gate to each route instead: g.Get(path, middleware.RequireX(), handler)",
				i+1, strings.TrimSpace(line))
		}
	}
}
