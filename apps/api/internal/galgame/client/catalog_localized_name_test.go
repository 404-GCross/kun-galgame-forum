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

func TestCatalogNameByScript_ReachesTheRenderingTheBucketsHid(t *testing.T) {
	localized := map[string]catLocalizedName{
		"zh-Hans": {Value: "美坂栞", Kind: "translation"},
	}

	ja, zh := CatalogNameByScript(localized, "みさか しおり", "ja")
	if ja != "みさか しおり" {
		t.Errorf("the record's own language fills its own script: ja = %q", ja)
	}
	if zh != "美坂栞" {
		t.Errorf("zh = %q, want the rendering the buckets could not publish", zh)
	}
	if headline := PickCatalogName("みさか しおり", ""); zh == headline {
		t.Error("a zh equal to the headline is dropped by the page — the dead slot the buckets produced")
	}
}

func TestCatalogNameByScript_AbsenceIsAnAnswer(t *testing.T) {
	for _, tc := range []struct {
		why            string
		localized      map[string]catLocalizedName
		displayName    string
		lang           string
		wantJa, wantZh string
	}{
		{
			why:       "no localized entry leaves the subtitle empty, not blank-labelled",
			localized: nil, displayName: "麻枝准", lang: "ja",
			wantJa: "麻枝准", wantZh: "",
		},
		{
			why: "a more specific tag wins over the bare one",
			localized: map[string]catLocalizedName{
				"zh-Hans": {Value: "简体"}, "zh": {Value: "通用"},
			},
			displayName: "原名", lang: "ja",
			wantJa: "原名", wantZh: "简体",
		},
		{
			why:         "zh-Hant answers when it is the only Chinese on file",
			localized:   map[string]catLocalizedName{"zh-Hant": {Value: "繁體"}},
			displayName: "原名", lang: "ja",
			wantJa: "原名", wantZh: "繁體",
		},
		{
			why:       "an undeclared language is not silently called Japanese",
			localized: nil, displayName: "Sound Horizon", lang: "",
			wantJa: "", wantZh: "",
		},
		{
			why:       "a Chinese record fills its own script and leaves ja empty",
			localized: nil, displayName: "轻文轻小说", lang: "zh-Hans",
			wantJa: "", wantZh: "轻文轻小说",
		},
		{
			why:         "an empty value is treated as absent, not as an answer",
			localized:   map[string]catLocalizedName{"zh-Hans": {Value: ""}},
			displayName: "原名", lang: "ja",
			wantJa: "原名", wantZh: "",
		},
	} {
		ja, zh := CatalogNameByScript(tc.localized, tc.displayName, tc.lang)
		if ja != tc.wantJa || zh != tc.wantZh {
			t.Errorf("%s: got (ja=%q, zh=%q), want (ja=%q, zh=%q)",
				tc.why, ja, zh, tc.wantJa, tc.wantZh)
		}
	}
}
