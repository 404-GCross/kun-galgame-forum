package client

import (
	"strings"
	"testing"
)

func creditGroup(key, name string, people ...string) catCreditGroup {
	g := catCreditGroup{RoleKey: key, RoleName: name}
	for i, p := range people {
		g.Credits = append(g.Credits, catCreditItem{ID: int64(i + 1), Name: p})
	}
	return g
}

func TestCatalogStaff_FoldsDuplicateRolesAndSpellings(t *testing.T) {
	staff := catalogStaffFromCredits([]catCreditGroup{
		creditGroup("voice-actor", "声优", "桃瀬ひな"),
		creditGroup("developer", "开发", "アンモライト"),
		creditGroup("illustration", "插画", "一河のあ"),
		creditGroup("scenario", "脚本", "海綿じろう"),
		creditGroup("music", "音乐", "水城 新人", "水城新人(獅子王院みづき、新澄トキ)"),
		creditGroup("原画", "原画", "一河のあ"),
		creditGroup("音乐", "音乐", "水城新人"),
	})

	byKey := map[string]int{}
	keys := make([]string, 0, len(staff))
	for i, g := range staff {
		byKey[g.RoleKey] = i
		keys = append(keys, g.RoleKey)
	}

	if _, hidden := byKey["developer"]; hidden {
		t.Errorf("开发 must not repeat the 会社 card, got groups %v", keys)
	}
	if _, split := byKey["原画"]; split {
		t.Errorf("原画 must fold into illustration, got groups %v", keys)
	}

	art := staff[byKey["illustration"]]
	if art.RoleName != "原画" {
		t.Errorf("folded art group renders as %q, want 原画", art.RoleName)
	}
	if len(art.People) != 1 {
		t.Errorf("one illustrator credited twice = %d rows, want 1", len(art.People))
	}

	music := staff[byKey["music"]]
	if len(music.People) != 1 {
		t.Fatalf("three spellings of one composer = %d rows, want 1", len(music.People))
	}
	if music.People[0].Name != "水城新人" {
		t.Errorf("composer renders as %q, want the unannotated 水城新人", music.People[0].Name)
	}

	if got := strings.Join(keys, ","); got != "scenario,illustration,music,voice-actor" {
		t.Errorf("panel order = %s, want the writer before the cast", got)
	}
}

func TestCatalogStaff_VoiceActorCollectsCharacters(t *testing.T) {
	g := catCreditGroup{RoleKey: "voice-actor", RoleName: "声优"}
	for _, ch := range []string{"藤田 佳奈", "ナレーション"} {
		g.Credits = append(g.Credits, catCreditItem{ID: 7, Name: "五十嵐裕美", Character: ch})
	}

	staff := catalogStaffFromCredits([]catCreditGroup{g})
	if len(staff) != 1 || len(staff[0].People) != 1 {
		t.Fatalf("one VA = %d groups / %d people, want 1/1", len(staff), len(staff[0].People))
	}
	if got := strings.Join(staff[0].People[0].Characters, ","); got != "藤田 佳奈,ナレーション" {
		t.Errorf("characters = %q, want both roles on the one VA", got)
	}
}

func TestCatalogStaff_OtherStaffSinksToTheBottom(t *testing.T) {
	staff := catalogStaffFromCredits([]catCreditGroup{
		creditGroup("other-staff", "其他", "STUDIO696", "ワムソフト", "胡太郎"),
		creditGroup("qa", "QA", "RainbowcatcherBM"),
		creditGroup("scenario", "脚本", "雪仁"),
	})
	keys := make([]string, 0, len(staff))
	for _, g := range staff {
		keys = append(keys, g.RoleKey)
	}
	if got := strings.Join(keys, ","); got != "scenario,qa,other-staff" {
		t.Errorf("panel order = %s, want 其他 last and the unranked qa in between", got)
	}
}

func TestCatalogStaff_OtherStaffYieldsToClassifiedRoles(t *testing.T) {
	staff := catalogStaffFromCredits([]catCreditGroup{
		creditGroup("scenario", "脚本", "雪仁"),
		creditGroup("原画", "原画", "一河のあ"),
		creditGroup("other-staff", "其他", "雪 仁", "一河のあ", "胡太郎"),
	})

	var other []string
	for _, g := range staff {
		if g.RoleKey == "other-staff" {
			for _, p := range g.People {
				other = append(other, p.Name)
			}
		}
	}
	if got := strings.Join(other, ","); got != "胡太郎" {
		t.Errorf("其他 = %s, want only the name with no classified credit", got)
	}
}

func TestCatalogStaff_OtherStaffDropsWhenFullyDuplicated(t *testing.T) {
	staff := catalogStaffFromCredits([]catCreditGroup{
		creditGroup("scenario", "脚本", "雪仁"),
		creditGroup("other-staff", "其他", "雪仁"),
	})
	for _, g := range staff {
		if g.RoleKey == "other-staff" {
			t.Errorf("emptied 其他 still renders with %d people", len(g.People))
		}
	}
}

func TestSortStaffRoleKeys_RanksUnpinnedRolesToo(t *testing.T) {
	got := SortStaffRoleKeys([]string{
		"other-staff", "企画", "theme-song-lyrics", "director", "lyric", "scenario",
	})
	want := "scenario,director,lyric,theme-song-lyrics,企画,other-staff"
	if strings.Join(got, ",") != want {
		t.Errorf("role order = %s, want %s", strings.Join(got, ","), want)
	}
}

func TestStaffRoleLabel_AgreesWithThePanel(t *testing.T) {
	for _, c := range []struct{ key, upstream, want string }{
		{"原画", "原画", "原画"},
		{"illustration", "插画", "原画"},
		{"剧本", "剧本", "脚本"},
		{"director-direction", "导演", "导演"},
		{"qa", "QA", "QA"},
	} {
		if got := StaffRoleLabel(c.key, c.upstream); got != c.want {
			t.Errorf("StaffRoleLabel(%q, %q) = %q, want %q", c.key, c.upstream, got, c.want)
		}
	}
}
