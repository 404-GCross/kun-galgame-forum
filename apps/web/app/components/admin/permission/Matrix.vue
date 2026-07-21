<script setup lang="ts">
import {
  KUN_PERMISSION_GROUP_ORDER,
  KUN_PERMISSION_KEYS,
  KUN_PERMISSION_META,
  KUN_PERM_ROLE_COLUMNS,
  type KunPermRoleColumn
} from '~/constants/permission'

// Presentational grid: rows = the 43 pure-forum permissions grouped by
// KUN_PERMISSION_GROUP_ORDER, columns = creator / moderator / admin / ren.
// State is owned by the parent page; this component only reflects it and emits
// intent (a cell toggle, a per-role reset).
const props = defineProps<{
  // Editable roles' WORKING effective sets (local pending edits folded in).
  working: Record<string, Set<string>>
  // Every role's compiled baseline — the deviation reference.
  baseline: Record<string, Set<string>>
  // Every role's current effective set — the display source for LOCKED columns.
  effective: Record<string, Set<string>>
  // Disable interaction while a save is in flight.
  disabled?: boolean
}>()

const emit = defineEmits<{
  toggle: [role: string, permission: string, value: boolean]
  reset: [role: string]
}>()

const columns = KUN_PERM_ROLE_COLUMNS

const groups = computed(() =>
  KUN_PERMISSION_GROUP_ORDER.map((group) => ({
    group,
    perms: KUN_PERMISSION_KEYS.filter(
      (key) => KUN_PERMISSION_META[key].group === group
    )
  })).filter((entry) => entry.perms.length)
)

// creator | moderator | admin cells read the working set; ren reads its
// effective set (always the full catalogue).
const isChecked = (col: KunPermRoleColumn, perm: string) =>
  (col.editable
    ? props.working[col.role]?.has(perm)
    : props.effective[col.role]?.has(perm)) ?? false

// A working state that diverges from the baseline is an override: 'grant' when
// the role now holds a permission its baseline lacks, 'revoke' when it lacks one
// its baseline holds. Locked columns can never deviate.
const deviation = (
  col: KunPermRoleColumn,
  perm: string
): 'grant' | 'revoke' | null => {
  if (!col.editable) return null
  const inWork = props.working[col.role]?.has(perm) ?? false
  const inBase = props.baseline[col.role]?.has(perm) ?? false
  if (inWork === inBase) return null
  return inWork ? 'grant' : 'revoke'
}

// 5 columns: a wide label column + one per role. min-width forces horizontal
// scroll on mobile instead of squashing the grid.
const gridStyle =
  'grid-template-columns: minmax(11rem, 1fr) repeat(4, minmax(4.5rem, 7rem));'
</script>

<template>
  <div class="overflow-x-auto">
    <div class="grid min-w-[44rem]" :style="gridStyle">
      <!-- Header row -->
      <div
        class="border-default-200 text-default-500 border-b px-3 py-2 text-sm font-medium"
      >
        权限
      </div>
      <div
        v-for="col in columns"
        :key="col.role"
        class="border-default-200 flex flex-col items-center gap-1 border-b px-2 py-2 text-center"
      >
        <div class="flex items-center gap-1">
          <span class="text-sm font-semibold">{{ col.label }}</span>
          <KunTooltip
            v-if="col.locked"
            text="站长角色恒持全部权限，不可调整"
            position="top"
          >
            <KunIcon name="lucide:lock" class="text-default-400 text-xs" />
          </KunTooltip>
        </div>
        <KunButton
          v-if="col.editable"
          size="xs"
          variant="light"
          color="default"
          :disabled="disabled"
          @click="emit('reset', col.role)"
        >
          重置默认
        </KunButton>
      </div>

      <!-- Grouped permission rows -->
      <template v-for="entry in groups" :key="entry.group">
        <div
          class="bg-default-100/60 text-default-600 col-span-full px-3 py-1.5 text-xs font-semibold"
        >
          {{ entry.group }}
        </div>
        <template v-for="perm in entry.perms" :key="perm">
          <div
            class="border-default-100 flex items-center border-b px-3 py-2 text-sm"
          >
            {{ KUN_PERMISSION_META[perm].label }}
          </div>
          <div
            v-for="col in columns"
            :key="col.role"
            class="border-default-100 flex items-center justify-center border-b px-2 py-2"
          >
            <div class="relative">
              <KunCheckBox
                :model-value="isChecked(col, perm)"
                :disabled="!col.editable || disabled"
                color="primary"
                @change="(value) => emit('toggle', col.role, perm, value)"
              />
              <KunTooltip
                v-if="deviation(col, perm)"
                :text="
                  deviation(col, perm) === 'grant'
                    ? '已授予（高于基线）'
                    : '已撤销（低于基线）'
                "
                position="top"
              >
                <span
                  :class="
                    cn(
                      'absolute -top-1.5 -right-1.5 block size-2 rounded-full',
                      deviation(col, perm) === 'grant'
                        ? 'bg-success'
                        : 'bg-warning'
                    )
                  "
                />
              </KunTooltip>
            </div>
          </div>
        </template>
      </template>
    </div>
  </div>
</template>
