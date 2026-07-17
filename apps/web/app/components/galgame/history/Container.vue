<script setup lang="ts">
// The engine-backed revision history page (E3a): the full log — including
// the 11,860 migrated pre-engine rows with their honest legacy_action
// provenance — plus any-two-versions field diff.
import {
  GALGAME_EDIT_LEGACY_ACTION_LABELS,
  galgameEditFieldConfig,
  galgameEditLabel
} from '~/constants/galgameEdit'

const route = useRoute()
const gid = computed(() => parseInt((route.params as { gid: string }).gid))

useKunDisableSeo('Galgame 修订历史')

const { data, status } = await useKunFetch<GalgameEditRevisionList>(
  `/galgame/${gid.value}/edit/revisions`,
  { method: 'GET', watch: false, query: { limit: 200 } }
)

const diffOpen = ref(false)
const diffLoading = ref(false)
const diff = ref<GalgameEditDiff | null>(null)

const handleDiff = async (fromSeq: number, toSeq: number) => {
  diffLoading.value = true
  diffOpen.value = true
  diff.value = await kunFetch<GalgameEditDiff>(
    `/galgame/${gid.value}/edit/diff`,
    { method: 'GET', query: { from: fromSeq, to: toSeq } }
  )
  diffLoading.value = false
}
</script>

<template>
  <div class="mx-auto flex max-w-3xl flex-col gap-3">
    <KunCard :is-hoverable="false" :is-transparent="false" content-class="space-y-2">
      <KunHeader
        name="修订历史"
        description="该 Galgame 的完整编辑记录（含旧版百科迁移的历史修订），可任选两个版本对比差异"
        scale="h2"
      />
      <div class="flex gap-2">
        <KunButton
          variant="light"
          color="default"
          size="sm"
          @click="navigateTo(`/galgame/${gid}`)"
        >
          <KunIcon name="lucide:arrow-left" />
          返回游戏页
        </KunButton>
        <KunButton
          variant="light"
          color="default"
          size="sm"
          @click="navigateTo(`/galgame/${gid}/edit`)"
        >
          <KunIcon name="lucide:pencil" />
          编辑资料
        </KunButton>
      </div>
    </KunCard>

    <KunCard
      v-if="data"
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-3"
    >
      <EditkitRevisionTimeline
        :items="data.items"
        :users="data.users"
        :label-for="galgameEditLabel"
        :legacy-action-labels="GALGAME_EDIT_LEGACY_ACTION_LABELS"
        @diff="handleDiff"
      />
    </KunCard>

    <KunNull
      v-else-if="status !== 'pending'"
      description="无法加载修订历史（条目不存在或编辑服务暂不可用）"
    />

    <KunModal v-model="diffOpen" size="lg">
      <div class="space-y-4">
        <KunHeader
          :name="diff ? `版本对比 #${diff.from_seq} → #${diff.to_seq}` : '版本对比'"
          scale="h3"
        />
        <KunLoading v-if="diffLoading" />
        <template v-else-if="diff">
          <KunNull v-if="!diff.fields.length" description="两个版本没有差异" />
          <div v-else class="space-y-4">
            <EditkitFieldDiff
              v-for="row in diff.fields"
              :key="row.key"
              :label="galgameEditLabel(row.key)"
              :diff-hint="row.diff_hint"
              :from="row.from"
              :to="row.to"
              :config="galgameEditFieldConfig(row.key)"
            />
          </div>
        </template>
      </div>
    </KunModal>
  </div>
</template>
