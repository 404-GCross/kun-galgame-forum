<script setup lang="ts">
import {
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP,
  KUN_GALGAME_OFFICIAL_LANGUAGE_MAP
} from '~/constants/galgameOfficial'

defineProps<{
  official: GalgameOfficialItem[]
}>()

// The detail chip now speaks the label's OWN kind (游戏品牌 / 同人社团 / …),
// which is the same vocabulary the /galgame/official index renders — one map,
// one wording. Before A2-R2 it printed the per-edge ROLE, which had no Chinese
// entry anywhere and fell through to the raw English "developer".
const getCategoryText = (category: string) =>
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP[category] || category
</script>

<template>
  <div>
    <dt class="text-default-500 text-sm font-medium">制作方</dt>
    <dd class="mt-1.5 space-y-3">
      <div class="space-y-2" v-for="item in official" :key="item.id">
        <KunLink
          :to="`/galgame/official/${item.id}`"
          underline="none"
          class-name="text-foreground hover:text-primary text-base font-semibold"
        >
          {{ item.name }}
          <!-- The badge (and its explanation) exist only when there is a real
               count: an upstream that has not published work_count yet, or a
               会社 with no other work, must render nothing rather than "+ 0". -->
          <KunTooltip
            v-if="item.galgame_count > 0"
            :text="`该会社制作了 ${item.galgame_count} 个 Galgame`"
          >
            <KunChip size="xs">
              {{ `+ ${item.galgame_count}` }}
            </KunChip>
          </KunTooltip>
        </KunLink>

        <div class="mt-1 flex items-center justify-between">
          <div class="flex items-center gap-x-2">
            <KunChip size="xs" class-name="rounded-md" color="default">
              {{ getCategoryText(item.category) }}
            </KunChip>
            <span class="text-default-500 dark:text-default-400 text-xs">
              {{ KUN_GALGAME_OFFICIAL_LANGUAGE_MAP[item.lang] || item.lang }}
            </span>
          </div>

          <!-- Only a real anchor when there's a site to go to. An empty
               item.link used to render a link-styled element that looked
               clickable but went nowhere — confusing. No site → a plain
               muted label instead. -->
          <KunLink
            v-if="item.link"
            :is-show-anchor-icon="true"
            target="_blank"
            :to="item.link"
            size="sm"
            underline="hover"
            rel="noopener noreferrer"
          >
            官方网站
          </KunLink>
          <span v-else class="text-default-400 text-xs"> 暂无官网 </span>
        </div>

        <div
          v-if="item.alias.length"
          class="text-default-500 flex flex-wrap gap-2"
        >
          <KunChip
            size="xs"
            color="success"
            v-for="(a, index) in item.alias"
            :key="index"
          >
            {{ a }}
          </KunChip>
        </div>
      </div>
    </dd>
  </div>
</template>
