package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"

	"kun-galgame-api/pkg/errors"
)

// galgame_upload.go — U2 cover/screenshot upload path.
//
// Distinct from UploadTopicImage (returns a `/image/<hash>` inline token):
// galgame covers/screenshots reference image_service by `image_hash`,
// which means we MUST upload through image_service first to obtain the
// hash before the user can submit a PUT/PR carrying that hash.
//
// Allowed presets are constrained at the handler boundary; this layer
// trusts its caller. image_service itself also gates presets against
// the calling site's `image_allowed_presets`, so a wrong preset returns
// a 4xx from upstream rather than silently mis-storing.
//
// Quota / size / format validation lives in image_service (see
// docs/image_service/03-api-design.md §1). We only forward + classify.

// UploadGalgameResult is the slim response surface the FE consumes for a
// new cover/screenshot row: the hash to attach to the galgame wire +
// the CDN URL it can render immediately (saves a hash→URL round-trip).
type UploadGalgameResult struct {
	Hash         string            `json:"hash"`
	URL          string            `json:"url"`
	Width        int               `json:"width"`
	Height       int               `json:"height"`
	SizeBytes    int64             `json:"size_bytes"`
	VariantURLs  map[string]string `json:"variant_urls,omitempty"`
	Deduplicated bool              `json:"deduplicated"`
}

// UploadGalgameImage proxies a galgame cover/screenshot to the WIKI's canonical
// upload endpoint (POST /galgame/image), forwarding the user's Bearer token.
// The wiki uploads under its own image client (site=galgame_wiki) and returns
// the hash, so every galgame image is OWNED by the wiki — forum no longer
// uploads galgame images under its own site=kungal (which the site-scoped
// galgame reference-ping can't keep alive). Topic/message inline images still
// go through forum's local image_service client (those are forum's own content).
//
// userID is kept for logging / future per-user accounting; the file is
// re-encoded as multipart {file, preset} and forwarded byte-for-byte.
func (s *ImageService) UploadGalgameImage(
	ctx context.Context,
	userID int,
	token string,
	r io.Reader,
	filename, preset string,
) (*UploadGalgameResult, *errors.AppError) {
	if s.wikiClient == nil {
		return nil, errors.ErrInternal("Wiki 客户端未配置")
	}

	// Re-encode the upload as multipart {file, preset}.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("preset", preset); err != nil {
		return nil, errors.ErrInternal("构建上传请求失败")
	}
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return nil, errors.ErrInternal("构建上传请求失败")
	}
	if _, err := io.Copy(fw, r); err != nil {
		return nil, errors.ErrInternal("读取上传文件失败")
	}
	if err := mw.Close(); err != nil {
		return nil, errors.ErrInternal("构建上传请求失败")
	}

	// PostWithToken forwards the multipart body verbatim (boundary preserved)
	// + the user's Bearer, and returns the wiki's `data` (or its error
	// code/status, surfaced to the caller unchanged).
	data, appErr := s.wikiClient.PostWithToken(
		ctx, "/galgame/image", token, buf.Bytes(), mw.FormDataContentType(),
	)
	if appErr != nil {
		return nil, appErr
	}

	// Wiki returns image_service's UploadResult (snake_case wire tags); adapt
	// it to the FE-facing UploadGalgameResult.
	var wiki struct {
		Hash         string            `json:"hash"`
		URL          string            `json:"url"`
		VariantURLs  map[string]string `json:"variant_urls"`
		Width        int               `json:"width"`
		Height       int               `json:"height"`
		SizeBytes    int64             `json:"size_bytes"`
		Deduplicated bool              `json:"deduplicated"`
	}
	if err := json.Unmarshal(data, &wiki); err != nil {
		return nil, errors.ErrInternal("解析 Wiki 上传响应失败")
	}
	return &UploadGalgameResult{
		Hash:         wiki.Hash,
		URL:          wiki.URL,
		Width:        wiki.Width,
		Height:       wiki.Height,
		SizeBytes:    wiki.SizeBytes,
		VariantURLs:  wiki.VariantURLs,
		Deduplicated: wiki.Deduplicated,
	}, nil
}
