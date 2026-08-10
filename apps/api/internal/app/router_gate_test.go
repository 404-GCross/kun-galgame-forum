package app

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// A Fiber group created with an EMPTY prefix registers its middleware as Use()
// on the PARENT path, so it runs for every route declared below it, not just
// the group's own. Written that way, `authed.Group("", middleware.RequireAdmin())`
// and its RequireModerator twin swallowed every staff route declared after them
// between 2026-07-21 and 2026-08-07: the trust inbox, the galgame submission
// queue, doc, website, update-log and friend-link. Moderators holding the
// permission were refused with 403 and no permission-console change could
// repair it.
//
// The two auth-boundary groups (OptionalAuth / Auth) rely on that same spread
// deliberately, so only the Require* gates are forbidden here.
func TestNoAuthorizationGateOnEmptyPrefixGroup(t *testing.T) {
	src, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}

	offender := regexp.MustCompile(`Group\(\s*""\s*,\s*middleware\.Require`)
	for i, line := range strings.Split(string(src), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		if offender.MatchString(line) {
			t.Errorf("router.go:%d puts an authorization gate on an empty-prefix "+
				"group, which leaks it onto every route registered below:\n\t%s\n"+
				"Pass the gate to each route instead: g.Get(path, middleware.RequireX(), handler)",
				i+1, strings.TrimSpace(line))
		}
	}
}
