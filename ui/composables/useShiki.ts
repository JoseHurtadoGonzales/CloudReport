// Lazy Shiki highlighter — singleton per page. Loads only the grammars we
// actually use to keep the bundle ~120KB instead of ~2MB.
import type { HighlighterCore } from 'shiki/core'

let highlighter: HighlighterCore | null = null
let loading: Promise<HighlighterCore> | null = null

const LANGS = ['html', 'css', 'javascript', 'json', 'handlebars'] as const

export type ShikiLang = (typeof LANGS)[number]

export function normalizeLang(input?: string): ShikiLang {
  switch (input) {
    case 'js':
    case 'javascript':
    case 'typescript':
    case 'ts':
      return 'javascript'
    case 'json':
      return 'json'
    case 'css':
    case 'scss':
      return 'css'
    case 'handlebars':
    case 'hbs':
      return 'handlebars'
    case 'html':
    default:
      return 'html'
  }
}

export async function getHighlighter(): Promise<HighlighterCore> {
  if (highlighter) return highlighter
  if (loading) return loading

  loading = (async () => {
    // Use the "core" build so we can pick only the langs we need (smaller bundle).
    const { createHighlighterCore } = await import('shiki/core')
    const { createOnigurumaEngine } = await import('shiki/engine/oniguruma')
    const wasm = await import('shiki/wasm')

    const langs = await Promise.all([
      import('shiki/langs/html.mjs').then(m => m.default),
      import('shiki/langs/css.mjs').then(m => m.default),
      import('shiki/langs/javascript.mjs').then(m => m.default),
      import('shiki/langs/json.mjs').then(m => m.default),
      import('shiki/langs/handlebars.mjs').then(m => m.default),
    ])

    highlighter = await createHighlighterCore({
      themes: [
        import('shiki/themes/github-light.mjs').then(m => m.default),
        import('shiki/themes/github-dark.mjs').then(m => m.default),
      ],
      langs,
      engine: createOnigurumaEngine(wasm),
    })

    return highlighter
  })()

  return loading
}

export function isDark(): boolean {
  if (typeof document === 'undefined') return false
  return document.documentElement.classList.contains('dark')
}
