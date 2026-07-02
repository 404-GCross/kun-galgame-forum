<script setup lang="ts">
import {
  topicSortItem,
  galgameSortItem,
  userSortItem,
  rankingPageTabs,
  rankingPageMetaData
} from '~/constants/ranking'
import {
  topicRankingPageData,
  galgameRankingPageData,
  userRankingPageData
} from '~/components/ranking/pageData'

const activeTab = computed(() => useRoute().fullPath.split('/').pop() ?? 'user')

const currentSortItems = computed(() => {
  switch (activeTab.value) {
    case 'topic':
      return topicSortItem
    case 'galgame':
      return galgameSortItem
    case 'user':
    default:
      return userSortItem
  }
})

const sortOptions = computed(() => {
  return currentSortItems.value.map((item) => ({
    value: item.sortField,
    label: item.label,
    icon: item.icon
  }))
})
</script>

<template>
  <div class="space-y-3">
    <div class="space-y-3">
      <KunHeader
        :name="rankingPageMetaData[activeTab]!.title"
        :description="rankingPageMetaData[activeTab]!.description"
      />

      <div class="flex items-center justify-between">
        <KunTab
          :model-value="activeTab"
          :items="rankingPageTabs"
          variant="underlined"
          color="primary"
        />

        <div class="w-48">
          <KunSelect
            v-if="activeTab === 'topic'"
            v-model="topicRankingPageData.sort_field"
            :options="sortOptions"
          />
          <KunSelect
            v-if="activeTab === 'galgame'"
            v-model="galgameRankingPageData.sort_field"
            :options="sortOptions"
          />
          <KunSelect
            v-if="activeTab === 'user'"
            v-model="userRankingPageData.sort_field"
            :options="sortOptions"
          />
        </div>
      </div>
    </div>

    <NuxtPage />
  </div>
</template>
