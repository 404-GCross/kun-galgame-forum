<script setup lang="ts">
import type { SerializeObject } from 'nitropack'

const props = defineProps<{ toolsetId: number; ownerId?: number }>()

const pageData = reactive({
  toolsetId: props.toolsetId,
  page: 1,
  limit: 30,
  sortOrder: 'desc'
})

const { data, status, refresh } = await useKunFetch<{
  commentData: SerializeObject<ToolsetComment>[]
  total: number
}>(`/toolset/${props.toolsetId}/comment/all`, {
  method: 'GET',
  query: pageData,
  lazy: true
})

// Own the comment list locally so publish / edit / delete are reactive: the
// fetch returns a SHALLOW data ref (Nuxt's default deep:false), so mutating the
// nested data.value.commentData wouldn't re-render. Re-seed from each fetch
// (sort change / page / refresh re-assign data.value → shallow ref fires).
const comments = ref<SerializeObject<ToolsetComment>[]>([])
const total = ref(0)
watch(
  data,
  (d) => {
    comments.value = d?.commentData ?? []
    total.value = d?.total ?? 0
  },
  { immediate: true }
)

const addNewComment = (comment: ToolsetComment) => {
  comments.value.unshift(comment as SerializeObject<ToolsetComment>)
  total.value++
}

const removeComment = (commentId: number) => {
  comments.value = comments.value.filter((c) => c.id !== commentId)
  total.value = Math.max(0, total.value - 1)
}

const updateComment = (commentId: number, content: string, edited: string) => {
  const target = comments.value.find((c) => c.id === commentId)
  if (target) {
    target.content = content
    target.edited = edited
  }
}
</script>

<template>
  <div class="space-y-3">
    <KunHeader
      name="评论"
      description="如果您对该工具有任何的使用疑问, 欢迎发布评论"
      scale="h2"
    />

    <ToolsetCommentPublish
      :toolset-id="toolsetId"
      @set-new-comment="addNewComment"
    />

    <KunLoading v-if="status === 'pending'" />

    <KunNull v-if="total === 0 && status !== 'pending'" />

    <div class="space-y-3" v-if="total > 0 && status !== 'pending'">
      <div class="flex items-center gap-2">
        <KunButton
          :is-icon-only="true"
          :variant="pageData.sortOrder === 'desc' ? 'flat' : 'light'"
          size="lg"
          @click="pageData.sortOrder = 'desc'"
        >
          <KunIcon class="text-inherit" name="lucide:arrow-down" />
        </KunButton>

        <KunButton
          :is-icon-only="true"
          :variant="pageData.sortOrder === 'asc' ? 'flat' : 'light'"
          size="lg"
          @click="pageData.sortOrder = 'asc'"
        >
          <KunIcon class="text-inherit" name="lucide:arrow-up" />
        </KunButton>
      </div>

      <ToolsetComment
        v-for="comment in comments"
        :key="comment.id"
        :comment="comment"
        :owner-id="ownerId || 0"
        @remove="removeComment"
        @replied="refresh()"
        @updated="updateComment"
      />

      <KunPagination
        v-if="total >= pageData.limit"
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(total / pageData.limit)"
      />
    </div>
  </div>
</template>
