import type { InjectionKey, Ref, ComputedRef } from 'vue'

export interface ReactionsState {
  list: Ref<KunReaction[]>
  mineKeys: ComputedRef<string[]>
  toggle: (key: string) => Promise<void>
  topicId?: number
  replyId?: number
}

export const reactionsKey: InjectionKey<ReactionsState> = Symbol('reactions')

interface UseReactionsOptions {
  topicId?: number
  replyId?: number
  targetUserId: number
  reactions: KunReaction[]
  sync?: () => KunReaction[]
  showReactors?: boolean
}

const MAX_AVATARS = 3

export const useReactions = (opts: UseReactionsOptions): ReactionsState => {
  const { id, name, avatar } = usePersistUserStore()

  const clone = (rs: KunReaction[]): KunReaction[] =>
    rs.map((r) => ({
      ...r,
      reactors: r.reactors ? [...r.reactors] : undefined
    }))

  const list = ref<KunReaction[]>(opts.reactions ? clone(opts.reactions) : [])
  const inflight = new Set<string>()

  let userTouched = false
  if (opts.sync) {
    watch(opts.sync, (v) => {
      if (!userTouched) list.value = clone(v)
    })
  }

  const mineKeys = computed(() =>
    list.value.filter((r) => r.mine).map((r) => r.reaction)
  )

  const post = (reaction: string) =>
    opts.topicId
      ? kunFetch(`/topic/${opts.topicId}/reaction`, {
          method: 'PUT',
          body: { reaction }
        })
      : kunFetch(`/topic/0/reply/reaction`, {
          method: 'PUT',
          body: { reply_id: opts.replyId, reaction }
        })

  const removeMine = (idx: number) => {
    const r = list.value[idx]!
    r.count--
    r.mine = false
    if (r.reactors) r.reactors = r.reactors.filter((u) => u.id !== id)
    if (r.count <= 0) list.value.splice(idx, 1)
  }

  const applyOptimistic = (key: string) => {
    const idx = list.value.findIndex((r) => r.reaction === key)
    if (idx >= 0 && list.value[idx]!.mine) {
      removeMine(idx)
    } else if (idx >= 0) {
      const r = list.value[idx]!
      r.count++
      r.mine = true
      if (opts.showReactors && (r.reactors?.length ?? 0) < MAX_AVATARS) {
        ;(r.reactors ??= []).push({ id, name, avatar })
      }
    } else {
      list.value.push({
        reaction: key,
        count: 1,
        mine: true,
        reactors: opts.showReactors ? [{ id, name, avatar }] : undefined
      })
    }
    const opposite =
      key === 'like' ? 'dislike' : key === 'dislike' ? 'like' : null
    if (opposite) {
      const j = list.value.findIndex((r) => r.reaction === opposite)
      if (j >= 0 && list.value[j]!.mine) removeMine(j)
    }
  }

  const toggle = async (key: string) => {
    if (!id) {
      useAuthModal().open()
      return
    }
    if (key === 'like' && id === opts.targetUserId) {
      useMessage(10236, 'warn')
      return
    }
    if (inflight.has(key)) return
    inflight.add(key)
    userTouched = true

    const snapshot = clone(list.value)
    applyOptimistic(key)

    const ok = await post(key)
    if (!ok) list.value = snapshot
    inflight.delete(key)
  }

  return {
    list,
    mineKeys,
    toggle,
    topicId: opts.topicId,
    replyId: opts.replyId
  }
}
