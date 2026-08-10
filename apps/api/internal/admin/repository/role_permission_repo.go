package repository

import (
	"context"
	"time"

	"kun-galgame-api/internal/admin/model"

	"gorm.io/gorm"
)

type RolePermissionRepository struct {
	db *gorm.DB
}

func NewRolePermissionRepository(db *gorm.DB) *RolePermissionRepository {
	return &RolePermissionRepository{db: db}
}

func (r *RolePermissionRepository) ListAll(ctx context.Context) ([]model.RolePermissionOverride, error) {
	var rows []model.RolePermissionOverride
	err := r.db.WithContext(ctx).Order("role ASC, permission ASC").Find(&rows).Error
	return rows, err
}

func (r *RolePermissionRepository) ReplaceForRole(ctx context.Context, role string, rows []model.RolePermissionOverride, operatorUID int) error {
	now := time.Now()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before []model.RolePermissionOverride
		if err := tx.Where("role = ?", role).Order("permission ASC").Find(&before).Error; err != nil {
			return err
		}
		if err := tx.Where("role = ?", role).Delete(&model.RolePermissionOverride{}).Error; err != nil {
			return err
		}
		for i := range rows {
			rows[i].Role = role
			rows[i].UpdatedBy = operatorUID
			rows[i].UpdatedAt = now
		}
		if len(rows) > 0 {
			if err := tx.Create(&rows).Error; err != nil {
				return err
			}
		}
		return writeAudit(tx, operatorUID, "role", role,
			roleRowsToDeltas(before), roleRowsToDeltas(rows))
	})
}

func roleRowsToDeltas(rows []model.RolePermissionOverride) []model.AuditDelta {
	out := make([]model.AuditDelta, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.AuditDelta{Permission: r.Permission, Effect: r.Effect})
	}
	return out
}
