package handler

import (
	"kun-galgame-api/internal/image/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type ImageHandler struct {
	imageService *service.ImageService
}

func NewImageHandler(imageService *service.ImageService) *ImageHandler {
	return &ImageHandler{imageService: imageService}
}

// Keeps the upload proxy from doubling as a generic image-service tunnel.
// Every preset here MUST also be in this site's image_allowed_presets on the
// image_service side, or the upload 403s there instead of here.
var allowedGalgamePresets = map[string]struct{}{
	"galgame_banner":     {},
	"galgame_screenshot": {},
}

func (h *ImageHandler) UploadGalgameImage(c fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	preset := c.FormValue("preset")
	if _, ok := allowedGalgamePresets[preset]; !ok {
		return response.Error(c, errors.ErrBadRequest(
			"preset 必须为 galgame_banner 或 galgame_screenshot"))
	}

	file, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("请选择要上传的图片"))
	}
	if file.Size > service.MaxImageSize {
		return response.Error(c, errors.ErrBadRequest("图片大小不能超过 10MB"))
	}

	f, err := file.Open()
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("读取图片失败"))
	}
	defer f.Close()

	res, sErr := h.imageService.UploadGalgameImage(
		c.Context(), middleware.GetAccessToken(c), f, file.Filename, preset,
	)
	if sErr != nil {
		return response.Error(c, sErr)
	}
	return response.OK(c, res)
}

func (h *ImageHandler) UploadCoverImage(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	file, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("请选择要上传的图片"))
	}
	if file.Size > service.MaxImageSize {
		return response.Error(c, errors.ErrBadRequest("图片大小不能超过 10MB"))
	}

	f, err := file.Open()
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("读取图片失败"))
	}
	defer f.Close()

	res, sErr := h.imageService.UploadCoverImage(c.Context(), user.ID, f, file.Filename)
	if sErr != nil {
		return response.Error(c, sErr)
	}
	return response.OK(c, res)
}

func (h *ImageHandler) UploadTopicImage(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	file, err := c.FormFile("image")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("请选择要上传的图片"))
	}
	if file.Size > service.MaxImageSize {
		return response.Error(c, errors.ErrBadRequest("图片大小不能超过 10MB"))
	}

	f, err := file.Open()
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("读取图片失败"))
	}
	defer f.Close()

	key, sErr := h.imageService.UploadTopicImage(c.Context(), user.ID, f, file.Filename)
	if sErr != nil {
		return response.Error(c, sErr)
	}
	return response.OK(c, key)
}

func (h *ImageHandler) UploadMessageImage(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	file, err := c.FormFile("image")
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("请选择要上传的图片"))
	}
	if file.Size > service.MaxImageSize {
		return response.Error(c, errors.ErrBadRequest("图片大小不能超过 10MB"))
	}

	f, err := file.Open()
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("读取图片失败"))
	}
	defer f.Close()

	key, sErr := h.imageService.UploadMessageImage(c.Context(), user.ID, f, file.Filename)
	if sErr != nil {
		return response.Error(c, sErr)
	}
	return response.OK(c, key)
}
