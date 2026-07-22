package handler

import (
	"context"
	stderrors "errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/role"
	"kun-galgame-api/pkg/userclient"

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
//
// Owner-review (E3b): the review surfaces admit the game's CREATOR alongside
// moderators — the BFF asserts is_entity_owner from the galgame row's uid and
// the engine's kungal overlay grants owners the review rule on the
// default-policy fields only (status/vndb_id stay perm-gated).
//
// Decision side effects (E3b ruling 1 — notifications are the product's job,
// the engine stays notification-free): merge/decline notify the proposer via
// the forum's own message system, merge additionally awards the contribution
// moemoepoint and bumps the local resource_update_time; a submit notifies the
// game owner and mirrors onto the activity timeline — all parity with the old
// PR chain, all best-effort (a failure logs a warning, never rolls back the
// decision).
type EditHandler struct {
	catalog       *catalogclient.Client
	galgameClient *client.GalgameClient // best-effort brief enrichment + owner lookup
	users         *userclient.Client    // best-effort attribution enrichment
	notifier      msgService.Notifier   // best-effort decision notifications
	repo          *repository.GalgameRepository
}

func NewEditHandler(
	catalog *catalogclient.Client, galgameClient *client.GalgameClient, users *userclient.Client,
	notifier msgService.Notifier, repo *repository.GalgameRepository,
) *EditHandler {
	return &EditHandler{catalog: catalog, galgameClient: galgameClient, users: users, notifier: notifier, repo: repo}
}

// editUser is the attribution shape lists carry (contribution attribution
// is a parity hardline — bare uids would be a regression from the old galgame).
type editUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// userMap resolves uids to display users, best-effort: a failed OAuth batch
// read degrades to an empty map and the UI falls back to "用户 #id".
func (h *EditHandler) userMap(ctx context.Context, uids map[int]bool) map[int]editUser {
	out := make(map[int]editUser, len(uids))
	if h.users == nil || len(uids) == 0 {
		return out
	}
	ids := make([]int, 0, len(uids))
	for id := range uids {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	resolved, err := h.users.Users(ctx, ids)
	if err != nil {
		slog.Warn("galgame edit: user enrichment failed", "error", err)
		return out
	}
	for id, u := range resolved {
		out[id] = editUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
	}
	return out
}

func collectProposalUIDs(items []catalogclient.EditProposal) map[int]bool {
	uids := make(map[int]bool)
	for i := range items {
		uids[int(items[i].ProposerUID)] = true
		if items[i].DecidedByUID != nil {
			uids[int(*items[i].DecidedByUID)] = true
		}
		for _, a := range items[i].Amendments {
			uids[int(a.AmenderUID)] = true
		}
	}
	return uids
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
	// Not an authorization gate — a trust-tier projection for the S2S actor
	// assertion (staff → 3). Stays on role.CanModerate (identity/tier, not a
	// pkg/perm forum capability).
	if role.CanModerate(user.Roles) {
		tier = 3 // staff (mirrors the community starter-boost staff floor)
	}
	return catalogclient.EditActor{UserID: int64(user.ID), Roles: user.Roles, TrustTier: tier}, nil
}

// gameBrief reads one galgame brief (best-effort; nil = unknown). The S2S
// batch read is short-TTL cached in the client, so per-request lookups for
// owner assertion / notification naming stay cheap.
func (h *EditHandler) gameBrief(ctx context.Context, gid int64) *client.GalgameBrief {
	if h.galgameClient == nil || gid <= 0 {
		return nil
	}
	briefs, appErr := h.galgameClient.GetBatch(ctx, []int{int(gid)})
	if appErr != nil {
		slog.Warn("galgame edit: brief lookup failed", "gid", gid, "error", appErr)
		return nil
	}
	if b, ok := briefs[int(gid)]; ok {
		return &b
	}
	return nil
}

// isGameOwner reports whether uid created the galgame row. Fail-closed: an
// unreachable galgame degrades the owner assertion to false (moderators are
// unaffected; the owner retries).
func (h *EditHandler) isGameOwner(ctx context.Context, gid, uid int64) bool {
	b := h.gameBrief(ctx, gid)
	return b != nil && b.UserID > 0 && int64(b.UserID) == uid
}

func briefName(b *client.GalgameBrief) string {
	if b == nil {
		return ""
	}
	for _, n := range []string{b.NameZhCn, b.NameZhTw, b.NameJaJp, b.NameEnUs} {
		if n != "" {
			return n
		}
	}
	return ""
}

// notifyDecision sends the proposer a forum in-site message about a merge /
// decline (E3b ruling 1). Best-effort and never rolls back the decision;
// Emit itself swallows self-notifications.
func (h *EditHandler) notifyDecision(prop *catalogclient.EditProposal, senderID int64, kind msgService.NotifyKind, content string) {
	if h.notifier == nil {
		return
	}
	if err := h.notifier.Emit(nil, msgService.Spec{
		SenderID: int(senderID), ReceiverID: int(prop.ProposerUID),
		Kind: kind, Content: content, GalgameID: int(prop.EntityID),
	}); err != nil {
		slog.Warn("galgame edit: decision notification failed",
			"proposal", prop.ID, "kind", kind, "error", err)
	}
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
	// Owner-review (E3b): the game's creator projects can_review on the
	// default keys — the capability rides the schema projection, the UI
	// holds zero policy logic.
	actor.IsEntityOwner = h.isGameOwner(ctx, gid, actor.UserID)
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

// Submit — POST /galgame/:gid/edit/proposals (auth). Files the proposal. On
// kungal a reviewer's own edit direct-merges (automerge=review): admin/ren via
// the review perm, and the game's owner via OwnerReview — result.Merged, the
// change applies immediately. Everyone else's proposal stays open for the queue.
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
	// Owner direct-edit (automerge=review): the game's creator reviews the
	// default keys, so — like admin/ren via perm — their own edit applies
	// immediately instead of queuing a proposal against themselves. editActor
	// asserts roles but not ownership; mirror Bootstrap's owner check so the
	// engine sees the same capability. Only for a valid request (avoids the
	// S2S brief lookup on rejected input; the brief is warm from Bootstrap).
	actor.IsEntityOwner = h.isGameOwner(c.Context(), gid, actor.UserID)
	result, err := h.catalog.CreateEditProposal(c.Context(), catalogclient.EditCreateRequest{
		EntityType: entityTypeGame, EntityID: gid, Site: catalogSite,
		Patch: req.Patch, Note: req.Note, Actor: actor,
	})
	if err != nil {
		return editError(c, err)
	}
	if !result.Merged {
		h.submitSideEffects(c.Context(), &result.Proposal)
	}
	out := fiber.Map{"merged": result.Merged, "proposal": result.Proposal}
	if result.Revision != nil {
		out["revision"] = result.Revision
	}
	return response.OK(c, out)
}

// submitSideEffects mirrors the old SubmitPR chain onto a new open proposal
// (best-effort): a "requested" notice to the game's owner and a
// GALGAME_PR_CREATION row on the activity timeline. The proposal id continues
// the old galgame PR id space (the E2 transform bumped the sequence past it), so
// wiki_pr_id stays the idempotency key.
func (h *EditHandler) submitSideEffects(ctx context.Context, prop *catalogclient.EditProposal) {
	brief := h.gameBrief(ctx, prop.EntityID)
	if h.notifier != nil && brief != nil && brief.UserID > 0 {
		if err := h.notifier.Emit(nil, msgService.Spec{
			SenderID: int(prop.ProposerUID), ReceiverID: brief.UserID,
			Kind: msgService.NotifyRequested, Content: briefName(brief),
			GalgameID: int(prop.EntityID),
		}); err != nil {
			slog.Warn("galgame edit: requested notification failed", "proposal", prop.ID, "error", err)
		}
	}
	if h.repo != nil {
		if err := h.repo.DB().WithContext(ctx).Exec(`
			INSERT INTO galgame_activity (wiki_pr_id, galgame_id, user_id, type, created)
			VALUES (?, ?, ?, 'GALGAME_PR_CREATION', now())
			ON CONFLICT (wiki_pr_id) DO NOTHING
		`, prop.ID, prop.EntityID, prop.ProposerUID).Error; err != nil {
			slog.Warn("galgame edit: activity timeline write failed", "proposal", prop.ID, "error", err)
		}
	}
}

// Revisions — GET /galgame/:gid/edit/revisions (optional auth; public like
// the galgame's revision history always was). Includes the E2-migrated history.
// A logged-in reviewer — moderator or the game's creator (E3b) — additionally
// gets can_revert so the history page can offer the revert control.
func (h *EditHandler) Revisions(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	items, err := h.catalog.ListEditRevisions(c.Context(), entityTypeGame, gid, queryInt(c, "limit"))
	if err != nil {
		return editError(c, err)
	}
	uids := make(map[int]bool)
	for i := range items {
		uids[int(items[i].ActorUID)] = true
		if items[i].AmenderUID != nil {
			uids[int(*items[i].AmenderUID)] = true
		}
	}
	canRevert := false
	if user := middleware.GetUser(c); user != nil {
		// Revert is review-grade: admin ⊂ ren, or the game's owner. Moderators
		// are NOT included — they hold no edit.galgame.game.review, so the
		// engine would reject their revert anyway.
		canRevert = role.CanAdminister(user.Roles) ||
			h.isGameOwner(c.Context(), gid, int64(user.ID))
	}
	return response.OK(c, fiber.Map{
		"gid": gid, "items": items, "users": h.userMap(c.Context(), uids),
		"can_revert": canRevert,
	})
}

// editRevertRequest restores the game to a historical revision.
type editRevertRequest struct {
	ToSeq int    `json:"to_seq"`
	Note  string `json:"note"`
}

// Revert — POST /galgame/:gid/edit/revert (auth; moderator or the game's
// creator — parity with the old wire's owner-or-admin revert). The engine
// enforces the field-level review rule on every restored field, so an owner
// can only revert what they may adjudicate (the kungal default keys).
func (h *EditHandler) Revert(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	actor, appErr := editActor(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req editRevertRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	if req.ToSeq < 1 {
		return response.Error(c, errors.ErrBadRequest("需要目标版本号"))
	}
	if len(req.Note) > 2000 {
		return response.Error(c, errors.ErrValidation("说明过长"))
	}
	ctx := c.Context()
	actor.IsEntityOwner = h.isGameOwner(ctx, gid, actor.UserID)
	// Review-grade gate: admin ⊂ ren, or the game's owner (mirrors can_revert
	// and the engine's per-field review rule; moderators excluded).
	//
	// INFRA-PROXY mirror: mirrors infra key `edit.galgame.game.review` /
	// `galgame.owner_override` (admin/ren or owner). The engine re-checks every
	// restored field, so this stays a role/owner gate, not a pkg/perm key.
	if !role.CanAdminister(actor.Roles) && !actor.IsEntityOwner {
		return response.Error(c, errors.ErrForbidden("你没有权限执行此操作"))
	}
	result, err := h.catalog.RevertEditEntity(ctx, catalogSite, entityTypeGame, gid, req.ToSeq, req.Note, actor)
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, result)
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
// galgame brief (nil when the galgame batch read fails — the UI falls back to
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
	if len(ids) > 0 && h.galgameClient != nil {
		var appErr *errors.AppError
		if briefs, appErr = h.galgameClient.GetBatch(ctx, ids); appErr != nil {
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
	return response.OK(c, fiber.Map{
		"items": h.enrich(c.Context(), items),
		"users": h.userMap(c.Context(), collectProposalUIDs(items)),
	})
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
	return response.OK(c, fiber.Map{
		"items": h.enrich(c.Context(), items),
		"users": h.userMap(c.Context(), collectProposalUIDs(items)),
	})
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

// stampOwner loads the proposal (pinned to the kungal tenant) and stamps
// is_entity_owner onto the actor — the shared preamble both review gates run
// before applying their own threshold.
func (h *EditHandler) stampOwner(ctx context.Context, id int64, actor *catalogclient.EditActor) (*catalogclient.EditProposal, error) {
	prop, err := h.proposalForReview(ctx, id)
	if err != nil {
		return nil, err
	}
	actor.IsEntityOwner = h.isGameOwner(ctx, prop.EntityID, actor.UserID)
	return prop, nil
}

// reviewEntry authorizes the proposal VIEW surface (ProposalDetail, E3b
// owner-review): the entry admits moderators AND the game's creator; everyone
// else 403s. The owner assertion is stamped onto the actor — the engine's
// kungal overlay holds the field-level policy.
//
// INFRA-PROXY mirror: mirrors infra key `galgame.review` (moderator+, plus the
// owner overlay) — this is a read gate, so the moderator threshold is correct.
// Adjudication is stricter; see decideEntry.
func (h *EditHandler) reviewEntry(ctx context.Context, id int64, actor *catalogclient.EditActor) (*catalogclient.EditProposal, error) {
	prop, err := h.stampOwner(ctx, id, actor)
	if err != nil {
		return nil, err
	}
	if !role.CanModerate(actor.Roles) && !actor.IsEntityOwner {
		return nil, &catalogclient.EditAPIError{Status: http.StatusForbidden, Message: "review entry denied"}
	}
	return prop, nil
}

// decideEntry authorizes the proposal ADJUDICATION surfaces (Amend / Merge /
// Decline): admits admin ⊂ ren AND the game's creator, but NOT a plain
// moderator. This closes a real bug — a bare moderator used to pass this forum
// gate, click merge, then get 403'd by infra, whose edit.galgame.game.review is
// admin/ren-only (the owner overlay grants owners the default-field review).
//
// INFRA-PROXY mirror: mirrors infra key `edit.galgame.game.review` (admin/ren
// or the entity owner).
func (h *EditHandler) decideEntry(ctx context.Context, id int64, actor *catalogclient.EditActor) (*catalogclient.EditProposal, error) {
	prop, err := h.stampOwner(ctx, id, actor)
	if err != nil {
		return nil, err
	}
	if !role.CanAdminister(actor.Roles) && !actor.IsEntityOwner {
		return nil, &catalogclient.EditAPIError{Status: http.StatusForbidden, Message: "decide entry denied"}
	}
	return prop, nil
}

// GameProposals — GET /galgame/:gid/edit/proposals (public — the old wire's
// per-game PR list was a public read). One game's proposals, open by
// default: the owner's per-game review surface and everyone's transparency
// read.
func (h *EditHandler) GameProposals(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	status := c.Query("status", "open")
	switch status {
	case "open", "merged", "declined", "withdrawn", "":
	default:
		return response.Error(c, errors.ErrBadRequest("未知的提案状态"))
	}
	items, err := h.catalog.ListEditProposals(c.Context(), catalogclient.EditProposalFilter{
		EntityType: entityTypeGame, EntityID: gid, Site: catalogSite,
		Status: status, Limit: queryInt(c, "limit"),
	})
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, fiber.Map{
		"gid": gid, "items": items,
		"users": h.userMap(c.Context(), collectProposalUIDs(items)),
	})
}

// ProposalDetail — GET /galgame-edit/proposals/:id (auth; moderator or the
// game's creator). The review workbench read: proposal + amendments +
// effective patch, the entity's CURRENT values (per-field old→new compare),
// the reviewer's capability projection, and the galgame brief.
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
	prop, err := h.reviewEntry(ctx, id, &actor)
	if err != nil {
		return editError(c, err)
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
		"users":    h.userMap(ctx, collectProposalUIDs([]catalogclient.EditProposal{*prop})),
		// can_decide projects the decideEntry predicate for the UI: only
		// admin/ren or the game's owner may amend/merge/decline (a plain
		// moderator can view but not adjudicate). actor.IsEntityOwner was
		// stamped by reviewEntry above.
		"can_decide": role.CanAdminister(actor.Roles) || actor.IsEntityOwner,
	})
}

// editAmendRequest carries the maintainer delta: corrected values (set) and
// rejected fields (unset).
type editAmendRequest struct {
	Set   map[string]any `json:"set"`
	Unset []string       `json:"unset"`
	Note  string         `json:"note"`
}

// Amend — POST /galgame-edit/proposals/:id/amend (auth; moderator or the
// game's creator). The crown mechanism: correct a value / reject a field
// before merging; the merged revision carries proposer + amender double
// attribution.
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
	if _, err := h.decideEntry(ctx, id, &actor); err != nil {
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

// Merge — POST /galgame-edit/proposals/:id/merge (auth; moderator or the
// game's creator).
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
	prop, err := h.decideEntry(ctx, id, &actor)
	if err != nil {
		return editError(c, err)
	}
	rev, err := h.catalog.MergeEditProposal(ctx, id, req.Note, actor)
	if err != nil {
		return editError(c, err)
	}
	h.mergeSideEffects(ctx, prop, actor.UserID, rev)
	return response.OK(c, rev)
}

// mergeSideEffects mirrors the old MergePR chain (best-effort, never fails
// the landed merge): the contribution moemoepoint (stable per-proposal
// idempotency key — a merge is exactly-once per proposal; self-merges earn
// nothing, matching the old chain), the local resource_update_time bump so
// the game rises in the update-sorted list, and the "merged" notice to the
// proposer — its content marks a reviewer correction when the merged
// revision carries an amender (E3b ruling 1).
func (h *EditHandler) mergeSideEffects(ctx context.Context, prop *catalogclient.EditProposal, mergerID int64, rev *catalogclient.EditRevision) {
	if prop.ProposerUID != mergerID {
		moemoepoint.Award(int(prop.ProposerUID), constants.RewardPRMerge,
			moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame_pr", int(prop.EntityID)),
			moemoepoint.Key("galgame_edit_merged", strconv.FormatInt(prop.ID, 10)))
	}
	if h.repo != nil {
		if err := h.repo.Touch(h.repo.DB().WithContext(ctx), int(prop.EntityID)); err != nil {
			slog.Warn("galgame edit: resource_update_time bump failed", "gid", prop.EntityID, "error", err)
		}
	}
	content := briefName(h.gameBrief(ctx, prop.EntityID))
	if rev != nil && rev.AmenderUID != nil {
		content = strings.TrimSpace(content + "（审核时有修正）")
	}
	h.notifyDecision(prop, mergerID, msgService.NotifyMerged, content)
}

// Decline — POST /galgame-edit/proposals/:id/decline (auth; moderator or
// the game's creator). The reason is required — a silent decline was the
// old galgame's worst reviewer habit; it travels to the proposer in full on
// the decline notice (E3b ruling 1).
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
	target, err := h.decideEntry(ctx, id, &actor)
	if err != nil {
		return editError(c, err)
	}
	prop, err := h.catalog.DeclineEditProposal(ctx, id, req.Note, actor)
	if err != nil {
		return editError(c, err)
	}
	content := req.Note
	if name := briefName(h.gameBrief(ctx, target.EntityID)); name != "" {
		content = name + "：" + req.Note
	}
	h.notifyDecision(target, actor.UserID, msgService.NotifyDeclined, content)
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
