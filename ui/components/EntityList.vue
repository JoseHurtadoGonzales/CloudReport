<script setup lang="ts">
interface Props {
  entitySet: string
  icon: string
  title: string
  description?: string
  newPath: string
  /** column descriptors for the table */
  columns?: { key: string; label: string; align?: 'left' | 'right' | 'center'; width?: string }[]
  /** secondary field shown under the name (e.g. 'engine') */
  subKey?: string
}

const { t } = useI18n()

const props = withDefaults(defineProps<Props>(), {
  description: '',
  subKey: '',
  columns: () => [
    { key: 'name', label: 'Nombre' },
    { key: 'updated', label: 'Actualizado' },
    { key: 'actions', label: '', align: 'right' as const, width: '120px' },
  ],
})

// Localized fallback columns (props.columns default uses ES; this gives a
// reactive set when callers don't override).
const effectiveColumns = computed(() => {
  // If a caller passed custom columns, keep them as-is.
  const def = props.columns
  // Only swap when the columns match the default shape — i.e. names "Nombre" /
  // "Actualizado". Otherwise we'd clobber custom labels.
  const isDefault =
    def.length === 3 &&
    def[0]?.key === 'name' &&
    def[1]?.key === 'updated' &&
    def[2]?.key === 'actions'
  if (!isDefault) return def
  return [
    { key: 'name',    label: t('list.name') },
    { key: 'updated', label: t('list.updated') },
    { key: 'actions', label: '', align: 'right' as const, width: '120px' },
  ]
})

const toasts = useToasts()
const entity = useEntity(props.entitySet)
const router = useRouter()

const rows = ref<any[]>([])
const loading = ref(true)
const search = ref('')
const toDelete = ref<any>(null)
const confirmOpen = ref(false)

// Bulk delete
const bulkRows = ref<any[]>([])
const bulkConfirmOpen = ref(false)
function askBulkDelete(selected: any[]) {
  bulkRows.value = selected
  bulkConfirmOpen.value = true
}
async function doBulkDelete() {
  const targets = bulkRows.value.slice()
  let ok = 0
  for (const row of targets) {
    try {
      await entity.remove(row.shortid)
      rows.value = rows.value.filter(r => r.shortid !== row.shortid)
      ok++
    } catch (err: any) {
      toasts.error(t('common.couldNotDelete'), extractError(err))
    }
  }
  bulkRows.value = []
  if (ok > 0) toasts.success(t('list.deleted'))
}

async function load() {
  loading.value = true
  try {
    const res = await entity.list({ $top: 200 })
    rows.value = res.value ?? []
  } catch (err: any) {
    toasts.error(t('list.couldNotLoadMany'), extractError(err))
  } finally {
    loading.value = false
  }
}
onMounted(load)

const filtered = computed(() => {
  if (!search.value) return rows.value
  const q = search.value.toLowerCase()
  return rows.value.filter(r => r.name?.toLowerCase().includes(q) || r.shortid?.includes(q))
})

function askDelete(row: any) {
  toDelete.value = row
  confirmOpen.value = true
}
async function doDelete() {
  if (!toDelete.value) return
  try {
    await entity.remove(toDelete.value.shortid)
    rows.value = rows.value.filter(r => r.shortid !== toDelete.value.shortid)
    toasts.success(t('list.deleted'))
  } catch (err: any) {
    toasts.error(t('common.couldNotDelete'), extractError(err))
  } finally {
    toDelete.value = null
  }
}

function fmt(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}
</script>

<template>
  <div class="cr-anim-fade-up">
    <PageHeader :icon="icon" :title="title" :description="description">
      <template #actions>
        <NuxtLink :to="newPath" class="cr-btn-primary !w-auto">
          <UIcon name="i-lucide-plus" class="w-4 h-4" />
          <span>{{ t('list.new') }}</span>
        </NuxtLink>
      </template>
    </PageHeader>

    <DataTable
      :columns="effectiveColumns"
      :rows="filtered"
      :loading="loading"
      :empty-title="`${t('list.empty')} — ${title.toLowerCase()}`"
      empty-icon="i-lucide-plus-circle"
      selectable
      :bulk-delete-label="t('list.delete')"
      @row-click="(row) => $router.push(`${newPath.replace('/new', '')}/${row.shortid}`)"
      @bulk-delete="askBulkDelete"
    >
      <template #toolbar>
        <div class="relative flex-1 max-w-md">
          <UIcon name="i-lucide-search" class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4" style="color: var(--cr-text-soft)" />
          <input v-model="search" type="text" :placeholder="t('list.searchPlaceholder')" class="cr-input !pl-10 !py-2 !text-[13px]" style="padding-top: 8px; padding-bottom: 8px" />
        </div>
      </template>

      <template #cell-name="{ row }">
        <div class="flex items-center gap-3">
          <span class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0" style="background: var(--cr-surface-soft)">
            <UIcon :name="icon" class="w-4 h-4" style="color: var(--cr-text-muted)" />
          </span>
          <div class="min-w-0">
            <div class="font-semibold truncate" style="color: var(--cr-text)">{{ row.name }}</div>
            <div class="text-[11.5px] font-mono" style="color: var(--cr-text-soft)">
              {{ row.shortid }} <template v-if="subKey && row[subKey]"> · {{ row[subKey] }}</template>
            </div>
          </div>
        </div>
      </template>

      <template #cell-updated="{ row }">
        <span class="text-[12px] tabular-nums" style="color: var(--cr-text-muted)">{{ fmt(row.modificationDate) }}</span>
      </template>

      <template #cell-actions="{ row }">
        <div class="flex items-center justify-end gap-1" @click.stop>
          <NuxtLink :to="`${newPath.replace('/new', '')}/${row.shortid}`" class="cr-row-action" :title="t('list.edit')">
            <UIcon name="i-lucide-edit-3" class="w-4 h-4" />
          </NuxtLink>
          <button class="cr-row-action cr-row-action--danger" :title="t('list.delete')" @click="askDelete(row)">
            <UIcon name="i-lucide-trash-2" class="w-4 h-4" />
          </button>
        </div>
      </template>

      <template v-for="col in columns.filter(c => !['name','updated','actions'].includes(c.key))" :key="col.key" #[`cell-${col.key}`]="{ row }">
        <slot :name="`cell-${col.key}`" :row="row">{{ row[col.key] ?? '—' }}</slot>
      </template>
    </DataTable>

    <ConfirmDialog
      v-model="confirmOpen"
      :title="t('list.deleteItem')"
      :description="`${t('list.willDelete')} &quot;${toDelete?.name}&quot;.`"
      destructive
      :confirm-label="t('list.delete')"
      @confirm="doDelete"
    />

    <ConfirmDialog
      v-model="bulkConfirmOpen"
      :title="t('list.deleteItem')"
      :description="`${t('list.willDelete')} ${bulkRows.length} ${bulkRows.length === 1 ? 'elemento' : 'elementos'}.`"
      destructive
      :confirm-label="t('list.delete')"
      @confirm="doBulkDelete"
    />
  </div>
</template>
