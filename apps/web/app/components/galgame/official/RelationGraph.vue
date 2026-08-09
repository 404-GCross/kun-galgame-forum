<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'

// The 会社关系 section: the corporate family around this 会社.
//
// Drawn, now, rather than listed. The three text lanes below the tab answer one
// question each and a publisher of any size has the fourth: how does all of
// this fit together? VisualArt's owns a dozen brands, two of which were renamed
// and one of which was split off from a third — as three stacked blocks that is
// four disconnected facts and a lot of scrolling, and the relationship between
// them is left for the reader to assemble. One picture is the assembly.
//
// The list did not go away (see RelationLanes for why); it is the other tab.
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

// Laid out once per payload, not per render: the arithmetic is pure and the
// payload cannot change without a navigation.
const layout = computed(() =>
  graph.value ? buildOfficialGraphLayout(graph.value) : null
)

// Several nodes joined by nothing drawable is still nothing to draw.
const hasContent = computed(() => (layout.value?.edges.length ?? 0) > 0)

const view = ref('graph')
const tabs: KunTabItem[] = [
  { value: 'graph', textValue: '关系图', icon: 'lucide:git-fork' },
  { value: 'list', textValue: '列表', icon: 'lucide:list-tree' }
]

// Starts on the 会社 the reader came for, so the panel is answering about
// something from the first frame rather than showing an empty state next to a
// full picture.
const selectedId = ref<number | null>(props.officialId)

// The fullscreen view is the SAME canvas in a bigger box, sharing the selection
// — not a second graph. A big family only becomes readable when it has the
// screen, and the inline strip is the preview that gets you there.
const isFullscreen = ref(false)

const open = (id: number) => {
  isFullscreen.value = false
  return navigateTo(taxonomyDetailPath('official', id))
}
</script>

<template>
  <div v-if="hasContent && graph && layout" class="space-y-4">
    <KunHeader
      name="会社关系"
      description="该会社所属的企业家族: 母公司、旗下品牌、更名沿革与拆分出的公司。资料来自 NextMoe 目录。"
      scale="h3"
    >
      <template #headerEndContent>
        <KunTab v-model="view" :items="tabs" variant="light" size="sm" />
      </template>
    </KunHeader>

    <!-- The panel is a column beside the canvas on a wide screen and a card
         under it on a narrow one — never a popover pinned to the node, which on
         a graph you can pan is a tooltip that walks off the screen. -->
    <div
      v-if="view === 'graph'"
      class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_17rem]"
    >
      <GalgameOfficialGraphCanvas
        v-model:selected-id="selectedId"
        :layout="layout"
        :current-id="officialId"
        @open="open"
        @expand="isFullscreen = true"
      />
      <GalgameOfficialGraphInspector
        :layout="layout"
        :current-id="officialId"
        :selected-id="selectedId"
        @select="selectedId = $event"
      />
    </div>

    <GalgameOfficialRelationLanes
      v-else
      :graph="graph"
      :official-id="officialId"
    />

    <!-- Outside the tab branch: it is a view OF the graph, not one of the two
         tabs, and putting it between them broke the v-if / v-else pair. -->
    <KunModal
      v-model="isFullscreen"
      inner-class-name="w-[96vw] h-[92vh] max-w-none"
      aria-label="会社关系图"
    >
      <!-- Mounted only while open, so the fullscreen canvas measures its real
           box on the first frame and frames the graph to THAT rather than to
           the strip it came from. -->
      <div
        v-if="isFullscreen"
        class="flex h-full flex-col gap-3 lg:flex-row lg:gap-4"
      >
        <div class="min-h-0 flex-1">
          <GalgameOfficialGraphCanvas
            v-model:selected-id="selectedId"
            :layout="layout"
            :current-id="officialId"
            :is-fullscreen="true"
            @open="open"
          />
        </div>
        <div class="lg:w-72 lg:shrink-0">
          <GalgameOfficialGraphInspector
            :layout="layout"
            :current-id="officialId"
            :selected-id="selectedId"
            @select="selectedId = $event"
          />
        </div>
      </div>
    </KunModal>
  </div>
</template>
