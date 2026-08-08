package service

import (
	"testing"

	"kun-galgame-api/internal/galgame/client"
)

// The catalog carries a curated Chinese name for most of the VNDB trait
// vocabulary but not all of it, so the page must survive a mixed set: the
// resolution happens per FIELD, not per trait, and a group must never end up
// half Chinese and half English just because one member is unrendered.
func TestCharacterTraitsResolveToChineseWhereAvailable(t *testing.T) {
	ch := &client.CatalogCharacter{
		Traits: []client.CatalogCharacterTrait{
			{ID: 1, Name: "Blonde", NameZh: "金发", Group: "Hair", GroupZh: "发型"},
			{ID: 2, Name: "Ahoge", Group: "Hair", GroupZh: "发型"},
			{ID: 3, Name: "Bust Size", NameZh: "胸围", Group: "Body", GroupZh: "身体", Sexual: true},
		},
	}

	got := characterTraits(ch, false)
	if len(got) != 3 {
		t.Fatalf("nsfw reader got %d traits, want 3", len(got))
	}
	if got[0].Name != "金发" || got[0].Group != "发型" {
		t.Errorf("rendered trait = %q / %q, want 金发 / 发型", got[0].Name, got[0].Group)
	}
	// No Chinese name, but its group has one: the English name falls back on its
	// own while the chip still files under 发型 alongside its Chinese sibling.
	if got[1].Name != "Ahoge" || got[1].Group != "发型" {
		t.Errorf("unrendered trait = %q / %q, want Ahoge / 发型", got[1].Name, got[1].Group)
	}

	// The SFW gate is unchanged by the localization — it still drops the sexual
	// family, Chinese name or not.
	sfw := characterTraits(ch, true)
	if len(sfw) != 2 {
		t.Fatalf("sfw reader got %d traits, want the sexual one dropped", len(sfw))
	}
}
