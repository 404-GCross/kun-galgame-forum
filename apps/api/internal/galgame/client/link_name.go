package client

import "strings"

var linkSourceName = map[string]string{
	"official_site": "官方网站",
	"twitter":       "X",
	"cien":          "Ci-en",
	"steam":         "Steam",
	"pixiv":         "pixiv",
	"vndb":          "VNDB",
	"bangumi":       "Bangumi",
	"erogamescape":  "批评空间",
	"dlsite":        "DLsite",
	"dmm":           "DMM",
	"getchu":        "Getchu",
}

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
	{"erogamescape.dyndns.org", "批评空间"},
	{"facebook.com", "Facebook"},
	{"instagram.com", "Instagram"},
	{"twitch.tv", "Twitch"},
	{"soundcloud.com", "SoundCloud"},
	{"tumblr.com", "Tumblr"},
	{"itch.io", "itch.io"},
	{"github.com", "GitHub"},
}

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
