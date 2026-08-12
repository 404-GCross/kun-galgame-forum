package dto

type RolePermissionMatrix struct {
	Catalog []string                      `json:"catalog"`
	Roles   map[string]RolePermissionRole `json:"roles"`
}

type RolePermissionRole struct {
	Baseline  []string                 `json:"baseline"`
	Overrides []RolePermissionOverride `json:"overrides"`
	Effective []string                 `json:"effective"`
	Locked    bool                     `json:"locked,omitempty"`
}

type RolePermissionOverride struct {
	Permission string `json:"permission"`
	Effect     string `json:"effect"`
	UpdatedBy  int    `json:"updated_by"`
	UpdatedAt  string `json:"updated_at"`
}

type ReplaceOverridesRequest struct {
	Overrides []ReplaceOverrideItem `json:"overrides"`
}

type ReplaceOverrideItem struct {
	Permission string `json:"permission"`
	Effect     string `json:"effect"`
}
