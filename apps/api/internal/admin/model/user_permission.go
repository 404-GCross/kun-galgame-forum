package model

import "time"

type UserPermissionOverride struct {
	UserID     int       `gorm:"column:user_id;primaryKey" json:"user_id"`
	Permission string    `gorm:"column:permission;primaryKey" json:"permission"`
	Effect     string    `gorm:"column:effect;not null" json:"effect"`
	UpdatedBy  int       `gorm:"column:updated_by;not null;default:0" json:"updated_by"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (UserPermissionOverride) TableName() string { return "user_permission_override" }
