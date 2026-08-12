import type { ToolsetComment } from './toolset-comment'

export interface ToolsetCard {
  id: number
  name: string
  user: KunUser
  type: string
  platform: string
  language: string
  version: string
  view: number
  download: number
  comment_count: number
  practicality_avg: number | null
  resource_update_time: Date | string
}

export interface ToolsetDetail {
  id: number
  name: string
  content_html: string
  content_markdown: string
  type: string
  platform: string
  language: string
  version: string
  homepage: string[]
  download: number
  view: number
  user: KunUser
  aliases: string[]
  practicality_avg: number | null
  practicality_count: number
  resource_update_time: Date | string
  resource: ToolsetResource[]
  edited: Date | string | null
  created: Date | string
  updated: Date | string
  rating_counts: Record<number, number>
  comment_count: number
  comment_preview: ToolsetComment[]
  contributors: KunUser[]
}

export interface ToolsetRating {
  counts: {
    [x: number]: number
  }
  avg: number
  mine: number | null
}

export interface ToolsetUploadInitResponse {
  artifact_uuid: string
  multipart: boolean
  upload_url?: string
  part_size?: number
  parts?: {
    part_number: number
    url: string
  }[]
  expires_at: string
}

export interface ToolsetUploadCompleteResponse {
  artifact_uuid: string
  size: number
}

export interface ToolsetUploadResumeResponse {
  artifact_uuid: string
  multipart: boolean
  upload_url?: string
  part_size?: number
  parts?: {
    part_number: number
    url: string
  }[]
  uploaded_parts?: {
    part_number: number
    etag: string
    size: number
  }[]
  expires_at: string
}

export interface ToolsetUploadResult {
  artifact_uuid: string
  size: number
}

export interface ToolsetPendingUpload {
  artifact_uuid: string
  name: string
  size: number
  last_modified: number
  progress: number
  updated_at: number
}

export interface ToolsetResource {
  id: number
  type: string
  size: string
  download: number
  status: number
}

export interface ToolsetResourceDetail extends ToolsetResource {
  user: KunUser
  content: string
  code: string
  note: string
  password: string
  edited: Date | string | null
  created: Date | string
  updated: Date | string
}
