package client

import "testing"

// TestPickCatalogName_HeadlineIsTheCreditedName pins the half of the bucket
// migration that must NOT move: the page title. The retired buckets always
// yielded the name of record — they held one name filed under its own language
// — so a translated rendering appearing in the headline would be a regression
// introduced by the migration, not a feature of it.
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

// TestCatalogNameByScript_ReachesTheRenderingTheBucketsHid is the whole point of
// adopting localized{}. Under the buckets, name_zh was non-empty ONLY when the
// record was itself Chinese — which is exactly when it equals the headline, and
// the staff and character pages drop a subtitle part equal to the headline. So
// the Chinese subtitle could never render, including for the names that had a
// Chinese rendering on file.
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
	// The subtitle only renders because it differs from the headline.
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
			// The common case by far: a real person's kanji name is not
			// translated, and a Chinese UI renders it verbatim. Empty here means
			// "no separate rendering exists", which the page must show as
			// nothing rather than as an empty slot.
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
			// The buckets filed an undeclared lang under ja, asserting a
			// language the source never claimed. It is now simply unknown.
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
