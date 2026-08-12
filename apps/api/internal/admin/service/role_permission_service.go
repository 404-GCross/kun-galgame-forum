package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
)

type overrideStore interface {
	ListAll(ctx context.Context) ([]model.RolePermissionOverride, error)
	ReplaceForRole(ctx context.Context, role string, rows []model.RolePermissionOverride, operatorUID int) error
}

type reloader interface {
	Load(ctx context.Context) error
}

type RolePermissionService struct {
	repo   overrideStore
	reload reloader
}

func NewRolePermissionService(repo overrideStore, reload reloader) *RolePermissionService {
	return &RolePermissionService{repo: repo, reload: reload}
}

var editableRoles = map[string]bool{"creator": true, "moderator": true, "admin": true}

var matrixRoles = []string{"creator", "moderator", "admin", "ren"}

func (s *RolePermissionService) Matrix(ctx context.Context) (dto.RolePermissionMatrix, error) {
	rows, err := s.repo.ListAll(ctx)
	if err != nil {
		return dto.RolePermissionMatrix{}, err
	}
	return buildMatrix(rows), nil
}

func (s *RolePermissionService) ReplaceOverrides(ctx context.Context, operatorUID int, operatorRoles []string, role string, items []dto.ReplaceOverrideItem) (dto.RolePermissionMatrix, *errors.AppError) {
	current, err := s.repo.ListAll(ctx)
	if err != nil {
		return dto.RolePermissionMatrix{}, errors.ErrInternal("读取角色权限失败")
	}
	if appErr := validateReplace(operatorUID, operatorRoles, role, items, current); appErr != nil {
		return dto.RolePermissionMatrix{}, appErr
	}

	if err := s.repo.ReplaceForRole(ctx, role, itemsToRows(role, items), operatorUID); err != nil {
		return dto.RolePermissionMatrix{}, errors.ErrInternal("保存角色权限失败")
	}
	if err := s.reload.Load(ctx); err != nil {
		slog.Warn("写入后刷新角色权限覆盖失败, 稍后自动收敛", "error", err)
	}
	matrix, err := s.Matrix(ctx)
	if err != nil {
		return dto.RolePermissionMatrix{}, errors.ErrInternal("获取角色权限矩阵失败")
	}
	return matrix, nil
}

func validateReplace(operatorUID int, operatorRoles []string, role string, items []dto.ReplaceOverrideItem, current []model.RolePermissionOverride) *errors.AppError {
	if role == "ren" {
		return errors.ErrBadRequest("ren 角色的权限被固定为全部, 不可修改")
	}
	if !editableRoles[role] {
		return errors.ErrBadRequest("不支持管理该角色, 仅可管理 creator / moderator / admin")
	}
	if perm.Rank(operatorRoles) <= perm.RoleRank(role) {
		return errors.ErrBadRequest("不可编辑与自身同级或更高的角色 (" + role + ")")
	}

	seen := make(map[string]bool, len(items))
	for _, it := range items {
		p := perm.Permission(it.Permission)
		if !perm.IsKnownPermission(p) {
			return errors.ErrBadRequest("未知的权限: " + it.Permission)
		}
		if it.Effect != perm.EffectGrant && it.Effect != perm.EffectRevoke {
			return errors.ErrBadRequest("非法的调整类型: " + it.Effect + " (仅支持 grant / revoke)")
		}
		if seen[it.Permission] {
			return errors.ErrBadRequest("权限 " + it.Permission + " 重复出现")
		}
		seen[it.Permission] = true

		inBaseline := perm.BaselineHas(role, p)
		if it.Effect == perm.EffectGrant && inBaseline {
			return errors.ErrBadRequest("权限 " + it.Permission + " 已在 " + role + " 的默认集合中, 无需授予")
		}
		if it.Effect == perm.EffectRevoke && !inBaseline {
			return errors.ErrBadRequest("权限 " + it.Permission + " 不在 " + role + " 的默认集合中, 无需撤销")
		}
	}

	if key, ok := possessionOffender(
		effectMapFromRoleRows(current, role),
		effectMapFromItems(items),
		operatorEffectiveSet(operatorUID, operatorRoles),
	); !ok {
		return errors.ErrBadRequest("不可增删自己未持有的权限: " + key)
	}

	prospective := rowsToOverrideMap(current)
	prospective[role] = itemsToOverrides(items)
	modEff := perm.EffectiveSet("moderator", prospective["moderator"])
	adminSet := make(map[perm.Permission]bool)
	for _, p := range perm.EffectiveSet("admin", prospective["admin"]) {
		adminSet[p] = true
	}
	var offending []string
	for _, p := range modEff {
		if !adminSet[p] {
			offending = append(offending, string(p))
		}
	}
	if len(offending) > 0 {
		sort.Strings(offending)
		return errors.ErrBadRequest("权限层级冲突: moderator 拥有但 admin 缺失的权限 — " +
			strings.Join(offending, ", ") + " (moderator 必须是 admin 的子集)")
	}
	return nil
}

func buildMatrix(rows []model.RolePermissionOverride) dto.RolePermissionMatrix {
	byRole := make(map[string][]model.RolePermissionOverride)
	for _, r := range rows {
		byRole[r.Role] = append(byRole[r.Role], r)
	}

	roles := make(map[string]dto.RolePermissionRole, len(matrixRoles))
	for _, role := range matrixRoles {
		roleRows := byRole[role]
		if role == "ren" {
			roleRows = nil
		}
		roles[role] = dto.RolePermissionRole{
			Baseline:  permsToStrings(perm.Baseline(role)),
			Overrides: overridesToDTO(roleRows),
			Effective: permsToStrings(perm.EffectiveSet(role, rowsToOverrides(roleRows))),
			Locked:    role == "ren",
		}
	}
	return dto.RolePermissionMatrix{
		Catalog: permsToStrings(perm.Catalog()),
		Roles:   roles,
	}
}

func overridesToDTO(rows []model.RolePermissionOverride) []dto.RolePermissionOverride {
	byPerm := make(map[perm.Permission]model.RolePermissionOverride, len(rows))
	for _, r := range rows {
		byPerm[perm.Permission(r.Permission)] = r
	}
	out := make([]dto.RolePermissionOverride, 0, len(rows))
	for _, p := range perm.Catalog() {
		if r, ok := byPerm[p]; ok {
			out = append(out, dto.RolePermissionOverride{
				Permission: r.Permission,
				Effect:     r.Effect,
				UpdatedBy:  r.UpdatedBy,
				UpdatedAt:  r.UpdatedAt.Format(time.RFC3339),
			})
		}
	}
	return out
}

func rowsToOverrideMap(rows []model.RolePermissionOverride) map[string][]perm.Override {
	out := make(map[string][]perm.Override)
	for _, r := range rows {
		out[r.Role] = append(out[r.Role], perm.Override{
			Permission: perm.Permission(r.Permission),
			Effect:     r.Effect,
		})
	}
	return out
}

func rowsToOverrides(rows []model.RolePermissionOverride) []perm.Override {
	out := make([]perm.Override, 0, len(rows))
	for _, r := range rows {
		out = append(out, perm.Override{Permission: perm.Permission(r.Permission), Effect: r.Effect})
	}
	return out
}

func itemsToOverrides(items []dto.ReplaceOverrideItem) []perm.Override {
	out := make([]perm.Override, 0, len(items))
	for _, it := range items {
		out = append(out, perm.Override{Permission: perm.Permission(it.Permission), Effect: it.Effect})
	}
	return out
}

func itemsToRows(role string, items []dto.ReplaceOverrideItem) []model.RolePermissionOverride {
	out := make([]model.RolePermissionOverride, 0, len(items))
	for _, it := range items {
		out = append(out, model.RolePermissionOverride{
			Role:       role,
			Permission: it.Permission,
			Effect:     it.Effect,
		})
	}
	return out
}

func permsToStrings(perms []perm.Permission) []string {
	out := make([]string, len(perms))
	for i, p := range perms {
		out[i] = string(p)
	}
	return out
}
