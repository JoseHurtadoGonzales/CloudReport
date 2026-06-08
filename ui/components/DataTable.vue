<script setup lang="ts">
interface Column {
  key: string
  label: string
  align?: 'left' | 'right' | 'center'
  width?: string
}

interface Props {
  columns: Column[]
  rows: Record<string, any>[]
  loading?: boolean
  emptyTitle?: string
  emptyDescription?: string
  emptyIcon?: string
  /** Enable multi-row selection (checkbox column + bulk-actions bar). */
  selectable?: boolean
  /** Label for the built-in bulk delete button. */
  bulkDeleteLabel?: string
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  emptyTitle: 'Sin resultados',
  emptyDescription: 'Todavía no hay elementos para mostrar.',
  emptyIcon: 'i-lucide-inbox',
  selectable: false,
  bulkDeleteLabel: 'Eliminar seleccionados',
})

const emit = defineEmits<{
  rowClick: [row: Record<string, any>]
  /** Fired when the user confirms the bulk action; payload = selected rows. */
  bulkDelete: [rows: Record<string, any>[]]
}>()

// Stable identity for a row — mirrors the :key resolution below.
function keyOf(row: Record<string, any>, i: number): string {
  return String(row._id ?? row.id ?? row.shortid ?? i)
}

const selected = ref<Set<string>>(new Set())

// Prune selection when the row set changes (e.g. after a delete) so stale
// keys don't linger and the header checkbox state stays correct.
watch(() => props.rows, (rows) => {
  const live = new Set(rows.map((r, i) => keyOf(r, i)))
  let changed = false
  for (const k of selected.value) {
    if (!live.has(k)) { selected.value.delete(k); changed = true }
  }
  if (changed) selected.value = new Set(selected.value)
})

const selectedRows = computed(() =>
  props.rows.filter((r, i) => selected.value.has(keyOf(r, i))),
)
const allSelected = computed(() =>
  props.rows.length > 0 && props.rows.every((r, i) => selected.value.has(keyOf(r, i))),
)
const someSelected = computed(() => selected.value.size > 0 && !allSelected.value)

function toggleRow(row: Record<string, any>, i: number) {
  const k = keyOf(row, i)
  if (selected.value.has(k)) selected.value.delete(k)
  else selected.value.add(k)
  selected.value = new Set(selected.value)
}
function toggleAll() {
  if (allSelected.value) selected.value = new Set()
  else selected.value = new Set(props.rows.map((r, i) => keyOf(r, i)))
}
function clearSelection() {
  selected.value = new Set()
}
function emitBulkDelete() {
  emit('bulkDelete', selectedRows.value)
}

// Let parents clear selection imperatively if they need to.
defineExpose({ clearSelection })

// Total column count incl. the optional checkbox column (for empty/loading rows).
const colCount = computed(() => props.columns.length + (props.selectable ? 1 : 0))
</script>

<template>
  <div class="cr-card overflow-hidden">
    <!-- Bulk-actions bar (replaces toolbar look while a selection is active) -->
    <div
      v-if="selectable && selected.size > 0"
      class="px-5 py-2.5 border-b flex items-center gap-3 cr-bulkbar"
    >
      <span class="text-[13px] font-semibold" style="color: var(--cr-text)">
        {{ selected.size }} seleccionado{{ selected.size === 1 ? '' : 's' }}
      </span>
      <button class="cr-bulkbar-link" @click="clearSelection">Limpiar selección</button>
      <div class="flex-1" />
      <slot name="bulk-actions" :rows="selectedRows" :clear="clearSelection">
        <button class="cr-bulkbar-delete" @click="emitBulkDelete">
          <UIcon name="i-lucide-trash-2" class="w-4 h-4" />
          <span>{{ bulkDeleteLabel }}</span>
        </button>
      </slot>
    </div>

    <!-- Header / toolbar -->
    <div v-else-if="$slots.toolbar" class="px-5 py-3 border-b flex items-center gap-3" style="border-color: var(--cr-border)">
      <slot name="toolbar" />
    </div>

    <div v-if="loading" class="p-8 text-center text-sm" style="color: var(--cr-text-muted)">
      <span class="inline-flex items-center gap-2">
        <span class="w-4 h-4 border-2 rounded-full cr-anim-spin" style="border-color: var(--color-wise-500); border-top-color: transparent" />
        Cargando…
      </span>
    </div>

    <EmptyState
      v-else-if="rows.length === 0"
      :icon="emptyIcon"
      :title="emptyTitle"
      :description="emptyDescription"
      class="!shadow-none !border-0 !bg-transparent"
    >
      <template v-if="$slots.emptyCta" #default>
        <slot name="emptyCta" />
      </template>
    </EmptyState>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-[13.5px]">
        <thead>
          <tr style="background: var(--cr-surface-soft)">
            <th v-if="selectable" class="px-4 py-3 w-px">
              <input
                type="checkbox"
                class="cr-checkbox"
                :checked="allSelected"
                :indeterminate="someSelected"
                aria-label="Seleccionar todo"
                @change="toggleAll"
                @click.stop
              />
            </th>
            <th
              v-for="c in columns"
              :key="c.key"
              :style="{ textAlign: c.align ?? 'left', width: c.width }"
              class="px-5 py-3 font-semibold cr-eyebrow whitespace-nowrap"
            >
              {{ c.label }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(row, i) in rows"
            :key="row._id ?? row.id ?? row.shortid ?? i"
            class="cr-row"
            :class="[
              $slots.rowClick || $attrs.onRowClick ? 'cursor-pointer' : '',
              selectable && selected.has(keyOf(row, i)) ? 'cr-row--selected' : '',
            ]"
            @click="$emit('rowClick', row)"
          >
            <td v-if="selectable" class="px-4 py-3.5 align-middle w-px" @click.stop>
              <input
                type="checkbox"
                class="cr-checkbox"
                :checked="selected.has(keyOf(row, i))"
                :aria-label="`Seleccionar fila ${i + 1}`"
                @change="toggleRow(row, i)"
              />
            </td>
            <td
              v-for="c in columns"
              :key="c.key"
              :style="{ textAlign: c.align ?? 'left' }"
              class="px-5 py-3.5 align-middle"
            >
              <slot :name="`cell-${c.key}`" :row="row" :value="row[c.key]">
                {{ row[c.key] ?? '—' }}
              </slot>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style>
.cr-row {
  border-top: 1px solid var(--cr-border);
  transition: background-color 100ms;
}
.cr-row:hover {
  background: var(--cr-surface-soft);
}
.cr-row--selected {
  background: rgb(159 232 112 / 0.12);
}
.cr-row--selected:hover {
  background: rgb(159 232 112 / 0.18);
}

/* Checkbox — accent-tinted, sized for comfortable clicking */
.cr-checkbox {
  width: 16px;
  height: 16px;
  cursor: pointer;
  accent-color: var(--color-wise-500, #5fb52e);
  vertical-align: middle;
}

/* Bulk-actions bar */
.cr-bulkbar {
  border-color: var(--cr-border);
  background: rgb(159 232 112 / 0.10);
}
html.dark .cr-bulkbar {
  background: rgb(159 232 112 / 0.08);
}
.cr-bulkbar-link {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--cr-text-muted);
  transition: color 120ms;
}
.cr-bulkbar-link:hover { color: var(--cr-text); }
.cr-bulkbar-delete {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  font-size: 12.5px;
  font-weight: 700;
  color: #fff;
  background: #d03238;
  box-shadow: 0 1px 3px rgb(14 15 12 / 0.18);
  transition: background-color 120ms, transform 100ms;
}
.cr-bulkbar-delete:hover { background: #b9272d; }
.cr-bulkbar-delete:active { transform: scale(0.97); }
</style>
