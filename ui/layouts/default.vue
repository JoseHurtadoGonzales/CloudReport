<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'
import { useAuthStore } from '~/stores/auth'

const auth = useAuthStore()
const api = useApi()
const route = useRoute()
const { t } = useI18n()

// The panel is an OVERLAY toggled by a button (no hover): a slim rail is always
// docked; pressing the toggle slides the full panel open ON TOP of the page
// content (with a dimming scrim) — it never pushes the content aside. Closing
// happens via the toggle, the scrim, Esc, or navigating.
const open = ref(false)
const userMenuOpen = ref(false)
const userMenuRef = ref<HTMLElement | null>(null)
const palette = ref<{ open: () => void } | null>(null)

function toggleSidebar() { open.value = !open.value }
function closeSidebar()  { open.value = false }

onClickOutside(userMenuRef, () => (userMenuOpen.value = false))

// Mount auth guard
onMounted(async () => {
  auth.hydrate()
  if (!auth.isAuthenticated) await navigateTo('/login')
})

// ──── Side nav data ─────────────────────────────────────────────────────────
// We keep a stable `id` per entry that's locale-independent — it's the key
// for badge updates and the lookup into the i18n dictionary. The `label` is
// derived reactively from t() so switching locales relabels every link
// without any extra plumbing.
type NavEntry = { id: string; key: string; to: string; icon: string; badge: number }
const navBadges = ref<Record<string, number>>({
  templates: 0,
  schedules: 0,
  profiles: 0,
})
const workspaceNav = computed<NavEntry[]>(() => [
  { id: 'dashboard',  key: 'nav.dashboard',  to: '/',           icon: 'i-lucide-layout-grid',     badge: 0 },
  { id: 'templates',  key: 'nav.templates',  to: '/templates',  icon: 'i-lucide-file-text',       badge: navBadges.value.templates ?? 0 },
  { id: 'assets',     key: 'nav.assets',     to: '/assets',     icon: 'i-lucide-image',           badge: 0 },
  { id: 'scripts',    key: 'nav.scripts',    to: '/scripts',    icon: 'i-lucide-braces',          badge: 0 },
  { id: 'components', key: 'nav.components', to: '/components', icon: 'i-lucide-blocks',          badge: 0 },
  { id: 'data',       key: 'nav.data',       to: '/data',       icon: 'i-lucide-database',        badge: 0 },
  { id: 'schedules',  key: 'nav.schedules',  to: '/schedules',  icon: 'i-lucide-calendar-clock',  badge: navBadges.value.schedules ?? 0 },
  { id: 'reports',    key: 'nav.reports',    to: '/reports',    icon: 'i-lucide-file-down',       badge: 0 },
  { id: 'profiles',   key: 'nav.profiles',   to: '/profiles',   icon: 'i-lucide-activity',        badge: navBadges.value.profiles ?? 0 },
  { id: 'docs',       key: 'nav.docs',       to: '/docs',       icon: 'i-lucide-book-open',       badge: 0 },
])
const settingsNav = computed<{ id: string; key: string; to: string; icon: string }[]>(() => [
  { id: 'apiKeys',      key: 'nav.apiKeys',      to: '/settings/api-keys',      icon: 'i-lucide-key-round' },
  { id: 'users',        key: 'nav.users',        to: '/settings/users',         icon: 'i-lucide-users' },
  { id: 'versions',     key: 'nav.versions',     to: '/settings/versions',      icon: 'i-lucide-git-branch' },
  { id: 'importExport', key: 'nav.importExport', to: '/settings/import-export', icon: 'i-lucide-package' },
])

// Pull live counts so we can render badges in the sidebar (e.g. running renders).
// Throttled + non-overlapping: navigating used to fire these 3 OData calls on
// EVERY route change, contending with the page's own data fetch and making
// pages feel slow to enter. Now it refreshes at most once every few seconds
// (and never while a refresh is already in flight); `force` bypasses the gate
// for the initial mount.
let badgesInFlight = false
let lastBadgeRefresh = 0
async function refreshBadges(force = false) {
  const now = Date.now()
  if (!force && (badgesInFlight || now - lastBadgeRefresh < 4000)) return
  badgesInFlight = true
  try {
    const [tpls, schedules, profiles] = await Promise.all([
      api.get<{ '@odata.count': number }>('/odata/templates', { query: { $top: 0, $count: 'true' } }),
      api.get<{ value: any[] }>('/odata/schedules', { query: { $top: 200, $select: 'enabled,state' } }),
      api.get<{ value: any[] }>('/odata/profiles',  { query: { $top: 50,  $select: 'state' } }),
    ])
    const enabled = (schedules.value ?? []).filter((s: any) => s.enabled).length
    const running = (profiles.value ?? []).filter((p: any) => p.state === 'running').length
    navBadges.value = {
      ...navBadges.value,
      templates: tpls['@odata.count'] ?? 0,
      schedules: enabled,
      profiles: running,
    }
  } catch {
    // non-fatal
  } finally {
    badgesInFlight = false
    lastBadgeRefresh = Date.now()
  }
}
onMounted(() => refreshBadges(true))
// Refresh badges when route changes (after the user adds/runs something), and
// tuck the panel away so a fresh page never opens under an overlay. The refresh
// is throttled inside refreshBadges so rapid navigation doesn't hammer OData.
watch(() => route.path, () => {
  refreshBadges()
  open.value = false
})

function isMac() {
  return typeof navigator !== 'undefined' && navigator.platform.toUpperCase().includes('MAC')
}

// Sidebar shortcut: Ctrl+B / Cmd+B toggles the overlay; Esc closes it.
function onKey(e: KeyboardEvent) {
  const ctrl = isMac() ? e.metaKey : e.ctrlKey
  if (ctrl && e.key.toLowerCase() === 'b') {
    e.preventDefault()
    toggleSidebar()
  }
  if (e.key === 'Escape') closeSidebar()
}
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))

function onLogout() {
  auth.logout()
  navigateTo('/login')
}
function openPalette() {
  palette.value?.open()
}

const kbdCmd = computed(() => (isMac() ? '⌘' : 'Ctrl'))
</script>

<template>
  <div class="min-h-dvh overflow-x-clip" style="background: var(--cr-app-bg)">
    <!-- Scrim — dims the page behind the floating panel and closes on click. -->
    <Transition
      enter-active-class="transition-opacity duration-200 ease-out"
      leave-active-class="transition-opacity duration-150 ease-in"
      enter-from-class="opacity-0" leave-to-class="opacity-0"
    >
      <div v-if="open" class="cr-scrim" @click="closeSidebar" />
    </Transition>

    <!-- Sidebar — a slim rail always docked; slides open OVER the content when
         the toggle is pressed (no hover trigger). -->
    <aside
      class="cr-aside fixed inset-y-0 left-0 z-40 flex flex-col overflow-visible"
      :class="[
        open ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
        open ? 'lg:w-[264px]' : 'lg:w-[72px]',
        open ? 'cr-aside--float' : '',
        'w-[264px]',
      ]"
    >
      <!-- Brand row -->
      <div class="cr-side-brand">
        <NuxtLink to="/" class="cr-side-brand-link group">
          <span class="cr-side-brand-logo"><BrandLogo :size="36" /></span>
          <Transition name="cr-fade">
            <span v-if="open" class="cr-side-brand-text">
              <span class="cr-side-brand-name">cloud-report</span>
              <span class="cr-side-brand-version">v1.0</span>
            </span>
          </Transition>
        </NuxtLink>
        <Transition name="cr-fade">
          <button
            v-if="open"
            type="button"
            class="cr-side-close"
            :aria-label="t('sidebar.collapse')"
            :title="`${t('sidebar.collapse')} (Esc)`"
            @click="closeSidebar"
          >
            <UIcon name="i-lucide-panel-left-close" class="w-[18px] h-[18px]" />
          </button>
        </Transition>
      </div>

      <!-- Search button -->
      <button type="button" class="cr-side-search" @click="openPalette">
        <span class="cr-side-search-ico">
          <UIcon name="i-lucide-search" class="w-[18px] h-[18px]" :style="open ? 'color: var(--cr-text-soft)' : ''" />
        </span>
        <Transition name="cr-fade">
          <span v-if="open" class="cr-side-search-text">
            <span class="flex-1 text-left">{{ t('nav.search') }}</span>
            <kbd class="cr-side-kbd">{{ kbdCmd }}K</kbd>
          </span>
        </Transition>
      </button>

      <!-- Nav -->
      <nav class="cr-side-nav">
        <div class="cr-side-section">
          <Transition name="cr-fade">
            <p v-if="open" class="cr-side-section-title">{{ t('nav.workspace') }}</p>
          </Transition>
          <ul class="space-y-0.5">
            <SidebarItem
              v-for="n in workspaceNav"
              :key="n.to"
              :to="n.to"
              :label="t(n.key)"
              :icon="n.icon"
              :badge="n.badge"
              :badge-tone="n.id === 'profiles' ? 'amber' : 'lime'"
              :collapsed="!open"
            />
          </ul>
        </div>

        <div class="cr-side-divider" />

        <div class="cr-side-section">
          <Transition name="cr-fade">
            <p v-if="open" class="cr-side-section-title">{{ t('nav.settings') }}</p>
          </Transition>
          <ul class="space-y-0.5">
            <SidebarItem
              v-for="n in settingsNav"
              :key="n.to"
              :to="n.to"
              :label="t(n.key)"
              :icon="n.icon"
              :collapsed="!open"
            />
          </ul>
        </div>
      </nav>

      <!-- Bottom: user card -->
      <div class="cr-side-bottom">
        <div class="cr-side-user">
          <span class="cr-side-avatar-zone">
            <span class="cr-side-avatar">
              {{ auth.user?.username?.[0]?.toUpperCase() ?? 'U' }}
            </span>
          </span>
          <Transition name="cr-fade">
            <div v-if="open" class="flex-1 min-w-0">
              <div class="text-[12.5px] font-semibold truncate" style="color: var(--cr-text)">
                {{ auth.user?.username ?? '—' }}
              </div>
              <div class="text-[10.5px] truncate" style="color: var(--cr-text-soft)">
                {{ auth.user?.isAdmin ? t('user.admin') : t('user.user') }}
              </div>
            </div>
          </Transition>
          <Transition name="cr-fade">
            <button
              v-if="open"
              type="button"
              class="cr-side-user-action"
              :title="t('nav.logout')"
              @click="onLogout"
            >
              <UIcon name="i-lucide-log-out" class="w-3.5 h-3.5" />
            </button>
          </Transition>
        </div>
      </div>
    </aside>

    <!-- Main column — always reserves just the slim rail (72px). The panel
         floats on top when open, so this content never shifts. -->
    <div class="cr-main min-w-0 flex flex-col lg:pl-[72px]">
      <!-- Topbar -->
      <header
        class="sticky top-0 z-20 h-16 px-4 sm:px-6 flex items-center gap-3 border-b backdrop-blur"
        style="background: color-mix(in oklab, var(--cr-app-bg) 85%, transparent); border-color: var(--cr-border)"
      >
        <button
          type="button"
          class="lg:hidden cr-icon-btn !w-10 !h-10"
          :aria-label="t('sidebar.expand')"
          @click="toggleSidebar"
        >
          <UIcon name="i-lucide-menu" class="w-5 h-5" />
        </button>

        <!-- Desktop toggle — slides the overlay panel open / closed -->
        <button
          type="button"
          class="hidden lg:inline-flex cr-topbar-toggle"
          :class="open ? '' : 'cr-topbar-toggle--collapsed'"
          :aria-label="open ? t('sidebar.collapse') : t('sidebar.expand')"
          :aria-pressed="open"
          :title="`${open ? t('sidebar.collapse') : t('sidebar.expand')} (${kbdCmd}+B)`"
          @click="toggleSidebar"
        >
          <Transition
            mode="out-in"
            enter-active-class="transition duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]"
            leave-active-class="transition duration-100"
            enter-from-class="opacity-0 scale-75 -rotate-12"
            leave-to-class="opacity-0 scale-75 rotate-12"
          >
            <UIcon
              v-if="open"
              key="close"
              name="i-lucide-panel-left-close"
              class="w-[18px] h-[18px]"
            />
            <UIcon
              v-else
              key="open"
              name="i-lucide-panel-left-open"
              class="w-[18px] h-[18px]"
            />
          </Transition>
        </button>

        <button type="button" class="cr-search-btn flex-1 max-w-md" @click="openPalette">
          <UIcon name="i-lucide-search" class="w-4 h-4 shrink-0" style="color: var(--cr-text-soft)" />
          <span class="flex-1 text-left text-[13px]" style="color: var(--cr-text-soft)">{{ t('sidebar.searchPlaceholder') }}</span>
          <kbd class="cr-kbd hidden sm:inline-flex">{{ kbdCmd }}K</kbd>
        </button>

        <div class="flex-1" />

        <LanguageToggle class="mr-1" />
        <ColorModeToggle />

        <div ref="userMenuRef" class="relative">
          <button
            type="button"
            class="flex items-center gap-2.5 pl-2 pr-3 py-1.5 rounded-full transition-shadow duration-150"
            :class="userMenuOpen ? 'shadow-[0_0_0_4px_rgb(159_232_112_/_0.30)]' : ''"
            style="border: 1px solid var(--cr-border-strong); background: var(--cr-surface);"
            @click="userMenuOpen = !userMenuOpen"
          >
            <span class="w-7 h-7 rounded-full flex items-center justify-center font-bold text-xs" style="background: var(--color-wise-400); color: #0e0f0c">
              {{ auth.user?.username?.[0]?.toUpperCase() ?? 'U' }}
            </span>
            <span class="text-[13px] font-semibold hidden sm:inline" style="color: var(--cr-text)">{{ auth.user?.username ?? '—' }}</span>
            <UIcon name="i-lucide-chevron-down" class="w-3.5 h-3.5" style="color: var(--cr-text-muted)" />
          </button>

          <Transition
            enter-active-class="transition duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]"
            leave-active-class="transition duration-100"
            enter-from-class="opacity-0 -translate-y-1 scale-95"
            leave-to-class="opacity-0 -translate-y-1 scale-95"
          >
            <div
              v-if="userMenuOpen"
              class="absolute right-0 top-full mt-2 w-56 rounded-xl border shadow-lg overflow-hidden origin-top-right"
              style="background: var(--cr-surface); border-color: var(--cr-border-strong);"
              @click.stop
            >
              <div class="px-4 py-3 border-b" style="border-color: var(--cr-border)">
                <div class="text-[13px] font-semibold" style="color: var(--cr-text)">{{ auth.user?.username ?? '—' }}</div>
                <div class="text-[11px] mt-0.5" style="color: var(--cr-text-muted)">{{ auth.user?.isAdmin ? t('user.admin') : t('user.user') }}</div>
              </div>
              <NuxtLink to="/settings/api-keys" class="cr-menu-item" @click="userMenuOpen = false">
                <UIcon name="i-lucide-key-round" class="w-4 h-4" />
                <span>API Keys</span>
              </NuxtLink>
              <button class="cr-menu-item w-full text-left" @click="onLogout">
                <UIcon name="i-lucide-log-out" class="w-4 h-4" />
                <span>{{ t('nav.logout') }}</span>
              </button>
            </div>
          </Transition>
        </div>
      </header>

      <main class="flex-1 px-4 sm:px-6 py-6">
        <slot />
      </main>
    </div>

    <CommandPalette ref="palette" />
  </div>
</template>

<style>
/* ─── Overlay shell ──────────────────────────────────────────────────────── */
/* The panel is a slim docked rail that floats open on top of the content.
   Width + elevation animate with spring-flavoured easing; content never moves. */
.cr-aside {
  top: 0;
  height: 100dvh;
  background: var(--cr-surface);
  border-right: 1px solid var(--cr-border);
  will-change: width, transform;
  transition:
    width 300ms cubic-bezier(0.22, 1, 0.36, 1),
    transform 320ms cubic-bezier(0.23, 1, 0.32, 1),
    box-shadow 260ms cubic-bezier(0.23, 1, 0.32, 1),
    border-color 260ms ease,
    border-radius 260ms cubic-bezier(0.23, 1, 0.32, 1);
}
/* Floating (peeked / pinned / mobile-open): lift off the edge with a soft
   shadow + hairline ring and round the right corners. */
.cr-aside--float {
  border-right-color: transparent;
  border-top-right-radius: 18px;
  border-bottom-right-radius: 18px;
  box-shadow:
    0 24px 60px -12px rgb(14 15 12 / 0.28),
    0 8px 20px -8px rgb(14 15 12 / 0.16),
    0 0 0 1px var(--cr-border);
}
html.dark .cr-aside--float {
  box-shadow:
    0 24px 60px -12px rgb(0 0 0 / 0.60),
    0 8px 20px -8px rgb(0 0 0 / 0.45),
    0 0 0 1px var(--cr-border-strong);
}

/* Scrim behind the open overlay panel — click anywhere to close. */
.cr-scrim {
  position: fixed;
  inset: 0;
  z-index: 30;
  background: rgb(14 15 12 / 0.38);
  -webkit-backdrop-filter: blur(2px);
  backdrop-filter: blur(2px);
  cursor: pointer;
}
html.dark .cr-scrim { background: rgb(0 0 0 / 0.58); }

/* Close (✕) button in the panel header, shown only while open. */
.cr-side-close {
  width: 32px;
  height: 32px;
  border-radius: 9px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--cr-text-soft);
  background: transparent;
  border: 1px solid transparent;
  cursor: pointer;
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    border-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-side-close:hover {
  background: var(--cr-surface-soft);
  border-color: var(--cr-border);
  color: var(--cr-text);
}
.cr-side-close:active { transform: scale(0.92); }

/* Generic fade+slide for labels revealed as the panel widens. */
.cr-fade-enter-active {
  transition: opacity 180ms cubic-bezier(0.22, 1, 0.36, 1),
              transform 220ms cubic-bezier(0.34, 1.56, 0.64, 1);
  transition-delay: 40ms;
}
.cr-fade-leave-active {
  transition: opacity 100ms ease-in, transform 100ms ease-in;
}
.cr-fade-enter-from { opacity: 0; transform: translateX(-6px); }
.cr-fade-leave-to   { opacity: 0; transform: translateX(-4px); }

/* Search label wrapper (kept as one node so it can fade as a unit). */
.cr-side-search-text {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
  padding-right: 12px;
  overflow: hidden;
  white-space: nowrap;
}

/* Honour reduced-motion: keep the state change, drop the travel. */
@media (prefers-reduced-motion: reduce) {
  .cr-aside { transition-duration: 1ms; }
  .cr-fade-enter-active,
  .cr-fade-leave-active { transition-duration: 1ms; transition-delay: 0ms; }
}

/* ─── Brand ──────────────────────────────────────────────────────────────── */
.cr-side-brand {
  height: 64px;
  padding: 0 14px;
  display: flex;
  align-items: center;
  gap: 8px;
  border-bottom: 1px solid var(--cr-border);
  flex-shrink: 0;
}
.cr-side-brand-link {
  display: flex;
  align-items: center;
  gap: 0;
  flex: 1;
  min-width: 0;
}
/* Logo sits in the same fixed 44px zone as every icon → it never moves and
   never gets clipped, rail or expanded. */
.cr-side-brand-logo {
  flex: 0 0 44px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.cr-side-brand-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.1;
}
.cr-side-brand-name {
  font-weight: 700;
  font-size: 14px;
  letter-spacing: -0.01em;
  color: var(--cr-text);
  white-space: nowrap;
}
.cr-side-brand-version {
  font-size: 10.5px;
  font-weight: 600;
  color: var(--cr-text-soft);
  letter-spacing: 0.04em;
}

/* Topbar sidebar-toggle button — same vocabulary as `.cr-icon-btn` but a bit
   smaller + a left-aligned hover indicator so its "panel" feel is obvious. */
.cr-topbar-toggle {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--cr-text-muted);
  background: var(--cr-surface);
  border: 1px solid var(--cr-border);
  cursor: pointer;
  position: relative;
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    border-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-topbar-toggle:hover {
  background: var(--cr-surface-soft);
  border-color: var(--cr-border-strong);
  color: var(--cr-text);
}
.cr-topbar-toggle:active { transform: scale(0.94); }

/* Subtle lime accent rail on the left side of the icon when expanded — hints
   the panel is "open" — and disappears when collapsed. */
.cr-topbar-toggle::before {
  content: "";
  position: absolute;
  left: 6px;
  top: 50%;
  width: 2px;
  height: 14px;
  border-radius: 2px;
  background: var(--color-wise-500);
  opacity: 0;
  transform: translateY(-50%) scaleY(0.6);
  transition:
    opacity 180ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 220ms cubic-bezier(0.34, 1.56, 0.64, 1);
}
.cr-topbar-toggle:not(.cr-topbar-toggle--collapsed)::before {
  opacity: 1;
  transform: translateY(-50%) scaleY(1);
}

/* ─── Search ────────────────────────────────────────────────────────────── */
.cr-side-search {
  display: flex;
  align-items: center;
  gap: 0;
  height: 40px;
  margin: 12px 14px 8px;
  padding: 0;
  border-radius: 10px;
  background: var(--cr-surface-soft);
  color: var(--cr-text-muted);
  border: 1px solid transparent;
  font-size: 13px;
  cursor: pointer;
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    border-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-side-search:hover {
  background: var(--cr-surface);
  border-color: var(--cr-border);
  color: var(--cr-text);
}
.cr-side-search:active { transform: scale(0.98); }
/* Fixed 44px icon zone — matches nav icons, so the magnifier never moves. */
.cr-side-search-ico {
  flex: 0 0 44px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cr-side-kbd {
  display: inline-flex;
  align-items: center;
  padding: 1px 5px;
  font-family: ui-monospace, "JetBrains Mono", monospace;
  font-size: 10px;
  font-weight: 600;
  border-radius: 4px;
  background: var(--cr-surface);
  color: var(--cr-text-soft);
  border: 1px solid var(--cr-border);
}

/* ─── Nav layout ────────────────────────────────────────────────────────── */
.cr-side-nav {
  flex: 1;
  overflow-y: auto;
  /* `clip` (not `visible`) so a label mid-transition can never spawn a phantom
     horizontal scrollbar; the margin still lets the active rail bleed left. */
  overflow-x: clip;
  overflow-clip-margin: 16px;
  padding: 4px 14px 12px;
  scrollbar-width: thin;
}
.cr-side-section { }
.cr-side-section-title {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.10em;
  text-transform: uppercase;
  color: var(--cr-text-soft);
  padding: 12px 12px 8px;
}
.cr-side-divider {
  height: 1px;
  margin: 12px 16px;
  background: linear-gradient(
    to right,
    transparent 0%,
    var(--cr-border) 30%,
    var(--cr-border) 70%,
    transparent 100%
  );
}

/* ─── Bottom area ───────────────────────────────────────────────────────── */
.cr-side-bottom {
  padding: 10px 14px 14px;
  border-top: 1px solid var(--cr-border);
  flex-shrink: 0;
}

.cr-side-user {
  display: flex;
  align-items: center;
  gap: 0;
  padding: 8px 0;
  border-radius: 12px;
  transition: background-color 140ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-side-user:hover {
  background: var(--cr-surface-soft);
}
/* Avatar in the same fixed 44px zone → aligned with every icon, never moves. */
.cr-side-avatar-zone {
  flex: 0 0 44px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.cr-side-avatar {
  width: 32px;
  height: 32px;
  border-radius: 9999px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  font-size: 13px;
  background: linear-gradient(135deg, #9fe870 0%, #38c8ff 100%);
  color: #0e0f0c;
  box-shadow: 0 2px 6px rgb(159 232 112 / 0.30);
}
.cr-side-user-action {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--cr-text-soft);
  background: transparent;
  cursor: pointer;
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-side-user-action:hover {
  background: var(--cr-surface);
  color: #d03238;
}
.cr-side-user-action:active { transform: scale(0.92); }

/* ─── Topbar bits (unchanged) ───────────────────────────────────────────── */
.cr-menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  font-size: 13px;
  font-weight: 500;
  color: var(--cr-text);
  width: 100%;
  transition: background-color 140ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-menu-item:hover { background: var(--cr-surface-soft); }

.cr-search-btn {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 10px;
  background: var(--cr-surface);
  border: 1px solid var(--cr-border);
  cursor: pointer;
  transition:
    border-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-search-btn:hover {
  border-color: var(--cr-border-strong);
  background: var(--cr-surface-soft);
}
.cr-search-btn:active { transform: scale(0.99); }
</style>
