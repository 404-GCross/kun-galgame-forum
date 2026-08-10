import type { Ref } from 'vue'

export interface KnownAccount {
  sub: string
  id: number
  name: string
  avatar: string
  roles: string[]
}

const STORAGE_KEY = 'kun-galgame-known-accounts'
const MAX_ACCOUNTS = 8

let loaded = false

const read = (): KnownAccount[] => {
  if (!import.meta.client) return []
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    const parsed = raw ? JSON.parse(raw) : []
    return Array.isArray(parsed) ? parsed.filter((a) => a && a.sub) : []
  } catch {
    return []
  }
}

const ensureLoaded = (accounts: Ref<KnownAccount[]>) => {
  if (loaded || !import.meta.client) return
  loaded = true
  const seen = new Set(accounts.value.map((a) => a.sub))
  accounts.value = [
    ...accounts.value,
    ...read().filter((a) => !seen.has(a.sub))
  ].slice(0, MAX_ACCOUNTS)
}

export const useKnownAccounts = () => {
  const accounts = useState<KnownAccount[]>('kun-known-accounts', () => [])
  onMounted(() => ensureLoaded(accounts))

  const persist = () => {
    if (!import.meta.client) return
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(accounts.value))
    } catch {
      // Quota exceeded or storage disabled (private mode). Losing the cached
      // account list is harmless; failing the caller is not.
    }
  }

  const upsert = (account: KnownAccount) => {
    if (!account.sub || !account.id) return
    ensureLoaded(accounts)
    accounts.value = [
      account,
      ...accounts.value.filter((a) => a.sub !== account.sub)
    ].slice(0, MAX_ACCOUNTS)
    persist()
  }

  const clearAll = () => {
    accounts.value = []
    loaded = true
    if (!import.meta.client) return
    try {
      localStorage.removeItem(STORAGE_KEY)
    } catch {
      // Storage disabled; the in-memory list is already cleared, which is what
      // the caller asked for.
    }
  }

  const rememberUser = (user: {
    sub?: string
    id?: number
    name?: string
    avatar?: string
    roles?: string[]
  }) => {
    if (!user.sub || !user.id || !user.name) return
    upsert({
      sub: user.sub,
      id: user.id,
      name: user.name,
      avatar: user.avatar || '',
      roles: user.roles ?? []
    })
  }

  return { accounts, upsert, clearAll, rememberUser }
}
