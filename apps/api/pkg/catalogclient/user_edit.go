// The editing engine on the USER-TOKEN plane — EVERY human lane of the
// schema-driven editor (wave 177 moved the contributor lanes here; wave 178
// finished the job with the adjudication lanes), spoken as the user instead of
// about them.
//
// What changed in wave 178 is where ownership lives. `is_entity_owner` used to
// be a FORUM fact (galgame.creator_user_id) with no token claim behind it, which
// is why the owner's direct-merge and the whole review chain had to keep riding
// the asserted-actor S2S face. The catalog now holds per-user ownership itself
// (catalog_work.owner_user_id, backfilled from that very column) and derives the
// capability from the token — so there is nothing left for the forum to assert
// and no lane left that a user token cannot carry.
//
// edit.go's S2S face survives only for the reads that make no claim at all
// (snapshot, revision log, diff, proposal lists) — nobody is "acting" there, so
// there is nothing for a token to say.
//
// The scope consequence is the same one wave 176 established: a token minted
// before `catalog:edit` existed cannot be widened by a refresh, so that 403
// comes back as ErrInsufficientScope ("log out and back in"), never as a
// generic denial.
package catalogclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// userEditBase is the user plane's editing-engine prefix. Same operations as
// editBase, different authority.
const userEditBase = userBase + "/edit"

// UserEditCreateRequest files a proposal AS the token's subject. It is
// EditCreateRequest minus the two fields the token already answers: the actor
// (uid + roles) and the site.
type UserEditCreateRequest struct {
	EntityType string         `json:"entity_type"`
	EntityID   int64          `json:"entity_id"`
	Patch      map[string]any `json:"patch"`
	Note       string         `json:"note,omitempty"`
}

// CreateEditProposalUser files an edit as the user. The result shape is the S2S
// one, automerge included: a token whose roles carry the review perm still
// direct-merges (Merged=true + Revision), an ordinary contributor's proposal
// lands open — the difference is decided upstream from the token's own roles,
// not by which method the BFF called.
func (c *Client) CreateEditProposalUser(ctx context.Context, accessToken string, req UserEditCreateRequest) (*EditCreateResult, error) {
	return userEditPost[EditCreateResult](ctx, c, accessToken, userEditBase+"/proposals", req)
}

// WithdrawEditProposalUser closes the token subject's OWN open proposal. The
// engine's proposer-uid check is the whole gate here: with the proposer taken
// from the token there is no way to withdraw somebody else's, so the BFF needs
// no local ownership pre-flight.
func (c *Client) WithdrawEditProposalUser(ctx context.Context, accessToken string, id int64) (*EditProposal, error) {
	return userEditPost[EditProposal](ctx, c, accessToken,
		userEditBase+"/proposals/"+strconv.FormatInt(id, 10)+"/withdraw", struct{}{})
}

// GetEditProposalUser reads one proposal — amendments and effective patch
// included — as the token's subject. Same decode shape as the S2S read; what
// differs is that the catalog fences the tenant and the viewer itself.
func (c *Client) GetEditProposalUser(ctx context.Context, accessToken string, id int64) (*EditProposal, error) {
	return userEditDo[EditProposal](ctx, c, http.MethodGet, accessToken,
		userEditBase+"/proposals/"+strconv.FormatInt(id, 10), nil)
}

// AmendEditProposalUser appends a maintainer patch delta as the user (the crown
// mechanism: correct a value / reject a field, then merge — double attribution).
// Whether the subject may amend at all is the catalog's call, derived from the
// token's roles and its ownership of the target entity.
func (c *Client) AmendEditProposalUser(ctx context.Context, accessToken string, id int64, set map[string]any, unset []string, note string) (*EditAmendment, error) {
	body := map[string]any{}
	if len(set) > 0 {
		body["set"] = set
	}
	if len(unset) > 0 {
		body["unset"] = unset
	}
	if note != "" {
		body["note"] = note
	}
	return userEditPost[EditAmendment](ctx, c, accessToken,
		userEditBase+"/proposals/"+strconv.FormatInt(id, 10)+"/amendments", body)
}

// MergeEditProposalUser merges an open proposal as the user (per-field rebase;
// a 409 reports unresolved conflicts).
func (c *Client) MergeEditProposalUser(ctx context.Context, accessToken string, id int64, note string) (*EditRevision, error) {
	return userEditPost[EditRevision](ctx, c, accessToken,
		userEditBase+"/proposals/"+strconv.FormatInt(id, 10)+"/merge", map[string]any{"note": note})
}

// DeclineEditProposalUser declines an open proposal with a reason, as the user.
func (c *Client) DeclineEditProposalUser(ctx context.Context, accessToken string, id int64, note string) (*EditProposal, error) {
	return userEditPost[EditProposal](ctx, c, accessToken,
		userEditBase+"/proposals/"+strconv.FormatInt(id, 10)+"/decline", map[string]any{"note": note})
}

// RevertEditEntityUser restores an entity's registered fields to a historical
// revision, as the user. No site rides in the body — the acting tenant comes off
// the token like everything else — and the engine still re-checks the review
// rule on every restored field.
func (c *Client) RevertEditEntityUser(ctx context.Context, accessToken string, entityType string, entityID int64, toSeq int, note string) (*EditRevertResult, error) {
	return userEditPost[EditRevertResult](ctx, c, accessToken, userEditBase+"/revert", map[string]any{
		"entity_type": entityType, "entity_id": entityID, "to_seq": toSeq, "note": note,
	})
}

// GetEditSchemaUser reads the field schema plus the TOKEN subject's capability
// projection. No actor query parameters exist on this face — passing a uid
// would be a claim, and the point of the plane is that the caller makes none.
//
// Since wave 178 the projection also reflects ENTITY OWNERSHIP: an owner's token
// gets can_review on the owner-reviewable fields with nothing asserted, because
// the catalog now knows who owns the work.
func (c *Client) GetEditSchemaUser(ctx context.Context, accessToken, entityType string, entityID int64) (*EditSchema, error) {
	path := userEditBase + "/schema/" + entityType
	if entityID > 0 {
		q := url.Values{}
		q.Set("entity_id", strconv.FormatInt(entityID, 10))
		path += "?" + q.Encode()
	}
	return userEditDo[EditSchema](ctx, c, http.MethodGet, accessToken, path, nil)
}

func userEditPost[T any](ctx context.Context, c *Client, accessToken, path string, body any) (*T, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return userEditDo[T](ctx, c, http.MethodPost, accessToken, path, raw)
}

// userEditDo is the user plane's transport: Bearer only, sentinels for the
// branches a handler must act on, and the upstream's own wording preserved on
// everything else (the edit face's 4xx replies carry validation details and
// policy reasons the user has to be able to read).
func userEditDo[T any](ctx context.Context, c *Client, method, accessToken, path string, body []byte) (*T, error) {
	// Only the base URL matters — this call authenticates with the user's token,
	// not with the client credential Configured() also demands.
	if c.baseURL == "" {
		return nil, ErrNotConfigured
	}
	if accessToken == "" {
		return nil, ErrUnauthorized
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// The Basic credential is deliberately NOT attached: sending both would let
	// the service fall back to the S2S posture and silently undo the dogfood.
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
	// error page in front of a dead service must not read as a filed proposal.
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
	var out T
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &out); err != nil {
			return nil, fmt.Errorf("%w: malformed data payload", ErrUpstream)
		}
	}
	return &out, nil
}
