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

// EditHandler is the BFF face onto the infra editing engine: the schema-driven
// galgame editor, the kungal review queue (amend / merge / decline — the
// maintainer-edit crown), the revision history + diff reads, and the "my
// proposals" list.
//
// EVERY act travels on the acting user's OWN OAuth token (`catalog:edit`
// scope) since wave 178. The catalog derives uid, roles, the acting site AND —
// the piece that finished the migration — whether the subject owns the target
// work (catalog_work.owner_user_id, backfilled from galgame.creator_user_id)
// straight from the token. So kungal asserts nothing about anybody any more,
// and the mirrored permission gates it used to run before each write are gone
// with the assertions: authorization is infra's, in one place, and this handler
// only maps the answer (see userEditError; a stale grant surfaces as code 235,
// "log out and back in", never as a plain 403).
//
// Wave 180 took the human-serving READS the same way — for three DIFFERENT
// reasons, which are worth keeping straight because only one of them is a gate:
//
//   - "my proposals" was the session uid written into a query parameter, the
//     last assertion in this handler. It moved to delete that: the question has
//     one honest answer and `mine=true` lets the catalog give it.
//   - the proposal DETAIL is genuinely fenced upstream — the user-plane get
//     answers for the token's own tenant — so this one buys a real check the
//     Basic lane could not make.
//   - the value SNAPSHOT buys no gate at all. Its op is deliberately not
//     tenant-fenced (the values are the same entity state a public read
//     renders); it moved for channel hygiene, so that no read a HUMAN triggers
//     is left on a lane where the forum is the authenticated party.
//
// What remains Basic-authed is what no person triggers on their own behalf:
// the revision log, the diff, and the public per-game proposal list.
//
// The forum still answers "who created this entry" (EntryOwners /
// galgame.creator_user_id) but ONLY as a view/UX fact: which surfaces a creator
// may open, and which control the page offers. Never as an authorization —
// infra decides that, and a UI gate that disagreed would merely be wrong in a
// direction the write path corrects.
//
// Degradation: an unconfigured catalog client → 503 on every endpoint and
// the frontend hides the entries. Never a local fallback.
//
// Decision side effects (E3b ruling 1 — notifications are the product's job,
// the engine stays notification-free): merge/decline notify the proposer via
// the forum's own message system, merge additionally awards the contribution
// moemoepoint and bumps the local resource_update_time; a submit notifies the
// game owner and mirrors onto the activity timeline — all parity with the old
// PR chain, all best-effort (a failure logs a warning, never rolls back the
// decision).
type EditHandler struct {
	catalog       *catalogclient.Client // both catalog faces: the user-token plane + the claim-free S2S reads
	galgameClient *client.GalgameClient // best-effort brief enrichment + owner lookup
	users         *userclient.Client    // best-effort attribution enrichment
	notifier      msgService.Notifier   // best-effort decision notifications
	repo          *repository.GalgameRepository
	// owners answers "who submitted this entry", by gid — a VIEW fact now (which
	// surfaces a creator may open), no longer an authorization input. It stays a
	// narrow port rather than a direct repo read because it must keep agreeing
	// with the author chip the product renders from the same column.
	owners EntryOwners
}

// EntryOwners resolves an entry's submitter, from the forum's own frozen
// snapshot (galgame.creator_user_id, migration 066) — the same column the author
// chip renders.
//
// The catalog holds its own copy of this fact now (catalog_work.owner_user_id,
// backfilled from here) and derives edit capability from it. This one survives
// for the VIEW gates only; nothing here decides whether a write is allowed.
type EntryOwners interface {
	// OwnerOf returns the submitter's uid, or 0 when unknown. Unknown fails the
	// owner view check CLOSED; moderators are unaffected.
	OwnerOf(gid int) int
}

// repoOwners adapts the galgame repository to EntryOwners.
type repoOwners struct{ repo *repository.GalgameRepository }

func (r repoOwners) OwnerOf(gid int) int {
	if r.repo == nil || gid <= 0 {
		return 0
	}
	row := r.repo.FindLocal(gid)
	if row.CreatorUserID == nil {
		return 0
	}
	return *row.CreatorUserID
}

func NewEditHandler(
	catalog *catalogclient.Client,
	galgameClient *client.GalgameClient, users *userclient.Client,
	notifier msgService.Notifier, repo *repository.GalgameRepository,
) *EditHandler {
	return &EditHandler{
		catalog: catalog, galgameClient: galgameClient,
		users: users, notifier: notifier, repo: repo, owners: repoOwners{repo: repo},
	}
}

// WithOwners swaps the submitter lookup. Only the assembly point and tests use
// it; production always gets the repository adapter.
func (h *EditHandler) WithOwners(owners EntryOwners) *EditHandler {
	h.owners = owners
	return h
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

// entityTypeGame is the entity type kungal's edits target.
//
// It is the REGISTRY's work now, not the wiki's galgame row — same history,
// same proposals, same revision numbers (the rekey moved them), but a different
// ID SPACE. Every id crossing this boundary therefore has to be translated:
// a kungal URL is a gid, the engine speaks registry work ids, and the two
// overlap, so a missed translation links to a different game instead of
// failing.
const entityTypeGame = catalogclient.EntityTypeWork

// fieldKeyPrefix guards the pass-through patch: only this family's keys may
// ride this BFF (the engine re-validates each key against the registry).
const fieldKeyPrefix = catalogclient.FieldKeyPrefix

var errEditDown = errors.New(errors.CodeBiz, "资料库编辑服务暂不可用", http.StatusServiceUnavailable)

// ownerOf answers "who created this entry", by gid — the forum's own
// `galgame.creator_user_id`, the frozen wiki-era submitter snapshot (migration
// 066) that already backs the author chip on every card. Reading the SAME column
// the product renders is the point: a creator-only surface that disagreed with
// the author shown on screen would be impossible to explain.
//
// 0 = unknown, which fails the owner view check closed. Moderators are
// unaffected, and no write depends on this answer.
func (h *EditHandler) ownerOf(_ context.Context, gid int64) int {
	if h.owners == nil {
		return 0
	}
	return h.owners.OwnerOf(int(gid))
}

// isGameOwner reports whether uid created the entry behind a REGISTRY work id.
// Fail-closed: an unresolvable work degrades the owner assertion to false.
func (h *EditHandler) isGameOwner(ctx context.Context, workID, uid int64) bool {
	gid := h.gidOf(ctx, workID)
	if gid == 0 {
		return false
	}
	owner := h.ownerOf(ctx, int64(gid))
	return owner > 0 && int64(owner) == uid
}

// workIDOf translates a kungal gid into the registry work id the engine keys
// on. A gid the registry does not know has no editable entity.
func (h *EditHandler) workIDOf(ctx context.Context, gid int64) (int64, *errors.AppError) {
	if h.galgameClient == nil {
		// No bridge, no editable entity. The edit face degrades as a whole (the
		// frontend hides its entries) rather than falling back to a gid, which
		// would address a different work.
		return 0, errEditDown
	}
	ids, appErr := h.galgameClient.CatalogWorkIDs(ctx, []int{int(gid)})
	if appErr != nil {
		return 0, appErr
	}
	workID, ok := ids[int(gid)]
	if !ok {
		return 0, errors.ErrNotFound("条目不存在")
	}
	return workID, nil
}

// editEntry is one entry resolved from a registry work id into everything the
// side-effect lanes need: the kungal id its rows and URLs are keyed by, the
// author the owner-review gate and the "someone proposed an edit" notice are
// addressed to, and a title for the notice body.
//
// It is resolved ONCE per decision rather than looked up per use — the three
// facts come from two different places (the registry for the title, the forum
// for the author) and reading them separately at three call sites is how they
// start disagreeing.
type editEntry struct {
	GID      int
	OwnerUID int
	Name     string
}

// entryOf resolves a registry work id. A zero GID means kungal does not claim
// the work, and every side effect keyed on it is then correctly skipped.
func (h *EditHandler) entryOf(ctx context.Context, workID int64) editEntry {
	gid := h.gidOf(ctx, workID)
	if gid == 0 {
		return editEntry{}
	}
	entry := editEntry{GID: gid, OwnerUID: h.ownerOf(ctx, int64(gid))}
	if h.galgameClient != nil {
		if rows, appErr := h.galgameClient.CatalogRowsByGIDs(ctx, []int{gid}, "names", "all"); appErr == nil {
			if row, ok := rows[gid]; ok {
				brief := client.CatalogItemToBrief(&row)
				entry.Name = client.BriefName(&brief)
			}
		}
	}
	return entry
}

// gidOf is the reverse: one registry work id home to its gid, 0 when kungal
// does not claim it.
func (h *EditHandler) gidOf(ctx context.Context, workID int64) int {
	if h.galgameClient == nil {
		return 0
	}
	gids, appErr := h.galgameClient.GIDsByCatalogIDs(ctx, []int64{workID})
	if appErr != nil {
		slog.Warn("galgame edit: work id → gid failed", "work", workID, "error", appErr)
		return 0
	}
	return gids[workID]
}

// notifyDecision sends the proposer a forum in-site message about a merge /
// decline. gid — NOT the proposal's entity id — because a forum message links
// by kungal id; a registry id in that column points the notice at a different
// entry, and 0 is the honest "no link" the message system already handles.
func (h *EditHandler) notifyDecision(prop *catalogclient.EditProposal, gid int, senderID int64, kind msgService.NotifyKind, content string) {
	if h.notifier == nil {
		return
	}
	if err := h.notifier.Emit(nil, msgService.Spec{
		SenderID: int(senderID), ReceiverID: int(prop.ProposerUID),
		Kind: kind, Content: content, GalgameID: gid,
	}); err != nil {
		slog.Warn("galgame edit: decision notification failed",
			"proposal", prop.ID, "kind", kind, "error", err)
	}
}

// editStatusError maps the edit engine's 4xx taxonomy onto the house envelope.
// BOTH planes share it: a lane that took the user token must not report a
// validation failure or a rebase conflict differently from the same lane on the
// S2S channel — which plane carried the write is an implementation detail the
// user never chose. nil = a status with no user-facing meaning (the caller
// degrades it to 503).
func editStatusError(status int, message string) *errors.AppError {
	switch status {
	case http.StatusForbidden:
		return errors.ErrForbidden("你没有权限执行此操作")
	case http.StatusNotFound:
		return errors.ErrNotFound("条目或提案不存在")
	case http.StatusUnprocessableEntity:
		return errors.ErrValidation(message)
	case http.StatusConflict:
		return errors.New(errors.CodeBiz,
			"操作冲突（条目已被他人修改或提案已关闭），请刷新后重试", http.StatusConflict)
	}
	return nil
}

// editError maps a catalogclient S2S edit error onto the house envelope. The
// edit face's 4xx replies carry actionable reasons (validation details,
// policy denials, rebase conflicts) — surface them; transport failures and
// the unconfigured client degrade to 503.
func editError(c fiber.Ctx, err error) error {
	var apiErr *catalogclient.EditAPIError
	switch {
	case stderrors.Is(err, catalogclient.ErrNotConfigured):
		return response.Error(c, errEditDown)
	case stderrors.As(err, &apiErr):
		if appErr := editStatusError(apiErr.Status, apiErr.Message); appErr != nil {
			return response.Error(c, appErr)
		}
		slog.Error("galgame edit: upstream error", "status", apiErr.Status, "code", apiErr.Code, "msg", apiErr.Message)
		return response.Error(c, errEditDown)
	default:
		slog.Warn("galgame edit: catalog unreachable", "error", err)
		return response.Error(c, errEditDown)
	}
}

// userEditError is the same mapping for the USER-TOKEN plane (wave 177), plus
// the two denials only a forwarded token can produce.
//
// The scope case is the one that must not be folded into the generic 403: the
// user's grant predates `catalog:edit` and no refresh can widen it, so the only
// fix is a re-login. Code 235 is what the frontend keys the re-login prompt on;
// a 233 there would tell a user they are not allowed to edit when in fact they
// are, and a 205 would log out a perfectly live session.
func userEditError(c fiber.Ctx, err error) error {
	var apiErr *catalogclient.UserAPIError
	switch {
	case stderrors.Is(err, catalogclient.ErrInsufficientScope):
		return response.Error(c, errors.ErrReauthRequired(
			"编辑资料需要新的授权，请退出登录后重新登录以授予该权限"))
	case stderrors.Is(err, catalogclient.ErrUnauthorized):
		return response.Error(c, errors.ErrAuthExpired())
	case stderrors.Is(err, catalogclient.ErrNotFound):
		return response.Error(c, errors.ErrNotFound("条目或提案不存在"))
	case stderrors.Is(err, catalogclient.ErrNotConfigured):
		return response.Error(c, errEditDown)
	case stderrors.As(err, &apiErr):
		if appErr := editStatusError(apiErr.Status, apiErr.Message); appErr != nil {
			return response.Error(c, appErr)
		}
		slog.Error("galgame edit: user-plane upstream error",
			"status", apiErr.Status, "code", apiErr.Code, "msg", apiErr.Message)
		return response.Error(c, errEditDown)
	default:
		slog.Warn("galgame edit: catalog user plane unreachable", "error", err)
		return response.Error(c, errEditDown)
	}
}

// userToken is the session's own OAuth access token, the credential the user
// lanes travel on. The browser never holds it (kun_session keeps it opaque in
// Redis), which is why these writes traverse kungal at all; an empty one is a
// dead session, not something to ask the catalog about.
func userToken(c fiber.Ctx) (string, *errors.AppError) {
	token := middleware.GetAccessToken(c)
	if token == "" {
		return "", errors.ErrAuthExpired()
	}
	return token, nil
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

// queryInt reads a non-negative int query param, returning 0 when absent /
// invalid (the catalog then applies its own default page size).
func queryInt(c fiber.Ctx, key string) int {
	n, err := strconv.Atoi(c.Query(key))
	if err != nil || n < 0 {
		return 0
	}
	return n
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
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	ctx := c.Context()
	workID, appErr := h.workIDOf(ctx, gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	values, err := h.catalog.EditSnapshotUser(ctx, token, entityTypeGame, workID)
	if err != nil {
		return userEditError(c, err)
	}
	// The schema projects can_review / would_automerge for the CALLER, and the
	// caller's writes all take this same plane — including the creator's, whose
	// ownership the catalog now derives from the token instead of hearing it
	// asserted. One projection, one plane, no way for the editor to render a
	// capability the submit lane does not have.
	schema, err := h.catalog.GetEditSchemaUser(ctx, token, entityTypeGame, workID)
	if err != nil {
		return userEditError(c, err)
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

// Submit — POST /galgame/:gid/edit/proposals (auth). Files the proposal as the
// user. On kungal a reviewer's own edit direct-merges (automerge=review):
// admin/ren via the review perm, and the game's owner via OwnerReview — both
// decided upstream from the token, both reported back as result.Merged.
// Everyone else's proposal stays open for the queue.
func (h *EditHandler) Submit(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
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
	// Only for a valid request (avoids the S2S bridge lookup on rejected input).
	workID, appErr := h.workIDOf(c.Context(), gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	// One plane for everyone — contributor, staff and the entry's creator alike.
	// The creator's direct-merge used to need an asserted is_entity_owner; the
	// catalog owns that fact now and reads it off the token, so the outcome that
	// once justified a second code path comes back through this one as
	// result.Merged, which is the only thing this handler ever read.
	result, err := h.catalog.CreateEditProposalUser(c.Context(), token, catalogclient.UserEditCreateRequest{
		EntityType: entityTypeGame, EntityID: workID,
		Patch: req.Patch, Note: req.Note,
	})
	if err != nil {
		return userEditError(c, err)
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
	// EntityID is a REGISTRY id; every row below is keyed by gid, so it comes
	// home first. Writing the work id into galgame_activity.galgame_id would
	// attach the card to whichever entry happens to hold that gid.
	entry := h.entryOf(ctx, prop.EntityID)
	if entry.GID == 0 {
		return
	}
	if h.notifier != nil && entry.OwnerUID > 0 {
		if err := h.notifier.Emit(nil, msgService.Spec{
			SenderID: int(prop.ProposerUID), ReceiverID: entry.OwnerUID,
			Kind: msgService.NotifyRequested, Content: entry.Name,
			GalgameID: entry.GID,
		}); err != nil {
			slog.Warn("galgame edit: requested notification failed", "proposal", prop.ID, "error", err)
		}
	}
	if h.repo != nil {
		if err := h.repo.DB().WithContext(ctx).Exec(`
			INSERT INTO galgame_activity (wiki_pr_id, galgame_id, user_id, type, created)
			VALUES (?, ?, ?, 'GALGAME_PR_CREATION', now())
			ON CONFLICT (wiki_pr_id) DO NOTHING
		`, prop.ID, entry.GID, prop.ProposerUID).Error; err != nil {
			slog.Warn("galgame edit: activity timeline write failed", "proposal", prop.ID, "error", err)
		}
	}
}

// Revisions — GET /galgame/:gid/edit/revisions (optional auth; public like
// the galgame's revision history always was). Includes the E2-migrated history.
// A logged-in caller additionally gets can_revert — projected from their own
// capabilities, see canRevert — so the history page can offer the control.
func (h *EditHandler) Revisions(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	workID, appErr := h.workIDOf(c.Context(), gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	items, err := h.catalog.ListEditRevisions(c.Context(), entityTypeGame, workID, queryInt(c, "limit"))
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
	return response.OK(c, fiber.Map{
		"gid": gid, "items": items, "users": h.userMap(c.Context(), uids),
		"can_revert": h.canRevert(c, workID),
	})
}

// canRevert projects, for the CALLER, whether the history page should offer the
// revert control.
//
// The predicate: the viewer's own capability projection loads, reports at least
// one field, and EVERY field that is still editable at all (not locked, not
// deprecated) reports can_review. That is the honest reading of what a revert
// is — it restores the whole registered field set at once, so it is offerable
// only to somebody who may adjudicate all of it. A viewer who may review some
// fields but not others would have the button 403 on them, which is worse than
// not showing it.
//
// No token → false, and any projection failure → false: the route is optionally
// authed, and a public reader simply has no capabilities to project. This is a
// UX gate; the engine re-checks every restored field on the write anyway.
func (h *EditHandler) canRevert(c fiber.Ctx, workID int64) bool {
	token := middleware.GetAccessToken(c)
	if token == "" {
		return false
	}
	schema, err := h.catalog.GetEditSchemaUser(c.Context(), token, entityTypeGame, workID)
	if err != nil {
		slog.Warn("galgame edit: revert projection failed", "work", workID, "error", err)
		return false
	}
	editable := 0
	for _, f := range schema.Fields {
		if f.Locked || f.Deprecated {
			continue
		}
		editable++
		if !f.CanReview {
			return false
		}
	}
	return editable > 0
}

// editRevertRequest restores the game to a historical revision.
type editRevertRequest struct {
	ToSeq int    `json:"to_seq"`
	Note  string `json:"note"`
}

// Revert — POST /galgame/:gid/edit/revert (auth). Restores the game to a
// historical revision as the user: infra enforces the field-level review rule on
// every restored field, so the caller can only revert what they may adjudicate.
// The forum runs no gate of its own here — it used to mirror one, and a mirror
// is only ever a second answer to a question that already has one.
func (h *EditHandler) Revert(c fiber.Ctx) error {
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
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
	workID, appErr := h.workIDOf(ctx, gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	result, err := h.catalog.RevertEditEntityUser(ctx, token, entityTypeGame, workID, req.ToSeq, req.Note)
	if err != nil {
		return userEditError(c, err)
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
	workID, appErr := h.workIDOf(c.Context(), gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	diff, err := h.catalog.DiffEditRevisions(c.Context(), entityTypeGame, workID, from, to)
	if err != nil {
		return editError(c, err)
	}
	return response.OK(c, diff)
}

// ──────────────────────────────────────────
// Review queue + my proposals (proposal-scoped)
// ──────────────────────────────────────────

// proposalItem is one enriched list row: the proposal, the kungal id it belongs
// to, and a best-effort brief.
//
// GID is not decoration — it is the ONLY safe way for a client to link to the
// entry. `entity_id` is a registry work id whose space OVERLAPS kungal's, so a
// UI that builds /galgame/{entity_id} lands on a different game and reports no
// error at all. Every list that reaches a template therefore carries the
// translated id, and the templates read that.
type proposalItem struct {
	catalogclient.EditProposal
	GID     int                  `json:"gid"`
	Galgame *client.GalgameBrief `json:"galgame,omitempty"`
}

func (h *EditHandler) enrich(ctx context.Context, items []catalogclient.EditProposal) []proposalItem {
	workIDs := make([]int64, 0, len(items))
	seen := make(map[int64]bool, len(items))
	for i := range items {
		if id := items[i].EntityID; !seen[id] {
			seen[id] = true
			workIDs = append(workIDs, id)
		}
	}
	var gidByWork map[int64]int
	if len(workIDs) > 0 && h.galgameClient != nil {
		var appErr *errors.AppError
		if gidByWork, appErr = h.galgameClient.GIDsByCatalogIDs(ctx, workIDs); appErr != nil {
			slog.Warn("galgame edit: work id → gid enrichment failed", "error", appErr)
		}
	}
	gids := make([]int, 0, len(gidByWork))
	for _, gid := range gidByWork {
		gids = append(gids, gid)
	}
	// The cards show the entry's title. `content_limit=all` because a review
	// queue that cannot see an entry's name cannot review it — the editorial
	// gate is a reader preference, not an authorization.
	var briefs map[int]client.GalgameBrief
	if len(gids) > 0 && h.galgameClient != nil {
		rows, appErr := h.galgameClient.CatalogRowsByGIDs(ctx, gids, "names,covers", "all")
		if appErr != nil {
			slog.Warn("galgame edit: brief enrichment failed", "error", appErr)
		} else {
			briefs = make(map[int]client.GalgameBrief, len(rows))
			for gid := range rows {
				row := rows[gid]
				briefs[gid] = client.CatalogItemToBrief(&row)
			}
		}
	}
	out := make([]proposalItem, 0, len(items))
	for i := range items {
		item := proposalItem{EditProposal: items[i], GID: gidByWork[items[i].EntityID]}
		if b, ok := briefs[item.GID]; ok {
			brief := b
			item.Galgame = &brief
		}
		out = append(out, item)
	}
	return out
}

// Queue — GET /galgame-edit/queue (auth). Open proposals on the caller's
// tenant, newest-first; ?status widens to the decided ones.
//
// Authority is the TOKEN's edit review permission, checked by the catalog:
// the queue face serves everybody's proposals only to a subject who may
// adjudicate them, so a contributor's token gets a 403 from infra rather than
// the queue. The route's RequireModerator survives as a pure VIEW gate — which
// nav entry the console offers — on the same ruling as the wave 178/179 review
// routes: a local mirror may decide what to render, never what is allowed, and
// a permission-console grant takes effect without a forum deploy.
func (h *EditHandler) Queue(c fiber.Ctx) error {
	status := c.Query("status", "open")
	switch status {
	case "open", "merged", "declined", "withdrawn", "":
	default:
		return response.Error(c, errors.ErrBadRequest("未知的提案状态"))
	}
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	items, err := h.catalog.ListEditProposalsUser(c.Context(), token, catalogclient.UserEditProposalFilter{
		EntityType: entityTypeGame,
		Status:     status, Limit: queryInt(c, "limit"),
	})
	if err != nil {
		return userEditError(c, err)
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
	// "Mine" is the token's, not a uid this handler names. The proposer filter
	// that used to ride the S2S list face was the last assertion left here; it
	// is gone because the question has exactly one honest answer and the
	// catalog already holds it.
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	// ?gid narrows to one entry — a kungal id, so it crosses the bridge before
	// it can name an entity.
	var entityID int64
	if gid := queryInt(c, "gid"); gid > 0 {
		workID, appErr := h.workIDOf(c.Context(), int64(gid))
		if appErr != nil {
			return response.Error(c, appErr)
		}
		entityID = workID
	}
	items, err := h.catalog.ListEditProposalsUser(c.Context(), token, catalogclient.UserEditProposalFilter{
		EntityType: entityTypeGame, EntityID: entityID, Mine: true,
		Status: c.Query("status"), Limit: queryInt(c, "limit"),
	})
	if err != nil {
		return userEditError(c, err)
	}
	return response.OK(c, fiber.Map{
		"items": h.enrich(c.Context(), items),
		"users": h.userMap(c.Context(), collectProposalUIDs(items)),
	})
}

// proposalForReview loads the proposal AS the caller (wave 180) and pins it to
// kungal's own tenant + the galgame entity type — this BFF adjudicates nothing
// else.
//
// The tenant pin is kept even though the catalog now fences the tenant from the
// token: it also pins the ENTITY TYPE, and it is what keeps the side-effect
// lanes (notify the proposer, bump the entry) from firing on a proposal that is
// not a kungal galgame edit. The refusal is shaped as a user-plane 404 so the
// callers map it through userEditError with everything else on this lane.
func (h *EditHandler) proposalForReview(ctx context.Context, token string, id int64) (*catalogclient.EditProposal, error) {
	prop, err := h.catalog.GetEditProposalUser(ctx, token, id)
	if err != nil {
		return nil, err
	}
	if prop.Site != catalogSite || prop.EntityType != entityTypeGame {
		return nil, &catalogclient.UserAPIError{Status: http.StatusNotFound, Message: "proposal outside the kungal tenant"}
	}
	return prop, nil
}

// reviewEntry gates the proposal VIEW surface (ProposalDetail): the workbench
// opens for moderators and for the entry's creator, everyone else 403s.
//
// This is a pure VIEW/UX gate, and the only local gate the edit chain has left.
// It decides which page a person can open, not what they may do — every write
// behind it goes out on their own token and is authorized by infra, which now
// derives ownership itself. Keeping the read narrow is a product choice (a
// half-usable workbench for a random visitor is noise, not transparency); the
// per-game proposal list right next to it stays fully public.
//
// Wave 180 moved the READ underneath it onto the token (so infra fences the
// proposal too) and left this check exactly as it was — neither tightened nor
// loosened. Two gates on a view surface are not a contradiction: infra answers
// "may this subject see this proposal", this one answers "does kungal offer
// them the workbench".
func (h *EditHandler) reviewEntry(c fiber.Ctx, token string, id int64) (*catalogclient.EditProposal, error) {
	ctx := c.Context()
	prop, err := h.proposalForReview(ctx, token, id)
	if err != nil {
		return nil, err
	}
	user := middleware.GetUser(c)
	if user == nil {
		return nil, &catalogclient.UserAPIError{Status: http.StatusForbidden, Message: "review entry denied"}
	}
	if !role.CanModerate(user.Roles) && !h.isGameOwner(ctx, prop.EntityID, int64(user.ID)) {
		return nil, &catalogclient.UserAPIError{Status: http.StatusForbidden, Message: "review entry denied"}
	}
	return prop, nil
}

// canDecide projects, for the caller, whether the workbench should offer the
// amend / merge / decline controls on THIS proposal.
//
// It is computed from the viewer's own capability projection against the
// proposal's own keys: every key it would land (the effective patch — what the
// amendments actually left — falling back to the original patch) must resolve to
// a field the viewer may review. An empty patch decides nothing, so it is false.
//
// Deriving it from the projection rather than from a role test is the point of
// wave 178: the forum no longer holds a second opinion about who may adjudicate,
// so the button can only ever appear when the write behind it would succeed.
func canDecide(prop *catalogclient.EditProposal, fields []catalogclient.EditSchemaField) bool {
	patch := prop.EffectivePatch
	if len(patch) == 0 {
		patch = prop.Patch
	}
	if len(patch) == 0 {
		return false
	}
	reviewable := make(map[string]bool, len(fields))
	for _, f := range fields {
		reviewable[f.Key] = f.CanReview
	}
	for key := range patch {
		if !reviewable[key] {
			return false
		}
	}
	return true
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
	workID, appErr := h.workIDOf(c.Context(), gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	items, err := h.catalog.ListEditProposals(c.Context(), catalogclient.EditProposalFilter{
		EntityType: entityTypeGame, EntityID: workID, Site: catalogSite,
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

// ProposalDetail — GET /galgame-edit/proposals/:id (auth; a VIEW gate admits
// moderators and the entry's creator). The review workbench read: proposal + amendments +
// effective patch, the entity's CURRENT values (per-field old→new compare),
// the reviewer's capability projection, and the galgame brief.
func (h *EditHandler) ProposalDetail(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	ctx := c.Context()
	prop, err := h.reviewEntry(c, token, id)
	if err != nil {
		return userEditError(c, err)
	}
	values, err := h.catalog.EditSnapshotUser(ctx, token, entityTypeGame, prop.EntityID)
	if err != nil {
		return userEditError(c, err)
	}
	// The projection is the viewer's own — the same one their amend / merge /
	// decline will be judged against, which is what makes can_decide below a
	// prediction rather than a guess.
	schema, err := h.catalog.GetEditSchemaUser(ctx, token, entityTypeGame, prop.EntityID)
	if err != nil {
		return userEditError(c, err)
	}
	enriched := h.enrich(ctx, []catalogclient.EditProposal{*prop})
	return response.OK(c, fiber.Map{
		"proposal":   enriched[0],
		"values":     values,
		"fields":     schema.Fields,
		"users":      h.userMap(ctx, collectProposalUIDs([]catalogclient.EditProposal{*prop})),
		"can_decide": canDecide(prop, schema.Fields),
	})
}

// editAmendRequest carries the maintainer delta: corrected values (set) and
// rejected fields (unset).
type editAmendRequest struct {
	Set   map[string]any `json:"set"`
	Unset []string       `json:"unset"`
	Note  string         `json:"note"`
}

// Amend — POST /galgame-edit/proposals/:id/amend (auth). The crown mechanism:
// correct a value / reject a field before merging; the merged revision carries
// proposer + amender double attribution. Who may amend is infra's answer,
// derived from the token's roles and its ownership of the target work.
func (h *EditHandler) Amend(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
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
	amendment, err := h.catalog.AmendEditProposalUser(c.Context(), token, id, req.Set, req.Unset, req.Note)
	if err != nil {
		return userEditError(c, err)
	}
	return response.OK(c, amendment)
}

type editDecisionRequest struct {
	Note string `json:"note"`
}

// Merge — POST /galgame-edit/proposals/:id/merge (auth). Authorization is
// infra's; the local pre-read exists for the side effects and the tenant pin.
func (h *EditHandler) Merge(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
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
	// The pre-read is not a gate — it is the side effects' input (who to notify,
	// which entry to bump) plus the tenant pin that keeps this BFF from
	// adjudicating a foreign site's proposal by id. Infra decides the merge.
	prop, err := h.proposalForReview(ctx, token, id)
	if err != nil {
		return userEditError(c, err)
	}
	rev, err := h.catalog.MergeEditProposalUser(ctx, token, id, req.Note)
	if err != nil {
		return userEditError(c, err)
	}
	h.mergeSideEffects(ctx, prop, int64(user.ID), rev)
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
	entry := h.entryOf(ctx, prop.EntityID)
	if h.repo != nil && entry.GID > 0 {
		if err := h.repo.Touch(h.repo.DB().WithContext(ctx), entry.GID); err != nil {
			slog.Warn("galgame edit: resource_update_time bump failed", "gid", entry.GID, "error", err)
		}
	}
	content := entry.Name
	if rev != nil && rev.AmenderUID != nil {
		content = strings.TrimSpace(content + "（审核时有修正）")
	}
	h.notifyDecision(prop, entry.GID, mergerID, msgService.NotifyMerged, content)
}

// Decline — POST /galgame-edit/proposals/:id/decline (auth). The reason is
// required — a silent decline was the
// old galgame's worst reviewer habit; it travels to the proposer in full on
// the decline notice (E3b ruling 1).
func (h *EditHandler) Decline(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
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
	// Same as Merge: the pre-read feeds the notice (the proposer to address, the
	// entry to name) and pins the tenant. Authorization is infra's.
	target, err := h.proposalForReview(ctx, token, id)
	if err != nil {
		return userEditError(c, err)
	}
	prop, err := h.catalog.DeclineEditProposalUser(ctx, token, id, req.Note)
	if err != nil {
		return userEditError(c, err)
	}
	entry := h.entryOf(ctx, target.EntityID)
	content := req.Note
	if entry.Name != "" {
		content = entry.Name + "：" + req.Note
	}
	h.notifyDecision(target, entry.GID, int64(user.ID), msgService.NotifyDeclined, content)
	return response.OK(c, prop)
}

// Withdraw — POST /galgame-edit/proposals/:id/withdraw (auth). Closing your own
// proposal is the purest user lane there is, so it rides the user token whole
// (wave 177): the engine reads both the proposer and the tenant off the token
// and answers a foreign id itself.
//
// That is why the S2S tenant pre-flight this used to run is gone rather than
// merely redundant. It existed because the ASSERTED-actor face fences neither
// proposer nor tenant, so the BFF had to. With the token carrying both, a
// pre-flight would only re-ask a question the write already answers — and it
// would answer it on a channel the caller is no longer speaking.
func (h *EditHandler) Withdraw(c fiber.Ctx) error {
	id, appErr := parseProposalID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	// The route is authed; keep the local requirement explicit so a withdraw can
	// never be attempted with an anonymous (therefore empty) token.
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	prop, err := h.catalog.WithdrawEditProposalUser(c.Context(), token, id)
	if err != nil {
		return userEditError(c, err)
	}
	return response.OK(c, prop)
}
