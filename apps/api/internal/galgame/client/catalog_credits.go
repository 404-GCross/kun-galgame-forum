package client

import (
	"slices"
	"sort"
	"strings"

	"kun-galgame-api/internal/galgame/dto"
)

var staffRoleFold = map[string]string{
	"剧本":                 "scenario",
	"原画":                 "illustration",
	"音乐":                 "music",
	"director-direction": "director",
}

var staffRoleName = map[string]string{
	"scenario":     "脚本",
	"illustration": "原画",
	"music":        "音乐",
	"director":     "导演",
}

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

var staffRoleHidden = map[string]bool{"developer": true, "publisher": true}

const staffRoleLast = "other-staff"

const StaffRoleOtherKey = staffRoleLast

func catalogStaffFromCredits(groups []catCreditGroup) []dto.NextMoeStaffGroup {
	type bucket struct {
		name   string
		people []dto.NextMoeStaffName
		at     map[string]int
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
				b.people[i].Name = c.Name
			}
			if c.Character != "" {
				b.people[i].Characters = appendUniqueStr(b.people[i].Characters, c.Character)
			}
		}
	}

	if other := byKey[staffRoleLast]; other != nil {
		elsewhere := make(map[string]bool)
		for key, b := range byKey {
			if key == staffRoleLast {
				continue
			}
			for norm := range b.at {
				elsewhere[norm] = true
			}
		}
		kept := other.people[:0]
		for _, p := range other.people {
			if !elsewhere[normalizeCreditName(p.Name)] {
				kept = append(kept, p)
			}
		}
		other.people = kept
	}

	rank := make(map[string]int, len(staffRoleDisplayOrder))
	for i, key := range staffRoleDisplayOrder {
		rank[key] = i
	}
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

func StaffRoleCanonicalKey(roleKey string) string {
	if folded, ok := staffRoleFold[roleKey]; ok {
		return folded
	}
	return roleKey
}

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
