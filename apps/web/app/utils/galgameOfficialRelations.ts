// Turning the catalog's flat relation edges into the three shapes the 会社
// page draws: a family FOREST, rename CHAINS and spin-off PAIRS.
//
// The one fact everything here turns on: an edge reads "`to` is the `relation`
// of `from`". That makes the two ownership words point in OPPOSITE directions —
//
//   {from: Key,          to: VisualArt's, relation: 'parent'}  → VisualArt's is above Key
//   {from: VisualArt's,  to: Key,         relation: 'imprint'} → Key is below VisualArt's
//
// — so a derivation that treated `to` as "the node above" in both cases would
// hang every brand under its own owner's owner. Both are normalised below into
// one (child, parent) pair.
//
// Everything is cycle-guarded. The upstream walk is cycle-SAFE (it never
// revisits a node), but the data it returns can still contain a cycle — two
// makers each recorded as the other's parent is a data error, not an
// impossibility, and it must cost a broken chip, never a hung render.

/** One node of the rendered family tree. */
export interface GalgameOfficialFamilyNode {
  official: GalgameOfficialRelationNode
  /** What this node is to the node ABOVE it: 'subsidiary' | 'imprint', or
   * 'root' when there is none. Keys into KUN_GALGAME_OFFICIAL_TREE_ROLE_MAP. */
  role: string
  children: GalgameOfficialFamilyNode[]
}

/** A node's siblings are ordered by how much of the catalogue they hold — the
 * only ranking that makes a 12-brand publisher readable — and by name when
 * that ties (so the order is stable rather than upstream-order-dependent). */
const byWeight = (a: GalgameOfficialFamilyNode, b: GalgameOfficialFamilyNode) =>
  b.official.work_count - a.official.work_count ||
  a.official.name.localeCompare(b.official.name)

/**
 * buildOfficialFamilyForest — the ownership hierarchy over parent + imprint
 * edges.
 *
 * Roots are the nodes with no edge leading upward. There can be SEVERAL: the
 * component is connected through rename / spin-off edges too, so a graph can
 * hold two unrelated ownership trees joined only by "A was renamed to B".
 * Every one of them is returned, with the tree holding `currentId` first.
 *
 * Single-node roots are dropped — a maker whose only relations are a rename or
 * a spin-off has nothing to draw as a tree, and those edges are already shown
 * by the other two lanes. A node reached by no ownership edge at all therefore
 * disappears from this lane rather than becoming a row of one.
 */
export const buildOfficialFamilyForest = (
  graph: GalgameOfficialRelationGraph,
  currentId: number
): GalgameOfficialFamilyNode[] => {
  const byId = new Map(graph.nodes.map((n) => [n.id, n]))
  // child → its single parent. First edge wins: the catalog can record a maker
  // under two owners (a joint venture, or a merge that kept both histories),
  // and a tree cannot draw that — the extra owner stays visible as the other
  // node's own subtree.
  const upward = new Map<number, { parent: number; role: string }>()
  const link = (child: number, parent: number, role: string) => {
    if (child === parent || upward.has(child)) return
    if (!byId.has(child) || !byId.has(parent)) return
    upward.set(child, { parent, role })
  }
  for (const e of graph.edges) {
    if (e.relation === 'parent') link(e.from, e.to, 'subsidiary')
    if (e.relation === 'imprint') link(e.to, e.from, 'imprint')
  }

  // Break any ownership cycle by cutting the link that closes it, so the node
  // becomes a root instead of an unreachable island.
  for (const id of byId.keys()) {
    const seen = new Set<number>([id])
    let cursor = upward.get(id)?.parent
    while (cursor !== undefined) {
      if (seen.has(cursor)) {
        upward.delete(cursor)
        break
      }
      seen.add(cursor)
      cursor = upward.get(cursor)?.parent
    }
  }

  const children = new Map<number, number[]>()
  for (const [child, { parent }] of upward) {
    children.set(parent, [...(children.get(parent) ?? []), child])
  }

  const build = (id: number, role: string): GalgameOfficialFamilyNode => ({
    official: byId.get(id)!,
    role,
    children: (children.get(id) ?? [])
      .map((c) => build(c, upward.get(c)!.role))
      .sort(byWeight)
  })

  const contains = (node: GalgameOfficialFamilyNode): boolean =>
    node.official.id === currentId || node.children.some(contains)

  return graph.nodes
    .filter((n) => !upward.has(n.id) && (children.get(n.id)?.length ?? 0) > 0)
    .map((n) => build(n.id, 'root'))
    .sort((a, b) => Number(contains(b)) - Number(contains(a)) || byWeight(a, b))
}

/**
 * buildOfficialRenameChains — 更名沿革, oldest name first.
 *
 * A `succeeded_by` edge reads "`to` is the successor of `from`", so the chain
 * simply follows `to`. A chain starts at a node nothing succeeds into; a cycle
 * (which would mean a company renamed itself back) yields no start at all, so
 * it is dropped rather than walked forever.
 */
export const buildOfficialRenameChains = (
  graph: GalgameOfficialRelationGraph
): GalgameOfficialRelationNode[][] => {
  const byId = new Map(graph.nodes.map((n) => [n.id, n]))
  const next = new Map<number, number>()
  const succeeded = new Set<number>()
  for (const e of graph.edges) {
    if (e.relation !== 'succeeded_by') continue
    if (!byId.has(e.from) || !byId.has(e.to) || next.has(e.from)) continue
    next.set(e.from, e.to)
    succeeded.add(e.to)
  }

  const chains: GalgameOfficialRelationNode[][] = []
  for (const start of next.keys()) {
    if (succeeded.has(start)) continue
    const chain: GalgameOfficialRelationNode[] = []
    const seen = new Set<number>()
    let cursor: number | undefined = start
    while (cursor !== undefined && !seen.has(cursor)) {
      seen.add(cursor)
      chain.push(byId.get(cursor)!)
      cursor = next.get(cursor)
    }
    if (chain.length > 1) chains.push(chain)
  }
  return chains
}

/** One spin-off: `parent` split `child` off. A `spawned` edge reads "`to` is
 * what `from` spawned", so `from` is the origin. */
export interface GalgameOfficialSpawnPair {
  parent: GalgameOfficialRelationNode
  child: GalgameOfficialRelationNode
}

export const buildOfficialSpawnPairs = (
  graph: GalgameOfficialRelationGraph
): GalgameOfficialSpawnPair[] => {
  const byId = new Map(graph.nodes.map((n) => [n.id, n]))
  const pairs: GalgameOfficialSpawnPair[] = []
  for (const e of graph.edges) {
    if (e.relation !== 'spawned') continue
    const parent = byId.get(e.from)
    const child = byId.get(e.to)
    if (parent && child) pairs.push({ parent, child })
  }
  return pairs
}
