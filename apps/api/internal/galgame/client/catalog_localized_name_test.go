package client

import "testing"

func TestPickCatalogName_HeadlineIsTheCreditedName(t *testing.T) {
	for _, tc := range []struct {
		why         string
		displayName string
		latin       string
		want        string
	}{
		{"the credited name titles the page", "麻枝准", "Maeda Jun", "麻枝准"},
		{"latin carries a record the registry holds in no other script", "", "Frank Klepacki", "Frank Klepacki"},
		{"nothing at all still renders nothing, never a placeholder", "", "", ""},
	} {
		if got := PickCatalogName(tc.displayName, tc.latin); got != tc.want {
			t.Errorf("%s: PickCatalogName(%q, %q) = %q, want %q",
				tc.why, tc.displayName, tc.latin, got, tc.want)
		}
	}
}

func TestCatalogEntityName_RendersTheChineseNameWhenThereIsOne(t *testing.T) {
	for _, tc := range []struct {
		why                  string
		localized            map[string]catLocalizedName
		displayName, latin   string
		wantName, wantOrigin string
	}{
		{
			why:         "the Chinese name is what a Chinese reader is shown",
			localized:   map[string]catLocalizedName{"zh-Hans": {Value: "美坂栞", Kind: "translation"}},
			displayName: "みさか しおり", wantName: "美坂栞", wantOrigin: "みさか しおり",
		},
		{
			why: "a machine fill-in renders like any other name — before wave 209 it was " +
				"structurally unreachable and the reader got the Japanese one",
			localized: map[string]catLocalizedName{
				"zh-Hans": {Value: "猫猫社", Kind: "translation", Machine: true},
			},
			displayName: "ねこねこソフト", wantName: "猫猫社", wantOrigin: "ねこねこソフト",
		},
		{
			why:       "no Chinese on file falls back to the record's own name, never a blank",
			localized: nil, displayName: "麻枝准", wantName: "麻枝准",
		},
		{
			why:       "latin is the last cell, for a record held in no other script",
			localized: nil, latin: "Frank Klepacki", wantName: "Frank Klepacki",
		},
		{
			why: "a more specific tag wins over the bare one",
			localized: map[string]catLocalizedName{
				"zh-Hans": {Value: "简体"}, "zh": {Value: "通用"},
			},
			displayName: "原名", wantName: "简体", wantOrigin: "原名",
		},
		{
			why:         "zh-Hant answers when it is the only Chinese on file",
			localized:   map[string]catLocalizedName{"zh-Hant": {Value: "繁體"}},
			displayName: "原名", wantName: "繁體", wantOrigin: "原名",
		},
		{
			why:         "an empty value is absent, not an answer",
			localized:   map[string]catLocalizedName{"zh-Hans": {Value: ""}},
			displayName: "原名", wantName: "原名",
		},
		{
			why:         "a name already in Chinese prints once — no second line repeating it",
			localized:   map[string]catLocalizedName{"zh-Hans": {Value: "轻文轻小说"}},
			displayName: "轻文轻小说", wantName: "轻文轻小说",
		},
	} {
		name, origin := CatalogEntityNames(tc.localized, tc.displayName, tc.latin)
		if name != tc.wantName || origin != tc.wantOrigin {
			t.Errorf("%s: got (name=%q, original=%q), want (name=%q, original=%q)",
				tc.why, name, origin, tc.wantName, tc.wantOrigin)
		}
	}
}
