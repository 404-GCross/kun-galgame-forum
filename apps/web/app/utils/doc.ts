const WORDS_PER_MINUTE = 300

const sanitizeMarkdown = (markdown: string) => {
  if (!markdown) {
    return ''
  }
  return markdown
    .replace(/`{3}[\s\S]*?`{3}/g, ' ')
    .replace(/`[^`]+`/g, ' ')
    .replace(/!\[.*?\]\(.*?\)/g, ' ')
    .replace(/\[[^\]]*?\]\(.*?\)/g, ' ')
    .replace(/[^\w\s\u4e00-\u9fa5-]/g, ' ')
}

export const computeReadingMinute = (markdown: string) => {
  const content = sanitizeMarkdown(markdown)
  if (!content.trim()) {
    return 0
  }

  const tokens = content.replace(/\s+/g, ' ').trim().split(' ').filter(Boolean)

  if (!tokens.length) {
    return 0
  }

  return Math.max(1, Math.ceil(tokens.length / WORDS_PER_MINUTE))
}

export const normalizeDocSlug = (value: string) =>
  value
    .toString()
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9-]/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
