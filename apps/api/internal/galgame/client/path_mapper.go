package client

import "strings"

// galgamePathPrefixes maps frontend resource prefixes (kebab-case, with the
// "galgame-" namespace) to the corresponding galgame path prefix (single word).
var galgamePathPrefixes = []string{
	"/galgame-tag", "/galgame-official",
	"/galgame-engine", "/galgame-series",
	"/galgame-resource",
}

// galgameSuffixMap maps frontend detail sub-route suffixes to their galgame
// equivalent. (The /pr/all and /history/all entries retired in E3b with the
// old-wire reads — only the detail page's links section remains.)
var galgameSuffixMap = map[string]string{
	"/link/all": "/links",
}

// ToGalgamePath converts a gateway request path (e.g. "/api/galgame-tag/foo") to
// the corresponding galgame service path (e.g. "/tag/foo"). It strips the
// leading "/api", rewrites kebab-prefixed resources to their galgame names, and
// translates the frontend detail-suffix convention to the galgame convention.
//
// Taxonomy revisions / revert (U3) deliberately fall through the standard
// prefix rule: galgame keeps these endpoints on the bare entity path
// (`GET /tag/:id/revisions`, `POST /tag/:id/revert`, …) — same shape as
// the existing `/tag/search`, `PUT /tag`, `DELETE /tag/:id` family — so
// the kebab prefix rewrite (/galgame-tag/* → /tag/*) handles them with
// zero special-casing.
//
// Paths that don't match any rule are returned as-is (minus /api).
func ToGalgamePath(path string) string {
	// Strip "/api" prefix.
	wp := path[4:]

	// Resource prefix translation: "/galgame-tag/..." → "/tag/..."
	for _, prefix := range galgamePathPrefixes {
		galgame := "/" + prefix[len("/galgame-"):]
		if strings.HasPrefix(wp, prefix) &&
			(len(wp) == len(prefix) || wp[len(prefix)] == '/') {
			return galgame + wp[len(prefix):]
		}
	}

	// Suffix translation: "/galgame/:gid/pr/all" → "/galgame/:gid/prs"
	for suffix, replacement := range galgameSuffixMap {
		if strings.HasSuffix(wp, suffix) {
			return wp[:len(wp)-len(suffix)] + replacement
		}
	}

	return wp
}
