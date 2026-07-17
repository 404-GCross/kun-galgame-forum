// Pure helpers for the editkit family. No forum imports (extraction-ready
// boundary — see types.ts).

import type { EditControl, EditFieldConfig, EditSchemaField } from './types'

/** Deterministic stringify (sorted object keys) so value identity survives
 * JSON round trips regardless of key order. */
export const stableStringify = (value: unknown): string => {
  if (value === null || value === undefined) {
    return 'null'
  }
  if (Array.isArray(value)) {
    return `[${value.map(stableStringify).join(',')}]`
  }
  if (typeof value === 'object') {
    const entries = Object.entries(value as Record<string, unknown>)
      .filter(([, v]) => v !== undefined)
      .sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0))
      .map(([k, v]) => `${JSON.stringify(k)}:${stableStringify(v)}`)
    return `{${entries.join(',')}}`
  }
  return JSON.stringify(value)
}

export const editValueEqual = (a: unknown, b: unknown): boolean =>
  stableStringify(a) === stableStringify(b)

/** Resolve the control a field renders with: host config wins, then the
 * schema's kind/diff_hint defaults. */
export const resolveControl = (
  field: EditSchemaField,
  config?: EditFieldConfig
): EditControl => {
  if (config?.control) {
    return config.control
  }
  switch (field.kind) {
    case 'text':
      return field.diff_hint === 'lines' ? 'textarea' : 'input'
    case 'enum':
      return config?.options?.length ? 'select' : 'input'
    case 'int':
    case 'ref':
      return 'number'
    case 'date':
      return 'date'
    case 'imagehash':
      return 'image'
    case 'list':
      return field.diff_hint === 'image' ? 'image-list' : 'string-list'
    default:
      return 'readonly'
  }
}

/** Human display for a scalar value (readonly rendering + inline diff). */
export const formatEditValue = (
  value: unknown,
  config?: EditFieldConfig
): string => {
  if (config?.formatValue) {
    return config.formatValue(value)
  }
  if (value === null || value === undefined || value === '') {
    return '（空）'
  }
  if (typeof value === 'boolean') {
    return value ? '是' : '否'
  }
  if (config?.options) {
    const hit = config.options.find((o) => o.value === value)
    if (hit) {
      return hit.label
    }
  }
  if (Array.isArray(value)) {
    return value.map((v) => formatEditItem(v, config)).join('、') || '（空）'
  }
  if (typeof value === 'object') {
    return JSON.stringify(value)
  }
  return String(value)
}

/** Human display for one list item. */
export const formatEditItem = (
  item: unknown,
  config?: EditFieldConfig
): string => {
  if (config?.formatItem) {
    return config.formatItem(item)
  }
  if (item === null || item === undefined) {
    return '（空）'
  }
  if (typeof item === 'object') {
    const o = item as Record<string, unknown>
    if (typeof o.name === 'string' && typeof o.link === 'string') {
      return `${o.name} → ${o.link}`
    }
    return JSON.stringify(item)
  }
  return String(item)
}

export interface DiffLine {
  type: 'same' | 'add' | 'del'
  text: string
}

/** Line-level LCS diff for text fields (diff_hint=lines). Both sides are
 * small wiki intros — the O(n·m) table is fine. */
export const diffLines = (from: string, to: string): DiffLine[] => {
  const a = from.length ? from.split('\n') : []
  const b = to.length ? to.split('\n') : []
  const n = a.length
  const m = b.length
  const lcs: number[][] = Array.from({ length: n + 1 }, () =>
    new Array<number>(m + 1).fill(0)
  )
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      lcs[i]![j] =
        a[i] === b[j]
          ? lcs[i + 1]![j + 1]! + 1
          : Math.max(lcs[i + 1]![j]!, lcs[i]![j + 1]!)
    }
  }
  const out: DiffLine[] = []
  let i = 0
  let j = 0
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      out.push({ type: 'same', text: a[i]! })
      i++
      j++
    } else if (lcs[i + 1]![j]! >= lcs[i]![j + 1]!) {
      out.push({ type: 'del', text: a[i]! })
      i++
    } else {
      out.push({ type: 'add', text: b[j]! })
      j++
    }
  }
  for (; i < n; i++) {
    out.push({ type: 'del', text: a[i]! })
  }
  for (; j < m; j++) {
    out.push({ type: 'add', text: b[j]! })
  }
  return out
}

export interface ItemsDiff {
  added: unknown[]
  removed: unknown[]
  kept: unknown[]
}

/** Item-level diff for list fields (diff_hint=items/image): identity is the
 * stable stringification. */
export const diffItems = (from: unknown, to: unknown): ItemsDiff => {
  const a = Array.isArray(from) ? from : []
  const b = Array.isArray(to) ? to : []
  const aKeys = new Set(a.map(stableStringify))
  const bKeys = new Set(b.map(stableStringify))
  return {
    added: b.filter((x) => !aKeys.has(stableStringify(x))),
    removed: a.filter((x) => !bKeys.has(stableStringify(x))),
    kept: a.filter((x) => bKeys.has(stableStringify(x)))
  }
}

/** Status → KunUI chip color + label. */
export const proposalStatusBadge = (
  status: string
): { label: string; color: 'primary' | 'success' | 'danger' | 'default' } => {
  switch (status) {
    case 'open':
      return { label: '待审核', color: 'primary' }
    case 'merged':
      return { label: '已合并', color: 'success' }
    case 'declined':
      return { label: '已拒绝', color: 'danger' }
    case 'withdrawn':
      return { label: '已撤回', color: 'default' }
    default:
      return { label: status, color: 'default' }
  }
}

/** Engine action → badge label + color. Hosts may override labels via the
 * RevisionTimeline prop. */
export const revisionActionBadge = (
  action: string
): { label: string; color: 'primary' | 'success' | 'warning' | 'default' } => {
  switch (action) {
    case 'created':
      return { label: '创建', color: 'default' }
    case 'merged':
      return { label: '合并提案', color: 'success' }
    case 'direct':
      return { label: '直接编辑', color: 'primary' }
    case 'reverted':
      return { label: '回滚', color: 'warning' }
    default:
      return { label: action, color: 'default' }
  }
}
