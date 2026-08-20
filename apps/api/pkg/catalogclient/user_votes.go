package catalogclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const userBase = "/api/v1/user/catalog"

var ErrInsufficientScope = errors.New("catalogclient: access token lacks the scope the call needs")

type UserAPIError struct {
	Status  int
	Code    int
	Message string
}

func (e *UserAPIError) Error() string {
	return fmt.Sprintf("catalog user plane: status=%d code=%d %s", e.Status, e.Code, e.Message)
}

type CoverVoteResult struct {
	CoverID   int64 `json:"cover_id"`
	VoteCount int64 `json:"vote_count"`
	Voted     bool  `json:"voted"`
}

func (c *Client) VoteCover(ctx context.Context, accessToken string, workID, coverID int64) (*CoverVoteResult, error) {
	return c.coverVote(ctx, http.MethodPut, accessToken, workID, coverID)
}

func (c *Client) UnvoteCover(ctx context.Context, accessToken string, workID, coverID int64) (*CoverVoteResult, error) {
	return c.coverVote(ctx, http.MethodDelete, accessToken, workID, coverID)
}

func (c *Client) coverVote(ctx context.Context, method, accessToken string, workID, coverID int64) (*CoverVoteResult, error) {
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

func isScopeDenial(message string) bool {
	return strings.Contains(strings.ToLower(message), "scope")
}
