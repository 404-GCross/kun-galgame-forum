<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import {
  quizCategoryOptions,
  quizTypeOptions,
  quizDifficultyOptions,
  quizSortFieldOptions
} from './_filters'

const userStore = usePersistUserStore()
const isLoggedIn = computed(() => !!userStore.id)

const tab = ref<'all' | 'mine'>('all')
const params = reactive({
  page: 1,
  limit: 24,
  sort_field: 'time',
  sort_order: 'desc',
  category: 'all',
  type: 'all',
  difficulty: 0
})

// A ref URL so switching tabs re-targets the endpoint (useKunFetch re-fetches
// on url / query change). `mine` is self-only and ignores the filters.
const requestUrl = computed(() =>
  tab.value === 'mine' ? '/galgame-quiz/mine/answered' : '/galgame-quiz/all'
)

const { data, status, refresh } = await useKunFetch<QuizListPage>(requestUrl, {
  method: 'GET',
  query: params
})

const tabItems: KunTabItem[] = [
  { value: 'all', textValue: '全部题库', icon: 'lucide:library-big' },
  { value: 'mine', textValue: '我的答题', icon: 'lucide:history' }
]
const onTab = (v: string) => {
  // Reset page + switch tab in one tick so useKunFetch fires a single request.
  params.page = 1
  tab.value = v as 'all' | 'mine'
}

const showPublish = ref(false)
const openPublish = () => {
  if (!requireLogin()) return
  showPublish.value = true
}
const onPublished = () => {
  params.page = 1
  tab.value = 'all'
  refresh()
}
</script>

<template>
  <div class="space-y-3">
    <div class="space-y-2">
      <KunHeader name="Galgame 题库">
        <template #description>
          <p class="text-default-500">
            由萌友共同建设的 Galgame 题库, 支持单选、多选、判断、填空、问答等题型。答对可获得萌萌点, 出题合格同样有奖励。
          </p>
        </template>
        <template #endContent>
          <KunButton @click="openPublish">
            <span class="flex items-center gap-1">
              <KunIcon name="lucide:plus" />
              出题
            </span>
          </KunButton>
        </template>
      </KunHeader>

      <KunTab
        v-if="isLoggedIn"
        :model-value="tab"
        :items="tabItems"
        variant="light"
        color="primary"
        @update:model-value="onTab"
      />

      <div
        v-if="tab === 'all'"
        class="flex w-full shrink-0 flex-wrap items-center justify-between gap-3 sm:flex-nowrap"
      >
        <div class="grid w-full grid-cols-2 gap-3 lg:grid-cols-4">
          <KunSelect v-model="params.category" :options="quizCategoryOptions" />
          <KunSelect v-model="params.type" :options="quizTypeOptions" />
          <KunSelect
            v-model="params.difficulty"
            :options="quizDifficultyOptions"
          />
          <KunSelect
            v-model="params.sort_field"
            :options="quizSortFieldOptions"
          />
        </div>

        <div class="flex items-center gap-2">
          <KunButton
            :is-icon-only="true"
            :variant="params.sort_order === 'desc' ? 'flat' : 'light'"
            size="lg"
            @click="params.sort_order = 'desc'"
          >
            <KunIcon class="text-inherit" name="lucide:arrow-down" />
          </KunButton>
          <KunButton
            :is-icon-only="true"
            :variant="params.sort_order === 'asc' ? 'flat' : 'light'"
            size="lg"
            @click="params.sort_order = 'asc'"
          >
            <KunIcon class="text-inherit" name="lucide:arrow-up" />
          </KunButton>
        </div>
      </div>
    </div>

    <GalgameQuizCard
      v-if="data && data.quiz_data.length"
      :quizzes="data.quiz_data"
      :is-transparent="false"
    />
    <KunNull
      v-else-if="status !== 'pending'"
      :description="
        tab === 'mine' ? '你还没有作答过任何题目' : '还没有人出题, 快来出第一题吧'
      "
    />

    <KunPagination
      v-if="(data?.total || 0) > params.limit"
      v-model:current-page="params.page"
      :total-page="Math.ceil((data?.total || 0) / params.limit)"
      :is-loading="status === 'pending'"
    />

    <GalgameQuizPublish v-model="showPublish" @on-published="onPublished" />
  </div>
</template>
