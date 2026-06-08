<script setup lang="ts">
// Custom <select> replacement — gives us full control over the popup styling
// (light/dark theme, search, keyboard nav) which native <select> never grants
// cross-browser. Same external API as a controlled component: v-model + options.

interface Opt {
  value: string
  label: string
  /** Optional small text shown to the right of the label (e.g. recipe name). */
  hint?: string
  disabled?: boolean
}

interface Props {
  modelValue: string
  options: Opt[]
  placeholder?: string
  /** When provided, a search input is shown at the top of the popup. */
  searchable?: boolean
  /** Disable the whole control. */
  disabled?: boolean
  /** Extra classes applied to the trigger button (lets callers tweak size). */
  triggerClass?: string
  /** Where the popup opens. 'auto' flips upward when there's not enough room
   *  below (e.g. a select inside a scroll/overflow-hidden footer). */
  placement?: 'auto' | 'top' | 'bottom'
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: '— elegir —',
  searchable: false,
  disabled: false,
  triggerClass: '',
  placement: 'bottom',
})

const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const root = ref<HTMLElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)
const panel = ref<HTMLElement | null>(null)
const searchInput = ref<HTMLInputElement | null>(null)

const open = ref(false)
const query = ref('')
const activeIdx = ref(0)
const dropUp = ref(false)

const filtered = computed(() => {
  if (!props.searchable || !query.value.trim()) return props.options
  const q = query.value.toLowerCase()
  return props.options.filter(o =>
    o.label.toLowerCase().includes(q) ||
    (o.hint?.toLowerCase().includes(q) ?? false),
  )
})

const selected = computed(() =>
  props.options.find(o => o.value === props.modelValue) ?? null,
)

function toggleOpen() {
  if (props.disabled) return
  open.value ? close() : openPanel()
}

function openPanel() {
  open.value = true
  query.value = ''
  // Decide direction so the popup never gets clipped by an overflow-hidden
  // ancestor (e.g. the paginated table footer).
  if (props.placement === 'top') {
    dropUp.value = true
  } else if (props.placement === 'bottom') {
    dropUp.value = false
  } else {
    const rect = trigger.value?.getBoundingClientRect()
    if (rect) {
      const estimated = Math.min(320, props.options.length * 38 + (props.searchable ? 44 : 0) + 8)
      const below = window.innerHeight - rect.bottom
      dropUp.value = below < estimated && rect.top > below
    }
  }
  // Highlight the currently selected option.
  const idx = props.options.findIndex(o => o.value === props.modelValue)
  activeIdx.value = idx >= 0 ? idx : 0
  nextTick(() => {
    if (props.searchable) searchInput.value?.focus()
    else trigger.value?.focus()
    scrollActiveIntoView()
  })
}

function close() {
  open.value = false
}

function choose(opt: Opt) {
  if (opt.disabled) return
  emit('update:modelValue', opt.value)
  close()
  nextTick(() => trigger.value?.focus())
}

function onTriggerKey(e: KeyboardEvent) {
  if (props.disabled) return
  if (e.key === 'ArrowDown' || e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    openPanel()
  }
}

function onPanelKey(e: KeyboardEvent) {
  const list = filtered.value
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    activeIdx.value = Math.min(activeIdx.value + 1, list.length - 1)
    scrollActiveIntoView()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    activeIdx.value = Math.max(activeIdx.value - 1, 0)
    scrollActiveIntoView()
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const o = list[activeIdx.value]
    if (o) choose(o)
  } else if (e.key === 'Escape') {
    e.preventDefault()
    close()
  } else if (e.key === 'Tab') {
    close()
  }
}

function scrollActiveIntoView() {
  if (!panel.value) return
  const el = panel.value.querySelectorAll<HTMLElement>('[data-cr-opt]')[activeIdx.value]
  el?.scrollIntoView({ block: 'nearest' })
}

function onDocClick(e: MouseEvent) {
  if (!root.value) return
  if (!root.value.contains(e.target as Node)) close()
}

watch(open, (v) => {
  if (v) document.addEventListener('mousedown', onDocClick)
  else document.removeEventListener('mousedown', onDocClick)
})
onBeforeUnmount(() => document.removeEventListener('mousedown', onDocClick))

// Re-highlight active row as the query narrows the list.
watch(query, () => { activeIdx.value = 0 })
</script>

<template>
  <div ref="root" class="cr-select" :class="{ 'cr-select--open': open, 'cr-select--disabled': disabled }">
    <button
      ref="trigger"
      type="button"
      class="cr-select-trigger"
      :class="triggerClass"
      :disabled="disabled"
      :aria-expanded="open"
      aria-haspopup="listbox"
      @click="toggleOpen"
      @keydown="onTriggerKey"
    >
      <span class="cr-select-trigger-label" :class="{ 'cr-select-trigger-label--empty': !selected }">
        {{ selected?.label ?? placeholder }}
      </span>
      <UIcon
        name="i-lucide-chevron-down"
        class="w-4 h-4 shrink-0 cr-select-chevron"
        :class="{ 'cr-select-chevron--open': open }"
      />
    </button>

    <Transition
      enter-active-class="transition-all duration-150 ease-[cubic-bezier(0.23,1,0.32,1)]"
      leave-active-class="transition-all duration-120"
      enter-from-class="opacity-0 -translate-y-1 scale-[0.98]"
      leave-to-class="opacity-0 -translate-y-1 scale-[0.98]"
    >
      <div
        v-if="open"
        ref="panel"
        class="cr-select-panel"
        :class="{ 'cr-select-panel--up': dropUp }"
        role="listbox"
        @keydown="onPanelKey"
      >
        <div v-if="searchable" class="cr-select-search">
          <UIcon name="i-lucide-search" class="w-3.5 h-3.5 shrink-0" style="color: var(--cr-text-soft)" />
          <input
            ref="searchInput"
            v-model="query"
            type="text"
            placeholder="Buscar…"
            class="cr-select-search-input"
          />
        </div>

        <div class="cr-select-list">
          <button
            v-for="(o, i) in filtered"
            :key="o.value || `__null-${i}`"
            type="button"
            data-cr-opt
            class="cr-select-opt"
            :class="{
              'cr-select-opt--active': i === activeIdx,
              'cr-select-opt--selected': o.value === modelValue,
              'cr-select-opt--disabled': o.disabled,
            }"
            :aria-selected="o.value === modelValue"
            @mouseenter="activeIdx = i"
            @click="choose(o)"
          >
            <span class="cr-select-opt-label">{{ o.label }}</span>
            <span v-if="o.hint" class="cr-select-opt-hint">{{ o.hint }}</span>
            <UIcon
              v-if="o.value === modelValue"
              name="i-lucide-check"
              class="w-3.5 h-3.5 shrink-0 cr-select-opt-check"
            />
          </button>
          <div v-if="filtered.length === 0" class="cr-select-empty">
            Sin resultados
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style>
.cr-select {
  position: relative;
  display: inline-block;
  width: 100%;
}

.cr-select-trigger {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  background: var(--cr-surface);
  color: var(--cr-text);
  border: 1px solid var(--cr-border);
  border-radius: 10px;
  padding: 8px 12px;
  font-size: 13px;
  line-height: 18px;
  cursor: pointer;
  text-align: left;
  transition:
    border-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    box-shadow 140ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-select-trigger:hover { border-color: var(--cr-border-strong); }
.cr-select-trigger:focus,
.cr-select-trigger:focus-visible {
  outline: none;
  border-color: var(--color-wise-500);
  /* Slim 2-px ring — the old 4-px shadow looked like a thick "selected" border. */
  box-shadow: 0 0 0 2px rgb(159 232 112 / 0.22);
}
.cr-select--open .cr-select-trigger {
  border-color: var(--color-wise-500);
  box-shadow: 0 0 0 2px rgb(159 232 112 / 0.22);
}
.cr-select--disabled .cr-select-trigger {
  opacity: 0.55;
  cursor: not-allowed;
}

.cr-select-trigger-label {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cr-select-trigger-label--empty {
  color: var(--cr-text-soft);
}
.cr-select-chevron {
  color: var(--cr-text-soft);
  transition: transform 160ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-select-chevron--open { transform: rotate(180deg); }

/* Popup */
.cr-select-panel {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  right: 0;
  z-index: 50;
  background: var(--cr-surface);
  border: 1px solid var(--cr-border);
  border-radius: 12px;
  box-shadow:
    0 1px 2px rgb(0 0 0 / 0.05),
    0 12px 32px -8px rgb(0 0 0 / 0.18);
  overflow: hidden;
  max-height: 320px;
  display: flex;
  flex-direction: column;
}
html.dark .cr-select-panel {
  box-shadow:
    0 1px 2px rgb(0 0 0 / 0.4),
    0 14px 36px -8px rgb(0 0 0 / 0.55);
}

/* Open upward — used when there's no room below (e.g. table footer). */
.cr-select-panel--up {
  top: auto;
  bottom: calc(100% + 6px);
}

.cr-select-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--cr-border);
  background: var(--cr-surface-soft);
}
.cr-select-search-input {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  font-size: 12.5px;
  color: var(--cr-text);
  min-width: 0;
}
.cr-select-search-input::placeholder { color: var(--cr-text-soft); }

.cr-select-list {
  overflow-y: auto;
  padding: 4px;
}
.cr-select-opt {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  background: transparent;
  border: none;
  border-radius: 8px;
  padding: 8px 10px;
  font-size: 13px;
  color: var(--cr-text);
  cursor: pointer;
  text-align: left;
  transition: background-color 100ms ease;
}
.cr-select-opt-label {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cr-select-opt-hint {
  font-size: 11px;
  color: var(--cr-text-soft);
  flex-shrink: 0;
}
.cr-select-opt-check {
  color: var(--color-wise-600);
}
html.dark .cr-select-opt-check { color: var(--color-wise-400); }

.cr-select-opt--active {
  background: var(--cr-surface-soft);
}
html.dark .cr-select-opt--active {
  background: rgb(255 255 255 / 0.04);
}
.cr-select-opt--selected {
  color: var(--cr-text);
  font-weight: 600;
}
.cr-select-opt--disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.cr-select-empty {
  padding: 14px;
  text-align: center;
  font-size: 12px;
  color: var(--cr-text-soft);
}

/* Soft scrollbar inside the popup */
.cr-select-list::-webkit-scrollbar { width: 8px; }
.cr-select-list::-webkit-scrollbar-thumb {
  background: var(--cr-border-strong);
  border-radius: 4px;
}
.cr-select-list::-webkit-scrollbar-thumb:hover { background: var(--cr-text-soft); }
</style>
