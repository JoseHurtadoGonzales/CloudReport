<script setup lang="ts">
// Read-only code block for docs: Shiki highlighting when the language is one
// of the bundled grammars (html/css/js/json/handlebars), plain monospace
// otherwise (bash/curl/text). Includes a copy-to-clipboard button.
import { getHighlighter, normalizeLang, isDark } from '~/composables/useShiki'

interface Props {
  code: string
  language?: string
  /** Optional filename / label shown in the header bar. */
  title?: string
}
const props = withDefaults(defineProps<Props>(), {
  language: 'text',
  title: '',
})

// Shiki only ships these grammars; anything else renders as plain text.
const HIGHLIGHTABLE = ['html', 'css', 'javascript', 'js', 'ts', 'typescript', 'json', 'handlebars', 'hbs']
const canHighlight = computed(() => HIGHLIGHTABLE.includes(props.language))

const html = ref('')
const dark = ref(false)
const copied = ref(false)

async function render() {
  if (!canHighlight.value) { html.value = ''; return }
  try {
    const hl = await getHighlighter()
    const out = hl.codeToHtml(props.code, {
      lang: normalizeLang(props.language),
      theme: dark.value ? 'github-dark' : 'github-light',
    })
    html.value = out.replace(/<pre[^>]*>/, '<pre>').replace(/<code[^>]*>/, '<code>')
  } catch {
    html.value = ''
  }
}

async function copy() {
  try {
    await navigator.clipboard.writeText(props.code)
    copied.value = true
    setTimeout(() => (copied.value = false), 1600)
  } catch { /* clipboard blocked — no-op */ }
}

let observer: MutationObserver | null = null
onMounted(async () => {
  dark.value = isDark()
  await render()
  observer = new MutationObserver(() => {
    const d = isDark()
    if (d !== dark.value) { dark.value = d; render() }
  })
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
})
onBeforeUnmount(() => observer?.disconnect())
watch(() => [props.code, props.language], render)
</script>

<template>
  <div class="cr-codeblock">
    <div class="cr-codeblock-head">
      <span class="cr-codeblock-lang">{{ title || language }}</span>
      <button type="button" class="cr-codeblock-copy" @click="copy">
        <UIcon :name="copied ? 'i-lucide-check' : 'i-lucide-copy'" class="w-3.5 h-3.5" />
        <span>{{ copied ? 'Copiado' : 'Copiar' }}</span>
      </button>
    </div>
    <div class="cr-codeblock-body">
      <div v-if="canHighlight && html" class="cr-codeblock-shiki" v-html="html" />
      <pre v-else class="cr-codeblock-plain"><code>{{ code }}</code></pre>
    </div>
  </div>
</template>

<style>
.cr-codeblock {
  border: 1px solid var(--cr-border);
  border-radius: 12px;
  overflow: hidden;
  background: var(--cr-surface);
  margin: 14px 0;
}
.cr-codeblock-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 36px;
  padding: 0 12px;
  background: var(--cr-surface-soft);
  border-bottom: 1px solid var(--cr-border);
}
.cr-codeblock-lang {
  font-size: 10.5px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--cr-text-muted);
}
.cr-codeblock-copy {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 9px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 600;
  color: var(--cr-text-muted);
  background: var(--cr-surface);
  border: 1px solid var(--cr-border);
  cursor: pointer;
  transition: background-color 140ms, color 140ms, border-color 140ms;
}
.cr-codeblock-copy:hover { color: var(--cr-text); border-color: var(--cr-border-strong); }
.cr-codeblock-body { overflow-x: auto; }
.cr-codeblock-shiki pre,
.cr-codeblock-plain {
  margin: 0;
  padding: 14px 16px;
  font-family: ui-monospace, "JetBrains Mono", "SF Mono", Menlo, Consolas, monospace;
  font-size: 12.5px;
  line-height: 1.65;
  background: transparent !important;
  white-space: pre;
  tab-size: 2;
}
.cr-codeblock-plain {
  color: var(--cr-text);
}
.cr-codeblock-plain code { background: transparent; font: inherit; }
</style>
