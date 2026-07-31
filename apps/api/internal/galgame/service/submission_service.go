package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"strconv"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/errors"
)

// SubmissionService handles the user-driven submission lifecycle for new
// galgames per docs/galgame_wiki/07-submission.md.
//
// Reward policy (decision 1 in the kungal/wiki integration plan):
//
//	submit → 0 (no reward; deferred until approval to deter spam)
//	claim  → +3 (claim publishes immediately, no review queue)
//	approved (via cron) → +3 (handled in GalgameMessageSync, not here)
//	declined / delete → no reward to reclaim (never granted)
type SubmissionService struct {
	galgameClient *client.GalgameClient
	galgameRepo   *repository.GalgameRepository
}

func NewSubmissionService(
	galgameClient *client.GalgameClient,
	galgameRepo *repository.GalgameRepository,
) *SubmissionService {
	return &SubmissionService{
		galgameClient: galgameClient,
		galgameRepo:   galgameRepo,
	}
}

// Submit forwards a pending submission to galgame. NO local side effects:
//   - no stub (status=3 isn't publicly listable, so no need to seed the
//     local `galgame` index; if the user self-interacts later the
//     interaction path lazy-creates the stub)
//   - no moemoepoint (deferred to approval via cron)
//
// Galgame enforces the daily quota (20009) and vndb_id format/uniqueness.
func (s *SubmissionService) Submit(
	ctx context.Context,
	token string,
	body []byte,
	contentType string,
) (json.RawMessage, *errors.AppError) {
	return s.galgameClient.SubmitDraft(ctx, token, body, contentType)
}

// Claim flips a VNDB-source draft (status=2) directly to published
// (status=0) on the galgame, then runs two best-effort local side effects:
//   - creates the local stub so the galgame appears in kungal's list query;
//   - awards the claimer +3 moemoepoint via OAuth.
//
// Idempotency does NOT rely on a local lock (there is none): re-claiming is
// blocked galgame-side (it only accepts status=2), and the OAuth award uses a
// STABLE key per (galgame, claimer) so even a retried claim can't double-award.
//
// Galgame failure → no local side effect runs (correct).
// Local failure after galgame success → log; the +3 is forfeited but the
// publish itself stands. Avoiding a half-rollback of galgame state.
func (s *SubmissionService) Claim(
	ctx context.Context,
	userID int,
	token string,
	gid int,
) (json.RawMessage, *errors.AppError) {
	data, appErr := s.galgameClient.ClaimDraft(ctx, token, gid)
	if appErr != nil {
		return nil, appErr
	}

	// Claiming is a content update: ensure the local stub exists AND bump
	// galgame.resource_update_time, so the claimed galgame both appears in
	// kungal's list query and rises to the top of the "sort by update time" view
	// (Touch does both — replaces the old stub-only create which didn't bump).
	if err := s.galgameRepo.Touch(s.galgameRepo.DB(), gid); err != nil {
		slog.Warn("claim: 刷新本地 galgame resource_update_time 失败", "gid", gid, "error", err)
	}
	// Award +3 via OAuth (no local +=). Stable key per (galgame, claimer) so a
	// re-claim can't double-award.
	moemoepoint.Award(userID, constants.RewardCreateGalgame,
		moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame", gid),
		moemoepoint.Key("claim", strconv.Itoa(gid), strconv.Itoa(userID)))
	return data, nil
}

// PatchDraft forwards an edit to the user's own pending/declined draft.
// No local side effects — galgame holds all metadata and the row remains
// invisible to other users until approval.
func (s *SubmissionService) PatchDraft(
	ctx context.Context,
	token string,
	gid int,
	body []byte,
	contentType string,
) (json.RawMessage, *errors.AppError) {
	return s.galgameClient.PatchDraft(ctx, token, gid, body, contentType)
}

// DeleteDraft hard-deletes the user's own pending/declined draft on the
// galgame, then defensively removes any kungal stub the user may have
// lazy-created by self-interacting with the pending row. Idempotent.
//
// Galgame failure → don't touch local state.
func (s *SubmissionService) DeleteDraft(
	ctx context.Context,
	token string,
	gid int,
) *errors.AppError {
	if appErr := s.galgameClient.DeleteDraft(ctx, token, gid); appErr != nil {
		return appErr
	}
	s.galgameRepo.DeleteLocalStub(gid)
	return nil
}

// ListMine proxies GET /galgame/mine — the user's own submissions across
// all statuses. Thin pass-through; identity comes from the forwarded Bearer.
func (s *SubmissionService) ListMine(
	ctx context.Context,
	token string,
	query url.Values,
) (json.RawMessage, *errors.AppError) {
	return s.galgameClient.GetWithToken(ctx, "/galgame/mine", token, query)
}

// ─── the publish wizard search ───────────────────────────────────────────

// wizardSearchInclude is the include= vocabulary a wizard row renders: the four
// localized titles, the cover art, and the identity anchors (the row prints the
// VNDB id).
const wizardSearchInclude = "names,covers,refs"

// wizardDefaultLimit mirrors the wizard's own page size; the FE always sends
// one, this is only the floor for a hand-made request.
const wizardDefaultLimit = 12

// WizardSearchPage is the 发布向导 payload. Its two halves are answered by two
// different faces, and that split is deliberate:
//
//   - Items is the CATALOG works search (claimed=true, claim_state=live,draft).
//     The registry is the supply of record for "does this game already exist",
//     which is the wizard's entire job — a miss here is a duplicate submission.
//   - Pending is the caller's OWN status 3/4 submissions. The catalog has no
//     per-user read face for that backlog (doc 156 P3: claim events only exist
//     for post-N2 actions, and the whole population here predates them), so
//     this half keeps querying the wiki face and is forwarded verbatim.
type WizardSearchPage struct {
	Items   []client.GalgameBrief `json:"items"`
	Pending json.RawMessage       `json:"pending"`
	Total   int64                 `json:"total"`
}

// SearchWithPending serves GET /galgame/search/wizard.
func (s *SubmissionService) SearchWithPending(
	ctx context.Context,
	token string,
	query url.Values,
) (*WizardSearchPage, *errors.AppError) {
	items, total, appErr := s.wizardItems(ctx, query)
	if appErr != nil {
		return nil, appErr
	}
	pending, appErr := s.wizardPending(ctx, token, query)
	if appErr != nil {
		return nil, appErr
	}
	return &WizardSearchPage{Items: items, Pending: pending, Total: total}, nil
}

// wizardItems runs the catalog search lane.
//
// `claim_state=live,draft` is the registry spelling of the wiki's old
// `status=0,2` filter, with one KNOWN and accepted widening: the catalog
// projects wiki status 2 (unclaimed VNDB draft) and status 3 (someone else's
// submission awaiting review) onto the same `draft` state, so other people's
// pending submissions now surface here too. That is the right side of the trade
// — an entry the wizard cannot see is an entry that gets submitted twice — and
// the two are physically indistinguishable on the catalog wire, so the row
// cannot be labelled any more precisely than "草稿". Claiming one of them still
// goes to the wiki, which answers with the correct refusal.
//
// Only the AGE gate is opened, exactly as the wiki lane had it: the wizard is a
// dedup tool for an authenticated submitter, and filtering its supply by the
// reader's editorial preference would hide the very entries it exists to
// surface.
func (s *SubmissionService) wizardItems(
	ctx context.Context,
	query url.Values,
) ([]client.GalgameBrief, int64, *errors.AppError) {
	q := url.Values{
		"q":           {query.Get("q")},
		"page":        {strconv.Itoa(atoiOr(query.Get("page"), 1))},
		"limit":       {strconv.Itoa(atoiOr(query.Get("limit"), wizardDefaultLimit))},
		"claimed":     {"true"},
		"claim_state": {client.ClaimStateLiveOrDraft},
		"include":     {wizardSearchInclude},
	}
	client.OpenPopulation(q)

	res, appErr := s.galgameClient.CatalogWorksSearch(ctx, q)
	if appErr != nil {
		return nil, 0, appErr
	}
	items := make([]client.GalgameBrief, 0, len(res.Items))
	for i := range res.Items {
		row := &res.Items[i]
		// A withdrawn claim must never be offered for 认领, and a row with no
		// gid has no wizard action at all (every branch of the card links or
		// posts by gid). `claimed=true` should already exclude the latter.
		if !client.CatalogItemRenderable(row) || client.CatalogItemGID(row) == 0 {
			continue
		}
		b := client.CatalogItemToBrief(row)
		// The card reads `banner`; on the catalog wire the art arrives as the
		// derived effective banner, which IS the same image the wiki lane put
		// in that field.
		b.Banner = b.EffectiveBannerURL
		items = append(items, b)
	}
	return items, res.Total, nil
}

// wizardPending forwards the caller's own pending/declined submissions off the
// wiki face. The query is the pre-switchover one byte for byte — include_pending
// plus status=0,2 — because the face merges the caller's hits only when it is
// serving a real search; we then keep just that half of the envelope.
func (s *SubmissionService) wizardPending(
	ctx context.Context,
	token string,
	query url.Values,
) (json.RawMessage, *errors.AppError) {
	q := make(url.Values, len(query)+2)
	for k, v := range query {
		q[k] = v
	}
	q.Set("include_pending", "true")
	q.Set("status", "0,2")

	data, appErr := s.galgameClient.GetWithToken(ctx, "/galgame/search", token, q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed struct {
		Pending json.RawMessage `json:"pending"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析投稿搜索响应失败")
	}
	if len(parsed.Pending) == 0 {
		return json.RawMessage("[]"), nil
	}
	return parsed.Pending, nil
}
