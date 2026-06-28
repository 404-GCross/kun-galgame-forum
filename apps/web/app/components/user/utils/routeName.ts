export interface Nav {
  name: string
  router: string
  redirect?: string
  collapsed?: boolean
  // Visible only when the logged-in viewer is on their OWN profile (id match,
  // never a role). Everything else is public, so the flag is opt-in.
  ownerOnly?: boolean
  child?: Nav[]
}

export const navBarRoute: Ref<Nav[]> = ref([
  {
    name: 'profile',
    router: 'info'
  },
  {
    name: 'setting',
    router: 'setting',
    ownerOnly: true
  },
  // Email + password edits moved to OAuth profile per the 2026-05-23
  // policy (docs/oauth/README.md + 02-user-profile.md §身份层操作 vs
  // 展示操作). The setting page surfaces redirect buttons — no kungal-
  // side route for either.
  {
    name: 'topic',
    redirect: 'topic/topic',
    router: 'topic'
  },
  {
    name: 'galgame',
    redirect: 'galgame/galgame-publish',
    router: 'galgame'
  },
  {
    name: 'rating',
    router: 'rating'
  },
  {
    name: 'resource',
    redirect: 'resource/valid',
    router: 'resource'
  },
  {
    name: 'reply',
    router: 'reply',
    redirect: 'reply/reply-created'
  },
  {
    name: 'comment',
    router: 'comment',
    redirect: 'comment/comment-created'
  }
])
