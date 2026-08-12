import { buildSitemapUrls } from '../../utils/kunSitemapSources'

export default defineCachedEventHandler(
  async () => {
    const apiBase = useRuntimeConfig().apiBaseUrl
    return await buildSitemapUrls(apiBase)
  },
  {
    name: 'sitemap-urls',
    getKey: () => 'all',
    swr: true,
    maxAge: 60 * 60 * 6,
    staleMaxAge: 60 * 60 * 24 * 7
  }
)
