package service

// The 角色 page: one character, everything the registry publishes about her,
// and every game she appears in.
//
// Structurally the twin of the 制作人员 page (see staff_service.go) and for the
// same reasons: two upstream calls at most, and the appearance list is hydrated
// through the works LANE rather than fetched record by record, because that is
// where the editorial content gate lives. A work the gate drops simply does not
// appear, and the page prints no total — the catalog publishes none for an
// offset list, and a number computed here would describe one filtered page
// rather than a career.
//
// The one policy this service owns that the staff page does not: the catalog is
// an R18 face and ships VNDB's sexual-family traits to anyone who unlocks the
// population. Unlocking it is unavoidable (94.5% of the registry is r18 — a
// closed gate would delete most of her appearances, not filter them), so the
// traits are gated back down HERE, against the reader's own SFW switch. That
// decision belongs on the server; the browser only gets what it may render.

import (
	"context"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

// CharacterService serves the character detail lane.
type CharacterService struct {
	galgameClient *client.GalgameClient
	// enricher fuses the catalog rows with the forum's own view/like counts,
	// platform badges and frozen author, so an appearance is the site's
	// ordinary galgame card rather than a lookalike.
	enricher *GalgameEnricher
}

func NewCharacterService(galgameClient *client.GalgameClient, enricher *GalgameEnricher) *CharacterService {
	return &CharacterService{galgameClient: galgameClient, enricher: enricher}
}

// characterPage renders one source's character page URL. Only templates
// verified against a live record are here: a wrong external link is a worse
// answer than no link, so an unlisted source renders as plain text.
//
// The VNDB id arrives WITH its `c` prefix (unlike a staff anchor, which is a
// bare number), so the prefix belongs to the data here, not to the template.
var characterPage = map[string]struct {
	name string
	url  func(string) string
}{
	"vndb": {"VNDB", func(id string) string {
		return "https://vndb.org/" + id
	}},
	"bangumi": {"Bangumi", func(id string) string {
		return "https://bgm.tv/character/" + id
	}},
}

// GetDetail — GET /galgame-character/:id
//
// withWorks off is the game page's modal: identity, art and traits, without the
// appearance list it does not render.
func (s *CharacterService) GetDetail(
	ctx context.Context, rawID string, offset, limit int, isSFW, withWorks bool,
) (*dto.GalgameCharacterDetail, *errors.AppError) {
	id, err := strconv.ParseInt(strings.TrimSpace(rawID), 10, 64)
	if err != nil || id <= 0 {
		return nil, errors.ErrBadRequest("无效的角色 ID")
	}
	ch, found, movedTo, appErr := s.galgameClient.CatalogCharacterDetail(ctx, id, limit, offset, withWorks)
	if appErr != nil {
		return nil, appErr
	}
	// A merged character keeps its old id addressable as a 301: moved_to
	// arrives instead of the record, never alongside it, so nothing of the
	// survivor is ever painted under the dead id.
	if movedTo != 0 {
		return &dto.GalgameCharacterDetail{MovedTo: int(movedTo)}, nil
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该角色")
	}

	detail := &dto.GalgameCharacterDetail{
		ID:         int(ch.ID),
		Name:       client.PickCatalogName(ch.Name, ch.Latin),
		NameJa:     ch.Name.JA,
		NameZh:     ch.Name.ZH,
		Latin:      ch.Latin,
		Image:      ch.Image,
		Figure:     ch.Figure,
		ImageMeta:  client.ArtMetaDTO(ch.ImageMeta),
		FigureMeta: client.ArtMetaDTO(ch.FigureMeta),
		Intros:     characterIntros(ch),
		Traits:     characterTraits(ch, isSFW),
		Links:      characterLinks(ch),
		Works:      []dto.GalgameCharacterWork{},
		NextOffset: ch.NextOffset,
	}
	detail.Intro = pickCharacterIntro(detail.Intros)

	ids := make([]int64, 0, len(ch.Works))
	for _, w := range ch.Works {
		ids = append(ids, w.Work.ID)
	}
	if len(ids) == 0 {
		return detail, nil
	}

	rows, appErr := s.galgameClient.CatalogRowsByCatalogIDs(ctx, ids, isSFW)
	if appErr != nil {
		return nil, appErr
	}

	// Built in one batch at the end rather than row by row: the appearances
	// render through the SHARED galgame card, and the local enrichment behind
	// it is a set of batch lookups that must see the whole page at once.
	items := make([]dto.NextMoeGalgameItem, 0, len(ch.Works))
	for _, w := range ch.Works {
		row, ok := rows[w.Work.ID]
		if !ok {
			continue // dropped by the editorial gate, or gone from the registry
		}
		voices := make([]dto.GalgameDetailCharacterVoice, 0, len(w.Voices))
		for _, v := range w.Voices {
			voices = append(voices, dto.GalgameDetailCharacterVoice{
				ID: int(v.ID), Name: v.Name, Lang: v.Lang, Latin: v.Latin,
			})
		}
		items = append(items, client.CatalogItemToNextMoeItem(&row))
		detail.Works = append(detail.Works, dto.GalgameCharacterWork{
			CatalogID: int(row.ID),
			Voices:    voices,
		})
	}

	// ToCards answers one card per item, in order, so the overlays built above
	// line up index for index.
	for i, card := range s.enricher.ToCards(ctx, items) {
		detail.Works[i].GalgameCard = card
		// The claim funnel is the calendar's, not this page's — same call the
		// filmography makes. Left alone, every unclaimed work would paint
		// itself as a 未在论坛发布 call to action, and an appearance list is
		// mostly old games, so the whole grid would shout it.
		detail.Works[i].Status = 0
	}
	return detail, nil
}

// characterIntros keeps ONE description per language: the registry can hold
// several rows for a language and stacking them under a character reads as a
// bug, the same call the staff and series pages already made.
func characterIntros(ch *client.CatalogCharacter) []dto.GalgameCharacterIntro {
	out := make([]dto.GalgameCharacterIntro, 0, len(ch.Intros))
	seen := make(map[string]bool, len(ch.Intros))
	for _, i := range ch.Intros {
		if strings.TrimSpace(i.Intro) == "" || seen[i.Lang] {
			continue
		}
		seen[i.Lang] = true
		out = append(out, dto.GalgameCharacterIntro{
			Lang: i.Lang, Intro: i.Intro, Source: i.Source, Machine: i.Machine,
		})
	}
	return out
}

// pickCharacterIntro chooses the one description the page leads with. Chinese
// first (unlike the display name, which is Japanese first): a name is
// identified by, a bio is read.
func pickCharacterIntro(intros []dto.GalgameCharacterIntro) string {
	for _, lang := range []string{"zh-Hans", "zh", "zh-Hant", "ja", "en"} {
		for _, i := range intros {
			if i.Lang == lang {
				return i.Intro
			}
		}
	}
	if len(intros) > 0 {
		return intros[0].Intro
	}
	return ""
}

// characterTraits passes the trait set through, minus the sexual family for a
// SFW reader — see the file header. Spoiler levels are NOT filtered here: every
// tier travels so the frontend's reveal costs no second request.
func characterTraits(ch *client.CatalogCharacter, isSFW bool) []dto.GalgameCharacterTrait {
	out := make([]dto.GalgameCharacterTrait, 0, len(ch.Traits))
	for _, t := range ch.Traits {
		if isSFW && t.Sexual {
			continue
		}
		out = append(out, dto.GalgameCharacterTrait{
			ID: int(t.ID), Name: t.Name, Group: t.Group, Spoiler: t.Spoiler, Lie: t.Lie,
		})
	}
	return out
}

func characterLinks(ch *client.CatalogCharacter) []dto.GalgameCharacterLink {
	out := make([]dto.GalgameCharacterLink, 0, len(ch.Refs))
	for _, ref := range ch.Refs {
		link := dto.GalgameCharacterLink{Source: ref.Source, Name: ref.Source}
		if tpl, ok := characterPage[ref.Source]; ok {
			link.Name, link.URL = tpl.name, tpl.url(ref.ExternalID)
		}
		out = append(out, link)
	}
	return out
}
