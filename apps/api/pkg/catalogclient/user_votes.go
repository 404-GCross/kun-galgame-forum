// The catalog USER-TOKEN write plane (wave 176), the first face this BFF
// speaks as the USER rather than as itself.
//
// Every other call in this package is S2S: HTTP Basic with kungal's OAuth
// client credentials, with the end user ASSERTED in the payload (see
// edit.go's EditActor). The user plane inverts that — the forum forwards the
// logged-in user's own OAuth access token as a Bearer, and the catalog derives
// BOTH the actor and the acting site from the token itself. Nothing about who
// is voting rides in the body; there is no body at all.
//
// Consequences worth stating, because they are the reason this file is
// separate rather than one more helper in edit.go:
//
//   - The token must carry the `catalog:edit` scope. A session authorized
//     BEFORE the scope existed does not have it and never will — refreshing
//     the token cannot widen a grant. Those users get a 403 that means
//     "log out and back in", not "you are not allowed", which is why it is
//     surfaced as its own sentinel (ErrInsufficientScope) instead of being
//     folded into a generic denial.
//   - The Basic credential is NOT sent. Attaching both would let the service
//     fall back to the S2S posture and silently defeat the point of the
//     dogfood.
package catalogclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// userBase is the user-token face's prefix, mounted on the same host as the
// S2S face (KUN_CATALOG_API_BASE).
const userBase = "/api/v1/user/catalog"

// ErrInsufficientScope is the one denial a caller must be able to act on: the
// user's access token is valid but predates (or otherwise lacks) the
// `catalog:edit` grant, so the fix is a re-login, not a permission change.
var ErrInsufficientScope = errors.New("catalogclient: access token lacks the catalog:edit scope")

// UserAPIError is a non-2xx user-plane reply whose message is worth showing.
// The sentinels above cover the cases a handler must branch on; this carries
// everything else with the upstream's own wording intact.
type UserAPIError struct {
	Status  int
	Code    int
	Message string
}

func (e *UserAPIError) Error() string {
	return fmt.Sprintf("catalog user plane: status=%d code=%d %s", e.Status, e.Code, e.Message)
}

// CoverVoteResult is the tally after the write, so the caller re-renders the
// cover without a second read.
type CoverVoteResult struct {
	CoverID   int64 `json:"cover_id"`
	VoteCount int64 `json:"vote_count"`
	// Voted is the acting user's resulting state on THIS cover: true after a
	// vote, false after a withdrawal.
	Voted bool `json:"voted"`
}

// VoteCover casts (or MOVES) the user's single best-cover ballot for a work
// onto coverID. A user holds at most one ballot per work, so voting a second
// cover moves the first rather than adding one.
func (c *Client) VoteCover(ctx context.Context, accessToken string, workID, coverID int64) (*CoverVoteResult, error) {
	return c.coverVote(ctx, http.MethodPut, accessToken, workID, coverID)
}

// UnvoteCover withdraws the user's ballot on a work. Idempotent: no ballot to
// withdraw is still a 200.
func (c *Client) UnvoteCover(ctx context.Context, accessToken string, workID, coverID int64) (*CoverVoteResult, error) {
	return c.coverVote(ctx, http.MethodDelete, accessToken, workID, coverID)
}

func (c *Client) coverVote(ctx context.Context, method, accessToken string, workID, coverID int64) (*CoverVoteResult, error) {
	// The user plane needs only the BASE URL — it authenticates with the user's
	// token, not with the client credential Configured() also requires.
	if c.baseURL == "" {
		return nil, ErrNotConfigured
	}
	if accessToken == "" {
		return nil, ErrUnauthorized
	}
	path := fmt.Sprintf("%s/works/%d/covers/%d/vote", userBase, workID, coverID)
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	// A body that does not parse still has a status worth honouring — a proxy
	// error page in front of a dead service must not read as a vote.
	decodeErr := json.NewDecoder(resp.Body).Decode(&env)

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusForbidden:
		if isScopeDenial(env.Message) {
			return nil, ErrInsufficientScope
		}
		return nil, &UserAPIError{Status: resp.StatusCode, Code: env.Code, Message: env.Message}
	case http.StatusNotFound:
		return nil, ErrNotFound
	}
	if decodeErr != nil {
		return nil, fmt.Errorf("%w: malformed envelope", ErrUpstream)
	}
	if resp.StatusCode != http.StatusOK || env.Code != 0 {
		return nil, &UserAPIError{Status: resp.StatusCode, Code: env.Code, Message: env.Message}
	}
	var out CoverVoteResult
	if err := json.Unmarshal(env.Data, &out); err != nil {
		return nil, fmt.Errorf("%w: malformed vote result", ErrUpstream)
	}
	return &out, nil
}

// isScopeDenial separates the ONE 403 a user can fix themselves from the ones
// they cannot. The catalog denies a missing scope with the OAuth vocabulary
// ("missing required scope: catalog:edit" / "insufficient_scope"); a client or
// site binding problem is an operator's bug and names neither. Matching on the
// word rather than on a numeric code is deliberate: the face publishes the
// house envelope's generic forbidden code for every 403, so the message is the
// only thing that distinguishes them, and a wrong guess here degrades to the
// generic denial rather than to a wrong instruction.
func isScopeDenial(message string) bool {
	return strings.Contains(strings.ToLower(message), "scope")
}
