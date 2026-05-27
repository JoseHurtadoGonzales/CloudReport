<script setup lang="ts">
const api = useApi()
const toasts = useToasts()
const { t } = useI18n()

const keys = ref<any[]>([])
const loading = ref(true)
const formOpen = ref(false)
const form = ref({ name: '', ttlDays: 90, scopes: ['render', 'read', 'write'] as string[] })
const newKey = ref<string | null>(null)

const toRevoke = ref<any>(null)
const confirmOpen = ref(false)

async function load() {
  loading.value = true
  try {
    keys.value = (await api.get<any[]>('/api/apikeys')) ?? []
  } catch (err: any) {
    toasts.error(t('settings.apiKeys.couldNotLoad'), extractError(err))
  } finally { loading.value = false }
}
onMounted(load)

function openCreate() {
  form.value = { name: '', ttlDays: 90, scopes: ['render', 'read', 'write'] }
  newKey.value = null
  formOpen.value = true
}

async function create() {
  if (!form.value.name) { toasts.error(t('settings.apiKeys.missingName')); return }
  try {
    const res = await api.post<{ apiKey: any; key: string }>('/api/apikeys', form.value)
    newKey.value = res.key
    keys.value = [res.apiKey, ...keys.value]
    toasts.success(t('settings.apiKeys.createdToast'))
  } catch (err: any) {
    toasts.error(t('common.couldNotCreate'), extractError(err))
  }
}

function copyKey() {
  if (!newKey.value) return
  navigator.clipboard.writeText(newKey.value)
  toasts.success(t('common.copied'))
}

function askRevoke(row: any) { toRevoke.value = row; confirmOpen.value = true }
async function doRevoke() {
  if (!toRevoke.value) return
  try {
    await api.delete(`/api/apikeys/${toRevoke.value._id}`)
    keys.value = keys.value.filter(k => k._id !== toRevoke.value._id)
    toasts.success(t('settings.apiKeys.revoked'))
  } catch (err: any) {
    toasts.error(t('settings.apiKeys.couldNotRevoke'), extractError(err))
  } finally { toRevoke.value = null }
}

function isExpired(k: any) {
  return k.expiresAt && new Date(k.expiresAt).getTime() < Date.now()
}
function isRevoked(k: any) {
  return !!k.revokedAt
}
function fmt(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString()
}
function statusOf(k: any): string {
  if (isRevoked(k)) return 'error'
  if (isExpired(k)) return 'queued'
  return 'success'
}
</script>

<template>
  <div class="cr-anim-fade-up">
    <PageHeader
      icon="i-lucide-key-round"
      :title="t('settings.apiKeys.title')"
      :description="t('settings.apiKeys.description')"
    >
      <template #actions>
        <button class="cr-btn-primary !w-auto" @click="openCreate">
          <UIcon name="i-lucide-plus" class="w-4 h-4" />
          <span>{{ t('settings.apiKeys.new') }}</span>
        </button>
      </template>
    </PageHeader>

    <DataTable
      :columns="[
        { key: 'name', label: t('settings.apiKeys.colName') },
        { key: 'key', label: t('settings.apiKeys.colKey') },
        { key: 'scopes', label: t('settings.apiKeys.colScopes') },
        { key: 'status', label: t('settings.apiKeys.colStatus') },
        { key: 'expires', label: t('settings.apiKeys.colExpires') },
        { key: 'used', label: t('settings.apiKeys.colUsed') },
        { key: 'actions', label: '', align: 'right', width: '80px' },
      ]"
      :rows="keys"
      :loading="loading"
      :empty-title="t('settings.apiKeys.empty')"
      :empty-description="t('settings.apiKeys.emptyDesc')"
      empty-icon="i-lucide-key-round"
    >
      <template #cell-name="{ row }">
        <div class="font-semibold" style="color: var(--cr-text)">{{ row.name }}</div>
      </template>
      <template #cell-key="{ row }">
        <code class="px-2 py-1 rounded text-[11.5px] font-mono" style="background: var(--cr-surface-soft); color: var(--cr-text)">
          cr_{{ row.keyPrefix }}_••••
        </code>
      </template>
      <template #cell-scopes="{ row }">
        <div class="flex flex-wrap gap-1">
          <span v-for="s in row.scopes" :key="s" class="text-[10.5px] font-semibold uppercase tracking-wider px-1.5 py-0.5 rounded" style="background: var(--cr-surface-soft); color: var(--cr-text-muted)">{{ s }}</span>
        </div>
      </template>
      <template #cell-status="{ row }">
        <StatusBadge :state="statusOf(row)" />
      </template>
      <template #cell-expires="{ row }">
        <span class="text-[12px] tabular-nums" style="color: var(--cr-text-muted)">{{ fmt(row.expiresAt) }}</span>
      </template>
      <template #cell-used="{ row }">
        <span class="text-[12px] tabular-nums" style="color: var(--cr-text-muted)">{{ fmt(row.lastUsedAt) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="flex items-center justify-end" @click.stop>
          <button v-if="!isRevoked(row)" class="cr-row-action cr-row-action--danger" :title="t('settings.apiKeys.revoke')" @click="askRevoke(row)">
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
        <div v-if="formOpen" class="fixed inset-0 z-50 flex items-center justify-center px-4 bg-black/50 backdrop-blur-sm" @click.self="formOpen = false; newKey = null">
          <div class="cr-card max-w-md w-full p-6 cr-anim-fade-up">
            <div v-if="!newKey">
              <h3 class="text-[18px] font-bold tracking-tight" style="color: var(--cr-text)">{{ t('settings.apiKeys.new') }}</h3>
              <p class="text-[13px] mt-1" style="color: var(--cr-text-muted)">{{ t('settings.apiKeys.modalDesc') }}</p>

              <div class="mt-5 space-y-4">
                <div>
                  <label class="cr-label">{{ t('settings.apiKeys.colName') }}</label>
                  <input v-model="form.name" type="text" :placeholder="t('settings.apiKeys.namePlaceholder')" class="cr-input !pl-4" />
                </div>
                <div>
                  <label class="cr-label">{{ t('settings.apiKeys.ttlLabel') }}</label>
                  <input v-model.number="form.ttlDays" type="number" min="0" class="cr-input !pl-4" />
                </div>
                <div>
                  <label class="cr-label">{{ t('settings.apiKeys.scopesLabel') }}</label>
                  <div class="flex flex-wrap gap-2">
                    <label v-for="s in ['render', 'read', 'write']" :key="s" class="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg cursor-pointer transition-colors duration-150"
                      :style="form.scopes.includes(s) ? 'background: var(--color-wise-100); color: var(--color-wise-800)' : 'background: var(--cr-surface-soft); color: var(--cr-text-muted)'">
                      <input type="checkbox" :checked="form.scopes.includes(s)" class="cr-checkbox"
                        @change="form.scopes = form.scopes.includes(s) ? form.scopes.filter(x => x !== s) : [...form.scopes, s]" />
                      <span class="text-[12px] font-semibold uppercase">{{ s }}</span>
                    </label>
                  </div>
                </div>
              </div>

              <div class="mt-6 flex items-center justify-end gap-2">
                <button class="cr-btn-secondary" @click="formOpen = false">{{ t('common.cancel') }}</button>
                <button class="cr-btn-primary !w-auto" @click="create">{{ t('common.create') }}</button>
              </div>
            </div>

            <div v-else>
              <h3 class="text-[18px] font-bold tracking-tight" style="color: var(--cr-text)">{{ t('settings.apiKeys.createdTitle') }}</h3>
              <p class="text-[13px] mt-1" style="color: var(--cr-text-muted)">{{ t('settings.apiKeys.createdDesc') }}</p>

              <div class="mt-4 p-3 rounded-xl border-2 border-dashed flex items-center gap-2" style="border-color: var(--color-wise-400); background: var(--color-wise-100)">
                <code class="flex-1 text-[12.5px] font-mono break-all" style="color: var(--color-wise-900)">{{ newKey }}</code>
                <button class="cr-icon-btn !w-9 !h-9 shrink-0" :title="t('common.copy')" @click="copyKey">
                  <UIcon name="i-lucide-copy" class="w-4 h-4" />
                </button>
              </div>

              <div class="mt-6 flex justify-end">
                <button class="cr-btn-primary !w-auto" @click="formOpen = false; newKey = null">{{ t('settings.apiKeys.done') }}</button>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <ConfirmDialog v-model="confirmOpen" :title="t('settings.apiKeys.confirmTitle')" :description="`${t('settings.apiKeys.confirmDescPre')} &quot;${toRevoke?.name}&quot; ${t('settings.apiKeys.confirmDescPost')}`" destructive :confirm-label="t('settings.apiKeys.revoke')" @confirm="doRevoke" />
  </div>
</template>
