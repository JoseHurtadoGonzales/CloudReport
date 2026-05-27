// Keeps the session "always alive". The JWT itself is long-lived (30 days by
// default, see SESSION_TTL_HOURS on the API), but we also slide it forward so
// an actively-used tab effectively never expires:
//
//   • on app boot — restore token from localStorage, fetch the user, refresh.
//   • every 30 min — silent refresh while the tab stays open.
//   • on window focus / tab becomes visible — refresh (covers laptop sleep,
//     long-idle tabs, etc.) but throttled so rapid focus changes don't spam.
//
// All refreshes are best-effort: a network blip leaves the existing token in
// place; only a hard 401 logs the user out.
import { useAuthStore } from '~/stores/auth'

export default defineNuxtPlugin(() => {
  const auth = useAuthStore()

  // Boot sequence.
  auth.hydrate()
  if (auth.token) {
    // Populate `user` (lost on hard reload) then slide the token forward.
    auth.fetchCurrentUser().then(() => auth.refresh())
  }

  // Periodic refresh — 30 min is comfortably below the 30-day TTL while
  // keeping the token fresh enough that a closed-and-reopened laptop the
  // next morning is still logged in.
  const THIRTY_MIN = 30 * 60 * 1000
  const interval = setInterval(() => {
    if (auth.token) auth.refresh()
  }, THIRTY_MIN)

  // Focus / visibility refresh, throttled to once per 5 min.
  let lastFocusRefresh = 0
  const onActive = () => {
    if (!auth.token) return
    const now = Date.now()
    if (now - lastFocusRefresh < 5 * 60 * 1000) return
    lastFocusRefresh = now
    auth.refresh()
  }
  window.addEventListener('focus', onActive)
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') onActive()
  })

  // Tidy up if the plugin context is ever torn down (HMR in dev).
  if (import.meta.hot) {
    import.meta.hot.dispose(() => {
      clearInterval(interval)
      window.removeEventListener('focus', onActive)
    })
  }
})
