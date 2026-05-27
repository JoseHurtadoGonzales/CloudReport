// Nuxt UI app config — semantic color slots
// (https://ui.nuxt.com/docs/getting-started/theme#colors)
export default defineAppConfig({
  ui: {
    colors: {
      primary: 'wise',     // mapped via @theme in main.css (wise-50…900)
      neutral: 'zinc',
    },
    // Default rounded-xl across the board to match Wise card chrome.
    button: { slots: { base: 'rounded-xl font-semibold' } },
    input:  { slots: { base: 'rounded-md' } },
    card:   { slots: { root: 'rounded-3xl' } },
  },
})
