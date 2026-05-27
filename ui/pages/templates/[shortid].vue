<script setup lang="ts">
const route = useRoute()
const router = useRouter()
const toasts = useToasts()
const api = useApi()
const templates = useEntity('templates')
const split = useSplitRatio('cr-split-template')
const { t } = useI18n()

const isNew = computed(() => route.params.shortid === 'new')

const form = ref({
  shortid: '',
  name: 'sin nombre',
  content: '<h1>Hola {{name}}</h1>',
  css: 'body { font-family: Inter, sans-serif; color: #1a1a1a; }',
  engine: 'handlebars',
  recipe: 'html',
  helpers: '',
  pageSize: 'A4',
  pageOrientation: 'portrait',
  pageMargin: '1cm',
  /** days to keep generated reports before auto-delete (0 = forever) */
  reportRetentionDays: 30,
  /** array of script refs: [{ shortid, content? }] */
  scripts: [] as { shortid?: string; name?: string }[],
  /** linked data item shortid (optional) — its dataJson is used as base data for renders */
  dataShortid: '' as string | null | undefined,
  /** is the template publicly renderable without auth? */
  isPublic: false,
  /** Recipe-specific JSON option bags (mirror backend JSONB columns) */
  chrome:     {} as Record<string, any>,
  weasyprint: {} as Record<string, any>,
  docx:       {} as Record<string, any>,
  xlsx:       {} as Record<string, any>,
  pptx:       {} as Record<string, any>,
  htmlToXlsx: {} as Record<string, any>,
  /** PDF post-processing operations */
  pdfOperations: [] as any[],
})
const lastSavedSnapshot = ref<string>('')

const data = ref('{\n  "name": "World"\n}')
const renderLogs = ref('')
const renderProfile = ref<any | null>(null)

const recipes = ref<string[]>([])
const engines = ref<string[]>([])
const allScripts = ref<any[]>([])
const allDataItems = ref<any[]>([])
const allAssets = ref<any[]>([])
const allTemplates = ref<any[]>([])

type Tab = 'content' | 'css' | 'helpers' | 'data' | 'scripts' | 'page' | 'recipe' | 'pdfops'
const tab = ref<Tab>('content')

type PreviewTab = 'output' | 'source' | 'logs' | 'profile'
const previewTab = ref<PreviewTab>('output')

const saving = ref(false)
const rendering = ref(false)
const previewUrl = ref<string | null>(null)
const previewBlob = ref<Blob | null>(null)
const previewText = ref<string>('')
const previewKind = ref<'pdf' | 'html' | 'text' | 'binary'>('html')
const renderError = ref('')
const lastSavedAt = ref<Date | null>(null)
const drawerOpen = ref(false)
const fullscreenPreview = ref(false)

const tabs = computed<{ id: Tab; label: string; icon: string }[]>(() => {
  const base = [
    { id: 'content' as Tab, label: t('tab.content'), icon: 'i-lucide-code-xml' },
    { id: 'css'     as Tab, label: t('tab.styles'),  icon: 'i-lucide-paintbrush' },
    { id: 'helpers' as Tab, label: t('tab.helpers'), icon: 'i-lucide-braces' },
    { id: 'data'    as Tab, label: t('tab.data'),    icon: 'i-lucide-database' },
    { id: 'scripts' as Tab, label: t('tab.scripts'), icon: 'i-lucide-square-function' },
    { id: 'recipe'  as Tab, label: t('tab.recipe'),  icon: 'i-lucide-settings-2' },
    { id: 'page'    as Tab, label: t('tab.page'),    icon: 'i-lucide-file-cog' },
  ]
  // PDF ops tab only when the recipe produces PDF.
  if (['chrome-pdf', 'weasyprint', 'static-pdf'].includes(form.value.recipe)) {
    base.push({ id: 'pdfops' as Tab, label: t('tab.pdfOps'), icon: 'i-lucide-layers' })
  }
  return base
})

// Bound proxies — pick the right options bag depending on the active recipe.
const recipeBag = computed<Record<string, any>>({
  get() {
    const r = form.value.recipe
    if (r === 'chrome-pdf')    return form.value.chrome     ?? {}
    if (r === 'weasyprint')    return form.value.weasyprint ?? {}
    if (r === 'docx')          return form.value.docx       ?? {}
    if (r === 'xlsx')          return form.value.xlsx       ?? {}
    if (r === 'pptx')          return form.value.pptx       ?? {}
    if (r === 'html-to-xlsx')  return form.value.htmlToXlsx ?? {}
    return {}
  },
  set(v) {
    const r = form.value.recipe
    if (r === 'chrome-pdf')    form.value.chrome     = v
    else if (r === 'weasyprint') form.value.weasyprint = v
    else if (r === 'docx')       form.value.docx       = v
    else if (r === 'xlsx')       form.value.xlsx       = v
    else if (r === 'pptx')       form.value.pptx       = v
    else if (r === 'html-to-xlsx') form.value.htmlToXlsx = v
  },
})

function snapshot() { return JSON.stringify(form.value) }
const isDirty = computed(() => lastSavedSnapshot.value && snapshot() !== lastSavedSnapshot.value)

async function loadMeta() {
  const [r, e, scripts, dataItems, assets, templates] = await Promise.all([
    api.get<string[]>('/api/recipe'),
    api.get<string[]>('/api/engine'),
    api.get<{ value: any[] }>('/odata/scripts',  { query: { $top: 500, $select: 'name,shortid,scope' } }),
    api.get<{ value: any[] }>('/odata/data',     { query: { $top: 500, $select: 'name,shortid' } }),
    api.get<{ value: any[] }>('/odata/assets',   { query: { $top: 500, $select: 'name,shortid,mimeType' } }),
    api.get<{ value: any[] }>('/odata/templates',{ query: { $top: 500, $select: 'name,shortid,recipe' } }),
  ])
  recipes.value = r
  engines.value = e
  allScripts.value = scripts.value ?? []
  allDataItems.value = dataItems.value ?? []
  allAssets.value = assets.value ?? []
  allTemplates.value = templates.value ?? []
}

function parseJSONField(v: any, fallback: any) {
  if (v == null) return fallback
  if (typeof v === 'object') return v
  if (typeof v === 'string') {
    try { return JSON.parse(v) } catch { return fallback }
  }
  return fallback
}

async function loadTemplate() {
  if (isNew.value) {
    lastSavedSnapshot.value = ''
    return
  }
  try {
    const t = await templates.get<any>(route.params.shortid as string)
    let scripts: any[] = []
    try {
      scripts = typeof t.scripts === 'string' ? JSON.parse(t.scripts) : (Array.isArray(t.scripts) ? t.scripts : [])
    } catch { scripts = [] }
    form.value = {
      shortid: t.shortid,
      name: t.name,
      content: t.content ?? '',
      css: t.css ?? '',
      engine: t.engine,
      recipe: t.recipe,
      helpers: t.helpers ?? '',
      pageSize: t.pageSize || 'A4',
      pageOrientation: t.pageOrientation || 'portrait',
      pageMargin: t.pageMargin || '1cm',
      reportRetentionDays: t.reportRetentionDays ?? 30,
      scripts: scripts || [],
      dataShortid: t.dataShortid ?? '',
      isPublic: !!t.isPublic,
      chrome:     parseJSONField(t.chrome, {}),
      weasyprint: parseJSONField(t.weasyprint, {}),
      docx:       parseJSONField(t.docx, {}),
      xlsx:       parseJSONField(t.xlsx, {}),
      pptx:       parseJSONField(t.pptx, {}),
      htmlToXlsx: parseJSONField(t.htmlToXlsx, {}),
      pdfOperations: parseJSONField(t.pdfOperations, []),
    }
    lastSavedSnapshot.value = snapshot()
    if (t.modificationDate) lastSavedAt.value = new Date(t.modificationDate)
    // If a data item is linked, pre-fill the data tab with its dataJson.
    if (t.dataShortid) {
      try {
        const dItem = await api.get<any>(`/odata/data/${t.dataShortid}`)
        if (dItem?.dataJson) {
          const j = typeof dItem.dataJson === 'string' ? dItem.dataJson : JSON.stringify(dItem.dataJson, null, 2)
          data.value = j
        }
      } catch {}
    }
  } catch (err: any) {
    toasts.error('No se pudo cargar la plantilla', extractError(err))
    router.push('/templates')
  }
}

onMounted(async () => {
  await loadMeta()
  await loadTemplate()
})

async function save() {
  if (saving.value) return
  saving.value = true
  try {
    const payload = {
      name: form.value.name,
      content: form.value.content,
      css: form.value.css,
      engine: form.value.engine,
      recipe: form.value.recipe,
      helpers: form.value.helpers,
      pageSize: form.value.pageSize,
      pageOrientation: form.value.pageOrientation,
      pageMargin: form.value.pageMargin,
      reportRetentionDays: Number(form.value.reportRetentionDays) || 0,
      scripts: form.value.scripts,
      dataShortid: form.value.dataShortid || null,
      chrome:        Object.keys(form.value.chrome).length     ? form.value.chrome     : null,
      weasyprint:    Object.keys(form.value.weasyprint).length ? form.value.weasyprint : null,
      docx:          Object.keys(form.value.docx).length       ? form.value.docx       : null,
      xlsx:          Object.keys(form.value.xlsx).length       ? form.value.xlsx       : null,
      pptx:          Object.keys(form.value.pptx).length       ? form.value.pptx       : null,
      pdfOperations: form.value.pdfOperations,
    }
    if (isNew.value) {
      const t = await templates.create(payload)
      toasts.success('Plantilla creada')
      router.push(`/templates/${(t as any).shortid}`)
    } else {
      await templates.update(form.value.shortid, payload)
      lastSavedSnapshot.value = snapshot()
      lastSavedAt.value = new Date()
      toasts.success('Cambios guardados')
    }
  } catch (err: any) {
    toasts.error('No se pudo guardar', extractError(err))
  } finally {
    saving.value = false
  }
}

async function render() {
  if (rendering.value) return
  rendering.value = true
  renderError.value = ''
  renderLogs.value = ''
  renderProfile.value = null
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = null
  }
  try {
    let parsedData: any = {}
    try { parsedData = JSON.parse(data.value) } catch { throw new Error('La data debe ser JSON válido.') }
    if (isNew.value || isDirty.value) {
      await save()
      if (isNew.value) return
    }
    const cfg = useRuntimeConfig()
    const auth = useAuthStore()
    const res = await fetch(`${cfg.public.apiBase}/api/report`, {
      method: 'POST',
      headers: {
        'content-type': 'application/json',
        ...(auth.token ? { Authorization: `Bearer ${auth.token}` } : {}),
      },
      body: JSON.stringify({
        template: { shortid: form.value.shortid },
        data: parsedData,
      }),
    })
    if (!res.ok) {
      const txt = await res.text()
      try { renderError.value = JSON.parse(txt).error ?? txt }
      catch { renderError.value = txt }
      previewTab.value = 'logs'
      throw new Error(renderError.value)
    }
    const profId = res.headers.get('Profile-Id')
    const contentType = res.headers.get('content-type') ?? ''
    const blob = await res.blob()
    previewBlob.value = blob
    if (contentType.includes('pdf')) previewKind.value = 'pdf'
    else if (contentType.includes('html')) {
      previewKind.value = 'html'
      previewText.value = await blob.text()
    }
    else if (contentType.includes('text/plain')) {
      previewKind.value = 'text'
      previewText.value = await blob.text()
    }
    else previewKind.value = 'binary'
    previewUrl.value = URL.createObjectURL(blob)
    previewTab.value = 'output'

    // Async: pull profile if available
    if (profId) {
      try {
        renderProfile.value = await api.get(`/api/profile/${profId}`)
      } catch {}
    }
  } catch (err: any) {
    if (!renderError.value) renderError.value = err.message ?? 'Error de render'
  } finally {
    rendering.value = false
  }
}

function download() {
  if (!previewUrl.value) return
  const a = document.createElement('a')
  a.href = previewUrl.value
  const ext = previewKind.value === 'pdf' ? 'pdf' :
              previewKind.value === 'html' ? 'html' :
              previewKind.value === 'text' ? 'txt' : 'bin'
  a.download = `${form.value.name || 'report'}.${ext}`
  document.body.appendChild(a)
  a.click()
  a.remove()
}

function copyError() {
  if (!renderError.value) return
  navigator.clipboard.writeText(renderError.value)
  toasts.success('Error copiado')
}

// ─── Public sharing ───────────────────────────────────────────────────────
const cfg = useRuntimeConfig()
const origin = computed(() => cfg.public.apiBase)
const publicUrl = computed(() =>
  form.value.shortid
    ? `${origin.value}/api/report?templateShortid=${form.value.shortid}` // hint for clients
    : '',
)
const outputExt = computed(() => {
  const r = form.value.recipe
  if (r.includes('pdf')) return 'pdf'
  if (r === 'docx') return 'docx'
  if (r === 'pptx') return 'pptx'
  if (r === 'xlsx' || r === 'html-to-xlsx') return 'xlsx'
  if (r === 'text') return 'txt'
  return 'html'
})

async function togglePublic(next: boolean) {
  if (!form.value.shortid) return
  try {
    await api.post(`/api/templates/sharing/${form.value.shortid}/access/${next ? 'public' : 'deny'}`)
    form.value.isPublic = next
    toasts.success(next ? 'Plantilla pública' : 'Plantilla privada')
  } catch (err: any) {
    toasts.error('No se pudo actualizar', extractError(err))
  }
}
function copyPublicUrl() {
  navigator.clipboard.writeText(publicUrl.value)
  toasts.success('URL copiada')
}

function onKey(e: KeyboardEvent) {
  const isMac = navigator.platform.toUpperCase().includes('MAC')
  const ctrl = isMac ? e.metaKey : e.ctrlKey
  if (ctrl && e.key.toLowerCase() === 's') {
    e.preventDefault(); save()
  } else if (ctrl && e.key === 'Enter') {
    e.preventDefault(); render()
  } else if (ctrl && e.key.toLowerCase() === 'p') {
    e.preventDefault(); fullscreenPreview.value = !fullscreenPreview.value
  }
}

onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
})

// ─── Splitter drag ────────────────────────────────────────────────────────
const bodyEl = ref<HTMLElement | null>(null)
const dragging = ref(false)
function onSplitMouseDown(e: MouseEvent) {
  e.preventDefault()
  dragging.value = true
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  window.addEventListener('mousemove', onDragMove)
  window.addEventListener('mouseup',   onDragEnd, { once: true })
}
function onDragMove(e: MouseEvent) {
  if (!bodyEl.value) return
  const rect = bodyEl.value.getBoundingClientRect()
  const pct = ((e.clientX - rect.left) / rect.width) * 100
  split.set(pct)
}
function onDragEnd() {
  dragging.value = false
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  window.removeEventListener('mousemove', onDragMove)
}
onBeforeUnmount(() => window.removeEventListener('mousemove', onDragMove))

// ─── Data tab helpers ─────────────────────────────────────────────────────
const dataIsValid = computed(() => {
  try { JSON.parse(data.value); return true } catch { return false }
})
const sampleData = [
  { label: 'Vacío',   data: '{}' },
  { label: 'Persona', data: '{\n  "name": "Ana",\n  "email": "ana@ejemplo.com"\n}' },
  { label: 'Lista',   data: '{\n  "title": "Productos",\n  "items": [\n    { "name": "Café", "price": 8 }\n  ]\n}' },
]

async function loadFromDataItem(shortid: string) {
  if (!shortid) return
  try {
    const d = await api.get<any>(`/odata/data/${shortid}`)
    const raw = typeof d.dataJson === 'string' ? d.dataJson : JSON.stringify(d.dataJson, null, 2)
    data.value = raw ?? '{}'
    form.value.dataShortid = shortid
    toasts.success(`Data «${d.name}» cargada`)
  } catch (err: any) {
    toasts.error('No se pudo cargar', extractError(err))
  }
}

// ─── Inline "crear data" popover (used from the Data tab) ──────────────────
// Saves the current JSON in the editor as a new `data` entity and links it
// to the template via `dataShortid`. Keeps the user in the editor — no need
// to navigate to /data/new and lose context.
const newDataOpen = ref(false)
const newDataName = ref('')
const newDataSaving = ref(false)
const dataEntity = useEntity('data')

function openNewData() {
  // Suggest a sensible name based on the template name.
  newDataName.value = form.value.name && form.value.name !== 'sin nombre'
    ? `${form.value.name} · data`
    : 'Nueva data'
  newDataOpen.value = true
}

async function createNewData() {
  if (!dataIsValid.value) {
    toasts.error('JSON inválido', 'No se puede crear data con JSON malformado')
    return
  }
  if (!newDataName.value.trim()) {
    toasts.error('Falta el nombre')
    return
  }
  newDataSaving.value = true
  try {
    const created = await dataEntity.create({
      name: newDataName.value.trim(),
      dataJson: JSON.parse(data.value),
    } as any) as any
    // Refresh the dropdown list and select the new entry.
    allDataItems.value = [{ shortid: created.shortid, name: created.name }, ...allDataItems.value]
    form.value.dataShortid = created.shortid
    toasts.success(`Data «${created.name}» creada y vinculada`)
    newDataOpen.value = false
    newDataName.value = ''
  } catch (err: any) {
    toasts.error('No se pudo crear', extractError(err))
  } finally {
    newDataSaving.value = false
  }
}

// ─── Scripts tab ──────────────────────────────────────────────────────────
function attachScript(shortid: string) {
  if (!shortid) return
  if (form.value.scripts.some(s => s.shortid === shortid)) return
  const s = allScripts.value.find(x => x.shortid === shortid)
  if (s) form.value.scripts.push({ shortid: s.shortid, name: s.name })
}
function detachScript(idx: number) {
  form.value.scripts.splice(idx, 1)
}
function moveScript(idx: number, delta: number) {
  const j = idx + delta
  if (j < 0 || j >= form.value.scripts.length) return
  const tmp = form.value.scripts[idx]
  form.value.scripts[idx] = form.value.scripts[j]
  form.value.scripts[j]   = tmp
}

// ─── Page settings ────────────────────────────────────────────────────────
const pageSizes = ['A4', 'A3', 'A5', 'Letter', 'Legal', 'Tabloid']
const pageMargins = [
  { label: 'Sin margen', value: '0' },
  { label: 'Estrecho',   value: '0.5cm' },
  { label: 'Normal',     value: '1cm' },
  { label: 'Ancho',      value: '2cm' },
]

function fmtSaved(d: Date | null) {
  if (!d) return ''
  const diff = (Date.now() - d.getTime()) / 1000
  if (diff < 60)   return 'guardado recién'
  if (diff < 3600) return `guardado hace ${Math.floor(diff / 60)} min`
  return `guardado a las ${d.toLocaleTimeString()}`
}

// Profile event durations. The backend column is `started_at` / `finished_at`
// (snake_case from Postgres). We also tolerate camelCase in case some path
// already normalized it.
function profileEvents() {
  if (!renderProfile.value) return [] as { label: string; ms: number }[]
  const p = renderProfile.value
  const events: { label: string; ms: number }[] = []
  const startISO  = p.started_at  ?? p.startedAt
  const finishISO = p.finished_at ?? p.finishedAt ?? p.finishedOn
  if (startISO && finishISO) {
    const start = new Date(startISO).getTime()
    const end   = new Date(finishISO).getTime()
    if (!Number.isNaN(start) && !Number.isNaN(end)) {
      events.push({ label: t('profile.totalRender'), ms: end - start })
    }
  }
  return events
}
</script>

<template>
  <div class="cr-edit-page" :class="fullscreenPreview ? 'cr-edit-page--fullscreen' : ''">
    <!-- Header -->
    <div class="cr-edit-header">
      <div class="flex items-start gap-3 min-w-0 flex-1">
        <NuxtLink to="/templates" class="cr-row-action !w-9 !h-9 mt-1">
          <UIcon name="i-lucide-arrow-left" class="w-4 h-4" />
        </NuxtLink>
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-1.5 text-[12px] mb-1" style="color: var(--cr-text-soft)">
            <NuxtLink to="/templates" class="hover:underline">Plantillas</NuxtLink>
            <UIcon name="i-lucide-chevron-right" class="w-3 h-3" />
            <span style="color: var(--cr-text-muted)">{{ isNew ? 'nueva' : form.shortid }}</span>
          </div>
          <input
            v-model="form.name"
            type="text"
            placeholder="Nombre de la plantilla"
            class="text-[22px] font-bold tracking-tight bg-transparent border-0 outline-none w-full"
            style="color: var(--cr-text)"
          />
          <div class="flex items-center gap-3 mt-1.5 text-[11.5px] flex-wrap" style="color: var(--cr-text-soft)">
            <Transition
              mode="out-in"
              enter-active-class="transition-opacity duration-150"
              leave-active-class="transition-opacity duration-100"
              enter-from-class="opacity-0" leave-to-class="opacity-0"
            >
              <span v-if="saving" key="saving" class="inline-flex items-center gap-1.5" style="color: var(--cr-text-muted)">
                <span class="w-2 h-2 border rounded-full cr-anim-spin" style="border-color: var(--color-wise-500); border-top-color: transparent; border-width: 1.5px"></span>
                Guardando…
              </span>
              <span v-else-if="isDirty" key="dirty" class="inline-flex items-center gap-1.5" style="color: #b86700">
                <span class="w-1.5 h-1.5 rounded-full" style="background: #ffd11a"></span>
                Cambios sin guardar
              </span>
              <span v-else-if="lastSavedAt" key="saved" class="inline-flex items-center gap-1.5" style="color: var(--color-positive)">
                <UIcon name="i-lucide-check" class="w-3 h-3" />
                {{ fmtSaved(lastSavedAt) }}
              </span>
              <span v-else key="new" class="inline-flex items-center gap-1.5">
                <span class="w-1.5 h-1.5 rounded-full" style="background: var(--cr-text-soft)"></span>
                Sin guardar
              </span>
            </Transition>
            <span class="w-px h-3" style="background: var(--cr-border)"></span>
            <span class="inline-flex items-center gap-1.5"><kbd class="cr-kbd">Ctrl</kbd>+<kbd class="cr-kbd">S</kbd> guardar</span>
            <span class="inline-flex items-center gap-1.5"><kbd class="cr-kbd">Ctrl</kbd>+<kbd class="cr-kbd">Enter</kbd> render</span>
            <span class="inline-flex items-center gap-1.5"><kbd class="cr-kbd">Ctrl</kbd>+<kbd class="cr-kbd">P</kbd> preview full</span>
          </div>
        </div>
      </div>

      <div class="flex items-center gap-2 shrink-0">
        <div class="w-36">
          <CrSelect
            v-model="form.engine"
            :options="engines.map(e => ({ value: e, label: e }))"
            placeholder="engine"
          />
        </div>
        <div class="w-40">
          <CrSelect
            v-model="form.recipe"
            :options="recipes.map(r => ({ value: r, label: r }))"
            placeholder="recipe"
          />
        </div>
        <button
          type="button" class="cr-icon-btn !w-10 !h-10"
          :title="fullscreenPreview ? 'Salir de preview completo (Ctrl+P)' : 'Preview pantalla completa (Ctrl+P)'"
          @click="fullscreenPreview = !fullscreenPreview"
        >
          <UIcon :name="fullscreenPreview ? 'i-lucide-minimize-2' : 'i-lucide-maximize-2'" class="w-4 h-4" />
        </button>
        <button
          type="button" class="cr-icon-btn !w-10 !h-10"
          title="Opciones avanzadas"
          @click="drawerOpen = !drawerOpen"
        >
          <UIcon name="i-lucide-sliders-horizontal" class="w-4 h-4" />
        </button>
        <button class="cr-btn-secondary" :disabled="rendering" title="Render · Ctrl+Enter" @click="render">
          <UIcon v-if="!rendering" name="i-lucide-play" class="w-4 h-4" />
          <span v-else class="w-4 h-4 border-2 rounded-full cr-anim-spin" style="border-color: var(--cr-text); border-top-color: transparent" />
          <span>{{ rendering ? 'Renderizando…' : 'Render' }}</span>
        </button>
        <button class="cr-btn-primary !w-auto" :disabled="saving || (!isDirty && !isNew)" title="Guardar · Ctrl+S" @click="save">
          <UIcon name="i-lucide-save" class="w-4 h-4" />
          <span>{{ isNew ? 'Crear' : 'Guardar' }}</span>
        </button>
      </div>
    </div>

    <!-- Body: editor | splitter | preview (resizable) -->
    <div
      ref="bodyEl"
      class="cr-edit-body"
      :class="dragging ? 'cr-edit-body--dragging' : ''"
      :style="{ '--split': split.ratio.value + '%' }"
    >
      <!-- LEFT pane: editor -->
      <div v-show="!fullscreenPreview" class="cr-edit-pane cr-edit-pane--left">
        <div class="cr-card flex-1 flex flex-col overflow-hidden">
          <div class="flex border-b overflow-x-auto shrink-0" style="border-color: var(--cr-border)">
            <button
              v-for="et in tabs" :key="et.id"
              class="cr-tab"
              :class="tab === et.id ? 'cr-tab--active' : ''"
              @click="tab = et.id"
            >
              <UIcon :name="et.icon" class="w-4 h-4" />
              {{ et.label }}
              <span v-if="et.id === 'scripts' && form.scripts.length > 0" class="cr-tab-badge">{{ form.scripts.length }}</span>
            </button>
          </div>

          <div class="flex-1 p-4 flex flex-col overflow-hidden">
            <CodeEditor
              v-if="tab === 'content'"
              v-model="form.content"
              :language="form.engine === 'handlebars' ? 'handlebars' : 'html'"
              height="100%"
              placeholder="<h1>Hola {{name}}</h1>"
            />
            <CodeEditor v-else-if="tab === 'css'" v-model="form.css" language="css" height="100%" placeholder="body { font-family: Inter, sans-serif; }" />
            <CodeEditor v-else-if="tab === 'helpers'" v-model="form.helpers" language="javascript" height="100%" placeholder="function upper(s) { return String(s).toUpperCase() }" />

            <div v-else-if="tab === 'data'" class="flex-1 flex flex-col overflow-hidden">
              <div class="flex items-center justify-between gap-3 flex-wrap mb-3">
                <p class="text-[12px] inline-flex items-center gap-1.5" :style="{ color: dataIsValid ? 'var(--color-positive)' : '#d03238' }">
                  <UIcon :name="dataIsValid ? 'i-lucide-check' : 'i-lucide-circle-alert'" class="w-3.5 h-3.5" />
                  {{ dataIsValid ? t('data.valid') : t('data.invalid') }}
                </p>
                <div class="flex items-center gap-1.5 flex-wrap">
                  <span class="cr-eyebrow" style="color: var(--cr-text-soft)">{{ t('data.loadFrom') }}</span>
                  <div class="w-56">
                    <CrSelect
                      :model-value="form.dataShortid ?? ''"
                      :options="[
                        { value: '', label: t('data.noLink') },
                        ...allDataItems.map(d => ({ value: d.shortid, label: d.name })),
                      ]"
                      :placeholder="t('data.noLink')"
                      :searchable="allDataItems.length > 6"
                      @update:model-value="(v) => { form.dataShortid = v; loadFromDataItem(v) }"
                    />
                  </div>
                  <span class="text-[11px]" style="color: var(--cr-text-soft)">{{ t('data.orSample') }}</span>
                  <button v-for="s in sampleData" :key="s.label" class="cr-chip" @click="data = s.data">
                    {{ s.label }}
                  </button>
                  <span class="w-px h-4" style="background: var(--cr-border)"></span>
                  <button
                    type="button"
                    class="cr-chip cr-chip--primary"
                    :title="t('data.newButton')"
                    @click="openNewData"
                  >
                    <UIcon name="i-lucide-plus" class="w-3.5 h-3.5" />
                    {{ t('data.newButton') }}
                  </button>
                </div>
              </div>

              <!-- Inline popover: name → create + link -->
              <Transition
                enter-active-class="transition-all duration-180 ease-[cubic-bezier(0.23,1,0.32,1)]"
                leave-active-class="transition-all duration-120"
                enter-from-class="opacity-0 -translate-y-1"
                leave-to-class="opacity-0 -translate-y-1"
              >
                <div v-if="newDataOpen" class="cr-newdata-pop mb-3">
                  <div class="flex items-center gap-2 mb-2">
                    <UIcon name="i-lucide-database" class="w-4 h-4" style="color: var(--color-wise-600)" />
                    <p class="text-[12.5px] font-semibold" style="color: var(--cr-text)">Crear data desde el JSON actual</p>
                  </div>
                  <div class="flex items-center gap-2 flex-wrap">
                    <input
                      v-model="newDataName"
                      type="text"
                      placeholder="Nombre (ej. reporte-seguridad-data)"
                      class="cr-input !pl-3 !py-2 !text-[13px] flex-1 min-w-[220px]"
                      @keydown.enter.prevent="createNewData"
                      @keydown.esc="newDataOpen = false"
                    />
                    <button
                      class="cr-btn-primary !w-auto !py-2"
                      :disabled="newDataSaving || !dataIsValid || !newDataName.trim()"
                      @click="createNewData"
                    >
                      <UIcon
                        v-if="!newDataSaving"
                        name="i-lucide-check"
                        class="w-4 h-4"
                      />
                      <span v-else class="w-4 h-4 border-2 rounded-full cr-anim-spin" style="border-color: currentColor; border-top-color: transparent"></span>
                      <span>{{ newDataSaving ? 'Creando…' : 'Crear y vincular' }}</span>
                    </button>
                    <button class="cr-row-action !w-9 !h-9" title="Cancelar (Esc)" @click="newDataOpen = false">
                      <UIcon name="i-lucide-x" class="w-4 h-4" />
                    </button>
                  </div>
                  <p class="text-[10.5px] mt-2 inline-flex items-center gap-1.5" style="color: var(--cr-text-soft)">
                    <UIcon name="i-lucide-info" class="w-3 h-3" />
                    Se guarda como entidad <code>data</code> y queda vinculada a esta plantilla en <em>dataShortid</em>.
                  </p>
                </div>
              </Transition>

              <CodeEditor v-model="data" language="json" height="100%" />
            </div>

            <div v-else-if="tab === 'scripts'" class="flex-1 overflow-y-auto">
              <p class="text-[13px] mb-3" style="color: var(--cr-text-muted)">
                Scripts que se ejecutan antes/después del render. El orden importa: corren de arriba a abajo.
              </p>
              <div v-if="form.scripts.length === 0" class="cr-card p-6 text-center" style="border-style: dashed">
                <UIcon name="i-lucide-square-function" class="w-10 h-10 mx-auto mb-2" style="color: var(--cr-text-soft)" />
                <p class="text-[13px]" style="color: var(--cr-text-muted)">Sin scripts adjuntos todavía.</p>
              </div>
              <ul v-else class="space-y-2 mb-4">
                <li v-for="(s, i) in form.scripts" :key="i" class="cr-attached">
                  <span class="cr-attached-handle">{{ i + 1 }}</span>
                  <div class="flex-1 min-w-0">
                    <div class="text-[13px] font-semibold truncate" style="color: var(--cr-text)">
                      {{ allScripts.find(x => x.shortid === s.shortid)?.name ?? s.shortid }}
                    </div>
                    <div class="text-[11px] font-mono" style="color: var(--cr-text-soft)">{{ s.shortid }}</div>
                  </div>
                  <button class="cr-row-action" :disabled="i === 0" title="Subir" @click="moveScript(i, -1)">
                    <UIcon name="i-lucide-arrow-up" class="w-4 h-4" />
                  </button>
                  <button class="cr-row-action" :disabled="i === form.scripts.length - 1" title="Bajar" @click="moveScript(i, 1)">
                    <UIcon name="i-lucide-arrow-down" class="w-4 h-4" />
                  </button>
                  <button class="cr-row-action cr-row-action--danger" title="Quitar" @click="detachScript(i)">
                    <UIcon name="i-lucide-x" class="w-4 h-4" />
                  </button>
                </li>
              </ul>
              <div class="flex items-center gap-2">
                <select class="cr-input !w-auto !py-2 !text-[13px] !pl-3" style="padding-top:8px; padding-bottom:8px" @change="(e) => { attachScript(((e.target) as HTMLSelectElement).value); ((e.target) as HTMLSelectElement).value = '' }">
                  <option value="">+ Adjuntar script…</option>
                  <option v-for="s in allScripts.filter(x => !form.scripts.some(a => a.shortid === x.shortid))" :key="s.shortid" :value="s.shortid">
                    {{ s.name }} · {{ s.scope }}
                  </option>
                </select>
                <NuxtLink to="/scripts/new" class="cr-btn-secondary !text-[12px] !py-1.5 !px-3">
                  <UIcon name="i-lucide-plus" class="w-3.5 h-3.5" />
                  Nuevo script
                </NuxtLink>
              </div>
            </div>

            <!-- Recipe-specific options form -->
            <RecipeOptions
              v-else-if="tab === 'recipe'"
              :recipe="form.recipe"
              v-model="recipeBag"
              :assets="allAssets"
              :templates="allTemplates"
              :self-shortid="form.shortid"
            />

            <!-- PDF post-processing operations -->
            <PdfOperations
              v-else-if="tab === 'pdfops'"
              v-model="form.pdfOperations"
            />

            <div v-else-if="tab === 'page'" class="space-y-5 max-w-md overflow-y-auto">
              <p class="text-[13px]" style="color: var(--cr-text-muted)">
                Configuración aplicada cuando el recipe es PDF. Se inyecta como reglas
                <code class="px-1 py-0.5 rounded text-[11px]" style="background: var(--cr-surface-soft)">@page</code>.
              </p>
              <div>
                <label class="cr-label">Tamaño</label>
                <div class="flex flex-wrap gap-1.5">
                  <button v-for="size in pageSizes" :key="size" class="cr-chip" :class="form.pageSize === size ? 'cr-chip--active' : ''" @click="form.pageSize = size">{{ size }}</button>
                </div>
              </div>
              <div>
                <label class="cr-label">Orientación</label>
                <div class="flex gap-2">
                  <button class="cr-chip-large" :class="form.pageOrientation === 'portrait' ? 'cr-chip-large--active' : ''" @click="form.pageOrientation = 'portrait'">
                    <UIcon name="i-lucide-rectangle-vertical" class="w-4 h-4" />
                    Vertical
                  </button>
                  <button class="cr-chip-large" :class="form.pageOrientation === 'landscape' ? 'cr-chip-large--active' : ''" @click="form.pageOrientation = 'landscape'">
                    <UIcon name="i-lucide-rectangle-horizontal" class="w-4 h-4" />
                    Apaisado
                  </button>
                </div>
              </div>
              <div>
                <label class="cr-label">Márgenes</label>
                <div class="flex flex-wrap gap-1.5 mb-2">
                  <button v-for="m in pageMargins" :key="m.value" class="cr-chip" :class="form.pageMargin === m.value ? 'cr-chip--active' : ''" @click="form.pageMargin = m.value">{{ m.label }}</button>
                </div>
                <input v-model="form.pageMargin" type="text" placeholder="1cm  ó  10px 20px 10px 20px" class="cr-input !pl-4 font-mono text-[13px]" />
              </div>

              <!-- Report retention -->
              <div class="pt-4 border-t" style="border-color: var(--cr-border)">
                <label class="cr-label">{{ t('tpl.retention') }}</label>
                <div class="flex flex-wrap gap-1.5 mb-2">
                  <button
                    v-for="d in [7, 30, 90, 365, 0]" :key="d"
                    class="cr-chip"
                    :class="Number(form.reportRetentionDays) === d ? 'cr-chip--active' : ''"
                    @click="form.reportRetentionDays = d"
                  >
                    {{ d === 0 ? t('tpl.retentionForever') : t('tpl.retentionDays').replace('{n}', String(d)) }}
                  </button>
                </div>
                <div class="flex items-center gap-2">
                  <input
                    v-model.number="form.reportRetentionDays"
                    type="number" min="0" max="3650"
                    class="cr-input !pl-4 font-mono text-[13px] !w-32"
                  />
                  <span class="text-[12px]" style="color: var(--cr-text-soft)">{{ t('tpl.retentionUnit') }}</span>
                </div>
                <p class="text-[11.5px] mt-2 inline-flex items-start gap-1.5" style="color: var(--cr-text-soft)">
                  <UIcon name="i-lucide-info" class="w-3.5 h-3.5 shrink-0 mt-px" />
                  {{ Number(form.reportRetentionDays) === 0
                      ? t('tpl.retentionHintForever')
                      : t('tpl.retentionHint').replace('{n}', String(form.reportRetentionDays)) }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Splitter (desktop only — hidden when fullscreen preview) -->
      <div
        v-show="!fullscreenPreview"
        class="cr-splitter"
        role="separator"
        aria-orientation="vertical"
        @mousedown="onSplitMouseDown"
        @dblclick="split.reset()"
        :title="`Arrastrá para ajustar (${split.ratio.value}%). Doble click para resetear.`"
      >
        <span class="cr-splitter-grip" aria-hidden="true">
          <span></span><span></span><span></span>
        </span>
      </div>

      <!-- RIGHT pane: preview -->
      <div class="cr-edit-pane cr-edit-pane--right">
        <div class="cr-card flex-1 flex flex-col overflow-hidden">
          <div class="flex items-center justify-between gap-3 border-b shrink-0 px-2" style="border-color: var(--cr-border)">
            <div class="flex">
              <button
                v-for="pt in [
                  { id: 'output',  label: t('preview.output'),  icon: 'i-lucide-monitor' },
                  { id: 'source',  label: t('preview.source'),  icon: 'i-lucide-code' },
                  { id: 'logs',    label: t('preview.logs'),    icon: 'i-lucide-scroll-text' },
                  { id: 'profile', label: t('preview.profile'), icon: 'i-lucide-activity' },
                ]"
                :key="pt.id"
                class="cr-tab cr-tab--compact"
                :class="previewTab === pt.id ? 'cr-tab--active' : ''"
                @click="previewTab = (pt.id as PreviewTab)"
              >
                <UIcon :name="pt.icon" class="w-3.5 h-3.5" />
                {{ pt.label }}
                <span v-if="pt.id === 'logs' && renderError" class="cr-tab-dot" style="background: #d03238"></span>
              </button>
            </div>
            <button v-if="previewUrl" class="cr-btn-secondary !text-[12px] !py-1.5 !px-3" @click="download">
              <UIcon name="i-lucide-download" class="w-3.5 h-3.5" />
              Descargar
            </button>
          </div>

          <div class="flex-1 relative overflow-hidden" style="background: var(--cr-surface-soft)">
            <!-- OUTPUT TAB -->
            <template v-if="previewTab === 'output'">
              <div v-if="!previewUrl && !renderError" class="absolute inset-0 flex flex-col items-center justify-center gap-3 p-6 text-center" style="color: var(--cr-text-muted)">
                <svg width="120" height="100" viewBox="0 0 120 100" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
                  <rect x="20" y="14" width="58" height="74" rx="4" fill="white" stroke="#0e0f0c" stroke-opacity="0.10" stroke-width="1.5"/>
                  <rect x="28" y="24" width="32" height="3" rx="1" fill="#0e0f0c" fill-opacity="0.75"/>
                  <rect x="28" y="32" width="22" height="2" rx="1" fill="#0e0f0c" fill-opacity="0.30"/>
                  <rect x="28" y="42" width="42" height="1.5" rx="0.75" fill="#0e0f0c" fill-opacity="0.18"/>
                  <rect x="28" y="47" width="36" height="1.5" rx="0.75" fill="#0e0f0c" fill-opacity="0.18"/>
                  <rect x="28" y="62" width="16" height="6" rx="2" fill="#9fe870"/>
                  <circle cx="92" cy="34" r="22" fill="#9fe870" fill-opacity="0.20"/>
                  <circle cx="92" cy="34" r="14" fill="#9fe870" fill-opacity="0.40"/>
                  <path d="M86 34l4 4 8-8" stroke="#0e0f0c" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
                </svg>
                <p class="text-[14px]">Click <span class="font-semibold" style="color: var(--cr-text)">Render</span> o <kbd class="cr-kbd">Ctrl</kbd>+<kbd class="cr-kbd">Enter</kbd>.</p>
              </div>
              <div v-if="renderError" class="absolute inset-0 flex flex-col items-center justify-center gap-3 p-6 text-center">
                <UIcon name="i-lucide-triangle-alert" class="w-10 h-10" style="color: #d03238" />
                <p class="text-[14px] font-semibold" style="color: #a72027">Error de render — ver tab "Logs"</p>
              </div>
              <iframe
                v-if="previewUrl && (previewKind === 'pdf' || previewKind === 'html')"
                :src="previewUrl"
                class="absolute inset-0 w-full h-full border-0"
                style="background: white"
              />
              <div v-else-if="previewUrl && previewKind === 'binary'" class="absolute inset-0 flex flex-col items-center justify-center gap-3">
                <UIcon name="i-lucide-file" class="w-10 h-10" style="color: var(--cr-text-soft)" />
                <p class="text-[13px]" style="color: var(--cr-text-muted)">Archivo binario.</p>
                <button class="cr-btn-primary !w-auto" @click="download">
                  <UIcon name="i-lucide-download" class="w-4 h-4" />
                  Descargar
                </button>
              </div>
            </template>

            <!-- SOURCE TAB (HTML/text view) -->
            <div v-if="previewTab === 'source'" class="absolute inset-0 overflow-auto">
              <CodeEditor
                v-if="previewText"
                :model-value="previewText"
                :language="previewKind === 'html' ? 'html' : 'text'"
                height="100%"
                readonly
              />
              <!-- Recipe produced binary content (PDF, xlsx, …) — no readable
                   source. Tell the user instead of leaving them confused. -->
              <div v-else-if="previewUrl && (previewKind === 'pdf' || previewKind === 'binary')"
                class="absolute inset-0 flex flex-col items-center justify-center gap-2 text-center p-6">
                <UIcon name="i-lucide-file-x" class="w-10 h-10" style="color: var(--cr-text-soft)" />
                <p class="text-[13px] font-semibold" style="color: var(--cr-text)">{{ t('source.binaryTitle') }}</p>
                <p class="text-[12px] max-w-md" style="color: var(--cr-text-muted)">
                  {{ t('source.binaryHint') }}
                </p>
              </div>
              <div v-else class="absolute inset-0 flex items-center justify-center text-[13px]" style="color: var(--cr-text-muted)">
                {{ t('source.empty') }}
              </div>
            </div>

            <!-- LOGS TAB -->
            <div v-if="previewTab === 'logs'" class="absolute inset-0 overflow-auto p-4">
              <div v-if="renderError" class="cr-card p-4" style="background: rgb(208 50 56 / 0.06); border-color: rgb(208 50 56 / 0.25)">
                <div class="flex items-center gap-2 mb-2">
                  <UIcon name="i-lucide-triangle-alert" class="w-4 h-4" style="color: #d03238" />
                  <span class="font-semibold text-[13px]" style="color: #a72027">Render error</span>
                  <button class="cr-row-action ml-auto" title="Copiar" @click="copyError">
                    <UIcon name="i-lucide-copy" class="w-3.5 h-3.5" />
                  </button>
                </div>
                <pre class="text-[12px] font-mono whitespace-pre-wrap" style="color: var(--cr-text)">{{ renderError }}</pre>
              </div>
              <div v-else-if="renderLogs">
                <pre class="text-[12px] font-mono whitespace-pre-wrap" style="color: var(--cr-text-muted)">{{ renderLogs }}</pre>
              </div>
              <div v-else class="text-center py-10 text-[13px]" style="color: var(--cr-text-soft)">
                Sin logs por ahora.
              </div>
            </div>

            <!-- PROFILE TAB -->
            <div v-if="previewTab === 'profile'" class="absolute inset-0 overflow-auto p-4">
              <div v-if="renderProfile">
                <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-4">
                  <div class="cr-card p-3">
                    <p class="cr-eyebrow" style="color: var(--cr-text-soft)">{{ t('profile.state') }}</p>
                    <StatusBadge :state="renderProfile.state" class="mt-1.5" />
                  </div>
                  <div class="cr-card p-3">
                    <p class="cr-eyebrow" style="color: var(--cr-text-soft)">{{ t('profile.mode') }}</p>
                    <p class="font-semibold text-[14px] mt-1" style="color: var(--cr-text)">{{ renderProfile.mode }}</p>
                  </div>
                  <div class="cr-card p-3">
                    <p class="cr-eyebrow" style="color: var(--cr-text-soft)">{{ t('profile.id') }}</p>
                    <p class="font-mono text-[11px] truncate mt-1" style="color: var(--cr-text)">{{ renderProfile.shortid }}</p>
                  </div>
                  <div class="cr-card p-3">
                    <p class="cr-eyebrow" style="color: var(--cr-text-soft)">{{ t('profile.startedAt') }}</p>
                    <p class="text-[12px] tabular-nums mt-1" style="color: var(--cr-text)">
                      {{ renderProfile.started_at || renderProfile.startedAt
                          ? new Date(renderProfile.started_at || renderProfile.startedAt).toLocaleTimeString()
                          : '—' }}
                    </p>
                  </div>
                </div>
                <div v-if="profileEvents().length">
                  <p class="cr-eyebrow mb-2" style="color: var(--cr-text-soft)">{{ t('profile.times') }}</p>
                  <ul class="space-y-1.5">
                    <li v-for="(ev, idx) in profileEvents()" :key="idx" class="flex items-center gap-3 text-[13px]">
                      <span class="flex-1" style="color: var(--cr-text)">{{ ev.label }}</span>
                      <span class="font-mono tabular-nums" style="color: var(--cr-text-muted)">{{ ev.ms }} ms</span>
                      <span class="h-2 rounded-full" style="background: var(--color-wise-400); width: 80px"></span>
                    </li>
                  </ul>
                </div>
              </div>
              <div v-else class="text-center py-10 text-[13px]" style="color: var(--cr-text-soft)">
                {{ t('profile.empty') }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Drawer (avanzado) -->
      <Transition
        enter-active-class="transition-all duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]"
        leave-active-class="transition-all duration-150"
        enter-from-class="opacity-0 translate-x-2"
        leave-to-class="opacity-0 translate-x-2"
      >
        <div v-if="drawerOpen" class="cr-edit-drawer">
          <div class="cr-card flex-1 flex flex-col overflow-hidden">
            <div class="px-4 py-3 border-b flex items-center justify-between shrink-0" style="border-color: var(--cr-border)">
              <div class="flex items-center gap-2 cr-eyebrow" style="color: var(--cr-text-muted)">
                <UIcon name="i-lucide-sliders-horizontal" class="w-3.5 h-3.5" />
                Avanzado
              </div>
              <button class="cr-row-action" title="Cerrar" @click="drawerOpen = false">
                <UIcon name="i-lucide-x" class="w-4 h-4" />
              </button>
            </div>
            <div class="p-4 space-y-4 overflow-y-auto text-[13px]">
              <div>
                <p class="cr-label">Shortid</p>
                <code class="text-[12px] font-mono">{{ form.shortid || 'sin guardar' }}</code>
              </div>
              <div>
                <p class="cr-label">Engine · Recipe</p>
                <p>{{ form.engine }} → <RecipePill :recipe="form.recipe" /></p>
              </div>
              <div>
                <p class="cr-label">Página</p>
                <p>{{ form.pageSize }} · {{ form.pageOrientation }} · márgenes {{ form.pageMargin }}</p>
              </div>
              <div>
                <p class="cr-label">Scripts adjuntos</p>
                <p>{{ form.scripts.length }}</p>
              </div>
              <div>
                <p class="cr-label">Data linkeada</p>
                <p>{{ form.dataShortid || '—' }}</p>
              </div>

              <!-- Public sharing -->
              <div class="pt-3 border-t" style="border-color: var(--cr-border)">
                <p class="cr-label">Compartir públicamente</p>
                <label class="cr-switch">
                  <input
                    type="checkbox"
                    :checked="form.isPublic"
                    :disabled="!form.shortid"
                    @change="(e) => togglePublic((e.target as HTMLInputElement).checked)"
                  />
                  <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
                  <span class="cr-switch-label">Permitir render sin auth</span>
                </label>
                <Transition
                  enter-active-class="transition-all duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]"
                  leave-active-class="transition-all duration-150"
                  enter-from-class="opacity-0 -translate-y-1"
                  leave-to-class="opacity-0"
                >
                  <div v-if="form.isPublic && form.shortid" class="mt-2 flex items-center gap-1">
                    <code class="text-[10.5px] font-mono px-2 py-1.5 rounded border flex-1 truncate" style="background: var(--cr-surface-soft); border-color: var(--cr-border); color: var(--cr-text)">{{ publicUrl }}</code>
                    <button class="cr-row-action" title="Copiar URL" @click="copyPublicUrl">
                      <UIcon name="i-lucide-copy" class="w-3.5 h-3.5" />
                    </button>
                  </div>
                </Transition>
              </div>

              <!-- API quick-reference -->
              <div class="pt-3 border-t" style="border-color: var(--cr-border)">
                <p class="cr-label">curl ejemplo</p>
                <pre class="text-[10.5px] font-mono p-2 rounded overflow-x-auto" style="background: var(--cr-surface-soft); color: var(--cr-text-muted)">curl -X POST {{ origin }}/api/report \
  -H "x-api-key: cr_…" \
  -d '{"template":{"shortid":"{{ form.shortid }}"},"data":{}}' \
  --output report.{{ outputExt }}</pre>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </div>
  </div>
</template>

<style>
/* ─── Editor page layout ────────────────────────────────────────────────── */
.cr-edit-page {
  display: flex;
  flex-direction: column;
  /* Fill from main content area down to viewport bottom.
     Topbar = 64px, page padding 24px*2 = 48px, ~80px header → ~196px. */
  height: calc(100dvh - 64px - 48px);
  min-height: 480px;
  margin-bottom: -24px; /* offset the parent main's py-6 bottom padding */
}
.cr-edit-page--fullscreen { /* hide everything to the left; preview takes all */ }

.cr-edit-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
  flex-shrink: 0;
}

.cr-edit-body {
  flex: 1;
  display: flex;
  gap: 0;
  min-height: 0;
  position: relative;
}
.cr-edit-body--dragging * {
  pointer-events: none;
  user-select: none;
}

.cr-edit-pane {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}
.cr-edit-pane--left {
  width: var(--split, 50%);
  min-width: 280px;
}
.cr-edit-pane--right {
  flex: 1;
  min-width: 280px;
}
.cr-edit-page--fullscreen .cr-edit-pane--right {
  width: 100%;
}

/* ─── Splitter ──────────────────────────────────────────────────────────── */
.cr-splitter {
  width: 10px;
  flex-shrink: 0;
  cursor: col-resize;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  transition: background-color 140ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-splitter::before {
  content: "";
  position: absolute;
  top: 0; bottom: 0;
  left: 50%;
  width: 1px;
  background: var(--cr-border);
  transform: translateX(-50%);
  transition: background-color 140ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-splitter:hover::before {
  background: var(--color-wise-500);
  width: 2px;
}
.cr-splitter-grip {
  position: relative;
  z-index: 1;
  width: 18px;
  height: 36px;
  border-radius: 9999px;
  background: var(--cr-surface);
  border: 1px solid var(--cr-border);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  opacity: 0;
  transition:
    opacity 140ms cubic-bezier(0.23, 1, 0.32, 1),
    border-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 200ms cubic-bezier(0.34, 1.56, 0.64, 1);
}
.cr-splitter-grip span {
  width: 2px;
  height: 2px;
  border-radius: 1px;
  background: var(--cr-text-muted);
}
.cr-splitter:hover .cr-splitter-grip,
.cr-edit-body--dragging .cr-splitter-grip {
  opacity: 1;
  border-color: var(--color-wise-500);
  transform: scale(1.1);
}

/* ─── Drawer (advanced) ─────────────────────────────────────────────────── */
.cr-edit-drawer {
  width: 300px;
  flex-shrink: 0;
  margin-left: 12px;
  display: flex;
  flex-direction: column;
}

/* ─── Tabs ──────────────────────────────────────────────────────────────── */
.cr-tab {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 18px;
  font-size: 13px;
  font-weight: 600;
  color: var(--cr-text-muted);
  border-bottom: 2px solid transparent;
  transition: color 140ms cubic-bezier(0.23, 1, 0.32, 1),
              border-color 140ms cubic-bezier(0.23, 1, 0.32, 1);
  white-space: nowrap;
  position: relative;
}
.cr-tab--compact { padding: 10px 12px; font-size: 12px; }
.cr-tab:hover { color: var(--cr-text); }
.cr-tab--active {
  color: var(--cr-text);
  border-bottom-color: var(--color-wise-500);
}
.cr-tab:active { transform: scale(0.98); }
.cr-tab-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 16px;
  height: 16px;
  padding: 0 5px;
  margin-left: 2px;
  border-radius: 9999px;
  background: var(--color-wise-400);
  color: #0e0f0c;
  font-size: 10px;
  font-weight: 700;
}
.cr-tab-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 9999px;
  margin-left: 4px;
}

/* ─── Inline buttons & chips (shared) ───────────────────────────────────── */
.cr-btn-secondary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 16px;
  border-radius: 12px;
  font-size: 13.5px;
  font-weight: 600;
  color: var(--cr-text);
  background: var(--cr-surface-soft);
  border: 1px solid var(--cr-border);
  cursor: pointer;
  transition: background-color 140ms cubic-bezier(0.23,1,0.32,1),
              transform 100ms cubic-bezier(0.23,1,0.32,1);
}
.cr-btn-secondary:hover { background: var(--cr-border); }
.cr-btn-secondary:active { transform: scale(0.97); }
.cr-btn-secondary:disabled { opacity: 0.5; cursor: not-allowed; }

.cr-chip {
  padding: 7px 12px;
  border-radius: 9999px;
  border: 1px solid var(--cr-border);
  background: var(--cr-surface);
  color: var(--cr-text-muted);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 140ms cubic-bezier(0.23,1,0.32,1),
              border-color 140ms cubic-bezier(0.23,1,0.32,1),
              color 140ms cubic-bezier(0.23,1,0.32,1),
              transform 100ms cubic-bezier(0.23,1,0.32,1);
}
.cr-chip:hover { border-color: var(--cr-border-strong); color: var(--cr-text); }
.cr-chip:active { transform: scale(0.97); }
.cr-chip--active {
  background: var(--color-wise-400);
  border-color: var(--color-wise-500);
  color: #0e0f0c;
}
/* Primary-tinted chip used for "Nueva data" — distinguishable from neutral
   sample chips without being as loud as cr-btn-primary. */
.cr-chip--primary {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  background: var(--color-wise-100);
  border-color: var(--color-wise-500);
  color: var(--color-wise-800);
}
.cr-chip--primary:hover {
  background: var(--color-wise-200);
  color: var(--color-wise-900);
}
html.dark .cr-chip--primary {
  background: rgb(159 232 112 / 0.14);
  color: var(--color-wise-300);
}
html.dark .cr-chip--primary:hover {
  background: rgb(159 232 112 / 0.22);
  color: var(--color-wise-200);
}

/* Inline "new data" popover */
.cr-newdata-pop {
  background: var(--cr-surface-soft);
  border: 1px solid var(--cr-border);
  border-radius: 12px;
  padding: 12px 14px;
}
.cr-chip-large {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 16px;
  border-radius: 12px;
  border: 1.5px solid var(--cr-border);
  background: var(--cr-surface);
  color: var(--cr-text-muted);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 140ms cubic-bezier(0.23,1,0.32,1),
              border-color 140ms cubic-bezier(0.23,1,0.32,1),
              color 140ms cubic-bezier(0.23,1,0.32,1),
              transform 100ms cubic-bezier(0.23,1,0.32,1);
}
.cr-chip-large:hover { border-color: var(--cr-border-strong); color: var(--cr-text); }
.cr-chip-large:active { transform: scale(0.98); }
.cr-chip-large--active {
  border-color: var(--color-wise-500);
  background: rgb(159 232 112 / 0.10);
  color: var(--cr-text);
}

.cr-row-action {
  width: 32px; height: 32px;
  border-radius: 8px;
  display: inline-flex; align-items: center; justify-content: center;
  color: var(--cr-text-muted);
  transition: background-color 140ms cubic-bezier(0.23,1,0.32,1),
              color 140ms cubic-bezier(0.23,1,0.32,1),
              transform 100ms cubic-bezier(0.23,1,0.32,1);
}
.cr-row-action:hover { background: var(--cr-surface-soft); color: var(--cr-text); }
.cr-row-action:active { transform: scale(0.94); }
.cr-row-action--danger:hover { background: rgb(208 50 56 / 0.10); color: #a72027; }
.cr-row-action:disabled { opacity: 0.4; cursor: not-allowed; }

.cr-attached {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background: var(--cr-surface);
  border: 1px solid var(--cr-border);
  border-radius: 10px;
}
.cr-attached-handle {
  width: 26px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: var(--cr-surface-soft);
  color: var(--cr-text-soft);
  font-size: 11px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}

.cr-kbd {
  display: inline-flex; align-items: center;
  padding: 1px 6px;
  font-family: ui-monospace, "JetBrains Mono", monospace;
  font-size: 10.5px;
  font-weight: 600;
  border-radius: 4px;
  border: 1px solid var(--cr-border-strong);
  background: var(--cr-surface);
  color: var(--cr-text);
  vertical-align: middle;
}
</style>
