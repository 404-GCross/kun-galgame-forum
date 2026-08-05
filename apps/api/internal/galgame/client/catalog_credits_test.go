package client

// The 制作人员 panel is a display projection over the registry's credit rows,
// and every rule it applies is one the catalog cannot apply for us: it does not
// know that 插画 and 原画 are one job here, that two spellings are one person, or
// that a reader wants the writer above the cast.
//
// Pinned because all three failures are silent — the panel still renders, just
// with the same person listed twice under two headings, below a 53-name cast.

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
	// The shape a real work arrives in: the same illustrator credited through
	// two source vocabularies, the same composer written three ways, and the
	// brand credit the 会社 card already renders.
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
	// The plainest written form wins: the parenthesised aliases belong to the
	// identity layer, not to a staff list.
	if music.People[0].Name != "水城新人" {
		t.Errorf("composer renders as %q, want the unannotated 水城新人", music.People[0].Name)
	}

	// Authorship first, cast last — the reverse of the registry's role-id order,
	// which is what the face hands us.
	if got := strings.Join(keys, ","); got != "scenario,illustration,music,voice-actor" {
		t.Errorf("panel order = %s, want the writer before the cast", got)
	}
}

func TestCatalogStaff_VoiceActorCollectsCharacters(t *testing.T) {
	// One VA voicing two characters is one row with two names, not two rows —
	// the face emits one credit per (name, character) pair.
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

// other-staff is the unmapped bucket and routinely the longest group on the
// page, so it sits below every named role however many rows it carries.
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
