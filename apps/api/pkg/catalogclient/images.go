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

// images.go — the edit face's byte-upload leg (infra wave 169).
// Covers/screenshots are edited as HASH ROWS on the proposal face; this call is
// where the hash comes from: the catalog uploads the bytes under its own
// image-service identity (site "catalog", kept alive by its refping cron) and
// hands the hash back for the edit payload to carry.
//
// It moved to the USER plane in wave 180 (POST /api/v1/user/catalog/edit/images).
// It was the last asserted-actor WRITE the forum had left: an `actor_uid` form
// field naming whoever the forum said was uploading, stamped into the image
// audit trail on that word alone. The token carries the subject now, so the
// field is gone and the audit trail records a person the catalog verified.

// EditImageResult is the image-service upload result the catalog forwards.
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

// UploadEditImageUser forwards one cover/screenshot to the catalog edit face AS
// the token's subject. preset is the image-service vocabulary the FE already
// speaks: "galgame_banner" (cover) or "galgame_screenshot".
//
// The transport is hand-rolled rather than routed through userEditDo because
// the body is multipart, but the posture is identical: Bearer only, never the
// Basic credential alongside it (sending both would let the service fall back
// to the S2S posture and silently undo the dogfood), and the same user-plane
// taxonomy — a scope denial reaches the caller as ErrInsufficientScope so an
// old session is told to log back in rather than that it may not upload.
func (c *Client) UploadEditImageUser(ctx context.Context, accessToken string, r io.Reader, filename, preset string) (*EditImageResult, error) {
	// Only the base URL matters — this call authenticates with the user's token.
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
	// A body that does not parse still has a status worth honouring — a proxy
	// error page in front of a dead service must not read as a stored image.
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
