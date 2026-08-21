package catalogclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// The playtime face hangs off /v1, not the /api/v1/user/catalog prefix the rest
// of the user plane uses: upstream mounts it as its own Fiber group so a desktop
// tracker can hold a token scoped to playtime alone.
const playtimeBase = "/v1/playtime"

const (
	PlaytimeStatusPlaying  = "playing"
	PlaytimeStatusFinished = "finished"
	PlaytimeStatusDropped  = "dropped"
	PlaytimeStatusOnHold   = "on_hold"
)

// Catalog refuses anything above this, and its aggregate job ignores anything
// below PlaytimeMinutesFloor — a report under the floor is stored but never
// reaches the public median, which is how a user withdraws one.
const (
	PlaytimeMinutesMax   = 60_000
	PlaytimeMinutesFloor = 10
)

type PlaytimeReport struct {
	Minutes int    `json:"minutes"`
	Status  string `json:"status,omitempty"`
}

type PlaytimeRecord struct {
	WorkID       int64   `json:"work_id"`
	Minutes      int     `json:"minutes"`
	Status       string  `json:"status"`
	LastPlayedAt *string `json:"last_played_at"`
	ClientID     string  `json:"client_id"`
	UpdatedAt    string  `json:"updated_at"`
}

type PlaytimeSelf struct {
	WorkID       int64   `json:"work_id"`
	Minutes      int     `json:"minutes"`
	Status       string  `json:"status"`
	LastPlayedAt *string `json:"last_played_at"`
	Clients      int     `json:"clients"`
}

func (c *Client) ReportPlaytime(ctx context.Context, accessToken string, workID int64,
	report PlaytimeReport) (*PlaytimeRecord, error) {

	body, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	var out PlaytimeRecord
	path := fmt.Sprintf("%s/works/%d", playtimeBase, workID)
	if err := c.playtimeCall(ctx, http.MethodPut, accessToken, path, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MyPlaytime answers (nil, nil) when the user has never reported on this work.
// Upstream calls that a 200 with a null payload, not a 404.
func (c *Client) MyPlaytime(ctx context.Context, accessToken string, workID int64) (*PlaytimeSelf, error) {
	var out *PlaytimeSelf
	path := fmt.Sprintf("%s/works/%d", playtimeBase, workID)
	if err := c.playtimeCall(ctx, http.MethodGet, accessToken, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListMyPlaytime pages the caller's own rows oldest-change-first, returning the
// cursor to hand back as updatedSince. One row per (work, client): the same work
// appears once per application that reported it.
func (c *Client) ListMyPlaytime(ctx context.Context, accessToken, updatedSince string,
	limit int) ([]PlaytimeRecord, string, error) {

	q := url.Values{}
	if updatedSince != "" {
		q.Set("updated_since", updatedSince)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	path := playtimeBase + "/mine"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out struct {
		Items  []PlaytimeRecord `json:"items"`
		Cursor *string          `json:"cursor"`
	}
	if err := c.playtimeCall(ctx, http.MethodGet, accessToken, path, nil, &out); err != nil {
		return nil, "", err
	}
	cursor := ""
	if out.Cursor != nil {
		cursor = *out.Cursor
	}
	return out.Items, cursor, nil
}

func (c *Client) playtimeCall(ctx context.Context, method, accessToken, path string,
	body []byte, out any) error {

	if c.baseURL == "" {
		return ErrNotConfigured
	}
	if accessToken == "" {
		return ErrUnauthorized
	}
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
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
		return ErrUnauthorized
	case http.StatusForbidden:
		if isScopeDenial(env.Message) {
			return ErrInsufficientScope
		}
		return &UserAPIError{Status: resp.StatusCode, Code: env.Code, Message: env.Message}
	case http.StatusNotFound:
		return ErrNotFound
	}
	if decodeErr != nil {
		return fmt.Errorf("%w: malformed envelope", ErrUpstream)
	}
	if resp.StatusCode != http.StatusOK || env.Code != 0 {
		return &UserAPIError{Status: resp.StatusCode, Code: env.Code, Message: env.Message}
	}
	if out == nil || len(env.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("%w: malformed playtime payload", ErrUpstream)
	}
	return nil
}
