<script setup lang="ts">
const reports = useEntity('reports')
const toasts = useToasts()
const cfg = useRuntimeConfig()
const { t } = useI18n()

const rows = ref<any[]>([])
const loading = ref(true)
const search = ref('')
const toDelete = ref<any>(null)
const confirmOpen = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await reports.list({ $top: 200 })
    rows.value = res.value ?? []
  } catch (err: any) {
    toasts.error(t('list.couldNotLoadMany'), extractError(err))
  } finally { loading.value = false }
}
onMounted(load)

const filtered = computed(() => {
  if (!search.value) return rows.value
  const q = search.value.toLowerCase()
  return rows.value.filter(r => (r.name ?? '').toLowerCase().includes(q) || (r.templateShortid ?? '').includes(q))
})

function downloadUrl(row: any) {
  return `${cfg.public.apiBase}/reports/${row.shortid}/content`
}

function askDelete(row: any) { toDelete.value = row; confirmOpen.value = true }
async function doDelete() {
  if (!toDelete.value) return
  try {
    await reports.remove(toDelete.value.shortid)
    rows.value = rows.value.filter(r => r.shortid !== toDelete.value.shortid)
    toasts.success(t('reports.deleted'))
  } catch (err: any) { toasts.error(t('common.couldNotDelete'), extractError(err)) }
  finally { toDelete.value = null }
}

function fmt(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}
function fmtBytes(n?: number) {
  if (n == null) return '—'
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}
</script>

<template>
  <div class="cr-anim-fade-up">
    <PageHeader
      icon="i-lucide-file-down"
      :title="t('reports.title')"
      :description="t('reports.description')"
    />

    <DataTable
      :columns="[
        { key: 'name', label: t('reports.colReport') },
        { key: 'template', label: t('reports.colTemplate') },
        { key: 'state', label: t('reports.colState') },
        { key: 'size', label: t('reports.colSize') },
        { key: 'created', label: t('reports.colCreated') },
        { key: 'actions', label: '', align: 'right', width: '120px' },
      ]"
      :rows="filtered"
      :loading="loading"
      :empty-title="t('reports.empty')"
      :empty-description="t('reports.emptyDesc')"
      empty-icon="i-lucide-file-down"
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
            <UIcon name="i-lucide-file-text" class="w-4 h-4" style="color: var(--cr-text-muted)" />
          </span>
          <div class="min-w-0">
            <div class="font-semibold truncate" style="color: var(--cr-text)">{{ row.name ?? row.shortid }}</div>
            <div class="text-[11.5px]" style="color: var(--cr-text-soft)">{{ row.contentType ?? '—' }}</div>
          </div>
        </div>
      </template>
      <template #cell-template="{ row }">
        <span class="font-mono text-[12px]" style="color: var(--cr-text-muted)">{{ row.templateShortid ?? '—' }}</span>
      </template>
      <template #cell-state="{ row }">
        <StatusBadge :state="row.state" />
      </template>
      <template #cell-size="{ row }">
        <span class="text-[12px] tabular-nums" style="color: var(--cr-text-muted)">{{ fmtBytes(row.size) }}</span>
      </template>
      <template #cell-created="{ row }">
        <span class="text-[12px] tabular-nums" style="color: var(--cr-text-muted)">{{ fmt(row.creationDate) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="flex items-center justify-end gap-1" @click.stop>
          <a :href="downloadUrl(row)" target="_blank" class="cr-row-action" :title="t('common.download')">
            <UIcon name="i-lucide-download" class="w-4 h-4" />
          </a>
          <button class="cr-row-action cr-row-action--danger" :title="t('list.delete')" @click="askDelete(row)">
            <UIcon name="i-lucide-trash-2" class="w-4 h-4" />
          </button>
        </div>
      </template>
    </DataTable>

    <ConfirmDialog v-model="confirmOpen" :title="t('reports.confirmTitle')" :description="`${t('reports.confirmDesc')} &quot;${toDelete?.name ?? toDelete?.shortid}&quot;.`" destructive :confirm-label="t('list.delete')" @confirm="doDelete" />
  </div>
</template>
