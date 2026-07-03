<script setup lang="ts">
const assets = useEntity('assets')
const api = useApi()
const toasts = useToasts()
const { t } = useI18n()

const rows = ref<any[]>([])
const loading = ref(true)
const uploading = ref(false)
const dragOver = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const search = ref('')

const toDelete = ref<any>(null)
const confirmOpen = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await assets.list({ $top: 200 })
    rows.value = res.value ?? []
  } catch (err: any) {
    toasts.error(t('assets.couldNotLoad'), extractError(err))
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

async function upload(files: FileList | File[]) {
  if (!files.length) return
  uploading.value = true
  const fd = new FormData()
  for (const f of Array.from(files)) fd.append('files', f)
  try {
    await api.post('/api/assets/upload', fd)
    toasts.success(`${files.length} archivo${files.length > 1 ? 's' : ''} subido${files.length > 1 ? 's' : ''}`)
    await load()
  } catch (err: any) {
    toasts.error(t('assets.couldNotUpload'), extractError(err))
  } finally {
    uploading.value = false
  }
}

function onDrop(e: DragEvent) {
  e.preventDefault()
  dragOver.value = false
  if (e.dataTransfer?.files?.length) upload(e.dataTransfer.files)
}

function onPickFiles() {
  fileInput.value?.click()
}
function onFileChange(e: Event) {
  const t = e.target as HTMLInputElement
  if (t.files) upload(t.files)
  t.value = ''
}

function askDelete(row: any) {
  toDelete.value = row
  confirmOpen.value = true
}
async function doDelete() {
  if (!toDelete.value) return
  try {
    await assets.remove(toDelete.value.shortid)
    rows.value = rows.value.filter(r => r.shortid !== toDelete.value.shortid)
    toasts.success(t('assets.deleted'))
  } catch (err: any) {
    toasts.error(t('assets.couldNotDelete'), extractError(err))
  } finally {
    toDelete.value = null
  }
}

function isImage(mime?: string) { return mime?.startsWith('image/') }
function isPdf(mime?: string)  { return mime === 'application/pdf' }
function iconFor(mime?: string) {
  if (!mime) return 'i-lucide-file'
  if (mime.startsWith('image/')) return 'i-lucide-image'
  if (mime === 'application/pdf') return 'i-lucide-file-text'
  if (mime.includes('spreadsheet')) return 'i-lucide-sheet'
  if (mime.includes('wordprocessing')) return 'i-lucide-file-text'
  if (mime.startsWith('text/')) return 'i-lucide-file-text'
  return 'i-lucide-file'
}
function fmtBytes(n?: number) {
  if (n == null) return '—'
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}
function assetUrl(row: any) {
  const cfg = useRuntimeConfig()
  const auth = useAuthStore()
  // The asset content endpoint doesn't accept query auth; we'd need a stamp.
  // For UI preview of images we just render the URL — backend has CORS *.
  return `${cfg.public.apiBase}/assets/${row.shortid}/content`
}
</script>

<template>
  <div class="cr-anim-fade-up">
    <PageHeader
      icon="i-lucide-image"
      :title="t('assets.title')"
      :description="t('assets.description')"
    >
      <template #actions>
        <button class="cr-btn-primary !w-auto" :disabled="uploading" @click="onPickFiles">
          <UIcon name="i-lucide-upload" class="w-4 h-4" />
          <span>{{ t('assets.upload') }}</span>
        </button>
        <input ref="fileInput" type="file" multiple class="hidden" @change="onFileChange" />
      </template>
    </PageHeader>

    <!-- Dropzone -->
    <div
      class="cr-card border-dashed border-2 mb-4 transition-colors duration-150"
      :class="dragOver ? 'cr-dropzone--active' : ''"
      style="border-color: var(--cr-border-strong); padding: 28px;"
      @dragover.prevent="dragOver = true"
      @dragleave="dragOver = false"
      @drop="onDrop"
    >
      <div class="flex items-center justify-center gap-4 text-center">
        <span class="w-12 h-12 rounded-xl flex items-center justify-center shrink-0" style="background: var(--color-wise-100); color: var(--color-wise-700)">
          <UIcon :name="uploading ? 'i-lucide-loader-2' : 'i-lucide-cloud-upload'" class="w-6 h-6" :class="uploading ? 'cr-anim-spin' : ''" />
        </span>
        <div class="text-left">
          <p class="font-semibold text-[14px]" style="color: var(--cr-text)">
            {{ uploading ? t('common.uploading') : t('assets.dropHere') }}
          </p>
          <p class="text-[12.5px] mt-0.5" style="color: var(--cr-text-muted)">
            {{ t('assets.or') }}
            <button class="font-semibold underline decoration-dotted transition-colors hover:text-wise-700" style="color: var(--cr-text)" @click="onPickFiles">
              {{ t('assets.pickFromPc') }}
            </button>
            {{ t('assets.multipleOk') }}
          </p>
        </div>
      </div>
    </div>

    <!-- Search + grid -->
    <div class="relative max-w-md mb-4">
      <UIcon name="i-lucide-search" class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4" style="color: var(--cr-text-soft)" />
      <input v-model="search" type="text" :placeholder="t('assets.searchPlaceholder')" class="cr-input !pl-10 !py-2 !text-[13px]" style="padding-top: 8px; padding-bottom: 8px" />
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
      <div v-for="i in 10" :key="i" class="cr-card overflow-hidden">
        <div class="aspect-square cr-skeleton" />
        <div class="p-3 flex flex-col gap-1">
          <div class="h-3 w-3/4 cr-skeleton rounded" />
          <div class="flex items-center justify-between">
            <div class="h-2.5 w-1/4 cr-skeleton rounded" />
            <div class="h-2.5 w-1/3 cr-skeleton rounded" />
          </div>
          <div class="flex items-center gap-1 mt-2">
            <div class="flex-1 h-7 cr-skeleton rounded" />
            <div class="flex-1 h-7 cr-skeleton rounded" />
          </div>
        </div>
      </div>
    </div>

    <EmptyState
      v-else-if="filtered.length === 0"
      icon="i-lucide-image"
      :title="t('assets.empty')"
      :description="t('assets.emptyDesc')"
    />

    <div v-else class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
      <div
        v-for="row in filtered"
        :key="row.shortid"
        class="cr-card overflow-hidden cr-anim-fade-up"
      >
        <div class="aspect-square flex items-center justify-center" style="background: var(--cr-surface-soft)">
          <img
            v-if="isImage(row.mimeType)"
            :src="assetUrl(row)"
            :alt="row.name"
            class="max-w-full max-h-full object-contain"
            loading="lazy"
          />
          <UIcon
            v-else
            :name="iconFor(row.mimeType)"
            class="w-12 h-12"
            style="color: var(--cr-text-soft)"
          />
        </div>
        <div class="p-3 flex flex-col gap-1">
          <div class="font-semibold text-[12.5px] truncate" :title="row.name" style="color: var(--cr-text)">
            {{ row.name }}
          </div>
          <div class="flex items-center justify-between text-[11px]" style="color: var(--cr-text-muted)">
            <span>{{ fmtBytes(row.size) }}</span>
            <span class="font-mono">{{ row.shortid }}</span>
          </div>
          <div class="flex items-center gap-1 mt-2">
            <a :href="assetUrl(row)" target="_blank" class="cr-row-action flex-1" :title="t('assets.open')">
              <UIcon name="i-lucide-external-link" class="w-4 h-4" />
            </a>
            <button class="cr-row-action cr-row-action--danger flex-1" :title="t('assets.delete')" @click="askDelete(row)">
              <UIcon name="i-lucide-trash-2" class="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <ConfirmDialog
      v-model="confirmOpen"
      :title="t('assets.confirmTitle')"
      :description="`${t('list.willDelete')} &quot;${toDelete?.name}&quot;.`"
      destructive
      :confirm-label="t('list.delete')"
      @confirm="doDelete"
    />
  </div>
</template>

<style>
.cr-dropzone--active {
  background: var(--color-wise-100) !important;
  border-color: var(--color-wise-500) !important;
}
</style>
