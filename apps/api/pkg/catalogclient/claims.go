package catalogclient

import (
	"context"
	"net/url"
	"strconv"
	"time"
)

// The claim-lifecycle face (infra wave 155 W2/W3 + 157): the eight semantic
// actions, the transition feed downstream inboxes are built from, and the
// per-user claim list.
//
// This is the registry-native replacement for the wiki's submission surface.
// Two vocabularies retire with it and are NOT re-declared here:
//
//   - the wiki `status` integers (0 published / 1 banned / 2 VNDB draft /
//     3 pending / 4 declined) become the five claim states below, which is a
//     re-shape and not a rename: status 2 and status 3 both projected onto
//     `draft`, and `hidden` is a state the wiki expressed as a deleted row;
//   - the wiki message `type` words (approved / declined / banned / unbanned)
//     become (from_state, to_state) PAIRS. A consumer that needs "was this an
//     approval" asks the transition, not a label — which is what lets the same
//     feed carry transitions the wiki had no word for.
//
// Every call rides the same Basic-authed S2S channel and envelope transport as
// the edit face (editDo), so a 409 keeps its body: an illegal transition
// answers with the claim's CURRENT state, which is the whole reason the action
// endpoints exist instead of a claim_state patch.
//
// Wave 179 emptied this file of WRITES. Every human lifecycle move — the mint
// and all eight actions — now speaks as the user over Bearer (user_claims.go),
// so ActOnClaim / SubmitWork and their asserted-actor request types were
// deleted rather than left as a second way in. What stays is exactly what has
// no user behind it: the transition feed the claim-event cron reads, and
// UserClaims(uid), which answers "what has THAT person published" for a profile
// page nobody holds a token for. The shared shapes (states, actions, results,
// the per-user page) are declared here and reused by both planes.

// Claim states — the public claim vocabulary (`claimed_by.state`). Eternal
// wire values.
const (
	ClaimStateNone     = "none"
	ClaimStateLive     = "live"
	ClaimStateDraft    = "draft"
	ClaimStatePending  = "pending"
	ClaimStateDeclined = "declined"
	ClaimStateHidden   = "hidden"
)

// Claim actions — the eight semantic moves. The first four are the owning
// site's, the last four a reviewer's (infra requires catalog.claim.review,
// which wave 157 granted moderator and up so the wiki's moderation staffing
// carries over unchanged).
const (
	ClaimActionClaim    = "claim"
	ClaimActionSubmit   = "submit"
	ClaimActionPublish  = "publish"
	ClaimActionWithdraw = "withdraw"
	ClaimActionApprove  = "approve"
	ClaimActionDecline  = "decline"
	ClaimActionBan      = "ban"
	ClaimActionUnban    = "unban"
)

// ClaimActionResult is what the transition did.
type ClaimActionResult struct {
	WorkID  int64   `json:"work_id"`
	From    *string `json:"from_state"`
	To      string  `json:"to_state"`
	EventID int64   `json:"event_id"`
}

// WorkSubmitDate is the fuzzy submitted date. The nullable tail IS the
// precision — {Y:2019} means "sometime in 2019" — so there is no separate
// precision enum and an omitted date means TBA.
type WorkSubmitDate struct {
	Y int16 `json:"y"`
	M int16 `json:"m,omitempty"`
	D int16 `json:"d,omitempty"`
}

// WorkSubmitResult is the minted identity plus its birth event.
type WorkSubmitResult struct {
	WorkID int64 `json:"work_id"`
	// ProductWorkID is the id the claim ended up anchored at. When the request
	// omitted one it is the registry-issued identity — and therefore the gid
	// kungal must use from here on. Always read this rather than assuming it
	// equals WorkID: it is the field that stays correct either way.
	ProductWorkID int64  `json:"product_work_id"`
	ClaimState    string `json:"claim_state"`
	EventID       int64  `json:"event_id"`
	ReleaseID     int64  `json:"release_id,omitempty"`
}

// ClaimEventFeedItem is one claim transition.
//
// FromState is null exactly once per claim — the transition that created it —
// so a consumer can recognise a birth without a second read. ProductWorkID is
// the claim's CURRENT product-side id (a snapshot taken when the page was
// served, not the value at event time); for a kungal claim that is the gid.
type ClaimEventFeedItem struct {
	ID            int64     `json:"id"`
	WorkID        int64     `json:"work_id"`
	FromState     *string   `json:"from_state"`
	ToState       string    `json:"to_state"`
	ActorUID      int64     `json:"actor_uid"`
	Reason        *string   `json:"reason"`
	Site          string    `json:"site"`
	ProductWorkID *int64    `json:"product_work_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// ClaimEventFeedPage is one page of the transition feed. NextSince echoes the
// request's cursor on an empty page, so a consumer that stores it
// unconditionally never rewinds.
type ClaimEventFeedPage struct {
	Items     []ClaimEventFeedItem `json:"items"`
	NextSince int64                `json:"next_since"`
}

// ClaimEventsSince reads one page of the transition feed after the exclusive
// cursor `since`, optionally narrowed to one tenant. Ascending by id, no
// has_more flag: a page shorter than the limit is the tail.
func (c *Client) ClaimEventsSince(ctx context.Context, since int64, limit int, site string) (*ClaimEventFeedPage, error) {
	q := url.Values{
		"since": {strconv.FormatInt(since, 10)},
		"limit": {strconv.Itoa(limit)},
	}
	if site != "" {
		q.Set("site", site)
	}
	return editGetQuery[ClaimEventFeedPage](ctx, c, "/api/v1/catalog/claim-events/feed", q)
}

// UserClaimItem is one work a user has moved through its lifecycle — the
// registry's answer to "my submissions".
//
// The Last* block is the work's LATEST transition BY ANYONE, not by this user.
// That is the point of it: what a submitter needs on their own submission is
// the reviewer's verdict and note, which is by definition an event they did
// not cause.
type UserClaimItem struct {
	WorkID        int64  `json:"work_id"`
	DisplayName   string `json:"display_name"`
	Site          string `json:"site"`
	ProductWorkID *int64 `json:"product_work_id"`
	ClaimState    string `json:"claim_state"`

	LastEventID   int64     `json:"last_event_id"`
	LastFromState *string   `json:"last_from_state"`
	LastToState   string    `json:"last_to_state"`
	LastReason    *string   `json:"last_reason"`
	LastActorUID  int64     `json:"last_actor_uid"`
	LastEventAt   time.Time `json:"last_event_at"`

	// FirstActedAt is when this user first touched the claim — "submitted on"
	// for the common case.
	FirstActedAt time.Time `json:"first_acted_at"`
	// ActedCount is how many transitions this user caused on this work.
	ActedCount int `json:"acted_count"`
}

// UserClaimPage is one DESCENDING cursor page.
//
// Total is the count under the SAME filter, independent of the cursor — which
// is what makes a separate per-user stats endpoint unnecessary: "how many works
// has this user published" is this call with claim_state=live and limit=1.
type UserClaimPage struct {
	Items []UserClaimItem `json:"items"`
	// NextBefore is the cursor for the following page; 0 = no more rows.
	NextBefore int64 `json:"next_before"`
	Total      int64 `json:"total"`
}

// UserClaimFilter is one page request against the per-user face.
type UserClaimFilter struct {
	// Site restricts to one tenant (empty = every site the user acted on).
	Site string
	// ClaimStates is the public vocabulary filter; empty = every state.
	ClaimStates []string
	// Before is the exclusive cursor (the previous page's next_before);
	// 0 = the first page.
	Before int64
	Limit  int
}

// UserClaims lists the works a user has acted on, newest activity first.
//
// This is the THIRD-PERSON read and the only reason the S2S claim face still
// exists: a profile page counts what somebody else has published, and there is
// no token of theirs to hold. The caller's OWN list is MyClaims — asking this
// one with the session's uid would work and would still be wrong, because it
// makes the forum the authority on who is asking.
func (c *Client) UserClaims(ctx context.Context, uid int64, f UserClaimFilter) (*UserClaimPage, error) {
	q := url.Values{}
	if f.Site != "" {
		q.Set("site", f.Site)
	}
	if len(f.ClaimStates) > 0 {
		q.Set("claim_state", joinStates(f.ClaimStates))
	}
	if f.Before > 0 {
		q.Set("before", strconv.FormatInt(f.Before, 10))
	}
	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}
	return editGetQuery[UserClaimPage](ctx,
		c, "/api/v1/catalog/users/"+strconv.FormatInt(uid, 10)+"/claims", q)
}

func joinStates(states []string) string {
	out := ""
	for i, s := range states {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}

// editGetQuery is editGet with a query string. It keeps the envelope's
// business code so a 400 on a bad claim_state token reaches the caller as an
// actionable message rather than a bare 503.
func editGetQuery[T any](ctx context.Context, c *Client, path string, q url.Values) (*T, error) {
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	return editGet[T](ctx, c, path)
}
