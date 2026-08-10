package client

import (
	"encoding/json"
	"strings"
	"testing"
)

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }

const cdn = "https://image.kungal.iloveren.link"

func TestBannerURLFromHash(t *testing.T) {
	cases := []struct {
		hash, want string
	}{
		{"abcd1234ef", cdn + "/ab/cd/abcd1234ef.webp"},
		{"00ff00ff00ff", cdn + "/00/ff/00ff00ff00ff.webp"},
		{"abc", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := bannerURLFromHash(cdn, c.hash); got != c.want {
			t.Errorf("bannerURLFromHash(%q) = %q, want %q", c.hash, got, c.want)
		}
	}
	if got := bannerURLFromHash(cdn+"/", "abcd"); got != cdn+"/ab/cd/abcd.webp" {
		t.Errorf("trailing-slash base not trimmed: %q", got)
	}
}

func TestRewriteBanners_NumberRoundtrip(t *testing.T) {
	in := json.RawMessage(`{"galgame":{"id":60744,"status":3,"effective_banner_hash":"abcd1234ef"}}`)
	out := rewriteBanners(in, cdn)

	var got struct {
		Galgame struct {
			ID     int    `json:"id"`
			Status int    `json:"status"`
			URL    string `json:"effective_banner_url"`
		} `json:"galgame"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v (out=%s)", err, out)
	}
	if got.Galgame.ID != 60744 || got.Galgame.Status != 3 {
		t.Errorf("number round-trip broke: id=%d status=%d", got.Galgame.ID, got.Galgame.Status)
	}
	if got.Galgame.URL != cdn+"/ab/cd/abcd1234ef.webp" {
		t.Errorf("effective_banner_url not injected: %q", got.Galgame.URL)
	}
}

func TestRewriteBanners_EffectiveBannerURL(t *testing.T) {
	in := json.RawMessage(`{"galgame":{"id":1,"effective_banner_hash":"abcd1234ef"}}`)
	out := rewriteBanners(in, cdn)
	var got struct {
		Galgame struct {
			URL string `json:"effective_banner_url"`
		} `json:"galgame"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Galgame.URL != cdn+"/ab/cd/abcd1234ef.webp" {
		t.Errorf("effective_banner_url not injected: %q", got.Galgame.URL)
	}

	keep := json.RawMessage(`{"effective_banner_hash":"abcd","effective_banner_url":"https://pinned/x.webp"}`)
	out2 := rewriteBanners(keep, cdn)
	if !contains(string(out2), `"https://pinned/x.webp"`) {
		t.Errorf("existing effective_banner_url should be preserved: %s", out2)
	}
}

func TestRewriteBanners_CoverAndScreenshotCDNURL(t *testing.T) {
	in := json.RawMessage(`{"galgame":{
		"covers":[
			{"image_hash":"deadbeef","sort_order":0,"sexual":0,"violence":0},
			{"image_hash":"cafef00d","sort_order":1,"sexual":0,"violence":0}
		],
		"screenshots":[
			{"image_hash":"feedbabe","sort_order":0,"caption":"OP","sexual":0,"violence":0}
		]
	}}`)
	out := rewriteBanners(in, cdn)
	var got struct {
		Galgame struct {
			Covers []struct {
				ImageHash string `json:"image_hash"`
				CDNURL    string `json:"cdn_url"`
			} `json:"covers"`
			Screenshots []struct {
				ImageHash string `json:"image_hash"`
				CDNURL    string `json:"cdn_url"`
				Caption   string `json:"caption"`
			} `json:"screenshots"`
		} `json:"galgame"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v (out=%s)", err, out)
	}
	if got.Galgame.Covers[0].CDNURL != cdn+"/de/ad/deadbeef.webp" {
		t.Errorf("cover[0] cdn_url wrong: %q", got.Galgame.Covers[0].CDNURL)
	}
	if got.Galgame.Covers[1].CDNURL != cdn+"/ca/fe/cafef00d.webp" {
		t.Errorf("cover[1] cdn_url wrong: %q", got.Galgame.Covers[1].CDNURL)
	}
	if got.Galgame.Screenshots[0].CDNURL != cdn+"/fe/ed/feedbabe.webp" {
		t.Errorf("screenshot cdn_url wrong: %q", got.Galgame.Screenshots[0].CDNURL)
	}
	if got.Galgame.Screenshots[0].Caption != "OP" {
		t.Errorf("sibling caption lost: %q", got.Galgame.Screenshots[0].Caption)
	}
}

func TestRewriteBanners_SkipBareImageRecord(t *testing.T) {
	in := json.RawMessage(`{"image":{"image_hash":"abcd1234","url":"https://wiki-image/x.webp"}}`)
	out := rewriteBanners(in, cdn)
	if contains(string(out), `"cdn_url"`) {
		t.Errorf("should not inject cdn_url on bare image record: %s", out)
	}
}

func TestRewriteBanners_FailSafe(t *testing.T) {
	raw := json.RawMessage(`{"galgame":{"effective_banner_hash":"abcd1234"}}`)
	if string(rewriteBanners(raw, "")) != string(raw) {
		t.Error("empty cdnBase should passthrough verbatim")
	}
	bad := json.RawMessage(`Cannot GET /api/galgame/mine`)
	if string(rewriteBanners(bad, cdn)) != string(bad) {
		t.Error("non-JSON should passthrough verbatim")
	}
	noop := json.RawMessage(`{"galgame":{"banner":"https://x/y.webp"}}`)
	if string(rewriteBanners(noop, cdn)) != string(noop) {
		t.Error("no-op payload should return original bytes")
	}
}
