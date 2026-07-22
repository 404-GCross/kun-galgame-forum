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

type nextMoeOfficialListItem struct {
	ID           int                `json:"id"`
	Name         string             `json:"name"`
	Link         string             `json:"link"`
	Category     string             `json:"category"`
	Lang         string             `json:"lang"`
	Alias        []dto.NextMoeAlias `json:"alias"`
	GalgameCount int                `json:"galgame_count"`
}

type nextMoeOfficialListResp struct {
	Items []nextMoeOfficialListItem `json:"items"`
	Total int64                     `json:"total"`
}

type nextMoeOfficialDetail struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Original-language name (added by galgame PR4 sub-change, K-PR6).
	// Pointer because galgame may omit / null when the field hasn't been
	// set yet; the FE edit modal needs to round-trip the current
	// value, so dropping it on the floor here makes the modal default
	// to an empty input every open.
	Original    *string            `json:"original"`
	Link        string             `json:"link"`
	Category    string             `json:"category"`
	Lang        string             `json:"lang"`
	Description string             `json:"description"`
	Alias       []dto.NextMoeAlias `json:"alias"`
}

type nextMoeOfficialDetailResp struct {
	Official nextMoeOfficialDetail    `json:"official"`
	Galgames []dto.NextMoeGalgameItem `json:"galgames"`
	Total    int64                    `json:"total"`
}

// ──────────────────────────────────────────
// GetList — GET /galgame-official
// ──────────────────────────────────────────

func (s *OfficialService) GetList(
	ctx context.Context,
	rawQuery url.Values,
) (*dto.OfficialListPage, *errors.AppError) {
	data, appErr := s.galgameClient.Get(ctx, "/official", rawQuery)
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
			Alias:        aliasesToNames(o.Alias),
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
	data, appErr := s.galgameClient.Get(ctx, "/official/search", rawQuery)
	if appErr != nil {
		return nil, appErr
	}
	var resp struct {
		Items []nextMoeOfficialListItem `json:"items"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}
	raw := resp.Items

	items := make([]dto.OfficialListItem, len(raw))
	for i, o := range raw {
		items[i] = dto.OfficialListItem{
			ID:           o.ID,
			Name:         o.Name,
			Link:         o.Link,
			Category:     o.Category,
			Lang:         o.Lang,
			Alias:        aliasesToNames(o.Alias),
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
	q := withSFWFilter(rawQuery, isSFW)
	// The galgame resolves the entity by the official_id QUERY param (the :name path
	// segment is cosmetic — 04-taxonomy). Source it from the path so the lookup
	// never depends on the FE echoing it in the query string.
	q.Set("official_id", name)
	q.Set("page", "1")
	q.Set("limit", "1")
	data, appErr := s.galgameClient.Get(ctx, "/official/"+name, q)
	if appErr != nil {
		return nil, appErr
	}

	var parsed nextMoeOfficialDetailResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Galgame 响应失败")
	}

	o := parsed.Official
	original := ""
	if o.Original != nil {
		original = *o.Original
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
		Original:     original,
		Link:         o.Link,
		Category:     o.Category,
		Lang:         o.Lang,
		Description:  o.Description,
		Alias:        aliasesToNames(o.Alias),
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
