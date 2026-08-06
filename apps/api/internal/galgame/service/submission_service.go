package service

import (
	"context"
	stderrors "errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
)

// SubmissionService owns the user-driven submission lifecycle: file a new
// entry, publish an existing unpublished one, withdraw, resubmit, and list your
// own submissions.
//
// Every one of those is now a SEMANTIC ACTION on the registry's claim, not a
// write to a wiki row. The shape of the change is worth stating once, because
// it is not a rename:
//
//   - The wiki's `status` integer is gone. Lifecycle is claim_state, and it
//     moves only through the eight actions — there is no field to patch, which
//     is what let the wiki's status column become a vocabulary nobody could
//     reason about.
//   - "Delete my draft" has no counterpart and deliberately gets none. A
//     registry row is an IDENTITY; it does not stop existing because a product
//     withdrew its claim on it. 撤稿 is `withdraw` (back to draft), and the
//     entry can be resubmitted rather than re-typed.
//   - "My submissions" is answered by the registry's per-user face, which reads
//     the claim-event log. A user's submissions are precisely the works whose
//     lifecycle they moved, so the list needs no owner column — and could not
//     have one, since a registry row outlives any account.
//
// Reward policy is unchanged, and each route pays from exactly one place:
//
//	submit   → 0 (deferred to approval, to deter spam)
//	claim    → +3 here, in the request path, under the original key
//	approved → +3 from the claim-event cron (never both: the cron pays only the
//	           pending → live route, this pays only the draft → live one)
type SubmissionService struct {
	galgameClient *client.GalgameClient
	catalog       *catalogclient.Client
	galgameRepo   *repository.GalgameRepository
}

func NewSubmissionService(
	galgameClient *client.GalgameClient,
	catalog *catalogclient.Client,
	galgameRepo *repository.GalgameRepository,
) *SubmissionService {
	return &SubmissionService{
		galgameClient: galgameClient,
		catalog:       catalog,
		galgameRepo:   galgameRepo,
	}
}

// submissionSite is the tenant kungal files its claims under. It must equal the
// forum OAuth client's catalog_site binding.
//
// Since wave 179 no WRITE sends it: the acting tenant is derived from the
// user's token, the same binding read from the other end. What still names it
// explicitly are the reads that have no token behind them — the moderation
// queue (whose whole job is to show kungal's backlog and nobody else's) and the
// third-person profile stats.
const submissionSite = client.ClaimSiteKungal

// SubmitResult is what the wizard needs after filing: the id the entry will
// live at on kungal, plus the registry identity behind it.
type SubmitResult struct {
	GID        int    `json:"gid"`
	WorkID     int64  `json:"work_id"`
	ClaimState string `json:"claim_state"`
}

// Submit files a brand-new entry.
//
// THE ID COMES BACK, IT IS NOT SENT. kungal names no id: the registry mints the
// work and the claim adopts that work's own primary key as the product id, which
// the response returns and which is the gid from here on. One allocator for one
// key space — the alternative, a local sequence advancing alongside the
// registry's, is only correct while somebody keeps reseeding it, and its failure
// mode is a silent collision that surfaces as "you already submitted this".
//
// Nothing local is written here either way. The browse list is `FROM galgame`,
// so a row IS an entry in the catalogue and a submission awaiting review is not
// one; the stub is created later by the claim-event cron at the moment the claim
// goes live. That is the invariant the wiki flow had ("a pending submission gets
// no stub"), kept without a second copy of the lifecycle living locally.
//
// accessToken is the submitter's own OAuth token and it now carries BOTH halves
// of the operation. Until wave 179 the mint was an asserted-actor S2S call
// sitting next to a Bearer banner edit — two planes for one user gesture, and
// the only one of them that let kungal name a submitter who was not the one
// logged in. The claim face derives the actor and the tenant from the token
// like the edit face already did, so the split is gone and the site constant no
// longer rides along.
func (s *SubmissionService) Submit(
	ctx context.Context,
	accessToken string,
	form *SubmissionForm,
) (*SubmitResult, *errors.AppError) {
	if form.DisplayName() == "" {
		return nil, errors.ErrValidation("请至少填写一个语言的标题")
	}
	released, appErr := form.Released()
	if appErr != nil {
		return nil, appErr
	}
	res, err := s.catalog.SubmitWorkUser(ctx, accessToken, catalogclient.UserWorkSubmitRequest{
		Fields: form.Fields(), Released: released,
	})
	if err != nil {
		return nil, claimActionError(err)
	}
	gid := int(res.ProductWorkID)
	// The banner is not a submittable facet — a cover is a REFERENCE to bytes
	// that must already exist, so it rides as the submission's first edit,
	// visible to the reviewer alongside it. Best-effort: a failure here costs
	// the cover, never the submission.
	if patch := form.CoverPatch(); patch != nil {
		if _, err := s.catalog.CreateEditProposalUser(ctx, accessToken, catalogclient.UserEditCreateRequest{
			EntityType: catalogclient.EntityTypeWork, EntityID: res.WorkID,
			Patch: patch, Note: "投稿时提交的横幅图",
		}); err != nil {
			slog.Warn("submit: 附加横幅图失败", "work", res.WorkID, "error", err)
		}
	}
	return &SubmitResult{GID: gid, WorkID: res.WorkID, ClaimState: res.ClaimState}, nil
}

// Claim publishes an entry the registry already holds as an unpublished draft
// — the wizard's "this game is already listed, I'll finish it" path.
//
// It needs no minting: the work already carries kungal's product id, so the
// whole operation is one state move. The two local side effects are unchanged,
// including the moemoepoint key, which is stable per (galgame, claimer) so a
// retried claim cannot double-award.
//
// uid is the LOCAL award's payee and nothing else — the registry learns who
// acted from the token. It is passed rather than derived because moemoepoint is
// a forum ledger keyed on a forum user id; reading it as an authority claim is
// the mistake wave 179 removed everywhere it mattered.
func (s *SubmissionService) Claim(
	ctx context.Context,
	accessToken string,
	uid int64,
	gid int,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	res, appErr := s.act(ctx, accessToken, gid, catalogclient.ClaimActionPublish, "")
	if appErr != nil {
		return nil, appErr
	}
	// Publishing is a content update: ensure the stub exists AND bump
	// resource_update_time, so the entry both appears in the browse list and
	// rises to the top of the "recently updated" view.
	if err := s.galgameRepo.Touch(s.galgameRepo.DB().WithContext(ctx), gid); err != nil {
		slog.Warn("claim: 刷新本地 galgame resource_update_time 失败", "gid", gid, "error", err)
	}
	moemoepoint.Award(int(uid), constants.RewardCreateGalgame,
		moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame", gid),
		moemoepoint.Key("claim", strconv.Itoa(gid), strconv.FormatInt(uid, 10)))
	return res, nil
}

// Resubmit sends a draft (or a declined submission) back to the review queue.
// It replaces the STATE half of the old "PATCH my draft"; the CONTENT half is
// an ordinary edit now, filed through the editing face like every other change
// to the same entry — one write path for a field, whoever is changing it.
func (s *SubmissionService) Resubmit(
	ctx context.Context,
	accessToken string,
	gid int,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	return s.act(ctx, accessToken, gid, catalogclient.ClaimActionSubmit, "")
}

// Withdraw pulls a submission back to draft.
//
// This is where "删除我的投稿" went. The registry row survives — it is an
// identity that anchors, edit history and other products point at — so the
// product's claim retreats to draft instead of the row being destroyed. The
// entry stops being publicly listed, which is what the user asked for, and can
// be resubmitted without re-typing it.
func (s *SubmissionService) Withdraw(
	ctx context.Context,
	accessToken string,
	gid int,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	return s.act(ctx, accessToken, gid, catalogclient.ClaimActionWithdraw, "")
}

// act resolves a gid to its registry work and performs one owner action as the
// token's subject.
//
// There is no local ownership pre-flight and deliberately none: since wave 179
// the registry checks that the token's uid IS the work's owner, so "publish
// somebody else's draft" is refused at the only place that can actually know.
// A forum-side mirror of that rule would be a second answer waiting to drift.
func (s *SubmissionService) act(
	ctx context.Context,
	accessToken string,
	gid int,
	action string,
	reason string,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	workID, appErr := s.workIDOf(ctx, gid)
	if appErr != nil {
		return nil, appErr
	}
	res, err := s.catalog.ActOnClaimUser(ctx, accessToken, workID, action, catalogclient.UserClaimActionRequest{
		Reason: reason,
	})
	if err != nil {
		return nil, claimActionError(err)
	}
	return res, nil
}

func (s *SubmissionService) workIDOf(ctx context.Context, gid int) (int64, *errors.AppError) {
	ids, appErr := s.galgameClient.CatalogWorkIDs(ctx, []int{gid})
	if appErr != nil {
		return 0, appErr
	}
	workID, ok := ids[gid]
	if !ok {
		return 0, errors.ErrNotFound("条目不存在")
	}
	return workID, nil
}

// claimActionError maps the lifecycle face's refusals onto the house envelope.
//
// It reads the USER plane's taxonomy since wave 179 (*UserAPIError plus the two
// sentinels only a forwarded token can produce). The response SHAPES are
// unchanged — which plane carried the write is an implementation detail no user
// chose — with one addition that could not exist before:
//
//   - ErrInsufficientScope is a re-login prompt (code 235), never a 403. The
//     user's grant predates `catalog:edit` and no refresh can widen it, so
//     telling them "你没有权限" would be a lie about a permission they hold.
//   - ErrUnauthorized is a dead session, not an upstream fault.
//
// A 409 still keeps its status AND its message: an illegal transition names the
// state the claim is actually in, which is the point of semantic actions — the
// losing side of a race has already been told what happened and should
// re-render rather than retry.
func claimActionError(err error) *errors.AppError {
	var apiErr *catalogclient.UserAPIError
	switch {
	case stderrors.Is(err, catalogclient.ErrInsufficientScope):
		return errors.ErrReauthRequired("投稿需要新的授权，请退出登录后重新登录以授予该权限")
	case stderrors.Is(err, catalogclient.ErrUnauthorized):
		return errors.ErrAuthExpired()
	case stderrors.Is(err, catalogclient.ErrNotFound):
		return errors.ErrNotFound("条目不存在")
	case stderrors.Is(err, catalogclient.ErrNotConfigured):
		return errors.New(errors.CodeBiz, "资料库服务暂不可用", http.StatusServiceUnavailable)
	case stderrors.As(err, &apiErr):
		switch apiErr.Status {
		case http.StatusForbidden:
			return errors.ErrForbidden("你没有权限执行此操作")
		case http.StatusUnprocessableEntity:
			return errors.ErrValidation(apiErr.Message)
		case http.StatusConflict:
			return errors.New(errors.CodeBiz, apiErr.Message, http.StatusConflict)
		}
		slog.Error("claim action: 上游错误", "status", apiErr.Status, "code", apiErr.Code, "msg", apiErr.Message)
	default:
		slog.Warn("claim action: catalog 不可达", "error", err)
	}
	return errors.New(errors.CodeBiz, "资料库服务暂不可用", http.StatusServiceUnavailable)
}

// ─── my submissions ──────────────────────────────────────────────────────

// mineStates is the default filter of the 我的提交 page: everything that is not
// yet a published entry. A live claim leaves this list because it has become a
// public entry with a page of its own — the same rule the wiki list followed
// when it filtered to status 3,4.
var mineStates = []string{
	catalogclient.ClaimStatePending,
	catalogclient.ClaimStateDeclined,
	catalogclient.ClaimStateDraft,
}

// ListMine returns the caller's own submissions, newest activity first.
//
// The summary on each row is the work's LATEST transition BY ANYONE, which is
// deliberate: what a submitter needs to see on their own submission is the
// reviewer's verdict and note — an event they did not cause. That is why the
// wiki's separate "my notifications" query has no successor here.
//
// "Mine" is the TOKEN's since wave 179: there is no uid to pass and therefore
// no way for a caller to mean somebody else by mistake, which is the class of
// bug an "own data" endpoint taking an id argument invites.
func (s *SubmissionService) ListMine(
	ctx context.Context,
	accessToken string,
	query url.Values,
) (*catalogclient.UserClaimPage, *errors.AppError) {
	states := mineStates
	if raw := query.Get("claim_state"); raw != "" {
		states = splitCSV(raw)
	}
	page, err := s.catalog.MyClaims(ctx, accessToken, catalogclient.UserClaimFilter{
		ClaimStates: states,
		Before:      int64(atoiOr(query.Get("before"), 0)),
		Limit:       atoiOr(query.Get("limit"), 20),
	})
	if err != nil {
		return nil, claimActionError(err)
	}
	if page.Items == nil {
		page.Items = []catalogclient.UserClaimItem{}
	}
	return page, nil
}

// CountMine is gone (wave 179). It was ListMine reduced to its total, and it
// had no caller: the profile statistic it described is served by
// GalgameUserStatsService, which reads the THIRD-PERSON face because a profile
// page is somebody else's. Migrating it would have meant an accessToken
// parameter nobody could supply.

// ─── the publish wizard search ───────────────────────────────────────────

// wizardSearchInclude is the include= vocabulary a wizard row renders: the four
// localized titles, the cover art, and the identity anchors (the row prints the
// VNDB id).
const wizardSearchInclude = "names,covers,refs"

// wizardDefaultLimit mirrors the wizard's own page size; the FE always sends
// one, this is only the floor for a hand-made request.
const wizardDefaultLimit = 12

// WizardSearchPage is the 发布向导 payload.
//
// Both halves are the registry's now. Items answers "does this game already
// exist" — a miss there is a duplicate submission, the failure the wizard
// exists to prevent — and Pending is the caller's own backlog, read from the
// per-user claim face.
type WizardSearchPage struct {
	Items   []client.GalgameBrief         `json:"items"`
	Pending []catalogclient.UserClaimItem `json:"pending"`
	Total   int64                         `json:"total"`
}

// SearchWithPending serves GET /galgame/search/wizard.
func (s *SubmissionService) SearchWithPending(
	ctx context.Context,
	accessToken string,
	query url.Values,
) (*WizardSearchPage, *errors.AppError) {
	items, total, appErr := s.wizardItems(ctx, query)
	if appErr != nil {
		return nil, appErr
	}
	pending, appErr := s.wizardPending(ctx, accessToken)
	if appErr != nil {
		return nil, appErr
	}
	return &WizardSearchPage{Items: items, Pending: pending, Total: total}, nil
}

// wizardItems runs the catalog search lane.
//
// The claim-state list is `live,draft,pending`, and `pending` is the addition.
// It closes the gap 160 recorded: the wiki projector folded "unclaimed VNDB
// draft" and "someone else's submission awaiting review" onto the same `draft`
// word, so the wizard could SHOW the second kind but had no way to say what it
// was. Asking for the state before the projector starts producing it is
// deliberate — today the term matches nothing, so this ships on its own and the
// projector fix follows without a coordinated deploy.
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
		"claim_state": {client.ClaimStateWizard},
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

// wizardPending is the caller's own backlog, off the per-user claim face — the
// terminal source. The wiki lane it replaces answered the same question by
// merging the caller's own rows into a search response, which is why it could
// only ever be asked as part of a search.
func (s *SubmissionService) wizardPending(
	ctx context.Context,
	accessToken string,
) ([]catalogclient.UserClaimItem, *errors.AppError) {
	page, err := s.catalog.MyClaims(ctx, accessToken, catalogclient.UserClaimFilter{
		ClaimStates: []string{catalogclient.ClaimStatePending, catalogclient.ClaimStateDeclined},
		Limit:       wizardDefaultLimit,
	})
	if err != nil {
		return nil, claimActionError(err)
	}
	if page.Items == nil {
		return []catalogclient.UserClaimItem{}, nil
	}
	return page.Items, nil
}
