package dto

type UserPermissionView struct {
	UserID        int                      `json:"user_id"`
	Roles         []string                 `json:"roles"`
	RoleEffective []string                 `json:"role_effective"`
	Overrides     []UserPermissionOverride `json:"overrides"`
	Effective     []string                 `json:"effective"`
}

type UserPermissionOverride struct {
	Permission string `json:"permission"`
	Effect     string `json:"effect"`
	UpdatedBy  int    `json:"updated_by"`
	UpdatedAt  string `json:"updated_at"`
}

type ReplaceUserOverridesRequest struct {
	Overrides []ReplaceOverrideItem `json:"overrides"`
}

type MyPermissionsResponse struct {
	Permissions []string `json:"permissions"`
}
