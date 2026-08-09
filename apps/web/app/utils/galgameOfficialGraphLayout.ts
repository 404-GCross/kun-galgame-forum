// Placing a corporate family on a plane.
//
// The three text lanes (tree / rename chain / spin-off rows) each answer one
// question well and none of them answers the one a reader of a big publisher
// actually has: how does all of this fit together? VisualArt's has a dozen
// brands, two of which were renamed and one of which was split off from a
// third — three separate blocks make that four disconnected facts. One picture
// makes it one shape.
//
// So: a layered ("Sugiyama-lite") drawing. Ownership decides the LAYER — a
// brand always sits below its owner — and the two lateral relations, a rename
// and a spin-off, stay on the layer they start on, because neither of them is
// a step down in a hierarchy: a renamed company is the same company, and a
// spin-off is a sibling, not a child.
//
// Everything here is pure, deterministic and cycle-guarded. Deterministic
// matters twice over: the component renders on the server and hydrates on the
// client, and a layout that consulted anything random would visibly jump.
//
// The same edge-direction trap the family forest documents applies here and is
// handled once, in `semanticEdges`: an edge reads "`to` is the `relation` of
// `from`", so `parent` runs child→parent while `imprint` runs owner→brand.

/** Node box, in layout units (= CSS pixels at scale 1). */
export const OFFICIAL_GRAPH_NODE_WIDTH = 184
export const OFFICIAL_GRAPH_NODE_HEIGHT = 56

const GAP_X = 28
/** Between the wrapped rows of ONE sibling block — tight, because those rows
 * are one group, not two. */
const ROW_GAP = 16
/** Between layers, which is a change of meaning and gets the air to say so. */
const LAYER_GAP = 92
const MIN_STEP = OFFICIAL_GRAPH_NODE_WIDTH + GAP_X
const ROW_STEP = OFFICIAL_GRAPH_NODE_HEIGHT + ROW_GAP
/** Room around the drawing for the arrows that run down the outside of it. */
const PADDING = 28

/** How the two ends of a drawn edge relate. Ownership is vertical (`subsidiary`
 * / `imprint`), the other two are lateral. Kept distinct from the wire
 * vocabulary: these are always read source→target as drawn. */
export type GalgameOfficialGraphEdgeKind =
  | 'subsidiary'
  | 'imprint'
  | 'succession'
  | 'spawn'

export interface GalgameOfficialGraphEdge {
  /** Stable across renders — used as the `v-for` key and as the hover token. */
  id: string
  kind: GalgameOfficialGraphEdgeKind
  from: number
  to: number
  /** SVG path, in layout coordinates. */
  path: string
  labelX: number
  labelY: number
}

export interface GalgameOfficialGraphPlacedNode {
  official: GalgameOfficialRelationNode
  layer: number
  /** Which wrapped row of its sibling block it sits on. 0 for everything that
   * did not need wrapping, and the reason edges into it are routed differently
   * when it is not. */
  row: number
  /** Centre of the box. */
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

/** Normalises the wire's eight relation words into four drawable ones, always
 * oriented the way the arrow points: owner→brand, old name→new name,
 * origin→spin-off. The catalog only emits the canonical half of each inverse
 * pair, but reading the other half costs one line each and means a widened
 * upstream vocabulary degrades into a correct drawing rather than a missing
 * edge. */
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

/** Union-find over rename edges: a company and its later names are ONE thing on
 * the org chart, so they share a layer and get drawn side by side. */
const renameGroups = (ids: number[], edges: SemanticEdge[]) => {
  const parent = new Map(ids.map((id) => [id, id]))
  const find = (id: number): number => {
    let root = id
    while (parent.get(root) !== root) root = parent.get(root)!
    // Path compression, so a long rename chain stays O(1) to query.
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

/** Longest-path layering over ownership edges, relaxed group by group.
 *
 * The relaxation is capped at one round per group rather than run to a fixed
 * point: the catalog's walk is cycle-safe but its DATA is not (two makers each
 * recorded as the other's parent is a data error, not an impossibility), and a
 * cycle must cost a slightly odd drawing, never a hung render. */
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

/** Reduce crossings by barycentre sweeps: repeatedly reorder a layer by the
 * mean position of each node's neighbours in the layer next to it. Four passes
 * is where this stops paying for itself on graphs the size the catalog caps at
 * (≤ 60 nodes, depth ≤ 4). */
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

/** Pull each rename chain back together after the sweeps and put it in
 * chronological order — a chain read right to left, or with an unrelated brand
 * wedged into the middle of it, is a chain that has to be decoded rather than
 * read. */
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

/** Child → its single owner, over ownership edges only. First edge wins: the
 * catalog can record a maker under two owners (a joint venture, or a merge that
 * kept both histories), and a sibling block cannot belong to two parents. */
const ownershipParents = (edges: SemanticEdge[]) => {
  const parent = new Map<number, number>()
  for (const e of edges) {
    if (e.kind !== 'subsidiary' && e.kind !== 'imprint') continue
    if (!parent.has(e.to)) parent.set(e.to, e.from)
  }
  return parent
}

/**
 * Wrapping — the fix for the ribbon.
 *
 * A publisher with a dozen imprints laid out in one row is 2,500px of drawing
 * three boxes tall: fitted to any real viewport it is a strip of unreadable
 * confetti, and no amount of zooming makes the SHAPE visible, which is the one
 * thing a picture was supposed to add. So a wide set of siblings folds into a
 * compact block of rows instead — the same twelve brands as 2 × 6, which is
 * both readable at fit scale and legible AS a group.
 *
 * Only true siblings fold together. Wrapping an arbitrary slice of a layer
 * would put two unrelated brands on the same row and imply they belong to each
 * other, which is worse than the ribbon.
 *
 * The result is a list of COLUMNS per layer (a column is one x, holding one
 * node per row), because everything downstream — spacing, barycentres, the
 * bounding box — is about horizontal room, and a column is the unit of that.
 */
const TARGET_BLOCK_ASPECT = 2.5
const MAX_BLOCK_ROWS = 4

const wrapLayer = (
  layer: number[],
  parentOf: Map<number, number>,
  hasChildren: Set<number>
) => {
  // What may share a block: true siblings, and the loose ends — makers with no
  // owner and nothing under them, which arrive in the walk because SOMETHING in
  // the family touches them and which are otherwise a long thin row of names.
  // Folding those together implies no relationship a row of them did not.
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
    // Row-major, so the block reads left to right then down — which is also
    // heaviest-first, since that is the order the sweeps left it in.
    block.ids.forEach((id, i) => fresh[i % perRow]!.push(id))
    columns.push(...fresh)
  }
  return columns
}

/** Horizontal placement: each column wants to sit under the average of what its
 * nodes are connected to, subject to never overlapping its neighbour.
 * Alternating the sweep direction is what lets a parent centre over its
 * children and children gather under their parent instead of one winning
 * outright. */
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

      // Left to right, each column taking its wish or the first free seat after
      // its left neighbour — whichever is further right.
      let cursor = -Infinity
      const placed = wanted.map((want) => {
        const at = Math.max(want, cursor + MIN_STEP)
        cursor = at
        return at
      })

      // The greedy pass can only ever push right, which walks the whole layer
      // away from what it is connected to. Shifting the finished layer back by
      // its own mean drift restores the centring without disturbing the
      // spacing it just resolved.
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
/** Arrow heads are drawn at the path end; stopping short of the border keeps
 * the head beside the card rather than on top of its outline. */
const ARROW_GAP = 6

const edgeGeometry = (
  a: GalgameOfficialGraphPlacedNode,
  b: GalgameOfficialGraphPlacedNode,
  layerTop: number[]
) => {
  // An arrow into a WRAPPED row cannot come down from the top: the box directly
  // above it belongs to the same block, and a line through a sibling reads as a
  // relation to that sibling. It comes down the empty channel to the left of
  // the column instead — the gap is exactly GAP_X wide and, because every
  // column keeps at least MIN_STEP from its neighbour, it is always free —
  // and turns in from the side, which is the org-chart comb everyone can read.
  if (b.row > 0 && b.y > a.y) {
    const spine = b.x - MIN_STEP / 2
    const busY = (layerTop[b.layer] ?? b.y) - LAYER_GAP / 3
    const entry = b.x - NODE_HALF_W - ARROW_GAP
    const y1 = a.y + NODE_HALF_H
    return {
      path: `M ${a.x} ${y1} C ${a.x} ${(y1 + busY) / 2} ${spine} ${(y1 + busY) / 2} ${spine} ${busY} L ${spine} ${b.y} L ${entry} ${b.y}`,
      labelX: spine,
      labelY: (busY + b.y) / 2
    }
  }

  if (a.y === b.y) {
    // Lateral: a shallow arc over the gap, so two nodes that happen to sit next
    // to each other without being related are never joined by a straight line
    // that looks like the row itself.
    const dir = b.x >= a.x ? 1 : -1
    const x1 = a.x + dir * NODE_HALF_W
    const x2 = b.x - dir * (NODE_HALF_W + ARROW_GAP)
    const lift = Math.min(56, Math.abs(x2 - x1) / 3 + 16)
    return {
      path: `M ${x1} ${a.y} C ${x1 + dir * lift} ${a.y - lift} ${x2 - dir * lift} ${b.y - lift} ${x2} ${b.y}`,
      labelX: (x1 + x2) / 2,
      labelY: a.y - lift * 0.72
    }
  }

  const down = b.y > a.y ? 1 : -1
  const y1 = a.y + down * NODE_HALF_H
  const y2 = b.y - down * (NODE_HALF_H + ARROW_GAP)
  const bend = (y2 - y1) * 0.5
  return {
    path: `M ${a.x} ${y1} C ${a.x} ${y1 + bend} ${b.x} ${y2 - bend} ${b.x} ${y2}`,
    labelX: (a.x + b.x) / 2,
    labelY: (y1 + y2) / 2
  }
}

/**
 * buildOfficialGraphLayout — the whole drawing, in one deterministic pass.
 *
 * Coordinates come out normalised to a (0,0)-anchored box of `width` × `height`
 * so the viewport can fit it without measuring the DOM, which is what lets the
 * graph arrive already framed on the server render.
 */
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
      // Heaviest first is the only ranking that makes a twelve-brand publisher
      // readable, and it is a stable seed for the sweeps below.
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

  // A layer is as tall as its deepest wrapped block, so a layer that needed no
  // wrapping does not pay for one that did.
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
