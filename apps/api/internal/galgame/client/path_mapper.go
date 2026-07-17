package client

import "strings"

// wikiPathPrefixes maps frontend resource prefixes (kebab-case, with the
// "galgame-" namespace) to the corresponding wiki path prefix (single word).
var wikiPathPrefixes = []string{
	"/galgame-tag", "/galgame-official",
	"/galgame-engine", "/galgame-series",
	"/galgame-resource",
}

// wikiSuffixMap maps frontend detail sub-route suffixes to their wiki
// equivalent. (The /pr/all and /history/all entries retired in E3b with the
// old-wire reads — only the detail page's links section remains.)
var wikiSuffixMap = map[string]string{
	"/link/all": "/links",
}

// ToWikiPath converts a gateway request path (e.g. "/api/galgame-tag/foo") to
// the corresponding wiki service path (e.g. "/tag/foo"). It strips the
// leading "/api", rewrites kebab-prefixed resources to their wiki names, and
// translates the frontend detail-suffix convention to the wiki convention.
//
// Taxonomy revisions / revert (U3) deliberately fall through the standard
// prefix rule: wiki keeps these endpoints on the bare entity path
// (`GET /tag/:id/revisions`, `POST /tag/:id/revert`, …) — same shape as
// the existing `/tag/search`, `PUT /tag`, `DELETE /tag/:id` family — so
// the kebab prefix rewrite (/galgame-tag/* → /tag/*) handles them with
// zero special-casing.
//
// Paths that don't match any rule are returned as-is (minus /api).
func ToWikiPath(path string) string {
	// Strip "/api" prefix.
	wp := path[4:]

	// Resource prefix translation: "/galgame-tag/..." → "/tag/..."
	for _, prefix := range wikiPathPrefixes {
		wiki := "/" + prefix[len("/galgame-"):]
		if strings.HasPrefix(wp, prefix) &&
			(len(wp) == len(prefix) || wp[len(prefix)] == '/') {
			return wiki + wp[len(prefix):]
		}
	}

	// Suffix translation: "/galgame/:gid/pr/all" → "/galgame/:gid/prs"
	for suffix, replacement := range wikiSuffixMap {
		if strings.HasSuffix(wp, suffix) {
			return wp[:len(wp)-len(suffix)] + replacement
		}
	}

	return wp
}
