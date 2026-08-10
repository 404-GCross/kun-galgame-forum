// @vitest-environment nuxt
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import { uploadGalgameImage } from './uploadGalgameImage'

const { fetchSpy } = vi.hoisted(() => ({ fetchSpy: vi.fn() }))
mockNuxtImport('kunFetch', () => fetchSpy)

beforeEach(() => {
  fetchSpy.mockReset()
})

describe('uploadGalgameImage', () => {
  it('POSTs to /image/galgame with multipart FormData carrying file + preset', async () => {
    fetchSpy.mockResolvedValueOnce({
      hash: 'abcd1234567890ab',
      url: 'https://cdn/ab/cd/abcd1234567890ab.webp',
      width: 1920,
      height: 1080,
      size_bytes: 12345,
      deduplicated: false
    })

    const blob = new Blob(['fake-bytes'], { type: 'image/png' })
    const res = await uploadGalgameImage(blob, 'galgame_banner', 'shot.png')

    expect(res?.hash).toBe('abcd1234567890ab')
    expect(res?.url).toContain('.webp')

    expect(fetchSpy).toHaveBeenCalledTimes(1)
    const [path, opts] = fetchSpy.mock.calls[0]!
    expect(path).toBe('/image/galgame')
    expect(opts.method).toBe('POST')

    const body = opts.body as FormData
    expect(body).toBeInstanceOf(FormData)
    expect(body.get('preset')).toBe('galgame_banner')
    const filePart = body.get('file')
    expect(filePart).toBeInstanceOf(Blob)
  })

  it('returns null when kunFetch surfaces a business error (returns null)', async () => {
    fetchSpy.mockResolvedValueOnce(null)
    const blob = new Blob([''], { type: 'image/png' })
    const res = await uploadGalgameImage(blob, 'galgame_banner')
    expect(res).toBeNull()
  })

  it('hits the same endpoint with default filename omitted', async () => {
    fetchSpy.mockResolvedValueOnce({
      hash: 'aaaa',
      url: 'x',
      width: 0,
      height: 0,
      size_bytes: 0,
      deduplicated: false
    })
    const blob = new Blob([''], { type: 'image/png' })
    await uploadGalgameImage(blob, 'galgame_banner')
    const body = fetchSpy.mock.calls[0]![1].body as FormData
    expect(body.get('preset')).toBe('galgame_banner')
    expect(body.get('file')).toBeInstanceOf(Blob)
  })
})
