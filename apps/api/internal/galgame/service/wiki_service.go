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

// ProxyGet forwards a GET request to the wiki service and returns the raw body.
// The gateway path is translated to the wiki path via client.ToWikiPath.
func (s *WikiService) ProxyGet(
	ctx context.Context,
	gatewayPath string,
	query url.Values,
) (json.RawMessage, *errors.AppError) {
	return s.wikiClient.Get(ctx, client.ToWikiPath(gatewayPath), query)
}

// GetTaxonomyRevisions lists a taxonomy entity's (tag/official/engine)
// revision history, hydrating each row's user_id into a full user object —
// exactly like GetGalgameHistory does for galgame revisions, so the FE sees
// one camelCase {items,total} shape with a real name/avatar instead of a raw
// snake_case wiki row (whose missing `user` field otherwise crashed the list
// renderer). `entity` is the wiki resource name (tag/official/engine); the
// bearer is forwarded because the wiki gates these GETs behind auth.
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

	items := make([]dto.GalgameRevision, len(parsed.Items))
	for i, r := range parsed.Items {
		items[i] = dto.GalgameRevision{
			ID:       r.ID,
			Revision: r.Revision,
			Action:   r.Action,
			Note:     r.Note,
			User:     userBriefToDTO(userMap[r.UserID]),
			IsMinor:  r.IsMinor,
			Created:  r.Created,
		}
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

	out := make([]dto.GalgameLink, len(rows))
	for i, r := range rows {
		out[i] = dto.GalgameLink{
			ID:        r.ID,
			User:      userBriefToDTO(userMap[r.UserID]),
			GalgameID: r.GalgameID,
			Name:      r.Name,
			Link:      r.Link,
		}
	}
	return out, nil
}

// ──────────────────────────────────────────
// Galgame History (revisions)
// ──────────────────────────────────────────

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

// GetGalgameHistory fetches revision history for a galgame and resolves users.
func (s *WikiService) GetGalgameHistory(
	ctx context.Context,
	gid string,
	query url.Values,
) (*dto.GalgameRevisionListPage, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/galgame/"+gid+"/revisions", query)
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

	items := make([]dto.GalgameRevision, len(parsed.Items))
	for i, r := range parsed.Items {
		items[i] = dto.GalgameRevision{
			ID:       r.ID,
			Revision: r.Revision,
			Action:   r.Action,
			Note:     r.Note,
			User:     userBriefToDTO(userMap[r.UserID]),
			IsMinor:  r.IsMinor,
			Created:  r.Created,
		}
	}
	return &dto.GalgameRevisionListPage{Items: items, Total: parsed.Total}, nil
}

// ──────────────────────────────────────────
// Galgame PRs
// ──────────────────────────────────────────

type wikiPRRow struct {
	ID            int     `json:"id"`
	GalgameID     int     `json:"galgame_id"`
	Status        int     `json:"status"`
	Title         string  `json:"title"`
	Message       string  `json:"message"`
	BaseRevision  int     `json:"base_revision"`
	UserID        int     `json:"user_id"`
	CompletedTime *string `json:"completed_time"`
	Created       string  `json:"created"`
}

type wikiPRListResp struct {
	Items []wikiPRRow `json:"items"`
	Total int64       `json:"total"`
}

// GetGalgamePRs fetches the PR list for a galgame and resolves submitter users.
func (s *WikiService) GetGalgamePRs(
	ctx context.Context,
	gid string,
	query url.Values,
) (*dto.GalgamePRListPage, *errors.AppError) {
	data, appErr := s.wikiClient.Get(ctx, "/galgame/"+gid+"/prs", query)
	if appErr != nil {
		return nil, appErr
	}

	var parsed wikiPRListResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return &dto.GalgamePRListPage{Items: []dto.GalgamePR{}, Total: 0}, nil
	}

	ids := make([]int, len(parsed.Items))
	for i, r := range parsed.Items {
		ids[i] = r.UserID
	}
	userMap := s.userClient.Hydrate(ctx, ids)

	items := make([]dto.GalgamePR, len(parsed.Items))
	for i, r := range parsed.Items {
		items[i] = dto.GalgamePR{
			ID:            r.ID,
			GalgameID:     r.GalgameID,
			Status:        r.Status,
			Title:         r.Title,
			Message:       r.Message,
			BaseRevision:  r.BaseRevision,
			User:          userBriefToDTO(userMap[r.UserID]),
			CompletedTime: r.CompletedTime,
			Created:       r.Created,
		}
	}
	return &dto.GalgamePRListPage{Items: items, Total: parsed.Total}, nil
}

