<script setup lang="ts">
// Report entry point mounted at every reportable content site. It only TRIGGERS
// the single global report modal (useReportModal → ReportModal at app root), so
// it's safe to place inside a ⋯ KunPopover without the modal being torn down
// when the popover closes. A new content type costs one <ReportButton> mount
// with its subject_kind + subject_id (+ optional snapshot for evidence).
const props = withDefaults(
  defineProps<{
    subjectKind: string
    subjectId: string | number
    snapshot?: string
    label?: string
    // `menu` renders a full-width row (for ⋯ popover menus); default is a
    // compact icon-only button (for action bars).
    menu?: boolean
  }>(),
  { snapshot: '', label: '举报', menu: false }
)

const userStore = usePersistUserStore()
const { open } = useReportModal()

const trigger = () => {
  if (!userStore.id) {
    useAuthModal().open()
    return
  }
  open({
    subjectKind: props.subjectKind,
    subjectId: props.subjectId,
    snapshot: props.snapshot
  })
}
</script>

<template>
  <KunButton
    v-if="menu"
    variant="light"
    color="danger"
    size="sm"
    class-name="w-full justify-start gap-2 whitespace-nowrap"
    @click="trigger"
  >
    <KunIcon class-name="text-lg" name="lucide:flag" />{{ label }}
  </KunButton>

  <KunTooltip v-else :text="label">
    <KunButton
      :is-icon-only="true"
      color="danger"
      variant="light"
      size="sm"
      @click="trigger"
    >
      <KunIcon name="lucide:flag" />
    </KunButton>
  </KunTooltip>
</template>
