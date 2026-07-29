package service

import (
	"context"
	"maps"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

// OfficialService serves the 会社 browse / search / detail lanes.
//
// The catalog calls this entity a LABEL; the kungal product wording (会社) is
// unchanged, so the rename shows up only in the upstream path and the new URL
// space. Two field-level notes:
//
//   - `category` is now the label KIND (game_brand / publisher / doujin_circle
//     …), a richer and better-defined vocabulary than the wiki's free-text one.
//   - `original` (the wiki's original-language name field) has no catalog
//     counterpart on the public face. It survives only where it is still
//     WRITTEN — the staff edit form, which reads it back off the /api face.
type OfficialService struct {
	galgameClient *client.GalgameClient
	// galgameSvc runs the shared local filter/sort/paginate + hydration flow
	// over the label's member ids (the catalog can't filter by kungal-local
	// resource data). See GetDetail.
	galgameSvc *GalgameService
}

func NewOfficialService(galgameClient *client.GalgameClient, galgameSvc *GalgameService) *OfficialService {
	return &OfficialService{galgameClient: galgameClient, galgameSvc: galgameSvc}
}

// GetList — GET /galgame-official
func (s *OfficialService) GetList(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) (*dto.OfficialListPage, *errors.AppError) {
	base := url.Values{}
	if !isSFW {
		base.Set("nsfw", "1")
	}
	if kind := rawQuery.Get("kind"); kind != "" {
		base.Set("kind", kind)
	}
	rows, total, appErr := s.galgameClient.CatalogTaxonomyPageAt(ctx, "labels", base,
		atoiOr(rawQuery.Get("page"), 1), atoiOr(rawQuery.Get("limit"), 50))
	if appErr != nil {
		return nil, appErr
	}

	items := make([]dto.OfficialListItem, 0, len(rows))
	for _, o := range rows {
		items = append(items, dto.OfficialListItem{
			ID:   int(o.ID),
			Name: o.Label(),
			// The browse row is identity + count; the maker's website lives on
			// the record (links[]), which the detail page fetches. The list
			// cards never rendered a website anyway.
			Category:     o.Kind,
			Alias:        emptyStrSliceIfNil(o.Aliases),
			GalgameCount: o.WorkCount,
		})
	}
	return &dto.OfficialListPage{Officials: items, Total: total}, nil
}

// Search — GET /galgame-official/search
//
// The frontend (galgame/official/Container.vue) does `searchResult.value = res`
// expecting a bare GalgameOfficialItem[], so the envelope is unwrapped here and
// the gateway response shape stays put.
func (s *OfficialService) Search(
	ctx context.Context,
	rawQuery url.Values,
	isSFW bool,
) ([]dto.OfficialListItem, *errors.AppError) {
	hits, appErr := s.galgameClient.CatalogEntitySearch(ctx, "labels",
		rawQuery.Get("q"), atoiOr(rawQuery.Get("limit"), 20), isSFW)
	if appErr != nil {
		return nil, appErr
	}
	items := make([]dto.OfficialListItem, 0, len(hits))
	for _, o := range hits {
		items = append(items, dto.OfficialListItem{
			ID: int(o.ID), Name: o.Name, Alias: []string{},
		})
	}
	return items, nil
}

// GetDetail — GET /galgame-official/:id (id = a catalog LABEL id)
//
// Entity detail lists the forum-LOCAL subset of the label's catalogue, so the
// kungal filters (类型/语言/平台/作品类型) + every sort work. Only the label's
// metadata comes from upstream; the galgame list is recomputed locally from the
// member ids below.
func (s *OfficialService) GetDetail(
	ctx context.Context,
	id string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.OfficialDetail, *errors.AppError) {
	o, found, appErr := s.galgameClient.CatalogLabel(ctx, id, isSFW)
	if appErr != nil {
		return nil, appErr
	}
	if !found {
		return nil, errors.ErrNotFound("未找到该会社")
	}

	memberIDs, appErr := s.galgameClient.CatalogMemberGIDs(ctx,
		url.Values{"label_id": {id}}, isSFW, taxonomyMemberPageCap)
	if appErr != nil {
		return nil, appErr
	}
	page, appErr := s.galgameSvc.hydrateListCards(ctx, buildEntityFilter(rawQuery, memberIDs), isSFW)
	if appErr != nil {
		return nil, appErr
	}

	return &dto.OfficialDetail{
		ID:   int(o.ID),
		Name: o.DisplayName,
		// The wiki's separate `original` field has no public catalog
		// counterpart; the label's own name IS the original in the common case
		// (a Japanese brand is stored under its Japanese name), and the
		// alternate spellings are in alias[].
		Link:         firstLabelLink(o),
		Category:     o.Kind,
		Lang:         o.Lang,
		Description:  preferredIntro(o.Intros),
		Alias:        emptyStrSliceIfNil(o.Aliases),
		Galgame:      listCardsToEntityCards(page.Galgames),
		GalgameCount: page.Total,
	}, nil
}

// firstLabelLink picks the maker's primary web presence for the header button.
// The official site is preferred; otherwise the first link the catalog carries
// (identity anchors never appear in links[], so anything here is a real page).
func firstLabelLink(o *client.CatalogLabelDetail) string {
	for _, l := range o.Links {
		if l.Source == "official" || l.Source == "homepage" {
			return l.URL
		}
	}
	if len(o.Links) > 0 {
		return o.Links[0].URL
	}
	return ""
}

// withSFWFilter clones q and pins `content_limit` per the NSFW protocol (see
// docs/galgame_wiki/00-handbook-for-downstream.md §16).
//
// Both modes are EXPLICIT: omitting the parameter would fall to each endpoint's
// own default (mostly `sfw`), so an SFW-cookie-off user would still get SFW
// from list/search endpoints. We must say `all` aloud to include NSFW.
//
//	isSFW=true  → content_limit=sfw  (only SFW; matches list/search default)
//	isSFW=false → content_limit=all  (user opted in; both SFW + NSFW)
//
// `nsfw`-only isn't reachable from the FE (the cookie only flips on/off).
func withSFWFilter(q url.Values, isSFW bool) url.Values {
	out := url.Values{}
	maps.Copy(out, q)
	if isSFW {
		out.Set("content_limit", "sfw")
	} else {
		out.Set("content_limit", "all")
	}
	return out
}
