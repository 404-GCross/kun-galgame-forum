export type KunRolePermEffect = 'grant' | 'revoke'

export interface KunRolePermOverride {
  permission: string
  effect: KunRolePermEffect
  updated_by?: number
  updated_at?: string
}

export interface KunRolePermState {
  baseline: string[]
  overrides: KunRolePermOverride[]
  effective: string[]
  locked?: boolean
}

export interface KunRolePermMatrix {
  catalog: string[]
  roles: Record<string, KunRolePermState>
}

export interface KunPermMine {
  permissions: string[]
}

export interface KunUserPermState {
  user_id: number
  roles: string[]
  role_effective: string[]
  overrides: KunRolePermOverride[]
  effective: string[]
}

export interface KunPermAuditEntry {
  id: number
  operator_id: number
  subject_kind: 'role' | 'user'
  subject: string
  action: 'replace' | 'reset'
  before: KunRolePermOverride[]
  after: KunRolePermOverride[]
  created_at: string
}

export interface KunPermAuditPage {
  total: number
  items: KunPermAuditEntry[]
  users: Record<string, KunUser>
}
