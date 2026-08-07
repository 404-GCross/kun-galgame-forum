package service

import (
	"context"
	"errors"
	"io"

	"kun-galgame-api/pkg/catalogclient"
	kunErrors "kun-galgame-api/pkg/errors"
)

// galgame_upload.go — U2 cover/screenshot upload path.
//
// Distinct from UploadTopicImage (returns a `/image/<hash>` inline token):
// galgame covers/screenshots reference image_service by `image_hash`,
// which means we MUST upload through image_service first to obtain the
// hash before the user can submit an edit carrying that hash.
//
// The bytes go through the CATALOG's edit-face upload leg (wave 169) rather
// than forum's own image client: uploads land under the catalog image-service
// identity (site "catalog"), the scope the catalog's daily refping keeps
// alive — forum's site=kungal could not keep a catalog-referenced image from
// the GC. The wiki-era proxy this replaces (POST /galgame/image) retired with
// the wiki in wave-161 P5.
//
// Allowed presets are constrained at the handler boundary; this layer
// trusts its caller. The catalog constrains them again (closed set), and
// image_service gates them per client a third time, so a wrong preset is a
// 4xx from upstream rather than a silent mis-store.

// UploadGalgameResult is the slim response surface the FE consumes for a
// new cover/screenshot row: the hash to attach to the edit payload +
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

// UploadGalgameImage forwards a galgame cover/screenshot to the catalog's
// edit-face upload leg and adapts the result.
//
// accessToken is the uploader's own OAuth token (wave 180). The asserted
// `userID` this used to pass is gone: it was a plain claim about who was
// uploading, honoured upstream on the forum's word alone, and it had no local
// use here to keep — no quota, no log line, nothing. The audit trail now
// records the subject the catalog verified for itself.
func (s *ImageService) UploadGalgameImage(
	ctx context.Context,
	accessToken string,
	r io.Reader,
	filename, preset string,
) (*UploadGalgameResult, *kunErrors.AppError) {
	// Configured() is deliberately not the pre-check any more: it asks whether
	// the S2S credential pair is present, and this lane no longer uses it. The
	// client answers "no base URL" itself, so an unwired catalog still reports
	// the same thing without a second, now-wrong, definition of wired.
	if s.catalogClient == nil {
		return nil, kunErrors.ErrInternal("Catalog 客户端未配置")
	}

	res, err := s.catalogClient.UploadEditImageUser(ctx, accessToken, r, filename, preset)
	if err != nil {
		if errors.Is(err, catalogclient.ErrNotConfigured) {
			return nil, kunErrors.ErrInternal("Catalog 客户端未配置")
		}
		// The user plane can refuse in two ways the S2S face never could, and
		// both are actionable: a session older than the `catalog:edit` scope
		// needs a re-login (no refresh can widen a grant), and a dead token
		// needs the ordinary re-auth.
		switch {
		case errors.Is(err, catalogclient.ErrInsufficientScope):
			return nil, kunErrors.ErrReauthRequired(
				"上传图片需要新的授权，请退出登录后重新登录以授予该权限")
		case errors.Is(err, catalogclient.ErrUnauthorized):
			return nil, kunErrors.ErrAuthExpired()
		}
		// A readable upstream envelope (quota, moderation, bad preset) is
		// worth relaying; transport-level failures stay generic.
		var apiErr *catalogclient.UserAPIError
		if errors.As(err, &apiErr) && apiErr.Message != "" {
			return nil, kunErrors.ErrBadRequest(apiErr.Message)
		}
		return nil, kunErrors.ErrInternal("上传 Galgame 图片失败")
	}
	return &UploadGalgameResult{
		Hash:         res.Hash,
		URL:          res.URL,
		Width:        res.Width,
		Height:       res.Height,
		SizeBytes:    res.SizeBytes,
		VariantURLs:  res.VariantURLs,
		Deduplicated: res.Deduplicated,
	}, nil
}
