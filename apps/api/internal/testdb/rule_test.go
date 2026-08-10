package testdb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var forbidden = map[string]string{
	"godotenv":          "a test must not load .env; the DSN comes from " + envVar + " alone",
	"KUN_DATABASE_URL":  "that is the application's DSN, not a test's; use " + envVar,
	"gorm.Open":         "open the test database with testdb.Open, which enforces the rule",
	"postgres.Open":     "open the test database with testdb.Open, which enforces the rule",
	"TEST_DATABASE_DSN": "read it through testdb.Open rather than directly",
	"os.Getenv(\"KUN_":  "a test must not reach into the application's environment",
}

const envVar = "TEST_DATABASE_DSN"

func TestNoTestFindsItsOwnDatabase(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}

	selfDir := filepath.Join(root, "internal", "testdb")

	count := 0
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" || path == selfDir {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		count++
		rel, _ := filepath.Rel(root, path)
		for line := range strings.SplitSeq(string(src), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			for token, why := range forbidden {
				if strings.Contains(line, token) {
					t.Errorf("%s uses %q — %s", rel, token, why)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if count == 0 {
		t.Fatal("no test files found — the walk root is wrong")
	}
}
