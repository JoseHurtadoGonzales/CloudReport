<script setup lang="ts">
const templates = useEntity('templates')
const toasts = useToasts()
const api = useApi()
const router = useRouter()
const cfg = useRuntimeConfig()
const auth = useAuthStore()
const { t } = useI18n()

const rows = ref<any[]>([])
const loading = ref(true)
const search = ref('')
const recipeFilter = ref<string>('')
const recipes = ref<string[]>([])
const view = ref<'grid' | 'list'>('grid')

const toDelete = ref<any>(null)
const confirmOpen = ref(false)

// Render thumbnails. Map<shortid, blob url>
const previews = ref<Record<string, string>>({})

async function load() {
  loading.value = true
  try {
    const res = await templates.list({ $top: 200 })
    rows.value = res.value ?? []
    recipes.value = await api.get<string[]>('/api/recipe')
  } catch (err: any) {
    toasts.error(t('list.couldNotLoadMany'), extractError(err))
  } finally {
    loading.value = false
  }
}
onMounted(load)

onBeforeUnmount(() => {
  Object.values(previews.value).forEach(u => URL.revokeObjectURL(u))
})

const filtered = computed(() => {
  return rows.value.filter(r => {
    if (recipeFilter.value && r.recipe !== recipeFilter.value) return false
    if (search.value) {
      const q = search.value.toLowerCase()
      return (r.name?.toLowerCase().includes(q) || r.shortid?.toLowerCase().includes(q))
    }
    return true
  })
})

function askDelete(row: any) {
  toDelete.value = row
  confirmOpen.value = true
}

async function doDelete() {
  if (!toDelete.value) return
  try {
    await templates.remove(toDelete.value.shortid)
    rows.value = rows.value.filter(r => r.shortid !== toDelete.value.shortid)
    if (previews.value[toDelete.value.shortid]) {
      URL.revokeObjectURL(previews.value[toDelete.value.shortid])
      delete previews.value[toDelete.value.shortid]
    }
    toasts.success(t('templatesList.deleted'))
  } catch (err: any) {
    toasts.error(t('common.couldNotDelete'), extractError(err))
  } finally {
    toDelete.value = null
  }
}

const bulkRows = ref<any[]>([])
const bulkConfirmOpen = ref(false)
function askBulkDelete(selected: any[]) { bulkRows.value = selected; bulkConfirmOpen.value = true }
async function doBulkDelete() {
  let ok = 0
  for (const row of bulkRows.value.slice()) {
    try {
      await templates.remove(row.shortid)
      rows.value = rows.value.filter(r => r.shortid !== row.shortid)
      if (previews.value[row.shortid]) {
        URL.revokeObjectURL(previews.value[row.shortid])
        delete previews.value[row.shortid]
      }
      ok++
    } catch (err: any) { toasts.error(t('common.couldNotDelete'), extractError(err)) }
  }
  bulkRows.value = []
  if (ok > 0) toasts.success(t('templatesList.deleted'))
}

async function renderQuick(row: any, e: Event) {
  e.preventDefault()
  e.stopPropagation()
  try {
    const res = await fetch(`${cfg.public.apiBase}/api/report`, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
        ...(auth.token ? { Authorization: `Bearer ${auth.token}` } : {}),
      },
      body: JSON.stringify({ template: { shortid: row.shortid }, data: {} }),
    })
    if (!res.ok) throw new Error(await res.text())
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    // open in new tab
    window.open(url, '_blank')
  } catch (err: any) {
    toasts.error(t('templatesList.couldNotRender'), err.message)
  }
}

function fmtDate(iso?: string) {
  if (!iso) return '—'
  const d = new Date(iso)
  const diff = (Date.now() - d.getTime()) / 1000
  if (diff < 60)    return t('dashboard.timeJustNow')
  if (diff < 3600)  return t('dashboard.timeMinAgo').replace('{n}', String(Math.floor(diff / 60)))
  if (diff < 86400) return t('dashboard.timeHAgo').replace('{n}', String(Math.floor(diff / 3600)))
  if (diff < 7 * 86400) return `${Math.floor(diff / 86400)} d`
  return d.toLocaleDateString()
}

// Generate a preview gradient based on the template name (deterministic).
function bgForName(name?: string): string {
  if (!name) return 'linear-gradient(135deg, #e2f6d5 0%, #c5edab 100%)'
  let h = 0
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) & 0xfffffff
  const hue1 = h % 360
  const hue2 = (hue1 + 40) % 360
  return `linear-gradient(135deg, hsl(${hue1} 70% 88%) 0%, hsl(${hue2} 65% 78%) 100%)`
}
</script>

<template>
  <div class="cr-anim-fade-up">
    <PageHeader
      icon="i-lucide-file-text"
      :title="t('templatesList.title')"
      :description="t('templatesList.description')"
    >
      <template #actions>
        <div class="hidden md:flex items-center gap-1 p-0.5 rounded-lg" style="background: var(--cr-surface-soft); border: 1px solid var(--cr-border)">
          <button
            v-for="opt in [
              { id: 'grid', icon: 'i-lucide-layout-grid', title: t('templatesList.viewGrid') },
              { id: 'list', icon: 'i-lucide-list',        title: t('templatesList.viewList') },
            ]" :key="opt.id"
            class="w-8 h-8 rounded-md flex items-center justify-center"
            :class="view === opt.id ? 'cr-view-toggle--active' : ''"
            :title="opt.title"
            style="transition: background-color 140ms cubic-bezier(0.23,1,0.32,1), color 140ms"
            @click="view = opt.id as any"
          >
            <UIcon :name="opt.icon" class="w-4 h-4" />
          </button>
        </div>
        <NuxtLink to="/templates/new" class="cr-btn-primary !w-auto">
          <UIcon name="i-lucide-plus" class="w-4 h-4" />
          <span>{{ t('templatesList.new') }}</span>
        </NuxtLink>
      </template>
    </PageHeader>

    <!-- Filters -->
    <div class="flex items-center gap-3 flex-wrap mb-5">
      <div class="relative flex-1 max-w-md">
        <UIcon name="i-lucide-search" class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4" style="color: var(--cr-text-soft)" />
        <input
          v-model="search"
          type="text"
          :placeholder="t('templatesList.searchPlaceholder')"
          class="cr-input !pl-10 !py-2 !text-[13px]"
          style="padding-top: 8px; padding-bottom: 8px"
        />
      </div>
      <div class="flex items-center gap-1.5 flex-wrap">
        <button
          class="cr-chip"
          :class="recipeFilter === '' ? 'cr-chip--active' : ''"
          @click="recipeFilter = ''"
        >{{ t('templatesList.filterAll') }}</button>
        <button
          v-for="r in recipes"
          :key="r"
          class="cr-chip"
          :class="recipeFilter === r ? 'cr-chip--active' : ''"
          @click="recipeFilter = r"
        >{{ r }}</button>
      </div>
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 cr-stagger">
      <div v-for="i in 6" :key="i" class="cr-card overflow-hidden">
        <div class="aspect-[16/10] cr-skeleton" />
        <div class="p-4 space-y-2.5">
          <div class="h-4 cr-skeleton w-3/4 rounded" />
          <div class="h-3 cr-skeleton w-1/2 rounded" />
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <EmptyState
      v-else-if="filtered.length === 0 && !search"
      art="templates"
      :title="t('templatesList.emptyTitle')"
      :description="t('templatesList.emptyDesc')"
      :cta="{ label: t('templatesList.cta'), to: '/templates/new' }"
    />

    <EmptyState
      v-else-if="filtered.length === 0"
      art="search"
      :title="t('templatesList.noResults')"
      :description="`${t('templatesList.noResultsPre')} &quot;${search}&quot;. ${t('templatesList.noResultsPost')}`"
    />

    <!-- Grid view -->
    <div v-else-if="view === 'grid'" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      <article
        v-for="row in filtered"
        :key="row.shortid"
        class="cr-template-card group cursor-pointer"
        @click="router.push(`/templates/${row.shortid}`)"
      >
        <!-- Thumbnail (gradient placeholder) -->
        <div
          class="aspect-[16/10] relative overflow-hidden"
          :style="{ background: bgForName(row.name) }"
        >
          <!-- Subtle pattern overlay -->
          <div class="absolute inset-0" style="background-image: radial-gradient(rgb(0 0 0 / 0.04) 1px, transparent 1px); background-size: 12px 12px"></div>

          <!-- Mini "page" representation -->
          <div class="absolute inset-0 flex items-center justify-center p-6">
            <div class="w-full max-w-[160px] aspect-[8.5/11] bg-white shadow-lg rounded-sm p-3 cr-template-page">
              <div class="h-1.5 w-2/3 bg-[#0e0f0c] rounded-sm mb-2"></div>
              <div class="h-1 w-1/2 bg-[#868685] rounded-sm mb-3"></div>
              <div class="space-y-1">
                <div class="h-0.5 w-full bg-[#e8ebe6] rounded-sm"></div>
                <div class="h-0.5 w-5/6 bg-[#e8ebe6] rounded-sm"></div>
                <div class="h-0.5 w-4/6 bg-[#e8ebe6] rounded-sm"></div>
              </div>
              <div class="mt-2 h-3 w-1/3 bg-[#9fe870] rounded-sm"></div>
            </div>
          </div>

          <!-- Recipe pill -->
          <div class="absolute top-3 left-3">
            <RecipePill :recipe="row.recipe" solid />
          </div>

          <!-- Hover overlay actions -->
          <div class="cr-template-actions">
            <button
              type="button"
              class="cr-template-action"
              :title="t('templatesList.quickRender')"
              @click="(e) => renderQuick(row, e)"
            >
              <UIcon name="i-lucide-play" class="w-3.5 h-3.5" />
            </button>
            <button
              type="button"
              class="cr-template-action cr-template-action--danger"
              :title="t('templatesList.delete')"
              @click.stop="askDelete(row)"
            >
              <UIcon name="i-lucide-trash-2" class="w-3.5 h-3.5" />
            </button>
          </div>
        </div>

        <!-- Footer -->
        <div class="p-4 border-t" style="border-color: var(--cr-border)">
          <h3 class="font-bold text-[14px] truncate" style="color: var(--cr-text)" :title="row.name">
            {{ row.name }}
          </h3>
          <div class="flex items-center justify-between mt-1 text-[11.5px]" style="color: var(--cr-text-soft)">
            <span class="font-mono">{{ row.shortid }}</span>
            <span>{{ fmtDate(row.modificationDate) }}</span>
          </div>
        </div>
      </article>
    </div>

    <!-- List view -->
    <DataTable
      v-else
      :columns="[
        { key: 'name', label: t('templatesList.colName') },
        { key: 'recipe', label: t('templatesList.colRecipe') },
        { key: 'engine', label: t('templatesList.colEngine') },
        { key: 'updated', label: t('templatesList.colUpdated') },
        { key: 'actions', label: '', align: 'right', width: '120px' },
      ]"
      :rows="filtered"
      selectable
      :bulk-delete-label="t('templatesList.delete')"
      @row-click="(row) => router.push(`/templates/${row.shortid}`)"
      @bulk-delete="askBulkDelete"
    >
      <template #cell-name="{ row }">
        <div class="flex items-center gap-3">
          <span class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0" style="background: var(--cr-surface-soft)">
            <UIcon name="i-lucide-file-text" class="w-4 h-4" style="color: var(--cr-text-muted)" />
          </span>
          <div class="min-w-0">
            <div class="font-semibold truncate" style="color: var(--cr-text)">{{ row.name }}</div>
            <div class="text-[11.5px] font-mono" style="color: var(--cr-text-soft)">{{ row.shortid }}</div>
          </div>
        </div>
      </template>
      <template #cell-recipe="{ row }"><RecipePill :recipe="row.recipe" /></template>
      <template #cell-engine="{ row }">
        <span class="text-[12px] font-medium" style="color: var(--cr-text-muted)">{{ row.engine }}</span>
      </template>
      <template #cell-updated="{ row }">
        <span class="text-[12px] tabular-nums" style="color: var(--cr-text-muted)">{{ fmtDate(row.modificationDate) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="flex items-center justify-end gap-1" @click.stop>
          <button class="cr-row-action" :title="t('templatesList.quickRender')" @click="(e) => renderQuick(row, e)">
            <UIcon name="i-lucide-play" class="w-4 h-4" />
          </button>
          <NuxtLink :to="`/templates/${row.shortid}`" class="cr-row-action" :title="t('templatesList.edit')">
            <UIcon name="i-lucide-edit-3" class="w-4 h-4" />
          </NuxtLink>
          <button class="cr-row-action cr-row-action--danger" :title="t('templatesList.delete')" @click="askDelete(row)">
            <UIcon name="i-lucide-trash-2" class="w-4 h-4" />
          </button>
        </div>
      </template>
    </DataTable>

    <ConfirmDialog
      v-model="confirmOpen"
      :title="t('templatesList.confirmTitle')"
      :description="`${t('templatesList.confirmDescPre')} &quot;${toDelete?.name}&quot;.`"
      destructive
      :confirm-label="t('templatesList.delete')"
      @confirm="doDelete"
    />

    <ConfirmDialog
      v-model="bulkConfirmOpen"
      :title="t('templatesList.confirmTitle')"
      :description="`${t('templatesList.confirmDescPre')} ${bulkRows.length} ${bulkRows.length === 1 ? 'plantilla' : 'plantillas'}.`"
      destructive
      :confirm-label="t('templatesList.delete')"
      @confirm="doBulkDelete"
    />
  </div>
</template>

<style>
/* Template card */
.cr-template-card {
  background: var(--cr-surface);
  border: 1px solid var(--cr-border);
  border-radius: 18px;
  overflow: hidden;
  transition:
    transform 200ms cubic-bezier(0.23, 1, 0.32, 1),
    box-shadow 200ms cubic-bezier(0.23, 1, 0.32, 1),
    border-color 200ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-template-card:hover {
  transform: translateY(-2px);
  border-color: var(--cr-border-strong);
  box-shadow: 0 18px 50px -12px rgb(14 15 12 / 0.18), 0 4px 12px rgb(14 15 12 / 0.06);
}
.cr-template-card:active {
  transform: translateY(-1px) scale(0.99);
}
html.dark .cr-template-card:hover {
  box-shadow: 0 18px 50px -12px rgb(0 0 0 / 0.5), 0 4px 12px rgb(0 0 0 / 0.3);
}

/* Mini page that "lifts" on hover, like Notion / Linear */
.cr-template-page {
  transition: transform 280ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-template-card:hover .cr-template-page {
  transform: translateY(-4px) rotate(-1deg);
}

/* Hover-only actions overlay */
.cr-template-actions {
  position: absolute;
  top: 12px;
  right: 12px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transform: translateY(-4px);
  transition:
    opacity 200ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 200ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-template-card:hover .cr-template-actions,
.cr-template-card:focus-within .cr-template-actions {
  opacity: 1;
  transform: translateY(0);
}
.cr-template-action {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.94);
  color: #0e0f0c;
  border: 1px solid rgba(0, 0, 0, 0.06);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 4px 12px rgb(0 0 0 / 0.10);
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-template-action:hover {
  background: white;
}
.cr-template-action:active { transform: scale(0.94); }
.cr-template-action--danger {
  color: #a72027;
}
.cr-template-action--danger:hover {
  background: #fef5f5;
}

/* View toggle */
.cr-view-toggle--active {
  background: var(--cr-surface);
  color: var(--cr-text);
  box-shadow: 0 1px 2px rgb(14 15 12 / 0.08);
}

/* Loading skeleton */
.cr-skeleton {
  position: relative;
  overflow: hidden;
  background: var(--cr-surface-soft);
  border-radius: 6px;
}
.cr-skeleton::after {
  content: "";
  position: absolute;
  inset: 0;
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgb(255 255 255 / 0.35) 50%,
    transparent 100%
  );
  animation: cr-skeleton-shimmer 1.4s linear infinite;
  transform: translateX(-100%);
}
html.dark .cr-skeleton::after {
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgb(255 255 255 / 0.06) 50%,
    transparent 100%
  );
}
@keyframes cr-skeleton-shimmer {
  to { transform: translateX(100%); }
}

/* Reused row-action styles for the list view */
.cr-row-action {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--cr-text-muted);
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-row-action:hover { background: var(--cr-surface-soft); color: var(--cr-text); }
.cr-row-action:active { transform: scale(0.94); }
.cr-row-action--danger:hover { background: rgb(208 50 56 / 0.10); color: #a72027; }
</style>
