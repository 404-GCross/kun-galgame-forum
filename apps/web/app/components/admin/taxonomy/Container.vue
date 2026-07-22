<script setup lang="ts">
// Staff console for the four wiki taxonomy entities (E3b: the minimal parity
// transplant of apps/wiki's tag/official/engine/series management). The whole
// console is admin-only — the /admin/taxonomy page is admin-gated (middleware),
// so only admin ⊂ ren reach it. Editing / deleting / reverting existing entries
// is additionally enforced by the API's RequireAdmin gate. CREATING entries is
// open to any logged-in user at the API (the public /galgame-series page + the
// "add a missing tag" contribution flow rely on it) — the create controls just
// live inside this admin-only console.
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
      name="资料库条目管理"
      description="集中管理 Galgame 的标签 / 会社 / 引擎 / 系列条目。仅管理员（admin / ren）可进入；编辑与删除由服务端强制校验。"
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
