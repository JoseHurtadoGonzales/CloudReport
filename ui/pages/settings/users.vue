<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const api = useApi()
const toasts = useToasts()
const auth = useAuthStore()
const { t } = useI18n()

interface UserRow {
  shortid: string
  username: string
  email?: string
  isAdmin?: boolean
  creationDate?: string
}

const rows = ref<UserRow[]>([])
const loading = ref(true)
const formOpen = ref(false)
const form = ref({ username: '', email: '', password: '', isAdmin: false })
const creating = ref(false)
// DataTable slot rows are typed as Record<string, any>; keep this loose so we
// can accept them directly without casting at every call site.
const confirmDelete = ref<any | null>(null)

const isAdmin = computed(() => !!auth.user?.isAdmin)

async function load() {
  loading.value = true
  try {
    const r = await api.get<{ value: UserRow[] }>('/odata/users', {
      query: { $orderby: 'created_at desc' },
    })
    rows.value = r.value ?? []
  } catch (err: any) {
    toasts.error(t('list.couldNotLoadMany'), extractError(err))
  } finally { loading.value = false }
}
onMounted(load)

function openCreate() {
  form.value = { username: '', email: '', password: '', isAdmin: false }
  formOpen.value = true
}

async function create() {
  if (!form.value.username || form.value.password.length < 6) {
    toasts.error(t('settings.users.validation'))
    return
  }
  creating.value = true
  try {
    // Admin endpoint — works regardless of AllowRegistration and lets us
    // set the isAdmin flag at create time.
    await api.post('/api/admin/users', form.value)
    toasts.success(t('settings.users.created'))
    formOpen.value = false
    await load()
  } catch (err: any) {
    toasts.error(t('common.couldNotCreate'), extractError(err))
  } finally {
    creating.value = false
  }
}

async function toggleAdmin(row: any) {
  // Self-demote guard mirrors the backend — feedback before the request.
  if (row.shortid === auth.user?.shortid && row.isAdmin) {
    toasts.error(t('settings.users.cantDemoteSelf'))
    return
  }
  try {
    await api.patch(`/api/admin/users/${row.shortid}`, { isAdmin: !row.isAdmin })
    toasts.success(t('common.saved'))
    await load()
  } catch (err: any) {
    toasts.error(t('common.couldNotSave'), extractError(err))
  }
}

async function doDelete() {
  if (!confirmDelete.value) return
  const row = confirmDelete.value
  confirmDelete.value = null
  try {
    await api.delete(`/api/admin/users/${row.shortid}`)
    toasts.success(t('list.deleted'))
    await load()
  } catch (err: any) {
    toasts.error(t('common.couldNotDelete'), extractError(err))
  }
}

function fmt(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString()
}
</script>

<template>
  <div class="cr-anim-fade-up">
    <PageHeader
      icon="i-lucide-users"
      :title="t('settings.users.title')"
      :description="t('settings.users.description')"
    >
      <template #actions>
        <button
          class="cr-btn-primary !w-auto"
          :disabled="!isAdmin"
          :title="isAdmin ? t('settings.users.new') : t('settings.users.adminOnly')"
          @click="openCreate"
        >
          <UIcon name="i-lucide-plus" class="w-4 h-4" />
          <span>{{ t('settings.users.new') }}</span>
        </button>
      </template>
    </PageHeader>

    <!-- Non-admin notice -->
    <div v-if="!isAdmin && !loading" class="cr-card p-4 mb-4 flex items-start gap-3" style="background: rgb(255 165 100 / 0.10); border-color: rgb(255 165 100 / 0.35)">
      <UIcon name="i-lucide-shield-alert" class="w-5 h-5 shrink-0" style="color: #c46b22" />
      <div>
        <p class="text-[13px] font-semibold" style="color: var(--cr-text)">
          {{ t('settings.users.adminOnly') }}
        </p>
        <p class="text-[12px] mt-0.5" style="color: var(--cr-text-muted)">
          {{ t('settings.users.adminOnlyHint') }}
        </p>
      </div>
    </div>

    <DataTable
      :columns="[
        { key: 'username', label: t('settings.users.colUser') },
        { key: 'email', label: t('settings.users.colEmail') },
        { key: 'role', label: t('settings.users.colRole') },
        { key: 'created', label: t('settings.users.colCreated') },
        { key: 'actions', label: '' },
      ]"
      :rows="rows" :loading="loading"
      :empty-title="t('settings.users.empty')" empty-icon="i-lucide-users"
    >
      <template #cell-username="{ row }">
        <div class="flex items-center gap-3">
          <span class="w-8 h-8 rounded-full flex items-center justify-center text-[12px] font-bold" style="background: var(--color-wise-400); color: #0e0f0c">
            {{ row.username?.[0]?.toUpperCase() ?? '?' }}
          </span>
          <div>
            <div class="font-semibold inline-flex items-center gap-1.5" style="color: var(--cr-text)">
              {{ row.username }}
              <span
                v-if="row.shortid === auth.user?.shortid"
                class="cr-self-pill"
                :title="t('settings.users.youHint')"
              >{{ t('settings.users.you') }}</span>
            </div>
            <div class="text-[11.5px] font-mono" style="color: var(--cr-text-soft)">{{ row.shortid }}</div>
          </div>
        </div>
      </template>
      <template #cell-email="{ row }">
        <span class="text-[13px]" style="color: var(--cr-text-muted)">{{ row.email ?? '—' }}</span>
      </template>
      <template #cell-role="{ row }">
        <button
          type="button"
          class="cr-role-pill"
          :class="row.isAdmin ? 'cr-role-pill--admin' : ''"
          :disabled="!isAdmin"
          :title="isAdmin ? t('settings.users.toggleRole') : ''"
          @click="toggleAdmin(row)"
        >
          <UIcon :name="row.isAdmin ? 'i-lucide-shield' : 'i-lucide-user'" class="w-3 h-3" />
          {{ row.isAdmin ? t('settings.users.roleAdmin') : t('settings.users.roleUser') }}
        </button>
      </template>
      <template #cell-created="{ row }">
        <span class="text-[12px] tabular-nums" style="color: var(--cr-text-muted)">{{ fmt(row.creationDate) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="flex items-center justify-end gap-1">
          <button
            class="cr-row-action cr-row-action--danger"
            :disabled="!isAdmin || row.shortid === auth.user?.shortid"
            :title="row.shortid === auth.user?.shortid ? t('settings.users.cantDeleteSelf') : t('common.delete')"
            @click="confirmDelete = row"
          >
            <UIcon name="i-lucide-trash-2" class="w-4 h-4" />
          </button>
        </div>
      </template>
    </DataTable>

    <!-- Create user dialog -->
    <Teleport to="body">
      <Transition
        enter-active-class="transition-opacity duration-200"
        leave-active-class="transition-opacity duration-150"
        enter-from-class="opacity-0" leave-to-class="opacity-0"
      >
        <div v-if="formOpen" class="fixed inset-0 z-50 flex items-center justify-center px-4 bg-black/50 backdrop-blur-sm" @click.self="formOpen = false">
          <div class="cr-card max-w-md w-full p-6 cr-anim-fade-up">
            <h3 class="text-[18px] font-bold tracking-tight" style="color: var(--cr-text)">{{ t('settings.users.new') }}</h3>
            <div class="mt-5 space-y-4">
              <div>
                <label class="cr-label">{{ t('settings.users.fieldUser') }}</label>
                <input v-model="form.username" type="text" autocomplete="username" class="cr-input !pl-4" :placeholder="t('settings.users.fieldUser')" />
              </div>
              <div>
                <label class="cr-label">{{ t('settings.users.fieldEmail') }}</label>
                <input v-model="form.email" type="email" class="cr-input !pl-4" placeholder="user@example.com" />
              </div>
              <div>
                <label class="cr-label">{{ t('settings.users.fieldPassword') }}</label>
                <input v-model="form.password" type="password" autocomplete="new-password" class="cr-input !pl-4" :placeholder="t('settings.users.passwordHint')" />
              </div>
              <label class="flex items-center gap-2 cursor-pointer select-none">
                <input type="checkbox" v-model="form.isAdmin" class="cr-checkbox" />
                <span class="text-[13px] inline-flex items-center gap-1.5" style="color: var(--cr-text)">
                  <UIcon name="i-lucide-shield" class="w-3.5 h-3.5" />
                  {{ t('settings.users.makeAdmin') }}
                </span>
              </label>
            </div>
            <div class="mt-6 flex items-center justify-end gap-2">
              <button class="cr-btn-secondary" :disabled="creating" @click="formOpen = false">{{ t('common.cancel') }}</button>
              <button
                class="cr-btn-primary !w-auto"
                :disabled="creating || !form.username || form.password.length < 6"
                @click="create"
              >
                <UIcon v-if="!creating" name="i-lucide-check" class="w-4 h-4" />
                <span v-else class="w-4 h-4 border-2 rounded-full cr-anim-spin" style="border-color: currentColor; border-top-color: transparent"></span>
                <span>{{ creating ? t('common.creating') : t('common.create') }}</span>
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- Delete confirm dialog -->
    <ConfirmDialog
      :model-value="!!confirmDelete"
      :title="t('settings.users.deleteTitle')"
      :description="t('settings.users.deleteHint').replace('{user}', confirmDelete?.username ?? '')"
      :confirm-label="t('common.delete')"
      destructive
      @confirm="doDelete"
      @cancel="confirmDelete = null"
      @update:model-value="(v: boolean) => { if (!v) confirmDelete = null }"
    />
  </div>
</template>

<style>
/* Role pill — same shape as StatusBadge, but clickable to toggle admin. */
.cr-role-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 11.5px;
  font-weight: 700;
  border: 1px solid rgb(134 134 133 / 0.35);
  background: rgb(134 134 133 / 0.14);
  color: #4a4d4b;
  cursor: pointer;
  transition: background-color 140ms, border-color 140ms, color 140ms;
}
.cr-role-pill:hover:not(:disabled) { border-color: var(--cr-border-strong); }
.cr-role-pill:disabled { cursor: default; opacity: 0.7; }
.cr-role-pill--admin {
  background: rgb(159 232 112 / 0.18);
  border-color: rgb(159 232 112 / 0.45);
  color: #1f5b15;
}
html.dark .cr-role-pill {
  background: rgb(255 255 255 / 0.06);
  border-color: rgb(255 255 255 / 0.14);
  color: #c8ccc7;
}
html.dark .cr-role-pill--admin {
  background: rgb(159 232 112 / 0.16);
  border-color: rgb(159 232 112 / 0.50);
  color: #cdffad;
}

/* "Tú" / "You" badge next to the current user's name */
.cr-self-pill {
  display: inline-flex;
  align-items: center;
  padding: 1px 7px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  background: var(--color-wise-100);
  color: var(--color-wise-800);
  border: 1px solid var(--color-wise-500);
}
html.dark .cr-self-pill {
  background: rgb(159 232 112 / 0.18);
  color: var(--color-wise-300);
}
</style>
