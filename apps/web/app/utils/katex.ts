import katex from 'katex'

const MATH_SPAN = /<span class="math (inline|display)">([\s\S]*?)<\/span>/g

const ENTITIES: Record<string, string> = {
  '&amp;': '&',
  '&lt;': '<',
  '&gt;': '>',
  '&quot;': '"',
  '&#34;': '"',
  '&#39;': "'"
}
const unescapeHtml = (s: string): string =>
  s.replace(/&(?:amp|lt|gt|quot|#34|#39);/g, (m) => ENTITIES[m] ?? m)

export const renderKatex = (html: string | null | undefined): string => {
  if (!html || !html.includes('class="math')) {
    return html ?? ''
  }
  return html.replace(MATH_SPAN, (original, kind: string, body: string) => {
    const tex = unescapeHtml(
      body
        .trim()
        .replace(/^\\[([]/, '')
        .replace(/\\[)\]]$/, '')
        .trim()
    )
    if (!tex) {
      return original
    }
    try {
      return katex.renderToString(tex, {
        displayMode: kind === 'display',
        throwOnError: false
      })
    } catch {
      return original
    }
  })
}
