// Pins the three things about the drawing that are invisible from the picture
// once it is wrong: the direction each relation word is read in, the fact that
// a rename keeps a company on its own layer, and that a cycle in the DATA (two
// makers each recorded as the other's parent — an error, not an impossibility)
// costs a slightly odd layout rather than a hung render.
import { describe, it, expect } from 'vitest'
import {
  buildOfficialGraphLayout,
  OFFICIAL_GRAPH_NODE_WIDTH
} from './galgameOfficialGraphLayout'

const node = (id: number, name: string, work_count = 0) => ({
  id,
  name,
  logo: '',
  work_count
})

const layerOf = (
  layout: ReturnType<typeof buildOfficialGraphLayout>,
  id: number
) => layout.nodes.find((n) => n.official.id === id)!.layer

describe('buildOfficialGraphLayout', () => {
  it('reads parent and imprint edges in OPPOSITE directions', () => {
    // VisualArt's owns Key (a parent edge, read upward) and Na-Ga (an imprint
    // edge, read downward). Both belong on the layer BELOW VisualArt's.
    const layout = buildOfficialGraphLayout({
      nodes: [
        node(24, 'Key', 33),
        node(993, "VisualArt's", 120),
        node(994, 'Na-Ga', 5)
      ],
      edges: [
        { from: 24, to: 993, relation: 'parent' },
        { from: 993, to: 994, relation: 'imprint' }
      ]
    })

    expect(layerOf(layout, 993)).toBe(0)
    expect(layerOf(layout, 24)).toBe(1)
    expect(layerOf(layout, 994)).toBe(1)
    // Both arrows are drawn owner→brand whichever word they arrived as.
    expect(layout.edges.map((e) => [e.kind, e.from, e.to])).toEqual([
      ['subsidiary', 993, 24],
      ['imprint', 993, 994]
    ])
  })

  it('keeps a renamed company on its own layer, in date order', () => {
    // A rename is not a step down an org chart: the company is the same one.
    const layout = buildOfficialGraphLayout({
      nodes: [node(1, 'Old'), node(2, 'New'), node(3, 'Owner', 10)],
      edges: [
        { from: 1, to: 2, relation: 'succeeded_by' },
        { from: 3, to: 1, relation: 'imprint' }
      ]
    })

    expect(layerOf(layout, 1)).toBe(layerOf(layout, 2))
    expect(layerOf(layout, 3)).toBeLessThan(layerOf(layout, 1))
    // Oldest name to the left, so the arrow reads forwards in time.
    const old = layout.nodes.find((n) => n.official.id === 1)!
    const renamed = layout.nodes.find((n) => n.official.id === 2)!
    expect(old.x).toBeLessThan(renamed.x)
  })

  it('folds a wide set of siblings into rows instead of a ribbon', () => {
    // Eight imprints in one row is 1,700px of drawing two boxes tall, which at
    // fit scale is unreadable confetti. They fold into a block.
    const layout = buildOfficialGraphLayout({
      nodes: [
        node(1, 'Owner', 500),
        ...Array.from({ length: 8 }, (_, i) => node(i + 2, `Brand ${i}`, 8 - i))
      ],
      edges: Array.from({ length: 8 }, (_, i) => ({
        from: 1,
        to: i + 2,
        relation: 'imprint' as const
      }))
    })

    const brands = layout.nodes.filter((n) => n.official.id !== 1)
    expect(new Set(brands.map((n) => n.y)).size).toBe(2)
    expect(new Set(brands.map((n) => n.row))).toEqual(new Set([0, 1]))
    // Four columns wide, not eight.
    expect(new Set(brands.map((n) => Math.round(n.x))).size).toBe(4)

    // And the arrows into the second row do NOT come down through the box
    // above them: they run down the empty channel to the left of the column.
    const wrapped = brands.filter((n) => n.row === 1)
    for (const target of wrapped) {
      const edge = layout.edges.find((e) => e.to === target.official.id)!
      expect(edge.path).toContain(`L ${target.x - 212 / 2}`)
    }
  })

  it('survives an ownership cycle', () => {
    const layout = buildOfficialGraphLayout({
      nodes: [node(1, 'A'), node(2, 'B')],
      edges: [
        { from: 1, to: 2, relation: 'parent' },
        { from: 2, to: 1, relation: 'parent' }
      ]
    })

    expect(layout.nodes).toHaveLength(2)
    expect(layout.edges).toHaveLength(2)
    expect(layout.nodes.every((n) => Number.isFinite(n.x))).toBe(true)
  })

  it('normalises the drawing to a (0,0)-anchored box', () => {
    // The viewport frames the graph from these numbers alone — it has no DOM to
    // measure on the server render — so a stray negative coordinate is a graph
    // that arrives half off screen.
    const layout = buildOfficialGraphLayout({
      nodes: [node(1, 'A'), node(2, 'B'), node(3, 'C')],
      edges: [
        { from: 1, to: 2, relation: 'imprint' },
        { from: 1, to: 3, relation: 'imprint' }
      ]
    })

    for (const n of layout.nodes) {
      expect(n.x).toBeGreaterThanOrEqual(OFFICIAL_GRAPH_NODE_WIDTH / 2 - 0.001)
      expect(n.x).toBeLessThanOrEqual(
        layout.width - OFFICIAL_GRAPH_NODE_WIDTH / 2 + 0.001
      )
      expect(n.y).toBeGreaterThan(0)
      expect(n.y).toBeLessThan(layout.height)
    }
  })

  it('draws nothing for a lone 会社', () => {
    const layout = buildOfficialGraphLayout({
      nodes: [node(1, 'A')],
      edges: []
    })
    expect(layout.edges).toHaveLength(0)
    expect(layout.nodes).toHaveLength(1)
  })
})
