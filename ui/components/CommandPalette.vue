<script setup lang="ts">
// Global command palette — Cmd/Ctrl+K to open. Searches across all OData
// entities + navigates. Keyboard-first: arrow keys to move, Enter to select,
// Esc to close.

interface Result {
  type: 'page' | 'template' | 'asset' | 'script' | 'component' | 'data' | 'schedule'
  label: string
  hint?: string
  icon: string
  to?: string
  onSelect?: () => void
}

const open = ref(false)
const query = ref('')
const selected = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)
const router = useRouter()
const api = useApi()

const staticPages: Result[] = [
  { type: 'page', label: 'Dashboard',  icon: 'i-lucide-layout-grid',   to: '/' },
  { type: 'page', label: 'Templates',  icon: 'i-lucide-file-text',     to: '/templates' },
  { type: 'page', label: 'Assets',     icon: 'i-lucide-image',         to: '/assets' },
  { type: 'page', label: 'Scripts',    icon: 'i-lucide-braces',        to: '/scripts' },
  { type: 'page', label: 'Components', icon: 'i-lucide-blocks',        to: '/components' },
  { type: 'page', label: 'Data',       icon: 'i-lucide-database',      to: '/data' },
  { type: 'page', label: 'Schedules',  icon: 'i-lucide-calendar-clock', to: '/schedules' },
  { type: 'page', label: 'Reports',    icon: 'i-lucide-file-down',     to: '/reports' },
  { type: 'page', label: 'Profiles',   icon: 'i-lucide-activity',      to: '/profiles' },
  { type: 'page', label: 'API Keys',   icon: 'i-lucide-key-round',     to: '/settings/api-keys' },
  { type: 'page', label: 'Users',      icon: 'i-lucide-users',         to: '/settings/users' },
  { type: 'page', label: 'Versions',   icon: 'i-lucide-git-branch',    to: '/settings/versions' },
  { type: 'page', label: 'Import / Export', icon: 'i-lucide-package',  to: '/settings/import-export' },
  { type: 'page', label: 'Nueva plantilla', icon: 'i-lucide-plus',     to: '/templates/new', hint: 'crear' },
]

const remoteResults = ref<Result[]>([])
let abort: AbortController | null = null

watch(query, async (q) => {
  selected.value = 0
  if (abort) abort.abort()
  if (!q || q.length < 2) { remoteResults.value = []; return }
  abort = new AbortController()
  try {
    const r = await api.get<{ results: any[] }>('/studio/text-search', {
      query: { q },
      signal: abort.signal,
    })
    remoteResults.value = (r.results ?? []).map(x => ({
      type: x.entitySet as Result['type'],
      label: x.name,
      hint: x.entitySet,
      icon: iconFor(x.entitySet),
      to: routeFor(x.entitySet, x.shortid),
    }))
  } catch {
    remoteResults.value = []
  }
})

function iconFor(set: string) {
  return ({
    templates: 'i-lucide-file-text',
    assets:    'i-lucide-image',
    scripts:   'i-lucide-braces',
    components:'i-lucide-blocks',
    data:      'i-lucide-database',
  } as Record<string, string>)[set] ?? 'i-lucide-file'
}
function routeFor(set: string, shortid: string) {
  return ({
    templates: `/templates/${shortid}`,
    scripts:   `/scripts/${shortid}`,
    components:`/components/${shortid}`,
    data:      `/data/${shortid}`,
  } as Record<string, string>)[set] ?? '/'
}

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  const matchingPages = q
    ? staticPages.filter(p => p.label.toLowerCase().includes(q) || (p.hint?.toLowerCase().includes(q) ?? false))
    : staticPages
  return [...matchingPages, ...remoteResults.value]
})

function openPalette() {
  open.value = true
  query.value = ''
  selected.value = 0
  nextTick(() => inputRef.value?.focus())
}
function close() {
  open.value = false
}
function choose(r: Result) {
  close()
  if (r.to) router.push(r.to)
  else r.onSelect?.()
}

function onKey(e: KeyboardEvent) {
  const isMac = navigator.platform.toUpperCase().includes('MAC')
  const ctrl = isMac ? e.metaKey : e.ctrlKey
  if (ctrl && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    open.value ? close() : openPalette()
  } else if (open.value && e.key === 'Escape') {
    e.preventDefault()
    close()
  } else if (open.value && e.key === 'ArrowDown') {
    e.preventDefault()
    selected.value = Math.min(selected.value + 1, filtered.value.length - 1)
  } else if (open.value && e.key === 'ArrowUp') {
    e.preventDefault()
    selected.value = Math.max(selected.value - 1, 0)
  } else if (open.value && e.key === 'Enter') {
    e.preventDefault()
    const r = filtered.value[selected.value]
    if (r) choose(r)
  }
}

onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))

defineExpose({ open: openPalette, close })
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-150"
      leave-active-class="transition-opacity duration-100"
      enter-from-class="opacity-0" leave-to-class="opacity-0"
    >
      <div
        v-if="open"
        class="fixed inset-0 z-[80] bg-black/40 backdrop-blur-[2px] flex items-start justify-center pt-[10vh] px-4"
        @click.self="close"
      >
        <Transition
          appear
          enter-active-class="transition duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]"
          enter-from-class="opacity-0 scale-95 -translate-y-2"
          leave-active-class="transition duration-150"
          leave-to-class="opacity-0 scale-95"
        >
          <div class="cr-card w-full max-w-xl overflow-hidden" style="transform-origin: top center">
            <div class="flex items-center gap-3 px-5 py-4 border-b" style="border-color: var(--cr-border)">
              <UIcon name="i-lucide-search" class="w-5 h-5" style="color: var(--cr-text-soft)" />
              <input
                ref="inputRef"
                v-model="query"
                type="text"
                placeholder="Buscar plantillas, assets, ir a una página…"
                class="flex-1 bg-transparent border-0 outline-none text-[15px]"
                style="color: var(--cr-text)"
              />
              <kbd class="cr-kbd">esc</kbd>
            </div>

            <div class="max-h-[60vh] overflow-y-auto p-2">
              <p v-if="filtered.length === 0" class="text-center py-8 text-[13px]" style="color: var(--cr-text-soft)">
                Sin resultados
              </p>
              <ul v-else class="space-y-0.5">
                <li v-for="(r, i) in filtered" :key="`${r.type}-${r.label}-${i}`">
                  <button
                    type="button"
                    class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left"
                    :class="i === selected ? 'cr-cmd-row--active' : ''"
                    style="transition: background-color 100ms cubic-bezier(0.23,1,0.32,1)"
                    @click="choose(r)"
                    @mouseenter="selected = i"
                  >
                    <span class="w-7 h-7 rounded-md flex items-center justify-center shrink-0" style="background: var(--cr-surface-soft); color: var(--cr-text-muted)">
                      <UIcon :name="r.icon" class="w-4 h-4" />
                    </span>
                    <div class="flex-1 min-w-0">
                      <div class="font-medium text-[13.5px] truncate" style="color: var(--cr-text)">{{ r.label }}</div>
                      <div v-if="r.hint" class="text-[11px] uppercase tracking-wider" style="color: var(--cr-text-soft)">{{ r.hint }}</div>
                    </div>
                    <UIcon v-if="i === selected" name="i-lucide-corner-down-left" class="w-3.5 h-3.5" style="color: var(--cr-text-soft)" />
                  </button>
                </li>
              </ul>
            </div>

            <div class="flex items-center justify-between px-5 py-3 border-t text-[11px]" style="border-color: var(--cr-border); color: var(--cr-text-soft)">
              <span><kbd class="cr-kbd">↑</kbd> <kbd class="cr-kbd">↓</kbd> navegar · <kbd class="cr-kbd">↵</kbd> abrir</span>
              <span><kbd class="cr-kbd">{{ navigator?.platform?.toUpperCase()?.includes('MAC') ? '⌘' : 'Ctrl' }}</kbd> <kbd class="cr-kbd">K</kbd> alternar</span>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<style>
.cr-cmd-row--active {
  background: var(--cr-surface-soft);
}
html.dark .cr-cmd-row--active {
  background: rgb(255 255 255 / 0.06);
}
</style>
