package catalogclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type EditImageResult struct {
	Hash         string            `json:"hash"`
	URL          string            `json:"url"`
	VariantURLs  map[string]string `json:"variant_urls"`
	Width        int               `json:"width"`
	Height       int               `json:"height"`
	Thumbhash    string            `json:"thumbhash"`
	SizeBytes    int64             `json:"size_bytes"`
	Deduplicated bool              `json:"deduplicated"`
}

func (c *Client) UploadEditImageUser(ctx context.Context, accessToken string, r io.Reader, filename, preset string) (*EditImageResult, error) {
	if c.baseURL == "" {
		return nil, ErrNotConfigured
	}
	if accessToken == "" {
		return nil, ErrUnauthorized
	}

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("build multipart: %w", err)
	}
	if _, err := io.Copy(fw, r); err != nil {
		return nil, fmt.Errorf("copy file: %w", err)
	}
	if err := mw.WriteField("preset", preset); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+userEditBase+"/images", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	decodeErr := json.Unmarshal(raw, &env)

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusForbidden:
		if isScopeDenial(env.Message) {
			return nil, ErrInsufficientScope
		}
		return nil, &UserAPIError{Status: resp.StatusCode, Code: env.Code, Message: env.Message}
	}
	if decodeErr != nil {
		return nil, fmt.Errorf("%w (status %d): malformed envelope", ErrUpstream, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK || env.Code != 0 {
		return nil, &UserAPIError{Status: resp.StatusCode, Code: env.Code, Message: env.Message}
	}
	var out EditImageResult
	if err := json.Unmarshal(env.Data, &out); err != nil {
		return nil, fmt.Errorf("%w: malformed upload result", ErrUpstream)
	}
	return &out, nil
}
