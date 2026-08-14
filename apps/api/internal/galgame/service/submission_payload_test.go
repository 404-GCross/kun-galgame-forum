package service

import (
	"reflect"
	"testing"
)

func sampleForm() *SubmissionForm {
	return &SubmissionForm{
		NameJaJP: "白恋サクラ", NameZhCN: "白恋樱", NameEnUS: "  ",
		IntroJaJP: "紹介", IntroZhCN: " ",
		ContentLimit: "nsfw", AgeLimit: "r18",
		OriginalLanguage: "ja-jp",
		Aliases:          []string{"シロコイ", "  "},
		ReleaseDate:      "2019-06-00",
	}
}

func TestSubmissionFieldsTranslateLocales(t *testing.T) {
	fields := sampleForm().Fields()

	titles, ok := fields["catalog.work.titles"].([]any)
	if !ok {
		t.Fatalf("titles missing: %#v", fields["catalog.work.titles"])
	}
	want := []map[string]any{
		{"lang": "ja", "title": "白恋サクラ", "kind": titleKindOfficial},
		{"lang": "zh-Hans", "title": "白恋樱", "kind": titleKindOfficial},
		{"lang": "", "title": "シロコイ", "kind": titleKindAlias},
	}
	if len(titles) != len(want) {
		t.Fatalf("titles = %v, want %d entries (blank ones dropped, not sent empty)", titles, len(want))
	}
	for i, w := range want {
		if !reflect.DeepEqual(titles[i], any(w)) {
			t.Errorf("title %d = %v, want %v", i, titles[i], w)
		}
	}

	intros, _ := fields["catalog.work.intros"].([]any)
	if len(intros) != 1 || !reflect.DeepEqual(intros[0], any(map[string]any{"lang": "ja", "intro": "紹介"})) {
		t.Errorf("intros = %v, want only the non-blank ja one", intros)
	}
	if fields["catalog.work.olang"] != "ja" {
		t.Errorf("olang = %v, want ja (not the product locale ja-jp)", fields["catalog.work.olang"])
	}
	if fields["catalog.work.content_rating"] != contentRatingR18 {
		t.Errorf("content_rating = %v, want r18", fields["catalog.work.content_rating"])
	}
	if fields["catalog.work.display_nsfw"] != true {
		t.Errorf("display_nsfw = %v, want true", fields["catalog.work.display_nsfw"])
	}
	if fields["catalog.work.display_name"] != "白恋サクラ" {
		t.Errorf("display_name = %v, want the ja title", fields["catalog.work.display_name"])
	}
	if _, present := fields["catalog.work.covers"]; present {
		t.Error("covers must not ride the mint — the bytes are referenced, not carried")
	}
}

// catalog's editspec refuses a cover element carrying any key outside this set,
// and the forum swallows that refusal, so a stray key silently loses the banner.
var coverPatchKeys = map[string]bool{
	"image_hash": true, "kind": true, "portrait_pinned": true,
	"sexual": true, "violence": true,
}

func TestSubmissionCoverPatchCarriesOnlyAcceptedKeys(t *testing.T) {
	patch := (&SubmissionForm{BannerHash: "  abc123  "}).CoverPatch()
	covers, ok := patch["catalog.work.covers"].([]any)
	if !ok || len(covers) != 1 {
		t.Fatalf("covers = %#v, want one element", patch["catalog.work.covers"])
	}
	cover, ok := covers[0].(map[string]any)
	if !ok {
		t.Fatalf("cover = %#v, want an object", covers[0])
	}
	for key := range cover {
		if !coverPatchKeys[key] {
			t.Errorf("cover carries %q, which catalog rejects as an unknown key", key)
		}
	}
	if cover["image_hash"] != "abc123" {
		t.Errorf("image_hash = %v, want the trimmed hash", cover["image_hash"])
	}
	if (&SubmissionForm{BannerHash: "   "}).CoverPatch() != nil {
		t.Error("a blank hash must produce no patch at all")
	}
}

func TestSubmissionOmitsEmptyLists(t *testing.T) {
	fields := (&SubmissionForm{NameJaJP: "x", AgeLimit: "all", ContentLimit: "sfw"}).Fields()
	if _, present := fields["catalog.work.intros"]; present {
		t.Error("an intro-less submission must omit the key, not send []")
	}
	if fields["catalog.work.content_rating"] != contentRatingAllAges {
		t.Errorf("content_rating = %v, want all_ages", fields["catalog.work.content_rating"])
	}
	if fields["catalog.work.display_nsfw"] != false {
		t.Errorf("display_nsfw = %v, want false", fields["catalog.work.display_nsfw"])
	}
}

func TestSubmissionReleaseDatePrecision(t *testing.T) {
	cases := []struct {
		raw     string
		want    *[3]int16
		wantErr bool
	}{
		{"", nil, false},
		{"2019", &[3]int16{2019, 0, 0}, false},
		{"2019-06-00", &[3]int16{2019, 6, 0}, false},
		{"2019-06-14", &[3]int16{2019, 6, 14}, false},
		{"2019-00-14", nil, true},
		{"0000-01-01", nil, true},
		{"not-a-date", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			got, appErr := (&SubmissionForm{ReleaseDate: tc.raw}).Released()
			if tc.wantErr {
				if appErr == nil {
					t.Fatalf("want a refusal for %q, got %+v", tc.raw, got)
				}
				return
			}
			if appErr != nil {
				t.Fatalf("unexpected refusal: %v", appErr)
			}
			if tc.want == nil {
				if got != nil {
					t.Fatalf("want TBA (nil), got %+v", got)
				}
				return
			}
			if got == nil || got.Y != tc.want[0] || got.M != tc.want[1] || got.D != tc.want[2] {
				t.Errorf("got %+v, want %v", got, tc.want)
			}
		})
	}
}

func TestSubmissionUnknownOLangIsPassedThrough(t *testing.T) {
	if got := olangOf("ko-kr"); got != "ko-kr" {
		t.Errorf("olangOf(ko-kr) = %q, want it forwarded for the registry to judge", got)
	}
	if got := olangOf("ZH-TW"); got != "zh-Hant" {
		t.Errorf("olangOf(ZH-TW) = %q, want zh-Hant", got)
	}
}
