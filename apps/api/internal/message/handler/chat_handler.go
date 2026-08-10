package handler

import (
	"kun-galgame-api/internal/message/dto"
	"kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type ChatHandler struct {
	chatService *service.ChatService
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

func (h *ChatHandler) GetNavContact(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	items, appErr := h.chatService.GetNavContact(c.Context(), user.ID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

func (h *ChatHandler) GetChatHistory(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.GetChatHistoryRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	items, appErr := h.chatService.GetChatHistory(c.Context(), user.ID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

func (h *ChatHandler) SendChatMessage(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.SendChatMessageRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	if appErr := h.chatService.SendChatMessage(c.Context(), user.ID, user.Name, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "发送成功")
}

func (h *ChatHandler) RecallChatMessage(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.RecallChatMessageRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if req.MessageID <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的消息 ID"))
	}

	if appErr := h.chatService.RecallMessage(c.Context(), user.ID, req.MessageID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "撤回成功")
}
