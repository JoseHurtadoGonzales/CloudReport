<script setup lang="ts">
interface Props {
  state: string
}
const props = defineProps<Props>()

const { t } = useI18n()

// Map raw state value → visual variant + icon. Localized label falls back to
// the raw state if the dictionary key is missing.
const map: Record<string, { variant: string; icon: string }> = {
  success:    { variant: 'success', icon: 'i-lucide-circle-check' },
  error:      { variant: 'error',   icon: 'i-lucide-circle-x' },
  running:    { variant: 'running', icon: 'i-lucide-loader-2' },
  queued:     { variant: 'queued',  icon: 'i-lucide-clock' },
  canceling:  { variant: 'warn',    icon: 'i-lucide-circle-pause' },
  planned:    { variant: 'queued',  icon: 'i-lucide-calendar' },
}

const meta = computed(() => map[props.state] ?? { variant: 'queued', icon: 'i-lucide-circle' })

const label = computed(() => {
  const key = `status.${props.state}`
  const translated = t(key)
  // If no translation exists, t() returns the key — fall back to raw state.
  return translated === key ? props.state : translated
})
</script>

<template>
  <span
    class="cr-status-badge"
    :class="`cr-status-badge--${meta.variant}`"
  >
    <UIcon
      :name="meta.icon"
      class="w-3 h-3 shrink-0"
      :class="state === 'running' ? 'cr-anim-spin' : ''"
    />
    {{ label }}
  </span>
</template>

<style>
/* StatusBadge — readable in both themes. The previous implementation used
   dark fg colours (#054d28, #0367a4) on faint tints which were invisible
   against the dark app shell. Each variant now ships a light + dark colour
   pair plus a tinted border so the chip reads as a discrete token. */
.cr-status-badge {
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
  line-height: 1.3;
  text-transform: capitalize;
}

/* success — green */
.cr-status-badge--success {
  background: rgb(46 173 75 / 0.14);
  border-color: rgb(46 173 75 / 0.40);
  color: #1a6b34;
}
html.dark .cr-status-badge--success {
  background: rgb(46 173 75 / 0.20);
  border-color: rgb(46 173 75 / 0.55);
  color: #6ee49a;
}

/* error — red */
.cr-status-badge--error {
  background: rgb(208 50 56 / 0.12);
  border-color: rgb(208 50 56 / 0.45);
  color: #a72027;
}
html.dark .cr-status-badge--error {
  background: rgb(208 50 56 / 0.22);
  border-color: rgb(208 50 56 / 0.55);
  color: #ff8a92;
}

/* running — cyan, animated icon spins */
.cr-status-badge--running {
  background: rgb(56 200 255 / 0.18);
  border-color: rgb(56 200 255 / 0.45);
  color: #0a5b85;
}
html.dark .cr-status-badge--running {
  background: rgb(56 200 255 / 0.18);
  border-color: rgb(56 200 255 / 0.55);
  color: #7ed8ff;
}

/* warn — amber/orange (canceling, etc.) */
.cr-status-badge--warn {
  background: rgb(255 165 100 / 0.24);
  border-color: rgb(255 165 100 / 0.50);
  color: #9a3a07;
}
html.dark .cr-status-badge--warn {
  background: rgb(255 165 100 / 0.20);
  border-color: rgb(255 165 100 / 0.55);
  color: #ffc091;
}

/* queued / planned — neutral grey, lowest visual weight */
.cr-status-badge--queued {
  background: rgb(134 134 133 / 0.14);
  border-color: rgb(134 134 133 / 0.35);
  color: #4a4d4b;
}
html.dark .cr-status-badge--queued {
  background: rgb(255 255 255 / 0.06);
  border-color: rgb(255 255 255 / 0.16);
  color: #c8ccc7;
}
</style>
