<script setup lang="ts">
// 制作人员: the game's credits — 脚本 / 原画 / 音乐 / 声优 — from the catalog's
// credit graph. The backend does all the editorial work (folds the duplicate
// role vocabularies, merges the spellings of one person, puts authorship above
// the cast and 其他 at the bottom), so this renders the list verbatim.
//
// Names are plain text on purpose: the catalog addresses them (each row carries
// its credit-name id) but the forum has no person page to link to yet, and a
// link that goes nowhere is worse than none.
const props = defineProps<{
  staff: GalgameDetailStaff[]
}>()

// A game credits six to thirty roles, and the tail is single-name oddities
// (QA / 协力 / 制作总指挥). Show the roles a reader came for, keep the rest one
// click away rather than pushing the screenshots off the screen.
const COLLAPSED_GROUPS = 6

const isExpanded = ref(false)
const isCollapsible = computed(() => props.staff.length > COLLAPSED_GROUPS)
const visibleStaff = computed(() =>
  isCollapsible.value && !isExpanded.value
    ? props.staff.slice(0, COLLAPSED_GROUPS)
    : props.staff
)
const hiddenCount = computed(() => props.staff.length - COLLAPSED_GROUPS)
</script>

<template>
  <div v-if="staff.length" class="space-y-3">
    <KunHeader
      name="制作人员"
      description="该 Galgame 的剧本, 原画, 音乐, 声优等制作人员名单, 数据来自 NextMoe 目录的署名图谱"
      scale="h2"
    />

    <dl class="space-y-4">
      <div
        v-for="group in visibleStaff"
        :key="group.role_key"
        class="grid grid-cols-1 gap-x-4 gap-y-1 sm:grid-cols-[6rem_1fr]"
      >
        <dt class="text-default-500 pt-0.5 text-sm font-medium">
          {{ group.role_name }}
        </dt>
        <dd class="flex flex-wrap items-baseline gap-x-4 gap-y-1.5">
          <span
            v-for="person in group.people"
            :key="person.id"
            class="text-default-800 text-base"
          >
            {{ person.name }}
            <!-- Voice acting is the one role where the credit means nothing
                 without the character it was for. -->
            <span
              v-if="person.characters?.length"
              class="text-default-400 text-sm"
            >
              （{{ person.characters.join(' / ') }}）
            </span>
          </span>
        </dd>
      </div>
    </dl>

    <KunButton
      v-if="isCollapsible"
      variant="flat"
      color="primary"
      size="sm"
      @click="isExpanded = !isExpanded"
    >
      <KunIcon
        :name="isExpanded ? 'lucide:chevron-up' : 'lucide:chevron-down'"
      />
      {{ isExpanded ? '收起制作人员' : `展开其余 ${hiddenCount} 项职位` }}
    </KunButton>
  </div>
</template>
