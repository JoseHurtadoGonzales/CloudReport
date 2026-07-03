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

  // Apply the theme class synchronously (so a View Transition snapshot is
  // captured with the right theme) AND update the ref (cookie + SSR class).
  function applyMode(next: ThemeMode) {
    if (typeof document !== 'undefined') {
      document.documentElement.classList.toggle('dark', next === 'dark')
    }
    mode.value = next
  }

  function prefersReducedMotion() {
    return typeof window !== 'undefined'
      && !!window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
  }

  // Switch theme with a circular "ripple" reveal expanding from the click point
  // (View Transitions API). Falls back to the instant swap when the browser has
  // no View Transitions or the user prefers reduced motion.
  function change(next: ThemeMode, event?: MouseEvent) {
    if (next === mode.value) return
    const doc = typeof document !== 'undefined' ? (document as any) : null

    if (!doc || prefersReducedMotion() || typeof doc.startViewTransition !== 'function') {
      suppressTransitions()
      applyMode(next)
      return
    }

    // Ripple origin: the pointer, else the top-right corner (where the toggle
    // lives). End radius = farthest viewport corner, so the circle covers all.
    const x = event?.clientX ?? window.innerWidth - 48
    const y = event?.clientY ?? 48
    const endRadius = Math.hypot(
      Math.max(x, window.innerWidth - x),
      Math.max(y, window.innerHeight - y),
    )

    // Keep the underlying live DOM from double-animating its colours; the
    // circular reveal (a static snapshot) is the only visible motion.
    suppressTransitions()
    const transition = doc.startViewTransition(() => { applyMode(next) })
    transition.ready.then(() => {
      document.documentElement.animate(
        {
          clipPath: [
            `circle(0px at ${x}px ${y}px)`,
            `circle(${endRadius}px at ${x}px ${y}px)`,
          ],
        },
        {
          // Even, clearly-visible circular sweep (ease-in-out reads as a smooth
          // wipe rather than an abrupt front-loaded snap).
          duration: 500,
          easing: 'ease-in-out',
          pseudoElement: '::view-transition-new(root)',
        },
      )
    }).catch(() => { /* transition skipped/interrupted — theme already applied */ })
  }

  function toggle(event?: MouseEvent) {
    change(mode.value === 'dark' ? 'light' : 'dark', event)
  }
  function set(next: ThemeMode, event?: MouseEvent) {
    change(next, event)
  }

  return { mode, toggle, set }
}
