package dto

import (
	"time"

	adminModel "kun-galgame-api/internal/admin/model"
)

type UserBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type TodoItem struct {
	ID            int        `json:"id"`
	Type          string     `json:"type"`
	Status        int        `json:"status"`
	Content       string     `json:"content"`
	CompletedTime *time.Time `json:"completed_time"`
	User          UserBrief  `json:"user"`
	ClaimedUser   *UserBrief `json:"claimed_user,omitempty"`
	Created       time.Time  `json:"created"`
	Updated       time.Time  `json:"updated"`
}

func TodoItemOf(t adminModel.Todo, user UserBrief) TodoItem {
	return TodoItem{
		ID: t.ID, Type: t.Type, Status: t.Status, Content: t.Content,
		CompletedTime: t.CompletedTime, User: user,
		Created: t.CreatedAt, Updated: t.UpdatedAt,
	}
}

type ListQuery struct {
	Page  int `query:"page" validate:"min=1"`
	Limit int `query:"limit" validate:"min=1,max=50"`
}

type TodoListQuery struct {
	Page   int  `query:"page" validate:"min=1"`
	Limit  int  `query:"limit" validate:"min=1,max=50"`
	Status *int `query:"status" validate:"omitempty,min=0,max=10"`
}

type CreateHistoryRequest struct {
	Type    string `json:"type" validate:"required"`
	Version string `json:"version" validate:"max=20"`
	Content string `json:"content" validate:"max=1000"`
}

type DeleteHistoryRequest struct {
	ID int `query:"update_log_id" validate:"required,min=1"`
}

// No Status: a todo is always born pending. Taking one from the request let
// any signed-in user open a todo that was already 已完成, and one born
// 进行中 had no claimer, so nothing could claim, complete or discard it ever
// again.
type CreateTodoRequest struct {
	Type    string `json:"type" validate:"required"`
	Content string `json:"content" validate:"max=1000"`
}

type DeleteTodoRequest struct {
	ID int `query:"todo_id" validate:"required,min=1"`
}

type UpdateHistoryRequest struct {
	ID      int    `json:"update_log_id" validate:"required,min=1"`
	Type    string `json:"type" validate:"required"`
	Version string `json:"version" validate:"max=20"`
	Content string `json:"content" validate:"max=1000"`
}

type UpdateTodoRequest struct {
	ID      int    `json:"todo_id" validate:"required,min=1"`
	Type    string `json:"type" validate:"required"`
	Content string `json:"content" validate:"max=1000"`
}

type TodoActionRequest struct {
	TodoID int `json:"todo_id" validate:"required,min=1"`
}
