export interface ReportTarget {
  subjectKind: string
  subjectId: string | number
  snapshot?: string
  subjectUrl?: string
}

const isOpen = ref(false)
const target = ref<ReportTarget | null>(null)

export const useReportModal = () => {
  const open = (t: ReportTarget) => {
    target.value = t
    isOpen.value = true
  }
  return { isOpen, target, open }
}
