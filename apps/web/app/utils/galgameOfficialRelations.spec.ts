import { describe, it, expect } from 'vitest'
import {
  buildOfficialFamilyForest,
  buildOfficialRenameChains,
  buildOfficialSpawnPairs
} from './galgameOfficialRelations'

const node = (id: number, name: string, work_count = 0) => ({
  id,
  name,
  logo: '',
  work_count
})

describe('buildOfficialFamilyForest', () => {
  it('reads parent and imprint edges in OPPOSITE directions', () => {
    const forest = buildOfficialFamilyForest(
      {
        nodes: [
          node(24, 'Key', 33),
          node(993, "VisualArt's", 120),
          node(994, 'Na-Ga', 5)
        ],
        edges: [
          { from: 24, to: 993, relation: 'parent' },
          { from: 993, to: 994, relation: 'imprint' }
        ]
      },
      24
    )

    expect(forest).toHaveLength(1)
    expect(forest[0]!.official.id).toBe(993)
    expect(forest[0]!.role).toBe('root')
    expect(forest[0]!.children.map((c) => [c.official.id, c.role])).toEqual([
      [24, 'subsidiary'],
      [994, 'imprint']
    ])
  })

  it('returns every root, with the current 会社’s tree first', () => {
    const forest = buildOfficialFamilyForest(
      {
        nodes: [node(1, 'A'), node(2, 'B'), node(3, 'C'), node(4, 'D')],
        edges: [
          { from: 2, to: 1, relation: 'parent' },
          { from: 4, to: 3, relation: 'parent' },
          { from: 1, to: 3, relation: 'succeeded_by' }
        ]
      },
      4
    )

    expect(forest.map((r) => r.official.id)).toEqual([3, 1])
  })

  it('drops a node no ownership edge reaches', () => {
    const forest = buildOfficialFamilyForest(
      {
        nodes: [node(1, 'A'), node(2, 'B')],
        edges: [{ from: 1, to: 2, relation: 'spawned' }]
      },
      1
    )

    expect(forest).toEqual([])
  })

  it('survives an ownership cycle', () => {
    const forest = buildOfficialFamilyForest(
      {
        nodes: [node(1, 'A'), node(2, 'B')],
        edges: [
          { from: 1, to: 2, relation: 'parent' },
          { from: 2, to: 1, relation: 'parent' }
        ]
      },
      1
    )

    expect(forest).toHaveLength(1)
    expect(forest[0]!.children).toHaveLength(1)
  })
})

describe('buildOfficialRenameChains', () => {
  it('walks a succession chain oldest-first', () => {
    const chains = buildOfficialRenameChains({
      nodes: [node(1, '旧名'), node(2, '中间名'), node(3, '现名')],
      edges: [
        { from: 2, to: 3, relation: 'succeeded_by' },
        { from: 1, to: 2, relation: 'succeeded_by' }
      ]
    })

    expect(chains).toHaveLength(1)
    expect(chains[0]!.map((n) => n.name)).toEqual(['旧名', '中间名', '现名'])
  })

  it('drops a succession cycle rather than walking it forever', () => {
    const chains = buildOfficialRenameChains({
      nodes: [node(1, 'A'), node(2, 'B')],
      edges: [
        { from: 1, to: 2, relation: 'succeeded_by' },
        { from: 2, to: 1, relation: 'succeeded_by' }
      ]
    })

    expect(chains).toEqual([])
  })
})

describe('buildOfficialSpawnPairs', () => {
  it('reads `from` as the origin', () => {
    const pairs = buildOfficialSpawnPairs({
      nodes: [node(1, '母体'), node(2, '拆分出的公司')],
      edges: [{ from: 1, to: 2, relation: 'spawned' }]
    })

    expect(pairs).toEqual([
      { parent: node(1, '母体'), child: node(2, '拆分出的公司') }
    ])
  })
})
