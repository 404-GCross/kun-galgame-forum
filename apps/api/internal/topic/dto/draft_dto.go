package dto

import "time"

type SaveTopicDraftRequest struct {
	Title       string   `json:"title" validate:"max=233"`
	Content     string   `json:"content" validate:"max=100007"`
	Category    string   `json:"category" validate:"omitempty,oneof=galgame technique others"`
	Sections    []string `json:"section" validate:"omitempty,max=3"`
	IsNSFW      bool     `json:"is_nsfw"`
	CoverImages []string `json:"cover_images" validate:"omitempty,max=9"`
}

type TopicDraftListItem struct {
	ID      int       `json:"id"`
	Title   string    `json:"title"`
	Summary string    `json:"summary"`
	Updated time.Time `json:"updated"`
}

type TopicDraftDetail struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Category    string    `json:"category"`
	Sections    []string  `json:"section"`
	IsNSFW      bool      `json:"is_nsfw"`
	CoverImages []string  `json:"cover_images"`
	Updated     time.Time `json:"updated"`
}
