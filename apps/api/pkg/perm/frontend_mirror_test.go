package perm

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The web app carries this vocabulary a second time by hand: useCan.ts gates the
// UI, permission.ts labels the /admin/permission matrix. Nothing else connects
// the two copies — a key added here and forgotten there stays invisible until a
// moderator finds a button that never appears, or the matrix silently omits a
// row nobody can grant.
const (
	useCanPath     = "../../../web/app/composables/useCan.ts"
	permMetaPath   = "../../../web/app/constants/permission.ts"
	moderatorArray = "MODERATOR_PERMISSIONS"
	adminOnlyArray = "ADMIN_ONLY_PERMISSIONS"
	metaObject     = "KUN_PERMISSION_META"
)

func TestFrontendMirrorsTheVocabulary(t *testing.T) {
	useCan := readMirror(t, useCanPath)
	meta := readMirror(t, permMetaPath)

	assertSameKeys(t, "useCan.ts "+moderatorArray, Bundles["moderator"],
		keysIn(t, useCan, "const "+moderatorArray+" = [", "]"))
	assertSameKeys(t, "useCan.ts "+adminOnlyArray, adminOnlyPerms(),
		keysIn(t, useCan, "const "+adminOnlyArray+" = [", "]"))
	assertSameKeys(t, "permission.ts "+metaObject, Catalog(),
		keysIn(t, meta, "const "+metaObject+": Record<ForumPermission, KunPermissionMeta> = {", "\n}"))
}

func adminOnlyPerms() []Permission {
	mod := set(Bundles["moderator"])
	out := make([]Permission, 0, 4)
	for _, p := range Bundles["admin"] {
		if !mod[p] {
			out = append(out, p)
		}
	}
	return out
}

func readMirror(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v — the web mirror moved; point this test at it "+
			"rather than deleting the check", path, err)
	}
	return string(raw)
}

var keyRe = regexp.MustCompile(`'([a-z][a-z0-9_.]*)'`)

func keysIn(t *testing.T, src, open, close string) []Permission {
	t.Helper()
	start := strings.Index(src, open)
	if start < 0 {
		t.Fatalf("no %q block in the web mirror — it was restructured", open)
	}
	start += len(open)
	end := strings.Index(src[start:], close)
	if end < 0 {
		t.Fatalf("unterminated %q block in the web mirror", open)
	}
	found := keyRe.FindAllStringSubmatch(src[start:start+end], -1)
	out := make([]Permission, 0, len(found))
	for _, m := range found {
		out = append(out, Permission(m[1]))
	}
	return out
}

func assertSameKeys(t *testing.T, what string, want, got []Permission) {
	t.Helper()
	wantSet, gotSet := set(want), set(got)
	for _, p := range want {
		if !gotSet[p] {
			t.Errorf("%s is missing %q — the Go registry grants it, so the web "+
				"app will never show what it unlocks", what, p)
		}
	}
	for _, p := range got {
		if !wantSet[p] {
			t.Errorf("%s carries %q, which this registry does not grant", what, p)
		}
	}
}

func set(ps []Permission) map[Permission]bool {
	m := make(map[Permission]bool, len(ps))
	for _, p := range ps {
		m[p] = true
	}
	return m
}
