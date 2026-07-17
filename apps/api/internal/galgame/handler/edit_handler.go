package handler

import (
	"context"
	stderrors "errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/role"

	"github.com/gofiber/fiber/v3"
)

// EditHandler is the BFF face onto the infra editing engine (E3a; contract
// /api/v1/catalog/edit/*): the schema-driven galgame editor, the kungal
// review queue (amend / merge / decline — the maintainer-edit crown), the
// revision history + diff reads, and the "my proposals" list. kungal
// authenticates its user locally and ASSERTS the actor over the Basic-authed
// S2S channel (the letmoe E1 posture); the engine's kungal site overlay then
// decides every write's fate — proposals never automerge, they land in the
// review queue.
//
// Actor assertion (E3a ruling 3): roles pass through VERBATIM — the edit
// face resolves them through the galgame family's own perm vocabulary
// (admin/ren hold edit.galgame.game.review), so the BFF holds no policy
// logic beyond the entry gates. Trust tier is the conservative staff
// mapping mirroring the community starter-boost floors (staff → 3,
// everyone else → 0; kungal declares no creator boost).
//
// Degradation: an unconfigured catalog client → 503 on every endpoint and
// the frontend hides the entries. Never a local fallback.
type EditHandler struct {
	catalog *catalogclient.Client
	wiki    *client.GalgameClient // best-effort brief enrichment for lists
}

func NewEditHandler(catalog *catalogclient.Client, wiki *client.GalgameClient) *EditHandler {
	return &EditHandler{catalog: catalog, wiki: wiki}
}

// catalogSite is the tenant key kungal files edits under — must equal the
// forum OAuth client's oauth_clients.catalog_site binding.
const catalogSite = "kungal"

// entityTypeGame is the galgame family's entity type (infra editspec).
const entityTypeGame = "galgame.game"

// fieldKeyPrefix guards the pass-through patch: only galgame.game.* keys may
// ride this BFF (the engine re-validates each key against the registry).
const fieldKeyPrefix = "galgame.game."

var errEditDown = errors.New(errors.CodeBiz, "资料库编辑服务暂不可用", http.StatusServiceUnavailable)

// editActor builds the asserted actor for the current session.
func editActor(c fiber.Ctx) (catalogclient.EditActor, *errors.AppError) {
	user := middleware.GetUser(c)
	if user == nil {
		return catalogclient.EditActor{}, errors.ErrAuthExpired()
	}
	var tier int16
	if role.CanModerate(user.Roles) {
		tier = 3 // staff (mirrors the community starter-boost staff floor)
	}
	return catalogclient.EditActor{UserID: int64(user.ID), Roles: user.Roles, TrustTier: tier}, nil
}

// editError maps a catalogclient edit error onto the house envelope. The
// edit face's 4xx replies carry actionable reasons (validation details,
// policy denials, rebase conflicts) — surface them; transport failures and
// the unconfigured client degrade to 503.
func editError(c fiber.Ctx, err error) error {
	var apiErr *catalogclient.EditAPIError
	switch {
	case stderrors.Is(err, catalogclient.ErrNotConfigured):
		return response.Error(c, errEditDown)
	case stderrors.As(err, &apiErr):
		switch apiErr.Status {
		case http.StatusForbidden:
			return response.Error(c, errors.ErrForbidden("你没有权限执行此操作"))
		case http.StatusNotFound:
			return response.Error(c, errors.ErrNotFound("条目或提案不存在"))
		case http.StatusUnprocessableEntity:
			return response.Error(c, errors.ErrValidation(apiErr.Message))
		case http.StatusConflict:
			return response.Error(c, errors.New(errors.CodeBiz,
				"操作冲突（条目已被他人修改或提案已关闭），请刷新后重试", http.StatusConflict))
		}
		slog.Error("galgame edit: upstream error", "status", apiErr.Status, "code", apiErr.Code, "msg", apiErr.Message)
		return response.Error(c, errEditDown)
	default:
		slog.Warn("galgame edit: catalog unreachable", "error", err)
		return response.Error(c, errEditDown)
	}
}

func parseGid(c fiber.Ctx) (int64, *errors.AppError) {
	gid, err := strconv.ParseInt(c.Params("gid"), 10, 64)
	if err != nil || gid <= 0 {
		return 0, errors.ErrBadRequest("无效的 Galgame ID")
	}
	return gid, nil
}

func parseProposalID(c fiber.Ctx) (int64, *errors.AppError) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.ErrBadRequest("无效的提案 ID")
	}
	return id, nil
}

// ──────────────────────────────────────────
// Editor (game-scoped)
// ──────────────────────────────────────────

// Bootstrap — GET /galgame/:gid/edit/bootstrap (auth). Everything the
// schema-driven editor needs in one read: the entity-aware capability
// projection for THIS user (the UI holds zero policy logic) + the current
// field values keyed by eternal field keys.
func (h *EditHandler) Bootstrap(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	actor, appErr := editActor(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	ctx := c.Context()
	values, err := h.catalog.EditSnapshot(ctx, entityTypeGame, gid)
	if err != nil {
		return editError(c, err)
	}
	schema, err := h.catalog.GetEditSchema(ctx, catalogSite, entityTypeGame, gid, actor)
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, fiber.Map{
		"gid":        gid,
		"values":     values,
		"fields":     schema.Fields,
		"can_review": anyReviewable(schema.Fields),
	})
}

func anyReviewable(fields []catalogclient.EditSchemaField) bool {
	for _, f := range fields {
		if f.CanReview {
			return true
		}
	}
	return false
}

// editSubmitRequest carries the dirty-field patch (eternal field keys only —
// the frontend submits exactly what the user changed) and an optional note.
type editSubmitRequest struct {
	Patch map[string]any `json:"patch"`
	Note  string         `json:"note"`
}

// Submit — POST /galgame/:gid/edit/proposals (auth). Files the proposal; on
// kungal nothing automerges, so the two-state outcome is normally "open".
func (h *EditHandler) Submit(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	actor, appErr := editActor(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req editSubmitRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	if len(req.Patch) == 0 {
		return response.Error(c, errors.ErrValidation("没有需要保存的修改"))
	}
	for key := range req.Patch {
		if !strings.HasPrefix(key, fieldKeyPrefix) {
			return response.Error(c, errors.ErrValidation("提案包含非法字段: "+key))
		}
	}
	if len(req.Note) > 2000 {
		return response.Error(c, errors.ErrValidation("编辑说明过长"))
	}
	result, err := h.catalog.CreateEditProposal(c.Context(), catalogclient.EditCreateRequest{
		EntityType: entityTypeGame, EntityID: gid, Site: catalogSite,
		Patch: req.Patch, Note: req.Note, Actor: actor,
	})
	if err != nil {
		return editError(c, err)
	}
	out := fiber.Map{"merged": result.Merged, "proposal": result.Proposal}
	if result.Revision != nil {
		out["revision"] = result.Revision
	}
	return response.OK(c, out)
}

// Revisions — GET /galgame/:gid/edit/revisions (public — the wiki's revision
// history has always been a public read). Includes the E2-migrated history.
func (h *EditHandler) Revisions(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	items, err := h.catalog.ListEditRevisions(c.Context(), entityTypeGame, gid, queryInt(c, "limit"))
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, fiber.Map{"gid": gid, "items": items})
}

// Diff — GET /galgame/:gid/edit/diff?from=&to= (public).
func (h *EditHandler) Diff(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	from, to := queryInt(c, "from"), queryInt(c, "to")
	if from < 1 || to < 1 {
		return response.Error(c, errors.ErrBadRequest("需要 from/to 版本号"))
	}
	diff, err := h.catalog.DiffEditRevisions(c.Context(), entityTypeGame, gid, from, to)
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, diff)
}

// ──────────────────────────────────────────
// Review queue + my proposals (proposal-scoped)
// ──────────────────────────────────────────

// proposalItem is one enriched list row: the proposal + a best-effort
// galgame brief (nil when the wiki batch read fails — the UI falls back to
// the bare gid link).
type proposalItem struct {
	catalogclient.EditProposal
	Galgame *client.GalgameBrief `json:"galgame,omitempty"`
}

func (h *EditHandler) enrich(ctx context.Context, items []catalogclient.EditProposal) []proposalItem {
	ids := make([]int, 0, len(items))
	seen := make(map[int]bool, len(items))
	for i := range items {
		gid := int(items[i].EntityID)
		if !seen[gid] {
			seen[gid] = true
			ids = append(ids, gid)
		}
	}
	var briefs map[int]client.GalgameBrief
	if len(ids) > 0 && h.wiki != nil {
		var appErr *errors.AppError
		if briefs, appErr = h.wiki.GetBatch(ctx, ids); appErr != nil {
			slog.Warn("galgame edit: brief enrichment failed", "error", appErr)
		}
	}
	out := make([]proposalItem, 0, len(items))
	for i := range items {
		item := proposalItem{EditProposal: items[i]}
		if b, ok := briefs[int(items[i].EntityID)]; ok {
			brief := b
			item.Galgame = &brief
		}
		out = append(out, item)
	}
	return out
}

// Queue — GET /galgame-edit/queue (auth + moderator entry). Open proposals
// on kungal's tenant, newest-first; ?status widens to the decided ones.
// Field-level adjudication rights still come from the engine's projection —
// this gate is the ENTRY, not the policy.
func (h *EditHandler) Queue(c fiber.Ctx) error {
	status := c.Query("status", "open")
	switch status {
	case "open", "merged", "declined", "withdrawn", "":
	default:
		return response.Error(c, errors.ErrBadRequest("未知的提案状态"))
	}
	items, err := h.catalog.ListEditProposals(c.Context(), catalogclient.EditProposalFilter{
		EntityType: entityTypeGame, Site: catalogSite,
		Status: status, Limit: queryInt(c, "limit"),
	})
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, fiber.Map{"items": h.enrich(c.Context(), items)})
}

// Mine — GET /galgame-edit/mine (auth). The session user's proposals, all
// states, newest-first; ?gid narrows to one galgame (the editor page's
// "my pending proposal" strip).
func (h *EditHandler) Mine(c fiber.Ctx) error {
	actor, appErr := editActor(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	items, err := h.catalog.ListEditProposals(c.Context(), catalogclient.EditProposalFilter{
		EntityType: entityTypeGame, Site: catalogSite,
		EntityID: int64(queryInt(c, "gid")), ProposerUID: actor.UserID,
		Status: c.Query("status"), Limit: queryInt(c, "limit"),
	})
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, fiber.Map{"items": h.enrich(c.Context(), items)})
}

// proposalForReview loads the proposal and pins it to kungal's own tenant +
// the galgame entity type — this BFF adjudicates nothing else (the S2S site
// binding would reject foreign writes anyway; this keeps reads honest too).
func (h *EditHandler) proposalForReview(ctx context.Context, id int64) (*catalogclient.EditProposal, error) {
	prop, err := h.catalog.GetEditProposal(ctx, id)
	if err != nil {
		return nil, err
	}
	if prop.Site != catalogSite || prop.EntityType != entityTypeGame {
		return nil, &catalogclient.EditAPIError{Status: http.StatusNotFound, Message: "proposal outside the kungal tenant"}
	}
	return prop, nil
}

// ProposalDetail — GET /galgame-edit/proposals/:id (auth + moderator entry).
// The review workbench read: proposal + amendments + effective patch, the
// entity's CURRENT values (per-field old→new compare), the reviewer's
// capability projection, and the galgame brief.
func (h *EditHandler) ProposalDetail(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	actor, appErr := editActor(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	ctx := c.Context()
	prop, err := h.catalog.GetEditProposal(ctx, id)
	if err != nil {
		return editError(c, err)
	}
	if prop.Site != catalogSite || prop.EntityType != entityTypeGame {
		return response.Error(c, errors.ErrNotFound("条目或提案不存在"))
	}
	values, err := h.catalog.EditSnapshot(ctx, entityTypeGame, prop.EntityID)
	if err != nil {
		return editError(c, err)
	}
	schema, err := h.catalog.GetEditSchema(ctx, catalogSite, entityTypeGame, prop.EntityID, actor)
	if err != nil {
		return editError(c, err)
	}
	enriched := h.enrich(ctx, []catalogclient.EditProposal{*prop})
	return response.OK(c, fiber.Map{
		"proposal": enriched[0],
		"values":   values,
		"fields":   schema.Fields,
	})
}

// editAmendRequest carries the maintainer delta: corrected values (set) and
// rejected fields (unset).
type editAmendRequest struct {
	Set   map[string]any `json:"set"`
	Unset []string       `json:"unset"`
	Note  string         `json:"note"`
}

// Amend — POST /galgame-edit/proposals/:id/amend (auth + moderator entry).
// The crown mechanism: correct a value / reject a field before merging;
// the merged revision carries proposer + amender double attribution.
func (h *EditHandler) Amend(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	actor, appErr := editActor(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req editAmendRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	if len(req.Set) == 0 && len(req.Unset) == 0 {
		return response.Error(c, errors.ErrValidation("没有需要修改的内容"))
	}
	for key := range req.Set {
		if !strings.HasPrefix(key, fieldKeyPrefix) {
			return response.Error(c, errors.ErrValidation("提案包含非法字段: "+key))
		}
	}
	if len(req.Note) > 2000 {
		return response.Error(c, errors.ErrValidation("说明过长"))
	}
	ctx := c.Context()
	if _, err := h.proposalForReview(ctx, id); err != nil {
		return editError(c, err)
	}
	amendment, err := h.catalog.AmendEditProposal(ctx, id, req.Set, req.Unset, req.Note, actor)
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, amendment)
}

type editDecisionRequest struct {
	Note string `json:"note"`
}

// Merge — POST /galgame-edit/proposals/:id/merge (auth + moderator entry).
func (h *EditHandler) Merge(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	actor, appErr := editActor(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req editDecisionRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	if len(req.Note) > 2000 {
		return response.Error(c, errors.ErrValidation("说明过长"))
	}
	ctx := c.Context()
	if _, err := h.proposalForReview(ctx, id); err != nil {
		return editError(c, err)
	}
	rev, err := h.catalog.MergeEditProposal(ctx, id, req.Note, actor)
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, rev)
}

// Decline — POST /galgame-edit/proposals/:id/decline (auth + moderator
// entry). The reason is required — a silent decline was the old wiki's
// worst reviewer habit.
func (h *EditHandler) Decline(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	actor, appErr := editActor(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req editDecisionRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	if strings.TrimSpace(req.Note) == "" {
		return response.Error(c, errors.ErrValidation("请填写拒绝理由"))
	}
	if len(req.Note) > 2000 {
		return response.Error(c, errors.ErrValidation("说明过长"))
	}
	ctx := c.Context()
	if _, err := h.proposalForReview(ctx, id); err != nil {
		return editError(c, err)
	}
	prop, err := h.catalog.DeclineEditProposal(ctx, id, req.Note, actor)
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, prop)
}

// Withdraw — POST /galgame-edit/proposals/:id/withdraw (auth). The engine
// enforces proposer-only; no moderator gate here.
func (h *EditHandler) Withdraw(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	actor, appErr := editActor(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	ctx := c.Context()
	if _, err := h.proposalForReview(ctx, id); err != nil {
		return editError(c, err)
	}
	prop, err := h.catalog.WithdrawEditProposal(ctx, id, actor)
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, prop)
}
