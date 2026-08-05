package client

// Projection of the catalog's credit groups onto the 制作人员 panel.
//
// The catalog answers with the registry's own shape: one group per role id, in
// role-id order, one row per signing edge. That is the right shape for a
// registry and the wrong one for a reader, for three reasons this file exists
// to fix:
//
//   - The role vocabulary is GENERATED from the Bangumi staff positions and
//     then hand-extended, so the same job exists twice under two keys. A game
//     credits 一河のあ under both `illustration`(插画) and `原画`, and 雪仁 under
//     both `scenario`(脚本) and `剧本` — upstream is not wrong, the two rows come
//     from two sources, but printing both is.
//   - The same human is credited under several spellings ("水城 新人",
//     "水城新人(獅子王院みづき、新澄トキ)"). The public credits face publishes the
//     credit-NAME id only, never person_id, so the merge cannot be done by
//     identity here; it is done on the written form, conservatively.
//   - Role-id order is the vocabulary's insertion order, which puts 声优 (53
//     rows) above 脚本 (one row). A reader wants the authorship first.
//
// Everything below is display policy, deliberately kept out of the wire types:
// nothing here changes what the catalog said, only which of it the panel shows
// first.

import (
	"slices"
	"sort"
	"strings"

	"kun-galgame-api/internal/galgame/dto"
)

// staffRoleFold maps a role key onto the key it is DISPLAYED under. Only truly
// duplicate positions are folded — each pair below is the same job reached from
// two source vocabularies, and empirically resolves to the same people on the
// same work. Roles that merely feel adjacent (作曲 vs 音乐, 监修 vs 导演) are left
// alone: they are distinct credits and readers of this genre read them as such.
var staffRoleFold = map[string]string{
	"剧本":                 "scenario",     // 剧本 == 脚本
	"原画":                 "illustration", // 原画 == 插画 in this medium
	"音乐":                 "music",
	"director-direction": "director", // both localize to 导演
}

// staffRoleName pins the label a folded group renders under, so the panel reads
// the same whether the work was credited through one source vocabulary or both.
// Only folded keys need an entry; every other group keeps the catalog's label.
var staffRoleName = map[string]string{
	"scenario":     "脚本",
	"illustration": "原画",
	"music":        "音乐",
	"director":     "导演",
}

// staffRoleDisplayOrder is the head of the panel: the six credits a reader
// actually came for first (writer, artist, character designer, composer, cast),
// then the rest of the sound department, then everything unranked. A role not
// named here keeps the catalog's own order behind this list — the tail is small
// (kungal's whole live population uses 45 distinct roles) and mostly
// single-credit oddities.
//
// The cut at six is not arbitrary: the median live game credits exactly six
// roles once folded, so the collapsed panel shows the whole list for half the
// catalogue and this exact head for the other half. 声优 must stay inside it —
// it is the single most-read group on the page and it would otherwise fall
// behind the fold for every game with a full sound crew.
var staffRoleDisplayOrder = []string{
	"原作",
	"scenario",
	"illustration",
	"character-design",
	"music",
	"voice-actor",
	"director",
	"composer",
	"lyric",
	"arrange",
	"vocal",
	"theme-song-composition",
	"theme-song-lyrics",
	"theme-song-performance",
	"inserted-song-performance",
}

// staffRoleHidden drops the two roles the detail page already answers better
// elsewhere: 开发 / 发行 are brand credits, and the 会社 card renders exactly the
// same organisations WITH their per-work role and a link to their catalogue
// page. Printing "开发: アンモライト" a second time as flat text is noise.
var staffRoleHidden = map[string]bool{"developer": true, "publisher": true}

// staffRoleLast is pinned to the bottom whatever its volume: `other-staff` is
// the unmapped bucket (studios, contractors, one-off jobs) and is routinely the
// longest group on the page.
const staffRoleLast = "other-staff"

// catalogStaffFromCredits folds, dedupes and orders a work's credit groups.
func catalogStaffFromCredits(groups []catCreditGroup) []dto.NextMoeStaffGroup {
	type bucket struct {
		name   string
		people []dto.NextMoeStaffName
		// at indexes people by normalized name so a second spelling merges into
		// the first row rather than adding one.
		at map[string]int
	}
	order := make([]string, 0, len(groups))
	byKey := make(map[string]*bucket, len(groups))

	for _, g := range groups {
		if staffRoleHidden[g.RoleKey] {
			continue
		}
		key := g.RoleKey
		if folded, ok := staffRoleFold[key]; ok {
			key = folded
		}
		b := byKey[key]
		if b == nil {
			b = &bucket{name: g.RoleName, at: map[string]int{}}
			byKey[key] = b
			order = append(order, key)
		}
		if pinned, ok := staffRoleName[key]; ok {
			b.name = pinned
		} else if b.name == "" {
			b.name = g.RoleName
		}
		for _, c := range g.Credits {
			norm := normalizeCreditName(c.Name)
			if norm == "" {
				continue
			}
			i, seen := b.at[norm]
			if !seen {
				b.at[norm] = len(b.people)
				b.people = append(b.people, dto.NextMoeStaffName{
					ID: int(c.ID), Name: c.Name, Latin: c.Latin,
				})
				i = len(b.people) - 1
			} else if len(c.Name) < len(b.people[i].Name) {
				// Prefer the plainest spelling: "水城新人" over
				// "水城新人(獅子王院みづき、新澄トキ)". The alias in the parenthesis is
				// real information, but it is the identity layer's to publish,
				// not a staff list's.
				b.people[i].Name = c.Name
			}
			if c.Character != "" {
				b.people[i].Characters = appendUniqueStr(b.people[i].Characters, c.Character)
			}
		}
	}

	rank := make(map[string]int, len(staffRoleDisplayOrder))
	for i, key := range staffRoleDisplayOrder {
		rank[key] = i
	}
	// Unranked roles sort after every ranked one and before other-staff, each
	// keeping its arrival position so the panel is stable across requests.
	weight := func(key string, arrival int) (int, int) {
		if key == staffRoleLast {
			return len(staffRoleDisplayOrder) + 1, arrival
		}
		if r, ok := rank[key]; ok {
			return r, arrival
		}
		return len(staffRoleDisplayOrder), arrival
	}
	arrival := make(map[string]int, len(order))
	for i, key := range order {
		arrival[key] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		ai, bi := weight(order[i], arrival[order[i]])
		aj, bj := weight(order[j], arrival[order[j]])
		if ai != aj {
			return ai < aj
		}
		return bi < bj
	})

	out := make([]dto.NextMoeStaffGroup, 0, len(order))
	for _, key := range order {
		b := byKey[key]
		if len(b.people) == 0 {
			continue
		}
		out = append(out, dto.NextMoeStaffGroup{RoleKey: key, RoleName: b.name, People: b.people})
	}
	return out
}

// normalizeCreditName is the merge key for two written forms of one credit.
// It drops every space (sources disagree on whether a Japanese name carries
// one) and everything from the first bracket on (the alias annotation). It is
// deliberately conservative: it can only merge names that are already equal
// once punctuation is gone, so it never collapses two different people.
func normalizeCreditName(name string) string {
	if i := strings.IndexAny(name, "(（"); i >= 0 {
		name = name[:i]
	}
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '　' {
			return -1
		}
		return r
	}, name)
}

// StaffRoleCanonicalKey folds a role key onto the key it is DISPLAYED under,
// so the two faces that render credits — a game's 制作人员 panel and a person's
// filmography — group the same way. Anything not folded is already canonical.
func StaffRoleCanonicalKey(roleKey string) string {
	if folded, ok := staffRoleFold[roleKey]; ok {
		return folded
	}
	return roleKey
}

// StaffRoleLabel is the label a role renders under anywhere on the site: the
// two faces must not disagree about whether 一河のあ did 插画 or 原画. A folded
// key resolves to the pinned label; everything else keeps the catalog's own,
// which already arrives localized.
func StaffRoleLabel(roleKey, roleName string) string {
	key := StaffRoleCanonicalKey(roleKey)
	if pinned, ok := staffRoleName[key]; ok {
		return pinned
	}
	if roleName != "" {
		return roleName
	}
	return key
}

// SortStaffRoleKeys orders canonical role KEYS by the same authorship-first
// ranking the panel uses, so a person's headline reads 脚本 · 原画 · 声优 rather
// than in whatever order the registry happened to return them.
//
// It ranks keys, not labels: only four roles carry a pinned Chinese label, so a
// label cannot be mapped back to its position — sorting the rendered strings
// silently left every unpinned role unranked.
func SortStaffRoleKeys(keys []string) []string {
	rank := make(map[string]int, len(staffRoleDisplayOrder))
	for i, key := range staffRoleDisplayOrder {
		rank[key] = i
	}
	weight := func(key string) int {
		if key == staffRoleLast {
			return len(staffRoleDisplayOrder) + 1
		}
		if r, ok := rank[key]; ok {
			return r
		}
		return len(staffRoleDisplayOrder)
	}
	out := slices.Clone(keys)
	sort.SliceStable(out, func(i, j int) bool { return weight(out[i]) < weight(out[j]) })
	return out
}
