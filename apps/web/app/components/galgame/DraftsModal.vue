<script setup lang="ts">
type EntityType = 'official' | 'tag' | 'engine' | 'series'

const props = defineProps<{
  entityType: EntityType
  entityId: number
  entityName: string
}>()

const open = defineModel<boolean>({ required: true })

const LIMIT = 24

const ENTITY_LABEL: Record<EntityType, string> = {
  official: '会社',
  tag: '标签',
  engine: '引擎',
  series: '系列'
}
const entityLabel = computed(() => ENTITY_LABEL[props.entityType])
const scopeParam = computed(() => `${props.entityType}_id`)

const items = ref<GalgameCard[]>([])
const total = ref(0)
const page = ref(0)
const status = ref<'idle' | 'loading' | 'loadingMore' | 'error'>('idle')

const hasMore = computed(() => items.value.length < total.value)

const fetchPage = async (more = false) => {
  if (more && (!hasMore.value || status.value === 'loadingMore')) {
    return
  }
  status.value = more ? 'loadingMore' : 'loading'
  const nextPage = more ? page.value + 1 : 1

  const res = await kunFetch<{ items: GalgameCard[]; total: number }>(
    '/galgame/drafts',
    {
      method: 'GET',
      query: {
        page: nextPage,
        limit: LIMIT,
        [scopeParam.value]: props.entityId,
        original_language: 'ja-jp,zh-cn,zh-tw'
      }
    }
  )

  if (res === null) {
    status.value = 'error'
    return
  }

  items.value = more ? [...items.value, ...res.items] : res.items
  total.value = res.total
  page.value = nextPage
  status.value = 'idle'
}

watch(open, (isOpen) => {
  if (!isOpen) {
    return
  }
  items.value = []
  total.value = 0
  page.value = 0
  fetchPage(false)
})
</script>

<template>
  <KunModal v-model="open" inner-class-name="max-w-4xl w-[92vw]">
    <div class="flex max-h-[80dvh] flex-col gap-3 p-1">
      <div class="flex items-center gap-2">
        <KunIcon class="text-primary text-2xl" name="lucide:library-big" />
        <span class="text-lg font-bold">{{ entityName }} 的未发布游戏</span>
        <KunChip v-if="total" size="sm" variant="flat">{{ total }}</KunChip>
      </div>

      <p class="text-default-500 text-xs">
        以下 Galgame 已从 VNDB 收录并归入该{{ entityLabel }}, 但尚未在本站发布,
        点击任意一个即可前往发布页认领并成为创建者。
      </p>

      <KunLoading v-if="status === 'loading'" />

      <KunNull
        v-else-if="status === 'error'"
        description="加载失败, 请稍后再试"
      />

      <KunNull
        v-else-if="!items.length"
        :description="`该${entityLabel}暂无未发布的游戏`"
      />

      <div v-else class="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto">
        <GalgameCard :galgames="items" />

        <div class="flex justify-center pt-1">
          <KunButton
            v-if="hasMore"
            variant="light"
            :loading="status === 'loadingMore'"
            @click="fetchPage(true)"
          >
            加载更多
          </KunButton>
          <span v-else class="text-default-400 text-sm">没有更多了</span>
        </div>
      </div>
    </div>
  </KunModal>
</template>
