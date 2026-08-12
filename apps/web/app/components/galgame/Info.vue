<script setup lang="ts">
import {
  KUN_GALGAME_AGE_LIMIT_MAP,
  getGalgameOriginalLanguageName
} from '~/constants/galgame'

defineProps<{
  official: GalgameOfficialItem[]
  engine: GalgameEngineItem[]
  series: GalgameDetailSeriesRef[]
  originalLanguage: string
  ageLimit: 'all' | 'r18'
  releaseDate?: string | null
  releaseDateTBA?: boolean
}>()

const galgame = inject<GalgameDetail>('galgame')

const getLanguageName = getGalgameOriginalLanguageName
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    class-name="overflow-visible"
    content-class="space-y-3"
  >
    <dl class="space-y-5">
      <GalgameDetailOfficial :official="official" />

      <div v-if="series && series.length > 0">
        <dt class="text-default-500 dark:text-default-400 text-sm font-medium">
          所属系列
        </dt>
        <dd
          class="text-default-900 mt-1.5 flex flex-wrap items-center gap-x-3 text-base font-medium dark:text-white"
        >
          <KunLink
            v-for="item in series"
            :key="item.id"
            :to="`/galgame/series/${item.id}`"
            underline="none"
            class-name="text-foreground hover:text-primary text-base font-semibold"
          >
            {{ item.name }}
          </KunLink>
        </dd>
      </div>

      <div
        v-if="engine && engine.length > 0"
        class="border-default-200 dark:border-default-700/50"
      >
        <dt class="text-default-500 dark:text-default-400 text-sm font-medium">
          游戏引擎
        </dt>
        <dd
          class="text-default-900 mt-1.5 flex flex-wrap items-center gap-x-3 text-base font-medium dark:text-white"
        >
          <KunLink
            v-for="item in engine"
            :key="item.id"
            :to="`/galgame/engine/${item.id}`"
            underline="none"
            class-name="text-foreground hover:text-primary text-base font-semibold"
          >
            {{ item.name }}
            <KunTooltip
              v-if="item.galgame_count > 0"
              :text="`${item.galgame_count} 个 Galgame 使用此引擎制作`"
            >
              <KunChip size="xs">
                {{ `+ ${item.galgame_count}` }}
              </KunChip>
            </KunTooltip>
          </KunLink>
        </dd>
      </div>

      <div class="flex items-center justify-between">
        <dt class="text-default-500 text-sm font-medium">游戏原语言</dt>
        <dd>
          <KunChip color="warning">
            {{ getLanguageName(originalLanguage) }}
          </KunChip>
        </dd>
      </div>

      <div class="flex items-center justify-between">
        <dt class="text-default-500 text-sm font-medium">发售日期</dt>
        <dd class="text-default-700 text-sm">
          {{ getReleaseDateText(releaseDate, releaseDateTBA) }}
        </dd>
      </div>

      <div class="flex items-center justify-between">
        <dt class="text-default-500 dark:text-default-400 text-sm font-medium">
          年龄限制
        </dt>
        <dd>
          <KunTooltip
            position="left"
            :text="KUN_GALGAME_AGE_LIMIT_MAP[ageLimit]"
          >
            <KunChip
              variant="flat"
              :color="ageLimit === 'all' ? 'success' : 'danger'"
            >
              {{ ageLimit === 'all' ? '全年龄' : 'R18' }}
            </KunChip>
          </KunTooltip>
        </dd>
      </div>
    </dl>

    <KunButton
      v-if="galgame"
      variant="flat"
      color="primary"
      size="sm"
      full-width
      @click="navigateTo(`/galgame/${galgame.id}/history`)"
    >
      <KunIcon name="lucide:history" />
      修订历史
    </KunButton>
  </KunCard>
</template>
