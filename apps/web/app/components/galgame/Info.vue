<script setup lang="ts">
import {
  KUN_GALGAME_AGE_LIMIT_MAP,
  getGalgameOriginalLanguageName
} from '~/constants/galgame'

defineProps<{
  official: GalgameOfficialItem[]
  engine: GalgameEngineItem[]
  originalLanguage: string
  ageLimit: 'all' | 'r18'
  // U1: nil/'' = unknown release; TBA flag is independent. Rendering
  // logic in shared/utils/getReleaseDateText.ts.
  releaseDate?: string | null
  releaseDateTBA?: boolean
}>()

// Merged "编辑历史 + 更新请求" launcher. The modal itself now lives once at the
// page root (Galgame.vue) so the pending-PR banner can share the same instance;
// this button just opens it on the 编辑历史 tab via the injected controller.
const activity = inject<{ open: boolean; tab: 'history' | 'pr' }>(
  'galgameActivity'
)
const openHistory = () => {
  if (!activity) {
    return
  }
  activity.tab = 'history'
  activity.open = true
}

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

      <div
        v-if="engine && engine.length > 0"
        class="border-default-200 dark:border-default-700/50"
      >
        <dt class="text-default-500 dark:text-default-400 text-sm font-medium">
          游戏引擎
        </dt>
        <dd
          class="text-default-900 mt-1.5 text-base font-medium dark:text-white"
        >
          <KunTooltip
            v-for="item in engine"
            :key="item.id"
            :text="`${item.galgame_count} 个 Galgame 使用此引擎制作`"
          >
            <KunLink
              :to="`/galgame-engine/${item.id}`"
              underline="none"
              class-name="text-foreground hover:text-primary text-base font-semibold"
            >
              {{ item.name }}
              <KunChip size="xs">
                {{ `+ ${item.galgame_count}` }}
              </KunChip>
            </KunLink>
          </KunTooltip>
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
      variant="flat"
      color="primary"
      size="sm"
      full-width
      @click="openHistory"
    >
      <KunIcon name="lucide:history" />
      查看编辑历史与更新请求
    </KunButton>
  </KunCard>
</template>
