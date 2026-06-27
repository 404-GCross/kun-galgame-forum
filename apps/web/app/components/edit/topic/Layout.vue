<script setup lang="ts">
const tempStore = useTempEditStore()

onBeforeRouteLeave(async () => {
  // Vue Router 4: return a value instead of calling next() (deprecated).
  // false = stay; anything else = proceed.
  if (tempStore.isTopicRewriting) {
    const res =
      await useComponentMessageStore().alert(
        '确认离开界面吗？您的更改将不会保存。'
      )
    if (!res) {
      return false
    }
    tempStore.resetRewriteTopicData()
  }
})
</script>

<template>
  <div class="contents">
    <ClientOnly>
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
        <KunCard
          :is-hoverable="false"
          :is-transparent="false"
          class-name="lg:col-span-1 order-2 sm:order-1"
          content-class="space-y-6"
        >
          <EditTopicMetadataEditor />
          <EditTopicSpecialNotice />
          <EditTopicSubmitActions />
        </KunCard>

        <div class="order-1 space-y-3 sm:order-2 lg:col-span-2">
          <KunCard :is-hoverable="false" :is-transparent="false">
            <EditTopicTitle />
          </KunCard>
          <KunCard :is-hoverable="false" :is-transparent="false">
            <EditTopicEditor />
          </KunCard>
          <KunCard :is-hoverable="false" :is-transparent="false">
            <EditTopicCoverPicker />
          </KunCard>
        </div>
      </div>
    </ClientOnly>
  </div>
</template>
