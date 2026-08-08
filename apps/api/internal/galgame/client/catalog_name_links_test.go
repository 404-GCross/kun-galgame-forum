package client

import (
	"encoding/json"
	"testing"
)

// The person web-presence lane (wave 186) arrives on the names face as its own
// `links[]`, DISJOINT from `refs[]`: a ref is an identity anchor with a bare
// external id, a link is a rendered address. The forum decoded only refs, so
// every one of these rows was dropped on the floor — this pins that they are
// read, and that each one has a name a reader can recognise.
//
// The fixture is a real /v1/catalog/names/{id} payload, trimmed to the two
// lanes.
const nameLinksFixture = `{
  "id": 900,
  "refs": [
    {"source": "vndb", "external_id": "7611"},
    {"source": "dlsite", "external_id": "RG12345"}
  ],
  "links": [
    {"source": "official_site", "url": "https://hk52.info"},
    {"source": "twitter", "url": "https://x.com/hozumik"},
    {"source": "pixiv", "url": "https://www.pixiv.net/users/3870762"},
    {"source": "web", "url": "https://cmc.booth.pm/"},
    {"source": "web", "url": "https://hozumik.fanbox.cc/"},
    {"source": "web", "url": "https://hozumik.tumblr.com/"},
    {"source": "web", "url": "https://anidb.net/cr1505"}
  ]
}`

func TestCatalogNameDecodesThePersonLinkLane(t *testing.T) {
	var n CatalogName
	if err := json.Unmarshal([]byte(nameLinksFixture), &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(n.Refs) != 2 {
		t.Fatalf("refs = %d, want 2 — the anchor lane must survive alongside links", len(n.Refs))
	}
	want := []string{"官方网站", "X", "pixiv", "BOOTH", "pixivFANBOX", "Tumblr", "AniDB"}
	if len(n.Links) != len(want) {
		t.Fatalf("links = %d, want %d", len(n.Links), len(want))
	}
	for i, link := range n.Links {
		if link.URL == "" {
			t.Errorf("links[%d] decoded with no URL", i)
		}
		if got := LinkDisplayName(link.Source, link.URL); got != want[i] {
			t.Errorf("links[%d] named %q, want %q (%s)", i, got, want[i], link.URL)
		}
	}
}

func TestCatalogNameWithNoPersonLinksDecodesEmpty(t *testing.T) {
	// An orphan name — or one whose person link the registry withholds — sends
	// []. It must not become a nil the caller has to special-case.
	var n CatalogName
	if err := json.Unmarshal([]byte(`{"id": 1, "links": []}`), &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(n.Links) != 0 {
		t.Errorf("links = %v, want empty", n.Links)
	}
}
