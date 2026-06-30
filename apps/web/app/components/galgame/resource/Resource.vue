<script setup lang="ts">
import {
  GALGAME_RESOURCE_PROVIDER_BUCKETS,
  bucketizeResourceProvider,
  type GalgameResourceProviderBucketKey
} from '~/constants/galgameResource'

const route = useRoute()
const gid = computed(() => {
  return parseInt((route.params as { gid: string }).gid)
})

// Provided by the detail page (Galgame.vue). For a wiki-catalogue game the forum
// hasn't ingested, the parent already shows a 未收录 notice, so we suppress this
// section's empty-state — but KEEP the 添加资源 CTA (the funnel that records it).
const galgame = inject<GalgameDetail>('galgame')

// Publish-modal toggle is purely local to this page — no longer needs
// the (now removed) tempGalgameResource Pinia store because no other
// component reads / writes it.
const isShowPublish = ref(false)
const { id } = usePersistUserStore()

const { data, status, refresh } = await useKunFetch<GalgameResource[]>(
  `/galgame/${gid.value}/resource/all`,
  {
    lazy: true,
    method: 'GET',
    query: { galgameId: gid.value }
  }
)

// Group resources into the 7 user-facing provider buckets; skip empty
// buckets so the tablist collapses when a galgame only has, say, baidu
// + quark links. Each resource appears in exactly one bucket (its
// primary provider per bucketizeResourceProvider's first-match-wins
// rule).
const groupedResources = computed(() => {
  const grouped: Record<GalgameResourceProviderBucketKey, GalgameResource[]> = {
    baidu: [],
    quark: [],
    caiyun: [],
    pan123: [],
    xunlei: [],
    lanzou: [],
    other: []
  }
  for (const r of data.value ?? []) {
    grouped[bucketizeResourceProvider(r.providerNames)].push(r)
  }
  return GALGAME_RESOURCE_PROVIDER_BUCKETS.flatMap((bucket) => {
    const items = grouped[bucket.key]
    return items.length ? [{ ...bucket, items }] : []
  })
})

// KunTab items map: each non-empty bucket becomes a tab. textValue
// embeds the bucket label + count so the tab itself doubles as a
// section header.
const providerTabs = computed(() =>
  groupedResources.value.map((g) => ({
    value: g.key,
    textValue: `${g.label} (${g.items.length})`,
    icon: g.icon
  }))
)

// activeProvider follows the first non-empty bucket on data load, then
// the user's choice. Re-pinned to a still-existing bucket if a refresh
// removes the previously-selected one (e.g. all baidu links deleted).
const activeProvider = ref<GalgameResourceProviderBucketKey | ''>('')
watchEffect(() => {
  const first = groupedResources.value[0]?.key
  if (!first) {
    activeProvider.value = ''
    return
  }
  const stillExists = groupedResources.value.some(
    (g) => g.key === activeProvider.value
  )
  if (!stillExists) {
    activeProvider.value = first
  }
})

const activeBucket = computed(() =>
  groupedResources.value.find((g) => g.key === activeProvider.value)
)
</script>

<template>
  <div class="space-y-3">
    <KunHeader name="Galgame 资源链接" scale="h2">
      <template #headerEndContent>
        <div class="ml-auto flex items-center gap-1">
          <KunButton
            v-if="id"
            :href="`/user/${id}/resource/expire`"
            color="success"
            variant="flat"
          >
            批量更改已失效资源链接
          </KunButton>
          <KunButton @click="isShowPublish = !isShowPublish">
            添加资源
          </KunButton>
        </div>
      </template>

      <template #endContent>
        <KunInfo
          color="info"
          title="一些小提示以及帮助文档"
          description="部分资源链接可能需要网络代理"
        >
          <div class="my-1 text-sm">
            <span>如果您找不到想要的资源链接, 可以去看看友站</span>
            <KunLink
              class-name="inline whitespace-nowrap"
              size="sm"
              to="https://www.touchgal.us/"
              target="_blank"
            >
              TouchGal
            </KunLink>
            和
            <KunLink
              class-name="inline whitespace-nowrap"
              size="sm"
              to="https://zi6.cc/"
              target="_blank"
            >
              zi0
            </KunLink>
          </div>

          <div class="mb-1 flex items-center gap-1">
            <KunLink class-name="inline" size="sm" to="/topic/2431">
              Galgame萌新入门(待补充)
            </KunLink>
            - by
            <KunUserChip
              size="sm"
              :user="{
                id: 19994,
                name: '大伊兜子',
                avatar: 'https://image.kungal.com/avatar/user_19994/avatar.webp'
              }"
            />
          </div>

          <div class="flex items-center gap-1">
            <KunLink class-name="inline" size="sm" to="/topic/2522">
              如何安装镜像文件(教程)
            </KunLink>
            - by
            <KunUserChip
              size="sm"
              :user="{
                id: 19994,
                name: '大伊兜子',
                avatar: 'https://image.kungal.com/avatar/user_19994/avatar.webp'
              }"
            />
          </div>
        </KunInfo>
      </template>
    </KunHeader>

    <KunAdAIFYBanner />

    <KunNull
      v-if="!data?.length && galgame?.isOnForum !== false"
      description="这个 Galgame 还没有资源链接, 快添加一个吧!"
    />

    <GalgameResourceLinkEditModal
      v-model="isShowPublish"
      :galgame-id="gid"
      :refresh="refresh"
    />

    <template v-if="status !== 'pending' && data?.length">
      <KunTab
        v-if="providerTabs.length > 1"
        v-model="activeProvider"
        :items="providerTabs"
        variant="light"
        color="primary"
        size="md"
        scrollable
      />
      <div v-if="activeBucket" class="space-y-3">
        <GalgameResourceLink
          v-for="resource in activeBucket.items"
          :key="resource.id"
          :resource="resource"
          :refresh="refresh"
        />
      </div>
    </template>

    <KunLoading v-if="status === 'pending'" />
  </div>
</template>
