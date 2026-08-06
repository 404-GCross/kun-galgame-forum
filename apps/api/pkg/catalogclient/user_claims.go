// The claim-lifecycle face on the USER-TOKEN plane (wave 179).
//
// Wave 178 moved every human lane of the editing engine onto the user's own
// Bearer token; this finishes the same move for the SUBMISSION lifecycle, which
// was the last human write still speaking as the forum about the user. The
// three faces below are the whole surface a human touches: mint a submission,
// move a claim, read your own claims.
//
// What actually changes is where authority lives, not the shapes:
//
//   - The owner half (claim / submit / publish / withdraw) used to be gated by
//     the ASSERTED actor plus kungal's site binding. The catalog now knows who
//     owns a work (catalog_work.owner_user_id) and reads the acting tenant off
//     the token, so ownership is checked against the token subject and a
//     mismatch is a 403 the forum cannot talk its way around.
//   - The review half (approve / decline / ban / unban) used to be authorized
//     against the roles kungal asserted. It is now `catalog.claim.review` over
//     the TOKEN's roles — hot-fed by the permission console, so a grant takes
//     effect without a forum deploy and the forum's own RequireModerator gate
//     stops being a second answer that can drift from the first.
//   - "My claims" needs no uid: /claims/mine is whoever holds the token. That
//     removes the one call shape where a caller could have asked about somebody
//     else by getting an argument wrong.
//
// claims.go's S2S face survives only for what makes no claim on the user's
// behalf: the transition feed the cron reads, and UserClaims(uid) — which reads
// OTHER people's public contribution counts, where there is no token to hold.
//
// Scope: same as every other user-plane lane, `catalog:edit`. A session minted
// before the scope existed cannot be widened by a refresh, so that 403 arrives
// as ErrInsufficientScope ("log out and back in"), never a generic denial.
package catalogclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// userClaimBase is the user plane's claims prefix. Same operations as the S2S
// claim face, different authority.
const userClaimBase = userBase

// UserWorkSubmitRequest mints a registry work AS the token's subject. It is
// WorkSubmitRequest minus the two fields the token already answers — the actor
// and the site — and nothing else moved: Fields is still keyed by the editing
// face's catalog.work field keys, and a work-level release date still travels
// as Released rather than as a field.
//
// ProductWorkID stays omittable and kungal still omits it: the registry mints
// the identity and the claim adopts that work's own key, which the response
// returns. One allocator for one key space.
type UserWorkSubmitRequest struct {
	ProductWorkID int64           `json:"product_work_id,omitempty"`
	Fields        map[string]any  `json:"fields"`
	Released      *WorkSubmitDate `json:"released,omitempty"`
}

// SubmitWorkUser files a new submission as the user. Idempotency is unchanged
// — the claim's own unique key (medium, site, product_work_id) — so a retried
// wizard still gets a 409 naming the work it already created rather than a
// second identity.
func (c *Client) SubmitWorkUser(ctx context.Context, accessToken string, req UserWorkSubmitRequest) (*WorkSubmitResult, error) {
	return userEditPost[WorkSubmitResult](ctx, c, accessToken, userClaimBase+"/works/submit", req)
}

// UserClaimActionRequest is the body of one lifecycle action on the user plane.
// ClaimActionRequest's Site and Actor are both gone: the acting tenant and the
// acting subject are the token's, and a review action was always cross-tenant
// anyway.
type UserClaimActionRequest struct {
	// ProductWorkID is read by `claim` alone and ignored by every other action.
	ProductWorkID int64 `json:"product_work_id,omitempty"`
	// Reason is recorded on the event and REQUIRED by decline: a refusal the
	// submitter cannot act on is the failure mode the field exists to prevent.
	Reason string `json:"reason,omitempty"`
}

// ActOnClaimUser performs one lifecycle action as the token's subject.
//
// A 409 still carries the claim's CURRENT state, which is the whole reason
// these are semantic actions instead of a claim_state patch — the losing side
// of a race has already been told what happened and should re-render, not
// retry. A 403 now means one of two real things the forum no longer decides:
// the subject does not own the work (owner half), or their token's roles do not
// carry catalog.claim.review (review half).
func (c *Client) ActOnClaimUser(ctx context.Context, accessToken string, workID int64, action string, req UserClaimActionRequest) (*ClaimActionResult, error) {
	return userEditPost[ClaimActionResult](ctx, c, accessToken,
		userClaimBase+"/works/"+strconv.FormatInt(workID, 10)+"/claim-actions/"+action, req)
}

// MyClaims lists the works the TOKEN's subject has acted on, newest activity
// first — the same descending cursor page as UserClaims, minus the uid.
//
// UserClaimFilter.Site is deliberately NOT sent: the acting tenant comes off
// the token like everything else on this plane, so a site in the query would be
// either redundant or a claim about another tenant.
func (c *Client) MyClaims(ctx context.Context, accessToken string, f UserClaimFilter) (*UserClaimPage, error) {
	q := url.Values{}
	if len(f.ClaimStates) > 0 {
		q.Set("claim_state", joinStates(f.ClaimStates))
	}
	if f.Before > 0 {
		q.Set("before", strconv.FormatInt(f.Before, 10))
	}
	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}
	path := userClaimBase + "/claims/mine"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	return userEditDo[UserClaimPage](ctx, c, http.MethodGet, accessToken, path, nil)
}
