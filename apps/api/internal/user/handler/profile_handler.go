package handler

import (
	stderrors "errors"
	"encoding/json"
	"fmt"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/userclient"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type ProfileHandler struct {
	oauthClient *oauth.Client
	userClient  *userclient.Client
}

func NewProfileHandler(oauthClient *oauth.Client, userClient *userclient.Client) *ProfileHandler {
	return &ProfileHandler{oauthClient: oauthClient, userClient: userClient}
}

func (h *ProfileHandler) UpdateBio(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.UpdateBioRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	data, err := h.callPatchAuthMe(c, map[string]any{"bio": req.Bio})
	if err != nil {
		return response.Error(c, err)
	}
	h.userClient.Invalidate(user.ID)
	return response.OK(c, json.RawMessage(data))
}

func (h *ProfileHandler) UpdateUsername(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.UpdateUsernameRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	data, err := h.callPatchAuthMe(c, map[string]any{"name": req.Username})
	if err != nil {
		return response.Error(c, err)
	}
	h.userClient.Invalidate(user.ID)
	return response.OK(c, json.RawMessage(data))
}

func (h *ProfileHandler) UploadAvatar(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	contentType := c.Get("Content-Type")
	body := c.Body()
	if len(body) == 0 || contentType == "" {
		return response.Error(c, errors.ErrBadRequest("缺少文件"))
	}

	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	data, err := h.oauthClient.UploadAvatar(token, body, contentType)
	if err != nil {
		return response.Error(c, mapOAuthError(err))
	}
	h.userClient.Invalidate(user.ID)
	return response.OK(c, json.RawMessage(data))
}

func (h *ProfileHandler) callPatchAuthMe(c fiber.Ctx, body map[string]any) (json.RawMessage, *errors.AppError) {
	token := middleware.GetAccessToken(c)
	if token == "" {
		return nil, errors.ErrAuthExpired()
	}
	data, err := h.oauthClient.PatchAuthMe(token, body)
	if err != nil {
		return nil, mapOAuthError(err)
	}
	return data, nil
}

func mapOAuthError(err error) *errors.AppError {
	var oe *oauth.Error
	if stderrors.As(err, &oe) {
		if oe.Code != 0 {
			return errors.New(oe.Code, oe.Message, oe.HTTPStatus)
		}
		return errors.ErrInternal(fmt.Sprintf("OAuth 服务不可达: %v", err))
	}
	return errors.ErrInternal("更新失败")
}
