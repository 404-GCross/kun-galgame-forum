<script setup lang="ts">
import { galgameIntroductionLanguageTabs } from '~/constants/galgame'

const introductionLanguage = ref<Language>('zh-cn')

const props = defineProps<{
  introduction: KunLanguage
  introductionMachine?: KunLanguageFlags
}>()

const isMachine = computed(
  () => !!props.introductionMachine?.[introductionLanguage.value as Language]
)
</script>

<template>
  <div class="space-y-3">
    <KunHeader
      name="游戏介绍"
      description="Galgame 的简体中文, 繁体中文, 英语, 日语 介绍。英语介绍来源于 VNDB, 日语介绍来源于游戏官网"
      scale="h2"
    >
      <template #endContent>
        <KunTab
          :items="galgameIntroductionLanguageTabs"
          v-model="introductionLanguage"
          size="sm"
          variant="underlined"
        />
      </template>
    </KunHeader>

    <div
      class="bg-primary/20 text-primary rounded-lg p-3"
      v-if="introduction[introductionLanguage as Language] === ''"
    >
      暂无对应翻译, 为您找到最近似的语言, 欢迎贡献翻译
    </div>
    <div v-else-if="isMachine" class="text-default-500 text-sm">
      本段简介由机器翻译生成, 与原文可能有出入
    </div>

    <KunContent
      class="pt-3"
      :content="
        renderKatex(
          getPreferredLanguageText(introduction, introductionLanguage)
        )
      "
    />
  </div>
</template>
