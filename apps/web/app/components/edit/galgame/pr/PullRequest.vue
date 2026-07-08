<script setup lang="ts">
// PR edit form for a PUBLISHED galgame (POST /galgame/:gid/prs).
//
// Layout: top segmented KunTab groups the form into focused sections
// (basic / intro / relations / links / advanced) + a sticky bottom
// action bar (PR note + submit) that's always reachable. This declutters
// what used to be one long scroll while keeping EVERY field bound &
// submitted regardless of the active tab — panels use v-show (NOT v-if)
// so child state (alias chips, editor buffer, loaded selector lists)
// survives tab switches and the wiki replace-all relations never reset.
// Footer always submits the whole store, so unvisited tabs round-trip
// their pre-hydrated values unchanged.
//
// Single root is <div class="contents"> (page renders us bare). The
// galgamePR[0] guard handles the persist:false edge; ?type=pr arms the
// `prevent` middleware against hard refreshes (see Rewrite.vue).
import type { KunTabItem } from '@kungal/ui-vue'
import { languageItems } from '~/constants/edit'

const { galgamePR } = storeToRefs(useTempGalgamePRStore())
const pr = computed(() => galgamePR.value[0])

const sectionTabs: KunTabItem[] = [
  { value: 'basic', textValue: '基本信息', icon: 'lucide:info' },
  { value: 'intro', textValue: '游戏介绍', icon: 'lucide:file-text' },
  { value: 'relations', textValue: '标签 / 会社 / 引擎', icon: 'lucide:tags' },
  { value: 'links', textValue: '相关链接', icon: 'lucide:link' },
  { value: 'media', textValue: '封面 / 画廊', icon: 'lucide:image' },
  { value: 'advanced', textValue: '分级 / 高级', icon: 'lucide:settings-2' }
]
const activeSection = ref('basic')

const introductionLanguage = ref<Language>('zh-cn')

const ageLimitOptions = [
  { value: 'all', label: '全年龄' },
  { value: 'r18', label: 'R18' }
] as const

const originalLanguageOptions = [
  { value: 'ja-jp', label: '日语' },
  { value: 'en-us', label: '英语' },
  { value: 'zh-cn', label: '简体中文' },
  { value: 'zh-tw', label: '繁体中文' },
  { value: 'others', label: '其它' }
] as const
</script>

<template>
  <div class="contents">
    <ClientOnly>
      <KunCard
        :is-hoverable="false"
        :is-transparent="false"
        content-class="space-y-6"
      >
        <EditGalgamePrHelp />

        <KunInfo
          v-if="!pr"
          color="warning"
          title="无法编辑"
          description="请从 Galgame 详情页点击编辑按钮进入此页面。"
        />

        <template v-else>
          <KunInfo
            :color="pr!.can_direct_edit ? 'success' : 'primary'"
            :title="pr!.can_direct_edit ? '直接编辑模式' : '更新请求模式'"
            :description="
              pr!.can_direct_edit
                ? '你是创建者或管理员, 保存后立即生效并记录一条版本历史。'
                : '你的修改将作为更新请求提交, 由创建者或管理员审核合并后生效。'
            "
          />

          <KunTab
            :items="sectionTabs"
            v-model="activeSection"
            variant="solid"
            color="primary"
            size="md"
          />

          <div v-show="activeSection === 'basic'" class="space-y-6">
            <EditGalgamePrTitle />
            <EditGalgamePrAlias type="rewrite" />
          </div>

          <div v-show="activeSection === 'intro'" class="space-y-2">
            <KunTab
              :items="languageItems"
              v-model="introductionLanguage"
              variant="underlined"
              color="primary"
              size="sm"
            />
            <EditGalgameEditor :lang="introductionLanguage" type="rewrite" />
          </div>

          <div v-show="activeSection === 'relations'" class="space-y-6">
            <EditGalgamePrTagSelector />
            <EditGalgamePrOfficialSelector />
            <EditGalgamePrEngineSelector />
          </div>

          <div v-show="activeSection === 'links'" class="space-y-6">
            <EditGalgamePrLinks />
          </div>

          <div v-show="activeSection === 'advanced'" class="space-y-6">
            <EditGalgameContentLimit type="rewrite" />

            <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
              <KunSelect
                v-model="pr!.age_limit"
                label="年龄分级"
                :options="ageLimitOptions"
              />
              <KunSelect
                v-model="pr!.original_language"
                label="原始语言"
                :options="originalLanguageOptions"
              />
            </div>

            <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
              <KunInput
                v-model="pr!.release_date"
                type="date"
                label="发售日期 (留空=未公布)"
              />
              <KunSwitch
                v-model="pr!.release_date_tba"
                label="发售日期待定 (TBA)"
              />
            </div>

            <div class="space-y-2">
              <h2 class="text-xl">VNDB 编号</h2>
              <p class="text-default-500 text-sm">
                该作品在 VNDB 的编号, 无对应条目 (如原创 / 同人作品) 可留空。
              </p>
              <KunInput
                v-model="pr!.vndb_id"
                placeholder="例如: v19658 (可留空)"
              />
            </div>

          </div>

          <div v-show="activeSection === 'media'" class="space-y-6">
            <EditGalgamePrCovers />
            <EditGalgamePrScreenshots />
          </div>

          <div class="sticky bottom-2 z-10">
            <div
              class="border-default-200 bg-content1/90 flex flex-col gap-3 rounded-xl border p-3 shadow-lg backdrop-blur sm:flex-row sm:items-center"
            >
              <!-- Direct edit writes a revision (uses `note`); a PR uses
                   title + message. Show only what the submit endpoint
                   consumes — see Footer.vue + docs 02-revisions-and-prs. -->
              <KunInput
                v-if="pr!.can_direct_edit"
                v-model="pr!.note"
                class="flex-1"
                placeholder="版本说明 (可选): 简述本次修改了什么 / 来源依据"
              />
              <template v-else>
                <KunInput
                  v-model="pr!.title"
                  class="flex-1"
                  placeholder="更新请求标题 (可选)"
                />
                <KunInput
                  v-model="pr!.message"
                  class="flex-1"
                  placeholder="更新请求说明 (可选): 简述改了什么 / 来源依据, 便于审核"
                />
              </template>
              <EditGalgamePrFooter />
            </div>
          </div>
        </template>
      </KunCard>
    </ClientOnly>
  </div>
</template>
