<script setup lang="ts">
import { randomUpvoteDescription } from '~/constants/upvote'

interface UpvoteRecord {
  id: number
  user: KunUser
  description: string
  created: string | Date
}

const props = defineProps<{
  topicId?: number
  records?: UpvoteRecord[]
}>()

const fetched = ref<UpvoteRecord[]>([])
const records = computed(() => props.records ?? fetched.value)

onMounted(async () => {
  if (props.records || !props.topicId) return
  const data = await kunFetch<UpvoteRecord[]>(`/topic/${props.topicId}/upvotes`)
  if (data) fetched.value = data
})

const blurb = (r: UpvoteRecord) =>
  r.description || randomUpvoteDescription(r.id)
</script>

<template>
  <div
    v-if="records.length"
    class="max-h-56 space-y-2 overflow-y-auto pr-1 text-sm"
  >
    <div
      v-for="r in records"
      :key="r.id"
      class="flex items-center justify-between gap-2"
    >
      <div class="flex min-w-0 items-center gap-1.5">
        <KunAvatar :user="r.user" size="sm" />
        <span class="text-default-700 shrink-0 font-medium">
          {{ r.user.name }}
        </span>
        <span class="text-default-500 truncate">
          推了这个话题，<span class="text-secondary font-bold">{{
            blurb(r)
          }}</span>
        </span>
      </div>
      <span class="text-default-400 shrink-0 whitespace-nowrap">
        {{ formatTimeDifference(r.created) }}
      </span>
    </div>
  </div>
</template>
