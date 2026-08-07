package main

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"time"
)

// seedRow is one (galgame, user) pair from the frozen wiki ledger.
type seedRow struct {
	GalgameID int64
	UserID    int64
	Created   time.Time
}

// parseStats is what the reader saw, so a run can be explained without the file.
type parseStats struct {
	Lines   int
	Skipped int
	// Folded counts pairs that appeared more than once. The ledger is one row
	// per contribution, and a person who touched a game twice is still one
	// contributor — folding here rather than in SQL keeps the INSERT's VALUES
	// list free of the duplicate ON CONFLICT postgres refuses to apply twice.
	Folded int
}

// contributorTSVColumns is the frozen ledger's shape:
// galgame_id, catalog_work_id, user_id, created. The catalog work id is read
// past deliberately — the seed is keyed in GID space, which is what the local
// table stores, and the mapping is already reflected in column 1.
const contributorTSVColumns = 4

// parseContributorTSV reads the ledger, folding repeat contributions onto the
// earliest timestamp seen for the pair.
//
// A row with no usable user is skipped rather than repaired: user 0 is the
// wiki's "unknown", and a contributor strip must not credit it.
func parseContributorTSV(r io.Reader) ([]seedRow, parseStats, error) {
	var stats parseStats
	rows := make([]seedRow, 0, 1024)
	index := map[[2]int64]int{}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		stats.Lines++

		parts := strings.Split(line, "\t")
		if len(parts) < contributorTSVColumns {
			stats.Skipped++
			continue
		}
		gid, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		uid, err2 := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
		created, err3 := parseLedgerTime(strings.TrimSpace(parts[3]))
		if err1 != nil || err2 != nil || err3 != nil || gid <= 0 || uid <= 0 {
			stats.Skipped++
			continue
		}

		key := [2]int64{gid, uid}
		if at, ok := index[key]; ok {
			stats.Folded++
			if created.Before(rows[at].Created) {
				rows[at].Created = created
			}
			continue
		}
		index[key] = len(rows)
		rows = append(rows, seedRow{GalgameID: gid, UserID: uid, Created: created})
	}
	return rows, stats, sc.Err()
}

// ledgerTimeLayouts are the postgres text timestamps the dump emits — with and
// without a fractional part, which the dump drops on a whole second.
var ledgerTimeLayouts = []string{
	"2006-01-02 15:04:05.999999-07",
	"2006-01-02 15:04:05-07",
	time.RFC3339,
}

func parseLedgerTime(s string) (time.Time, error) {
	var err error
	for _, layout := range ledgerTimeLayouts {
		var t time.Time
		if t, err = time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}
