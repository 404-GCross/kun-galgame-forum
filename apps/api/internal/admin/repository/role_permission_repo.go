package repository

import (
	"context"
	"time"

	"kun-galgame-api/internal/admin/model"

	"gorm.io/gorm"
)

// RolePermissionRepository persists the role→permission override deltas
// (migration 062). The table is tiny (only deviations from the compiled
// baseline), so a full scan is cheap.
type RolePermissionRepository struct {
	db *gorm.DB
}

func NewRolePermissionRepository(db *gorm.DB) *RolePermissionRepository {
	return &RolePermissionRepository{db: db}
}

// ListAll returns every override row across all roles, in a stable order.
func (r *RolePermissionRepository) ListAll(ctx context.Context) ([]model.RolePermissionOverride, error) {
	var rows []model.RolePermissionOverride
	err := r.db.WithContext(ctx).Order("role ASC, permission ASC").Find(&rows).Error
	return rows, err
}

// ReplaceForRole atomically replaces one role's ENTIRE override set: delete all
// its existing rows, then insert the new set stamped with the operator + now. One
// transaction, so a partial replace can never be observed. An empty rows slice
// resets the role to its pure baseline (delete-only).
func (r *RolePermissionRepository) ReplaceForRole(ctx context.Context, role string, rows []model.RolePermissionOverride, operatorUID int) error {
	now := time.Now()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role = ?", role).Delete(&model.RolePermissionOverride{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for i := range rows {
			rows[i].Role = role
			rows[i].UpdatedBy = operatorUID
			rows[i].UpdatedAt = now
		}
		return tx.Create(&rows).Error
	})
}
