export interface GalgameOfficialFamilyNode {
  official: GalgameOfficialRelationNode
  role: string
  children: GalgameOfficialFamilyNode[]
}

const byWeight = (a: GalgameOfficialFamilyNode, b: GalgameOfficialFamilyNode) =>
  b.official.work_count - a.official.work_count ||
  a.official.name.localeCompare(b.official.name)

export const buildOfficialFamilyForest = (
  graph: GalgameOfficialRelationGraph,
  currentId: number
): GalgameOfficialFamilyNode[] => {
  const byId = new Map(graph.nodes.map((n) => [n.id, n]))
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
