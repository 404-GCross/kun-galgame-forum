<script setup lang="ts">
// The schema-driven galgame editor (E3a): bootstrap = the entity-aware
// capability projection + current values; the form emits only the dirty
// subset; submission files an edit proposal into the kungal review queue
// (nothing automerges on kungal — the outcome message reflects both states).
import {
  createGalgameEditConfig,
  GALGAME_EDIT_GROUP_ORDER,
  galgameEditLabel,
  type GalgameEditNames
} from '~/constants/galgameEdit'

const route = useRoute()
const gid = computed(() => parseInt((route.params as { gid: string }).gid))

useKunDisableSeo('编辑 Galgame 资料')

const { data: bootstrap, status } = await useKunFetch<GalgameEditBootstrap>(
  `/galgame/${gid.value}/edit/bootstrap`,
  { method: 'GET', watch: false }
)

// The galgame detail already ships tag/official/engine/series resolved with
// NAMES — we build id→name maps off it so the relation pickers show names for
// the CURRENT values (new picks carry their own names from search).
const { data: detail } = await useKunFetch<GalgameDetail>(
  `/galgame/${gid.value}`,
  { method: 'GET', watch: false }
)
const editNames = computed<GalgameEditNames>(() => {
  const d = detail.value
  const toMap = (arr?: { id: number; name: string }[]) =>
    new Map((arr ?? []).map((x) => [x.id, x.name]))
  return {
    tag: toMap(d?.tag),
    official: toMap(d?.official),
    engine: toMap(d?.engine),
    series: d?.series ? new Map([[d.series.id, d.series.name]]) : new Map()
  }
})
const editConfig = computed(() => createGalgameEditConfig(editNames.value))

const { data: mine, refresh: refreshMine } =
  await useKunFetch<GalgameEditProposalList>('/galgame-edit/mine', {
    method: 'GET',
    watch: false,
    query: { gid: gid.value }
  })

// Open proposals on this game (public read). For a reviewer — a moderator OR
// the game's creator (owner-review, E3b) — the ones from other users render
// as an inline review surface linking to the workbench.
const { data: pending, refresh: refreshPending } =
  await useKunFetch<GalgameEditProposalList>(
    `/galgame/${gid.value}/edit/proposals`,
    { method: 'GET', watch: false }
  )

const userStore = usePersistUserStore()
const reviewable = computed(() => {
  if (!bootstrap.value?.can_review) {
    return []
  }
  return (pending.value?.items ?? []).filter(
    (p) => p.status === 'open' && p.proposer_uid !== userStore.id
  )
})

const { canModerate } = useRole()

const gameName = computed(() => {
  const values = bootstrap.value?.values ?? {}
  const name: KunLanguage = {
    'en-us': String(values['galgame.game.name_en_us'] ?? ''),
    'ja-jp': String(values['galgame.game.name_ja_jp'] ?? ''),
    'zh-cn': String(values['galgame.game.name_zh_cn'] ?? ''),
    'zh-tw': String(values['galgame.game.name_zh_tw'] ?? '')
  }
  return getPreferredLanguageText(name)
})

const patch = ref<Record<string, unknown>>({})
const note = ref('')
const showNote = ref(false)
const submitting = ref(false)

const dirtyCount = computed(() => Object.keys(patch.value).length)

const handleSubmit = async () => {
  if (!dirtyCount.value || submitting.value) {
    return
  }
  submitting.value = true
  const result = await kunFetch<GalgameEditSubmitResult>(
    `/galgame/${gid.value}/edit/proposals`,
    { method: 'POST', body: { patch: patch.value, note: note.value } }
  )
  submitting.value = false
  if (!result) {
    return
  }
  if (result.merged) {
    useMessage('修改已生效', 'success')
  } else {
    useMessage('提案已提交，等待审核', 'success')
  }
  note.value = ''
  await Promise.all([refreshMine(), refreshPending()])
}

const withdrawing = ref(false)
const handleWithdraw = async (id: number) => {
  if (withdrawing.value) {
    return
  }
  withdrawing.value = true
  const result = await kunFetch<GalgameEditProposal>(
    `/galgame-edit/proposals/${id}/withdraw`,
    { method: 'POST' }
  )
  withdrawing.value = false
  if (result) {
    useMessage('提案已撤回', 'success')
    await refreshMine()
  }
}
</script>

<template>
  <div class="flex w-full flex-col gap-3">
    <template v-if="bootstrap">
      <KunCard :is-hoverable="false" :is-transparent="false" content-class="space-y-2">
        <KunHeader
          :name="`编辑资料 · ${gameName}`"
          description="提交的修改将进入审核队列，审核通过后生效；审核人可在合并前修正您的提案（双方都会署名）"
          scale="h2"
        />
        <div class="flex flex-wrap gap-2">
          <KunButton
            variant="light"
            color="default"
            size="sm"
            @click="navigateTo(`/galgame/${gid}`)"
          >
            <KunIcon name="lucide:arrow-left" />
            返回游戏页
          </KunButton>
          <KunButton
            variant="light"
            color="default"
            size="sm"
            @click="navigateTo(`/galgame/${gid}/history`)"
          >
            <KunIcon name="lucide:history" />
            修订历史
          </KunButton>
          <KunButton
            v-if="canModerate"
            variant="light"
            color="primary"
            size="sm"
            @click="navigateTo('/galgame-edit/review')"
          >
            <KunIcon name="lucide:list-checks" />
            审核队列
          </KunButton>
        </div>
      </KunCard>

      <!-- Open proposals from OTHER users, for reviewers: moderators and the
           game's creator (owner-review, E3b) adjudicate in the workbench -->
      <div v-if="reviewable.length" class="space-y-2">
        <EditkitProposalCard
          v-for="item in reviewable"
          :key="item.id"
          :proposal="item"
          :label-for="galgameEditLabel"
          :proposer="pending?.users?.[item.proposer_uid]"
        >
          <template #title>
            <span class="text-default-700 text-sm font-medium">
              待审提案 #{{ item.id }}
            </span>
          </template>
          <template #actions>
            <KunButton
              variant="flat"
              color="primary"
              size="sm"
              @click="navigateTo(`/galgame-edit/review/${item.id}`)"
            >
              审阅
            </KunButton>
          </template>
        </EditkitProposalCard>
      </div>

      <!-- Pending proposals of MINE on this galgame -->
      <div v-if="mine?.items.length" class="space-y-2">
        <EditkitProposalCard
          v-for="item in mine.items.filter((p) => p.status === 'open')"
          :key="item.id"
          :proposal="item"
          :label-for="galgameEditLabel"
        >
          <template #title>
            <span class="text-default-700 text-sm font-medium">
              我的待审提案 #{{ item.id }}
            </span>
          </template>
          <template #actions>
            <KunButton
              variant="flat"
              color="danger"
              size="sm"
              :loading="withdrawing"
              @click="handleWithdraw(item.id)"
            >
              撤回
            </KunButton>
          </template>
        </EditkitProposalCard>
      </div>

      <KunCard :is-hoverable="false" :is-transparent="false">
        <EditkitSchemaForm
          :fields="bootstrap.fields"
          :values="bootstrap.values"
          :config="editConfig"
          :group-order="GALGAME_EDIT_GROUP_ORDER"
          :disabled="submitting"
          layout="tabs"
          @update:patch="patch = $event"
        />
      </KunCard>

      <!-- Sticky save bar: keeps submit reachable across the tabbed form, with
           an optional edit note. -->
      <div class="sticky bottom-0 z-20 pb-3">
        <KunCard
          :is-hoverable="false"
          :is-transparent="false"
          content-class="space-y-2"
        >
          <KunTextarea
            v-if="showNote"
            v-model="note"
            label="编辑说明（可选）"
            placeholder="说明这次修改的内容与依据，帮助审核人更快处理"
            :maxlength="2000"
          />
          <div class="flex items-center justify-between gap-3">
            <span class="text-default-500 text-sm">
              {{ dirtyCount ? `已修改 ${dirtyCount} 个字段` : '尚未修改' }}
            </span>
            <div class="flex items-center gap-2">
              <KunButton
                variant="light"
                color="default"
                size="sm"
                @click="showNote = !showNote"
              >
                <KunIcon name="lucide:pencil-line" />
                编辑说明
              </KunButton>
              <KunButton
                color="primary"
                :disabled="!dirtyCount"
                :loading="submitting"
                @click="handleSubmit"
              >
                提交提案
              </KunButton>
            </div>
          </div>
        </KunCard>
      </div>
    </template>

    <KunNull
      v-else-if="status !== 'pending'"
      description="无法加载编辑数据（条目不存在或编辑服务暂不可用）"
    />
  </div>
</template>
