export const useGalgameOfficialDetail = async (
  query: Record<string, unknown>,
  subPath = ''
) => {
  const route = useRoute()
  const officialId = Number((route.params as { id: string }).id)

  if (!Number.isInteger(officialId) || officialId <= 0) {
    throw createError({
      statusCode: 404,
      statusMessage: '未找到 Galgame 会社',
      fatal: true
    })
  }

  const { data, status } = await useKunFetch<GalgameOfficialDetail>(
    `/galgame-official/${officialId}`,
    { method: 'GET', query }
  )

  if (data.value?.moved_to) {
    await navigateTo(
      `${taxonomyDetailPath('official', data.value.moved_to)}${subPath}`,
      { redirectCode: 301, replace: true }
    )
  }

  if (!data.value) {
    throw createError({
      statusCode: 404,
      statusMessage: '未找到 Galgame 会社',
      fatal: true
    })
  }

  return { officialId, data, status, moved: !!data.value.moved_to }
}
