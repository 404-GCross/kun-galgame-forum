package dto

type GetCategoriesRequest struct {
	Page    int    `query:"page" validate:"min=1"`
	Limit   int    `query:"limit" validate:"min=1,max=100"`
	Keyword string `query:"keyword"`
}

type CreateCategoryRequest struct {
	Slug        string `json:"slug" validate:"required,min=1,max=100"`
	Title       string `json:"title" validate:"required,min=1,max=233"`
	Description string `json:"description" validate:"max=500"`
	Icon        string `json:"icon" validate:"max=200"`
	SortOrder   int    `json:"sort_order" validate:"min=0,max=9999"`
}

type UpdateCategoryRequest struct {
	CategoryID  int    `json:"category_id" validate:"required,min=1"`
	Slug        string `json:"slug" validate:"required,min=1,max=100"`
	Title       string `json:"title" validate:"required,min=1,max=233"`
	Description string `json:"description" validate:"max=500"`
	Icon        string `json:"icon" validate:"max=200"`
	SortOrder   int    `json:"sort_order" validate:"min=0,max=9999"`
}

type DeleteCategoryRequest struct {
	CategoryID int `query:"category_id" validate:"required,min=1"`
}
