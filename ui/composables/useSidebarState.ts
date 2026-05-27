// Sidebar collapsed state, persisted to a cookie so SSR can render the right
// width on the first byte (no FOUC / flash on reload).
//
// `useCookie` is auto-imported by Nuxt: it reads the incoming request on the
// server and document.cookie on the client. Same source of truth on both
// sides → no hydration mismatch.

const COOKIE = 'cr-sidebar-collapsed'

export function useSidebarState() {
  const collapsed = useCookie<boolean>(COOKIE, {
    default: () => false,
    sameSite: 'lax',
    maxAge: 60 * 60 * 24 * 365,
    path: '/',
  })

  return {
    collapsed,
    toggle:   () => { collapsed.value = !collapsed.value },
    expand:   () => { collapsed.value = false },
    collapse: () => { collapsed.value = true  },
  }
}
