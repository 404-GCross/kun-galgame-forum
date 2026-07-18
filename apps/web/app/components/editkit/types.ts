// editkit — the schema-driven editing component family (infra doc 21 §2.7).
//
// EXTRACTION-READY BOUNDARY (E3a ruling 4): this directory is the source of
// the future @kungal/edit-ui package. Nothing in here may import forum
// business modules — the components consume only (a) these self-contained
// types, (b) KunUI primitives, and (c) props/slots the host page passes in
// (field configs, label maps, image resolvers, user chips). Entity-specific
// knowledge (galgame field labels, option lists, image URL resolution) lives
// on the HOST side and arrives as data.
import type { Component } from 'vue'

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
  | 'entity-picker'
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
  /** Short label used when this field is one tab of a tabbed group (see
   * SchemaForm `tabbedGroups`, e.g. a language switch for the intro fields).
   * Falls back to `label`. */
  tabLabel?: string
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
  /** Host-supplied upload for image / image-list controls (E3b): uploads
   * the file and resolves to the VALUE to store (image control) or the LIST
   * ITEM to append (image-list) — null when the upload failed (the host
   * surfaces its own error toast). Absent = the field renders read-only. */
  uploadImage?: (file: File, currentItems: unknown[]) => Promise<unknown | null>
  /** image-list: renormalize item-internal order after the list changes
   * (e.g. stamp sort_order = index). Runs on every add/remove/reorder. */
  normalizeItems?: (items: unknown[]) => unknown[]
  /** image-list: the first item is semantically pinned (e.g. the cover set's
   * banner). Renders a badge with this label on item 0 plus a pin-to-front
   * control on the rest. */
  pinFirstLabel?: string
  /** entity-picker: the value is an entity id (single) or id array (multiple),
   * but the user searches + sees NAMES. `multiple` picks the shape. */
  multiple?: boolean
  /** entity-picker: host-supplied async search for a keyword → {value:id,
   * label:name} options (e.g. the site's taxonomy search endpoint). */
  searchEntities?: (keyword: string) => Promise<EditSelectOption[]>
  /** entity-picker: host-supplied batch resolve of the current id value(s) to
   * {value:id, label:name} so pre-existing picks render as names on load. An
   * id it cannot resolve is simply shown as its id. */
  resolveEntities?: (
    ids: (string | number)[]
  ) => Promise<EditSelectOption[]> | EditSelectOption[]
  /** Escape hatch: a host-supplied Vue component that renders this field's
   * control instead of the built-ins. It receives `modelValue` + `disabled`
   * and emits `update:modelValue`. Lets the host inject a rich editor (e.g. an
   * image-free markdown editor for intro fields) without editkit importing it. */
  component?: Component
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
  legacy_id?: number
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
