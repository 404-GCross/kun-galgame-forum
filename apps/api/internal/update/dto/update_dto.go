package dto

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

type CreateTodoRequest struct {
	Type    string `json:"type" validate:"required"`
	Status  int    `json:"status" validate:"min=0,max=10"`
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
	Status  int    `json:"status" validate:"min=0,max=10"`
	Content string `json:"content" validate:"max=1000"`
}
