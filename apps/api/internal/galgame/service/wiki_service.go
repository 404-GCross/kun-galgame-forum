package service

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"
)

// renameTaxonomyIDField translates the camelCase id field that the FE
// (KunUI tag/official/engine edit modals) sends — `tagId`, `officialId`,
// `engineId` — into the snake_case form (`tag_id`, `official_id`,
// `engine_id`) that the wiki taxonomy PUT endpoints actually validate.
//
// Without this rename the wiki silently drops the unknown camelCase key,
// fails to find the target row (required `*_id` missing), and the entire
// edit flow breaks. We do it here rather than in the FE so the three
// modals don't each need their own body-translation step (mirrors the
// per-call manual translation we already do for series `galgame_ids`).
//
// Only PUTs need it (POST creates have no id field in the body, DELETEs
// have no body). For unrecognized paths it returns the body untouched.
func renameTaxonomyIDField(wikiPath string, body []byte) []byte {
	if len(body) == 0 {
		return body
	}

	var srcKey, dstKey string
	switch {
	case strings.HasPrefix(wikiPath, "/tag"):
		srcKey, dstKey = "tagId", "tag_id"
	case strings.HasPrefix(wikiPath, "/official"):
		srcKey, dstKey = "officialId", "official_id"
	case strings.HasPrefix(wikiPath, "/engine"):
		srcKey, dstKey = "engineId", "engine_id"
	default:
		return body
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return body
	}
	v, ok := m[srcKey]
	if !ok {
		return body
	}
	delete(m, srcKey)
	m[dstKey] = v
	out, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return out
}

// WikiService handles pass-through proxying to the wiki service and the
// common "wiki + local user resolution" pattern used by galgame sub-routes.
type WikiService struct {
	wikiClient  *client.GalgameClient
	galgameRepo *repository.GalgameRepository
	userClient  *userclient.Client
}

func NewWikiService(
	wikiClient *client.GalgameClient,
	galgameRepo *repository.GalgameRepository,
	userClient *userclient.Client,
) *WikiService {
	return &WikiService{wikiClient: wikiClient, galgameRepo: galgameRepo, userClient: userClient}
}

// ──────────────────────────────────────────
// Generic proxy
// ──────────────────────────────────────────

// The bare ProxyGet (the old-wire revision/PR/links/aliases read proxies'
// vehicle) retired in E3b with its routes; ProxyGetWithToken below stays for
// the taxonomy revision reads.

// wikiRevisionRow / wikiRevisionListResp mirror the wiki's revision list
// shape (shared by the four taxonomy entities).
type wikiRevisionRow struct {
	ID        int    `json:"id"`
	GalgameID int    `json:"galgame_id"`
	Revision  int    `json:"revision"`
	UserID    int    `json:"user_id"`
	Action    string `json:"action"`
	Note      string `json:"note"`
	IsMinor   bool   `json:"is_minor"`
	Created   string `json:"created"`
}

type wikiRevisionListResp struct {
	Items []wikiRevisionRow `json:"items"`
	Total int64             `json:"total"`
}

// GetTaxonomyRevisions lists a taxonomy entity's (tag/official/engine)
// revision history, hydrating each row's user_id into a full user object,
// so the FE sees one camelCase {items,total} shape with a real name/avatar
// instead of a raw snake_case wiki row (whose missing `user` field otherwise
// crashed the list renderer). `entity` is the wiki resource name
// (tag/official/engine); the bearer is forwarded because the wiki gates
// these GETs behind auth.
func (s *WikiService) GetTaxonomyRevisions(
	ctx context.Context,
	entity, id, token string,
	query url.Values,
) (*dto.GalgameRevisionListPage, *errors.AppError) {
	data, appErr := s.wikiClient.GetWithToken(ctx, "/"+entity+"/"+id+"/revisions", token, query)
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiRevisionListResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return &dto.GalgameRevisionListPage{Items: []dto.GalgameRevision{}, Total: 0}, nil
	}

	ids := make([]int, len(parsed.Items))
	for i, r := range parsed.Items {
		ids[i] = r.UserID
	}
	userMap := s.userClient.Hydrate(ctx, ids)

	// Drop revisions authored by a banned user. Total is left wiki-sourced —
	// a small over-report matching the SFW filtering trade-off elsewhere.
	items := make([]dto.GalgameRevision, 0, len(parsed.Items))
	for _, r := range parsed.Items {
		if !userclient.IsRenderable(userMap[r.UserID]) {
			continue
		}
		items = append(items, dto.GalgameRevision{
			ID:       r.ID,
			Revision: r.Revision,
			Action:   r.Action,
			Note:     r.Note,
			User:     userBriefToDTO(userMap[r.UserID]),
			IsMinor:  r.IsMinor,
			Created:  r.Created,
		})
	}
	return &dto.GalgameRevisionListPage{Items: items, Total: parsed.Total}, nil
}

// ProxyGetWithToken is ProxyGet but forwards the caller's wiki bearer token
// (empty = anonymous). Used by the taxonomy revision-history GETs: the wiki
// gates `/tag|official|engine/:id/revisions` behind auth (per the 02-revisions
// "后端透传 Bearer 代理" contract), so a token-less ProxyGet 401s for everyone.
// Empty token still works for any endpoint the wiki actually serves publicly.
func (s *WikiService) ProxyGetWithToken(
	ctx context.Context,
	gatewayPath, token string,
	query url.Values,
) (json.RawMessage, *errors.AppError) {
	return s.wikiClient.GetWithToken(ctx, client.ToWikiPath(gatewayPath), token, query)
}

// ProxyWrite forwards a write request (POST/PUT/DELETE) with OAuth token.
//
// contentType is the original Content-Type from the gateway request — kept
// verbatim so multipart boundaries / form-encoded payloads survive the hop.
// Empty defaults to application/json (what the wiki expects on JSON bodies).
// query is the original gateway query string (url.Values). It MUST be
// forwarded — e.g. the two-stage safe delete on tag/official/engine
// keys off `?force=true` (docs 04-taxonomy / 00-handbook): dropping it
// would make every force-delete silently behave as a plain delete.
func (s *WikiService) ProxyWrite(
	ctx context.Context,
	method, gatewayPath, token string,
	query url.Values,
	body []byte,
	contentType string,
) (json.RawMessage, *errors.AppError) {
	wikiPath := client.ToWikiPath(gatewayPath)

	// Translate FE camelCase taxonomy id (tagId/officialId/engineId) to
	// the wiki's required snake_case form. Done on PUT only — see helper
	// doc above for the rationale.
	if method == "PUT" {
		body = renameTaxonomyIDField(wikiPath, body)
	}

	if len(query) > 0 {
		wikiPath += "?" + query.Encode()
	}
	payload := json.RawMessage(body)

	switch method {
	case "POST":
		return s.wikiClient.PostWithToken(ctx, wikiPath, token, payload, contentType)
	case "PUT":
		return s.wikiClient.PutWithToken(ctx, wikiPath, token, payload, contentType)
	case "DELETE":
		return s.wikiClient.DeleteWithToken(ctx, wikiPath, token, payload, contentType)
	default:
		return nil, errors.ErrBadRequest("不支持的方法")
	}
}

// ──────────────────────────────────────────
// Galgame Links
// ──────────────────────────────────────────

type wikiLinkRow struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Link      string `json:"link"`
	GalgameID int    `json:"galgame_id"`
	UserID    int    `json:"user_id"`
}

// GetGalgameLinks fetches the links for a galgame from wiki and resolves the
// author user in the local DB.
func (s *WikiService) GetGalgameLinks(
	ctx context.Context,
	gid string,
) ([]dto.GalgameLink, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/galgame/"+gid+"/links", nil)
	if appErr != nil {
		return nil, appErr
	}

	var rows []wikiLinkRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return []dto.GalgameLink{}, nil
	}

	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.UserID
	}
	userMap := s.userClient.Hydrate(ctx, ids)

	// Drop links added by a banned user.
	out := make([]dto.GalgameLink, 0, len(rows))
	for _, r := range rows {
		if !userclient.IsRenderable(userMap[r.UserID]) {
			continue
		}
		out = append(out, dto.GalgameLink{
			ID:        r.ID,
			User:      userBriefToDTO(userMap[r.UserID]),
			GalgameID: r.GalgameID,
			Name:      r.Name,
			Link:      r.Link,
		})
	}
	return out, nil
}

// The old-wire galgame revision/PR list reads (GetGalgameHistory /
// GetGalgamePRs and their row shapes) retired in E3b — kungal reads the
// editing engine now (edit_handler.go).
