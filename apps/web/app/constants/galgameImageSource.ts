export const GALGAME_IMAGE_SOURCE_LABEL: Record<string, string> = {
  vndb: 'VNDB',
  curated: '本站收录',
  user: '用户上传',
  dlsite: 'DLsite',
  getchu: 'Getchu',
  bangumi: 'Bangumi',
  steam: 'Steam',
  upscale: 'AI 超分',
  '': '未知来源'
}

const ORDER = Object.keys(GALGAME_IMAGE_SOURCE_LABEL)

export const galgameImageSourceLabel = (source: string | undefined): string =>
  GALGAME_IMAGE_SOURCE_LABEL[source ?? ''] ?? source ?? '未知来源'

export const galgameImageSourceRank = (source: string | undefined): number => {
  const index = ORDER.indexOf(source ?? '')
  return index === -1 ? ORDER.length : index
}
