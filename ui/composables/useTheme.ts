// Light/dark theme state, persisted to a cookie so SSR renders the right
// `html.dark` class from the very first byte. Avoids the white-flash-then-dark
// FOUC that localStorage-based approaches suffer.
//
// The composable also registers a `useHead` call so the rendered HTML root
// gets the correct class attribute during SSR.

const COOKIE = 'cr-theme'
export type ThemeMode = 'light' | 'dark'

export function useTheme() {
  const mode = useCookie<ThemeMode>(COOKIE, {
    default: () => 'light',
    sameSite: 'lax',
    maxAge: 60 * 60 * 24 * 365,
    path: '/',
  })

  // Bind to <html class> via useHead so SSR emits the right class.
  useHead({
    htmlAttrs: {
      class: () => (mode.value === 'dark' ? 'dark' : ''),
      style: () => (mode.value === 'dark' ? 'color-scheme: dark' : 'color-scheme: light'),
    },
  })

  // Avoid the cascading transition on theme swap (Emil rule):
  // toggle a `data-theme-switching` attribute for 1 frame so the CSS layer
  // disables `transition` for that instant.
  function suppressTransitions() {
    if (typeof document === 'undefined') return
    document.documentElement.setAttribute('data-theme-switching', '')
    requestAnimationFrame(() => {
      document.documentElement.removeAttribute('data-theme-switching')
    })
  }

  function toggle() {
    suppressTransitions()
    mode.value = mode.value === 'dark' ? 'light' : 'dark'
  }
  function set(next: ThemeMode) {
    if (next === mode.value) return
    suppressTransitions()
    mode.value = next
  }

  return { mode, toggle, set }
}
