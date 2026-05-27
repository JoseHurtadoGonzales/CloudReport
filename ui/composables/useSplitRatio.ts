// Persisted left/right split ratio for the template editor. Stored as a cookie
// so the layout boots correctly on first paint (no flash).
//
// `ratio` is the % width of the LEFT pane (0..100). Defaults to 50.

export function useSplitRatio(cookieName = 'cr-split-template') {
  const ratio = useCookie<number>(cookieName, {
    default: () => 50,
    sameSite: 'lax',
    maxAge: 60 * 60 * 24 * 365,
    path: '/',
  })

  function set(value: number) {
    const clamped = Math.max(20, Math.min(80, Math.round(value)))
    ratio.value = clamped
  }
  function reset() {
    ratio.value = 50
  }

  return { ratio, set, reset }
}
