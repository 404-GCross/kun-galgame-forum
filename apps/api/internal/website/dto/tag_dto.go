package dto

import "time"

type CreateWebsiteTagRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=30"`
	Label       string `json:"label" validate:"required,min=1,max=30"`
	Description string `json:"description" validate:"max=300"`
	Level       int    `json:"level" validate:"min=0,max=20"`
}

type UpdateWebsiteTagRequest struct {
	TagID       int    `json:"tag_id" validate:"required,min=1"`
	Name        string `json:"name" validate:"required,min=1,max=30"`
	Label       string `json:"label" validate:"required,min=1,max=30"`
	Description string `json:"description" validate:"max=300"`
	Level       int    `json:"level" validate:"min=0,max=20"`
}

type DeleteWebsiteTagRequest struct {
	TagID int `query:"tag_id" validate:"required,min=1"`
}

type WebsiteTagDetailResponse struct {
	ID           int           `json:"id"`
	Name         string        `json:"name"`
	Label        string        `json:"label"`
	Level        int           `json:"level"`
	Description  string        `json:"description"`
	WebsiteCount int           `json:"website_count"`
	Websites     []WebsiteCard `json:"websites"`
	Created      time.Time     `json:"created"`
	Updated      time.Time     `json:"updated"`
}
