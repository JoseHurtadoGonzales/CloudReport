<script setup lang="ts">
// Lightweight accordion section. Uses the native <details>/<summary> elements
// for accessibility (Enter/Space toggles, screen-reader friendly), with the
// `name=""` attribute so groups can be exclusive (one-open-at-a-time) when
// the caller wants.

interface Props {
  title: string
  icon?: string
  description?: string
  /** initial open state */
  open?: boolean
  /** sections that share the same `group` close each other (HTML name attr) */
  group?: string
  /** small badge on the right (e.g. count of active options) */
  badge?: string | number
  /** optional id (used by anchor jump-links) */
  anchorId?: string
}

defineProps<Props>()
</script>

<template>
  <details class="cr-collapsible" :open="open" :name="group" :id="anchorId">
    <summary class="cr-collapsible-summary">
      <span v-if="icon" class="cr-collapsible-icon">
        <UIcon :name="icon" class="w-4 h-4" />
      </span>
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2">
          <span class="cr-collapsible-title">{{ title }}</span>
          <span v-if="badge != null && badge !== ''" class="cr-collapsible-badge">{{ badge }}</span>
        </div>
        <p v-if="description" class="cr-collapsible-desc">{{ description }}</p>
      </div>
      <UIcon name="i-lucide-chevron-down" class="cr-collapsible-chevron w-4 h-4" />
    </summary>
    <div class="cr-collapsible-body">
      <slot />
    </div>
  </details>
</template>

<style>
.cr-collapsible {
  border: 1px solid var(--cr-border);
  border-radius: 14px;
  background: var(--cr-surface);
  overflow: hidden;
  transition:
    border-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    box-shadow 140ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-collapsible:hover { border-color: var(--cr-border-strong); }
.cr-collapsible[open] {
  border-color: var(--color-wise-500);
  box-shadow: 0 0 0 1px rgb(159 232 112 / 0.18);
}

.cr-collapsible-summary {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  cursor: pointer;
  list-style: none;
  user-select: none;
  transition: background-color 140ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-collapsible-summary::-webkit-details-marker { display: none; }
.cr-collapsible-summary:hover { background: var(--cr-surface-soft); }
.cr-collapsible[open] .cr-collapsible-summary {
  background: rgb(159 232 112 / 0.05);
  border-bottom: 1px solid var(--cr-border);
}
html.dark .cr-collapsible[open] .cr-collapsible-summary {
  background: rgb(159 232 112 / 0.04);
}

.cr-collapsible-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 9px;
  background: var(--color-wise-100);
  color: var(--color-wise-800);
  flex-shrink: 0;
  transition: transform 200ms cubic-bezier(0.34, 1.56, 0.64, 1),
              background-color 180ms cubic-bezier(0.23, 1, 0.32, 1),
              color 180ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-collapsible[open] .cr-collapsible-icon {
  background: var(--color-wise-400);
  color: #0e0f0c;
  transform: scale(1.05);
}
html.dark .cr-collapsible-icon {
  background: rgb(159 232 112 / 0.14);
  color: var(--color-wise-300);
}
html.dark .cr-collapsible[open] .cr-collapsible-icon {
  background: var(--color-wise-400);
  color: #0e0f0c;
}

.cr-collapsible-title {
  font-weight: 700;
  font-size: 14px;
  color: var(--cr-text);
  letter-spacing: -0.01em;
}
.cr-collapsible-desc {
  font-size: 12px;
  color: var(--cr-text-muted);
  margin-top: 3px;
  line-height: 1.4;
}
.cr-collapsible-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 7px;
  border-radius: 9999px;
  background: var(--color-wise-400);
  color: #0e0f0c;
  font-size: 10.5px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.08);
}

.cr-collapsible-chevron {
  color: var(--cr-text-soft);
  transition: transform 240ms cubic-bezier(0.34, 1.56, 0.64, 1),
              color 180ms cubic-bezier(0.23, 1, 0.32, 1);
  flex-shrink: 0;
}
.cr-collapsible[open] .cr-collapsible-chevron {
  transform: rotate(180deg);
  color: var(--color-wise-700);
}

.cr-collapsible-body {
  padding: 20px 20px 22px;
  background: var(--cr-surface);
}
.cr-collapsible-body > * + * { margin-top: 18px; }
</style>
