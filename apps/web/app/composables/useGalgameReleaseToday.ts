export const useGalgameReleaseToday = () => {
  const { data } = useKunFetch<GalgameCalendarMonth>('/galgame/calendar', {
    key: 'galgame-release-today',
    server: false
  })

  const hasReleaseToday = computed(
    () =>
      !!data.value?.items.some(
        (g) =>
          g.release_precision === 'day' && g.release_date === data.value!.today
      )
  )

  return { hasReleaseToday }
}
