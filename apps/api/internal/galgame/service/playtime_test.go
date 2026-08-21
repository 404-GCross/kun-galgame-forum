package service

import (
	"testing"

	"kun-galgame-api/pkg/catalogclient"
)

func TestFoldRecords_CollapsesClientsTheWayCatalogDoes(t *testing.T) {
	rows := []catalogclient.PlaytimeRecord{
		{WorkID: 7, Minutes: 300, Status: catalogclient.PlaytimeStatusFinished,
			ClientID: "forum", UpdatedAt: "2026-08-01T00:00:00Z", LastPlayedAt: ptr("2026-07-30T00:00:00Z")},
		{WorkID: 7, Minutes: 720, Status: catalogclient.PlaytimeStatusPlaying,
			ClientID: "tracker", UpdatedAt: "2026-08-09T00:00:00Z", LastPlayedAt: ptr("2026-08-08T00:00:00Z")},
		{WorkID: 9, Minutes: 60, Status: catalogclient.PlaytimeStatusDropped,
			ClientID: "forum", UpdatedAt: "2026-08-05T00:00:00Z"},
	}

	order, byWork := foldRecords(rows, "forum")
	if len(order) != 2 || order[0] != 7 || order[1] != 9 {
		t.Fatalf("order = %v, want [7 9]", order)
	}

	seven := byWork[7]
	if seven.minutes != 720 {
		t.Errorf("minutes = %d, want the larger 720", seven.minutes)
	}
	if seven.status != catalogclient.PlaytimeStatusFinished {
		t.Errorf("status = %q, want finished to survive a later playing row", seven.status)
	}
	if seven.clients != 2 {
		t.Errorf("clients = %d, want 2", seven.clients)
	}
	if !seven.external {
		t.Error("external = false, want true when the largest report is another app's")
	}
	if seven.lastPlayedAt != "2026-08-08T00:00:00Z" {
		t.Errorf("lastPlayedAt = %q, want the latest of the two", seven.lastPlayedAt)
	}
	if seven.updatedAt != "2026-08-09T00:00:00Z" {
		t.Errorf("updatedAt = %q, want the latest of the two", seven.updatedAt)
	}

	nine := byWork[9]
	if nine.external {
		t.Error("external = true for a forum-only row")
	}
	if nine.status != catalogclient.PlaytimeStatusDropped {
		t.Errorf("status = %q, want dropped", nine.status)
	}
}

// A cleared record is not deleted upstream — it is written down to 0 and the
// aggregate stops counting it. The profile page counted it anyway and printed
// "1 部作品 · 合计 · 已通关 1 部" for a work with nothing left to show.
func TestPlaytimeWithdrawn_TreatsASubFloorReportAsAbsent(t *testing.T) {
	cases := []struct {
		minutes int
		want    bool
	}{
		{0, true},
		{catalogclient.PlaytimeMinutesFloor - 1, true},
		{catalogclient.PlaytimeMinutesFloor, false},
		{720, false},
	}
	for _, c := range cases {
		if got := playtimeWithdrawn(c.minutes); got != c.want {
			t.Errorf("playtimeWithdrawn(%d) = %v, want %v", c.minutes, got, c.want)
		}
	}
}

// The fold takes MAX across a user's clients, so a work is withdrawn only when
// EVERY client is under the floor: clearing the forum's own row must not hide a
// desktop tracker's live one.
func TestFoldRecords_KeepsAWorkAnotherClientStillReports(t *testing.T) {
	rows := []catalogclient.PlaytimeRecord{
		{WorkID: 7, Minutes: 0, Status: catalogclient.PlaytimeStatusFinished,
			ClientID: "forum", UpdatedAt: "2026-08-10T00:00:00Z"},
		{WorkID: 7, Minutes: 480, Status: catalogclient.PlaytimeStatusPlaying,
			ClientID: "tracker", UpdatedAt: "2026-08-09T00:00:00Z"},
	}
	_, byWork := foldRecords(rows, "forum")
	if playtimeWithdrawn(byWork[7].minutes) {
		t.Errorf("folded minutes = %d, want the tracker's 480 to keep the work visible", byWork[7].minutes)
	}
}
