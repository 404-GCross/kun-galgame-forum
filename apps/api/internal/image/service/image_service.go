package service

import (
	"context"
	"io"

	"kun-galgame-api/internal/image/repository"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/imageclient"
)

const (
	MaxImageSize    = 10 * 1024 * 1024
	dailyImageLimit = 50
)

type ImageService struct {
	repo          *repository.ImageRepository
	imgCli        *imageclient.Client
	catalogClient *catalogclient.Client
}

func NewImageService(
	repo *repository.ImageRepository,
	imgCli *imageclient.Client,
	catalogClient *catalogclient.Client,
) *ImageService {
	return &ImageService{repo: repo, imgCli: imgCli, catalogClient: catalogClient}
}

type UploadCoverResult struct {
	Hash      string `json:"hash"`
	URL       string `json:"url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Thumbhash string `json:"thumbhash,omitempty"`
}

func (s *ImageService) UploadCoverImage(ctx context.Context, userID int, r io.Reader, filename string) (*UploadCoverResult, *errors.AppError) {
	if s.imgCli == nil {
		return nil, errors.ErrBadRequest(
			"图片上传服务未配置 (KUN_IMAGE_CLIENT_ID / KUN_IMAGE_CLIENT_SECRET)",
		)
	}

	count, err := s.repo.GetDailyCount(userID)
	if err != nil {
		return nil, errors.ErrInternal("查询用户失败")
	}
	if count >= dailyImageLimit {
		return nil, errors.ErrBadRequest("今日图片上传次数已达上限")
	}

	res, uErr := s.imgCli.Upload(ctx, r, filename, "topic")
	if uErr != nil {
		if ie, ok := uErr.(*imageclient.Error); ok {
			return nil, errors.New(ie.Code, ie.Message, ie.StatusCode)
		}
		return nil, errors.ErrInternal("图片上传失败")
	}

	s.repo.IncrementDailyCount(userID)
	return &UploadCoverResult{
		Hash:      res.Hash,
		URL:       res.URL,
		Width:     res.Width,
		Height:    res.Height,
		Thumbhash: res.Thumbhash,
	}, nil
}

func (s *ImageService) UploadTopicImage(ctx context.Context, userID int, r io.Reader, filename string) (string, *errors.AppError) {
	if s.imgCli == nil {
		return "", errors.ErrBadRequest(
			"图片上传服务未配置 (KUN_IMAGE_CLIENT_ID / KUN_IMAGE_CLIENT_SECRET)",
		)
	}

	count, err := s.repo.GetDailyCount(userID)
	if err != nil {
		return "", errors.ErrInternal("查询用户失败")
	}
	if count >= dailyImageLimit {
		return "", errors.ErrBadRequest("今日图片上传次数已达上限")
	}

	res, uErr := s.imgCli.Upload(ctx, r, filename, "topic")
	if uErr != nil {
		if ie, ok := uErr.(*imageclient.Error); ok {
			return "", errors.New(ie.Code, ie.Message, ie.StatusCode)
		}
		return "", errors.ErrInternal("图片上传失败")
	}

	s.repo.IncrementDailyCount(userID)
	return "/image/" + res.Hash, nil
}

func (s *ImageService) UploadMessageImage(ctx context.Context, userID int, r io.Reader, filename string) (string, *errors.AppError) {
	if s.imgCli == nil {
		return "", errors.ErrBadRequest(
			"图片上传服务未配置 (KUN_IMAGE_CLIENT_ID / KUN_IMAGE_CLIENT_SECRET)",
		)
	}

	count, err := s.repo.GetDailyCount(userID)
	if err != nil {
		return "", errors.ErrInternal("查询用户失败")
	}
	if count >= dailyImageLimit {
		return "", errors.ErrBadRequest("今日图片上传次数已达上限")
	}

	res, uErr := s.imgCli.Upload(ctx, r, filename, "message")
	if uErr != nil {
		if ie, ok := uErr.(*imageclient.Error); ok {
			return "", errors.New(ie.Code, ie.Message, ie.StatusCode)
		}
		return "", errors.ErrInternal("图片上传失败")
	}

	s.repo.IncrementDailyCount(userID)
	return "/image/" + res.Hash, nil
}
