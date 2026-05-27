<script setup lang="ts">
// In-app API documentation. Layout mirrors how modern API docs sites work
// (Stripe / Resend / Vercel): sticky left section nav, scrollable content,
// copy-able code samples, live base-URL substitution so examples are
// runnable against this exact deployment.
const { t } = useI18n()
const cfg = useRuntimeConfig()

// The real base URL of this deployment — examples interpolate it so users can
// copy/paste runnable curl commands.
const base = computed(() => cfg.public.apiBase || 'http://localhost:5488')

interface Section { id: string; label: string; icon: string }
const sections: Section[] = [
  { id: 'intro',      label: t('docs.s.intro'),      icon: 'i-lucide-book-open' },
  { id: 'auth',       label: t('docs.s.auth'),       icon: 'i-lucide-key-round' },
  { id: 'render',     label: t('docs.s.render'),     icon: 'i-lucide-play' },
  { id: 'templates',  label: t('docs.s.templates'),  icon: 'i-lucide-file-text' },
  { id: 'engines',    label: t('docs.s.engines'),    icon: 'i-lucide-braces' },
  { id: 'recipes',    label: t('docs.s.recipes'),    icon: 'i-lucide-layers' },
  { id: 'assets',     label: t('docs.s.assets'),     icon: 'i-lucide-image' },
  { id: 'pdfops',     label: t('docs.s.pdfops'),     icon: 'i-lucide-file-stack' },
  { id: 'errors',     label: t('docs.s.errors'),     icon: 'i-lucide-triangle-alert' },
]

const active = ref('intro')

// Scroll-spy: highlight the section currently in view.
let io: IntersectionObserver | null = null
onMounted(() => {
  io = new IntersectionObserver(
    (entries) => {
      for (const e of entries) {
        if (e.isIntersecting) active.value = e.target.id
      }
    },
    { rootMargin: '-20% 0px -70% 0px', threshold: 0 },
  )
  for (const s of sections) {
    const el = document.getElementById(s.id)
    if (el) io.observe(el)
  }
})
onBeforeUnmount(() => io?.disconnect())

function jump(id: string) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

// ── Code samples (computed so the live base URL is interpolated) ──────────
const curlRender = computed(() => `curl -X POST ${base.value}/api/report \\
  -H "Authorization: Bearer <API_KEY>" \\
  -H "Content-Type: application/json" \\
  -d '{
    "template": { "shortid": "<TEMPLATE_SHORTID>" },
    "data": { "name": "Mundo" }
  }' \\
  --output reporte.pdf`)

const curlLogin = computed(() => `curl -X POST ${base.value}/api/auth/login \\
  -H "Content-Type: application/json" \\
  -d '{ "username": "admin", "password": "tu-password" }'`)

const jsRender = computed(() => `const res = await fetch("${base.value}/api/report", {
  method: "POST",
  headers: {
    "Authorization": \`Bearer \${apiKey}\`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    template: { shortid: "TEMPLATE_SHORTID" },
    data: { name: "Mundo", items: [{ sku: "A1", qty: 3 }] },
  }),
})
const pdf = await res.blob()  // application/pdf`)

const inlineTemplate = computed(() => `curl -X POST ${base.value}/api/report \\
  -H "Authorization: Bearer <API_KEY>" \\
  -H "Content-Type: application/json" \\
  -d '{
    "template": {
      "content": "<h1>Hola {{name}}</h1>",
      "engine": "handlebars",
      "recipe": "chrome-pdf"
    },
    "data": { "name": "Ana" }
  }' --output out.pdf`)

const assetExample = `<!-- Inlinar CSS guardado como asset -->
<style>{#asset styles.css}</style>

<!-- Imagen como data URI (no requiere servidor de archivos) -->
<img src="{#asset logo.png @encoding=dataURI}" />`

const pdfOpsExample = `{
  "template": { "shortid": "TPL" },
  "data": {},
  "options": {
    "pdfOperations": [
      { "type": "merge", "templateShortid": "HEADER_TPL", "renderForEveryPage": true },
      { "type": "prepend", "templateShortid": "PORTADA_TPL" }
    ]
  }
}`

const errorExample = `{
  "error": "pdfOperations: pdf op stamp: ..."
}`
</script>

<template>
  <div class="cr-anim-fade-up">
    <PageHeader
      icon="i-lucide-book-open"
      :title="t('docs.title')"
      :description="t('docs.description')"
    >
      <template #actions>
        <a :href="`${base}/api/recipe`" target="_blank" class="cr-btn-secondary">
          <UIcon name="i-lucide-external-link" class="w-4 h-4" />
          <span>{{ t('docs.openApi') }}</span>
        </a>
      </template>
    </PageHeader>

    <div class="cr-docs-grid">
      <!-- Sticky section nav -->
      <aside class="cr-docs-nav">
        <p class="cr-docs-nav-title">{{ t('docs.onThisPage') }}</p>
        <ul>
          <li v-for="s in sections" :key="s.id">
            <button
              class="cr-docs-nav-item"
              :class="active === s.id ? 'cr-docs-nav-item--active' : ''"
              @click="jump(s.id)"
            >
              <UIcon :name="s.icon" class="w-3.5 h-3.5 shrink-0" />
              <span>{{ s.label }}</span>
            </button>
          </li>
        </ul>
        <div class="cr-docs-base">
          <p class="cr-docs-base-label">Base URL</p>
          <code class="cr-docs-base-url">{{ base }}</code>
        </div>
      </aside>

      <!-- Content -->
      <div class="cr-docs-content">
        <!-- INTRO -->
        <section id="intro" class="cr-docs-section">
          <h2>{{ t('docs.intro.h') }}</h2>
          <p>{{ t('docs.intro.p1') }}</p>
          <p>{{ t('docs.intro.p2') }}</p>
          <div class="cr-docs-steps">
            <div class="cr-docs-step"><span>1</span>{{ t('docs.intro.step1') }}</div>
            <div class="cr-docs-step"><span>2</span>{{ t('docs.intro.step2') }}</div>
            <div class="cr-docs-step"><span>3</span>{{ t('docs.intro.step3') }}</div>
          </div>
        </section>

        <!-- AUTH -->
        <section id="auth" class="cr-docs-section">
          <h2>{{ t('docs.auth.h') }}</h2>
          <p>{{ t('docs.auth.p1') }}</p>
          <ul class="cr-docs-list">
            <li><strong>API Key</strong> — {{ t('docs.auth.apikey') }} <code>Authorization: Bearer cr_...</code></li>
            <li><strong>JWT</strong> — {{ t('docs.auth.jwt') }}</li>
          </ul>
          <p>{{ t('docs.auth.p2') }}</p>
          <CodeBlock :code="curlLogin" language="bash" title="POST /api/auth/login" />
          <p class="cr-docs-note">
            <UIcon name="i-lucide-info" class="w-4 h-4 shrink-0" />
            {{ t('docs.auth.keyTip') }}
            <NuxtLink to="/settings/api-keys" class="cr-docs-link">/settings/api-keys</NuxtLink>
          </p>
        </section>

        <!-- RENDER -->
        <section id="render" class="cr-docs-section">
          <h2>{{ t('docs.render.h') }}</h2>
          <p>{{ t('docs.render.p1') }}</p>
          <CodeBlock :code="curlRender" language="bash" title="POST /api/report" />
          <p>{{ t('docs.render.p2') }}</p>
          <CodeBlock :code="jsRender" language="javascript" title="fetch (JS)" />
          <p>{{ t('docs.render.inline') }}</p>
          <CodeBlock :code="inlineTemplate" language="bash" title="Inline template" />
        </section>

        <!-- TEMPLATES -->
        <section id="templates" class="cr-docs-section">
          <h2>{{ t('docs.templates.h') }}</h2>
          <p>{{ t('docs.templates.p1') }}</p>
          <table class="cr-docs-table">
            <thead>
              <tr><th>{{ t('docs.field') }}</th><th>{{ t('docs.type') }}</th><th>{{ t('docs.desc') }}</th></tr>
            </thead>
            <tbody>
              <tr><td><code>content</code></td><td>string</td><td>{{ t('docs.templates.content') }}</td></tr>
              <tr><td><code>engine</code></td><td>string</td><td>handlebars · jsrender</td></tr>
              <tr><td><code>recipe</code></td><td>string</td><td>chrome-pdf · weasyprint · docx · xlsx · pptx · html</td></tr>
              <tr><td><code>helpers</code></td><td>string</td><td>{{ t('docs.templates.helpers') }}</td></tr>
              <tr><td><code>data</code></td><td>object</td><td>{{ t('docs.templates.data') }}</td></tr>
            </tbody>
          </table>
        </section>

        <!-- ENGINES -->
        <section id="engines" class="cr-docs-section">
          <h2>{{ t('docs.engines.h') }}</h2>
          <p>{{ t('docs.engines.p1') }}</p>
          <CodeBlock code="<h1>Hola {{name}}</h1>
<ul>
  {{#each items}}
  <li>{{this.sku}} — {{this.qty}} u.</li>
  {{/each}}
</ul>" language="handlebars" title="Handlebars" />
        </section>

        <!-- RECIPES -->
        <section id="recipes" class="cr-docs-section">
          <h2>{{ t('docs.recipes.h') }}</h2>
          <p>{{ t('docs.recipes.p1') }}</p>
          <div class="cr-docs-cards">
            <div class="cr-docs-card"><RecipePill recipe="chrome-pdf" /><p>{{ t('docs.recipes.chrome') }}</p></div>
            <div class="cr-docs-card"><RecipePill recipe="weasyprint" /><p>{{ t('docs.recipes.weasy') }}</p></div>
            <div class="cr-docs-card"><RecipePill recipe="docx" /><p>{{ t('docs.recipes.docx') }}</p></div>
            <div class="cr-docs-card"><RecipePill recipe="xlsx" /><p>{{ t('docs.recipes.xlsx') }}</p></div>
            <div class="cr-docs-card"><RecipePill recipe="pptx" /><p>{{ t('docs.recipes.pptx') }}</p></div>
            <div class="cr-docs-card"><RecipePill recipe="html" /><p>{{ t('docs.recipes.html') }}</p></div>
          </div>
        </section>

        <!-- ASSETS -->
        <section id="assets" class="cr-docs-section">
          <h2>{{ t('docs.assets.h') }}</h2>
          <p>{{ t('docs.assets.p1') }}</p>
          <CodeBlock :code="assetExample" language="html" title="{#asset ...}" />
          <table class="cr-docs-table">
            <thead><tr><th>encoding</th><th>{{ t('docs.desc') }}</th></tr></thead>
            <tbody>
              <tr><td><code>utf8</code></td><td>{{ t('docs.assets.utf8') }}</td></tr>
              <tr><td><code>base64</code></td><td>{{ t('docs.assets.base64') }}</td></tr>
              <tr><td><code>dataURI</code></td><td>{{ t('docs.assets.datauri') }}</td></tr>
              <tr><td><code>link</code></td><td>{{ t('docs.assets.link') }}</td></tr>
            </tbody>
          </table>
        </section>

        <!-- PDF OPS -->
        <section id="pdfops" class="cr-docs-section">
          <h2>{{ t('docs.pdfops.h') }}</h2>
          <p>{{ t('docs.pdfops.p1') }}</p>
          <ul class="cr-docs-list">
            <li><code>append</code> — {{ t('docs.pdfops.append') }}</li>
            <li><code>prepend</code> — {{ t('docs.pdfops.prepend') }}</li>
            <li><code>merge</code> + <code>renderForEveryPage</code> — {{ t('docs.pdfops.merge') }}</li>
          </ul>
          <CodeBlock :code="pdfOpsExample" language="json" title="pdfOperations" />
        </section>

        <!-- ERRORS -->
        <section id="errors" class="cr-docs-section">
          <h2>{{ t('docs.errors.h') }}</h2>
          <p>{{ t('docs.errors.p1') }}</p>
          <CodeBlock :code="errorExample" language="json" title="Error 500" />
          <table class="cr-docs-table">
            <thead><tr><th>{{ t('docs.code') }}</th><th>{{ t('docs.desc') }}</th></tr></thead>
            <tbody>
              <tr><td><code>401</code></td><td>{{ t('docs.errors.e401') }}</td></tr>
              <tr><td><code>403</code></td><td>{{ t('docs.errors.e403') }}</td></tr>
              <tr><td><code>404</code></td><td>{{ t('docs.errors.e404') }}</td></tr>
              <tr><td><code>500</code></td><td>{{ t('docs.errors.e500') }}</td></tr>
            </tbody>
          </table>
        </section>
      </div>
    </div>
  </div>
</template>

<style>
.cr-docs-grid {
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 32px;
  align-items: start;
}
@media (max-width: 900px) {
  .cr-docs-grid { grid-template-columns: 1fr; }
  .cr-docs-nav { display: none; }
}

/* Sticky nav */
.cr-docs-nav {
  position: sticky;
  top: 16px;
}
.cr-docs-nav-title {
  font-size: 10.5px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--cr-text-soft);
  margin-bottom: 10px;
}
.cr-docs-nav ul { display: flex; flex-direction: column; gap: 1px; }
.cr-docs-nav-item {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 7px 10px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  color: var(--cr-text-muted);
  cursor: pointer;
  text-align: left;
  border-left: 2px solid transparent;
  transition: background-color 140ms, color 140ms, border-color 140ms;
}
.cr-docs-nav-item:hover { background: var(--cr-surface-soft); color: var(--cr-text); }
.cr-docs-nav-item--active {
  color: var(--cr-text);
  background: var(--cr-surface-soft);
  border-left-color: var(--color-wise-500);
  font-weight: 600;
}
.cr-docs-base {
  margin-top: 18px;
  padding: 12px;
  border-radius: 10px;
  background: var(--cr-surface-soft);
  border: 1px solid var(--cr-border);
}
.cr-docs-base-label {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--cr-text-soft);
  margin-bottom: 4px;
}
.cr-docs-base-url {
  font-size: 11.5px;
  font-family: ui-monospace, monospace;
  color: var(--color-wise-700);
  word-break: break-all;
}
html.dark .cr-docs-base-url { color: var(--color-wise-300); }

/* Content */
.cr-docs-content { max-width: 760px; min-width: 0; }
.cr-docs-section {
  padding-bottom: 36px;
  margin-bottom: 36px;
  border-bottom: 1px solid var(--cr-border);
  scroll-margin-top: 16px;
}
.cr-docs-section:last-child { border-bottom: none; }
.cr-docs-section h2 {
  font-size: 24px;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--cr-text);
  margin-bottom: 12px;
}
.cr-docs-section p {
  font-size: 14.5px;
  line-height: 1.7;
  color: var(--cr-text-muted);
  margin-bottom: 12px;
}
.cr-docs-section code {
  font-family: ui-monospace, monospace;
  font-size: 0.88em;
  padding: 1px 6px;
  border-radius: 5px;
  background: var(--cr-surface-soft);
  border: 1px solid var(--cr-border);
  color: var(--cr-text);
}
.cr-docs-list {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 4px 0 14px;
}
.cr-docs-list li {
  font-size: 14px;
  line-height: 1.6;
  color: var(--cr-text-muted);
  padding-left: 18px;
  position: relative;
}
.cr-docs-list li::before {
  content: "";
  position: absolute;
  left: 2px; top: 9px;
  width: 6px; height: 6px;
  border-radius: 999px;
  background: var(--color-wise-500);
}
.cr-docs-note {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px 14px;
  border-radius: 10px;
  background: rgb(56 200 255 / 0.08);
  border: 1px solid rgb(56 200 255 / 0.25);
  font-size: 13px !important;
  color: var(--cr-text-muted) !important;
  margin-top: 12px;
}
.cr-docs-link { color: var(--color-wise-700); font-weight: 600; text-decoration: underline; }
html.dark .cr-docs-link { color: var(--color-wise-300); }

.cr-docs-steps { display: flex; flex-direction: column; gap: 10px; margin-top: 14px; }
.cr-docs-step {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 10px;
  background: var(--cr-surface-soft);
  border: 1px solid var(--cr-border);
  font-size: 14px;
  color: var(--cr-text);
}
.cr-docs-step span {
  width: 24px; height: 24px;
  border-radius: 999px;
  background: var(--color-wise-400);
  color: #0e0f0c;
  font-weight: 800;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.cr-docs-table {
  width: 100%;
  border-collapse: collapse;
  margin: 8px 0 14px;
  font-size: 13.5px;
  border: 1px solid var(--cr-border);
  border-radius: 10px;
  overflow: hidden;
}
.cr-docs-table th {
  text-align: left;
  padding: 9px 12px;
  background: var(--cr-surface-soft);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--cr-text-muted);
  border-bottom: 1px solid var(--cr-border);
}
.cr-docs-table td {
  padding: 9px 12px;
  border-bottom: 1px solid var(--cr-border);
  color: var(--cr-text-muted);
  vertical-align: top;
}
.cr-docs-table tr:last-child td { border-bottom: none; }

.cr-docs-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 10px;
  margin-top: 6px;
}
.cr-docs-card {
  padding: 14px;
  border-radius: 10px;
  border: 1px solid var(--cr-border);
  background: var(--cr-surface);
}
.cr-docs-card p {
  font-size: 12.5px !important;
  margin: 8px 0 0 !important;
  line-height: 1.5 !important;
}
</style>
