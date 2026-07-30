// @vitest-environment nuxt
//
// SnapshotDiff structural-diff component spec. The Nuxt environment is
// needed because the component reaches for auto-imported symbols at render
// time (KunChip / KunNull / EditkitTextDiff); happy-dom alone would resolve
// them to undefined and the test would assert against an empty HTML shell.
import { describe, it, expect } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import SnapshotDiff from './SnapshotDiff.vue'

describe('SnapshotDiff', () => {
  it('renders KunNull when changedKeys is empty', async () => {
    const w = await mountSuspended(SnapshotDiff, {
      props: { changedKeys: {}, oldSnap: {}, newSnap: {} }
    })
    expect(w.html()).toContain('无字段变化')
  })

  it('scalar diff: one unified block tinting only the changed run', async () => {
    const w = await mountSuspended(SnapshotDiff, {
      props: {
        changedKeys: { name_zh_cn: true },
        oldSnap: { name_zh_cn: '碧蓝航线' },
        newSnap: { name_zh_cn: '碧蓝幻想' }
      }
    })
    const html = w.html()
    expect(html).toContain('简体中文标题')
    // Deletions render as <del>, insertions as <ins> — the unified form.
    expect(html).toContain('<del')
    expect(html).toContain('<ins')
    // The shared 碧蓝 run is emitted ONCE and untinted. Printing the whole
    // field twice, one side red and one green, is what the 2-column view this
    // replaced did — and why a small edit was invisible in it.
    expect(w.text().match(/碧蓝/g)?.length).toBe(1)
  })

  it('array-of-scalars: tag_ids renders +/- badges, not raw JSON', async () => {
    const w = await mountSuspended(SnapshotDiff, {
      props: {
        changedKeys: { tag_ids: true },
        oldSnap: { tag_ids: [1, 2, 3] },
        newSnap: { tag_ids: [2, 3, 4] }
      }
    })
    const html = w.html()
    // Added items render as "+ <value>" chips, removed as "- <value>".
    expect(html).toContain('+ 4')
    expect(html).toContain('- 1')
    // Critically: must NOT fall back to JSON-LCS garble.
    expect(html).not.toContain('[1,2,3]')
  })

  it('array-of-objects (covers): added / removed / changed split', async () => {
    const w = await mountSuspended(SnapshotDiff, {
      props: {
        changedKeys: { covers: true },
        oldSnap: {
          covers: [
            { image_hash: 'aaaa', sort_order: 0, sexual: 0, violence: 0 },
            { image_hash: 'bbbb', sort_order: 1, sexual: 0, violence: 0 }
          ]
        },
        newSnap: {
          covers: [
            // aaaa changed sort_order
            { image_hash: 'aaaa', sort_order: 5, sexual: 0, violence: 0 },
            // bbbb removed; cccc added
            { image_hash: 'cccc', sort_order: 1, sexual: 0, violence: 0 }
          ]
        }
      }
    })
    const html = w.html()
    expect(html).toContain('新增')
    expect(html).toContain('cccc')
    expect(html).toContain('删除')
    expect(html).toContain('bbbb')
    expect(html).toContain('修改')
    expect(html).toContain('sort_order')
  })

  it('array-of-objects (links): composite key — same (name,link) not a change', async () => {
    const w = await mountSuspended(SnapshotDiff, {
      props: {
        changedKeys: { links: true },
        oldSnap: {
          links: [
            { name: 'a', link: 'x' },
            { name: 'b', link: 'y' }
          ]
        },
        newSnap: {
          links: [
            { name: 'b', link: 'y' }, // reorder only
            { name: 'a', link: 'x' }
          ]
        }
      }
    })
    // Pure reorder produces no add/remove/changed → row hidden →
    // KunNull placeholder kicks in.
    expect(w.html()).toContain('无字段变化')
  })

  it('falls back gracefully on an unknown array-of-scalars field', async () => {
    const w = await mountSuspended(SnapshotDiff, {
      props: {
        changedKeys: { some_future_ids: true },
        oldSnap: { some_future_ids: [10, 20] },
        newSnap: { some_future_ids: [20, 30] }
      }
    })
    expect(w.html()).toContain('+ 30')
  })

  it('names dict resolves tag_ids to display names', async () => {
    const w = await mountSuspended(SnapshotDiff, {
      props: {
        changedKeys: { tag_ids: true },
        oldSnap: { tag_ids: [1, 2] },
        newSnap: { tag_ids: [2, 3] },
        names: {
          tags: { '1': '校园', '2': '治愈', '3': 'RPG' }
        }
      }
    })
    const html = w.html()
    // Removed chip should show the resolved name, not the raw id.
    expect(html).toContain('校园')
    // Added chip likewise.
    expect(html).toContain('RPG')
  })

  it('missing key in names dict ⇒ "已删除 #<id>" fallback', async () => {
    const w = await mountSuspended(SnapshotDiff, {
      props: {
        changedKeys: { tag_ids: true },
        oldSnap: { tag_ids: [99] },
        newSnap: { tag_ids: [] },
        // 99 intentionally missing — entity was deleted.
        names: { tags: {} }
      }
    })
    expect(w.html()).toContain('已删除 #99')
  })

  it('undefined names ⇒ raw id rendered (backward compat)', async () => {
    const w = await mountSuspended(SnapshotDiff, {
      props: {
        changedKeys: { tag_ids: true },
        oldSnap: { tag_ids: [1] },
        newSnap: { tag_ids: [2] }
        // no names prop at all — older wiki / synthesised taxonomy diff
      }
    })
    const html = w.html()
    expect(html).toContain('- 1')
    expect(html).toContain('+ 2')
    // No fallback string should appear when names is absent.
    expect(html).not.toContain('已删除')
  })

  it('names dict resolves series_id scalar to display name', async () => {
    const w = await mountSuspended(SnapshotDiff, {
      props: {
        changedKeys: { series_id: true },
        oldSnap: { series_id: 1 },
        newSnap: { series_id: 2 },
        names: {
          series: { '1': '蜂群系列', '2': '克莱托系列' }
        }
      }
    })
    // Both names are resolved from the dict rather than shown as raw ids —
    // but they are DIFFED, not printed whole: 蜂群系列 → 克莱托系列 shares the
    // 系列 suffix, so only 蜂群 / 克莱托 are tinted and 系列 is emitted once.
    // Asserting on the full names would be asserting the old duplicated view.
    const text = w.text()
    expect(text).toContain('蜂群')
    expect(text).toContain('克莱托')
    expect(text.match(/系列/g)?.length).toBe(1)
  })
})
