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
}

withDefaults(defineProps<Props>(), {
  loading: false,
  emptyTitle: 'Sin resultados',
  emptyDescription: 'Todavía no hay elementos para mostrar.',
  emptyIcon: 'i-lucide-inbox',
})

defineEmits<{ rowClick: [row: Record<string, any>] }>()
</script>

<template>
  <div class="cr-card overflow-hidden">
    <!-- Header -->
    <div v-if="$slots.toolbar" class="px-5 py-3 border-b flex items-center gap-3" style="border-color: var(--cr-border)">
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
            :class="$slots.rowClick || $attrs.onRowClick ? 'cursor-pointer' : ''"
            @click="$emit('rowClick', row)"
          >
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
</style>
