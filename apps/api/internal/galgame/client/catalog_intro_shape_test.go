package client

import (
	"encoding/json"
	"testing"
)

// Wave 212's second half reshapes the works intro blocks: the list brief's
// object of four product-locale slots becomes the [{lang, intro, …}] array the
// rest of the catalog already sends, and the detail face's "intro" key is
// renamed "intros". Both faces must read the same on either side of that
// deploy — the forum and catalog ship independently, so whichever goes first
// must not blank every game's introduction.
func TestCatalogIntros_ListBriefReadsBothShapes(t *testing.T) {
	const object = `{"id":9,"display_name":"Kun","intros":{` +
		`"ja-jp":{"intro":"日本語","source":"vndb"},` +
		`"zh-cn":{"intro":"简体","machine":true},` +
		`"zh-tw":{"intro":""}}}`
	const array = `{"id":9,"display_name":"Kun","intros":[` +
		`{"lang":"ja","intro":"日本語","source":"vndb"},` +
		`{"lang":"zh-Hans","intro":"简体","machine":true}]}`

	for _, tc := range []struct{ shape, body string }{
		{"the retiring object of product-locale slots", object},
		{"the array every other catalog face sends", array},
	} {
		var it CatalogWorkListItem
		if err := json.Unmarshal([]byte(tc.body), &it); err != nil {
			t.Fatalf("%s: unmarshal: %v", tc.shape, err)
		}
		b := CatalogItemToDetailBrief(&it)
		if b.IntroJaJP != "日本語" || b.IntroZhCN != "简体" {
			t.Errorf("%s: ja=%q zh-cn=%q, want 日本語 / 简体", tc.shape, b.IntroJaJP, b.IntroZhCN)
		}
		if b.IntroZhTW != "" || b.IntroEnUS != "" {
			t.Errorf("%s: zh-tw=%q en-us=%q, want both empty — an absent language is "+
				"absent, never another locale's text", tc.shape, b.IntroZhTW, b.IntroEnUS)
		}
	}
}

func TestCatalogIntros_ListBriefKeepsTheMachineFlagAcrossShapes(t *testing.T) {
	for _, tc := range []struct{ shape, body string }{
		{"object", `{"intros":{"zh-cn":{"intro":"简体","machine":true}}}`},
		{"array", `{"intros":[{"lang":"zh-Hans","intro":"简体","machine":true}]}`},
	} {
		var it CatalogWorkListItem
		if err := json.Unmarshal([]byte(tc.body), &it); err != nil {
			t.Fatalf("%s: unmarshal: %v", tc.shape, err)
		}
		if len(it.Intros) != 1 || !it.Intros[0].Machine {
			t.Errorf("%s: Intros = %+v, want the machine flag carried through the "+
				"shape change — it is the only thing separating a translation from the "+
				"publisher's own text", tc.shape, it.Intros)
		}
	}
}

func TestCatalogIntros_ListBriefSurvivesAnAbsentBlock(t *testing.T) {
	for _, body := range []string{`{"id":9}`, `{"id":9,"intros":null}`} {
		var it CatalogWorkListItem
		if err := json.Unmarshal([]byte(body), &it); err != nil {
			t.Fatalf("%s: unmarshal: %v", body, err)
		}
		if got := it.intro("zh-cn"); got != "" {
			t.Errorf("%s: intro = %q, want empty", body, got)
		}
	}
}

func TestCatalogIntros_DetailReadsBothKeys(t *testing.T) {
	const row = `{"lang":"ja","intro":"日本語"},{"lang":"zh-Hans","intro":"简体","machine":true}`

	for _, tc := range []struct{ key, body string }{
		{"intro", `{"id":9,"intro":[` + row + `]}`},
		{"intros", `{"id":9,"intros":[` + row + `]}`},
	} {
		var d catWorkDetail
		if err := json.Unmarshal([]byte(tc.body), &d); err != nil {
			t.Fatalf("%s: unmarshal: %v", tc.key, err)
		}
		text, machine := catalogIntros(&d)
		if text.JaJp != "日本語" || text.ZhCn != "简体" {
			t.Errorf("%s: ja=%q zh-cn=%q, want 日本語 / 简体", tc.key, text.JaJp, text.ZhCn)
		}
		if machine.JaJp || !machine.ZhCn {
			t.Errorf("%s: machine = %+v, want only zh-cn flagged", tc.key, machine)
		}
	}
}

// The rename ships as a dual-emit window, so a payload carrying both keys must
// resolve to one answer rather than concatenating or picking by struct order.
func TestCatalogIntros_DetailPrefersTheNewKeyWhenBothArrive(t *testing.T) {
	var d catWorkDetail
	body := `{"intro":[{"lang":"zh-Hans","intro":"旧"}],"intros":[{"lang":"zh-Hans","intro":"新"}]}`
	if err := json.Unmarshal([]byte(body), &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if text, _ := catalogIntros(&d); text.ZhCn != "新" {
		t.Errorf("zh-cn = %q, want 新 — during dual emit the renamed key is the live one", text.ZhCn)
	}
}
