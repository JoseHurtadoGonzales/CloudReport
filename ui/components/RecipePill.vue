<script setup lang="ts">
interface Props {
  recipe: string
  /** Solid, opaque variant — for placing over coloured / image backgrounds
   *  (e.g. the template thumbnails) where the translucent tint is unreadable. */
  solid?: boolean
}
const props = defineProps<Props>()

const recipeMeta: Record<string, { icon: string; color: string }> = {
  'html':         { icon: 'i-lucide-code-xml',     color: 'wise' },
  'text':         { icon: 'i-lucide-type',         color: 'mute' },
  'xlsx':         { icon: 'i-lucide-sheet',        color: 'success' },
  'static-pdf':   { icon: 'i-lucide-file-text',    color: 'slate' },
  'chrome-pdf':   { icon: 'i-lucide-chrome',       color: 'indigo' },
  'weasyprint':   { icon: 'i-lucide-file-text',    color: 'violet' },
  'docx':         { icon: 'i-lucide-file-text',    color: 'cyan' },
  'pptx':         { icon: 'i-lucide-presentation', color: 'orange' },
  'html-to-xlsx': { icon: 'i-lucide-table-2',      color: 'pink' },
}

const meta = computed(() => recipeMeta[props.recipe] ?? { icon: 'i-lucide-box', color: 'mute' })
</script>

<template>
  <span
    class="cr-recipe-pill"
    :class="[`cr-recipe-pill--${meta.color}`, { 'cr-recipe-pill--solid': solid }]"
  >
    <UIcon :name="meta.icon" class="w-3 h-3 shrink-0" />
    {{ recipe }}
  </span>
</template>

<style>
/* Recipe pill base — readable size, rounded, subtle border that lifts the
   chip off the surface in both themes (the old "tint-only" version was
   nearly invisible against dark backgrounds because the foreground colours
   were near-black). Each variant overrides background / border / text. */
.cr-recipe-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 11.5px;
  font-weight: 700;
  letter-spacing: 0.01em;
  white-space: nowrap;
  border: 1px solid transparent;
  background: var(--cr-surface-soft);
  color: var(--cr-text);
  line-height: 1.3;
}

/* ── Color variants ──────────────────────────────────────────────────
   Light mode: deep-tinted text on pale-tinted background (high contrast
   against the off-white app shell).
   Dark mode: bright accent text on a slightly stronger transparent tint,
   plus a tinted 1-px border so each chip reads as a distinct token. */

/* wise (lime accent — chrome-pdf, html) */
.cr-recipe-pill--wise {
  background: rgb(159 232 112 / 0.18);
  border-color: rgb(159 232 112 / 0.45);
  color: #1f5b15;
}
html.dark .cr-recipe-pill--wise {
  background: rgb(159 232 112 / 0.16);
  border-color: rgb(159 232 112 / 0.50);
  color: #cdffad;
}

/* success (green — xlsx) */
.cr-recipe-pill--success {
  background: rgb(46 173 75 / 0.14);
  border-color: rgb(46 173 75 / 0.40);
  color: #186b34;
}
html.dark .cr-recipe-pill--success {
  background: rgb(46 173 75 / 0.18);
  border-color: rgb(46 173 75 / 0.55);
  color: #6ee49a;
}

/* cyan (docx) */
.cr-recipe-pill--cyan {
  background: rgb(56 200 255 / 0.18);
  border-color: rgb(56 200 255 / 0.45);
  color: #0a5b85;
}
html.dark .cr-recipe-pill--cyan {
  background: rgb(56 200 255 / 0.18);
  border-color: rgb(56 200 255 / 0.55);
  color: #7ed8ff;
}

/* orange (pptx) */
.cr-recipe-pill--orange {
  background: rgb(255 165 100 / 0.24);
  border-color: rgb(255 165 100 / 0.50);
  color: #9a3a07;
}
html.dark .cr-recipe-pill--orange {
  background: rgb(255 165 100 / 0.20);
  border-color: rgb(255 165 100 / 0.55);
  color: #ffc091;
}

/* violet (weasyprint — distinct from chrome-pdf so users can tell engines apart) */
.cr-recipe-pill--violet {
  background: rgb(168 134 255 / 0.20);
  border-color: rgb(168 134 255 / 0.50);
  color: #5933a8;
}
html.dark .cr-recipe-pill--violet {
  background: rgb(168 134 255 / 0.20);
  border-color: rgb(168 134 255 / 0.55);
  color: #d2bdff;
}

/* indigo (chrome-pdf — own colour, distinct from html's lime) */
.cr-recipe-pill--indigo {
  background: rgb(99 102 241 / 0.16);
  border-color: rgb(99 102 241 / 0.45);
  color: #3730a3;
}
html.dark .cr-recipe-pill--indigo {
  background: rgb(129 140 248 / 0.18);
  border-color: rgb(129 140 248 / 0.55);
  color: #c7cbff;
}

/* pink (html-to-xlsx) */
.cr-recipe-pill--pink {
  background: rgb(255 138 184 / 0.22);
  border-color: rgb(255 138 184 / 0.50);
  color: #963058;
}
html.dark .cr-recipe-pill--pink {
  background: rgb(255 138 184 / 0.20);
  border-color: rgb(255 138 184 / 0.55);
  color: #ffb8d2;
}

/* slate (static-pdf — neutral, dense) */
.cr-recipe-pill--slate {
  background: rgb(100 116 139 / 0.16);
  border-color: rgb(100 116 139 / 0.40);
  color: #334155;
}
html.dark .cr-recipe-pill--slate {
  background: rgb(148 163 184 / 0.16);
  border-color: rgb(148 163 184 / 0.40);
  color: #cbd5e1;
}

/* mute (text — neutral, lowest visual weight) */
.cr-recipe-pill--mute {
  background: rgb(134 134 133 / 0.14);
  border-color: rgb(134 134 133 / 0.35);
  color: #4a4d4b;
}
html.dark .cr-recipe-pill--mute {
  background: rgb(255 255 255 / 0.06);
  border-color: rgb(255 255 255 / 0.14);
  color: #c8ccc7;
}

/* ── Solid variant ───────────────────────────────────────────────────
   Opaque, fully-saturated chips for placing OVER the coloured thumbnail
   gradients, where the translucent tints above vanish. Each engine gets a
   distinct solid colour + a drop shadow so it reads at a glance and lifts
   off the gradient. Identical in light & dark (the gradient is the same in
   both), so no html.dark overrides are needed. */
.cr-recipe-pill--solid {
  border-color: transparent;
  box-shadow:
    0 2px 6px rgb(14 15 12 / 0.22),
    0 0 0 1px rgb(255 255 255 / 0.30) inset;
  backdrop-filter: none;
}
.cr-recipe-pill--solid.cr-recipe-pill--wise,
html.dark .cr-recipe-pill--solid.cr-recipe-pill--wise {
  background: #5fb52e; color: #ffffff;
}
.cr-recipe-pill--solid.cr-recipe-pill--success,
html.dark .cr-recipe-pill--solid.cr-recipe-pill--success {
  background: #1f9e47; color: #ffffff;
}
.cr-recipe-pill--solid.cr-recipe-pill--cyan,
html.dark .cr-recipe-pill--solid.cr-recipe-pill--cyan {
  background: #0a8fc4; color: #ffffff;
}
.cr-recipe-pill--solid.cr-recipe-pill--orange,
html.dark .cr-recipe-pill--solid.cr-recipe-pill--orange {
  background: #e0701c; color: #ffffff;
}
.cr-recipe-pill--solid.cr-recipe-pill--violet,
html.dark .cr-recipe-pill--solid.cr-recipe-pill--violet {
  background: #7c4dde; color: #ffffff;
}
.cr-recipe-pill--solid.cr-recipe-pill--pink,
html.dark .cr-recipe-pill--solid.cr-recipe-pill--pink {
  background: #d83f7e; color: #ffffff;
}
.cr-recipe-pill--solid.cr-recipe-pill--indigo,
html.dark .cr-recipe-pill--solid.cr-recipe-pill--indigo {
  background: #4f46e5; color: #ffffff;
}
.cr-recipe-pill--solid.cr-recipe-pill--slate,
html.dark .cr-recipe-pill--solid.cr-recipe-pill--slate {
  background: #475569; color: #ffffff;
}
.cr-recipe-pill--solid.cr-recipe-pill--mute,
html.dark .cr-recipe-pill--solid.cr-recipe-pill--mute {
  background: #57574f; color: #ffffff;
}
</style>
