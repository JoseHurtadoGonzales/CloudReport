<script setup lang="ts">
// Recipe-specific options. Mirrors jsreport's per-recipe property panels with
// collapsible sections grouped by intent. Each section shows a badge with the
// count of active settings so users can see at a glance what they have set.

interface Props {
  recipe: string
  /** Per-recipe options bag (templates.<recipe> JSONB column on the backend) */
  modelValue: Record<string, any>
  /** Asset list — for templateAsset pickers (docx/xlsx/pptx) */
  assets?: { shortid: string; name: string }[]
  /** Other templates — for header/footer template selectors, appendTemplate, etc. */
  templates?: { shortid: string; name: string; recipe: string }[]
  /** Current template's own shortid (to exclude itself from selectors) */
  selfShortid?: string
}
const props = withDefaults(defineProps<Props>(), {
  assets: () => [],
  templates: () => [],
  selfShortid: '',
})

const emit = defineEmits<{ 'update:modelValue': [v: Record<string, any>] }>()

function patch(p: Record<string, any>) {
  emit('update:modelValue', { ...(props.modelValue || {}), ...p })
}
function setNested(parentKey: string, key: string, value: any) {
  const parent = (props.modelValue || {})[parentKey] ?? {}
  emit('update:modelValue', { ...(props.modelValue || {}), [parentKey]: { ...parent, [key]: value } })
}
function get(path: string, fallback: any = '') {
  const segs = path.split('.')
  let cur: any = props.modelValue
  for (const s of segs) {
    if (cur == null) return fallback
    cur = cur[s]
  }
  return cur ?? fallback
}

// Count of active (non-default) options for a section — drives the chip badge.
function activeCount(...paths: string[]) {
  let n = 0
  for (const p of paths) {
    const v = get(p, undefined)
    if (v == null || v === '' || v === false) continue
    if (Array.isArray(v) && v.length === 0) continue
    if (typeof v === 'object' && Object.keys(v).length === 0) continue
    n++
  }
  return n || ''
}

const recipeInfo = computed(() => {
  switch (props.recipe) {
    case 'chrome-pdf': return { icon: 'i-lucide-chrome',       title: 'Chrome PDF',  desc: 'Render HTML→PDF con Chromium headless. Soporta JS, fonts web, gráficos Chart.js / ECharts.' }
    case 'weasyprint': return { icon: 'i-lucide-printer',      title: 'WeasyPrint',  desc: 'Render HTML→PDF basado en CSS Paged Media. No ejecuta JS, ideal para tipografía fina.' }
    case 'docx':       return { icon: 'i-lucide-file-text',    title: 'DOCX',        desc: 'Reemplaza tags Jinja2 (docxtpl) en un .docx subido como asset.' }
    case 'xlsx':       return { icon: 'i-lucide-sheet',        title: 'XLSX',        desc: 'Genera Excel desde JSON o reemplaza tags en una plantilla .xlsx.' }
    case 'pptx':       return { icon: 'i-lucide-presentation',title: 'PPTX',        desc: 'Reemplaza placeholders en un .pptx con tu data.' }
    case 'html-to-xlsx':return{icon: 'i-lucide-table-2',       title: 'HTML → XLSX', desc: 'Extrae las <table> del HTML rendereado y las exporta como hojas.' }
    case 'static-pdf': return { icon: 'i-lucide-file-text',    title: 'Static PDF',  desc: 'Devuelve un PDF pre-existente sin renderizar (anexar, watermarkear, etc.).' }
    case 'html':       return { icon: 'i-lucide-code-xml',     title: 'HTML',        desc: 'Devuelve el HTML rendereado directamente.' }
    case 'text':       return { icon: 'i-lucide-type',         title: 'Texto plano', desc: 'Output del engine como texto.' }
    default:           return { icon: 'i-lucide-box',          title: props.recipe,  desc: 'Recipe sin descripción.' }
  }
})
</script>

<template>
  <div class="space-y-3 overflow-y-auto h-full pb-4">
    <!-- Recipe header card -->
    <div class="cr-recipe-info">
      <span class="cr-recipe-info-icon">
        <UIcon :name="recipeInfo.icon" class="w-5 h-5" />
      </span>
      <div class="min-w-0 flex-1">
        <p class="font-bold text-[14px]" style="color: var(--cr-text)">{{ recipeInfo.title }}</p>
        <p class="text-[12px] mt-0.5 leading-snug" style="color: var(--cr-text-muted)">{{ recipeInfo.desc }}</p>
      </div>
    </div>

    <!-- Quick anchors for PDF recipes (super-discoverable shortcuts) -->
    <div
      v-if="recipe === 'chrome-pdf' || recipe === 'weasyprint'"
      class="grid grid-cols-2 sm:grid-cols-4 gap-2"
    >
      <a href="#cr-section-page" class="cr-quick-anchor">
        <UIcon name="i-lucide-layout-template" class="w-4 h-4" />
        <span>Página</span>
      </a>
      <a href="#cr-section-hf" class="cr-quick-anchor">
        <UIcon name="i-lucide-panel-top" class="w-4 h-4" />
        <span>Header / Footer</span>
      </a>
      <a href="#cr-section-meta" class="cr-quick-anchor">
        <UIcon name="i-lucide-info" class="w-4 h-4" />
        <span>Metadata</span>
      </a>
      <a href="#cr-section-render" class="cr-quick-anchor">
        <UIcon name="i-lucide-settings" class="w-4 h-4" />
        <span>Render</span>
      </a>
    </div>

    <!-- ═══════════════════════ chrome-pdf ═══════════════════════ -->
    <template v-if="recipe === 'chrome-pdf'">
      <!-- Formato, orientación y márgenes viven en la pestaña Página (única fuente
           de verdad). Acá sólo dejamos lo que es específico de Chrome PDF:
           rango de páginas y, más abajo, Header/Footer y otros toggles. -->
      <Collapsible
        anchor-id="cr-section-page"
        title="Rango de páginas"
        description="Imprimir sólo un subconjunto del documento renderizado"
        icon="i-lucide-list-ordered"
        :badge="get('pageRanges') ? 'set' : ''"
      >
        <div>
          <label class="cr-label">Rango (formato: <code>1-5, 8, 11-13</code>)</label>
          <input type="text" :value="get('pageRanges', '')" placeholder="1-5, 8, 11-13"
            class="cr-input !pl-4 font-mono text-[13px]"
            @input="(e) => patch({ pageRanges: (e.target as HTMLInputElement).value })" />
          <p class="text-[11px] mt-2 inline-flex items-center gap-1.5" style="color: var(--cr-text-soft)">
            <UIcon name="i-lucide-info" class="w-3 h-3" />
            Tamaño, orientación y márgenes se configuran en la pestaña <strong>Página</strong>.
          </p>
        </div>
      </Collapsible>

      <Collapsible
        anchor-id="cr-section-hf"
        title="Header & Footer"
        description="HTML repetido en cada página — inline o desde otra plantilla"
        icon="i-lucide-panel-top"
        :badge="get('displayHeaderFooter') ? 'on' : ''"
      >
        <label class="cr-switch">
          <input type="checkbox" :checked="!!get('displayHeaderFooter', false)" @change="(e) => patch({ displayHeaderFooter: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Mostrar header y footer en cada página</span>
        </label>

        <Transition
          enter-active-class="transition-all duration-220 ease-[cubic-bezier(0.23,1,0.32,1)]"
          leave-active-class="transition-all duration-150"
          enter-from-class="opacity-0 -translate-y-1" leave-to-class="opacity-0"
        >
          <div v-if="get('displayHeaderFooter')" class="space-y-5">
            <!-- HEADER block -->
            <div class="cr-hf-block">
              <div class="cr-hf-block-head">
                <span class="cr-hf-block-icon"><UIcon name="i-lucide-arrow-up-from-line" class="w-4 h-4" /></span>
                <div class="flex-1">
                  <p class="cr-hf-block-title">Header</p>
                  <p class="cr-hf-block-desc">Se imprime en la parte superior de cada página</p>
                </div>
              </div>

              <!-- Mode tabs -->
              <div class="cr-hf-tabs">
                <button class="cr-hf-tab" :class="(get('headerMode') || 'inline') === 'inline' ? 'cr-hf-tab--active' : ''"
                  @click="patch({ headerMode: 'inline' })">
                  <UIcon name="i-lucide-code-xml" class="w-3.5 h-3.5" />
                  HTML inline
                </button>
                <button class="cr-hf-tab" :class="get('headerMode') === 'template' ? 'cr-hf-tab--active' : ''"
                  @click="patch({ headerMode: 'template' })">
                  <UIcon name="i-lucide-file-text" class="w-3.5 h-3.5" />
                  Desde plantilla
                </button>
              </div>

              <div v-if="(get('headerMode') || 'inline') === 'inline'">
                <CodeEditor :model-value="get('headerTemplate', '')" language="html" height="110px"
                  placeholder='<div style="font-size:9pt;width:100%;text-align:right">Pág <span class="pageNumber"></span>/<span class="totalPages"></span></div>'
                  @update:model-value="(v) => patch({ headerTemplate: v })" />
              </div>
              <div v-else>
                <select :value="get('headerTemplateShortid', '')" class="cr-input !pl-3 text-[13px] w-full"
                  @change="(e) => patch({ headerTemplateShortid: (e.target as HTMLSelectElement).value || null })">
                  <option value="">— elegir plantilla para usar como header —</option>
                  <option v-for="t in templates.filter(x => x.shortid !== selfShortid)" :key="t.shortid" :value="t.shortid">
                    {{ t.name }} ({{ t.recipe }})
                  </option>
                </select>
                <p class="text-[11px] mt-2 inline-flex items-center gap-1.5" style="color: var(--cr-text-soft)">
                  <UIcon name="i-lucide-info" class="w-3 h-3" />
                  Se renderiza con la misma data y su HTML se usa como header.
                </p>
              </div>
            </div>

            <!-- FOOTER block -->
            <div class="cr-hf-block">
              <div class="cr-hf-block-head">
                <span class="cr-hf-block-icon"><UIcon name="i-lucide-arrow-down-from-line" class="w-4 h-4" /></span>
                <div class="flex-1">
                  <p class="cr-hf-block-title">Footer</p>
                  <p class="cr-hf-block-desc">Se imprime en la parte inferior de cada página</p>
                </div>
              </div>

              <div class="cr-hf-tabs">
                <button class="cr-hf-tab" :class="(get('footerMode') || 'inline') === 'inline' ? 'cr-hf-tab--active' : ''"
                  @click="patch({ footerMode: 'inline' })">
                  <UIcon name="i-lucide-code-xml" class="w-3.5 h-3.5" />
                  HTML inline
                </button>
                <button class="cr-hf-tab" :class="get('footerMode') === 'template' ? 'cr-hf-tab--active' : ''"
                  @click="patch({ footerMode: 'template' })">
                  <UIcon name="i-lucide-file-text" class="w-3.5 h-3.5" />
                  Desde plantilla
                </button>
              </div>

              <div v-if="(get('footerMode') || 'inline') === 'inline'">
                <CodeEditor :model-value="get('footerTemplate', '')" language="html" height="110px"
                  placeholder='<div style="font-size:8pt;width:100%;text-align:center;color:#888">© 2026 — Página <span class="pageNumber"></span></div>'
                  @update:model-value="(v) => patch({ footerTemplate: v })" />
              </div>
              <div v-else>
                <select :value="get('footerTemplateShortid', '')" class="cr-input !pl-3 text-[13px] w-full"
                  @change="(e) => patch({ footerTemplateShortid: (e.target as HTMLSelectElement).value || null })">
                  <option value="">— elegir plantilla para usar como footer —</option>
                  <option v-for="t in templates.filter(x => x.shortid !== selfShortid)" :key="t.shortid" :value="t.shortid">
                    {{ t.name }} ({{ t.recipe }})
                  </option>
                </select>
                <p class="text-[11px] mt-2 inline-flex items-center gap-1.5" style="color: var(--cr-text-soft)">
                  <UIcon name="i-lucide-info" class="w-3 h-3" />
                  Se renderiza con la misma data y su HTML se usa como footer.
                </p>
              </div>
            </div>

            <!-- Variables cheatsheet -->
            <div class="cr-hf-cheat">
              <p class="cr-eyebrow" style="color: var(--cr-text-soft); margin-bottom: 6px">Variables disponibles (Chromium las inyecta)</p>
              <div class="grid grid-cols-2 gap-1.5 text-[11px]" style="color: var(--cr-text-muted)">
                <code class="cr-hf-var">&lt;span class="pageNumber"&gt;&lt;/span&gt;</code>
                <code class="cr-hf-var">&lt;span class="totalPages"&gt;&lt;/span&gt;</code>
                <code class="cr-hf-var">&lt;span class="date"&gt;&lt;/span&gt;</code>
                <code class="cr-hf-var">&lt;span class="title"&gt;&lt;/span&gt;</code>
                <code class="cr-hf-var">&lt;span class="url"&gt;&lt;/span&gt;</code>
              </div>
            </div>
          </div>
        </Transition>
      </Collapsible>

      <Collapsible
        anchor-id="cr-section-render"
        title="Comportamiento de render"
        description="Escala, fondos, espera, medio CSS"
        icon="i-lucide-settings"
        :badge="activeCount('scale','printBackground','preferCSSPageSize','emulateMediaType','waitForNetworkIdle','waitForSelector','timeout')"
      >
        <div>
          <label class="cr-label">Escala</label>
          <div class="flex items-center gap-3">
            <input type="range" min="0.5" max="2" step="0.05" :value="get('scale', 1)" class="cr-slider"
              @input="(e) => patch({ scale: parseFloat((e.target as HTMLInputElement).value) })" />
            <span class="font-mono tabular-nums text-[13px] w-14 text-right" style="color: var(--cr-text)">
              {{ Number(get('scale', 1)).toFixed(2) }}×
            </span>
          </div>
        </div>

        <label class="cr-switch">
          <input type="checkbox" :checked="get('printBackground', true)" @change="(e) => patch({ printBackground: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Imprimir fondos (colores + imágenes)</span>
        </label>

        <label class="cr-switch">
          <input type="checkbox" :checked="!!get('preferCSSPageSize', false)" @change="(e) => patch({ preferCSSPageSize: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Honrar el @page size del CSS</span>
        </label>

        <div>
          <label class="cr-label">Emular media type</label>
          <div class="flex gap-2">
            <button v-for="m in ['screen','print']" :key="m"
              class="cr-chip flex-1" :class="get('emulateMediaType', 'print') === m ? 'cr-chip--active' : ''"
              @click="patch({ emulateMediaType: m })">{{ m }}</button>
          </div>
        </div>

        <label class="cr-switch">
          <input type="checkbox" :checked="!!get('waitForNetworkIdle', false)" @change="(e) => patch({ waitForNetworkIdle: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Wait for network idle</span>
        </label>

        <div>
          <label class="cr-label">Wait for selector (CSS)</label>
          <input type="text" :value="get('waitForSelector', '')" placeholder=".ready, #chart svg"
            class="cr-input !pl-4 font-mono text-[13px]"
            @input="(e) => patch({ waitForSelector: (e.target as HTMLInputElement).value })" />
          <p class="text-[10.5px] mt-1" style="color: var(--cr-text-soft)">
            Espera hasta que el selector aparezca en el DOM. Útil para gráficos JS asíncronos.
          </p>
        </div>

        <div>
          <label class="cr-label">Timeout (ms)</label>
          <input type="number" :value="get('timeout', 30000)" min="1000" step="500"
            class="cr-input !pl-4 font-mono text-[13px]"
            @input="(e) => patch({ timeout: parseInt((e.target as HTMLInputElement).value) || 30000 })" />
        </div>
      </Collapsible>

      <Collapsible
        anchor-id="cr-section-meta"
        title="Metadata del documento"
        description="Title, author, subject, keywords"
        icon="i-lucide-info"
        :badge="activeCount('meta.title','meta.author','meta.subject','meta.keywords','meta.creator')"
      >
        <div v-for="field in [
          { key: 'title',    label: 'Título' },
          { key: 'author',   label: 'Autor' },
          { key: 'subject',  label: 'Asunto' },
          { key: 'keywords', label: 'Keywords (coma)' },
          { key: 'creator',  label: 'Creator' },
        ]" :key="field.key">
          <label class="cr-label">{{ field.label }}</label>
          <input type="text" :value="get(`meta.${field.key}`, '')"
            class="cr-input !pl-4 text-[13px]"
            @input="(e) => setNested('meta', field.key, (e.target as HTMLInputElement).value)" />
        </div>
      </Collapsible>

      <Collapsible
        title="Accesibilidad y PDF/A"
        description="Tagged PDF, outline, compliance"
        icon="i-lucide-accessibility"
        :badge="activeCount('taggedPdf','outline','pdfA')"
      >
        <label class="cr-switch">
          <input type="checkbox" :checked="!!get('taggedPdf', false)" @change="(e) => patch({ taggedPdf: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Tagged PDF (accesibilidad)</span>
        </label>
        <label class="cr-switch">
          <input type="checkbox" :checked="!!get('outline', false)" @change="(e) => patch({ outline: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Generar outline desde &lt;h1..h6&gt;</span>
        </label>
        <div>
          <label class="cr-label">Cumplimiento PDF/A</label>
          <select :value="get('pdfA', '')" class="cr-input !pl-3 text-[13px] w-full"
            @change="(e) => patch({ pdfA: (e.target as HTMLSelectElement).value || null })">
            <option value="">Sin PDF/A</option>
            <option value="1b">PDF/A-1b</option>
            <option value="2b">PDF/A-2b</option>
            <option value="3b">PDF/A-3b</option>
          </select>
        </div>
      </Collapsible>
    </template>

    <!-- ═══════════════════════ weasyprint ═══════════════════════ -->
    <template v-else-if="recipe === 'weasyprint'">
      <Collapsible anchor-id="cr-section-page" title="Página" icon="i-lucide-layout-template" open
        description="Tamaño, orientación, márgenes — se manejan en la tab Página">
        <p class="text-[12px]" style="color: var(--cr-text-muted)">
          WeasyPrint respeta las reglas <code>@page</code> del CSS. Ajustá tamaño y márgenes en la tab <strong>Página</strong>.
        </p>
      </Collapsible>

      <!-- Header & Footer for WeasyPrint via @page running headers -->
      <Collapsible
        anchor-id="cr-section-hf"
        title="Header & Footer"
        description="Texto fijo en cada página — se inyecta como reglas @page running"
        icon="i-lucide-panel-top"
        :badge="activeCount('header.text','header.center','header.right','footer.text','footer.center','footer.right')"
        open
      >
        <p class="text-[12px]" style="color: var(--cr-text-muted)">
          WeasyPrint no usa el mismo sistema que Chromium. En su lugar, definimos
          regiones <code>@top-left</code>, <code>@top-center</code>, <code>@bottom-right</code>, etc.
          Llenalas con el texto que querés y nosotros generamos el CSS automáticamente.
        </p>

        <!-- HEADER block -->
        <div class="cr-hf-block">
          <div class="cr-hf-block-head">
            <span class="cr-hf-block-icon"><UIcon name="i-lucide-arrow-up-from-line" class="w-4 h-4" /></span>
            <div class="flex-1">
              <p class="cr-hf-block-title">Header (arriba)</p>
              <p class="cr-hf-block-desc">Tres regiones: izquierda · centro · derecha</p>
            </div>
          </div>
          <div class="grid grid-cols-3 gap-2">
            <input type="text" :value="get('header.left', '')" placeholder="izquierda"
              class="cr-input !pl-3 text-[12px]"
              @input="(e) => setNested('header', 'left', (e.target as HTMLInputElement).value)" />
            <input type="text" :value="get('header.center', '')" placeholder="centro"
              class="cr-input !pl-3 text-[12px]"
              @input="(e) => setNested('header', 'center', (e.target as HTMLInputElement).value)" />
            <input type="text" :value="get('header.right', '')" placeholder="derecha"
              class="cr-input !pl-3 text-[12px]"
              @input="(e) => setNested('header', 'right', (e.target as HTMLInputElement).value)" />
          </div>
        </div>

        <!-- FOOTER block -->
        <div class="cr-hf-block">
          <div class="cr-hf-block-head">
            <span class="cr-hf-block-icon"><UIcon name="i-lucide-arrow-down-from-line" class="w-4 h-4" /></span>
            <div class="flex-1">
              <p class="cr-hf-block-title">Footer (abajo)</p>
              <p class="cr-hf-block-desc">Tres regiones · usá <code>{page}</code> para número y <code>{pages}</code> para total</p>
            </div>
          </div>
          <div class="grid grid-cols-3 gap-2">
            <input type="text" :value="get('footer.left', '')" placeholder="izquierda"
              class="cr-input !pl-3 text-[12px]"
              @input="(e) => setNested('footer', 'left', (e.target as HTMLInputElement).value)" />
            <input type="text" :value="get('footer.center', '')" placeholder="centro · ej: Pág {page}/{pages}"
              class="cr-input !pl-3 text-[12px]"
              @input="(e) => setNested('footer', 'center', (e.target as HTMLInputElement).value)" />
            <input type="text" :value="get('footer.right', '')" placeholder="derecha"
              class="cr-input !pl-3 text-[12px]"
              @input="(e) => setNested('footer', 'right', (e.target as HTMLInputElement).value)" />
          </div>
        </div>

        <div class="cr-hf-cheat">
          <p class="cr-eyebrow" style="color: var(--cr-text-soft); margin-bottom: 6px">Variables disponibles</p>
          <div class="grid grid-cols-2 gap-1.5 text-[11px]" style="color: var(--cr-text-muted)">
            <code class="cr-hf-var">{page}</code>
            <code class="cr-hf-var">{pages}</code>
          </div>
          <p class="text-[10.5px] mt-2" style="color: var(--cr-text-soft)">
            Ej. en footer-center: <code>Página {page} de {pages}</code>
          </p>
        </div>
      </Collapsible>

      <Collapsible anchor-id="cr-section-render" title="Render" icon="i-lucide-settings"
        :badge="activeCount('presentationalHints','baseUrl','optimize','dpi')" open>
        <label class="cr-switch">
          <input type="checkbox" :checked="get('presentationalHints', true)" @change="(e) => patch({ presentationalHints: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Honrar atributos HTML legacy (align, bgcolor…)</span>
        </label>
        <div>
          <label class="cr-label">Base URL</label>
          <input type="text" :value="get('baseUrl', '')" placeholder="https://cdn.tu-app.com/assets/"
            class="cr-input !pl-4 text-[13px]"
            @input="(e) => patch({ baseUrl: (e.target as HTMLInputElement).value })" />
          <p class="text-[10.5px] mt-1" style="color: var(--cr-text-soft)">Resuelve URLs relativas (imágenes, fonts, CSS).</p>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="cr-label">DPI imágenes</label>
            <input type="number" :value="get('dpi', 300)" min="72" step="24"
              class="cr-input !pl-4 font-mono text-[13px]"
              @input="(e) => patch({ dpi: parseInt((e.target as HTMLInputElement).value) || 300 })" />
          </div>
          <div>
            <label class="cr-label">Comprimir streams</label>
            <label class="cr-switch !mt-2">
              <input type="checkbox" :checked="!!get('optimize', false)" @change="(e) => patch({ optimize: (e.target as HTMLInputElement).checked })" />
              <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
              <span class="cr-switch-label">Optimizar</span>
            </label>
          </div>
        </div>
      </Collapsible>

      <Collapsible title="CSS extra" icon="i-lucide-paintbrush" :badge="get('css') ? 'on' : ''">
        <CodeEditor :model-value="get('css', '')" language="css" height="160px"
          placeholder="@page { @top-center { content: 'Confidencial' } }"
          @update:model-value="(v) => patch({ css: v })" />
        <p class="text-[11px]" style="color: var(--cr-text-soft)">
          Se aplica DESPUÉS de la tab Estilos. Útil para overrides print-only.
        </p>
      </Collapsible>

      <Collapsible anchor-id="cr-section-meta" title="Metadata" icon="i-lucide-info"
        :badge="activeCount('meta.title','meta.author','meta.subject','meta.keywords')">
        <div v-for="field in [
          { key: 'title',    label: 'Título' },
          { key: 'author',   label: 'Autor' },
          { key: 'subject',  label: 'Asunto' },
          { key: 'keywords', label: 'Keywords' },
        ]" :key="field.key">
          <label class="cr-label">{{ field.label }}</label>
          <input type="text" :value="get(`meta.${field.key}`, '')"
            class="cr-input !pl-4 text-[13px]"
            @input="(e) => setNested('meta', field.key, (e.target as HTMLInputElement).value)" />
        </div>
      </Collapsible>

      <Collapsible title="Formularios PDF" icon="i-lucide-square-pen"
        description="Hacer interactivos los <input>/<select>"
        :badge="get('forms') ? 'on' : ''">
        <label class="cr-switch">
          <input type="checkbox" :checked="!!get('forms', false)" @change="(e) => patch({ forms: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Habilitar PDF forms (AcroForm)</span>
        </label>
      </Collapsible>
    </template>

    <!-- ═══════════════════════ docx ═══════════════════════ -->
    <template v-else-if="recipe === 'docx'">
      <Collapsible title="Plantilla" icon="i-lucide-file-text" open>
        <p class="text-[12px] mb-3" style="color: var(--cr-text-muted)">
          Subí un .docx con tags Jinja (<code>&#123;&#123; variable &#125;&#125;</code>, <code>&#123;% for x in items %&#125;</code>).
        </p>
        <div>
          <label class="cr-label">Asset .docx</label>
          <select :value="get('templateAsset.shortid', '')" class="cr-input !pl-4 text-[13px] w-full"
            @change="(e) => setNested('templateAsset', 'shortid', (e.target as HTMLSelectElement).value || null)">
            <option value="">— elegir asset —</option>
            <option v-for="a in assets" :key="a.shortid" :value="a.shortid">{{ a.name }}</option>
          </select>
          <NuxtLink to="/assets" class="text-[12px] mt-1 inline-flex items-center gap-1 hover:underline" style="color: var(--cr-text-muted)">
            <UIcon name="i-lucide-arrow-up-right" class="w-3 h-3" />Subir asset
          </NuxtLink>
        </div>
      </Collapsible>

      <Collapsible title="Modo de templating" icon="i-lucide-settings" :badge="get('mode') || 'jinja'">
        <div class="flex gap-2">
          <button class="cr-chip flex-1" :class="(get('mode') || 'jinja') === 'jinja' ? 'cr-chip--active' : ''" @click="patch({ mode: 'jinja' })">Jinja (docxtpl)</button>
          <button class="cr-chip flex-1" :class="get('mode') === 'block' ? 'cr-chip--active' : ''" @click="patch({ mode: 'block' })">Block</button>
        </div>
      </Collapsible>

      <Collapsible title="HTML embebido" icon="i-lucide-code-xml"
        description="Inyectar HTML dentro del .docx (formato, listas, tablas)"
        :badge="get('htmlEmbed') ? 'on' : ''">
        <label class="cr-switch">
          <input type="checkbox" :checked="!!get('htmlEmbed', false)" @change="(e) => patch({ htmlEmbed: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Procesar tags <code>&#123;&#123; html(...) &#125;&#125;</code></span>
        </label>
      </Collapsible>

      <Collapsible title="Imágenes dinámicas" icon="i-lucide-image"
        description="Variables que contienen URLs/data-URIs de imágenes"
        :badge="get('imageMode') || ''">
        <label class="cr-label">Modo de imagen</label>
        <select :value="get('imageMode', 'auto')" class="cr-input !pl-3 text-[13px] w-full"
          @change="(e) => patch({ imageMode: (e.target as HTMLSelectElement).value })">
          <option value="auto">Auto (detecta data URI / URL)</option>
          <option value="dataUri">Solo data URIs</option>
          <option value="url">Solo URLs (descargar)</option>
        </select>
      </Collapsible>

      <Collapsible title="Protección con password" icon="i-lucide-lock" :badge="get('password') ? 'set' : ''">
        <input type="text" :value="get('password', '')" placeholder="••••••••"
          class="cr-input !pl-4 font-mono text-[13px]"
          @input="(e) => patch({ password: (e.target as HTMLInputElement).value })" />
        <p class="text-[10.5px] mt-1" style="color: var(--cr-text-soft)">Aplicado vía LibreOffice en post-procesado.</p>
      </Collapsible>

      <Collapsible title="Convertir a PDF" icon="i-lucide-file-text" :badge="get('exportPdf') ? 'on' : ''">
        <label class="cr-switch">
          <input type="checkbox" :checked="!!get('exportPdf', false)" @change="(e) => patch({ exportPdf: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Convertir a PDF tras el render (LibreOffice)</span>
        </label>
      </Collapsible>
    </template>

    <!-- ═══════════════════════ xlsx ═══════════════════════ -->
    <template v-else-if="recipe === 'xlsx'">
      <Collapsible title="Modo de generación" icon="i-lucide-sheet" open>
        <div class="flex gap-2 mb-3">
          <button class="cr-chip flex-1" :class="(get('mode') || 'json') === 'json' ? 'cr-chip--active' : ''" @click="patch({ mode: 'json' })">JSON → XLSX</button>
          <button class="cr-chip flex-1" :class="get('mode') === 'asset' ? 'cr-chip--active' : ''" @click="patch({ mode: 'asset' })">Asset .xlsx</button>
        </div>
        <div v-if="get('mode') === 'asset'">
          <label class="cr-label">Asset .xlsx</label>
          <select :value="get('templateAsset.shortid', '')" class="cr-input !pl-4 text-[13px] w-full"
            @change="(e) => setNested('templateAsset', 'shortid', (e.target as HTMLSelectElement).value || null)">
            <option value="">— elegir —</option>
            <option v-for="a in assets" :key="a.shortid" :value="a.shortid">{{ a.name }}</option>
          </select>
        </div>
        <div v-else>
          <p class="text-[12px]" style="color: var(--cr-text-muted)">
            El engine debe producir JSON con shape <code>&#123;sheets:[&#123;name, rows:[[...]]&#125;]&#125;</code>.
          </p>
        </div>
      </Collapsible>

      <Collapsible title="Estilos" icon="i-lucide-paintbrush"
        :badge="activeCount('boldHeader','autofit','frozenRows','zebra')">
        <label class="cr-switch">
          <input type="checkbox" :checked="get('boldHeader', true)" @change="(e) => patch({ boldHeader: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Negrita en primera fila (header)</span>
        </label>
        <label class="cr-switch">
          <input type="checkbox" :checked="get('autofit', true)" @change="(e) => patch({ autofit: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Autoajustar anchos de columna</span>
        </label>
        <label class="cr-switch">
          <input type="checkbox" :checked="!!get('zebra', false)" @change="(e) => patch({ zebra: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Filas alternadas (zebra)</span>
        </label>
        <div>
          <label class="cr-label">Filas congeladas (top)</label>
          <input type="number" :value="get('frozenRows', 1)" min="0" max="10"
            class="cr-input !pl-4 font-mono text-[13px]"
            @input="(e) => patch({ frozenRows: parseInt((e.target as HTMLInputElement).value) || 0 })" />
        </div>
      </Collapsible>

      <Collapsible title="Convertir a PDF" icon="i-lucide-file-text" :badge="get('exportPdf') ? 'on' : ''">
        <label class="cr-switch">
          <input type="checkbox" :checked="!!get('exportPdf', false)" @change="(e) => patch({ exportPdf: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Generar también PDF (vía LibreOffice)</span>
        </label>
      </Collapsible>
    </template>

    <!-- ═══════════════════════ pptx ═══════════════════════ -->
    <template v-else-if="recipe === 'pptx'">
      <Collapsible title="Plantilla" icon="i-lucide-presentation" open>
        <p class="text-[12px] mb-3" style="color: var(--cr-text-muted)">
          Subí un .pptx con texto Jinja en los placeholders. Solo se reemplazan texts existentes — no agregar slides.
        </p>
        <label class="cr-label">Asset .pptx</label>
        <select :value="get('templateAsset.shortid', '')" class="cr-input !pl-4 text-[13px] w-full"
          @change="(e) => setNested('templateAsset', 'shortid', (e.target as HTMLSelectElement).value || null)">
          <option value="">— elegir —</option>
          <option v-for="a in assets" :key="a.shortid" :value="a.shortid">{{ a.name }}</option>
        </select>
      </Collapsible>

      <Collapsible title="Imágenes" icon="i-lucide-image"
        description="Reemplazar placeholders de imagen por variables"
        :badge="get('imageReplace') ? 'on' : ''">
        <label class="cr-switch">
          <input type="checkbox" :checked="!!get('imageReplace', false)" @change="(e) => patch({ imageReplace: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Procesar directivas <code>{% image variable %}</code></span>
        </label>
      </Collapsible>

      <Collapsible title="Convertir a PDF" icon="i-lucide-file-text" :badge="get('exportPdf') ? 'on' : ''">
        <label class="cr-switch">
          <input type="checkbox" :checked="!!get('exportPdf', false)" @change="(e) => patch({ exportPdf: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Generar también PDF</span>
        </label>
      </Collapsible>
    </template>

    <!-- ═══════════════════════ html-to-xlsx ═══════════════════════ -->
    <template v-else-if="recipe === 'html-to-xlsx'">
      <Collapsible title="Hojas" icon="i-lucide-rows-3" open>
        <label class="cr-label">Nombres de hojas (uno por línea)</label>
        <textarea :value="(get('sheetNames', []) as string[]).join('\n')" rows="5"
          placeholder="Resumen&#10;Detalle&#10;Anexos"
          class="cr-input !pl-4 font-mono text-[13px]" style="resize: vertical"
          @input="(e) => patch({ sheetNames: (e.target as HTMLTextAreaElement).value.split('\n').filter(x => x.trim()) })" />
        <p class="text-[11px] mt-1" style="color: var(--cr-text-soft)">
          Cada nombre se asigna a la N-ésima &lt;table&gt; del HTML.
        </p>
      </Collapsible>

      <Collapsible title="Estilos" icon="i-lucide-paintbrush" :badge="activeCount('headerBold','autofit','zebra')">
        <label class="cr-switch">
          <input type="checkbox" :checked="get('headerBold', true)" @change="(e) => patch({ headerBold: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Negrita en headers</span>
        </label>
        <label class="cr-switch">
          <input type="checkbox" :checked="get('autofit', true)" @change="(e) => patch({ autofit: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Autoajustar columnas</span>
        </label>
        <label class="cr-switch">
          <input type="checkbox" :checked="!!get('zebra', false)" @change="(e) => patch({ zebra: (e.target as HTMLInputElement).checked })" />
          <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
          <span class="cr-switch-label">Filas zebra</span>
        </label>
      </Collapsible>
    </template>

    <!-- ═══════════════════════ static-pdf ═══════════════════════ -->
    <template v-else-if="recipe === 'static-pdf'">
      <Collapsible title="PDF base" icon="i-lucide-file-text" open>
        <select :value="get('assetBlobKey', '')" class="cr-input !pl-4 text-[13px] w-full"
          @change="(e) => patch({ assetBlobKey: (e.target as HTMLSelectElement).value || null })">
          <option value="">— elegir asset PDF —</option>
          <option v-for="a in assets" :key="a.shortid" :value="a.shortid">{{ a.name }}</option>
        </select>
        <p class="text-[11px] mt-2" style="color: var(--cr-text-soft)">
          Devuelve este PDF tal cual. Útil para post-procesar (watermark, merge, sign) sin renderizar HTML.
        </p>
      </Collapsible>
    </template>

    <template v-else>
      <div class="cr-card p-6 text-center" style="border-style: dashed">
        <UIcon name="i-lucide-info" class="w-8 h-8 mx-auto mb-2" style="color: var(--cr-text-soft)" />
        <p class="text-[13px]" style="color: var(--cr-text-muted)">
          <code class="px-1 py-0.5 rounded text-[11px]" style="background: var(--cr-surface-soft)">{{ recipe }}</code> no requiere opciones — la salida es directa del engine.
        </p>
      </div>
    </template>
  </div>
</template>

<style>
.cr-recipe-info {
  display: flex;
  gap: 12px;
  padding: 14px;
  border-radius: 12px;
  background: var(--color-wise-100);
  color: var(--color-wise-900);
}
html.dark .cr-recipe-info {
  background: rgb(159 232 112 / 0.10);
  color: var(--color-wise-300);
}
.cr-recipe-info-icon {
  width: 36px; height: 36px;
  border-radius: 10px;
  background: var(--color-wise-400);
  color: #0e0f0c;
  display: inline-flex; align-items: center; justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 2px 6px rgb(159 232 112 / 0.35);
}

/* Quick-jump anchors at the top of the Recipe tab */
.cr-quick-anchor {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 10px;
  background: var(--cr-surface);
  border: 1px solid var(--cr-border);
  font-size: 12px;
  font-weight: 600;
  color: var(--cr-text-muted);
  text-decoration: none;
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    border-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-quick-anchor:hover {
  background: var(--cr-surface-soft);
  border-color: var(--cr-border-strong);
  color: var(--cr-text);
}
.cr-quick-anchor:active { transform: scale(0.97); }

html { scroll-behavior: smooth; }

.cr-switch {
  display: inline-flex; align-items: center; gap: 10px;
  cursor: pointer; user-select: none;
}
.cr-switch input { position: absolute; opacity: 0; pointer-events: none; }
.cr-switch-track {
  position: relative;
  width: 36px; height: 20px;
  border-radius: 9999px;
  background: var(--cr-border-strong);
  transition: background-color 180ms cubic-bezier(0.23, 1, 0.32, 1);
  flex-shrink: 0;
}
.cr-switch-thumb {
  position: absolute; top: 2px; left: 2px;
  width: 16px; height: 16px;
  border-radius: 9999px;
  background: white;
  box-shadow: 0 1px 3px rgb(0 0 0 / 0.20);
  transition: transform 220ms cubic-bezier(0.34, 1.56, 0.64, 1);
}
.cr-switch input:checked + .cr-switch-track { background: var(--color-wise-400); }
.cr-switch input:checked + .cr-switch-track .cr-switch-thumb { transform: translateX(16px); }
.cr-switch-label { font-size: 13px; color: var(--cr-text); }

.cr-slider {
  flex: 1;
  -webkit-appearance: none; appearance: none;
  height: 4px; border-radius: 9999px;
  background: var(--cr-border-strong);
  outline: none; cursor: pointer;
}
.cr-slider::-webkit-slider-thumb {
  -webkit-appearance: none; appearance: none;
  width: 16px; height: 16px; border-radius: 9999px;
  background: var(--color-wise-400);
  border: 2px solid #0e0f0c;
  cursor: grab;
  box-shadow: 0 1px 3px rgb(0 0 0 / 0.20);
  transition: transform 140ms cubic-bezier(0.34, 1.56, 0.64, 1);
}
.cr-slider::-webkit-slider-thumb:active { transform: scale(1.2); cursor: grabbing; }
.cr-slider::-moz-range-thumb {
  width: 16px; height: 16px; border-radius: 9999px;
  background: var(--color-wise-400);
  border: 2px solid #0e0f0c;
  cursor: grab;
}

/* ─── Header/Footer block (visual container for each) ─── */
.cr-hf-block {
  border: 1px solid var(--cr-border);
  border-radius: 12px;
  background: var(--cr-app-bg);
  padding: 14px;
}
.cr-hf-block-head {
  display: flex; align-items: center; gap: 10px; margin-bottom: 12px;
}
.cr-hf-block-icon {
  width: 28px; height: 28px;
  border-radius: 8px;
  background: var(--color-wise-100);
  color: var(--color-wise-800);
  display: inline-flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
html.dark .cr-hf-block-icon {
  background: rgb(159 232 112 / 0.14);
  color: var(--color-wise-300);
}
.cr-hf-block-title {
  font-weight: 700;
  font-size: 13px;
  color: var(--cr-text);
  line-height: 1.2;
}
.cr-hf-block-desc {
  font-size: 11.5px;
  color: var(--cr-text-muted);
  margin-top: 2px;
}

.cr-hf-tabs {
  display: inline-flex;
  padding: 3px;
  margin-bottom: 10px;
  border-radius: 9px;
  background: var(--cr-surface-soft);
  border: 1px solid var(--cr-border);
}
.cr-hf-tab {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--cr-text-muted);
  background: transparent;
  border: none;
  cursor: pointer;
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    box-shadow 140ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-hf-tab:hover { color: var(--cr-text); }
.cr-hf-tab--active {
  background: var(--cr-surface);
  color: var(--cr-text);
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.06);
}

.cr-hf-cheat {
  padding: 12px 14px;
  border-radius: 10px;
  background: var(--cr-surface-soft);
  border: 1px dashed var(--cr-border);
}
.cr-hf-var {
  padding: 2px 6px;
  background: var(--cr-surface);
  border-radius: 5px;
  border: 1px solid var(--cr-border);
  font-family: ui-monospace, "JetBrains Mono", monospace;
  font-size: 10.5px;
  color: var(--cr-text);
  display: inline-block;
}
</style>
