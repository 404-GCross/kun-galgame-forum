// The editing engine on the USER-TOKEN plane (wave 177) — the human lanes of
// the schema-driven editor, spoken as the user instead of about them.
//
// edit.go's S2S face is not going away: it stays the channel for the lanes
// where kungal knows something the token cannot carry. Two facts split the
// traffic:
//
//   - Who is acting. On the Bearer face the catalog derives uid, roles and the
//     acting site from the token itself, so nothing about the actor rides in the
//     body — a forum that asserted an actor here would be asserting nothing.
//   - What the forum knows that the token does not. `is_entity_owner` is a
//     FORUM fact (galgame.creator_user_id), and there is no token claim for it;
//     on this face it is always false. So the owner lane — the entry creator's
//     direct-merge — keeps riding the asserted-actor S2S path, and only the
//     ordinary contributor lanes move here.
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

// GetEditSchemaUser reads the field schema plus the TOKEN subject's capability
// projection. No actor query parameters exist on this face — passing a uid
// would be a claim, and the point of the plane is that the caller makes none.
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
