package model

import (
	"testing"
	"time"
)

func TestQuizBumpCutoff(t *testing.T) {
	now := time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC)

	got := QuizBumpCutoff(now)
	want := now.AddDate(0, 0, -3)
	if !got.Equal(want) {
		t.Fatalf("QuizBumpCutoff(%v) = %v, want %v", now, got, want)
	}

	cases := []struct {
		name      string
		created   time.Time
		wantBumps bool
	}{
		{"within window (1 day old)", now.AddDate(0, 0, -1), true},
		{"just inside (cutoff + 1s)", got.Add(time.Second), true},
		{"exactly at cutoff (aged out)", got, false},
		{"aged out (5 days old)", now.AddDate(0, 0, -5), false},
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
