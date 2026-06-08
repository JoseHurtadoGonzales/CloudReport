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
  /** Enable client-side pagination (footer pager + page-size selector). */
  paginate?: boolean
  /** Initial rows per page. */
  pageSize?: number
  /** Selectable page sizes shown in the footer dropdown. */
  pageSizeOptions?: number[]
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  emptyTitle: 'Sin resultados',
  emptyDescription: 'Todavía no hay elementos para mostrar.',
  emptyIcon: 'i-lucide-inbox',
  selectable: false,
  bulkDeleteLabel: 'Eliminar seleccionados',
  paginate: true,
  pageSize: 10,
  pageSizeOptions: () => [10, 25, 50, 100],
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

// ── Pagination ───────────────────────────────────────────────────────
const pageSize = ref(props.pageSize)
const currentPage = ref(1)

const totalPages = computed(() =>
  props.paginate ? Math.max(1, Math.ceil(props.rows.length / pageSize.value)) : 1,
)
const offset = computed(() => (currentPage.value - 1) * pageSize.value)
const pagedRows = computed(() =>
  props.paginate ? props.rows.slice(offset.value, offset.value + pageSize.value) : props.rows,
)
function rowIndex(i: number) {
  return props.paginate ? offset.value + i : i
}

const rangeFrom = computed(() => (props.rows.length === 0 ? 0 : offset.value + 1))
const rangeTo = computed(() => Math.min(offset.value + pageSize.value, props.rows.length))

const pageSizeSel = computed({
  get: () => String(pageSize.value),
  set: (v: string) => { pageSize.value = Number(v) || props.pageSize },
})
const pageSizeOpts = computed(() =>
  props.pageSizeOptions.map(n => ({ value: String(n), label: `${n} / página` })),
)

// Windowed page buttons with ellipsis for long lists.
const pageItems = computed<(number | string)[]>(() => {
  const tp = totalPages.value
  const cur = currentPage.value
  if (tp <= 7) return Array.from({ length: tp }, (_, i) => i + 1)
  const out: (number | string)[] = [1]
  const start = Math.max(2, cur - 1)
  const end = Math.min(tp - 1, cur + 1)
  if (start > 2) out.push('…')
  for (let p = start; p <= end; p++) out.push(p)
  if (end < tp - 1) out.push('…')
  out.push(tp)
  return out
})

function goTo(p: number) {
  currentPage.value = Math.min(Math.max(1, p), totalPages.value)
}

// Keep currentPage valid as rows shrink/grow (filter, delete) and reset when
// the page size changes.
watch(pageSize, () => { currentPage.value = 1 })
watch(() => props.rows.length, () => {
  if (currentPage.value > totalPages.value) currentPage.value = totalPages.value
})
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
            v-for="(row, i) in pagedRows"
            :key="row._id ?? row.id ?? row.shortid ?? rowIndex(i)"
            class="cr-row"
            :class="[
              $slots.rowClick || $attrs.onRowClick ? 'cursor-pointer' : '',
              selectable && selected.has(keyOf(row, rowIndex(i))) ? 'cr-row--selected' : '',
            ]"
            @click="$emit('rowClick', row)"
          >
            <td v-if="selectable" class="px-4 py-3.5 align-middle w-px" @click.stop>
              <input
                type="checkbox"
                class="cr-checkbox"
                :checked="selected.has(keyOf(row, rowIndex(i)))"
                :aria-label="`Seleccionar fila ${rowIndex(i) + 1}`"
                @change="toggleRow(row, rowIndex(i))"
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

    <!-- Pagination footer -->
    <div
      v-if="paginate && !loading && rows.length > 0"
      class="cr-pager"
    >
      <div class="cr-pager-info">
        <span>{{ rangeFrom }}–{{ rangeTo }} de {{ rows.length }}</span>
        <div class="cr-pager-size">
          <CrSelect v-model="pageSizeSel" :options="pageSizeOpts" />
        </div>
      </div>

      <div v-if="totalPages > 1" class="cr-pager-nav">
        <button
          class="cr-pager-btn"
          :disabled="currentPage === 1"
          aria-label="Página anterior"
          @click="goTo(currentPage - 1)"
        >
          <UIcon name="i-lucide-chevron-left" class="w-4 h-4" />
        </button>
        <template v-for="(p, idx) in pageItems" :key="idx">
          <span v-if="p === '…'" class="cr-pager-ellipsis">…</span>
          <button
            v-else
            class="cr-pager-btn"
            :class="p === currentPage ? 'cr-pager-btn--active' : ''"
            @click="goTo(p as number)"
          >{{ p }}</button>
        </template>
        <button
          class="cr-pager-btn"
          :disabled="currentPage === totalPages"
          aria-label="Página siguiente"
          @click="goTo(currentPage + 1)"
        >
          <UIcon name="i-lucide-chevron-right" class="w-4 h-4" />
        </button>
      </div>
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

/* Pagination footer */
.cr-pager {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 10px 16px;
  border-top: 1px solid var(--cr-border);
}
.cr-pager-info {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12.5px;
  color: var(--cr-text-muted);
  font-variant-numeric: tabular-nums;
}
.cr-pager-size {
  width: 130px;
}
.cr-pager-nav {
  display: flex;
  align-items: center;
  gap: 4px;
}
.cr-pager-btn {
  min-width: 30px;
  height: 30px;
  padding: 0 8px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--cr-text-muted);
  border: 1px solid transparent;
  transition: background-color 120ms, color 120ms, border-color 120ms;
}
.cr-pager-btn:hover:not(:disabled) {
  background: var(--cr-surface-soft);
  color: var(--cr-text);
}
.cr-pager-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.cr-pager-btn--active {
  background: var(--cr-surface-soft);
  border-color: var(--cr-border-strong);
  color: var(--cr-text);
}
html.dark .cr-pager-btn--active {
  background: rgb(255 255 255 / 0.06);
}
.cr-pager-ellipsis {
  padding: 0 4px;
  color: var(--cr-text-soft);
  font-size: 12.5px;
}
</style>
