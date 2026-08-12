
export const OFFICIAL_GRAPH_NODE_WIDTH = 184
export const OFFICIAL_GRAPH_NODE_HEIGHT = 56

const GAP_X = 28
const ROW_GAP = 16
const LAYER_GAP = 92
const MIN_STEP = OFFICIAL_GRAPH_NODE_WIDTH + GAP_X
const ROW_STEP = OFFICIAL_GRAPH_NODE_HEIGHT + ROW_GAP
const PADDING = 28

export type GalgameOfficialGraphEdgeKind =
  | 'subsidiary'
  | 'imprint'
  | 'succession'
  | 'spawn'

export interface GalgameOfficialGraphEdge {
  id: string
  kind: GalgameOfficialGraphEdgeKind
  from: number
  to: number
  path: string
  head: { x: number; y: number; angle: number }
  labelX: number
  labelY: number
}

export interface GalgameOfficialGraphPlacedNode {
  official: GalgameOfficialRelationNode
  layer: number
  row: number
  x: number
  y: number
}

export interface GalgameOfficialGraphLayout {
  nodes: GalgameOfficialGraphPlacedNode[]
  edges: GalgameOfficialGraphEdge[]
  width: number
  height: number
}

interface SemanticEdge {
  kind: GalgameOfficialGraphEdgeKind
  from: number
  to: number
}

const semanticEdges = (graph: GalgameOfficialRelationGraph): SemanticEdge[] => {
  const known = new Set(graph.nodes.map((n) => n.id))
  const seen = new Set<string>()
  const out: SemanticEdge[] = []

  for (const e of graph.edges) {
    if (e.from === e.to || !known.has(e.from) || !known.has(e.to)) continue

    let edge: SemanticEdge | null = null
    switch (e.relation) {
      case 'parent':
        edge = { kind: 'subsidiary', from: e.to, to: e.from }
        break
      case 'subsidiary':
        edge = { kind: 'subsidiary', from: e.from, to: e.to }
        break
      case 'imprint':
        edge = { kind: 'imprint', from: e.from, to: e.to }
        break
      case 'imprint_of':
        edge = { kind: 'imprint', from: e.to, to: e.from }
        break
      case 'succeeded_by':
        edge = { kind: 'succession', from: e.from, to: e.to }
        break
      case 'formerly':
        edge = { kind: 'succession', from: e.to, to: e.from }
        break
      case 'spawned':
        edge = { kind: 'spawn', from: e.from, to: e.to }
        break
      case 'origin':
        edge = { kind: 'spawn', from: e.to, to: e.from }
        break
    }
    if (!edge) continue

    const key = `${edge.kind}:${edge.from}:${edge.to}`
    if (seen.has(key)) continue
    seen.add(key)
    out.push(edge)
  }
  return out
}

const renameGroups = (ids: number[], edges: SemanticEdge[]) => {
  const parent = new Map(ids.map((id) => [id, id]))
  const find = (id: number): number => {
    let root = id
    while (parent.get(root) !== root) root = parent.get(root)!
    let cursor = id
    while (parent.get(cursor) !== root) {
      const next = parent.get(cursor)!
      parent.set(cursor, root)
      cursor = next
    }
    return root
  }
  for (const e of edges) {
    if (e.kind !== 'succession') continue
    const a = find(e.from)
    const b = find(e.to)
    if (a !== b) parent.set(a, b)
  }
  return { find }
}

const assignLayers = (
  ids: number[],
  edges: SemanticEdge[],
  groupOf: (id: number) => number
) => {
  const groupLayer = new Map<number, number>(ids.map((id) => [groupOf(id), 0]))
  const ownership = edges.filter(
    (e) => e.kind === 'subsidiary' || e.kind === 'imprint'
  )

  for (let round = 0; round < groupLayer.size; round++) {
    let changed = false
    for (const e of ownership) {
      const above = groupOf(e.from)
      const below = groupOf(e.to)
      if (above === below) continue
      const want = groupLayer.get(above)! + 1
      if (groupLayer.get(below)! < want) {
        groupLayer.set(below, want)
        changed = true
      }
    }
    if (!changed) break
  }

  return new Map(ids.map((id) => [id, groupLayer.get(groupOf(id))!]))
}

const orderLayers = (
  layers: number[][],
  neighbours: Map<number, number[]>,
  layerOf: Map<number, number>
) => {
  for (let pass = 0; pass < 4; pass++) {
    const downward = pass % 2 === 0
    const indices = layers.map((_, i) => i)
    for (const li of downward ? indices : indices.reverse()) {
      const ref = downward ? li - 1 : li + 1
      const refLayer = layers[ref]
      const current = layers[li]
      if (!refLayer || !current) continue

      const position = new Map(refLayer.map((id, i) => [id, i]))
      const keyed = current.map((id, i) => {
        const seats = (neighbours.get(id) ?? [])
          .filter((other) => layerOf.get(other) === ref)
          .map((other) => position.get(other)!)
        const key = seats.length
          ? seats.reduce((a, b) => a + b, 0) / seats.length
          : i
        return { id, key, i }
      })
      keyed.sort((a, b) => a.key - b.key || a.i - b.i)
      layers[li] = keyed.map((k) => k.id)
    }
  }
}

const groupRenameChains = (layers: number[][], edges: SemanticEdge[]) => {
  const next = new Map<number, number>()
  const succeeded = new Set<number>()
  for (const e of edges) {
    if (e.kind !== 'succession' || next.has(e.from)) continue
    next.set(e.from, e.to)
    succeeded.add(e.to)
  }

  for (const start of next.keys()) {
    if (succeeded.has(start)) continue
    const chain: number[] = []
    const walked = new Set<number>()
    let cursor: number | undefined = start
    while (cursor !== undefined && !walked.has(cursor)) {
      walked.add(cursor)
      chain.push(cursor)
      cursor = next.get(cursor)
    }
    if (chain.length < 2) continue

    for (const layer of layers) {
      const members = chain.filter((id) => layer.includes(id))
      if (members.length < 2) continue
      const at = Math.min(...members.map((id) => layer.indexOf(id)))
      const rest = layer.filter((id) => !members.includes(id))
      layer.splice(0, layer.length, ...rest)
      layer.splice(at, 0, ...members)
    }
  }
}

const ownershipParents = (edges: SemanticEdge[]) => {
  const parent = new Map<number, number>()
  for (const e of edges) {
    if (e.kind !== 'subsidiary' && e.kind !== 'imprint') continue
    if (!parent.has(e.to)) parent.set(e.to, e.from)
  }
  return parent
}

const TARGET_BLOCK_ASPECT = 2.5
const MAX_BLOCK_ROWS = 4

const wrapLayer = (
  layer: number[],
  parentOf: Map<number, number>,
  hasChildren: Set<number>
) => {
  const blockKey = (id: number) => {
    const owner = parentOf.get(id)
    if (owner !== undefined) return `owned:${owner}`
    return hasChildren.has(id) ? `root:${id}` : 'loose'
  }

  const blocks: { key: string; ids: number[] }[] = []
  for (const id of layer) {
    const key = blockKey(id)
    const last = blocks.at(-1)
    if (last && last.key === key) last.ids.push(id)
    else blocks.push({ key, ids: [id] })
  }

  const columns: number[][] = []
  for (const block of blocks) {
    const count = block.ids.length
    const rows = Math.min(
      MAX_BLOCK_ROWS,
      Math.max(1, Math.round(Math.sqrt(count / TARGET_BLOCK_ASPECT)))
    )
    const perRow = Math.ceil(count / rows)
    const fresh: number[][] = Array.from({ length: perRow }, () => [])
    block.ids.forEach((id, i) => fresh[i % perRow]!.push(id))
    columns.push(...fresh)
  }
  return columns
}

const assignX = (
  layerColumns: number[][][],
  neighbours: Map<number, number[]>,
  layerOf: Map<number, number>
) => {
  const x = new Map<number, number>()
  const setColumn = (column: number[], at: number) => {
    for (const id of column) x.set(id, at)
  }
  for (const columns of layerColumns) {
    columns.forEach((column, i) => setColumn(column, i * MIN_STEP))
  }

  for (let pass = 0; pass < 6; pass++) {
    const downward = pass % 2 === 0
    const indices = layerColumns.map((_, i) => i)
    for (const li of downward ? indices : indices.reverse()) {
      const ref = downward ? li - 1 : li + 1
      const current = layerColumns[li]
      if (!current || !layerColumns[ref]) continue

      const wanted = current.map((column) => {
        const seats = column
          .flatMap((id) => neighbours.get(id) ?? [])
          .filter((other) => layerOf.get(other) === ref)
          .map((other) => x.get(other)!)
        return seats.length
          ? seats.reduce((a, b) => a + b, 0) / seats.length
          : x.get(column[0]!)!
      })

      let cursor = -Infinity
      const placed = wanted.map((want) => {
        const at = Math.max(want, cursor + MIN_STEP)
        cursor = at
        return at
      })

      const drift =
        placed.reduce((sum, at, i) => sum + (wanted[i]! - at), 0) /
        (placed.length || 1)
      current.forEach((column, i) => setColumn(column, placed[i]! + drift))
    }
  }
  return x
}

const NODE_HALF_W = OFFICIAL_GRAPH_NODE_WIDTH / 2
const NODE_HALF_H = OFFICIAL_GRAPH_NODE_HEIGHT / 2
const ARROW_GAP = 5
export const OFFICIAL_GRAPH_HEAD_LENGTH = 9

const headAt = (x: number, y: number, ux: number, uy: number) => ({
  stopX: x - ux * OFFICIAL_GRAPH_HEAD_LENGTH,
  stopY: y - uy * OFFICIAL_GRAPH_HEAD_LENGTH,
  head: { x, y, angle: (Math.atan2(uy, ux) * 180) / Math.PI }
})

const edgeGeometry = (
  a: GalgameOfficialGraphPlacedNode,
  b: GalgameOfficialGraphPlacedNode,
  layerTop: number[]
) => {
  if (b.row > 0 && b.y > a.y) {
    const spine = b.x - NODE_HALF_W - GAP_X + 4
    const busY = (layerTop[b.layer] ?? b.y) - LAYER_GAP / 3
    const entry = b.x - NODE_HALF_W - ARROW_GAP
    const y1 = a.y + NODE_HALF_H
    const { stopX, head } = headAt(entry, b.y, 1, 0)
    const r = Math.max(0, Math.min(8, (b.y - busY) / 2, stopX - spine))
    return {
      path: `M ${a.x} ${y1} C ${a.x} ${(y1 + busY) / 2} ${spine} ${(y1 + busY) / 2} ${spine} ${busY} L ${spine} ${b.y - r} Q ${spine} ${b.y} ${spine + r} ${b.y} L ${stopX} ${b.y}`,
      head,
      labelX: spine,
      labelY: (busY + b.y) / 2
    }
  }

  if (a.y === b.y) {
    const dir = b.x >= a.x ? 1 : -1
    const x1 = a.x + dir * NODE_HALF_W
    const x2 = b.x - dir * (NODE_HALF_W + ARROW_GAP)
    const lift = Math.min(56, Math.abs(x2 - x1) / 3 + 16)
    const { stopX, stopY, head } = headAt(x2, b.y, dir / Math.SQRT2, Math.SQRT1_2)
    return {
      path: `M ${x1} ${a.y} C ${x1 + dir * lift} ${a.y - lift} ${x2 - dir * lift} ${b.y - lift} ${stopX} ${stopY}`,
      head,
      labelX: (x1 + x2) / 2,
      labelY: a.y - lift * 0.72
    }
  }

  const down = b.y > a.y ? 1 : -1
  const y1 = a.y + down * NODE_HALF_H
  const y2 = b.y - down * (NODE_HALF_H + ARROW_GAP)
  const bend = (y2 - y1) * 0.5
  const { stopY, head } = headAt(b.x, y2, 0, down)
  return {
    path: `M ${a.x} ${y1} C ${a.x} ${y1 + bend} ${b.x} ${y2 - bend} ${b.x} ${stopY}`,
    head,
    labelX: (a.x + b.x) / 2,
    labelY: (y1 + y2) / 2
  }
}

export const buildOfficialGraphLayout = (
  graph: GalgameOfficialRelationGraph
): GalgameOfficialGraphLayout => {
  const edges = semanticEdges(graph)
  const ids = graph.nodes.map((n) => n.id)
  if (!ids.length) return { nodes: [], edges: [], width: 0, height: 0 }

  const { find } = renameGroups(ids, edges)
  const layerOf = assignLayers(ids, edges, find)

  const neighbours = new Map<number, number[]>(ids.map((id) => [id, []]))
  for (const e of edges) {
    neighbours.get(e.from)!.push(e.to)
    neighbours.get(e.to)!.push(e.from)
  }

  const depth = Math.max(...layerOf.values()) + 1
  const byId = new Map(graph.nodes.map((n) => [n.id, n]))
  const layers: number[][] = Array.from({ length: depth }, (_, li) =>
    graph.nodes
      .filter((n) => layerOf.get(n.id) === li)
      .sort(
        (a, b) => b.work_count - a.work_count || a.name.localeCompare(b.name)
      )
      .map((n) => n.id)
  )

  orderLayers(layers, neighbours, layerOf)
  groupRenameChains(layers, edges)

  const parentOf = ownershipParents(edges)
  const hasChildren = new Set([...parentOf.values()])
  const layerColumns = layers.map((layer) =>
    wrapLayer(layer, parentOf, hasChildren)
  )
  const x = assignX(layerColumns, neighbours, layerOf)

  const rowsIn = layerColumns.map((columns) =>
    Math.max(1, ...columns.map((column) => column.length))
  )
  const layerTop: number[] = []
  for (let li = 0; li < depth; li++) {
    const previous = layerTop[li - 1]
    layerTop[li] =
      previous === undefined
        ? PADDING
        : previous + rowsIn[li - 1]! * ROW_STEP - ROW_GAP + LAYER_GAP
  }

  const rowOf = new Map<number, number>()
  for (const columns of layerColumns) {
    for (const column of columns) {
      column.forEach((id, row) => rowOf.set(id, row))
    }
  }

  const originX = Math.min(...x.values()) - NODE_HALF_W - PADDING
  const nodes: GalgameOfficialGraphPlacedNode[] = ids.map((id) => {
    const layer = layerOf.get(id)!
    const row = rowOf.get(id) ?? 0
    return {
      official: byId.get(id)!,
      layer,
      row,
      x: x.get(id)! - originX,
      y: layerTop[layer]! + row * ROW_STEP + NODE_HALF_H
    }
  })
  const placed = new Map(nodes.map((n) => [n.official.id, n]))

  return {
    nodes,
    edges: edges.map((e) => {
      const a = placed.get(e.from)!
      const b = placed.get(e.to)!
      return {
        id: `${e.kind}:${e.from}:${e.to}`,
        ...e,
        ...edgeGeometry(a, b, layerTop)
      }
    }),
    width: Math.max(...nodes.map((n) => n.x)) + NODE_HALF_W + PADDING,
    height:
      layerTop[depth - 1]! + rowsIn[depth - 1]! * ROW_STEP - ROW_GAP + PADDING
  }
}
