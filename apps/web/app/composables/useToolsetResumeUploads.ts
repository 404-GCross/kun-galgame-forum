export const useToolsetResumeUploads = (toolsetId: number) => {
  const key = `kungal:toolset-upload-resume:${toolsetId}`

  const read = (): ToolsetPendingUpload[] => {
    if (!import.meta.client) {
      return []
    }
    try {
      const raw = localStorage.getItem(key)
      const parsed = raw ? JSON.parse(raw) : []
      return Array.isArray(parsed) ? (parsed as ToolsetPendingUpload[]) : []
    } catch {
      return []
    }
  }

  const write = (items: ToolsetPendingUpload[]) => {
    if (!import.meta.client) {
      return
    }
    try {
      localStorage.setItem(key, JSON.stringify(items))
    } catch {
      // Quota exceeded or storage disabled: resume state is an optimisation, so
      // losing it must not fail the upload.
    }
  }

  const list = (): ToolsetPendingUpload[] =>
    read().sort((a, b) => b.updated_at - a.updated_at)

  const upsert = (record: ToolsetPendingUpload) => {
    write([
      ...read().filter((p) => p.artifact_uuid !== record.artifact_uuid),
      record
    ])
  }

  const setProgress = (artifactUuid: string, progress: number) => {
    const all = read()
    const record = all.find((p) => p.artifact_uuid === artifactUuid)
    if (!record) {
      return
    }
    record.progress = progress
    record.updated_at = Date.now()
    write(all)
  }

  const remove = (artifactUuid: string) => {
    write(read().filter((p) => p.artifact_uuid !== artifactUuid))
  }

  return { list, upsert, setProgress, remove }
}
