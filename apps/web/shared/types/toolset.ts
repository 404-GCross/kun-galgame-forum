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
  commentCount: number
  practicalityAvg: number | null
  resource_update_time: Date | string
}

export interface ToolsetDetail {
  id: number
  name: string
  contentHtml: string
  contentMarkdown: string
  type: string
  platform: string
  language: string
  version: string
  homepage: string[]
  download: number
  view: number
  user: KunUser
  aliases: string[]
  practicalityAvg: number | null
  practicalityCount: number
  resource_update_time: Date | string
  resource: ToolsetResource[]
  edited: Date | string | null
  created: Date | string
  updated: Date | string
  ratingCounts: Record<number, number>
  commentCount: number
  commentPreview: ToolsetComment[]
  contributors: KunUser[]
}

export interface ToolsetRating {
  counts: {
    [x: number]: number
  }
  avg: number
  // BE returns `null` when the caller hasn't rated yet (PracticalityResponse.Mine *int).
  mine: number | null
}

// Server-driven init response from the artifact service (via kungal's BFF).
// When multipart is false the browser does one PUT to uploadUrl; otherwise it
// slices by partSize and PUTs each part to parts[i].url, collecting ETags.
export interface ToolsetUploadInitResponse {
  artifactUuid: string
  multipart: boolean
  uploadUrl?: string
  partSize?: number
  parts?: {
    partNumber: number
    url: string
  }[]
  expiresAt: string
}

export interface ToolsetUploadCompleteResponse {
  artifactUuid: string
  size: number
}

// Resume an interrupted multipart upload: uploadedParts are already stored in B2
// (skip them, reuse the etag at complete), parts are fresh presigned URLs for
// only the missing parts. Same shape as init so the multipart PUT loop is shared;
// a single-part upload comes back multipart=false + a fresh uploadUrl.
export interface ToolsetUploadResumeResponse {
  artifactUuid: string
  multipart: boolean
  uploadUrl?: string
  partSize?: number
  parts?: {
    partNumber: number
    url: string
  }[]
  uploadedParts?: {
    partNumber: number
    etag: string
    size: number
  }[]
  expiresAt: string
}

// Result emitted from the S3 upload widget once a full upload (init → PUT →
// complete) succeeds. The artifact uuid binds the upload to a toolset_resource
// row at create time; size pre-fills the file size input.
export interface ToolsetUploadResult {
  artifactUuid: string
  size: number
}

// A multipart upload started but not completed, persisted in localStorage per
// toolset so the upload modal can surface "you have unfinished uploads" across
// page reloads. The browser can't read a file by path, so resuming needs the
// user to re-select it (matched by size+lastModified, so a moved/renamed file
// still resumes); the uploaded parts live in B2 on the artifact side. name is
// display-only. progress is the last-known % (updated
// as parts upload + on interruption) so the resume list can show how far it got.
export interface ToolsetPendingUpload {
  artifactUuid: string
  name: string
  size: number
  lastModified: number
  progress: number
  updatedAt: number
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
