<script setup lang="ts">
// Tiny ES/EN toggle. Click cycles between the two locales — no dropdown to
// keep it one-click and unambiguous. The currently active flag is opaque,
// the inactive one dimmed.
const { locale, setLocale, t } = useI18n()

function toggle() {
  setLocale(locale.value === 'es' ? 'en' : 'es')
}
</script>

<template>
  <button
    type="button"
    class="cr-lang-toggle"
    :title="t('common.language')"
    :aria-label="t('common.language')"
    @click="toggle"
  >
    <span class="cr-lang-pill" :class="{ 'cr-lang-pill--active': locale === 'es' }">ES</span>
    <span class="cr-lang-pill" :class="{ 'cr-lang-pill--active': locale === 'en' }">EN</span>
  </button>
</template>

<style>
.cr-lang-toggle {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 3px;
  border-radius: 999px;
  background: var(--cr-surface-soft);
  border: 1px solid var(--cr-border);
  cursor: pointer;
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    border-color 140ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-lang-toggle:hover { border-color: var(--cr-border-strong); }
.cr-lang-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 28px;
  height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 10.5px;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--cr-text-soft);
  transition:
    background-color 160ms cubic-bezier(0.23, 1, 0.32, 1),
    color 160ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-lang-pill--active {
  background: var(--color-wise-400);
  color: #0e0f0c;
}
html.dark .cr-lang-pill--active {
  background: var(--color-wise-400);
  color: #0e0f0c;
}
</style>
