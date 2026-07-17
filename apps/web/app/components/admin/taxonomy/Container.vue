<script setup lang="ts">
// Staff console for the four wiki taxonomy entities (E3b: the minimal parity
// transplant of apps/wiki's tag/official/engine/series management). CRUD
// proxies + permission enforcement live on the wiki side (update/delete =
// moderator+, create = any logged-in user); this UI is UX gating only.
type TaxonomyTab = 'tag' | 'official' | 'engine' | 'series'

const activeTab = ref<TaxonomyTab>('tag')

const tabItems = [
  { value: 'tag', textValue: '标签' },
  { value: 'official', textValue: '会社' },
  { value: 'engine', textValue: '引擎' },
  { value: 'series', textValue: '系列' }
]
</script>

<template>
  <div class="w-full space-y-6">
    <KunHeader
      name="Wiki 条目管理"
      description="集中管理 Galgame 的标签 / 会社 / 引擎 / 系列条目。编辑与删除需要管理员权限, 由 Wiki 服务端强制校验。"
    />

    <KunTab
      :model-value="activeTab"
      :items="tabItems"
      variant="underlined"
      color="primary"
      size="md"
      @update:model-value="(value) => (activeTab = value as TaxonomyTab)"
    />

    <AdminTaxonomyTag v-if="activeTab === 'tag'" />
    <AdminTaxonomyOfficial v-if="activeTab === 'official'" />
    <AdminTaxonomyEngine v-if="activeTab === 'engine'" />
    <AdminTaxonomySeries v-if="activeTab === 'series'" />
  </div>
</template>
