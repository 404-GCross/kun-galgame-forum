package handler

import (
	"encoding/json"
	"log/slog"

	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// GalgameMessageHandler exposes the galgame notification stream to kungal users
// and admins, plus the kungal-local read-state cursor.
type GalgameMessageHandler struct {
	svc *service.GalgameMessageService
}

func NewGalgameMessageHandler(svc *service.GalgameMessageService) *GalgameMessageHandler {
	return &GalgameMessageHandler{svc: svc}
}

// MessagesMine — GET /api/galgame/messages/mine (any authenticated user)
func (h *GalgameMessageHandler) MessagesMine(c fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	data, appErr := h.svc.MessagesMine(c.Context(), token, collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": json.RawMessage(data)})
}

// AdminMessages — GET /api/admin/galgame/messages (moderator+)
// Caller must already be in a RequireModerator()-gated route group.
func (h *GalgameMessageHandler) AdminMessages(c fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	data, appErr := h.svc.AdminMessages(c.Context(), token, collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return c.JSON(fiber.Map{"code": 0, "message": "成功", "data": json.RawMessage(data)})
}

// ReadStateRequest matches the PUT body for /api/galgame/messages/read-state.
type ReadStateRequest struct {
	LastReadMessageID int64 `json:"last_read_message_id"`
}

// GetReadState — GET /api/galgame/messages/read-state
//
// Returns the cursor as { last_read_message_id: <int64> }. Frontend uses
// this together with the /messages/mine list to compute unread counts.
func (h *GalgameMessageHandler) GetReadState(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	last, err := h.svc.GetReadState(user.ID)
	if err != nil {
		slog.Warn("查询 galgame 消息已读游标失败", "userID", user.ID, "error", err)
		return response.Error(c, errors.ErrInternal("查询失败"))
	}
	return response.OK(c, fiber.Map{"last_read_message_id": last})
}

// SetReadState — PUT /api/galgame/messages/read-state
func (h *GalgameMessageHandler) SetReadState(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req ReadStateRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求体格式错误"))
	}
	if req.LastReadMessageID < 0 {
		return response.Error(c, errors.ErrBadRequest("last_read_message_id 不能为负"))
	}

	if err := h.svc.SetReadState(user.ID, req.LastReadMessageID); err != nil {
		slog.Warn("更新 galgame 消息已读游标失败",
			"userID", user.ID, "last_id", req.LastReadMessageID, "error", err)
		return response.Error(c, errors.ErrInternal("更新失败"))
	}
	return response.OKMessage(c, "已读状态已更新")
}
