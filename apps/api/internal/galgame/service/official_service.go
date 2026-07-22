package service

import (
	"context"
	"encoding/json"
	"maps"
	"net/url"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

type OfficialService struct {
	galgameClient *client.GalgameClient
	// galgameSvc runs the shared local filter/sort/paginate + hydration flow
	// over the official's member ids (the galgame can't filter by kungal-local
	// resource data). See GetDetail.
	galgameSvc *GalgameService
}

func NewOfficialService(galgameClient *client.GalgameClient, galgameSvc *GalgameService) *OfficialService {
	return &OfficialService{galgameClient: galgameClient, galgameSvc: galgameSvc}
}

// ──────────────────────────────────────────
// Galgame response shapes
// ──────────────────────────────────────────

// nextMoeOfficialListItem is the /v1 curated maker record (GET
// /v1/galgame/officials list item = PublicOfficialEntity). Aliases is a plain
// name slice ([]string) on /v1, unlike the bridge's alias-row objects.
type nextMoeOfficialListItem struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Link         string   `json:"link"`
	Category     string   `json:"category"`
	Lang         string   `json:"lang"`
	Aliases      []string `json:"aliases"`
	GalgameCount int      `json:"galgame_count"`
}

type nextMoeOfficialListResp struct {
	Items []nextMoeOfficialListItem `json:"items"`
	Total int64                     `json:"total"`
}

// nextMoeOfficialSearchItem is one /v1 Meilisearch maker hit
// (PublicOfficialSearchItem). It carries no `link` (the search curation drops
// it) — the search-result cards do not render a maker website, so Link falls to
// "" (the entity detail page keeps link via the full record below).
type nextMoeOfficialSearchItem struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Lang         string   `json:"lang"`
	Aliases      []string `json:"aliases"`
	GalgameCount int      `json:"galgame_count"`
}

// nextMoeOfficialDetail is the /v1 curated maker record
// (GET /v1/galgame/officials/{id} = PublicOfficialEntity). Original is a plain
// string ("" when unset) on /v1; the FE edit modal round-trips it.
type nextMoeOfficialDetail struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Original     string   `json:"original"`
	Link         string   `json:"link"`
	Category     string   `json:"category"`
	Lang         string   `json:"lang"`
	Description  string   `json:"description"`
	Aliases      []string `json:"aliases"`
	GalgameCount int      `json:"galgame_count"`
}

// ──────────────────────────────────────────
// GetList — GET /galgame-official
// ──────────────────────────────────────────

func (s *OfficialService) GetList(
	ctx context.Context,
	rawQuery url.Values,
) (*dto.OfficialListPage, *errors.AppError) {
	data, appErr := s.galgameClient.GetV1(ctx, "/galgame/officials", rawQuery)
	if appErr != nil {
		return nil, appErr
	}

	var parsed nextMoeOfficialListResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}

	items := make([]dto.OfficialListItem, len(parsed.Items))
	for i, o := range parsed.Items {
		items[i] = dto.OfficialListItem{
			ID:           o.ID,
			Name:         o.Name,
			Link:         o.Link,
			Category:     o.Category,
			Lang:         o.Lang,
			Alias:        emptyStrSliceIfNil(o.Aliases),
			GalgameCount: o.GalgameCount,
		}
	}
	return &dto.OfficialListPage{Officials: items, Total: parsed.Total}, nil
}

// ──────────────────────────────────────────
// Search — GET /galgame-official/search
// ──────────────────────────────────────────
//
// Galgame search is Meilisearch-backed and returns the standard
// `{items, total, processing_time_ms}` envelope. The alias field on each
// item may be missing entirely or populated with {id, name, ...} objects;
// aliasesToNames(nil) → []string{} keeps the frontend contract intact.
//
// The frontend (galgame/official/Container.vue) does
//
//	`searchResult.value = res`  expecting a bare GalgameOfficialItem[].
//
// We unwrap `items` here so the gateway response stays compatible without
// touching the frontend.
func (s *OfficialService) Search(
	ctx context.Context,
	rawQuery url.Values,
) ([]dto.OfficialListItem, *errors.AppError) {
	data, appErr := s.galgameClient.GetV1(ctx, "/galgame/officials/search", rawQuery)
	if appErr != nil {
		return nil, appErr
	}
	var resp struct {
		Items []nextMoeOfficialSearchItem `json:"items"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}
	raw := resp.Items

	items := make([]dto.OfficialListItem, len(raw))
	for i, o := range raw {
		items[i] = dto.OfficialListItem{
			ID:   o.ID,
			Name: o.Name,
			// The /v1 maker-search hit drops `link` (curation) — the search-result
			// cards don't render a website, so Link stays "" here (the entity
			// detail page keeps it via the full record).
			Category:     o.Category,
			Lang:         o.Lang,
			Alias:        emptyStrSliceIfNil(o.Aliases),
			GalgameCount: o.GalgameCount,
		}
	}
	return items, nil
}

// ──────────────────────────────────────────
// GetDetail — GET /galgame-official/:name
// ──────────────────────────────────────────

func (s *OfficialService) GetDetail(
	ctx context.Context,
	name string,
	rawQuery url.Values,
	isSFW bool,
) (*dto.OfficialDetail, *errors.AppError) {
	// Entity detail lists the forum-LOCAL subset of the official's catalogue, so
	// the kungal filters (类型/语言/平台/作品类型) + every sort work. Only the
	// official's metadata is used from the galgame here (cheapest page); the galgame
	// list is recomputed locally from the member ids below.
	// /v1 maker entity is by-id (the :name segment carries the numeric id) and
	// needs no query params — the curated record has the full metadata
	// (original / link / lang / description / aliases). The kungal member list is
	// recomputed locally below.
	data, appErr := s.galgameClient.GetV1(ctx, "/galgame/officials/"+name, nil)
	if appErr != nil {
		return nil, appErr
	}

	var o nextMoeOfficialDetail
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}

	// Member ids from the galgame, then the SAME local filter/sort/paginate as
	// /galgame over them (RestrictIDs). Un-ingested members drop out naturally.
	memberIDs, appErr := s.galgameClient.EntityGalgameIDs(ctx, "official", o.ID)
	if appErr != nil {
		return nil, appErr
	}
	page, appErr := s.galgameSvc.hydrateListCards(ctx, buildEntityFilter(rawQuery, memberIDs), isSFW)
	if appErr != nil {
		return nil, appErr
	}

	return &dto.OfficialDetail{
		ID:           o.ID,
		Name:         o.Name,
		Original:     o.Original,
		Link:         o.Link,
		Category:     o.Category,
		Lang:         o.Lang,
		Description:  o.Description,
		Alias:        emptyStrSliceIfNil(o.Aliases),
		Galgame:      listCardsToEntityCards(page.Galgames),
		GalgameCount: page.Total,
	}, nil
}

// aliasesToNames extracts the name field from a slice of NextMoeAlias.
func aliasesToNames(aliases []dto.NextMoeAlias) []string {
	out := make([]string, len(aliases))
	for i, a := range aliases {
		out[i] = a.Name
	}
	return out
}

// withSFWFilter clones q and pins `content_limit` per the galgame NSFW
// protocol (see docs/galgame_wiki/00-handbook-for-downstream.md §16).
//
// Both modes are EXPLICIT: omitting the parameter would fall to each
// endpoint's own default (mostly `sfw`), so an SFW-cookie-off user would
// still get SFW from list/search endpoints. We must say `all` aloud to
// include NSFW.
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
