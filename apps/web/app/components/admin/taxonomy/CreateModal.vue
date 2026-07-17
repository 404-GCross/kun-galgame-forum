<script setup lang="ts">
// Create modal for tag / official / engine, fields switch by `type` — the
// admin-console home of the old PR-editor's "没有就新建" modal (that editor
// retires with the old wire; series creation reuses GalgameSeriesModal).
// POST is allowed for any logged-in user; duplicate name → wiki 400 with a
// clear message (kunFetch toasts it, the modal stays open).
import { KUN_GALGAME_TAG_CATEGORY_MAP } from '~/constants/galgameTag'
import { KUN_GALGAME_OFFICIAL_CATEGORY_MAP } from '~/constants/galgameOfficial'

const props = defineProps<{
  type: 'tag' | 'official' | 'engine'
}>()

const open = defineModel<boolean>({ required: true })

const emit = defineEmits<{ created: [] }>()

const tagCategoryOptions = Object.entries(KUN_GALGAME_TAG_CATEGORY_MAP).map(
  ([value, label]) => ({ value, label })
)

const officialCategoryOptions = Object.entries(
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP
).map(([value, label]) => ({ value, label }))

const form = reactive({
  name: '',
  category: '',
  original: '',
  link: '',
  lang: '',
  description: '',
  aliasText: ''
})

const title = computed(
  () =>
    ({ tag: '新建标签', official: '新建会社', engine: '新建引擎' })[props.type]
)

// Re-seed every open; category gets a sane default so the required select is
// never empty.
watch(open, (isOpen) => {
  if (!isOpen) return
  form.name = ''
  form.category = props.type === 'tag' ? 'content' : 'company'
  form.original = ''
  form.link = ''
  form.lang = ''
  form.description = ''
  form.aliasText = ''
  submitting.value = false
})

const submitting = ref(false)

const handleSubmit = async () => {
  const name = form.name.trim()
  if (!name) {
    useMessage('请填写名称', 'warn')
    return
  }
  const alias = form.aliasText
    .split(',')
    .map((a) => a.trim())
    .filter(Boolean)

  let body: Record<string, unknown>
  if (props.type === 'tag') {
    body = {
      name,
      category: form.category,
      description: form.description.trim(),
      alias
    }
  } else if (props.type === 'official') {
    body = {
      name,
      category: form.category,
      original: form.original.trim(),
      link: form.link.trim(),
      lang: form.lang.trim(),
      description: form.description.trim(),
      alias
    }
  } else {
    body = { name, description: form.description.trim(), alias }
  }

  submitting.value = true
  const res = await kunFetch<{ id: number }>(`/galgame-${props.type}`, {
    method: 'POST',
    body
  })
  submitting.value = false

  // null = error; kunFetch already toasted the wiki message (e.g. 已存在同名
  // 条目). Keep the modal open so the user can rename or cancel.
  if (res && res.id) {
    emit('created')
    open.value = false
    useMessage('创建成功', 'success')
  }
}
</script>

<template>
  <KunModal v-model="open" inner-class-name="max-w-md" :is-dismissable="false">
    <form @submit.prevent>
      <h2 class="mb-6 text-xl font-bold">{{ title }}</h2>

      <div class="space-y-4">
        <KunInput v-model="form.name" label="名称" required />

        <KunSelect
          v-if="type === 'tag'"
          v-model="form.category"
          label="标签分类"
          :options="tagCategoryOptions"
        />
        <template v-if="type === 'official'">
          <KunSelect
            v-model="form.category"
            label="会社分类"
            :options="officialCategoryOptions"
          />
          <KunInput v-model="form.original" label="原文名 (可选)" />
          <KunInput v-model="form.link" label="链接 (可选)" />
          <KunInput v-model="form.lang" label="语言 (如 ja, 可选)" />
        </template>

        <KunTextarea v-model="form.description" label="描述 (可选)" />
        <KunTextarea
          v-model="form.aliasText"
          label="别名 (请使用 , 分隔, 可选)"
          :rows="3"
        />
      </div>

      <div class="mt-6 flex justify-end gap-3">
        <KunButton variant="light" color="danger" @click="open = false">
          取消
        </KunButton>
        <KunButton color="primary" :loading="submitting" @click="handleSubmit">
          创建
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
