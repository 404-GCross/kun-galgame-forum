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

const userEditBase = userBase + "/edit"

type UserEditCreateRequest struct {
	EntityType string         `json:"entity_type"`
	EntityID   int64          `json:"entity_id"`
	Patch      map[string]any `json:"patch"`
	Note       string         `json:"note,omitempty"`
}

func (c *Client) CreateEditProposalUser(ctx context.Context, accessToken string, req UserEditCreateRequest) (*EditCreateResult, error) {
	return userEditPost[EditCreateResult](ctx, c, accessToken, userEditBase+"/proposals", req)
}

func (c *Client) WithdrawEditProposalUser(ctx context.Context, accessToken string, id int64) (*EditProposal, error) {
	return userEditPost[EditProposal](ctx, c, accessToken,
		userEditBase+"/proposals/"+strconv.FormatInt(id, 10)+"/withdraw", struct{}{})
}

func (c *Client) GetEditProposalUser(ctx context.Context, accessToken string, id int64) (*EditProposal, error) {
	return userEditDo[EditProposal](ctx, c, http.MethodGet, accessToken,
		userEditBase+"/proposals/"+strconv.FormatInt(id, 10), nil)
}

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

func (c *Client) MergeEditProposalUser(ctx context.Context, accessToken string, id int64, note string) (*EditRevision, error) {
	return userEditPost[EditRevision](ctx, c, accessToken,
		userEditBase+"/proposals/"+strconv.FormatInt(id, 10)+"/merge", map[string]any{"note": note})
}

func (c *Client) DeclineEditProposalUser(ctx context.Context, accessToken string, id int64, note string) (*EditProposal, error) {
	return userEditPost[EditProposal](ctx, c, accessToken,
		userEditBase+"/proposals/"+strconv.FormatInt(id, 10)+"/decline", map[string]any{"note": note})
}

func (c *Client) RevertEditEntityUser(ctx context.Context, accessToken string, entityType string, entityID int64, toSeq int, note string) (*EditRevertResult, error) {
	return userEditPost[EditRevertResult](ctx, c, accessToken, userEditBase+"/revert", map[string]any{
		"entity_type": entityType, "entity_id": entityID, "to_seq": toSeq, "note": note,
	})
}

func (c *Client) GetEditSchemaUser(ctx context.Context, accessToken, entityType string, entityID int64) (*EditSchema, error) {
	path := userEditBase + "/schema/" + entityType
	if entityID > 0 {
		q := url.Values{}
		q.Set("entity_id", strconv.FormatInt(entityID, 10))
		path += "?" + q.Encode()
	}
	return userEditDo[EditSchema](ctx, c, http.MethodGet, accessToken, path, nil)
}

func (c *Client) EditSnapshotUser(ctx context.Context, accessToken, entityType string, entityID int64) (map[string]any, error) {
	q := url.Values{}
	q.Set("entity_type", entityType)
	q.Set("entity_id", strconv.FormatInt(entityID, 10))
	data, err := userEditDo[struct {
		Values map[string]any `json:"values"`
	}](ctx, c, http.MethodGet, accessToken, userEditBase+"/snapshot?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	return data.Values, nil
}

type UserEditProposalFilter struct {
	EntityType string
	EntityID   int64
	Status     string
	Limit      int
	Mine       bool
}

func (c *Client) ListEditProposalsUser(ctx context.Context, accessToken string, f UserEditProposalFilter) ([]EditProposal, error) {
	q := url.Values{}
	if f.EntityType != "" {
		q.Set("entity_type", f.EntityType)
	}
	if f.EntityID > 0 {
		q.Set("entity_id", strconv.FormatInt(f.EntityID, 10))
	}
	if f.Status != "" {
		q.Set("status", f.Status)
	}
	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}
	if f.Mine {
		q.Set("mine", "true")
	}
	page, err := userEditDo[proposalListPage](ctx, c, http.MethodGet, accessToken,
		userEditBase+"/proposals?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

func (c *Client) WorkCoversUser(ctx context.Context, accessToken string, workID int64) ([]CoverTally, error) {
	data, err := userEditDo[struct {
		Covers []CoverTally `json:"covers"`
	}](ctx, c, http.MethodGet, accessToken,
		userClaimBase+"/works/"+strconv.FormatInt(workID, 10)+"/covers", nil)
	if err != nil {
		return nil, err
	}
	return data.Covers, nil
}

func userEditPost[T any](ctx context.Context, c *Client, accessToken, path string, body any) (*T, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return userEditDo[T](ctx, c, http.MethodPost, accessToken, path, raw)
}

func userEditDo[T any](ctx context.Context, c *Client, method, accessToken, path string, body []byte) (*T, error) {
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
