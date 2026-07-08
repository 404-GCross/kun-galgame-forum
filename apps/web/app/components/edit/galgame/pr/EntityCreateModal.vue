<script setup lang="ts">
// "没有就新建" modal for tag / official / engine, used by the PR-edit
// entity selectors (00-handbook §15 / 04-taxonomy: POST is allowed for
// any logged-in user so people can add an entity missing for an
// original / doujin work). One modal, fields switch by `type`.
//
// On success the wiki returns the full new entity (with id); we emit it
// so the selector adds it to the selection exactly like an existing
// pick. Duplicate name → wiki 400 with a clear message; kunFetch's
// response handler surfaces it and we keep the modal open so the user
// can rename or cancel and pick the existing one instead.
import {
  KUN_GALGAME_TAG_CATEGORY_MAP
} from '~/constants/galgameTag'
import {
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP
} from '~/constants/galgameOfficial'

interface CreatedEntity {
  id: number
  name: string
  [k: string]: unknown
}

const props = defineProps<{
  type: 'tag' | 'official' | 'engine'
  initialName: string
}>()

const open = defineModel<boolean>({ required: true })

const emit = defineEmits<{ created: [entity: CreatedEntity] }>()

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

// Re-seed every open: name from the query the user typed, category to a
// sane default so the required select is never empty.
watch(open, (isOpen) => {
  if (!isOpen) return
  form.name = props.initialName
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
  const res = await kunFetch<CreatedEntity>(`/galgame-${props.type}`, {
    method: 'POST',
    body
  })
  submitting.value = false

  // null = error; kunFetch's response handler already showed the wiki
  // message (e.g. "已存在同名 tag"). Keep the modal open.
  if (res && res.id) {
    emit('created', res)
    open.value = false
    useMessage('创建成功, 已加入选择', 'success')
  }
}
</script>

<template>
  <KunModal v-model="open" inner-class-name="max-w-md" :is-dismissable="false">
    <form class="space-y-4" @submit.prevent>
      <h2 class="text-xl font-bold">{{ title }}</h2>

      <KunInput v-model="form.name" label="名称" required />

      <KunSelect
        v-if="type === 'tag'"
        v-model="form.category"
        label="分类"
        :options="tagCategoryOptions"
      />

      <template v-if="type === 'official'">
        <KunSelect
          v-model="form.category"
          label="分类"
          :options="officialCategoryOptions"
        />
        <KunInput v-model="form.original" label="原文名 (可选)" />
        <KunInput v-model="form.link" label="官网链接 (可选)" type="url" />
        <KunInput v-model="form.lang" label="语言 (可选, 如 ja)" />
      </template>

      <KunTextarea v-model="form.description" label="描述 (可选)" />
      <KunTextarea
        v-model="form.aliasText"
        label="别名 (可选, 用 , 分隔)"
        :rows="2"
      />

      <div class="flex justify-end gap-3">
        <KunButton variant="light" color="danger" @click="open = false">
          取消
        </KunButton>
        <KunButton
          color="primary"
          :loading="submitting"
          @click="handleSubmit"
        >
          创建
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
