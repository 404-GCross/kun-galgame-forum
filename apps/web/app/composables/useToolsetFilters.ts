import { useRouteQuery } from '@vueuse/router'

type ToolsetType =
  | 'all'
  | 'emulator'
  | 'translator'
  | 'extractor'
  | 'converter'
  | 'debug'
  | 'launcher'
  | 'script'
  | 'docs'
  | 'others'
type ToolsetLanguage = 'all' | 'ja-jp' | 'en-us' | 'zh-cn' | 'zh-tw' | 'others'
type ToolsetPlatform =
  | 'all'
  | 'windows'
  | 'mac'
  | 'linux'
  | 'emulator'
  | 'others'
type ToolsetVersion = 'all' | 'alpha' | 'beta' | 'rc' | 'stable'
type ToolsetSortField = 'resource_update_time' | 'created' | 'view'

export const useToolsetFilters = () => {
  const opts = { mode: 'replace' as const }

  const page = useRouteQuery('page', 1, { ...opts, transform: Number })
  const type = useRouteQuery<ToolsetType>('type', 'all', opts)
  const language = useRouteQuery<ToolsetLanguage>('language', 'all', opts)
  const platform = useRouteQuery<ToolsetPlatform>('platform', 'all', opts)
  const version = useRouteQuery<ToolsetVersion>('version', 'all', opts)
  const sortField = useRouteQuery<ToolsetSortField>(
    'sortField',
    'resource_update_time',
    opts
  )
  const sortOrder = useRouteQuery<KunOrder>('sortOrder', 'desc', opts)

  const query = useRouteQuery<string>('query', '', opts)

  const limit = 24

  return {
    page,
    limit,
    type,
    language,
    platform,
    version,
    sortField,
    sortOrder,
    query
  }
}
