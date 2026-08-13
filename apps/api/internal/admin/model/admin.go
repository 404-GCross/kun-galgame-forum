package model

import "time"

type UpdateLog struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Type        string `gorm:"not null" json:"type"`
	Version     string `gorm:"default:''" json:"version"`
	ContentEnUS string `gorm:"column:content_en_us;type:text;default:''" json:"content_en_us"`
	ContentJaJP string `gorm:"column:content_ja_jp;type:text;default:''" json:"content_ja_jp"`
	ContentZhCN string `gorm:"column:content_zh_cn;type:text;default:''" json:"content_zh_cn"`
	ContentZhTW string `gorm:"column:content_zh_tw;type:text;default:''" json:"content_zh_tw"`

	UserID int `gorm:"column:user_id;not null;default:2" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (UpdateLog) TableName() string { return "update_log" }

type Todo struct {
	ID            int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Type          string     `gorm:"default:'forum'" json:"type"`
	Status        int        `gorm:"default:0" json:"status"`
	ContentEnUS   string     `gorm:"column:content_en_us;type:text;default:''" json:"content_en_us"`
	ContentJaJP   string     `gorm:"column:content_ja_jp;type:text;default:''" json:"content_ja_jp"`
	ContentZhCN   string     `gorm:"column:content_zh_cn;type:text;default:''" json:"content_zh_cn"`
	ContentZhTW   string     `gorm:"column:content_zh_tw;type:text;default:''" json:"content_zh_tw"`
	CompletedTime *time.Time `gorm:"column:completed_time" json:"completed_time"`

	UserID int `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (Todo) TableName() string { return "todo" }
