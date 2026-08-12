<script setup lang="ts">
import { reactionAsset } from '~/constants/reaction'

const props = defineProps<{ topicId?: number; replyId?: number }>()

const isReply = computed(() => !!props.replyId)
const endpoint = computed(() =>
  props.replyId
    ? `/topic/0/reply/reaction/history?reply_id=${props.replyId}`
    : `/topic/${props.topicId}/reaction/history`
)
const open = defineModel<boolean>({ required: true })

interface ReactionHistoryItem {
  user: { id: number; name: string; avatar: string }
  reaction: string
  created: string
}

const records = ref<ReactionHistoryItem[]>([])
const isLoading = ref(false)
let loaded = false

watch(open, async (isOpen) => {
  if (!isOpen || loaded) return
  isLoading.value = true
  const result = await kunFetch<ReactionHistoryItem[]>(endpoint.value, {
    method: 'GET'
  })
  isLoading.value = false
  if (result) {
    records.value = result
    loaded = true
  }
})
</script>

<template>
  <KunModal v-model="open" inner-class-name="max-w-md w-[92vw]">
    <div class="space-y-3">
      <h3 class="text-lg font-bold">
        {{ isReply ? '回复回应历史' : '话题回应历史' }}
      </h3>

      <div v-if="isLoading" class="flex justify-center py-8">
        <KunLoading />
      </div>

      <KunNull
        v-else-if="!records.length"
        :description="isReply ? '还没有人回应这条回复' : '还没有人回应这个话题'"
      />

      <div v-else class="max-h-[60vh] space-y-0.5 overflow-y-auto">
        <KunLink
          v-for="(item, index) in records"
          :key="index"
          :to="`/user/${item.user.id}`"
          underline="none"
          color="default"
          class-name="hover:bg-default-100 flex items-center gap-3 rounded-lg p-2 transition-colors"
          @click="open = false"
        >
          <KunAvatar :user="item.user" size="sm" />
          <span class="text-foreground min-w-0 flex-1 truncate font-medium">
            {{ item.user.name }}
          </span>
          <img
            :src="reactionAsset(item.reaction)"
            :alt="item.reaction"
            class="size-6 max-w-none shrink-0"
          />
          <KunTime
            :time="item.created"
            class="text-default-500 shrink-0 text-xs"
          />
        </KunLink>
      </div>
    </div>
  </KunModal>
</template>
