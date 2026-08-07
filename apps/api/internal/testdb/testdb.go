// Package testdb hands a DB-backed test the one database it is allowed to
// touch, and refuses to find one any other way.
//
// The rule it enforces: a test may only ever use the DSN handed to it
// explicitly through TEST_DATABASE_DSN. It must not read .env, must not fall
// back to the application's own KUN_DATABASE_URL, and must never print the
// DSN. Those tests write real rows; a test that goes looking for a database
// eventually finds the developer's dev database — or worse — and writes to it
// without anybody having asked for that.
//
// Suites that share one database must also run serially:
//
//	TEST_DATABASE_DSN=... go test ./... -count=1 -p 1
//
// -p 1 keeps two packages from interleaving on the same tables (go test runs
// separate packages in parallel by default), and -count=1 keeps a cached PASS
// from hiding a suite that never actually ran.
package testdb

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// EnvVar names the only source of a test DSN.
const EnvVar = "TEST_DATABASE_DSN"

// Open connects to the test database, or skips the test when none was
// provided. The returned handle is silent: gorm's default logger would echo
// every statement, and statements carry test data around, not signal.
func Open(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv(EnvVar)
	if dsn == "" {
		t.Skipf("%s not set — DB-backed test skipped", EnvVar)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test database: %v", redact(err.Error(), dsn))
	}
	return db
}

// redact keeps the DSN out of a failure message. A driver's connection error
// quotes back what it was given — and pgx reassembles the parts rather than
// echoing the string, so masking the DSN verbatim is not enough. Everything
// identifying stays masked; the cause (refused, timeout, auth) survives, which
// is the half worth reading.
func redact(msg, dsn string) string {
	msg = strings.ReplaceAll(msg, dsn, "<"+EnvVar+">")
	for _, part := range dsnParts(dsn) {
		msg = strings.ReplaceAll(msg, part, "<redacted>")
	}
	return msg
}

// dsnParts lists the identifying substrings of either spelling postgres
// accepts: a URL ("postgres://user:pw@host:port/db") or a keyword string
// ("host=... user=... password=... dbname=..."). Short parts are dropped —
// masking a two-character database name would shred the whole message.
func dsnParts(dsn string) []string {
	var out []string
	add := func(s string) {
		if len(s) >= 3 {
			out = append(out, s)
		}
	}

	if u, err := url.Parse(dsn); err == nil && u.Host != "" {
		add(u.Hostname())
		add(strings.TrimPrefix(u.Path, "/"))
		if u.User != nil {
			add(u.User.Username())
			if pw, ok := u.User.Password(); ok {
				add(pw)
			}
		}
	}
	for field := range strings.FieldsSeq(dsn) {
		if k, v, ok := strings.Cut(field, "="); ok {
			switch k {
			case "host", "user", "password", "dbname":
				add(v)
			}
		}
	}
	return out
}
