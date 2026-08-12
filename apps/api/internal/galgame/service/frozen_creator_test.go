package service

import (
	"testing"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/userclient"
)

func creatorID(id int) *int { return &id }

func userMapWithCatalogDecoy() map[int]userclient.User {
	return map[int]userclient.User{
		0:  userclient.Placeholder(0),
		42: {ID: 42, Name: "鲲", Avatar: "https://example.invalid/kun.webp"},
	}
}

func TestFrozenCreatorBrief_ReadsLocalSnapshotNotCatalogFace(t *testing.T) {
	got := frozenCreatorBrief(
		repository.GalgameLocalRow{ID: 7, CreatorUserID: creatorID(42)},
		userMapWithCatalogDecoy(),
	)

	if got.ID != 42 || got.Name != "鲲" {
		t.Fatalf("brief = %+v, want the frozen local creator (id 42) — a zero/placeholder "+
			"here means the lane is keying off the catalog brief's UserID again", got)
	}
	if got.Avatar == "" {
		t.Errorf("avatar = %q, want the creator's — the chip renders name + avatar", got.Avatar)
	}
}

func TestFrozenCreatorBrief_UnknownCreatorRendersNoChip(t *testing.T) {
	rows := map[string]repository.GalgameLocalRow{
		"null creator": {ID: 7},
		"no local row": {},
	}
	for name, row := range rows {
		t.Run(name, func(t *testing.T) {
			got := frozenCreatorBrief(row, userMapWithCatalogDecoy())
			if got != (dto.UserBrief{}) {
				t.Fatalf("brief = %+v, want the zero brief — an unknown creator must render "+
					"as no chip, never as the 已注销用户 placeholder", got)
			}
		})
	}
}

func TestFrozenCreatorIDs_SkipsUnknownAndDeduplicates(t *testing.T) {
	ids := []int{1, 2, 3, 4, 5}
	localMap := map[int]repository.GalgameLocalRow{
		1: {ID: 1, CreatorUserID: creatorID(42)},
		2: {ID: 2},
		3: {ID: 3, CreatorUserID: creatorID(9)},
		4: {ID: 4, CreatorUserID: creatorID(42)},
	}

	got := frozenCreatorIDs(ids, localMap)

	want := []int{42, 9}
	if len(got) != len(want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids = %v, want %v (card order, deduplicated)", got, want)
		}
	}
	for _, id := range got {
		if id == 0 {
			t.Fatal("user 0 was queued for hydration — Hydrate answers with a " +
				"已注销用户 placeholder for any id it is asked about, so an unknown " +
				"creator would render as a deleted account instead of no chip")
		}
	}
}
