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
	wikiClient *client.GalgameClient
	enricher   *GalgameEnricher
}

func NewOfficialService(wikiClient *client.GalgameClient, enricher *GalgameEnricher) *OfficialService {
	return &OfficialService{wikiClient: wikiClient, enricher: enricher}
}

// ──────────────────────────────────────────
// Wiki response shapes
// ──────────────────────────────────────────

type wikiOfficialListItem struct {
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	Link         string          `json:"link"`
	Category     string          `json:"category"`
	Lang         string          `json:"lang"`
	Alias        []dto.WikiAlias `json:"alias"`
	GalgameCount int             `json:"galgame_count"`
}

type wikiOfficialListResp struct {
	Items []wikiOfficialListItem `json:"items"`
	Total int64                  `json:"total"`
}

type wikiOfficialDetail struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Original-language name (added by wiki PR4 sub-change, K-PR6).
	// Pointer because wiki may omit / null when the field hasn't been
	// set yet; the FE edit modal needs to round-trip the current
	// value, so dropping it on the floor here makes the modal default
	// to an empty input every open.
	Original    *string         `json:"original"`
	Link        string          `json:"link"`
	Category    string          `json:"category"`
	Lang        string          `json:"lang"`
	Description string          `json:"description"`
	Alias       []dto.WikiAlias `json:"alias"`
}

type wikiOfficialDetailResp struct {
	Official wikiOfficialDetail    `json:"official"`
	Galgames []dto.WikiGalgameItem `json:"galgames"`
	Total    int64                 `json:"total"`
}

// ──────────────────────────────────────────
// GetList — GET /galgame-official
// ──────────────────────────────────────────

func (s *OfficialService) GetList(
	ctx context.Context,
	rawQuery url.Values,
) (*dto.OfficialListPage, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/official", rawQuery)
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiOfficialListResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
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
// Wiki search is Meilisearch-backed and returns the standard
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
	data, appErr := s.wikiClient.Get(ctx, "/official/search", rawQuery)
	if appErr != nil {
		return nil, appErr
	}
	var resp struct {
		Items []wikiOfficialListItem `json:"items"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
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
	// The galgame list is the wiki's FULL member catalogue — paginated,
	// content_limit-aware, with a matching total — NOT the forum-local subset.
	// The wiki /official/:id response already carries the galgame page, so we
	// forward page/limit/sort + content_limit and use it directly. The enricher
	// overlays forum-local data (view/like/platform/language + IsOnForum) per
	// card; games the forum has never ingested come back IsOnForum=false so the
	// frontend shows a "未收录" state instead of misleading zeros.
	q := withSFWFilter(rawQuery, isSFW)
	data, appErr := s.wikiClient.Get(ctx, "/official/"+name, q)
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiOfficialDetailResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 响应失败")
	}

	o := parsed.Official
	original := ""
	if o.Original != nil {
		original = *o.Original
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
		Galgame:      s.enricher.ToCards(ctx, parsed.Galgames),
		GalgameCount: parsed.Total,
	}, nil
}

// aliasesToNames extracts the name field from a slice of WikiAlias.
func aliasesToNames(aliases []dto.WikiAlias) []string {
	out := make([]string, len(aliases))
	for i, a := range aliases {
		out[i] = a.Name
	}
	return out
}

// withSFWFilter clones q and pins `content_limit` per the wiki NSFW
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
