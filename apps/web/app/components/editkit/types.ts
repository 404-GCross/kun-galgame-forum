// editkit — the schema-driven editing component family (infra doc 21 §2.7).
//
// EXTRACTION-READY BOUNDARY (E3a ruling 4): this directory is the source of
// the future @kungal/edit-ui package. Nothing in here may import forum
// business modules — the components consume only (a) these self-contained
// types, (b) KunUI primitives, and (c) props/slots the host page passes in
// (field configs, label maps, image resolvers, user chips). Entity-specific
// knowledge (galgame field labels, option lists, image URL resolution) lives
// on the HOST side and arrives as data.

/** One field of the engine's edit-schema projection: shape + the current
 * user's evaluated capabilities. The UI holds ZERO policy logic. */
export interface EditSchemaField {
  key: string
  kind: string
  diff_hint: string
  deprecated?: boolean
  locked: boolean
  can_propose: boolean
  can_review: boolean
  would_automerge: boolean
}

/** How a field is edited. Derived from kind/diff_hint when the host config
 * does not pin one explicitly. */
export type EditControl =
  | 'input'
  | 'number'
  | 'textarea'
  | 'select'
  | 'switch'
  | 'date'
  | 'string-list'
  | 'number-list'
  | 'link-list'
  | 'image'
  | 'image-list'
  | 'readonly'

export interface EditSelectOption {
  value: string | number
  label: string
}

/** Host-supplied per-field presentation config (labels, controls, option
 * lists, formatting). Functions are fine here — this is a programmatic prop,
 * not serialized data. */
export interface EditFieldConfig {
  label: string
  description?: string
  control?: EditControl
  options?: EditSelectOption[]
  group?: string
  placeholder?: string
  /** number control: empty input maps to null instead of 0. */
  nullable?: boolean
  /** Custom scalar display (diff + readonly rendering). */
  formatValue?: (value: unknown) => string
  /** Custom list-item display (items diff + readonly rendering). */
  formatItem?: (item: unknown) => string
  /** Resolve an image field value / list item to a renderable URL. */
  resolveImage?: (value: unknown) => string
}

export type EditFieldConfigMap = Record<string, EditFieldConfig>

/** One amendment: a maintainer's patch delta (set / unset), seq-ordered. */
export interface EditAmendment {
  id: number
  seq: number
  set?: Record<string, unknown>
  unset?: string[]
  amender_uid: number
  note: string
  created_at: string
}

export type EditProposalStatus = 'open' | 'merged' | 'declined' | 'withdrawn'

export interface EditProposal {
  id: number
  entity_type: string
  entity_id: number
  base_revision_seq: number
  patch: Record<string, unknown>
  effective_patch?: Record<string, unknown>
  proposer_uid: number
  note: string
  site: string
  status: EditProposalStatus
  decided_by_uid?: number
  decided_at?: string
  decision_note?: string
  created_at: string
  updated_at: string
  amendments?: EditAmendment[]
}

export interface EditRevision {
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
  /** Migrated-history provenance (empty on new-era rows). */
  legacy_action?: string
  legacy_note?: string
  legacy_minor?: boolean
}

export interface EditFieldDiffRow {
  key: string
  kind?: string
  diff_hint?: string
  from: unknown
  to: unknown
}

/** Minimal display identity the host resolves for uids. */
export interface EditUser {
  id: number
  name: string
  avatar: string
}
