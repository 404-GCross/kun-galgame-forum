const TOKEN_RE = /^\/image\/([0-9a-f]{64})(?:_([a-z0-9]+))?$/

export default defineEventHandler((event) => {
  if (event.method !== 'GET') return
  const path = event.path.split('?')[0]!
  const m = path.match(TOKEN_RE)
  if (!m) return

  const hash = m[1]!
  const variant = m[2]
  const base = (useRuntimeConfig(event).imageCdnBase || '').replace(/\/+$/, '')
  const file = variant ? `${hash}_${variant}.webp` : `${hash}.webp`
  return sendRedirect(
    event,
    `${base}/${hash.slice(0, 2)}/${hash.slice(2, 4)}/${file}`,
    302
  )
})
