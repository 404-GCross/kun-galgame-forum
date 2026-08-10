export type GalgameImageUploadPreset = 'galgame_banner' | 'galgame_screenshot'

export interface UploadGalgameImageResult {
  hash: string
  url: string
  width: number
  height: number
  size_bytes: number
  variant_urls?: Record<string, string>
  deduplicated: boolean
}

export const uploadGalgameImage = async (
  file: Blob,
  preset: GalgameImageUploadPreset,
  filename = 'image'
): Promise<UploadGalgameImageResult | null> => {
  const form = new FormData()
  form.append('file', file, filename)
  form.append('preset', preset)
  return await kunFetch<UploadGalgameImageResult>('/image/galgame', {
    method: 'POST',
    body: form
  })
}
