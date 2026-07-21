// Wire types for the permission-first authorization admin surface.
//
// The runtime truth is a DELTA model: a role's effective bundle is its compiled
// code baseline, adjusted by admin-set OVERRIDE rows (grant / revoke) persisted
// in Postgres. An override row exists ONLY where the role deviates from its
// baseline, so `effective = (baseline ∪ grants) − revokes`.
//
// `permission` / `catalog` values are the pure-forum ForumPermission keys (see
// app/composables/useCan.ts); they are typed as plain strings here because this
// is untransformed wire data — the constants layer is where the ForumPermission
// vocabulary is enforced.

export type KunRolePermEffect = 'grant' | 'revoke'

// A single deviation from the compiled baseline for one role.
export interface KunRolePermOverride {
  permission: string
  effect: KunRolePermEffect
  updated_by?: number
  updated_at?: string
}

// One role's row in the matrix. `locked` marks the `ren` (站长) safety anchor:
// it always holds the full catalogue and is never adjustable.
export interface KunRolePermState {
  baseline: string[]
  overrides: KunRolePermOverride[]
  effective: string[]
  locked?: boolean
}

// GET /admin/role-permissions and every PUT /admin/role-permissions/:role
// response share this shape. `roles` is keyed by the raw role claim
// (creator / moderator / admin / ren); `user` never appears (implicit claim).
export interface KunRolePermMatrix {
  catalog: string[]
  roles: Record<string, KunRolePermState>
}

// GET /perm/bundles (public): the EFFECTIVE bundles (baseline ± overrides),
// consumed by the useCan plugin. snake-less simple shape.
export type KunPermBundles = Record<string, string[]>
