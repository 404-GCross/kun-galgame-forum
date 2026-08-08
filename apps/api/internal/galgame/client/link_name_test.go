package client

import "testing"

func TestLinkDisplayName(t *testing.T) {
	for _, tc := range []struct {
		why    string
		source string
		url    string
		want   string
	}{
		{"a modelled source is named by its key", "official_site", "https://www.broccoli.co.jp", "官方网站"},
		{"an X account is never dressed up as 官方网站", "twitter", "https://x.com/bro_store", "X"},
		// The four rows that used to render as four identical `web` chips on
		// ブロッコリー's page.
		{"web is named by its host, not by its key", "web", "https://en.wikipedia.org/wiki/Broccoli_(company)", "维基百科"},
		{"a second language subdomain lands on the same name", "web", "https://ja.wikipedia.org/wiki/ブロッコリー", "维基百科"},
		{"wikidata is its own site", "web", "https://www.wikidata.org/wiki/Q758547", "Wikidata"},
		{"youtube is its own site", "web", "https://www.youtube.com/@BROCCOLI2009", "YouTube"},
		{"a subdomain host matches exactly", "web", "https://gamefaqs.gamespot.com/company/73062-", "GameFAQs"},
		// The specificity that a map's random iteration order would break.
		{"ci-en beats its parent dlsite.com", "web", "https://ci-en.dlsite.com/creator/1234", "Ci-en"},
		{"and dlsite.com still answers for itself", "web", "https://www.dlsite.com/maniax/circle/profile/=/maker_id/RG12345.html", "DLsite"},
		// Honest rather than invented: an unlisted host is a name already.
		{"an unlisted host falls back to the bare host", "web", "https://example.co.jp/company", "example.co.jp"},
		{"www is not part of a site's name", "web", "https://www.example.com/", "example.com"},
		// Identity anchors: a source and no address.
		{"an anchor with no URL is named by its source", "dlsite", "", "DLsite"},
		{"an unknown anchor is returned as itself, not guessed at", "kusoge_db", "", "kusoge_db"},
		{"an unknown source with a URL is named by the host", "kusoge_db", "https://www.youtube.com/x", "YouTube"},
		{"nothing at all stays nothing", "", "", ""},
	} {
		if got := LinkDisplayName(tc.source, tc.url); got != tc.want {
			t.Errorf("%s: LinkDisplayName(%q, %q) = %q, want %q", tc.why, tc.source, tc.url, got, tc.want)
		}
	}
}

func TestLinkHostRejectsWhatIsNotAURL(t *testing.T) {
	// An identity anchor's external id must never be printed as a host: it
	// would read as a site the link does not point at.
	for _, in := range []string{"", "RJ249792", "v27920", "1234", "not a url"} {
		if got := linkHost(in); got != "" {
			t.Errorf("linkHost(%q) = %q, want \"\"", in, got)
		}
	}
}
