package client

import "strings"

// Naming an external link, for every face that carries one.
//
// Works, labels (会社) and persons all store their web presence in the SAME
// catalog_external_ref rows under the same six source keys, and the catalog
// renders all three through one template table. The forum used to name them
// three different ways — the work face printed the raw key, the label face had
// a three-entry Chinese map in the frontend, the person face named links from
// its own URL-template table — so the same wikipedia row read as `web` on a
// galgame page and as `web` again, four times over, on its maker's page.
//
// The whole vocabulary is: official_site / twitter / cien / steam / pixiv,
// plus `web`, which is the catalog's explicit "a link whose host we do not
// model" — its external id IS the full URL. That last one is why a source→名字
// table cannot be the whole answer: ~35 whitelisted sites (wikipedia, youtube,
// wikidata, gamefaqs…) all arrive under the single key `web`, so the site
// identity lives in the URL and nowhere else. Hence: name by source when the
// source says something, otherwise name by host.
//
// Identity anchors (vndb / bangumi / dlsite / …) are a different lane — they
// are not web presences and they carry a bare external id rather than a URL —
// but they are named for a reader out of the same table, so a page never shows
// a bare lowercase `dlsite`.

// linkSourceName is the source key → display name table. `web` is deliberately
// ABSENT: it carries no site identity, so it must fall through to the host.
var linkSourceName = map[string]string{
	"official_site": "官方网站",
	"twitter":       "X",
	"cien":          "Ci-en",
	"steam":         "Steam",
	"pixiv":         "pixiv",
	"vndb":          "VNDB",
	"bangumi":       "Bangumi",
	"erogamescape":  "ErogameScape",
	"dlsite":        "DLsite",
	"dmm":           "DMM",
	"getchu":        "Getchu",
}

// linkHostName is the host → display name table, consulted for `web` links and
// for any source key the table above does not know.
//
// An ORDERED slice, not a map: entries match on suffix so a language-subdomain
// host (en.wikipedia.org) hits its registrable domain, and that makes the more
// specific entries order-dependent — ci-en.dlsite.com must be tried before
// dlsite.com. Map iteration order would make the answer random.
//
// The list is not exhaustive and is not meant to be: an unlisted host falls
// back to the bare host, which is a name a reader can already recognise.
var linkHostName = []struct{ suffix, name string }{
	{"ci-en.dlsite.com", "Ci-en"},
	{"wikipedia.org", "维基百科"},
	{"wikidata.org", "Wikidata"},
	{"youtube.com", "YouTube"},
	{"nicovideo.jp", "niconico"},
	{"x.com", "X"},
	{"twitter.com", "X"},
	{"pixiv.net", "pixiv"},
	{"fanbox.cc", "pixivFANBOX"},
	{"fantia.jp", "Fantia"},
	{"patreon.com", "Patreon"},
	{"booth.pm", "BOOTH"},
	{"melonbooks.co.jp", "Melonbooks"},
	{"steampowered.com", "Steam"},
	{"dlsite.com", "DLsite"},
	{"dmm.co.jp", "DMM"},
	{"getchu.com", "Getchu"},
	{"gamefaqs.gamespot.com", "GameFAQs"},
	{"mobygames.com", "MobyGames"},
	{"anidb.net", "AniDB"},
	{"myanimelist.net", "MyAnimeList"},
	{"bgm.tv", "Bangumi"},
	{"bangumi.tv", "Bangumi"},
	{"vndb.org", "VNDB"},
	{"erogamescape.dyndns.org", "ErogameScape"},
	{"facebook.com", "Facebook"},
	{"instagram.com", "Instagram"},
	{"twitch.tv", "Twitch"},
	{"soundcloud.com", "SoundCloud"},
	{"tumblr.com", "Tumblr"},
	{"itch.io", "itch.io"},
	{"github.com", "GitHub"},
}

// LinkDisplayName names one external link. rawURL may be empty — an identity
// anchor has a source and no address — in which case only the source table can
// answer, and an unknown source is returned as itself rather than dressed up.
func LinkDisplayName(source, rawURL string) string {
	if name, ok := linkSourceName[strings.ToLower(strings.TrimSpace(source))]; ok {
		return name
	}
	if host := linkHost(rawURL); host != "" {
		for _, entry := range linkHostName {
			if host == entry.suffix || strings.HasSuffix(host, "."+entry.suffix) {
				return entry.name
			}
		}
		return host
	}
	if source != "" {
		return source
	}
	return rawURL
}

// linkHost reduces a URL to its bare host: no scheme, no port, no leading www.
// A value that is not a URL at all (an identity anchor's external id, say)
// yields "", so the caller falls back rather than printing a fragment of it.
func linkHost(rawURL string) string {
	host := strings.TrimSpace(rawURL)
	if host == "" {
		return ""
	}
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	} else if !strings.Contains(host, ".") {
		return ""
	}
	if i := strings.IndexAny(host, "/?#"); i >= 0 {
		host = host[:i]
	}
	if i := strings.IndexByte(host, '@'); i >= 0 {
		host = host[i+1:]
	}
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	host = strings.ToLower(strings.TrimPrefix(strings.ToLower(host), "www."))
	if !strings.Contains(host, ".") {
		return ""
	}
	return host
}
