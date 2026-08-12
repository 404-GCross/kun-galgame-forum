import { fetchKunApi, useKunFeed } from '../../utils/kunFeed'

interface TopicRSSItem {
  id: number
  title: string
  description: string
  user_id: number
  user_name: string
  created: string
}

export default defineCachedEventHandler(
  async (event): Promise<string> => {
    const baseUrl = useRuntimeConfig().public.KUN_GALGAME_URL || ''
    const feed = useKunFeed(baseUrl, 'topic')

    const topics = await fetchKunApi<TopicRSSItem[]>('/rss/topic')
    for (const t of topics) {
      feed.addItem({
        link: `${baseUrl}/topic/${t.id}`,
        title: t.title,
        date: new Date(t.created),
        description: t.description,
        author: [
          {
            name: t.user_name,
            link: `${baseUrl}/user/${t.user_id}/info`
          }
        ]
      })
    }

    setHeader(event, 'Content-Type', 'application/xml')
    return feed.rss2()
  },
  {
    name: 'rss-topic',
    getKey: () => 'all',
    swr: true,
    maxAge: 60 * 5,
    staleMaxAge: 60 * 60 * 24 * 7
  }
)
