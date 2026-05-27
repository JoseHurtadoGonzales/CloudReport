<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

const auth = useAuthStore()
const api = useApi()
const cfg = useRuntimeConfig()
const toasts = useToasts()
const { t } = useI18n()

interface Counts {
  templates: number
  assets: number
  schedulesActive: number
  reports: number
}

const counts = ref<Counts>({ templates: 0, assets: 0, schedulesActive: 0, reports: 0 })
const recentReports = ref<any[]>([])
const recentProfiles = ref<any[]>([])
const recipes = ref<string[]>([])
const allTemplates = ref<{ shortid: string; name: string; recipe?: string }[]>([])
const loading = ref(true)

// Derived: success rate over the recent renders we just pulled. We compute it
// client-side because the API doesn't expose an aggregate endpoint yet.
const successRate = computed(() => {
  const sample = recentProfiles.value
  if (sample.length === 0) return null
  const ok = sample.filter(p => p.state === 'success').length
  return Math.round((ok / sample.length) * 100)
})
const errorCount = computed(() => recentProfiles.value.filter(p => p.state === 'error').length)

async function load() {
  loading.value = true
  try {
    const [tpls, ast, sch, rep, rec, profs] = await Promise.all([
      api.get<{ '@odata.count': number; value: any[] }>('/odata/templates', { query: { $top: 50, $count: 'true', $orderby: 'updated_at desc', $select: 'shortid,name,recipe' } }),
      api.get<{ '@odata.count': number }>('/odata/assets',    { query: { $top: 0, $count: 'true' } }),
      api.get<{ '@odata.count': number; value: any[] }>('/odata/schedules', { query: { $top: 200, $count: 'true', $select: 'enabled' } }),
      api.get<{ '@odata.count': number; value: any[] }>('/odata/reports', { query: { $top: 6, $count: 'true', $orderby: 'created_at desc' } }),
      api.get<string[]>('/api/recipe'),
      // Pull the last 20 profiles so we can compute a meaningful success rate
      // — 5 is too small a sample. We still only render the top 5 in the list.
      api.get<{ value: any[] }>('/odata/profiles', { query: { $top: 20, $orderby: 'started_at desc' } }),
    ])
    const enabledSchedules = (sch.value ?? []).filter((s: any) => s.enabled).length
    counts.value = {
      templates: tpls['@odata.count'] ?? 0,
      assets:    ast['@odata.count'] ?? 0,
      schedulesActive: enabledSchedules,
      reports:   rep['@odata.count'] ?? 0,
    }
    allTemplates.value = tpls.value ?? []
    recentReports.value = rep.value ?? []
    recipes.value = rec ?? []
    recentProfiles.value = profs.value ?? []
  } catch (err: any) {
    toasts.error(t('dashboard.loadError') || 'No se pudo cargar', extractError(err))
  } finally {
    loading.value = false
  }
}
onMounted(load)

const isEmpty = computed(() => counts.value.templates === 0)

function fmtDate(iso?: string) {
  if (!iso) return '—'
  const d = new Date(iso)
  const diff = (Date.now() - d.getTime()) / 1000
  if (diff < 60)      return t('dashboard.timeJustNow')
  if (diff < 3600)    return t('dashboard.timeMinAgo').replace('{n}', String(Math.floor(diff / 60)))
  if (diff < 86400)   return t('dashboard.timeHAgo').replace('{n}', String(Math.floor(diff / 3600)))
  return d.toLocaleDateString()
}

const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 12) return t('dashboard.greetingMorning')
  if (h < 19) return t('dashboard.greetingAfter')
  return t('dashboard.greetingEvening')
})

// Resolve a profile's template shortid → readable name for the recent-renders
// list. Falls back to the shortid itself if we don't have it in the
// already-loaded templates list (avoids extra round trips).
function templateName(shortid?: string) {
  if (!shortid) return '—'
  const found = allTemplates.value.find(t => t.shortid === shortid)
  return found?.name ?? shortid
}

// Download a report by streaming its blob through the API endpoint. Doing it
// via fetch (instead of <a href>) keeps the Authorization header attached.
async function downloadReport(shortid: string, name?: string) {
  try {
    const res = await fetch(`${cfg.public.apiBase}/reports/${shortid}/content`, {
      headers: auth.token ? { Authorization: `Bearer ${auth.token}` } : {},
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = name || shortid
    a.click()
    URL.revokeObjectURL(url)
  } catch (err: any) {
    toasts.error(t('common.couldNotLoad'), extractError(err))
  }
}

// ── Quick-render widget ──────────────────────────────────────────────────
// Pick any template + click Render. We hit the same /api/report endpoint
// the editor uses and open the resulting file in a new tab. No data
// payload: the template is rendered with `{}`.
const quickShortid = ref<string>('')
const quickRendering = ref(false)

async function quickRender() {
  if (!quickShortid.value) return
  quickRendering.value = true
  try {
    const res = await fetch(`${cfg.public.apiBase}/api/report`, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
        ...(auth.token ? { Authorization: `Bearer ${auth.token}` } : {}),
      },
      body: JSON.stringify({ template: { shortid: quickShortid.value }, data: {} }),
    })
    if (!res.ok) {
      const txt = await res.text()
      let msg = txt
      try { msg = JSON.parse(txt).error ?? txt } catch {}
      throw new Error(msg)
    }
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    window.open(url, '_blank')
    // Don't revoke yet — the new tab needs it. The browser GCs it on close.
    toasts.success(t('dashboard.quickOk'))
    // Refresh stats so the new render appears immediately.
    load()
  } catch (err: any) {
    toasts.error(t('dashboard.quickFail'), extractError(err))
  } finally {
    quickRendering.value = false
  }
}
</script>

<template>
  <div class="cr-anim-fade-up">
    <!-- Welcome hero (only when there's nothing yet) -->
    <section v-if="isEmpty && !loading" class="cr-card overflow-hidden mb-6 relative">
      <!-- Animated backdrop -->
      <div class="absolute inset-0 pointer-events-none overflow-hidden" aria-hidden="true">
        <div class="absolute -top-32 -right-32 w-[420px] h-[420px] rounded-full opacity-50 cr-anim-drift-1"
             style="background: radial-gradient(circle, #9fe870 0%, transparent 70%); filter: blur(60px);"></div>
        <div class="absolute -bottom-32 -left-32 w-[360px] h-[360px] rounded-full opacity-40 cr-anim-drift-2"
             style="background: radial-gradient(circle, #38c8ff 0%, transparent 70%); filter: blur(80px);"></div>
      </div>

      <div class="relative grid lg:grid-cols-[1.2fr_1fr] gap-8 p-8 lg:p-10 items-center">
        <div class="cr-stagger">
          <span class="cr-badge-prod self-start">
            <span class="cr-badge-prod-dot cr-anim-pulse-dot" />
            {{ greeting }}, {{ auth.user?.username ?? '' }}
          </span>
          <h1 class="text-[36px] sm:text-[44px] font-black tracking-tight leading-[1.05] mt-3" style="color: var(--cr-text)">
            {{ t('dashboard.heroTitle1') }}<br>
            {{ t('dashboard.heroTitle2') }}
          </h1>
          <p class="text-[15px] mt-3 max-w-md" style="color: var(--cr-text-muted)">
            {{ t('dashboard.heroDesc') }}
          </p>
          <div class="flex items-center gap-3 mt-6">
            <NuxtLink to="/templates/new" class="cr-btn-primary !w-auto">
              <UIcon name="i-lucide-plus" class="w-4 h-4" />
              <span>{{ t('dashboard.createTemplate') }}</span>
            </NuxtLink>
            <NuxtLink to="/settings/import-export" class="cr-btn-secondary">
              <UIcon name="i-lucide-package" class="w-4 h-4" />
              <span>{{ t('dashboard.importWorkspace') }}</span>
            </NuxtLink>
          </div>
        </div>

        <div class="hidden lg:block">
          <svg width="100%" height="280" viewBox="0 0 360 280" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
            <defs>
              <linearGradient id="paper" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" stop-color="#ffffff"/>
                <stop offset="100%" stop-color="#f5f7f3"/>
              </linearGradient>
            </defs>
            <rect x="60" y="20" width="240" height="240" rx="8" fill="url(#paper)" stroke="#0e0f0c" stroke-opacity="0.10"/>
            <rect x="80" y="40" width="120" height="6" rx="2" fill="#0e0f0c"/>
            <rect x="80" y="54" width="80" height="3" rx="1.5" fill="#0e0f0c" fill-opacity="0.4"/>
            <rect x="80" y="180" width="22" height="50" rx="3" fill="#9fe870"/>
            <rect x="108" y="160" width="22" height="70" rx="3" fill="#9fe870" fill-opacity="0.7"/>
            <rect x="136" y="195" width="22" height="35" rx="3" fill="#9fe870" fill-opacity="0.5"/>
            <rect x="164" y="140" width="22" height="90" rx="3" fill="#9fe870"/>
            <rect x="192" y="170" width="22" height="60" rx="3" fill="#9fe870" fill-opacity="0.7"/>
            <rect x="220" y="155" width="22" height="75" rx="3" fill="#9fe870" fill-opacity="0.85"/>
            <rect x="248" y="125" width="22" height="105" rx="3" fill="#9fe870"/>
            <rect x="80" y="80" width="200" height="3" rx="1.5" fill="#0e0f0c" fill-opacity="0.16"/>
            <rect x="80" y="90" width="180" height="3" rx="1.5" fill="#0e0f0c" fill-opacity="0.16"/>
            <rect x="80" y="100" width="160" height="3" rx="1.5" fill="#0e0f0c" fill-opacity="0.16"/>
            <circle cx="290" cy="50" r="22" fill="#9fe870" fill-opacity="0.20"/>
            <circle cx="290" cy="50" r="14" fill="#9fe870"/>
            <path d="M284 50l4 4 8-8" stroke="#0e0f0c" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
          </svg>
        </div>
      </div>
    </section>

    <!-- Welcome bar (returning user) -->
    <PageHeader
      v-else-if="!loading"
      :title="`${greeting}, ${auth.user?.username ?? ''}`"
      :description="t('dashboard.summary')"
    >
      <template #actions>
        <button
          type="button"
          class="cr-btn-secondary"
          :title="t('list.refresh')"
          @click="load"
        >
          <UIcon name="i-lucide-refresh-cw" class="w-4 h-4" :class="loading ? 'cr-anim-spin' : ''" />
          <span>{{ t('list.refresh') }}</span>
        </button>
        <NuxtLink to="/templates/new" class="cr-btn-primary !w-auto">
          <UIcon name="i-lucide-plus" class="w-4 h-4" />
          <span>{{ t('dashboard.newTemplate') }}</span>
        </NuxtLink>
      </template>
    </PageHeader>

    <!-- Stats / skeleton -->
    <div v-if="loading" class="grid grid-cols-2 lg:grid-cols-4 gap-4 cr-stagger">
      <div v-for="i in 4" :key="i" class="cr-card p-5">
        <div class="h-3 cr-skeleton w-1/2 rounded mb-3"></div>
        <div class="h-7 cr-skeleton w-2/3 rounded"></div>
      </div>
    </div>
    <div v-else class="grid grid-cols-2 lg:grid-cols-4 gap-4 cr-stagger">
      <NuxtLink to="/templates" class="cr-dash-stat-link">
        <StatCard :label="t('dashboard.statTemplates')" :value="counts.templates" icon="i-lucide-file-text" />
      </NuxtLink>
      <NuxtLink to="/assets" class="cr-dash-stat-link">
        <StatCard :label="t('dashboard.statAssets')" :value="counts.assets" icon="i-lucide-image" />
      </NuxtLink>
      <NuxtLink to="/schedules" class="cr-dash-stat-link">
        <StatCard
          :label="t('dashboard.statSchedulesActive')"
          :value="counts.schedulesActive"
          icon="i-lucide-calendar-clock"
        />
      </NuxtLink>
      <NuxtLink to="/reports" class="cr-dash-stat-link">
        <StatCard :label="t('dashboard.statReports')" :value="counts.reports" icon="i-lucide-file-down" />
      </NuxtLink>
    </div>

    <!-- Success-rate bar + recent renders summary -->
    <section v-if="!loading && successRate !== null" class="cr-card p-5 mt-6">
      <div class="flex flex-wrap items-center gap-5">
        <div class="flex items-center gap-3 flex-1 min-w-[220px]">
          <span class="cr-dash-ring" :style="{ '--pct': successRate + '%' }">
            <span class="cr-dash-ring-text">{{ successRate }}%</span>
          </span>
          <div>
            <p class="text-[11px] font-semibold tracking-wide uppercase" style="color: var(--cr-text-soft)">
              {{ t('dashboard.successRate') }}
            </p>
            <p class="text-[13px]" style="color: var(--cr-text-muted)">
              {{ t('dashboard.lastN').replace('{n}', String(recentProfiles.length)) }}
            </p>
          </div>
        </div>

        <div class="flex items-center gap-5 flex-wrap">
          <div>
            <p class="cr-eyebrow" style="color: var(--cr-text-soft)">{{ t('dashboard.errors') }}</p>
            <p class="text-[22px] font-bold tabular-nums" :style="{ color: errorCount > 0 ? '#d03238' : 'var(--cr-text)' }">
              {{ errorCount }}
            </p>
          </div>
          <div>
            <p class="cr-eyebrow" style="color: var(--cr-text-soft)">{{ t('dashboard.recipesCount') }}</p>
            <p class="text-[22px] font-bold tabular-nums" style="color: var(--cr-text)">{{ recipes.length }}</p>
          </div>
          <NuxtLink to="/profiles" class="cr-btn-secondary !text-[12px] !py-1.5 !px-3 ml-auto">
            <UIcon name="i-lucide-activity" class="w-3.5 h-3.5" />
            {{ t('dashboard.viewProfiles') }}
          </NuxtLink>
        </div>
      </div>
    </section>

    <!-- Quick-render widget -->
    <section v-if="!loading && allTemplates.length > 0" class="cr-card p-5 mt-6">
      <div class="flex items-start gap-3 mb-3">
        <span class="cr-dash-quick-icon">
          <UIcon name="i-lucide-zap" class="w-4 h-4" />
        </span>
        <div class="flex-1">
          <p class="text-[14px] font-bold" style="color: var(--cr-text)">{{ t('dashboard.quickRender') }}</p>
          <p class="text-[12px]" style="color: var(--cr-text-muted)">{{ t('dashboard.quickRenderHint') }}</p>
        </div>
      </div>
      <div class="flex items-center gap-2 flex-wrap">
        <div class="flex-1 min-w-[240px]">
          <CrSelect
            v-model="quickShortid"
            :options="[
              { value: '', label: t('dashboard.pickTemplate') },
              ...allTemplates.map(t => ({ value: t.shortid, label: t.name, hint: t.recipe })),
            ]"
            :placeholder="t('dashboard.pickTemplate')"
            :searchable="allTemplates.length > 6"
          />
        </div>
        <button
          class="cr-btn-primary !w-auto"
          :disabled="!quickShortid || quickRendering"
          @click="quickRender"
        >
          <UIcon v-if="!quickRendering" name="i-lucide-play" class="w-4 h-4" />
          <span v-else class="w-4 h-4 border-2 rounded-full cr-anim-spin" style="border-color: currentColor; border-top-color: transparent" />
          <span>{{ quickRendering ? t('header.rendering') : t('header.render') }}</span>
        </button>
      </div>
    </section>

    <!-- Two columns: recent + quick actions -->
    <div class="grid lg:grid-cols-3 gap-4 mt-6">
      <div class="lg:col-span-2 space-y-4">
        <SectionCard :title="t('dashboard.recentReports')" icon="i-lucide-history">
          <template #headerActions>
            <NuxtLink to="/reports" class="text-[12px] font-semibold hover:underline" style="color: var(--cr-text); transition: color 140ms">{{ t('dashboard.viewAll') }}</NuxtLink>
          </template>
          <div v-if="loading" class="space-y-3 py-1">
            <div v-for="i in 3" :key="i" class="flex items-center gap-3">
              <div class="w-9 h-9 cr-skeleton rounded-lg shrink-0"></div>
              <div class="flex-1 space-y-1.5">
                <div class="h-3 cr-skeleton w-1/3 rounded"></div>
                <div class="h-2.5 cr-skeleton w-1/4 rounded"></div>
              </div>
            </div>
          </div>
          <EmptyState
            v-else-if="recentReports.length === 0"
            icon="i-lucide-file-down"
            :title="t('dashboard.noReports')"
            :description="t('dashboard.noReportsDesc')"
            class="!shadow-none !border-0 !bg-transparent !p-6"
          />
          <ul v-else class="divide-y" style="border-color: var(--cr-border)">
            <li
              v-for="r in recentReports"
              :key="r.shortid"
              class="py-3 flex items-center gap-3 cr-dash-row"
              :title="t('common.download')"
              @click="downloadReport(r.shortid, r.name)"
            >
              <span class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0" style="background: var(--cr-surface-soft)">
                <UIcon name="i-lucide-file-text" class="w-4 h-4" style="color: var(--cr-text-muted)" />
              </span>
              <div class="flex-1 min-w-0">
                <div class="font-semibold text-[13.5px] truncate" style="color: var(--cr-text)">
                  {{ r.name ?? r.shortid }}
                </div>
                <div class="text-[12px] truncate" style="color: var(--cr-text-muted)">
                  {{ fmtDate(r.creationDate) }} · {{ r.mimeType ?? '—' }}
                </div>
              </div>
              <StatusBadge :state="r.state" />
              <UIcon name="i-lucide-download" class="w-3.5 h-3.5 opacity-0 cr-dash-row-icon" style="color: var(--cr-text-soft)" />
            </li>
          </ul>
        </SectionCard>

        <SectionCard :title="t('dashboard.recentRenders')" icon="i-lucide-activity">
          <template #headerActions>
            <NuxtLink to="/profiles" class="text-[12px] font-semibold hover:underline" style="color: var(--cr-text); transition: color 140ms">{{ t('dashboard.viewAll') }}</NuxtLink>
          </template>
          <div v-if="loading" class="space-y-3 py-1">
            <div v-for="i in 3" :key="i" class="flex items-center gap-3">
              <div class="w-9 h-9 cr-skeleton rounded-lg shrink-0"></div>
              <div class="flex-1 space-y-1.5">
                <div class="h-3 cr-skeleton w-1/4 rounded"></div>
                <div class="h-2.5 cr-skeleton w-1/3 rounded"></div>
              </div>
            </div>
          </div>
          <EmptyState
            v-else-if="recentProfiles.length === 0"
            icon="i-lucide-activity"
            :title="t('dashboard.noRenders')"
            :description="t('dashboard.noRendersDesc')"
            class="!shadow-none !border-0 !bg-transparent !p-6"
          />
          <ul v-else class="divide-y" style="border-color: var(--cr-border)">
            <li v-for="p in recentProfiles.slice(0, 5)" :key="p.shortid" class="py-3 flex items-center gap-3">
              <span class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0" style="background: var(--cr-surface-soft)">
                <UIcon name="i-lucide-activity" class="w-4 h-4" style="color: var(--cr-text-muted)" />
              </span>
              <div class="flex-1 min-w-0">
                <div class="font-semibold text-[13.5px] truncate" style="color: var(--cr-text)">
                  {{ templateName(p.templateShortid) }}
                </div>
                <div class="text-[12px] truncate" style="color: var(--cr-text-muted)">
                  {{ fmtDate(p.timestamp) }} · {{ t('dashboard.modeLabel') }} {{ p.mode ?? '—' }}
                </div>
              </div>
              <StatusBadge :state="p.state" />
            </li>
          </ul>
        </SectionCard>
      </div>

      <div class="space-y-4">
        <SectionCard :title="t('dashboard.quickActions')" icon="i-lucide-zap">
          <ul class="space-y-1">
            <li>
              <NuxtLink to="/templates/new" class="cr-quick-action">
                <UIcon name="i-lucide-file-plus" class="w-4 h-4" />
                <span>{{ t('dashboard.newTemplate') }}</span>
                <UIcon name="i-lucide-chevron-right" class="w-3.5 h-3.5 ml-auto" style="color: var(--cr-text-soft)" />
              </NuxtLink>
            </li>
            <li>
              <NuxtLink to="/assets" class="cr-quick-action">
                <UIcon name="i-lucide-upload" class="w-4 h-4" />
                <span>{{ t('dashboard.uploadAsset') }}</span>
                <UIcon name="i-lucide-chevron-right" class="w-3.5 h-3.5 ml-auto" style="color: var(--cr-text-soft)" />
              </NuxtLink>
            </li>
            <li>
              <NuxtLink to="/settings/api-keys" class="cr-quick-action">
                <UIcon name="i-lucide-key-round" class="w-4 h-4" />
                <span>{{ t('dashboard.createApiKey') }}</span>
                <UIcon name="i-lucide-chevron-right" class="w-3.5 h-3.5 ml-auto" style="color: var(--cr-text-soft)" />
              </NuxtLink>
            </li>
            <li>
              <NuxtLink to="/settings/import-export" class="cr-quick-action">
                <UIcon name="i-lucide-package" class="w-4 h-4" />
                <span>{{ t('dashboard.importWorkspace') }}</span>
                <UIcon name="i-lucide-chevron-right" class="w-3.5 h-3.5 ml-auto" style="color: var(--cr-text-soft)" />
              </NuxtLink>
            </li>
          </ul>
        </SectionCard>

        <SectionCard :title="t('dashboard.recipes')" icon="i-lucide-layers" :description="t('dashboard.recipesDesc')">
          <div class="flex flex-wrap gap-1.5">
            <RecipePill v-for="r in recipes" :key="r" :recipe="r" />
          </div>
        </SectionCard>
      </div>
    </div>
  </div>
</template>

<style>
.cr-quick-action {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 10px;
  font-size: 13.5px;
  font-weight: 500;
  color: var(--cr-text);
  transition: background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
              transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-quick-action:hover { background: var(--cr-surface-soft); }
.cr-quick-action:active { transform: scale(0.99); }

.cr-skeleton {
  position: relative;
  overflow: hidden;
  background: var(--cr-surface-soft);
}
.cr-skeleton::after {
  content: "";
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, transparent 0%, rgb(255 255 255 / 0.35) 50%, transparent 100%);
  animation: cr-skeleton-shimmer 1.4s linear infinite;
  transform: translateX(-100%);
}
html.dark .cr-skeleton::after {
  background: linear-gradient(90deg, transparent 0%, rgb(255 255 255 / 0.06) 50%, transparent 100%);
}
@keyframes cr-skeleton-shimmer { to { transform: translateX(100%); } }

/* Stat cards as full-block links — keep the StatCard look but make the whole
   tile feel clickable on hover. */
.cr-dash-stat-link {
  display: block;
  border-radius: 14px;
  transition: transform 140ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-dash-stat-link:hover { transform: translateY(-2px); }
.cr-dash-stat-link:active { transform: translateY(0) scale(0.99); }

/* Conic success-rate ring */
.cr-dash-ring {
  position: relative;
  width: 60px;
  height: 60px;
  border-radius: 999px;
  background: conic-gradient(var(--color-wise-500) var(--pct, 0%), var(--cr-border) 0);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.cr-dash-ring::after {
  content: '';
  position: absolute;
  inset: 5px;
  background: var(--cr-surface);
  border-radius: inherit;
}
.cr-dash-ring-text {
  position: relative;
  z-index: 1;
  font-size: 13px;
  font-weight: 800;
  color: var(--cr-text);
  font-variant-numeric: tabular-nums;
}

/* Quick render section accent icon */
.cr-dash-quick-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--color-wise-100);
  color: var(--color-wise-800);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
html.dark .cr-dash-quick-icon {
  background: rgb(159 232 112 / 0.14);
  color: var(--color-wise-300);
}

/* Recent report rows: hoverable + reveal download icon on hover */
.cr-dash-row {
  cursor: pointer;
  border-radius: 8px;
  padding-left: 4px;
  padding-right: 6px;
  margin-left: -4px;
  margin-right: -6px;
  transition: background-color 140ms ease;
}
.cr-dash-row:hover { background: var(--cr-surface-soft); }
.cr-dash-row:hover .cr-dash-row-icon { opacity: 1; }
.cr-dash-row-icon {
  transition: opacity 140ms ease;
}
</style>
