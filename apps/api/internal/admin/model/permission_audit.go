package model

import "time"

type PermissionAuditLog struct {
	ID          int64        `gorm:"column:id;primaryKey" json:"id"`
	OperatorID  int          `gorm:"column:operator_id" json:"operator_id"`
	SubjectKind string       `gorm:"column:subject_kind" json:"subject_kind"`
	Subject     string       `gorm:"column:subject" json:"subject"`
	Action      string       `gorm:"column:action" json:"action"`
	BeforeRows  []AuditDelta `gorm:"column:before_rows;serializer:json;type:jsonb;default:'[]'" json:"before_rows"`
	AfterRows   []AuditDelta `gorm:"column:after_rows;serializer:json;type:jsonb;default:'[]'" json:"after_rows"`
	CreatedAt   time.Time    `gorm:"column:created_at" json:"created_at"`
}

func (PermissionAuditLog) TableName() string { return "permission_audit_log" }

type AuditDelta struct {
	Permission string `json:"permission"`
	Effect     string `json:"effect"`
}
