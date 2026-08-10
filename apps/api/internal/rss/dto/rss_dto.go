package dto

import "time"

type TopicRSSItem struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	UserID      int       `json:"user_id"`
	UserName    string    `json:"user_name"`
	Created     time.Time `json:"created"`
}

type GalgameRSSUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type GalgameRSSItem struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	Banner      string         `json:"banner"`
	User        GalgameRSSUser `json:"user"`
	Description string         `json:"description"`
	Created     string         `json:"created"`
}
