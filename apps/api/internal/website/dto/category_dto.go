package dto

import "time"

type UpdateWebsiteCategoryRequest struct {
	CategoryID  int    `json:"category_id" validate:"required,min=1"`
	Name        string `json:"name" validate:"required,min=1,max=30"`
	Label       string `json:"label" validate:"required,min=1,max=30"`
	Description string `json:"description" validate:"max=300"`
}

type WebsiteCategoryDetailResponse struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Label        string        `json:"label"`
	Description  string        `json:"description"`
	WebsiteCount int           `json:"website_count"`
	Websites     []WebsiteCard `json:"websites"`
	Created      time.Time     `json:"created"`
	Updated      time.Time     `json:"updated"`
}
