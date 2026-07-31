package service

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
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
// `engine_id`) that the galgame taxonomy PUT endpoints actually validate.
//
// Without this rename the galgame silently drops the unknown camelCase key,
// fails to find the target row (required `*_id` missing), and the entire
// edit flow breaks. We do it here rather than in the FE so the three
// modals don't each need their own body-translation step (mirrors the
// per-call manual translation we already do for series `galgame_ids`).
//
// Only PUTs need it (POST creates have no id field in the body, DELETEs
// have no body). For unrecognized paths it returns the body untouched.
func renameTaxonomyIDField(galgamePath string, body []byte) []byte {
	if len(body) == 0 {
		return body
	}

	var srcKey, dstKey string
	switch {
	case strings.HasPrefix(galgamePath, "/tag"):
		srcKey, dstKey = "tagId", "tag_id"
	case strings.HasPrefix(galgamePath, "/official"):
		srcKey, dstKey = "officialId", "official_id"
	case strings.HasPrefix(galgamePath, "/engine"):
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

// GalgameProxyService handles pass-through proxying to the galgame service and the
// common "galgame + local user resolution" pattern used by galgame sub-routes.
type GalgameProxyService struct {
	galgameClient *client.GalgameClient
	galgameRepo   *repository.GalgameRepository
	userClient    *userclient.Client
}

func NewGalgameProxyService(
	galgameClient *client.GalgameClient,
	galgameRepo *repository.GalgameRepository,
	userClient *userclient.Client,
) *GalgameProxyService {
	return &GalgameProxyService{galgameClient: galgameClient, galgameRepo: galgameRepo, userClient: userClient}
}

// ──────────────────────────────────────────
// Generic proxy
// ──────────────────────────────────────────

// ProxyWrite forwards a write request (POST/PUT/DELETE) with OAuth token.
//
// contentType is the original Content-Type from the gateway request — kept
// verbatim so multipart boundaries / form-encoded payloads survive the hop.
// Empty defaults to application/json (what the galgame expects on JSON bodies).
// query is the original gateway query string (url.Values). It MUST be
// forwarded — e.g. the two-stage safe delete on tag/official/engine
// keys off `?force=true` (docs 04-taxonomy / 00-handbook): dropping it
// would make every force-delete silently behave as a plain delete.
func (s *GalgameProxyService) ProxyWrite(
	ctx context.Context,
	method, gatewayPath, token string,
	query url.Values,
	body []byte,
	contentType string,
) (json.RawMessage, *errors.AppError) {
	galgamePath := client.ToGalgamePath(gatewayPath)

	// Translate FE camelCase taxonomy id (tagId/officialId/engineId) to
	// the galgame's required snake_case form. Done on PUT only — see helper
	// doc above for the rationale.
	if method == "PUT" {
		body = renameTaxonomyIDField(galgamePath, body)
	}

	if len(query) > 0 {
		galgamePath += "?" + query.Encode()
	}
	payload := json.RawMessage(body)

	switch method {
	case "POST":
		return s.galgameClient.PostWithToken(ctx, galgamePath, token, payload, contentType)
	case "PUT":
		return s.galgameClient.PutWithToken(ctx, galgamePath, token, payload, contentType)
	case "DELETE":
		return s.galgameClient.DeleteWithToken(ctx, galgamePath, token, payload, contentType)
	default:
		return nil, errors.ErrBadRequest("不支持的方法")
	}
}

// ──────────────────────────────────────────
// Galgame Links
// ──────────────────────────────────────────

type nextMoeLinkRow struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Link      string `json:"link"`
	GalgameID int    `json:"galgame_id"`
	UserID    int    `json:"user_id"`
}

// GetGalgameLinks fetches a galgame's curated external links.
//
// These are platform-curated rows now: the retirement wave absorbed the wiki's
// user-submitted links WITHOUT their submitter, so there is no author to
// resolve and the banned-author filter has no subject any more (doc 126 D6 —
// the user_id field sunsets with the wiki). Link moderation for new rows runs
// through the editing engine's own review chain instead.
func (s *GalgameProxyService) GetGalgameLinks(
	ctx context.Context,
	gid string,
) ([]dto.GalgameLink, *errors.AppError) {
	gidInt, err := strconv.Atoi(gid)
	if err != nil || gidInt <= 0 {
		return nil, errors.ErrBadRequest("无效的 Galgame ID")
	}
	rows, appErr := s.galgameClient.CatalogWorkLinks(ctx, gidInt)
	if appErr != nil {
		return nil, appErr
	}
	out := make([]dto.GalgameLink, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.GalgameLink{
			GalgameID: gidInt,
			Name:      r.Name,
			Link:      r.Link,
		})
	}
	return out, nil
}

// The old-wire galgame revision/PR list reads (GetGalgameHistory /
// GetGalgamePRs and their row shapes) retired in E3b — kungal reads the
// editing engine now (edit_handler.go).
