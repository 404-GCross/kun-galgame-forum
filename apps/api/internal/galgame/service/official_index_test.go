package service

import (
	"testing"

	"kun-galgame-api/internal/galgame/dto"
)

func officials(spec ...any) []dto.OfficialListItem {
	out := make([]dto.OfficialListItem, 0, len(spec)/2)
	for i := 0; i < len(spec); i += 2 {
		out = append(out, dto.OfficialListItem{
			Name:         spec[i].(string),
			GalgameCount: spec[i+1].(int),
		})
	}
	return out
}

func names(items []dto.OfficialListItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out
}

func eq(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestSortOfficialsByCount_MostGamesFirst(t *testing.T) {
	items := officials("Key", 24, "ブロッコリー", 9, "Yuzusoft", 40)
	sortOfficialsByCount(items)
	eq(t, names(items), []string{"Yuzusoft", "Key", "ブロッコリー"})
}

func TestSortOfficialsByCount_TiesAreStableByName(t *testing.T) {
	items := officials("charlie", 1, "alpha", 1, "bravo", 1)
	sortOfficialsByCount(items)
	eq(t, names(items), []string{"alpha", "bravo", "charlie"})
}

func TestSliceOfficialPage(t *testing.T) {
	all := officials("a", 5, "b", 4, "c", 3, "d", 2, "e", 1)

	eq(t, names(sliceOfficialPage(all, 1, 2)), []string{"a", "b"})
	eq(t, names(sliceOfficialPage(all, 3, 2)), []string{"e"})
	eq(t, names(sliceOfficialPage(all, 99, 2)), nil)
	eq(t, names(sliceOfficialPage(all, 0, 2)), []string{"a", "b"})
}
