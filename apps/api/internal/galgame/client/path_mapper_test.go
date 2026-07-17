package client

import "testing"

func TestToWikiPath(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		// U3 taxonomy revisions / revert — wiki keeps them on the bare
		// entity path (same shape as /tag/search, PUT /tag, DELETE
		// /tag/:id). No namespace special-case: the standard kebab
		// prefix rewrite handles them.
		{"tag revisions list", "/api/galgame-tag/5/revisions", "/tag/5/revisions"},
		{"tag revision single", "/api/galgame-tag/5/revisions/3", "/tag/5/revisions/3"},
		{"tag revert", "/api/galgame-tag/5/revert", "/tag/5/revert"},
		{"official revisions list", "/api/galgame-official/9/revisions", "/official/9/revisions"},
		{"engine revert", "/api/galgame-engine/2/revert", "/engine/2/revert"},
		// Series revisions are now wired too (name/alias/description edits;
		// membership changes still live in galgame-side revisions).
		{"series revisions list", "/api/galgame-series/4/revisions", "/series/4/revisions"},
		{"series revert", "/api/galgame-series/4/revert", "/series/4/revert"},

		// Regression: existing top-level endpoints on the SAME entities
		// must keep the bare single-word path (we only namespace revs).
		{"tag search untouched", "/api/galgame-tag/search", "/tag/search"},
		{"tag detail untouched", "/api/galgame-tag/5", "/tag/5"},
		{"tag PUT shape untouched", "/api/galgame-tag", "/tag"},
		{"official detail untouched", "/api/galgame-official/9", "/official/9"},
		{"series modal untouched", "/api/galgame-series/modal", "/series/modal"},
		{"resource untouched", "/api/galgame-resource/3", "/resource/3"},

		// Regression: galgame's own detail sub-routes are NOT under the
		// /galgame-<entity>/ prefix (different shape) and must keep flowing
		// through the as-is / suffix routes. (The /history/all and /pr/all
		// suffixes retired in E3b with the old-wire reads.)
		{"galgame :gid link/all → /links", "/api/galgame/8/link/all", "/galgame/8/links"},

		// Negative: junk suffixes on taxonomy entities still fall through
		// the namespace rule (only revisions/revert keep the namespace).
		{"tag :id/links would not exist but verify rule scoped",
			"/api/galgame-tag/5/links", "/tag/5/links"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ToWikiPath(c.in)
			if got != c.want {
				t.Errorf("ToWikiPath(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
