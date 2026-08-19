package client

import "testing"

// Wave 212 wave A supersedes a trait's name_zh / group_zh with the localized
// primitive and wave B deletes them. Both generations must resolve to the same
// Chinese: catalog and the forum deploy independently in either order, and
// these fields are the only reason a trait reads 金发 rather than Blonde, so
// getting it wrong reverts every character's traits to English in silence.
func TestCatalogCharacterTrait_ReadsBothGenerationsOfTheChineseName(t *testing.T) {
	zh := func(v string) map[string]catLocalizedName {
		return map[string]catLocalizedName{"zh-Hans": {Value: v}}
	}
	for _, tc := range []struct {
		why               string
		trait             CatalogCharacterTrait
		wantName, wantGrp string
	}{
		{
			why: "the primitive answers once catalog has caught up",
			trait: CatalogCharacterTrait{
				Name: "Blonde", Localized: zh("金发"),
				Group: "Hair", GroupLocalized: zh("发型"),
			},
			wantName: "金发", wantGrp: "发型",
		},
		{
			why: "the superseded flat field still answers until catalog deploys",
			trait: CatalogCharacterTrait{
				Name: "Blonde", NameZh: "金发", Group: "Hair", GroupZh: "发型",
			},
			wantName: "金发", wantGrp: "发型",
		},
		{
			why: "the primitive wins when both arrive, so the rename is not a cutover",
			trait: CatalogCharacterTrait{
				Name: "Blonde", Localized: zh("新"), NameZh: "旧",
				Group: "Hair", GroupLocalized: zh("新组"), GroupZh: "旧组",
			},
			wantName: "新", wantGrp: "新组",
		},
		{
			why: "no Chinese anywhere renders the vocabulary's English, never a blank",
			trait: CatalogCharacterTrait{
				Name: "Ahoge", Group: "Hair",
				Localized: map[string]catLocalizedName{}, GroupLocalized: map[string]catLocalizedName{},
			},
			wantName: "Ahoge", wantGrp: "Hair",
		},
		{
			why: "an empty localized value is absent, not an answer that blanks the row",
			trait: CatalogCharacterTrait{
				Name: "Blonde", Localized: zh(""), NameZh: "金发",
				Group: "Hair", GroupLocalized: zh(""), GroupZh: "发型",
			},
			wantName: "金发", wantGrp: "发型",
		},
	} {
		if got := tc.trait.LocalName(); got != tc.wantName {
			t.Errorf("%s: name = %q, want %q", tc.why, got, tc.wantName)
		}
		if got := tc.trait.LocalGroup(); got != tc.wantGrp {
			t.Errorf("%s: group = %q, want %q", tc.why, got, tc.wantGrp)
		}
	}
}
