package handler

import (
	"time"

	adminModel "kun-galgame-api/internal/admin/model"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/update/dto"
	"kun-galgame-api/internal/update/repository"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type UpdateHandler struct {
	repo *repository.UpdateRepository
}

func NewUpdateHandler(repo *repository.UpdateRepository) *UpdateHandler {
	return &UpdateHandler{repo: repo}
}

func (h *UpdateHandler) GetHistory(c fiber.Ctx) error {
	var req dto.ListQuery
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	logs := h.repo.FindHistoryPaginated(req.Page, req.Limit)
	total := h.repo.CountHistory()

	return response.OK(c, fiber.Map{
		"updates": logs,
		"total":   total,
	})
}

func (h *UpdateHandler) CreateHistory(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.CreateHistoryRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	log := adminModel.UpdateLog{
		Type: req.Type, Version: req.Version,
		Content: req.Content,
		UserID:  user.ID,
	}
	if err := h.repo.CreateHistory(&log); err != nil {
		return response.Error(c, errors.ErrInternal("创建更新日志失败"))
	}
	return response.OKMessage(c, "更新日志已创建")
}

func (h *UpdateHandler) UpdateHistory(c fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateHistoryRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	fields := map[string]any{
		"type":    req.Type,
		"version": req.Version,
		"content": req.Content,
	}
	if err := h.repo.UpdateHistory(req.ID, fields); err != nil {
		return response.Error(c, errors.ErrInternal("更新日志失败"))
	}
	return response.OKMessage(c, "更新日志已更新")
}

func (h *UpdateHandler) DeleteHistory(c fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.DeleteHistoryRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if err := h.repo.DeleteHistory(req.ID); err != nil {
		return response.Error(c, errors.ErrInternal("删除更新日志失败"))
	}
	return response.OKMessage(c, "更新日志已删除")
}

func (h *UpdateHandler) GetTodos(c fiber.Ctx) error {
	var req dto.TodoListQuery
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	todos := h.repo.FindTodosPaginated(req.Page, req.Limit, req.Status)
	total := h.repo.CountTodos(req.Status)

	return response.OK(c, fiber.Map{
		"todos": todos,
		"total": total,
	})
}

func (h *UpdateHandler) CreateTodo(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.CreateTodoRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	todo := adminModel.Todo{
		Type: req.Type, Status: req.Status,
		Content: req.Content,
		UserID:  user.ID,
	}
	if err := h.repo.CreateTodo(&todo); err != nil {
		return response.Error(c, errors.ErrInternal("创建待办失败"))
	}
	return response.OKMessage(c, "待办已创建")
}

func (h *UpdateHandler) UpdateTodo(c fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateTodoRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	fields := map[string]any{
		"type":    req.Type,
		"status":  req.Status,
		"content": req.Content,
	}
	if req.Status == 2 {
		fields["completed_time"] = time.Now()
	} else {
		fields["completed_time"] = nil
	}
	if err := h.repo.UpdateTodo(req.ID, fields); err != nil {
		return response.Error(c, errors.ErrInternal("更新待办失败"))
	}
	return response.OKMessage(c, "待办已更新")
}

func (h *UpdateHandler) DeleteTodo(c fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.DeleteTodoRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if err := h.repo.DeleteTodo(req.ID); err != nil {
		return response.Error(c, errors.ErrInternal("删除待办失败"))
	}
	return response.OKMessage(c, "待办已删除")
}
