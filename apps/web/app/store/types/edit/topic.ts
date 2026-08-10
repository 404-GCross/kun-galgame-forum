export interface EditStorePersist {
  mode: 'preview' | 'code'

  title: string
  content: string
  category: string
  section: string[]
  isNSFW: boolean
  coverImages: string[]
}

export interface EditStoreTemp {
  id: number
  title: string
  content: string
  category: string
  section: string[]
  isNSFW: boolean
  coverImages: string[]

  isTopicRewriting: boolean
}
