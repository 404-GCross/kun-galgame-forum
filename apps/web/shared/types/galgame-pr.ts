export interface GalgamePR {
  id: number
  galgame_id: number
  status: number
  title: string
  message: string
  base_revision: number
  user: KunUser
  completed_time: Date | string | null
  created: Date | string
}

export interface NextMoeSnapshotNames {
  tags?: Record<string, string>
  officials?: Record<string, string>
  engines?: Record<string, string>
  series?: Record<string, string>
}

export interface NextMoePRDetailResponse {
  pr: {
    id: number
    galgame_id: number
    user_id: number
    status: number
    title: string
    message: string
    base_revision: number
    snapshot: Record<string, unknown>
    completed_by: number | null
    revision_id: number | null
    created: string
  }
  changed_keys: Record<string, boolean>
  names?: NextMoeSnapshotNames
}

export interface GalgamePRDiffView {
  id: number
  galgame_id: number
  status: number
  changed_keys: Record<string, boolean>
  old_snap: Record<string, unknown>
  new_snap: Record<string, unknown>
  names?: NextMoeSnapshotNames
}
