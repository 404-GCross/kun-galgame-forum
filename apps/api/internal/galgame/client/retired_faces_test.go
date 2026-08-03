package client

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Wave-161 P5 (infra 0e919457 / dab23ea9) deleted every wiki HTTP face behind
// this client's two legacy bases: internalBase ({base}/internal — the
// platform-workflow reads, the user-write set, the propose face, the two cron
// feeds) and legacyBase ({base}/api — the staff taxonomy + admin face). Infra
// pins them as plain 404s (TestRetiredSiblingFacesFallThroughTo404).
//
// This is the consumer-side mirror of that pin, and it exists because the
// failure is SILENT in both directions: infra's access log recorded those 404s
// as 200 (since fixed in api/internal/middleware/logger.go) and this client
// surfaces an upstream 404 as an ordinary not-found. kungal lost game editing
// for ~67 hours and fired 158,959 ownership lookups a day into a face that had
// not existed for three days, with nothing red anywhere.
//
// The census is anchored on the BASE FIELDS rather than on URLs: readTarget /
// writeTarget assemble the prefix from c.internalBase / c.legacyBase, so a call
// site's string literal ("/galgame/messages/mine") does not reveal which face it
// lands on. Every path to a retired face goes through one of these two fields,
// so pinning their reference sites is what actually closes the door.
//
// The map may only SHRINK. A new entry means a new call into a face that
// answers 404; growing a count means the same thing inside a file that already
// had one.
//
// It is deliberately not an empty map with the offenders deleted — the four
// features still on these bases (submission-form tag picker, staff taxonomy
// read-back, wiki notification page, admin message queue + stats) need a
// product decision about their replacement, tracked in infra
// .pi/tracks/data-layer-retirement.md. Deleting the calls would delete the
// features.
var retiredBaseRefs = map[string]int{
	// readTarget + writeTarget: the two routers that hand out these bases.
	"internal/galgame/client/client.go": 4,
	// [1] staff taxonomy read-back  /api/{family}/{id}
	// [2] submission-form tag picker /internal/galgame/taxonomy/{family}/search
	"internal/galgame/client/catalog_taxonomy.go": 2,
}

var retiredBasePattern = regexp.MustCompile(`c\.(internalBase|legacyBase)\b`)

func TestNoNewCallsIntoRetiredWikiFaces(t *testing.T) {
	root := moduleRootForTest(t)

	found := map[string]int{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		n := len(retiredBasePattern.FindAllString(string(src), -1))
		if n == 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		found[filepath.ToSlash(rel)] = n
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	files := make([]string, 0, len(found)+len(retiredBaseRefs))
	for f := range found {
		files = append(files, f)
	}
	for f := range retiredBaseRefs {
		if _, ok := found[f]; !ok {
			files = append(files, f)
		}
	}
	sort.Strings(files)

	for _, f := range files {
		got, want := found[f], retiredBaseRefs[f]
		switch {
		case want == 0:
			t.Errorf("%s reaches a retired wiki face (%d ref(s)) — those routes answer 404 and fail silently; "+
				"read the base-field census in this file before adding one", f, got)
		case got > want:
			t.Errorf("%s: %d references to a retired base, census pins %d — a new call into a 404 face", f, got, want)
		case got < want:
			t.Errorf("%s: %d references, census pins %d — a lane was retired, lower the count here", f, got, want)
		}
	}
}

// moduleRootForTest walks up from the package directory to the directory
// holding go.mod.
func moduleRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test package")
		}
		dir = parent
	}
}
