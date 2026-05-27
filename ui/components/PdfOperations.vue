<script setup lang="ts">
// PDF utils configuration — mirrors jsreport-pdf-utils UX:
//   • Quick action buttons at the top (Add header/footer · Add TOC · Add cover)
//   • Tabs: Operations · Meta · Password · Sign · PDF/A · Accessibility · Compression
//
// The underlying data model is a single `pdfOperations` array. Each tab edits
// either:
//   – a list of dynamic operations (Operations tab: merge/append/stamp/prepend)
//   – or a single typed operation (Meta/Password/Sign/PDF/A) — auto-created
//     when the user fills any field

// Op types mirror jsreport-pdf-utils canonical model (pdfProcessing.js):
//   append / prepend / merge — operate on a referenced templateShortid OR
//   a raw uploaded pdfBase64. `merge` + renderForEveryPage = header/footer.
type PdfOpType =
  | 'append'           // jsreport: concat the rendered template at the end
  | 'prepend'          // jsreport: concat the rendered template at the start (cover)
  | 'merge'            // jsreport: stamp/overlay (header/footer when renderForEveryPage)
  | 'addToc'           // generate a Table of Contents from headings
  | 'watermark'        // text/image watermark
  | 'removePages'      // delete pages by range
  | 'encrypt'          // password
  | 'meta'             // title/author/subject/keywords
  | 'sign'             // digital signature
  | 'pdfA'             // PDF/A conformance
  | 'accessibility'    // tagged PDF flags
  | 'compression'      // optimize streams / image quality

interface PdfOp {
  type: PdfOpType
  enabled?: boolean
  // generic
  pdfBase64?: string
  templateShortid?: string
  // jsreport-pdf-utils flags (only meaningful for `merge`)
  renderForEveryPage?: boolean   // true → stamp/overlay every page (header/footer)
  mergeWholeDocument?: boolean   // true → merge page-by-page using the rendered doc
  mergeToFront?: boolean         // true → stamp goes ON TOP of the main page
  pages?: string
  // watermark
  text?: string
  image?: string
  options?: string
  // encrypt
  password?: string
  ownerPassword?: string
  // meta
  title?: string
  author?: string
  subject?: string
  keywords?: string
  creator?: string
  // sign
  certBase64?: string
  certPassword?: string
  reason?: string
  location?: string
  // pdfA
  conformance?: string
  // accessibility
  taggedPdf?: boolean
  documentLanguage?: string
  // compression
  imageQuality?: number
  optimize?: boolean
  // TOC
  tocTitle?: string
  tocDepth?: number
}

interface Props {
  modelValue: PdfOp[]
}
const props = withDefaults(defineProps<Props>(), { modelValue: () => [] })
const emit = defineEmits<{ 'update:modelValue': [v: PdfOp[]] }>()

const ops = computed<PdfOp[]>({
  get() {
    if (Array.isArray(props.modelValue)) return props.modelValue
    try { return JSON.parse(props.modelValue as any) } catch { return [] }
  },
  set(v) { emit('update:modelValue', v) },
})

// ────────────── tabs ──────────────
type Tab = 'operations' | 'meta' | 'password' | 'sign' | 'pdfA' | 'accessibility' | 'compression'
const tab = ref<Tab>('operations')
const tabs: { id: Tab; label: string; icon: string }[] = [
  { id: 'operations',     label: 'Operations',    icon: 'i-lucide-layers' },
  { id: 'meta',           label: 'Meta',          icon: 'i-lucide-info' },
  { id: 'password',       label: 'Password',      icon: 'i-lucide-lock' },
  { id: 'sign',           label: 'Sign',          icon: 'i-lucide-signature' },
  { id: 'pdfA',           label: 'PDF/A',         icon: 'i-lucide-shield-check' },
  { id: 'accessibility',  label: 'Accessibility', icon: 'i-lucide-accessibility' },
  { id: 'compression',    label: 'Compression',   icon: 'i-lucide-archive' },
]

// ────────────── helpers ──────────────
function addOp(op: PdfOp) {
  ops.value = [...ops.value, { enabled: true, ...op }]
}
function removeOp(i: number) {
  ops.value = ops.value.filter((_, idx) => idx !== i)
}
function moveOp(i: number, delta: number) {
  const j = i + delta
  if (j < 0 || j >= ops.value.length) return
  const next = [...ops.value]
  ;[next[i], next[j]] = [next[j], next[i]]
  ops.value = next
}
function updateOp(i: number, patch: Partial<PdfOp>) {
  const next = [...ops.value]
  next[i] = { ...next[i], ...patch }
  ops.value = next
}

/** Single-op tab editor — auto-creates the op the first time you set a field */
function getSingleOp(type: PdfOpType): PdfOp | null {
  return ops.value.find(o => o.type === type) ?? null
}
function upsertSingleOp(type: PdfOpType, patch: Partial<PdfOp>) {
  const idx = ops.value.findIndex(o => o.type === type)
  if (idx >= 0) {
    updateOp(idx, patch)
  } else {
    addOp({ type, ...patch, enabled: true } as PdfOp)
  }
}

// ────────────── quick actions ──────────────
// Header/footer in jsreport = `merge` with renderForEveryPage + mergeToFront.
function quickAddHeaderFooter() {
  addOp({
    type: 'merge',
    templateShortid: '',
    renderForEveryPage: true,
    mergeToFront: true,
    enabled: true,
  })
  tab.value = 'operations'
}
function quickAddTOC() {
  addOp({ type: 'addToc', tocTitle: 'Tabla de contenidos', tocDepth: 3, enabled: true })
  tab.value = 'operations'
}
function quickAddCoverPage() {
  addOp({ type: 'prepend', templateShortid: '', enabled: true })
  tab.value = 'operations'
}

// ────────────── operations tab ──────────────
const opMeta: Record<string, { icon: string; label: string; color: string }> = {
  append:      { icon: 'i-lucide-corner-down-right', label: 'Append template',  color: '#9fe870' },
  prepend:     { icon: 'i-lucide-corner-up-left',    label: 'Cover / Prepend',  color: '#ffc091' },
  merge:       { icon: 'i-lucide-panel-top',         label: 'Merge / Header',   color: '#cdffad' },
  addToc:      { icon: 'i-lucide-list-tree',         label: 'Table of Contents', color: '#38c8ff' },
  watermark:   { icon: 'i-lucide-droplet',           label: 'Watermark',        color: '#ffc091' },
  removePages: { icon: 'i-lucide-scissors',          label: 'Remove pages',     color: '#d03238' },
}

// Backwards-compat: old persisted ops used stampTemplate / appendTemplate /
// prependTemplate. Map them on-the-fly so existing data keeps rendering.
const legacyAlias: Record<string, PdfOpType> = {
  stampTemplate: 'merge',
  appendTemplate: 'append',
  prependTemplate: 'prepend',
}
function effectiveType(op: PdfOp): string {
  return legacyAlias[op.type as string] ?? op.type
}

const dynamicOps = computed(() => {
  return ops.value
    .map((op, i) => ({ op, i }))
    .filter(({ op }) => effectiveType(op) in opMeta)
})

const api = useApi()
const allTemplates = ref<{ shortid: string; name: string; recipe: string }[]>([])
onMounted(async () => {
  try {
    const r = await api.get<{ value: any[] }>('/odata/templates', {
      query: { $top: 500, $select: 'shortid,name,recipe' },
    })
    allTemplates.value = r.value ?? []
  } catch {}
})

const pdfTemplates = computed(() =>
  allTemplates.value.filter(t => ['chrome-pdf', 'weasyprint', 'static-pdf', 'html'].includes(t.recipe)),
)

async function onMergeFile(i: number, e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  const buf = await f.arrayBuffer()
  let bin = ''
  const bytes = new Uint8Array(buf)
  for (let k = 0; k < bytes.byteLength; k++) bin += String.fromCharCode(bytes[k])
  updateOp(i, { pdfBase64: btoa(bin) })
}
async function onCertFile(i: number, e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  const buf = await f.arrayBuffer()
  let bin = ''
  const bytes = new Uint8Array(buf)
  for (let k = 0; k < bytes.byteLength; k++) bin += String.fromCharCode(bytes[k])
  updateOp(i, { certBase64: btoa(bin) })
}

const advancedOpenIdx = ref<number | null>(null)
</script>

<template>
  <div class="cr-pu-root overflow-y-auto h-full pb-4">
    <!-- Title -->
    <div class="cr-pu-title">
      <span class="cr-pu-title-icon">PDF</span>
      <h2>pdf utils configuration</h2>
    </div>

    <!-- Quick actions -->
    <section class="cr-pu-section">
      <p class="cr-pu-eyebrow">Quick actions</p>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-2">
        <button class="cr-pu-quick" @click="quickAddHeaderFooter">
          <UIcon name="i-lucide-panel-top" class="w-4 h-4" />
          <span>Add header/footer</span>
        </button>
        <button class="cr-pu-quick" @click="quickAddTOC">
          <UIcon name="i-lucide-list-tree" class="w-4 h-4" />
          <span>Add Table of Contents</span>
        </button>
        <button class="cr-pu-quick" @click="quickAddCoverPage">
          <UIcon name="i-lucide-image" class="w-4 h-4" />
          <span>Add cover page</span>
        </button>
      </div>
    </section>

    <!-- Tabs -->
    <nav class="cr-pu-tabs">
      <button
        v-for="t in tabs" :key="t.id"
        class="cr-pu-tab"
        :class="tab === t.id ? 'cr-pu-tab--active' : ''"
        @click="tab = t.id"
      >
        <UIcon :name="t.icon" class="w-3.5 h-3.5" />
        <span>{{ t.label }}</span>
      </button>
    </nav>

    <!-- ═══ Operations tab ═══ -->
    <section v-if="tab === 'operations'" class="cr-pu-section">
      <p class="text-[12.5px] mb-3" style="color: var(--cr-text-muted)">
        Use merge/append/stamp operations to add dynamic headers, concatenate multiple PDFs, or insert cover pages.
        Operations run <strong>top to bottom</strong>.
      </p>

      <div v-if="dynamicOps.length === 0" class="cr-pu-empty">
        <UIcon name="i-lucide-layers" class="w-10 h-10 mx-auto mb-2" style="color: var(--cr-text-soft)" />
        <p class="text-[13px]" style="color: var(--cr-text-muted)">Sin operaciones. Empezá con un <strong>Quick action</strong> arriba.</p>
      </div>

      <!-- Operations table -->
      <table v-else class="cr-pu-table">
        <thead>
          <tr>
            <th></th>
            <th>Tipo</th>
            <th>Plantilla / Fuente</th>
            <th>Modo</th>
            <th style="width: 90px">Advanced</th>
            <th style="width: 60px">Enabled</th>
            <th style="width: 32px"></th>
          </tr>
        </thead>
        <tbody>
          <template v-for="{ op, i } in dynamicOps" :key="i">
            <tr class="cr-pu-row">
              <td class="cr-pu-handle">
                <span class="cr-pu-handle-num">{{ i + 1 }}</span>
                <span
                  class="cr-pu-handle-icon"
                  :style="{ background: `${opMeta[effectiveType(op)]?.color}22`, color: opMeta[effectiveType(op)]?.color }"
                >
                  <UIcon :name="opMeta[effectiveType(op)]?.icon" class="w-3.5 h-3.5" />
                </span>
              </td>
              <td>
                <span class="font-semibold text-[13px]" style="color: var(--cr-text)">{{ opMeta[effectiveType(op)]?.label }}</span>
              </td>
              <td>
                <!-- jsreport ops that reference a template -->
                <template v-if="['append','prepend','merge','appendTemplate','prependTemplate','stampTemplate'].includes(op.type)">
                  <!-- merge with no templateShortid AND with an uploaded pdfBase64 = static PDF merge -->
                  <template v-if="op.type === 'merge' && op.pdfBase64 && !op.templateShortid">
                    <input type="file" accept=".pdf,application/pdf" class="text-[11px]"
                      @change="(e) => onMergeFile(i, e)" />
                    <span class="text-[10.5px] ml-1" style="color: var(--color-positive)">
                      ✓ {{ Math.round(op.pdfBase64.length * 0.75 / 1024) }} KB
                    </span>
                  </template>
                  <template v-else>
                    <select :value="op.templateShortid" class="cr-input !pl-2 !py-1.5 !text-[12px] w-full"
                      @change="(e) => updateOp(i, { templateShortid: (e.target as HTMLSelectElement).value })">
                      <option value="">— elegir —</option>
                      <option v-for="t in pdfTemplates" :key="t.shortid" :value="t.shortid">{{ t.name }}</option>
                    </select>
                  </template>
                </template>
                <template v-else-if="op.type === 'watermark'">
                  <input type="text" :value="op.text" placeholder="Texto del watermark"
                    class="cr-input !pl-2 !py-1.5 !text-[12px] w-full"
                    @input="(e) => updateOp(i, { text: (e.target as HTMLInputElement).value })" />
                </template>
                <template v-else-if="op.type === 'removePages'">
                  <input type="text" :value="op.pages" placeholder="1-3, 7, last"
                    class="cr-input !pl-2 !py-1.5 !text-[12px] w-full font-mono"
                    @input="(e) => updateOp(i, { pages: (e.target as HTMLInputElement).value })" />
                </template>
                <template v-else-if="op.type === 'addToc'">
                  <input type="text" :value="op.tocTitle" placeholder="Tabla de contenidos"
                    class="cr-input !pl-2 !py-1.5 !text-[12px] w-full"
                    @input="(e) => updateOp(i, { tocTitle: (e.target as HTMLInputElement).value })" />
                </template>
              </td>
              <td>
                <!-- Mode badge -->
                <template v-if="op.type === 'merge' || op.type === 'stampTemplate'">
                  <span class="cr-pu-pill" style="background: rgb(205 255 173 / 0.40); color: var(--color-wise-800)">
                    {{ op.renderForEveryPage || op.type === 'stampTemplate' ? 'stamp' : (op.mergeWholeDocument ? 'whole' : 'merge') }}
                  </span>
                </template>
                <template v-else-if="op.type === 'prepend' || op.type === 'prependTemplate'">
                  <span class="cr-pu-pill" style="background: rgb(255 192 145 / 0.30); color: #974407">prepend</span>
                </template>
                <template v-else-if="op.type === 'append' || op.type === 'appendTemplate'">
                  <span class="cr-pu-pill" style="background: rgb(56 200 255 / 0.16); color: #0367a4">append</span>
                </template>
                <template v-else>
                  <span class="text-[11px]" style="color: var(--cr-text-soft)">—</span>
                </template>
              </td>
              <td>
                <button class="cr-pu-advanced-btn" @click="advancedOpenIdx = advancedOpenIdx === i ? null : i">
                  advanced
                  <UIcon :name="advancedOpenIdx === i ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" class="w-3 h-3" />
                </button>
              </td>
              <td class="text-center">
                <input type="checkbox" :checked="op.enabled !== false"
                  class="cr-checkbox"
                  @change="(e) => updateOp(i, { enabled: (e.target as HTMLInputElement).checked })" />
              </td>
              <td>
                <button class="cr-row-action cr-row-action--danger !w-7 !h-7" @click="removeOp(i)">
                  <UIcon name="i-lucide-x" class="w-3.5 h-3.5" />
                </button>
              </td>
            </tr>

            <!-- Advanced sub-row -->
            <tr v-if="advancedOpenIdx === i" class="cr-pu-advanced-row">
              <td colspan="7">
                <div class="cr-pu-advanced">
                  <div class="grid grid-cols-2 gap-3">
                    <button class="cr-row-action" :disabled="i === 0" title="Subir" @click="moveOp(i, -1)">
                      <UIcon name="i-lucide-arrow-up" class="w-4 h-4" />
                      <span class="text-[11.5px] ml-1">Mover arriba</span>
                    </button>
                    <button class="cr-row-action" :disabled="i === ops.length - 1" title="Bajar" @click="moveOp(i, 1)">
                      <UIcon name="i-lucide-arrow-down" class="w-4 h-4" />
                      <span class="text-[11.5px] ml-1">Mover abajo</span>
                    </button>
                  </div>
                  <template v-if="['append','prepend','merge','appendTemplate','prependTemplate','stampTemplate'].includes(op.type)">
                    <div>
                      <label class="cr-label">Rango de páginas (opcional)</label>
                      <input type="text" :value="op.pages" placeholder="1-3, last (vacío = todas)"
                        class="cr-input !pl-3 font-mono text-[12px]"
                        @input="(e) => updateOp(i, { pages: (e.target as HTMLInputElement).value })" />
                    </div>
                  </template>
                  <template v-if="op.type === 'merge'">
                    <div class="grid grid-cols-3 gap-2">
                      <label class="inline-flex items-center gap-2 text-[12px]" style="color: var(--cr-text)">
                        <input type="checkbox" :checked="!!op.renderForEveryPage" class="cr-checkbox"
                          @change="(e) => updateOp(i, { renderForEveryPage: (e.target as HTMLInputElement).checked })" />
                        renderForEveryPage
                      </label>
                      <label class="inline-flex items-center gap-2 text-[12px]" style="color: var(--cr-text)">
                        <input type="checkbox" :checked="!!op.mergeToFront" class="cr-checkbox"
                          @change="(e) => updateOp(i, { mergeToFront: (e.target as HTMLInputElement).checked })" />
                        mergeToFront
                      </label>
                      <label class="inline-flex items-center gap-2 text-[12px]" style="color: var(--cr-text)">
                        <input type="checkbox" :checked="!!op.mergeWholeDocument" class="cr-checkbox"
                          @change="(e) => updateOp(i, { mergeWholeDocument: (e.target as HTMLInputElement).checked })" />
                        mergeWholeDocument
                      </label>
                    </div>
                  </template>
                  <template v-if="op.type === 'watermark'">
                    <div>
                      <label class="cr-label">Opciones pdfcpu</label>
                      <input type="text" :value="op.options" placeholder="scale:0.5 abs, op:0.3, rot:30"
                        class="cr-input !pl-3 font-mono text-[12px]"
                        @input="(e) => updateOp(i, { options: (e.target as HTMLInputElement).value })" />
                    </div>
                  </template>
                  <template v-if="op.type === 'addToc'">
                    <div>
                      <label class="cr-label">Profundidad (h1..h&lt;n&gt;)</label>
                      <input type="number" :value="op.tocDepth || 3" min="1" max="6"
                        class="cr-input !pl-3 font-mono text-[12px]"
                        @input="(e) => updateOp(i, { tocDepth: parseInt((e.target as HTMLInputElement).value) })" />
                    </div>
                  </template>
                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>

      <!-- Add operation button -->
      <details class="cr-pu-add">
        <summary>
          <UIcon name="i-lucide-plus" class="w-3.5 h-3.5" />
          Add operation
        </summary>
        <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 mt-3">
          <button v-for="(m, t) in opMeta" :key="t" class="cr-pdfop-add"
            @click="addOp({ type: t as PdfOpType, enabled: true })">
            <span class="cr-pdfop-add-icon" :style="{ background: `${m.color}22`, color: m.color }">
              <UIcon :name="m.icon" class="w-4 h-4" />
            </span>
            <span class="cr-pdfop-add-text">
              <span class="cr-pdfop-add-label">{{ m.label }}</span>
            </span>
          </button>
        </div>
      </details>
    </section>

    <!-- ═══ Meta tab ═══ -->
    <section v-else-if="tab === 'meta'" class="cr-pu-section">
      <p class="text-[12.5px] mb-3" style="color: var(--cr-text-muted)">
        Set the PDF Info dictionary — visible in the file properties of any PDF viewer.
      </p>
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label class="cr-label">Title</label>
          <input type="text" :value="getSingleOp('meta')?.title || ''"
            class="cr-input !pl-3 text-[13px]"
            @input="(e) => upsertSingleOp('meta', { title: (e.target as HTMLInputElement).value })" />
        </div>
        <div>
          <label class="cr-label">Author</label>
          <input type="text" :value="getSingleOp('meta')?.author || ''"
            class="cr-input !pl-3 text-[13px]"
            @input="(e) => upsertSingleOp('meta', { author: (e.target as HTMLInputElement).value })" />
        </div>
        <div>
          <label class="cr-label">Subject</label>
          <input type="text" :value="getSingleOp('meta')?.subject || ''"
            class="cr-input !pl-3 text-[13px]"
            @input="(e) => upsertSingleOp('meta', { subject: (e.target as HTMLInputElement).value })" />
        </div>
        <div>
          <label class="cr-label">Keywords (coma)</label>
          <input type="text" :value="getSingleOp('meta')?.keywords || ''"
            placeholder="reporte, q4, finanzas"
            class="cr-input !pl-3 text-[13px]"
            @input="(e) => upsertSingleOp('meta', { keywords: (e.target as HTMLInputElement).value })" />
        </div>
        <div class="col-span-2">
          <label class="cr-label">Creator</label>
          <input type="text" :value="getSingleOp('meta')?.creator || 'cloud-report'"
            class="cr-input !pl-3 text-[13px]"
            @input="(e) => upsertSingleOp('meta', { creator: (e.target as HTMLInputElement).value })" />
        </div>
      </div>
    </section>

    <!-- ═══ Password tab ═══ -->
    <section v-else-if="tab === 'password'" class="cr-pu-section">
      <p class="text-[12.5px] mb-3" style="color: var(--cr-text-muted)">
        AES-256 encryption — the PDF asks for the password to open.
      </p>
      <div>
        <label class="cr-label">User password</label>
        <input type="text" :value="getSingleOp('encrypt')?.password || ''"
          placeholder="••••••••"
          class="cr-input !pl-3 font-mono text-[13px]"
          @input="(e) => upsertSingleOp('encrypt', { password: (e.target as HTMLInputElement).value })" />
        <p class="text-[10.5px] mt-1" style="color: var(--cr-text-soft)">
          Se aplica como user + owner password (256-bit AES vía pdfcpu).
        </p>
      </div>
    </section>

    <!-- ═══ Sign tab ═══ -->
    <section v-else-if="tab === 'sign'" class="cr-pu-section">
      <p class="text-[12.5px] mb-3" style="color: var(--cr-text-muted)">
        Firma digital con certificado .p12 / .pfx. Soporta motivo y ubicación visibles.
      </p>
      <div class="space-y-3">
        <div>
          <label class="cr-label">Certificado .p12 / .pfx</label>
          <input type="file" accept=".p12,.pfx" class="text-[12px]"
            @change="(e) => upsertSingleOp('sign', { certBase64: undefined } /* placeholder */) || onCertFile(ops.findIndex(o => o.type === 'sign'), e)" />
          <p v-if="getSingleOp('sign')?.certBase64" class="text-[10.5px] mt-1" style="color: var(--color-positive)">
            ✓ Certificado cargado ({{ Math.round((getSingleOp('sign')?.certBase64?.length || 0) * 0.75 / 1024) }} KB)
          </p>
        </div>
        <div>
          <label class="cr-label">Password del certificado</label>
          <input type="text" :value="getSingleOp('sign')?.certPassword || ''"
            class="cr-input !pl-3 font-mono text-[13px]"
            @input="(e) => upsertSingleOp('sign', { certPassword: (e.target as HTMLInputElement).value })" />
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="cr-label">Razón</label>
            <input type="text" :value="getSingleOp('sign')?.reason || ''"
              placeholder="Aprobado por dirección"
              class="cr-input !pl-3 text-[13px]"
              @input="(e) => upsertSingleOp('sign', { reason: (e.target as HTMLInputElement).value })" />
          </div>
          <div>
            <label class="cr-label">Lugar</label>
            <input type="text" :value="getSingleOp('sign')?.location || ''"
              placeholder="La Paz, Bolivia"
              class="cr-input !pl-3 text-[13px]"
              @input="(e) => upsertSingleOp('sign', { location: (e.target as HTMLInputElement).value })" />
          </div>
        </div>
        <p class="text-[11px]" style="color: var(--cr-text-soft)">
          ⚠ Backend stub — necesita un servicio de firma externo (próximamente).
        </p>
      </div>
    </section>

    <!-- ═══ PDF/A tab ═══ -->
    <section v-else-if="tab === 'pdfA'" class="cr-pu-section">
      <p class="text-[12.5px] mb-3" style="color: var(--cr-text-muted)">
        Marca el documento como compliant con un perfil PDF/A.
      </p>
      <label class="cr-label">Nivel de conformidad</label>
      <div class="flex gap-2">
        <button class="cr-chip flex-1" :class="!getSingleOp('pdfA')?.conformance ? 'cr-chip--active' : ''"
          @click="upsertSingleOp('pdfA', { conformance: '' })">Sin PDF/A</button>
        <button v-for="lvl in ['1b','2b','2u','3b']" :key="lvl"
          class="cr-chip flex-1"
          :class="getSingleOp('pdfA')?.conformance === lvl ? 'cr-chip--active' : ''"
          @click="upsertSingleOp('pdfA', { conformance: lvl })">PDF/A-{{ lvl }}</button>
      </div>
      <p class="text-[11px] mt-3" style="color: var(--cr-text-soft)">
        2b: visual (recomendado) · 2u: con texto extraíble · 3b: permite archivos embebidos · 1b: legacy
      </p>
    </section>

    <!-- ═══ Accessibility tab ═══ -->
    <section v-else-if="tab === 'accessibility'" class="cr-pu-section">
      <p class="text-[12.5px] mb-3" style="color: var(--cr-text-muted)">
        Accesibilidad para lectores de pantalla, alto contraste, etc.
      </p>
      <label class="cr-switch">
        <input type="checkbox" :checked="!!getSingleOp('accessibility')?.taggedPdf"
          @change="(e) => upsertSingleOp('accessibility', { taggedPdf: (e.target as HTMLInputElement).checked })" />
        <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
        <span class="cr-switch-label">Tagged PDF (estructura semántica para screen readers)</span>
      </label>
      <div class="mt-4">
        <label class="cr-label">Idioma del documento</label>
        <select :value="getSingleOp('accessibility')?.documentLanguage || 'es'"
          class="cr-input !pl-3 text-[13px] w-full"
          @change="(e) => upsertSingleOp('accessibility', { documentLanguage: (e.target as HTMLSelectElement).value })">
          <option value="es">Español</option>
          <option value="en">English</option>
          <option value="pt">Português</option>
          <option value="fr">Français</option>
          <option value="de">Deutsch</option>
        </select>
      </div>
    </section>

    <!-- ═══ Compression tab ═══ -->
    <section v-else-if="tab === 'compression'" class="cr-pu-section">
      <p class="text-[12.5px] mb-3" style="color: var(--cr-text-muted)">
        Reduce el tamaño del archivo final (imágenes recomprimidas, streams comprimidos).
      </p>
      <label class="cr-switch">
        <input type="checkbox" :checked="!!getSingleOp('compression')?.optimize"
          @change="(e) => upsertSingleOp('compression', { optimize: (e.target as HTMLInputElement).checked })" />
        <span class="cr-switch-track"><span class="cr-switch-thumb"></span></span>
        <span class="cr-switch-label">Optimizar streams (Flate / deflate)</span>
      </label>
      <div class="mt-4">
        <label class="cr-label">Calidad de imágenes JPEG (1-100)</label>
        <div class="flex items-center gap-3">
          <input type="range" min="30" max="100" step="5"
            :value="getSingleOp('compression')?.imageQuality || 85"
            class="cr-slider"
            @input="(e) => upsertSingleOp('compression', { imageQuality: parseInt((e.target as HTMLInputElement).value) })" />
          <span class="font-mono tabular-nums text-[13px] w-10 text-right" style="color: var(--cr-text)">
            {{ getSingleOp('compression')?.imageQuality || 85 }}
          </span>
        </div>
        <p class="text-[10.5px] mt-1" style="color: var(--cr-text-soft)">
          85 es default — balance calidad/tamaño. Bajalo a 60 para máxima compresión.
        </p>
      </div>
    </section>
  </div>
</template>

<style>
.cr-pu-root {
  padding: 4px 2px;
}

.cr-pu-title {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-bottom: 14px;
  margin-bottom: 14px;
  border-bottom: 1px solid var(--cr-border);
}
.cr-pu-title-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 9px;
  background: var(--color-wise-100);
  color: var(--color-wise-800);
  font-weight: 800;
  font-size: 11px;
  letter-spacing: 0.05em;
}
html.dark .cr-pu-title-icon {
  background: rgb(159 232 112 / 0.14);
  color: var(--color-wise-300);
}
.cr-pu-title h2 {
  font-size: 18px;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--cr-text);
}

.cr-pu-section { margin-bottom: 18px; }

.cr-pu-eyebrow {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.10em;
  text-transform: uppercase;
  color: var(--cr-text-soft);
  margin-bottom: 8px;
}

.cr-pu-quick {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 11px 14px;
  border-radius: 10px;
  background: var(--color-wise-100);
  color: var(--color-wise-800);
  border: 1px solid var(--color-wise-500);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1),
    box-shadow 140ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-pu-quick:hover {
  background: var(--color-wise-200);
  box-shadow: 0 4px 12px rgb(159 232 112 / 0.20);
}
.cr-pu-quick:active { transform: scale(0.98); }
html.dark .cr-pu-quick {
  background: rgb(159 232 112 / 0.14);
  color: var(--color-wise-300);
  border-color: var(--color-wise-500);
}

.cr-pu-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
  padding: 3px;
  background: var(--cr-surface-soft);
  border: 1px solid var(--cr-border);
  border-radius: 10px;
  margin-bottom: 16px;
}
.cr-pu-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 12px;
  border-radius: 7px;
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
.cr-pu-tab:hover { color: var(--cr-text); }
.cr-pu-tab--active {
  background: var(--cr-surface);
  color: var(--cr-text);
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.06);
}

/* Operations table */
.cr-pu-table {
  width: 100%;
  border-collapse: collapse;
  border: 1px solid var(--cr-border);
  border-radius: 10px;
  overflow: hidden;
  background: var(--cr-surface);
}
.cr-pu-table th {
  background: var(--cr-surface-soft);
  padding: 9px 10px;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--cr-text-muted);
  text-align: left;
  border-bottom: 1px solid var(--cr-border);
}
.cr-pu-row td {
  padding: 10px;
  border-bottom: 1px solid var(--cr-border);
  vertical-align: middle;
}
.cr-pu-row:last-child td { border-bottom: none; }

.cr-pu-handle {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 60px;
}
.cr-pu-handle-num {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 6px;
  background: var(--cr-surface-soft);
  color: var(--cr-text-soft);
  font-size: 10px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}
.cr-pu-handle-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 6px;
  flex-shrink: 0;
}

.cr-pu-pill {
  display: inline-flex;
  align-items: center;
  padding: 3px 8px;
  border-radius: 9999px;
  font-size: 10.5px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.cr-pu-advanced-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 10px;
  border-radius: 6px;
  background: var(--cr-surface-soft);
  color: var(--cr-text);
  border: 1px solid var(--cr-border);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-pu-advanced-btn:hover { background: var(--cr-border); }
.cr-pu-advanced-btn:active { transform: scale(0.96); }

.cr-pu-advanced-row td {
  background: var(--cr-surface-soft);
  padding: 14px 14px 14px 76px;
}
.cr-pu-advanced { display: flex; flex-direction: column; gap: 12px; }

.cr-pu-empty {
  padding: 30px 20px;
  border: 1px dashed var(--cr-border);
  border-radius: 10px;
  text-align: center;
  background: var(--cr-surface);
}

.cr-pu-add {
  margin-top: 14px;
}
.cr-pu-add > summary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: 8px;
  background: var(--cr-surface);
  border: 1px solid var(--cr-border);
  font-size: 12px;
  font-weight: 600;
  color: var(--cr-text);
  cursor: pointer;
  list-style: none;
  transition: background-color 140ms;
}
.cr-pu-add > summary::-webkit-details-marker { display: none; }
.cr-pu-add > summary:hover { background: var(--cr-surface-soft); }
</style>
