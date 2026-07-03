// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2024-11-01',
  devtools: { enabled: true },

  modules: [
    '@nuxt/ui',          // brings Tailwind v4 + Reka primitives + UApp + UIcon
    '@vueuse/motion/nuxt',
    '@vueuse/nuxt',
    '@pinia/nuxt',
  ],

  css: ['~/assets/css/main.css'],

  // Icons (@nuxt/icon, pulled in by @nuxt/ui). By default icons are resolved
  // at runtime from the remote Iconify API (api.iconify.design) — on a LAN
  // server with slow/blocked outbound that means icons only appear after a
  // reload (once cached). Bundling the `lucide` set locally makes the Nuxt
  // server serve + SSR-render every icon from the installed
  // `@iconify-json/lucide` package: no internet, no flash, works for dynamic
  // `:name` bindings too.
  icon: {
    serverBundle: {
      collections: ['lucide'],
    },
    // Default is /api/_nuxt_icon, but behind Caddy the /api/* prefix is routed
    // to the Go backend — so the icon endpoint must live OUTSIDE /api. Caddy's
    // catch-all then proxies /_nuxt_icon/* to the Nuxt server.
    localApiEndpoint: '/_nuxt_icon',
    // Same-origin endpoint only (through Caddy), never the public Iconify API.
    fallbackToApi: false,
  },

  app: {
    head: {
      title: 'Cloud-Report',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Cloud-Report — design and render beautiful reports.' },
      ],
      link: [
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
        // Preconnect for the small webfont bundle.
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&display=swap',
        },
      ],
    },
    // No page transition: `out-in` forced the old page to animate OUT before
    // the new one mounted, adding dead-time on every click. With it off, the
    // new page (and its skeleton) mounts INSTANTLY; the per-page
    // `.cr-anim-fade-up` gives a single quick entrance fade.
    pageTransition: false,
    layoutTransition: false,
  },

  runtimeConfig: {
    public: {
      // `??` (not `||`) so an explicit empty string means "use relative URLs"
      // — that's the mode when the app runs behind the Caddy reverse proxy
      // (the browser hits /api, /odata… on the same origin). Only an *unset*
      // var falls back to the dev default.
      apiBase: process.env.NUXT_PUBLIC_API_BASE ?? 'http://10.71.1.125:5488',
    },
  },

  devServer: {
    host: '0.0.0.0',
    port: 3030,
  },
})
