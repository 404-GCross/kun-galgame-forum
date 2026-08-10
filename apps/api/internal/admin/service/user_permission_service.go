package service

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/userclient"
)

type userOverrideStore interface {
	ListForUser(ctx context.Context, userID int) ([]model.UserPermissionOverride, error)
	ReplaceForUser(ctx context.Context, userID int, rows []model.UserPermissionOverride, operatorUID int) error
}

type userLookup interface {
	User(ctx context.Context, id int) (userclient.User, bool, error)
}

type UserPermissionService struct {
	repo       userOverrideStore
	userClient userLookup
	reload     reloader
}

func NewUserPermissionService(repo userOverrideStore, userClient userLookup, reload reloader) *UserPermissionService {
	return &UserPermissionService{repo: repo, userClient: userClient, reload: reload}
}

func (s *UserPermissionService) View(ctx context.Context, uid int) (dto.UserPermissionView, *errors.AppError) {
	if uid <= 0 {
		return dto.UserPermissionView{}, errors.ErrBadRequest("非法的用户 ID")
	}
	roles, appErr := s.targetRoles(ctx, uid)
	if appErr != nil {
		return dto.UserPermissionView{}, appErr
	}
	rows, err := s.repo.ListForUser(ctx, uid)
	if err != nil {
		return dto.UserPermissionView{}, errors.ErrInternal("读取用户权限失败")
	}
	return buildUserView(uid, roles, rows), nil
}

func (s *UserPermissionService) ReplaceOverrides(ctx context.Context, operatorUID int, operatorRoles []string, uid int, items []dto.ReplaceOverrideItem) (dto.UserPermissionView, *errors.AppError) {
	if uid <= 0 {
		return dto.UserPermissionView{}, errors.ErrBadRequest("非法的用户 ID")
	}
	roles, appErr := s.targetRoles(ctx, uid)
	if appErr != nil {
		return dto.UserPermissionView{}, appErr
	}
	currentRows, err := s.repo.ListForUser(ctx, uid)
	if err != nil {
		return dto.UserPermissionView{}, errors.ErrInternal("读取用户权限失败")
	}
	if appErr := validateUserReplace(operatorUID, operatorRoles, roles, items, currentRows); appErr != nil {
		return dto.UserPermissionView{}, appErr
	}

	if err := s.repo.ReplaceForUser(ctx, uid, userItemsToRows(items), operatorUID); err != nil {
		return dto.UserPermissionView{}, errors.ErrInternal("保存用户权限失败")
	}
	if err := s.reload.Load(ctx); err != nil {
		slog.Warn("写入后刷新用户权限覆盖失败, 稍后自动收敛", "error", err)
	}

	rows, err := s.repo.ListForUser(ctx, uid)
	if err != nil {
		return dto.UserPermissionView{}, errors.ErrInternal("读取用户权限失败")
	}
	return buildUserView(uid, roles, rows), nil
}

func (s *UserPermissionService) MyPermissions(uid int, roles []string) dto.MyPermissionsResponse {
	return dto.MyPermissionsResponse{Permissions: permsToStrings(perm.EffectiveForUser(uid, roles))}
}

func (s *UserPermissionService) targetRoles(ctx context.Context, uid int) ([]string, *errors.AppError) {
	u, found, err := s.userClient.User(ctx, uid)
	if err != nil || !found {
		return nil, errors.ErrBadRequest("无法核验目标用户身份, 已中止")
	}
	return u.Roles, nil
}

func validateUserReplace(operatorUID int, operatorRoles, roles []string, items []dto.ReplaceOverrideItem, current []model.UserPermissionOverride) *errors.AppError {
	if hasRen(roles) {
		return errors.ErrBadRequest("ren 持有者恒持全部权限, 不可调整")
	}
	if perm.Rank(operatorRoles) <= perm.Rank(roles) {
		return errors.ErrBadRequest("不可编辑与自身同级或更高的用户")
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

		roleHolds := perm.Can(roles, p)
		if it.Effect == perm.EffectGrant && roleHolds {
			return errors.ErrBadRequest("权限 " + it.Permission + " 已由该用户的角色授予, 无需重复授予")
		}
		if it.Effect == perm.EffectRevoke && !roleHolds {
			return errors.ErrBadRequest("权限 " + it.Permission + " 不在该用户的角色权限内, 无需撤销")
		}
	}

	if key, ok := possessionOffender(
		effectMapFromUserRows(current),
		effectMapFromItems(items),
		operatorEffectiveSet(operatorUID, operatorRoles),
	); !ok {
		return errors.ErrBadRequest("不可增删自己未持有的权限: " + key)
	}
	return nil
}

func hasRen(roles []string) bool {
	return slices.Contains(roles, "ren")
}

func buildUserView(uid int, roles []string, rows []model.UserPermissionOverride) dto.UserPermissionView {
	roleEffective := make([]string, 0, len(perm.Catalog()))
	for _, p := range perm.Catalog() {
		if perm.Can(roles, p) {
			roleEffective = append(roleEffective, string(p))
		}
	}
	effective := permsToStrings(perm.EffectiveForUser(uid, roles))
	return dto.UserPermissionView{
		UserID:        uid,
		Roles:         nonNilStrings(roles),
		RoleEffective: roleEffective,
		Overrides:     userOverridesToDTO(rows),
		Effective:     effective,
	}
}

func userOverridesToDTO(rows []model.UserPermissionOverride) []dto.UserPermissionOverride {
	byPerm := make(map[perm.Permission]model.UserPermissionOverride, len(rows))
	for _, r := range rows {
		byPerm[perm.Permission(r.Permission)] = r
	}
	out := make([]dto.UserPermissionOverride, 0, len(rows))
	for _, p := range perm.Catalog() {
		if r, ok := byPerm[p]; ok {
			out = append(out, dto.UserPermissionOverride{
				Permission: r.Permission,
				Effect:     r.Effect,
				UpdatedBy:  r.UpdatedBy,
				UpdatedAt:  r.UpdatedAt.Format(time.RFC3339),
			})
		}
	}
	return out
}

func userItemsToRows(items []dto.ReplaceOverrideItem) []model.UserPermissionOverride {
	out := make([]model.UserPermissionOverride, 0, len(items))
	for _, it := range items {
		out = append(out, model.UserPermissionOverride{
			Permission: it.Permission,
			Effect:     it.Effect,
		})
	}
	return out
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
