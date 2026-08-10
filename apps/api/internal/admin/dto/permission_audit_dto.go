package dto

type PermissionAuditPage struct {
	Total int64                          `json:"total"`
	Items []PermissionAuditItem          `json:"items"`
	Users map[string]PermissionAuditUser `json:"users"`
}

type PermissionAuditItem struct {
	ID          int64                `json:"id"`
	OperatorID  int                  `json:"operator_id"`
	SubjectKind string               `json:"subject_kind"`
	Subject     string               `json:"subject"`
	Action      string               `json:"action"`
	Before      []PermissionAuditRow `json:"before"`
	After       []PermissionAuditRow `json:"after"`
	CreatedAt   string               `json:"created_at"`
}

type PermissionAuditRow struct {
	Permission string `json:"permission"`
	Effect     string `json:"effect"`
}

type PermissionAuditUser struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}
