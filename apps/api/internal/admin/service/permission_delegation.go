package service

import (
	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/pkg/perm"
)

func operatorEffectiveSet(operatorUID int, operatorRoles []string) map[perm.Permission]bool {
	set := make(map[perm.Permission]bool)
	for _, p := range perm.EffectiveForUser(operatorUID, operatorRoles) {
		set[p] = true
	}
	return set
}

func possessionOffender(stored, submitted map[string]string, holds map[perm.Permission]bool) (string, bool) {
	for _, p := range perm.Catalog() {
		key := string(p)
		if stored[key] == submitted[key] {
			continue
		}
		if !holds[p] {
			return key, false
		}
	}
	return "", true
}

func effectMapFromItems(items []dto.ReplaceOverrideItem) map[string]string {
	m := make(map[string]string, len(items))
	for _, it := range items {
		m[it.Permission] = it.Effect
	}
	return m
}

func effectMapFromRoleRows(rows []model.RolePermissionOverride, role string) map[string]string {
	m := make(map[string]string)
	for _, r := range rows {
		if r.Role == role {
			m[r.Permission] = r.Effect
		}
	}
	return m
}

func effectMapFromUserRows(rows []model.UserPermissionOverride) map[string]string {
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.Permission] = r.Effect
	}
	return m
}
