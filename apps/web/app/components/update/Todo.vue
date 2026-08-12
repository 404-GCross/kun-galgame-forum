<script setup lang="ts">
import {
  KUN_TODO_TYPE_MAP,
  KUN_UPDATE_LOG_STATUS_MAP
} from '~/constants/update'
import type { UpdateTodoPayload } from './types'

const canCreateUpdateLog = useCan('update_log.create')
const canEditUpdateLog = useCan('update_log.edit')

const iconMap: Record<number, string> = {
  0: 'lucide:circle-divide',
  1: 'lucide:loader',
  2: 'lucide:check',
  3: 'lucide:x'
}

const textMap: Record<number, string> = {
  0: 'text-default',
  1: 'text-primary',
  2: 'text-success',
  3: 'text-danger'
}

const statusFilters = [
  { value: undefined, label: '全部', icon: 'lucide:list-filter', text: '' },
  ...Object.entries(KUN_UPDATE_LOG_STATUS_MAP).map(([key, label]) => ({
    value: Number(key),
    label,
    icon: iconMap[Number(key)]!,
    text: textMap[Number(key)]!
  }))
]

const pageData = ref({
  page: 1,
  limit: 30,
  language: 'zh-cn',
  status: undefined as number | undefined
})

const setStatusFilter = (status?: number) => {
  pageData.value.status = status
  pageData.value.page = 1
}

const { data, status, refresh } = await useKunFetch<UpdateTodoList>(
  '/update/todo',
  { query: pageData }
)

const showTodoModal = ref(false)
const editingTodo = ref<UpdateTodoPayload>({} as UpdateTodoPayload)

const openCreateTodoModal = () => {
  editingTodo.value = {} as UpdateTodoPayload
  showTodoModal.value = true
}

const openEditTodoModal = (log: UpdateTodo) => {
  if (!data.value) {
    return
  }
  editingTodo.value = {
    status: log.status,
    type: 'forum',
    content_en_us: log.content_en_us,
    content_ja_jp: log.content_ja_jp,
    content_zh_cn: log.content_zh_cn,
    content_zh_tw: log.content_zh_tw,
    todo_id: log.id
  } satisfies UpdateTodoPayload
  showTodoModal.value = true
}

const handleTodoAction = async (data: UpdateTodoPayload) => {
  const result = await kunFetch('/update/todo', {
    method: data.todo_id ? 'PUT' : 'POST',
    body: data
  })

  if (result) {
    useMessage(data.todo_id ? '更新成功' : '发布待办成功', 'success')
    refresh()
  }
}
</script>

<template>
  <div class="space-y-6" v-if="data">
    <KunHeader
      name="待办列表"
      description="这里记录了网站将会实现的功能, 以及更改的功能, 包括 Galgame 以及话题, 以及网站所有方向可能发生的各种更新等等"
    >
      <template #endContent>
        <div v-if="canCreateUpdateLog" class="flex justify-end">
          <KunButton @click="openCreateTodoModal">创建待办</KunButton>
        </div>
      </template>
    </KunHeader>

    <KunCard :is-hoverable="false" :is-transparent="false">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <KunButton
            v-for="filter in statusFilters"
            :key="filter.label"
            size="sm"
            :variant="pageData.status === filter.value ? 'solid' : 'flat'"
            :color="pageData.status === filter.value ? 'primary' : 'default'"
            @click="setStatusFilter(filter.value)"
          >
            <KunIcon
              :name="filter.icon"
              :class="
                cn('h-4 w-4', pageData.status !== filter.value && filter.text)
              "
            />
            {{ filter.label }}
          </KunButton>
        </div>

        <span class="text-default-500 text-sm">共 {{ data.total }} 项</span>
      </div>
    </KunCard>

    <KunCard
      :is-hoverable="false"
      :is-transparent="false"
      v-for="todo in data.todos"
      :key="todo.id"
      content-class="space-y-3"
    >
      <div class="flex items-center gap-3">
        <KunChip color="primary">
          {{ KUN_TODO_TYPE_MAP[todo.type] }}
        </KunChip>

        <span class="text-default-600 text-sm">
          该企划创建于 <KunTime :time="todo.created" type="datetime" />
        </span>
      </div>

      <pre class="font-mono break-all whitespace-pre-line">
        {{ todo.content_zh_cn }}
      </pre>

      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2 text-sm">
          <span v-if="todo.completed_time" class="text-default-500">
            <KunTime :time="todo.completed_time" type="datetime" />
          </span>
          <KunIcon
            :name="iconMap[todo.status]"
            :class="cn('h-4 w-4', textMap[todo.status])"
          />
          <span :class="cn(textMap[todo.status])">
            {{ KUN_UPDATE_LOG_STATUS_MAP[todo.status] }}
          </span>
        </div>

        <KunButton
          variant="flat"
          size="sm"
          v-if="canEditUpdateLog"
          @click="openEditTodoModal(todo)"
        >
          编辑
        </KunButton>
      </div>
    </KunCard>

    <KunNull v-if="!data.todos.length" description="当前筛选条件下没有待办" />

    <UpdateTodoModal
      v-model="showTodoModal"
      :initial-data="editingTodo"
      @submit="handleTodoAction"
    />

    <KunCard
      v-if="data.total > pageData.limit"
      :is-hoverable="false"
      :is-transparent="false"
    >
      <KunPagination
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </KunCard>
  </div>
</template>
