package service

import (
	"context"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type StaffService struct {
	galgameClient *client.GalgameClient
	enricher      *GalgameEnricher
}

func NewStaffService(galgameClient *client.GalgameClient, enricher *GalgameEnricher) *StaffService {
	return &StaffService{galgameClient: galgameClient, enricher: enricher}
}

func (s *StaffService) Search(ctx context.Context, rawQuery url.Values) ([]dto.TaxonomySearchItem, *errors.AppError) {
	return searchCatalogEntities(ctx, s.galgameClient, "names", rawQuery)
}

var staffPersonPage = map[string]func(string) string{
	"vndb": func(id string) string {
		return "https://vndb.org/s" + id
	},
	"bangumi": func(id string) string {
		return "https://bgm.tv/person/" + id
	},
	"erogamescape": func(id string) string {
		return "https://erogamescape.dyndns.org/~ap2/ero/toukei_kaiseki/creater.php?creater=" + id
	},
}

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
	if movedTo != 0 {
		return &dto.StaffDetail{MovedTo: int(movedTo)}, nil
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该制作人员")
	}

	rendered, original := client.CatalogEntityNames(ctx, name.Localized, name.DisplayName, name.Latin)
	intro := preferredIntro(name.Intros)

	detail := &dto.StaffDetail{
		ID:           int(name.ID),
		Name:         rendered,
		NameOriginal: original,
		Latin:        name.Latin,
		Intro:        intro.Intro,
		IntroMachine: intro.Machine,
		Photo:        s.galgameClient.ImageURLFromHash(name.PhotoHash),
		Gender:       name.Gender,
		BirthY:       name.BirthY,
		BirthM:       name.BirthM,
		BirthD:       name.BirthD,
		Links:        staffLinks(name),
		Siblings:     staffSiblings(ctx, name),
		Works:        []dto.StaffWork{},
		NextOffset:   name.NextOffset,
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

	var roleKeys []string
	labelOf := map[string]string{}
	items := make([]dto.NextMoeGalgameItem, 0, len(name.Credits))
	for _, c := range name.Credits {
		row, ok := rows[c.Work.ID]
		if !ok {
			continue
		}
		var onThisWork, characters []string
		for _, r := range c.Roles {
			key := client.StaffRoleCanonicalKey(r.RoleKey)
			labelOf[key] = client.StaffRoleLabel(r.RoleKey, r.RoleName)
			onThisWork = appendUniqueStr(onThisWork, key)
			if r.Character != "" {
				characters = appendUniqueStr(characters, r.Character)
			}
		}
		if len(onThisWork) > 1 {
			onThisWork = slices.DeleteFunc(onThisWork, func(key string) bool {
				return key == client.StaffRoleOtherKey
			})
		}
		for _, key := range onThisWork {
			roleKeys = appendUniqueStr(roleKeys, key)
		}
		labels := make([]string, 0, len(onThisWork))
		for _, key := range client.SortStaffRoleKeys(onThisWork) {
			labels = append(labels, labelOf[key])
		}
		items = append(items, client.CatalogItemToNextMoeItem(ctx, &row))
		detail.Works = append(detail.Works, dto.StaffWork{
			CatalogID:  int(row.ID),
			Roles:      labels,
			Characters: characters,
		})
	}

	for i, card := range s.enricher.ToCards(ctx, items) {
		detail.Works[i].GalgameCard = card
		detail.Works[i].Status = 0
	}

	detail.Roles = make([]string, 0, len(roleKeys))
	for _, key := range client.SortStaffRoleKeys(roleKeys) {
		detail.Roles = append(detail.Roles, labelOf[key])
	}
	return detail, nil
}

func staffLinks(n *client.CatalogName) []dto.StaffLink {
	out := make([]dto.StaffLink, 0, len(n.Refs)+len(n.Links))
	for _, ref := range n.Refs {
		link := dto.StaffLink{
			Source: ref.Source,
			Name:   client.LinkDisplayName(ref.Source, ""),
		}
		if tpl, ok := staffPersonPage[ref.Source]; ok {
			link.URL = tpl(ref.ExternalID)
		}
		out = append(out, link)
	}
	for _, l := range n.Links {
		out = append(out, dto.StaffLink{
			Source: l.Source,
			Name:   client.LinkDisplayName(l.Source, l.URL),
			URL:    l.URL,
		})
	}
	return out
}

func staffSiblings(ctx context.Context, n *client.CatalogName) []dto.StaffSibling {
	out := make([]dto.StaffSibling, 0, len(n.Siblings))
	for _, sib := range n.Siblings {
		out = append(out, dto.StaffSibling{ID: int(sib.ID), Name: sib.Name(ctx)})
	}
	return out
}
