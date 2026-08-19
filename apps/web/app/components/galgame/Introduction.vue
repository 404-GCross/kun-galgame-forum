<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import { getGalgameIntroLanguageName } from '~/constants/galgame'

const props = defineProps<{
  introduction: GalgameIntro[]
}>()

const tabs = computed<KunTabItem[]>(() =>
  props.introduction.map((intro) => ({
    textValue: getGalgameIntroLanguageName(intro.lang),
    value: intro.lang
  }))
)

const language = ref(props.introduction[0]?.lang ?? '')

watch(
  () => props.introduction,
  (rows) => {
    if (!rows.some((row) => row.lang === language.value)) {
      language.value = rows[0]?.lang ?? ''
    }
  }
)

const current = computed(() =>
  props.introduction.find((intro) => intro.lang === language.value)
)
</script>

<template>
  <div class="space-y-3">
    <KunHeader
      name="游戏介绍"
      description="英语介绍来源于 VNDB, 日语介绍来源于游戏官网, 中文介绍来自 NextMoe 资料库"
      scale="h2"
    >
      <template #endContent>
        <KunTab
          v-if="tabs.length > 1"
          :items="tabs"
          v-model="language"
          size="sm"
          variant="underlined"
        />
      </template>
    </KunHeader>

    <div v-if="!current" class="bg-primary/20 text-primary rounded-lg p-3">
      暂无简介, 欢迎贡献
    </div>

    <template v-else>
      <div v-if="current.machine" class="text-default-500 text-sm">
        本段简介由机器翻译生成, 与原文可能有出入
      </div>

      <KunContent class="pt-3" :content="renderKatex(current.intro)" />
    </template>
  </div>
</template>
