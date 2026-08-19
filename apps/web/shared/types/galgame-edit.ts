export type GalgameEditProposalStatus =
  | 'open'
  | 'merged'
  | 'declined'
  | 'withdrawn'

export interface GalgameEditSchemaField {
  key: string
  kind: string
  diff_hint: string
  deprecated?: boolean
  locked: boolean
  can_propose: boolean
  can_review: boolean
  would_automerge: boolean
}

export interface GalgameEditAmendment {
  id: number
  seq: number
  set?: Record<string, unknown>
  unset?: string[]
  amender_uid: number
  note: string
  created_at: string
}

export interface GalgameEditProposal {
  id: number
  entity_type: string
  entity_id: number
  base_revision_seq: number
  patch: Record<string, unknown>
  effective_patch?: Record<string, unknown>
  proposer_uid: number
  note: string
  site: string
  status: GalgameEditProposalStatus
  decided_by_uid?: number
  decided_at?: string
  decision_note?: string
  created_at: string
  updated_at: string
  amendments?: GalgameEditAmendment[]
}

export interface GalgameEditRevision {
  id: number
  seq: number
  action: string
  changed_fields: string[]
  snapshot: Record<string, unknown>
  actor_uid: number
  amender_uid?: number
  proposal_id?: number
  site: string
  created_at: string
  legacy_action?: string
  legacy_note?: string
  legacy_minor?: boolean
  legacy_id?: number
}

export interface GalgameEditUser {
  id: number
  name: string
  avatar: string
}

export interface GalgameEditGameBrief {
  id: number
  vndb_id: string
  name_en_us: string
  name_ja_jp: string
  name_zh_cn: string
  name_zh_tw: string
  status: number
  content_limit: string
}

export interface GalgameEditProposalItem extends GalgameEditProposal {
  gid: number
  galgame?: GalgameEditGameBrief
}

export interface GalgameEditBootstrap {
  gid: number
  values: Record<string, unknown>
  fields: GalgameEditSchemaField[]
  can_review: boolean
}

export interface GalgameEditSubmitResult {
  merged: boolean
  proposal: GalgameEditProposal
  revision?: GalgameEditRevision
}

export interface GalgameEditProposalList {
  items: GalgameEditProposalItem[]
  users: Record<number, GalgameEditUser>
}

export interface GalgameEditProposalDetail {
  proposal: GalgameEditProposalItem
  values: Record<string, unknown>
  fields: GalgameEditSchemaField[]
  users: Record<number, GalgameEditUser>
  can_decide: boolean
}

export interface GalgameEditRevisionList {
  gid: number
  items: GalgameEditRevision[]
  users: Record<number, GalgameEditUser>
  can_revert?: boolean
}

export interface GalgameEditRevertResult {
  proposal: GalgameEditProposal
  revision: GalgameEditRevision
}

export interface GalgameEditFieldDiff {
  key: string
  kind?: string
  diff_hint?: string
  from: unknown
  to: unknown
}

export interface GalgameEditDiff {
  from_seq: number
  to_seq: number
  fields: GalgameEditFieldDiff[]
}
