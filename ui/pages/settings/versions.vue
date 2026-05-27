<script setup lang="ts">
const api = useApi()
const toasts = useToasts()
const { t } = useI18n()

const rows = ref<any[]>([])
const loading = ref(true)
const commitOpen = ref(false)
const commitMsg = ref('')
const committing = ref(false)

const toRevert = ref<any>(null)
const confirmOpen = ref(false)

async function load() {
  loading.value = true
  try {
    rows.value = (await api.get<any[]>('/api/version-control/history')) ?? []
  } catch (err: any) {
    toasts.error(t('settings.versions.couldNotLoadHistory'), extractError(err))
  } finally { loading.value = false }
}
onMounted(load)

async function commit() {
  if (!commitMsg.value) { toasts.error(t('settings.versions.missingMsg')); return }
  committing.value = true
  try {
    await api.post('/api/version-control/commit', { message: commitMsg.value })
    toasts.success(t('settings.versions.snapshotCreated'))
    commitOpen.value = false
    commitMsg.value = ''
    await load()
  } catch (err: any) {
    toasts.error(t('settings.versions.couldNotCommit'), extractError(err))
  } finally { committing.value = false }
}

function askRevert(row: any) { toRevert.value = row; confirmOpen.value = true }
async function doRevert() {
  if (!toRevert.value) return
  try {
    await api.post('/api/version-control/revert', { shortid: toRevert.value.shortid })
    toasts.success(t('settings.versions.restored'))
  } catch (err: any) {
    toasts.error(t('settings.versions.couldNotRestore'), extractError(err))
  } finally { toRevert.value = null }
}

function fmt(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}
</script>

<template>
  <div class="cr-anim-fade-up">
    <PageHeader
      icon="i-lucide-git-branch"
      :title="t('settings.versions.title')"
      :description="t('settings.versions.description')"
    >
      <template #actions>
        <button class="cr-btn-primary !w-auto" @click="commitOpen = true">
          <UIcon name="i-lucide-git-commit" class="w-4 h-4" />
          <span>{{ t('settings.versions.new') }}</span>
        </button>
      </template>
    </PageHeader>

    <DataTable
      :columns="[
        { key: 'message', label: t('settings.versions.colMessage') },
        { key: 'short', label: t('settings.versions.colId') },
        { key: 'when', label: t('settings.versions.colDate') },
        { key: 'actions', label: '', align: 'right', width: '120px' },
      ]"
      :rows="rows" :loading="loading"
      :empty-title="t('settings.versions.empty')" empty-icon="i-lucide-git-branch"
    >
      <template #cell-message="{ row }">
        <div class="flex items-center gap-3">
          <span class="w-8 h-8 rounded-full flex items-center justify-center shrink-0" style="background: var(--cr-surface-soft)">
            <UIcon name="i-lucide-git-commit" class="w-4 h-4" style="color: var(--cr-text-muted)" />
          </span>
          <div class="font-semibold" style="color: var(--cr-text)">{{ row.message }}</div>
        </div>
      </template>
      <template #cell-short="{ row }">
        <code class="px-2 py-1 rounded text-[11.5px] font-mono" style="background: var(--cr-surface-soft); color: var(--cr-text)">{{ row.shortid }}</code>
      </template>
      <template #cell-when="{ row }">
        <span class="text-[12px] tabular-nums" style="color: var(--cr-text-muted)">{{ fmt(row.created_at ?? row.creationDate) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="flex items-center justify-end gap-1" @click.stop>
          <button class="cr-row-action" :title="t('settings.versions.restoreTip')" @click="askRevert(row)">
            <UIcon name="i-lucide-rotate-ccw" class="w-4 h-4" />
          </button>
        </div>
      </template>
    </DataTable>

    <Teleport to="body">
      <Transition
        enter-active-class="transition-opacity duration-200"
        leave-active-class="transition-opacity duration-150"
        enter-from-class="opacity-0" leave-to-class="opacity-0"
      >
        <div v-if="commitOpen" class="fixed inset-0 z-50 flex items-center justify-center px-4 bg-black/50 backdrop-blur-sm" @click.self="commitOpen = false">
          <div class="cr-card max-w-md w-full p-6 cr-anim-fade-up">
            <h3 class="text-[18px] font-bold tracking-tight" style="color: var(--cr-text)">{{ t('settings.versions.new') }}</h3>
            <p class="text-[13px] mt-1" style="color: var(--cr-text-muted)">{{ t('settings.versions.modalDesc') }}</p>
            <div class="mt-4">
              <label class="cr-label">{{ t('settings.versions.msgLabel') }}</label>
              <input v-model="commitMsg" type="text" :placeholder="t('settings.versions.msgPlaceholder')" class="cr-input !pl-4" />
            </div>
            <div class="mt-6 flex items-center justify-end gap-2">
              <button class="cr-btn-secondary" @click="commitOpen = false">{{ t('common.cancel') }}</button>
              <button class="cr-btn-primary !w-auto" :disabled="committing" @click="commit">
                <UIcon name="i-lucide-git-commit" class="w-4 h-4" />
                {{ t('settings.versions.commit') }}
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <ConfirmDialog v-model="confirmOpen"
      :title="t('settings.versions.confirmTitle')"
      :description="`${t('settings.versions.confirmDescPre')} &quot;${toRevert?.message}&quot;.`"
      destructive :confirm-label="t('settings.versions.confirmLabel')" @confirm="doRevert" />
  </div>
</template>
