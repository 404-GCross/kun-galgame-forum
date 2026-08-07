<script setup lang="ts">
// The 会社关系 section: the corporate family around this 会社, drawn as a tree
// rather than the flat "Relations" list every other VN database prints — a
// list tells you VisualArt's is Key's parent, a tree tells you which OTHER
// eleven brands that makes Key a sibling of.
//
// Its own fetch, deliberately. The page's detail payload is refetched on every
// pagination and filter change of the games grid, and none of that can move a
// company under a different parent — folding the graph into that payload would
// re-walk it on every page turn. `lazy` + `watch: false`: the family is
// supporting material and must never hold up the games the page is FOR, and
// there is nothing for it to react to.
const props = defineProps<{
  officialId: number
}>()

const { data } = await useKunFetch<GalgameOfficialRelationGraph>(
  `/galgame-official/${props.officialId}/relation-graph`,
  { lazy: true, method: 'GET', watch: false }
)

// The seed is ALWAYS in the graph, so one node means "no recorded relations"
// and the whole section stays away — an empty 会社关系 heading is worse than
// no heading.
const graph = computed(() =>
  (data.value?.nodes.length ?? 0) > 1 ? data.value : null
)

const forest = computed(() =>
  graph.value ? buildOfficialFamilyForest(graph.value, props.officialId) : []
)
const renameChains = computed(() =>
  graph.value ? buildOfficialRenameChains(graph.value) : []
)

// A spin-off reads from the CURRENT 会社's point of view where it can — "拆分出
// X" / "从 Y 拆分而来" — and neutrally when the pair is between two other
// members of the family, which the capped walk can perfectly well include.
const spawnRows = computed(() =>
  (graph.value ? buildOfficialSpawnPairs(graph.value) : []).map((pair) => {
    if (pair.parent.id === props.officialId) {
      return { prefix: '拆分出', official: pair.child, suffix: '' }
    }
    if (pair.child.id === props.officialId) {
      return { prefix: '从', official: pair.parent, suffix: '拆分而来' }
    }
    return {
      prefix: `${pair.parent.name} 拆分出`,
      official: pair.child,
      suffix: ''
    }
  })
)

// A graph of several nodes joined by nothing this section can draw (only
// relation words outside the three lanes) still renders nothing.
const hasContent = computed(
  () =>
    forest.value.length > 0 ||
    renameChains.value.length > 0 ||
    spawnRows.value.length > 0
)
</script>

<template>
  <div v-if="hasContent" class="space-y-4">
    <KunHeader
      name="会社关系"
      description="该会社所属的企业家族: 母公司、旗下品牌、更名沿革与拆分出的公司。资料来自 NextMoe 目录。"
      scale="h3"
    />

    <div v-if="forest.length" class="space-y-3">
      <h4 class="text-default-500 text-sm">企业家族</h4>
      <!-- Several roots is normal, not an error: the component is connected
           through rename and spin-off edges too, so one graph can hold two
           ownership trees joined only by "A was renamed to B". -->
      <GalgameOfficialRelationTree
        v-for="root in forest"
        :key="root.official.id"
        :nodes="[root]"
        :current-id="officialId"
      />
    </div>

    <div v-if="renameChains.length" class="space-y-3">
      <h4 class="text-default-500 text-sm">更名沿革</h4>
      <!-- Horizontal and scrollable: a rename chain is a sequence in TIME, and
           stacking it vertically would read as a hierarchy — which is exactly
           what the block above it is. -->
      <div
        v-for="(chain, index) in renameChains"
        :key="index"
        class="flex items-center gap-2 overflow-x-auto pb-1"
      >
        <template v-for="(item, step) in chain" :key="item.id">
          <KunIcon
            v-if="step"
            name="lucide:arrow-right"
            class-name="text-default-400 shrink-0"
          />
          <span
            v-if="item.id === officialId"
            class="border-primary-500 bg-primary-50 text-primary-600 shrink-0 rounded-md border px-2 py-1 text-sm font-medium"
          >
            {{ item.name }}
          </span>
          <KunButton
            v-else
            variant="flat"
            color="default"
            size="sm"
            class-name="shrink-0"
            :href="taxonomyDetailPath('official', item.id)"
          >
            {{ item.name }}
          </KunButton>
        </template>
      </div>
    </div>

    <div v-if="spawnRows.length" class="space-y-3">
      <h4 class="text-default-500 text-sm">衍生</h4>
      <div
        v-for="(row, index) in spawnRows"
        :key="index"
        class="text-default-600 flex flex-wrap items-center gap-2 text-sm"
      >
        <KunIcon name="lucide:git-branch" class-name="text-default-400" />
        <span>{{ row.prefix }}</span>
        <span
          v-if="row.official.id === officialId"
          class="text-primary-600 font-medium"
        >
          {{ row.official.name }}
        </span>
        <KunLink
          v-else
          size="sm"
          underline="hover"
          :to="taxonomyDetailPath('official', row.official.id)"
        >
          {{ row.official.name }}
        </KunLink>
        <span v-if="row.suffix">{{ row.suffix }}</span>
      </div>
    </div>
  </div>
</template>
