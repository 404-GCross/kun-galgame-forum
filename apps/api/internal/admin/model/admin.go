package model

import "time"

type UpdateLog struct {
	ID      int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Type    string `gorm:"not null" json:"type"`
	Version string `gorm:"default:''" json:"version"`
	Content string `gorm:"column:content;type:text;default:''" json:"content"`

	UserID int `gorm:"column:user_id;not null;default:2" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (UpdateLog) TableName() string { return "update_log" }

type Todo struct {
	ID            int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Type          string     `gorm:"default:'forum'" json:"type"`
	Status        int        `gorm:"default:0" json:"status"`
	Content       string     `gorm:"column:content;type:text;default:''" json:"content"`
	CompletedTime *time.Time `gorm:"column:completed_time" json:"completed_time"`

	ClaimedUserID *int `gorm:"column:claimed_user_id" json:"claimed_user_id"`
	UserID        int  `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (Todo) TableName() string { return "todo" }
