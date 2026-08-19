package service

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type CharacterService struct {
	galgameClient *client.GalgameClient
	enricher      *GalgameEnricher
}

func NewCharacterService(galgameClient *client.GalgameClient, enricher *GalgameEnricher) *CharacterService {
	return &CharacterService{galgameClient: galgameClient, enricher: enricher}
}

func (s *CharacterService) Search(ctx context.Context, rawQuery url.Values) ([]dto.TaxonomySearchItem, *errors.AppError) {
	return searchCatalogEntities(ctx, s.galgameClient, "characters", rawQuery)
}

var characterPage = map[string]func(string) string{
	"vndb": func(id string) string {
		return "https://vndb.org/" + id
	},
	"bangumi": func(id string) string {
		return "https://bgm.tv/character/" + id
	},
}

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
	if movedTo != 0 {
		return &dto.GalgameCharacterDetail{MovedTo: int(movedTo)}, nil
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该角色")
	}

	name, original := client.CatalogEntityNames(ctx, ch.Localized, ch.DisplayName, ch.Latin)

	detail := &dto.GalgameCharacterDetail{
		ID:           int(ch.ID),
		Name:         name,
		NameOriginal: original,
		Latin:        ch.Latin,
		Image:        ch.Image,
		Figure:       ch.Figure,
		ImageMeta:    client.ArtMetaDTO(ch.ImageMeta),
		FigureMeta:   client.ArtMetaDTO(ch.FigureMeta),
		Intros:       characterIntros(ch),
		Traits:       characterTraits(ch, isSFW),
		Links:        characterLinks(ch),
		Works:        []dto.GalgameCharacterWork{},
		NextOffset:   ch.NextOffset,
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

	items := make([]dto.NextMoeGalgameItem, 0, len(ch.Works))
	for _, w := range ch.Works {
		row, ok := rows[w.Work.ID]
		if !ok {
			continue
		}
		voices := make([]dto.GalgameDetailCharacterVoice, 0, len(w.Voices))
		for _, v := range w.Voices {
			voices = append(voices, dto.GalgameDetailCharacterVoice{
				ID: int(v.ID), Name: v.Name(ctx), Lang: v.Lang, Latin: v.Latin,
			})
		}
		items = append(items, client.CatalogItemToNextMoeItem(ctx, &row))
		detail.Works = append(detail.Works, dto.GalgameCharacterWork{
			CatalogID: int(row.ID),
			Voices:    voices,
		})
	}

	for i, card := range s.enricher.ToCards(ctx, items) {
		detail.Works[i].GalgameCard = card
		detail.Works[i].Status = 0
	}
	return detail, nil
}

func characterIntros(ch *client.CatalogCharacter) []dto.GalgameCharacterIntro {
	out := make([]dto.GalgameCharacterIntro, 0, len(ch.Intros))
	for _, i := range catalogIntrosByLang(ch.Intros) {
		out = append(out, dto.GalgameCharacterIntro{
			Lang: i.Lang, Intro: i.Intro, Source: i.Source, Machine: i.Machine,
		})
	}
	return out
}

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

func characterTraits(ch *client.CatalogCharacter, isSFW bool) []dto.GalgameCharacterTrait {
	out := make([]dto.GalgameCharacterTrait, 0, len(ch.Traits))
	for _, t := range ch.Traits {
		if isSFW && t.Sexual {
			continue
		}
		out = append(out, dto.GalgameCharacterTrait{
			ID:      int(t.ID),
			Name:    t.LocalName(),
			Group:   t.LocalGroup(),
			Spoiler: t.Spoiler,
			Lie:     t.Lie,
		})
	}
	return out
}

func characterLinks(ch *client.CatalogCharacter) []dto.GalgameCharacterLink {
	out := make([]dto.GalgameCharacterLink, 0, len(ch.Refs))
	for _, ref := range ch.Refs {
		link := dto.GalgameCharacterLink{
			Source: ref.Source,
			Name:   client.LinkDisplayName(ref.Source, ""),
		}
		if tpl, ok := characterPage[ref.Source]; ok {
			link.URL = tpl(ref.ExternalID)
		}
		out = append(out, link)
	}
	return out
}
