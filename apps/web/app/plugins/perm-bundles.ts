// Fetch the EFFECTIVE role→permission bundles (compiled baseline ± admin-set
// overrides) once per app load, into useState('kun-perm-bundles'). useCan reads
// them as its runtime truth, falling back to the static compiled table until /
// unless this resolves.
//
// Universal (not .client) on purpose: fetching during SSR populates the state
// before first paint and transfers it via the Nuxt payload, so a role-holder's
// SSR and hydrated renders agree — no capability flash, no hydration mismatch.
// The guard below means the client re-run is a no-op once the payload is in.
//
// Skipped entirely for anonymous / plain users: with roles = [] every capability
// resolves to false whether we fetch or not, so the request would be pure waste.
export default defineNuxtPlugin(async () => {
  const bundles = useState<Record<string, string[]> | null>(
    'kun-perm-bundles',
    () => null
  )

  // Already populated (SSR payload transferred to the client) — don't refetch.
  if (bundles.value) return

  // Roles come from the persisted store (cookie-backed, so available on SSR too).
  const { roles } = usePersistUserStore()
  if (!roles.length) return

  // Public endpoint; failure leaves the state null (useCan falls back silently).
  const data = await kunFetch<Record<string, string[]>>('/perm/bundles')
  if (data) {
    bundles.value = data
  }
})
