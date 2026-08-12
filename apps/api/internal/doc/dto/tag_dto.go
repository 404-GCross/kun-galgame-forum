package dto

type GetTagsRequest struct {
	Page    int    `query:"page" validate:"min=1"`
	Limit   int    `query:"limit" validate:"min=1,max=100"`
	Keyword string `query:"keyword"`
}

type DeleteTagRequest struct {
	TagID int `query:"tag_id" validate:"required,min=1"`
}

type CreateTagRequest struct {
	Slug        string `json:"slug" validate:"required,min=1,max=100"`
	Title       string `json:"title" validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"max=500"`
}

type UpdateTagRequest struct {
	TagID       int    `json:"tag_id" validate:"required,min=1"`
	Slug        string `json:"slug" validate:"required,min=1,max=100"`
	Title       string `json:"title" validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"max=500"`
}
