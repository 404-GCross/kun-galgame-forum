import type { ComputedRef } from 'vue'
import { storeToRefs } from 'pinia'

// Frontend mirror of apps/api/pkg/perm — the pure-forum permission bundle.
//
// Two sources of truth, in priority order:
//   1. RUNTIME: /perm/bundles ships the EFFECTIVE bundles (compiled baseline ±
//      admin-set grant/revoke overrides). The perm-bundles plugin fetches them
//      once per app load into useState('kun-perm-bundles'); when present, they
//      are authoritative here.
//   2. FALLBACK: the static table below is the COMPILED BASELINE — used before
//      the fetch resolves and if it fails (offline). It is hand-maintained and
//      MUST stay in lockstep with pkg/perm, byte-for-byte (the string values are
//      the wire contract).
//
// Frontend gating is UX ONLY — the backend (pkg/perm, with the same overrides
// applied) is the real boundary; components branch on a named CAPABILITY here,
// never on a role tier.
//
// Scope: exactly the 43 PURE-FORUM permissions. The 9 INFRA-PROXY permissions
// (galgame edit-proposal review, taxonomy admin, trust moderation inbox,
// galgame submission review, …) deliberately live OUTSIDE this table — their
// source of truth is infra, not this service. Those UI spots keep useRole()
// (or an API-provided capability projection, e.g. `can_review` / `can_decide`)
// with a mirror comment naming the infra permission they stand in for.
//
// Thresholds: `moderator` holds the 41 mod keys; `admin` and `ren` hold all 43
// (the 41 mod keys + the 2 admin-only keys). `user` and `creator` hold NO forum
// permissions. The admin bundle is built FROM the moderator list plus the two
// admin-only keys, so `moderator ⊂ admin` holds by construction.

// The 41 permissions a moderator holds (admin & ren inherit these).
const MODERATOR_PERMISSIONS = [
  'topic.edit_any',
  'topic.hide',
  'topic.set_best_answer',
  'reply.delete_any',
  'reply.pin',
  'comment.topic.delete',
  'comment.galgame.edit',
  'comment.galgame.delete',
  'comment.rating.edit',
  'comment.rating.delete',
  'comment.website.edit',
  'comment.website.delete',
  'comment.toolset.edit',
  'comment.toolset.delete',
  'poll.create_any',
  'poll.edit_any',
  'poll.delete_any',
  'poll.view_restricted',
  'galgame.ban_resource_publish',
  'quiz.edit_any',
  'quiz.delete_any',
  'resource.edit_any',
  'resource.delete_any',
  'rating.delete_any',
  'toolset.edit_any',
  'toolset.delete_any',
  'toolset.resource.edit_any',
  'toolset.resource.delete_any',
  'toolset.upload_bypass',
  'doc.create',
  'doc.edit',
  'doc.delete',
  'website.create',
  'website.edit',
  'website.delete',
  'friend_link.create',
  'friend_link.edit',
  'friend_link.delete',
  'update_log.create',
  'update_log.edit',
  'update_log.delete'
] as const

// The 2 permissions only admin & ren hold, on top of the moderator set.
const ADMIN_ONLY_PERMISSIONS = [
  'admin.dashboard',
  'user.purge_content'
] as const

// admin === ren: the full 43. Built from the moderator list so containment is
// structural, not a copy that could drift.
const ADMIN_PERMISSIONS = [
  ...MODERATOR_PERMISSIONS,
  ...ADMIN_ONLY_PERMISSIONS
] as const

// The string-literal union of every pure-forum permission key.
export type ForumPermission =
  | (typeof MODERATOR_PERMISSIONS)[number]
  | (typeof ADMIN_ONLY_PERMISSIONS)[number]

// Role → granted-permission bundles. Keyed by the raw OAuth role claim; roles
// not present here (`user`, `creator`) grant nothing. `has()` lookups keep
// useCan O(1) regardless of table size.
const ROLE_PERMISSIONS: Record<string, ReadonlySet<ForumPermission>> = {
  moderator: new Set(MODERATOR_PERMISSIONS),
  admin: new Set(ADMIN_PERMISSIONS),
  ren: new Set(ADMIN_PERMISSIONS)
}

// Reactive, UX-only capability check. Reads the same role set useRole() reads
// (usePersistUserStore, via storeToRefs so it stays reactive to role changes),
// and reacts to the fetched runtime bundles landing in useState.
export const useCan = (permission: ForumPermission): ComputedRef<boolean> => {
  const { roles } = storeToRefs(usePersistUserStore())
  // The perm-bundles plugin populates this; null until fetched / on failure.
  const bundles = useState<Record<string, string[]> | null>(
    'kun-perm-bundles',
    () => null
  )

  return computed(() => {
    const effective = bundles.value
    if (effective) {
      // Runtime truth: the effective bundle already folds in admin overrides,
      // so a plain membership test suffices (roles not in the map grant nothing,
      // including `creator`, which the static fallback doesn't even list).
      return roles.value.some(
        (role) => effective[role]?.includes(permission) ?? false
      )
    }
    // Compiled-baseline fallback (pre-fetch / offline).
    return roles.value.some(
      (role) => ROLE_PERMISSIONS[role]?.has(permission) ?? false
    )
  })
}
