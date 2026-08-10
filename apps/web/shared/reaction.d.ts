interface KunReactorUser {
  id: number
  name: string
  avatar: string
}

interface KunReaction {
  reaction: string
  count: number
  mine: boolean
  reactors?: KunReactorUser[]
}
