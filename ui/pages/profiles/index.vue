<script setup lang="ts">
const profiles = useEntity('profiles')
const toasts = useToasts()
const { t } = useI18n()

const rows = ref<any[]>([])
const loading = ref(true)
const stateFilter = ref('')

async function load() {
  loading.value = true
  try {
    const res = await profiles.list({ $top: 200 })
    rows.value = res.value ?? []
  } catch (err: any) {
    toasts.error(t('list.couldNotLoadMany'), extractError(err))
  } finally { loading.value = false }
}
onMounted(load)

const filtered = computed(() => {
  if (!stateFilter.value) return rows.value
  return rows.value.filter(r => r.state === stateFilter.value)
})

const stats = computed(() => {
  const total = rows.value.length
  const success = rows.value.filter(r => r.state === 'success').length
  const error = rows.value.filter(r => r.state === 'error').length
  const running = rows.value.filter(r => r.state === 'running').length
  return { total, success, error, running, successRate: total ? Math.round((success / total) * 100) : 0 }
})

function fmt(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}
function durationMs(p: any) {
  if (!p.timestamp || !p.finishedOn) return null
  return new Date(p.finishedOn).getTime() - new Date(p.timestamp).getTime()
}
</script>

<template>
  <div class="cr-anim-fade-up">
    <PageHeader
      icon="i-lucide-activity"
      :title="t('profiles.title')"
      :description="t('profiles.description')"
    />

    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6 cr-stagger">
      <StatCard :label="t('profiles.statTotal')" :value="stats.total" icon="i-lucide-activity" />
      <StatCard :label="t('profiles.statSuccess')" :value="stats.success" icon="i-lucide-circle-check" :hint="t('profiles.successRate').replace('{n}', String(stats.successRate))" />
      <StatCard :label="t('profiles.statErrors')" :value="stats.error" icon="i-lucide-circle-x" />
      <StatCard :label="t('profiles.statRunning')" :value="stats.running" icon="i-lucide-loader-2" />
    </div>

    <DataTable
      :columns="[
        { key: 'template', label: t('profiles.colTemplate') },
        { key: 'state', label: t('profiles.colState') },
        { key: 'mode', label: t('profiles.colMode') },
        { key: 'duration', label: t('profiles.colDuration') },
        { key: 'started', label: t('profiles.colStarted') },
        { key: 'finished', label: t('profiles.colFinished') },
      ]"
      :rows="filtered"
      :loading="loading"
      :empty-title="t('profiles.empty')"
      :empty-description="t('profiles.emptyDesc')"
      empty-icon="i-lucide-activity"
    >
      <template #toolbar>
        <div class="flex items-center gap-2">
          <button
            v-for="opt in [
              { v: '', label: t('profiles.filterAll') },
              { v: 'success', label: t('profiles.filterSuccess') },
              { v: 'error', label: t('profiles.filterError') },
              { v: 'running', label: t('profiles.filterRunning') },
            ]" :key="opt.v"
            class="px-3 py-1.5 rounded-full text-[12px] font-semibold transition-colors duration-150"
            :class="stateFilter === opt.v ? 'shadow-soft' : ''"
            :style="stateFilter === opt.v
              ? 'background: var(--color-wise-400); color: #0e0f0c'
              : 'background: var(--cr-surface-soft); color: var(--cr-text-muted)'"
            @click="stateFilter = opt.v"
          >
            {{ opt.label }}
          </button>
        </div>
      </template>

      <template #cell-template="{ row }">
        <span class="font-mono text-[12.5px]" style="color: var(--cr-text)">{{ row.templateShortid ?? '—' }}</span>
      </template>
      <template #cell-state="{ row }">
        <StatusBadge :state="row.state" />
      </template>
      <template #cell-mode="{ row }">
        <span class="text-[12px] font-medium" style="color: var(--cr-text-muted)">{{ row.mode }}</span>
      </template>
      <template #cell-duration="{ row }">
        <span class="text-[12px] tabular-nums font-mono" style="color: var(--cr-text-muted)">
          {{ durationMs(row) != null ? durationMs(row) + ' ms' : '—' }}
        </span>
      </template>
      <template #cell-started="{ row }">
        <span class="text-[12px] tabular-nums" style="color: var(--cr-text-muted)">{{ fmt(row.timestamp) }}</span>
      </template>
      <template #cell-finished="{ row }">
        <span class="text-[12px] tabular-nums" style="color: var(--cr-text-muted)">{{ fmt(row.finishedOn) }}</span>
      </template>
    </DataTable>
  </div>
</template>
