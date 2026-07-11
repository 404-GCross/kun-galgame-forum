<script setup lang="ts">
// Shared report entry point. The ONE component mounted at every reportable
// content site — a new content type costs just a `<ReportButton>` mount with
// its subject_kind + subject_id (+ an optional snapshot for evidence). Reasons
// come from the trust registry via GET /report/reasons; the reporter is the
// session user (added server-side). Reports are anonymous to other users but
// the platform records the reporter to curb abuse.
const props = withDefaults(
  defineProps<{
    subjectKind: string
    subjectId: string | number
    // A short text snapshot of the reported content (title / body) stored by
    // trust as evidence — the moderator sees it without dereferencing kungal.
    snapshot?: string
    // Tooltip / aria label.
    label?: string
    // `menu` renders a full-width row (for ⋯ popover menus); default is a
    // compact icon-only button (for action bars).
    menu?: boolean
  }>(),
  { snapshot: '', label: '举报', menu: false }
)

const userStore = usePersistUserStore()
const { reasons, load } = useReportReasons()

const isOpen = ref(false)
const reasonKey = ref('')
const note = ref('')
const isSubmitting = ref(false)

const reasonOptions = computed(() =>
  reasons.value.map((r) => ({ value: r.key, label: r.label }))
)

const openModal = () => {
  if (!userStore.id) {
    useAuthModal().open()
    return
  }
  load()
  reasonKey.value = ''
  note.value = ''
  isOpen.value = true
}

const submit = async () => {
  if (!reasonKey.value) {
    useMessage('请选择举报理由', 'warn')
    return
  }
  isSubmitting.value = true
  const result = await kunFetch('/report/submit', {
    method: 'POST',
    body: {
      subject_kind: props.subjectKind,
      subject_id: String(props.subjectId),
      reason_key: reasonKey.value,
      note: note.value,
      // Cap evidence length (BFF validates snapshot ≤ 2000).
      snapshot: props.snapshot.slice(0, 1000)
    }
  })
  isSubmitting.value = false
  if (result) {
    useMessage('举报已提交，感谢你的反馈', 'success')
    isOpen.value = false
  }
}
</script>

<template>
  <KunButton
    v-if="menu"
    variant="light"
    color="danger"
    size="sm"
    class-name="w-full justify-start gap-2 whitespace-nowrap"
    @click="openModal"
  >
    <KunIcon class-name="text-lg" name="lucide:flag" />{{ label }}
  </KunButton>

  <KunTooltip v-else :text="label">
    <KunButton
      :is-icon-only="true"
      color="danger"
      variant="light"
      size="sm"
      @click="openModal"
    >
      <KunIcon name="lucide:flag" />
    </KunButton>
  </KunTooltip>

  <KunModal v-model="isOpen" inner-class-name="max-w-md w-[92vw]">
    <div class="space-y-4">
      <div>
        <span class="text-xl">举报内容</span>
        <p class="text-default-500 text-sm">
          举报对其他用户匿名；为防止滥用，平台会记录举报人。请选择理由并按需补充说明。
        </p>
      </div>

      <KunSelect
        v-model="reasonKey"
        :options="reasonOptions"
        label="举报理由"
        placeholder="请选择举报理由"
      />

      <div class="space-y-2">
        <span class="text-default-600 text-sm font-medium">补充说明（可选）</span>
        <KunTextarea
          name="report-note"
          placeholder="补充违规的具体位置或情况，便于审核"
          :rows="4"
          v-model="note"
        />
      </div>

      <div class="flex justify-end gap-2">
        <KunButton variant="light" color="default" @click="isOpen = false">
          取消
        </KunButton>
        <KunButton color="danger" :loading="isSubmitting" @click="submit">
          提交举报
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
