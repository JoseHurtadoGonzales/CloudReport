<script setup lang="ts">
const schedules = useEntity('schedules')
const api = useApi()
const toasts = useToasts()
const { t } = useI18n()

const rows = ref<any[]>([])
const loading = ref(true)
const formOpen = ref(false)
const form = ref({ shortid: '', name: '', cron: '0 9 * * 1-5', templateShortid: '', enabled: true })
const nextRunPreview = ref<string>('')

const templates = ref<any[]>([])

const toDelete = ref<any>(null)
const confirmOpen = ref(false)

async function load() {
  loading.value = true
  try {
    const [s, t] = await Promise.all([
      schedules.list({ $top: 200 }),
      api.get<{ value: any[] }>('/odata/templates', { query: { $top: 500, $select: 'name,shortid' } }),
    ])
    rows.value = s.value ?? []
    templates.value = t.value ?? []
  } catch (err: any) {
    toasts.error(t('list.couldNotLoadMany'), extractError(err))
  } finally { loading.value = false }
}
onMounted(load)

const cronCommon = computed(() => [
  { label: t('schedules.cron5min'),     v: '*/5 * * * *' },
  { label: t('schedules.cronHourly'),   v: '0 * * * *' },
  { label: t('schedules.cronDaily9'),   v: '0 9 * * *' },
  { label: t('schedules.cronWeekdays9'),v: '0 9 * * 1-5' },
  { label: t('schedules.cronMon8'),     v: '0 8 * * 1' },
  { label: t('schedules.cronMonthly'),  v: '0 0 1 * *' },
])

async function previewCron() {
  if (!form.value.cron) return
  try {
    const cronEncoded = encodeURIComponent(form.value.cron)
    const r = await api.get<{ nextRun: string }>(`/api/scheduling/nextRun/${cronEncoded}`)
    nextRunPreview.value = new Date(r.nextRun).toLocaleString()
  } catch (err: any) {
    nextRunPreview.value = t('schedules.invalidCron')
  }
}
watch(() => form.value.cron, previewCron, { immediate: false })

function openCreate() {
  form.value = { shortid: '', name: '', cron: '0 9 * * 1-5', templateShortid: '', enabled: true }
  nextRunPreview.value = ''
  formOpen.value = true
}

async function save() {
  try {
    if (!form.value.name) { toasts.error(t('schedules.missingName')); return }
    if (!form.value.templateShortid) { toasts.error(t('schedules.missingTpl')); return }
    await schedules.create({
      name: form.value.name,
      cron: form.value.cron,
      templateShortid: form.value.templateShortid,
      enabled: form.value.enabled,
    })
    toasts.success(t('schedules.created'))
    formOpen.value = false
    await load()
  } catch (err: any) {
    toasts.error(t('common.couldNotCreate'), extractError(err))
  }
}

async function runNow(row: any) {
  try {
    await api.post('/api/scheduling/runNow', { scheduleShortid: row.shortid })
    toasts.success(t('schedules.queued'))
  } catch (err: any) {
    toasts.error(t('schedules.couldNotRun'), extractError(err))
  }
}

async function toggle(row: any) {
  try {
    await schedules.update(row.shortid, { enabled: !row.enabled })
    row.enabled = !row.enabled
    toasts.success(row.enabled ? t('schedules.activated') : t('schedules.paused'))
  } catch (err: any) {
    toasts.error(t('schedules.couldNotToggle'), extractError(err))
  }
}

function askDelete(row: any) { toDelete.value = row; confirmOpen.value = true }
async function doDelete() {
  if (!toDelete.value) return
  await schedules.remove(toDelete.value.shortid)
  rows.value = rows.value.filter(r => r.shortid !== toDelete.value.shortid)
  toDelete.value = null
  toasts.success(t('list.deleted'))
}

function fmt(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}
</script>

<template>
  <div class="cr-anim-fade-up">
    <PageHeader
      icon="i-lucide-calendar-clock"
      :title="t('schedules.title')"
      :description="t('schedules.description')"
    >
      <template #actions>
        <button class="cr-btn-primary !w-auto" @click="openCreate">
          <UIcon name="i-lucide-plus" class="w-4 h-4" />
          <span>{{ t('schedules.new') }}</span>
        </button>
      </template>
    </PageHeader>

    <DataTable
      :columns="[
        { key: 'name', label: t('schedules.colName') },
        { key: 'cron', label: t('schedules.colCron') },
        { key: 'template', label: t('schedules.colTemplate') },
        { key: 'next', label: t('schedules.colNext') },
        { key: 'enabled', label: t('schedules.colState') },
        { key: 'actions', label: '', align: 'right', width: '160px' },
      ]"
      :rows="rows"
      :loading="loading"
      :empty-title="t('schedules.empty')"
      :empty-description="t('schedules.emptyDesc')"
      empty-icon="i-lucide-calendar-clock"
    >
      <template #cell-name="{ row }">
        <div class="font-semibold" style="color: var(--cr-text)">{{ row.name }}</div>
        <div class="text-[11px] font-mono" style="color: var(--cr-text-soft)">{{ row.shortid }}</div>
      </template>
      <template #cell-cron="{ row }">
        <code class="px-2 py-1 rounded text-[12px] font-mono" style="background: var(--cr-surface-soft); color: var(--cr-text)">{{ row.cron }}</code>
      </template>
      <template #cell-template="{ row }">
        <span class="font-mono text-[12px]" style="color: var(--cr-text-muted)">{{ row.templateShortid }}</span>
      </template>
      <template #cell-next="{ row }">
        <span class="text-[12px] tabular-nums" style="color: var(--cr-text-muted)">{{ fmt(row.nextRun) }}</span>
      </template>
      <template #cell-enabled="{ row }">
        <StatusBadge :state="row.enabled ? 'planned' : 'queued'" />
      </template>
      <template #cell-actions="{ row }">
        <div class="flex items-center justify-end gap-1" @click.stop>
          <button class="cr-row-action" :title="t('schedules.runNow')" @click="runNow(row)">
            <UIcon name="i-lucide-play" class="w-4 h-4" />
          </button>
          <button class="cr-row-action" :title="row.enabled ? t('schedules.pause') : t('schedules.activate')" @click="toggle(row)">
            <UIcon :name="row.enabled ? 'i-lucide-pause' : 'i-lucide-play'" class="w-4 h-4" />
          </button>
          <button class="cr-row-action cr-row-action--danger" :title="t('list.delete')" @click="askDelete(row)">
            <UIcon name="i-lucide-trash-2" class="w-4 h-4" />
          </button>
        </div>
      </template>
    </DataTable>

    <!-- Create modal -->
    <Teleport to="body">
      <Transition
        enter-active-class="transition-opacity duration-200"
        leave-active-class="transition-opacity duration-150"
        enter-from-class="opacity-0" leave-to-class="opacity-0"
      >
        <div v-if="formOpen" class="fixed inset-0 z-50 flex items-center justify-center px-4 bg-black/50 backdrop-blur-sm" @click.self="formOpen = false">
          <div class="cr-card max-w-lg w-full p-6 cr-anim-fade-up">
            <h3 class="text-[18px] font-bold tracking-tight" style="color: var(--cr-text)">{{ t('schedules.modalTitle') }}</h3>
            <p class="text-[13px] mt-1" style="color: var(--cr-text-muted)">{{ t('schedules.modalDesc') }}</p>

            <div class="mt-5 space-y-4">
              <div>
                <label class="cr-label">{{ t('schedules.fieldName') }}</label>
                <input v-model="form.name" type="text" :placeholder="t('schedules.namePlaceholder')" class="cr-input !pl-4" />
              </div>
              <div>
                <label class="cr-label">{{ t('schedules.fieldTemplate') }}</label>
                <select v-model="form.templateShortid" class="cr-input !pl-4">
                  <option value="">{{ t('schedules.choosePlaceholder') }}</option>
                  <option v-for="tpl in templates" :key="tpl.shortid" :value="tpl.shortid">{{ tpl.name }} ({{ tpl.shortid }})</option>
                </select>
              </div>
              <div>
                <label class="cr-label">{{ t('schedules.fieldCron') }}</label>
                <input v-model="form.cron" type="text" class="cr-input !pl-4 font-mono" />
                <div class="flex flex-wrap gap-1.5 mt-2">
                  <button v-for="p in cronCommon" :key="p.v" class="text-[11px] px-2 py-1 rounded-full transition-colors duration-150" style="background: var(--cr-surface-soft); color: var(--cr-text-muted)" @click="form.cron = p.v">
                    {{ p.label }}
                  </button>
                </div>
                <p v-if="nextRunPreview" class="text-[12px] mt-2 inline-flex items-center gap-1.5" style="color: var(--cr-text-muted)">
                  <UIcon name="i-lucide-clock" class="w-3.5 h-3.5" />
                  {{ t('schedules.nextLabel') }} <span class="font-semibold tabular-nums" style="color: var(--cr-text)">{{ nextRunPreview }}</span>
                </p>
              </div>
              <label class="inline-flex items-center gap-2.5 cursor-pointer">
                <input v-model="form.enabled" type="checkbox" class="cr-checkbox" />
                <span class="text-[13px]" style="color: var(--cr-text-muted)">{{ t('schedules.enableOnCreate') }}</span>
              </label>
            </div>

            <div class="mt-6 flex items-center justify-end gap-2">
              <button class="cr-btn-secondary" @click="formOpen = false">{{ t('common.cancel') }}</button>
              <button class="cr-btn-primary !w-auto" @click="save">{{ t('common.create') }}</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <ConfirmDialog v-model="confirmOpen" :title="t('schedules.confirmTitle')" :description="`${t('list.willDelete')} &quot;${toDelete?.name}&quot;.`" destructive :confirm-label="t('list.delete')" @confirm="doDelete" />
  </div>
</template>
