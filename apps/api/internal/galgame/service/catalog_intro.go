package service

import (
	"strings"

	"kun-galgame-api/internal/galgame/client"
)

// catalogIntroLangs is the order a Chinese reader wants an entity blurb in.
var catalogIntroLangs = []string{"zh-Hans", "zh", "zh-Hant", "ja", "en"}

// catalogIntrosByLang keeps the first row per language and drops blank ones.
// First-per-language IS the election: the public face already ranks the rows of
// one language — source provenance wins, then the derived extraction, then a
// translation — so re-picking here would only be a chance to get it wrong.
func catalogIntrosByLang(intros []client.CatalogIntro) []client.CatalogIntro {
	out := make([]client.CatalogIntro, 0, len(intros))
	seen := make(map[string]bool, len(intros))
	for _, in := range intros {
		if strings.TrimSpace(in.Intro) == "" || seen[in.Lang] {
			continue
		}
		seen[in.Lang] = true
		out = append(out, in)
	}
	return out
}

// preferredIntro returns the row to show, blank when the entity has no blurb at
// all. It returns the row rather than its text because the caller has to be
// able to tell the reader that a machine wrote it.
func preferredIntro(intros []client.CatalogIntro) client.CatalogIntro {
	byLang := catalogIntrosByLang(intros)
	for _, lang := range catalogIntroLangs {
		for _, in := range byLang {
			if in.Lang == lang {
				return in
			}
		}
	}
	if len(byLang) > 0 {
		return byLang[0]
	}
	return client.CatalogIntro{}
}
