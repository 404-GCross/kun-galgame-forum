const reasons = ref<ReportReason[]>([])
const loaded = ref(false)

export const useReportReasons = () => {
  const load = async () => {
    if (loaded.value) {
      return
    }
    const result = await kunFetch<ReportReason[]>('/report/reasons')
    if (result) {
      reasons.value = result
      loaded.value = true
    }
  }
  return { reasons, load }
}
