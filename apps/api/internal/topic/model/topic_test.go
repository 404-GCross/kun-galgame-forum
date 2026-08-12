package model

import (
	"testing"
	"time"
)

func TestBumpCutoff(t *testing.T) {
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)

	got := BumpCutoff(now)
	want := now.AddDate(0, -3, 0)
	if !got.Equal(want) {
		t.Fatalf("BumpCutoff(%v) = %v, want %v", now, got, want)
	}

	cases := []struct {
		name      string
		created   time.Time
		wantBumps bool
	}{
		{"within window (2 months old)", now.AddDate(0, -2, 0), true},
		{"just inside (cutoff + 1s)", got.Add(time.Second), true},
		{"exactly at cutoff (aged out)", got, false},
		{"aged out (4 months old)", now.AddDate(0, -4, 0), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bumps := c.created.After(got)
			if bumps != c.wantBumps {
				t.Errorf("created=%v bumps=%v, want %v", c.created, bumps, c.wantBumps)
			}
		})
	}
}
