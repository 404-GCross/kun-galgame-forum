package service

// The 制作人员 page: one credited name, everything the registry publishes about
// it, and the games it signed.
//
// Two upstream calls, never more. The credits list arrives with each work
// already identified by catalog id but described only by its registry display
// name — no localized title, no cover — so the whole page's worth of ids is
// hydrated in one batch through the works lane. That lane is also where the
// EDITORIAL content gate lives, which is the second reason to route through it:
// a SFW visitor's filmography must not be assembled from rows the site would
// refuse to render anywhere else.
//
// A work the gate drops simply does not appear. That is the same bargain every
// list on the site makes, and it is why this page never prints a total: the
// catalog publishes none for a credits list, and the number we could compute
// would describe one page after filtering rather than a career.

import (
	"context"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

// StaffService serves the credited-name detail lane.
type StaffService struct {
	galgameClient *client.GalgameClient
	// enricher fuses the catalog rows with the forum's own view/like counts,
	// platform badges and frozen author — the same overlay the entity detail
	// pages and the calendar use, so a filmography card is the site's ordinary
	// galgame card rather than a lookalike.
	enricher *GalgameEnricher
}

func NewStaffService(galgameClient *client.GalgameClient, enricher *GalgameEnricher) *StaffService {
	return &StaffService{galgameClient: galgameClient, enricher: enricher}
}

// staffPersonPage renders one source's person page URL from the identity anchor
// the registry stores. Only templates verified against a live record are here:
// a wrong external link is a worse answer than no link, so an unlisted source
// renders as plain text (see dto.StaffLink.URL).
//
// The stored ids are BARE (doc: vndb staff anchors are bare numbers, unlike
// characters' c-prefix and labels' p-prefix), so the prefix belongs to the
// template, not to the data.
var staffPersonPage = map[string]struct {
	name string
	url  func(string) string
}{
	"vndb": {"VNDB", func(id string) string {
		return "https://vndb.org/s" + id
	}},
	"bangumi": {"Bangumi", func(id string) string {
		return "https://bgm.tv/person/" + id
	}},
	"erogamescape": {"ErogameScape", func(id string) string {
		return "https://erogamescape.dyndns.org/~ap2/ero/toukei_kaiseki/creater.php?creater=" + id
	}},
}

// GetDetail — GET /galgame-staff/:id
func (s *StaffService) GetDetail(
	ctx context.Context, rawID string, offset, limit int, isSFW bool,
) (*dto.StaffDetail, *errors.AppError) {
	id, err := strconv.ParseInt(strings.TrimSpace(rawID), 10, 64)
	if err != nil || id <= 0 {
		return nil, errors.ErrBadRequest("无效的制作人员 ID")
	}
	name, found, movedTo, appErr := s.galgameClient.CatalogNameDetail(ctx, id, limit, offset)
	if appErr != nil {
		return nil, appErr
	}
	// A folded name (wave 171) keeps its old id addressable as a 301: moved_to
	// arrives instead of the record, never alongside it, so nothing of the
	// survivor is ever painted under the dead id.
	if movedTo != 0 {
		return &dto.StaffDetail{MovedTo: int(movedTo)}, nil
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该制作人员")
	}

	detail := &dto.StaffDetail{
		ID:     int(name.ID),
		Name:   staffDisplayName(name),
		NameJa: name.Name.JA,
		NameZh: name.Name.ZH,
		Latin:  name.Latin,
		Intro:  staffIntro(name),
		// Pure passthrough, with no policy of its own: the registry has already
		// applied the person-link visibility gate, and a name whose link is
		// private arrives with all five zeroed. Re-deciding anything here could
		// only ever loosen a deliberate decision made upstream.
		Photo:      s.galgameClient.ImageURLFromHash(name.PhotoHash),
		Gender:     name.Gender,
		BirthY:     name.BirthY,
		BirthM:     name.BirthM,
		BirthD:     name.BirthD,
		Links:      staffLinks(name),
		Siblings:   staffSiblings(name),
		Works:      []dto.StaffWork{},
		NextOffset: name.NextOffset,
	}

	ids := make([]int64, 0, len(name.Credits))
	for _, c := range name.Credits {
		ids = append(ids, c.Work.ID)
	}
	if len(ids) == 0 {
		detail.Roles = []string{}
		return detail, nil
	}

	rows, appErr := s.galgameClient.CatalogRowsByCatalogIDs(ctx, ids, isSFW)
	if appErr != nil {
		return nil, appErr
	}

	// Roles are accumulated as canonical KEYS and rendered at the end: the
	// display order is defined over keys, and only four roles carry a pinned
	// label to map back from.
	var roleKeys []string
	labelOf := map[string]string{}
	// The cards are built in one batch at the end rather than row by row: the
	// filmography renders through the SHARED galgame card, and the local
	// enrichment behind it (views, likes, platform badges, the frozen author)
	// is a set of batch lookups that must see the whole page at once.
	items := make([]dto.NextMoeGalgameItem, 0, len(name.Credits))
	for _, c := range name.Credits {
		row, ok := rows[c.Work.ID]
		if !ok {
			continue // dropped by the editorial gate, or gone from the registry
		}
		var onThisWork, characters []string
		for _, r := range c.Roles {
			key := client.StaffRoleCanonicalKey(r.RoleKey)
			labelOf[key] = client.StaffRoleLabel(r.RoleKey, r.RoleName)
			roleKeys = appendUniqueStr(roleKeys, key)
			onThisWork = appendUniqueStr(onThisWork, key)
			// One credit per character voiced, so a VA arrives several times on
			// the same work. Collect the cast onto the one card rather than
			// repeating the 声优 chip — for a voice actor this IS the credit.
			if r.Character != "" {
				characters = appendUniqueStr(characters, r.Character)
			}
		}
		// Sorted per card too, not just in the header: unordered, every card led
		// with 其他 — the unmapped bucket is the one role nearly every credit
		// carries, and it was burying the 脚本 the reader is here for.
		labels := make([]string, 0, len(onThisWork))
		for _, key := range client.SortStaffRoleKeys(onThisWork) {
			labels = append(labels, labelOf[key])
		}
		items = append(items, client.CatalogItemToNextMoeItem(&row))
		detail.Works = append(detail.Works, dto.StaffWork{
			CatalogID:  int(row.ID),
			Roles:      labels,
			Characters: characters,
		})
	}

	// ToCards answers one card per item, in order, so the overlays built above
	// line up index for index.
	for i, card := range s.enricher.ToCards(ctx, items) {
		detail.Works[i].GalgameCard = card
		// The claim funnel is the calendar's, not this page's. Left alone, every
		// unclaimed work would paint itself as a 未在论坛发布 call to action —
		// and a filmography is mostly decades-old games, so the whole grid would
		// shout it and bury the works the forum actually has. Those rows stay
		// the neutral 未收录 card that this page has always shown.
		detail.Works[i].Status = 0
	}

	detail.Roles = make([]string, 0, len(roleKeys))
	for _, key := range client.SortStaffRoleKeys(roleKeys) {
		detail.Roles = append(detail.Roles, labelOf[key])
	}
	return detail, nil
}

// staffDisplayName picks the one name the page is titled with. Japanese first:
// this is a Japanese-medium registry and the ja bucket is the form the person
// actually signs with — the zh bucket is a translation and the latin is a
// transliteration, both useful as subtitles and wrong as the headline.
func staffDisplayName(n *client.CatalogName) string {
	return client.PickCatalogName(n.Name, n.Latin)
}

// staffIntro picks ONE description. The registry keeps a row per language and
// stacking them under a name reads as a bug — the same call the series page
// already made. Chinese first here (unlike the display name): a description is
// read, not identified by.
func staffIntro(n *client.CatalogName) string {
	byLang := make(map[string]string, len(n.Intros))
	for _, i := range n.Intros {
		if strings.TrimSpace(i.Intro) == "" {
			continue
		}
		if _, seen := byLang[i.Lang]; !seen {
			byLang[i.Lang] = i.Intro
		}
	}
	for _, lang := range []string{"zh-Hans", "zh", "zh-Hant", "ja", "en"} {
		if intro, ok := byLang[lang]; ok {
			return intro
		}
	}
	for _, i := range n.Intros {
		if strings.TrimSpace(i.Intro) != "" {
			return i.Intro
		}
	}
	return ""
}

func staffLinks(n *client.CatalogName) []dto.StaffLink {
	out := make([]dto.StaffLink, 0, len(n.Refs))
	for _, ref := range n.Refs {
		link := dto.StaffLink{Source: ref.Source, Name: ref.Source}
		if tpl, ok := staffPersonPage[ref.Source]; ok {
			link.Name, link.URL = tpl.name, tpl.url(ref.ExternalID)
		}
		out = append(out, link)
	}
	return out
}

func staffSiblings(n *client.CatalogName) []dto.StaffSibling {
	out := make([]dto.StaffSibling, 0, len(n.Siblings))
	for _, sib := range n.Siblings {
		out = append(out, dto.StaffSibling{ID: int(sib.ID), Name: client.PickCatalogName(sib.Name, sib.Latin)})
	}
	return out
}
