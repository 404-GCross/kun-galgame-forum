package repository

import (
	"context"

	"kun-galgame-api/internal/admin/model"

	"gorm.io/gorm"
)

func writeAudit(tx *gorm.DB, operatorID int, subjectKind, subject string, before, after []model.AuditDelta) error {
	row := buildAuditRow(operatorID, subjectKind, subject, before, after)
	return tx.Create(&row).Error
}

func buildAuditRow(operatorID int, subjectKind, subject string, before, after []model.AuditDelta) model.PermissionAuditLog {
	action := "replace"
	if len(after) == 0 {
		action = "reset"
	}
	return model.PermissionAuditLog{
		OperatorID:  operatorID,
		SubjectKind: subjectKind,
		Subject:     subject,
		Action:      action,
		BeforeRows:  nonNilDeltas(before),
		AfterRows:   nonNilDeltas(after),
	}
}

func nonNilDeltas(d []model.AuditDelta) []model.AuditDelta {
	if d == nil {
		return []model.AuditDelta{}
	}
	return d
}

type PermissionAuditRepository struct {
	db *gorm.DB
}

func NewPermissionAuditRepository(db *gorm.DB) *PermissionAuditRepository {
	return &PermissionAuditRepository{db: db}
}

func (r *PermissionAuditRepository) List(ctx context.Context, offset, limit int) ([]model.PermissionAuditLog, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.PermissionAuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.PermissionAuditLog
	err := r.db.WithContext(ctx).Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}
