<script setup lang="ts">
// Code editor with Shiki syntax highlighting.
//
// Implementation: a transparent <textarea> stacked on top of a <pre> showing
// the highlighted version. Same font / line-height / padding so cursors align
// pixel-perfect with the colored text underneath. This is the technique used
// by react-simple-code-editor, CodeMirror's prototype, etc.
//
// Highlighting runs lazy (debounced) so typing stays responsive even on large
// files.

import { getHighlighter, normalizeLang, isDark, type ShikiLang } from '~/composables/useShiki'

interface Props {
  modelValue: string
  language?: string
  /** Fixed pixel height, or "100%" / "fill" to grow with the parent flex item. */
  height?: string
  placeholder?: string
  readonly?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  language: 'html',
  height: '420px',
  placeholder: '',
  readonly: false,
})

/** When height === '100%' / 'fill' we let the parent flex layout drive the
 *  height; the editor then becomes `flex: 1; min-height: 0`. */
const isFill = computed(() => props.height === '100%' || props.height === 'fill')

const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const textarea = ref<HTMLTextAreaElement | null>(null)
const gutter = ref<HTMLElement | null>(null)
const highlightedHTML = ref<string>('')
const scrollTop = ref(0)
const scrollLeft = ref(0)
const dark = ref(false)

// Soft-wrap toggle — persisted in a cookie so it survives reloads.
//
// Default is OFF (matches VS Code / GitHub / most code editors). When OFF the
// line numbers in the gutter stay perfectly aligned with their lines; when ON
// we accept the gutter misalignment in exchange for not having a horizontal
// scrollbar. We dim the gutter when wrap is on as a visual hint.
const wrap = useCookie<boolean>('cr-editor-wrap', {
  default: () => false, sameSite: 'lax', maxAge: 60 * 60 * 24 * 365, path: '/',
})

const lang = computed<ShikiLang>(() => normalizeLang(props.language))
const lineCount = computed(() => props.modelValue.split('\n').length || 1)

let highlightTimer: ReturnType<typeof setTimeout> | null = null

async function rehighlight() {
  try {
    const hl = await getHighlighter()
    const theme = dark.value ? 'github-dark' : 'github-light'
    let html = hl.codeToHtml(props.modelValue || ' ', {
      lang: lang.value,
      theme,
    })
    // Shiki ships <pre> with `tabindex`, `style="background-color:...;color:..."`,
    // and sometimes `class="shiki ..."`. Strip everything except the inner
    // content so our CSS reset is the only style applied. This also keeps
    // pre/code from accidentally capturing focus or scroll.
    html = html
      .replace(/<pre[^>]*>/, '<pre>')
      .replace(/<code[^>]*>/, '<code>')
    highlightedHTML.value = html
  } catch (err) {
    // If shiki fails (rare), fall back to plain pre so we never end up with
    // an empty highlight overlay that hides part of the textarea behind it.
    const safe = (props.modelValue || ' ')
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
    highlightedHTML.value = `<pre><code>${safe}</code></pre>`
  }
}

function scheduleHighlight() {
  if (highlightTimer) clearTimeout(highlightTimer)
  highlightTimer = setTimeout(rehighlight, 40)
}

function onInput(e: Event) {
  emit('update:modelValue', (e.target as HTMLTextAreaElement).value)
  scheduleHighlight()
}

function onScroll(e: Event) {
  const t = e.target as HTMLTextAreaElement
  scrollTop.value = t.scrollTop
  scrollLeft.value = t.scrollLeft
  // Sync the gutter (line numbers) with vertical scroll so numbers track lines.
  if (gutter.value) gutter.value.scrollTop = t.scrollTop
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Tab' && !props.readonly) {
    e.preventDefault()
    const t = e.target as HTMLTextAreaElement
    const s = t.selectionStart
    const E = t.selectionEnd
    const v = t.value
    const next = v.slice(0, s) + '  ' + v.slice(E)
    emit('update:modelValue', next)
    nextTick(() => {
      t.selectionStart = t.selectionEnd = s + 2
      scheduleHighlight()
    })
    return
  }

  // Find: Ctrl/Cmd + F → open the find bar
  const isMac = navigator.platform.toUpperCase().includes('MAC')
  const ctrl = isMac ? e.metaKey : e.ctrlKey
  if (ctrl && e.key.toLowerCase() === 'f') {
    e.preventDefault()
    openFind()
  }
}

// ────────────── FIND BAR ──────────────────────────────────────────────────
const findVisible = ref(false)
const findQuery = ref('')
const findReplace = ref('')
const findInput = ref<HTMLInputElement | null>(null)
const matches = computed<number[]>(() => {
  if (!findQuery.value) return []
  const q = findQuery.value
  const v = props.modelValue
  const re = new RegExp(q.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi')
  const out: number[] = []
  let m: RegExpExecArray | null
  while ((m = re.exec(v))) {
    out.push(m.index)
    if (m.index === re.lastIndex) re.lastIndex++ // avoid zero-length infinite loop
  }
  return out
})
const matchIdx = ref(0)

function openFind() {
  findVisible.value = true
  // Pre-fill with selected text if any.
  if (textarea.value) {
    const t = textarea.value
    if (t.selectionEnd > t.selectionStart) {
      findQuery.value = t.value.slice(t.selectionStart, t.selectionEnd)
    }
  }
  nextTick(() => findInput.value?.focus())
}
function closeFind() {
  findVisible.value = false
  textarea.value?.focus()
}
function focusMatch(idx: number) {
  if (!textarea.value || matches.value.length === 0) return
  const norm = ((idx % matches.value.length) + matches.value.length) % matches.value.length
  matchIdx.value = norm
  const pos = matches.value[norm]
  const t = textarea.value
  t.focus()
  t.setSelectionRange(pos, pos + findQuery.value.length)
  // Scroll so the match is visible
  const before = t.value.slice(0, pos)
  const lineNum = before.split('\n').length - 1
  const lineHeight = 21
  t.scrollTop = Math.max(0, lineNum * lineHeight - 100)
}
function findNext() { focusMatch(matchIdx.value + 1) }
function findPrev() { focusMatch(matchIdx.value - 1) }
function replaceOne() {
  if (matches.value.length === 0 || props.readonly) return
  const pos = matches.value[matchIdx.value]
  const v = props.modelValue
  const next = v.slice(0, pos) + findReplace.value + v.slice(pos + findQuery.value.length)
  emit('update:modelValue', next)
  scheduleHighlight()
}
function replaceAll() {
  if (!findQuery.value || props.readonly) return
  const re = new RegExp(findQuery.value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi')
  emit('update:modelValue', props.modelValue.replace(re, findReplace.value))
  scheduleHighlight()
}
function onFindKey(e: KeyboardEvent) {
  if (e.key === 'Escape') { e.preventDefault(); closeFind() }
  else if (e.key === 'Enter') {
    e.preventDefault()
    if (e.shiftKey) findPrev()
    else findNext()
  }
}
watch(findQuery, () => { matchIdx.value = 0; if (findQuery.value) nextTick(() => focusMatch(0)) })

watch(() => props.modelValue, scheduleHighlight)
watch(() => props.language, scheduleHighlight)

// React to dark-mode toggles by observing the html.dark class.
let observer: MutationObserver | null = null

onMounted(async () => {
  dark.value = isDark()
  await rehighlight()
  observer = new MutationObserver(() => {
    const nowDark = isDark()
    if (nowDark !== dark.value) {
      dark.value = nowDark
      rehighlight()
    }
  })
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
})

onBeforeUnmount(() => {
  observer?.disconnect()
  if (highlightTimer) clearTimeout(highlightTimer)
})
</script>

<template>
  <div class="cr-editor" :class="isFill ? 'cr-editor--fill' : ''" :style="isFill ? undefined : { height }">
    <div class="cr-editor-toolbar">
      <span class="cr-editor-language">{{ language }}</span>
      <div class="flex items-center gap-2">
        <button
          type="button"
          class="cr-editor-toolbar-btn"
          :class="wrap ? 'cr-editor-toolbar-btn--active' : ''"
          :title="wrap ? 'Word-wrap ON · click para desactivar' : 'Word-wrap OFF · click para activar'"
          @click="wrap = !wrap"
        >
          <UIcon name="i-lucide-wrap-text" class="w-3.5 h-3.5" />
        </button>
        <button
          v-if="!readonly"
          type="button"
          class="cr-editor-toolbar-btn"
          title="Buscar (Ctrl+F)"
          @click="openFind"
        >
          <UIcon name="i-lucide-search" class="w-3.5 h-3.5" />
        </button>
        <span class="text-[11px] tabular-nums" style="color: var(--cr-text-soft)">
          {{ lineCount }} líneas · {{ modelValue.length }} chars
        </span>
      </div>
    </div>

    <!-- Find / Replace bar -->
    <Transition
      enter-active-class="transition-all duration-180 ease-[cubic-bezier(0.23,1,0.32,1)]"
      leave-active-class="transition-all duration-120"
      enter-from-class="opacity-0 -translate-y-1"
      leave-to-class="opacity-0 -translate-y-1"
    >
      <div v-if="findVisible" class="cr-editor-find" @keydown="onFindKey">
        <div class="flex items-center gap-1.5">
          <UIcon name="i-lucide-search" class="w-3.5 h-3.5 shrink-0" style="color: var(--cr-text-soft)" />
          <input
            ref="findInput"
            v-model="findQuery"
            type="text"
            placeholder="Buscar"
            class="cr-editor-find-input flex-1"
          />
          <span class="text-[10.5px] font-mono tabular-nums w-12 text-right" style="color: var(--cr-text-soft)">
            {{ matches.length ? `${matchIdx + 1}/${matches.length}` : '0/0' }}
          </span>
          <button type="button" class="cr-row-action !w-7 !h-7" title="Anterior (Shift+Enter)" @click="findPrev">
            <UIcon name="i-lucide-chevron-up" class="w-3.5 h-3.5" />
          </button>
          <button type="button" class="cr-row-action !w-7 !h-7" title="Siguiente (Enter)" @click="findNext">
            <UIcon name="i-lucide-chevron-down" class="w-3.5 h-3.5" />
          </button>
          <button type="button" class="cr-row-action !w-7 !h-7" title="Cerrar (Esc)" @click="closeFind">
            <UIcon name="i-lucide-x" class="w-3.5 h-3.5" />
          </button>
        </div>
        <div v-if="!readonly" class="flex items-center gap-1.5 mt-1.5">
          <UIcon name="i-lucide-pen-line" class="w-3.5 h-3.5 shrink-0" style="color: var(--cr-text-soft)" />
          <input
            v-model="findReplace"
            type="text"
            placeholder="Reemplazar"
            class="cr-editor-find-input flex-1"
            @keydown.enter.prevent="replaceOne"
          />
          <button type="button" class="cr-editor-find-btn" :disabled="matches.length === 0" @click="replaceOne">
            Reemplazar
          </button>
          <button type="button" class="cr-editor-find-btn" :disabled="!findQuery" @click="replaceAll">
            Todo
          </button>
        </div>
      </div>
    </Transition>

    <div class="cr-editor-body" :class="wrap ? 'cr-editor-body--wrap' : ''">
      <pre ref="gutter" class="cr-editor-gutter" aria-hidden="true">{{
        Array.from({ length: lineCount }, (_, i) => i + 1).join('\n')
      }}</pre>

      <!-- Stacked highlighter + textarea. Scroll positions are synced.
           NOTE: the transform must be applied to an *inner* wrapper, not to
           .cr-editor-highlight itself. The outer box has `inset: 0` + `overflow:
           hidden`, so moving the box (instead of the content) shrinks the
           visible area as you scroll down — causing the bottom of the file to
           disappear. By transforming the inner element, the clipping window
           stays put and the syntax-coloured slice scrolls within it. -->
      <div class="cr-editor-stack">
        <div class="cr-editor-highlight" aria-hidden="true">
          <div
            class="cr-editor-highlight-inner"
            :style="{
              transform: `translate(${-scrollLeft}px, ${-scrollTop}px)`,
            }"
            v-html="highlightedHTML"
          />
        </div>
        <textarea
          ref="textarea"
          :value="modelValue"
          :placeholder="placeholder"
          :readonly="readonly"
          class="cr-editor-textarea"
          spellcheck="false"
          autocomplete="off"
          autocorrect="off"
          autocapitalize="off"
          @input="onInput"
          @scroll="onScroll"
          @keydown="onKeyDown"
        />
      </div>
    </div>
  </div>
</template>

<style>
.cr-editor {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--cr-border);
  border-radius: 14px;
  overflow: hidden;
  background: var(--cr-surface);
}
.cr-editor--fill {
  flex: 1;
  min-height: 0;
}
.cr-editor-toolbar {
  height: 38px;
  padding: 0 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--cr-surface-soft);
  border-bottom: 1px solid var(--cr-border);
  font-size: 11px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--cr-text-muted);
  font-weight: 700;
}
.cr-editor-language {
  display: inline-flex;
  align-items: center;
  padding: 3px 8px;
  border-radius: 6px;
  background: var(--cr-surface);
  border: 1px solid var(--cr-border);
  color: var(--cr-text);
}
.cr-editor-toolbar-btn {
  width: 24px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  background: var(--cr-surface);
  border: 1px solid var(--cr-border);
  color: var(--cr-text-muted);
  cursor: pointer;
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-editor-toolbar-btn:hover {
  color: var(--cr-text);
  border-color: var(--cr-border-strong);
}
.cr-editor-toolbar-btn:active { transform: scale(0.94); }
.cr-editor-toolbar-btn--active {
  background: var(--color-wise-400);
  color: #0e0f0c;
  border-color: var(--color-wise-500);
}
html.dark .cr-editor-toolbar-btn--active {
  background: var(--color-wise-400);
  color: #0e0f0c;
}

/* Find / Replace bar */
.cr-editor-find {
  padding: 8px 10px;
  background: var(--cr-surface-soft);
  border-bottom: 1px solid var(--cr-border);
  flex-shrink: 0;
}
.cr-editor-find-input {
  background: var(--cr-surface);
  border: 1px solid var(--cr-border);
  border-radius: 6px;
  padding: 4px 8px;
  font-size: 12px;
  font-family: ui-monospace, "JetBrains Mono", monospace;
  color: var(--cr-text);
  outline: none;
  transition: border-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
              box-shadow 140ms cubic-bezier(0.23, 1, 0.32, 1);
  min-width: 0;
}
.cr-editor-find-input:focus {
  border-color: var(--color-wise-500);
  box-shadow: 0 0 0 3px rgb(159 232 112 / 0.20);
}
.cr-editor-find-btn {
  padding: 4px 8px;
  border-radius: 6px;
  background: var(--cr-surface);
  border: 1px solid var(--cr-border);
  color: var(--cr-text);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    border-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-editor-find-btn:hover { background: var(--cr-border); }
.cr-editor-find-btn:active { transform: scale(0.96); }
.cr-editor-find-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.cr-editor-body {
  flex: 1;
  display: flex;
  overflow: hidden;
  position: relative;
}
.cr-editor-gutter {
  padding: 12px 12px 12px 16px;
  text-align: right;
  font-family: ui-monospace, "JetBrains Mono", "SF Mono", Menlo, Consolas, monospace;
  font-size: 13px;
  line-height: 21px;
  color: var(--cr-text-soft);
  background: var(--cr-surface-soft);
  border-right: 1px solid var(--cr-border);
  user-select: none;
  white-space: pre;
  min-width: 50px;
  overflow: hidden;
  flex-shrink: 0;
  transition: opacity 140ms cubic-bezier(0.23, 1, 0.32, 1);
}
/* When wrap is on, the gutter numbers no longer align 1:1 with visible lines
   (each logical line may take several visual rows). Dim the gutter so users
   know the numbers are approximate. */
.cr-editor-body--wrap .cr-editor-gutter {
  opacity: 0.45;
}
.cr-editor-stack {
  flex: 1;
  position: relative;
  overflow: hidden;
}

/* highlighted layer — fixed window that clips the scrolling inner wrapper.
   Padding lives on the inner wrapper so the transform offsets are measured
   against the same origin as the textarea's scrollTop. */
.cr-editor-highlight {
  position: absolute;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
  background: transparent;
}
.cr-editor-highlight-inner {
  padding: 12px 16px;
  font-family: ui-monospace, "JetBrains Mono", "SF Mono", Menlo, Consolas, monospace;
  font-size: 13px;
  line-height: 21px;
  white-space: pre;
  tab-size: 2;
  background: transparent;
  will-change: transform;
}

/* When word-wrap is on, both layers wrap long lines and the horizontal
   scroll disappears. */
.cr-editor-body--wrap .cr-editor-highlight-inner,
.cr-editor-body--wrap .cr-editor-highlight pre,
.cr-editor-body--wrap .cr-editor-highlight code,
.cr-editor-body--wrap .cr-editor-textarea {
  white-space: pre-wrap !important;
  word-break: break-word;
  overflow-wrap: anywhere;
}
.cr-editor-body--wrap .cr-editor-textarea {
  overflow-x: hidden;
}
/* Force ALL Shiki output to behave as a static, unstyled flow that the
   transform on .cr-editor-highlight can position. Shiki ships <pre> with
   overflow:auto and a background color by default — neither plays well with
   our overlay technique, so we reset them aggressively. */
.cr-editor-highlight-inner pre,
.cr-editor-highlight-inner code {
  margin: 0 !important;
  padding: 0 !important;
  background: transparent !important;
  font-family: inherit !important;
  font-size: inherit !important;
  line-height: inherit !important;
  white-space: pre !important;
  tab-size: 2 !important;
  overflow: visible !important;
  max-height: none !important;
  height: auto !important;
  min-height: 0 !important;
  min-width: 0 !important;
  display: block;
}
/* Shiki emits each line as <span class="line"> separated by \n. With
   white-space: pre the \n becomes the actual line break — keep spans
   `inline` (default) to avoid doubling the row height. Empty lines still
   reserve a full row thanks to `line-height: 21px` on the parent. */

/* transparent textarea above */
.cr-editor-textarea {
  position: relative;
  z-index: 1;
  width: 100%;
  height: 100%;
  padding: 12px 16px;
  font-family: ui-monospace, "JetBrains Mono", "SF Mono", Menlo, Consolas, monospace;
  font-size: 13px;
  line-height: 21px;
  color: transparent;
  caret-color: var(--cr-text);
  background: transparent;
  border: none;
  resize: none;
  outline: none;
  white-space: pre;
  overflow: auto;
  tab-size: 2;
}
.cr-editor-textarea::placeholder {
  color: var(--cr-text-soft);
}
.cr-editor-textarea::selection {
  background: rgb(159 232 112 / 0.30);
}

/* Soft scrollbars */
.cr-editor-textarea::-webkit-scrollbar { width: 10px; height: 10px; }
.cr-editor-textarea::-webkit-scrollbar-thumb { background: var(--cr-border-strong); border-radius: 6px; }
.cr-editor-textarea::-webkit-scrollbar-thumb:hover { background: var(--cr-text-soft); }
</style>
