package service

import (
	"context"
	"errors"
	"io"

	"kun-galgame-api/pkg/catalogclient"
	kunErrors "kun-galgame-api/pkg/errors"
)

type UploadGalgameResult struct {
	Hash         string            `json:"hash"`
	URL          string            `json:"url"`
	Width        int               `json:"width"`
	Height       int               `json:"height"`
	SizeBytes    int64             `json:"size_bytes"`
	VariantURLs  map[string]string `json:"variant_urls,omitempty"`
	Deduplicated bool              `json:"deduplicated"`
}

func (s *ImageService) UploadGalgameImage(
	ctx context.Context,
	accessToken string,
	r io.Reader,
	filename, preset string,
) (*UploadGalgameResult, *kunErrors.AppError) {
	if s.catalogClient == nil {
		return nil, kunErrors.ErrInternal("Catalog 客户端未配置")
	}

	res, err := s.catalogClient.UploadEditImageUser(ctx, accessToken, r, filename, preset)
	if err != nil {
		if errors.Is(err, catalogclient.ErrNotConfigured) {
			return nil, kunErrors.ErrInternal("Catalog 客户端未配置")
		}
		switch {
		case errors.Is(err, catalogclient.ErrInsufficientScope):
			return nil, kunErrors.ErrReauthRequired(
				"上传图片需要新的授权，请退出登录后重新登录以授予该权限")
		case errors.Is(err, catalogclient.ErrUnauthorized):
			return nil, kunErrors.ErrAuthExpired()
		}
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
