package model

import "time"

type FriendLink struct {
	ID              int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Category        string    `gorm:"not null" json:"category"`
	Name            string    `gorm:"not null" json:"name"`
	Link            string    `gorm:"not null" json:"link"`
	Description     string    `gorm:"type:text;default:''" json:"description"`
	Banner          string    `gorm:"default:''" json:"banner"`
	BannerImageHash string    `gorm:"column:banner_image_hash;default:''" json:"banner_image_hash"`
	BannerURL       string    `gorm:"-" json:"banner_url"`
	Status          string    `gorm:"default:'normal'" json:"status"`
	SortOrder       int       `gorm:"column:sort_order;not null;default:0" json:"sort_order"`
	CreatedAt       time.Time `gorm:"column:created" json:"created"`
	UpdatedAt       time.Time `gorm:"column:updated" json:"updated"`
}

func (FriendLink) TableName() string { return "friend_link" }
