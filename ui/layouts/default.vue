<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'
import { useAuthStore } from '~/stores/auth'

const auth = useAuthStore()
const api = useApi()
const route = useRoute()
const sidebar = useSidebarState()
const { t } = useI18n()

const sidebarOpen = ref(false)  // mobile drawer
const userMenuOpen = ref(false)
const userMenuRef = ref<HTMLElement | null>(null)
const palette = ref<{ open: () => void } | null>(null)

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
async function refreshBadges() {
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
  }
}
onMounted(refreshBadges)
// Refresh badges when route changes (after the user adds/runs something).
watch(() => route.path, () => refreshBadges())

function isMac() {
  return typeof navigator !== 'undefined' && navigator.platform.toUpperCase().includes('MAC')
}

// Sidebar shortcut: Ctrl+B / Cmd+B
function onKey(e: KeyboardEvent) {
  const ctrl = isMac() ? e.metaKey : e.ctrlKey
  if (ctrl && e.key.toLowerCase() === 'b') {
    e.preventDefault()
    sidebar.toggle()
  }
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
  <div class="min-h-dvh flex" style="background: var(--cr-app-bg)">
    <!-- Mobile overlay -->
    <Transition
      enter-active-class="transition-opacity duration-200"
      leave-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0" leave-to-class="opacity-0"
    >
      <div v-if="sidebarOpen" class="fixed inset-0 z-30 bg-black/40 lg:hidden" @click="sidebarOpen = false" />
    </Transition>

    <!-- Sidebar -->
    <aside
      class="fixed lg:sticky inset-y-0 left-0 z-40 shrink-0 flex flex-col border-r overflow-visible"
      :class="[
        sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
        sidebar.collapsed.value ? 'lg:w-[72px]' : 'lg:w-[252px]',
        'w-[252px]',
      ]"
      style="
        background: var(--cr-surface);
        border-color: var(--cr-border);
        top: 0; height: 100dvh;
        transition: width 280ms cubic-bezier(0.34, 1.20, 0.64, 1),
                    transform 280ms cubic-bezier(0.23, 1, 0.32, 1);
      "
    >
      <!-- Brand row -->
      <div class="cr-side-brand" :class="sidebar.collapsed.value ? 'cr-side-brand--collapsed' : ''">
        <NuxtLink to="/" class="cr-side-brand-link group">
          <BrandLogo :size="36" />
          <span v-if="!sidebar.collapsed.value" class="cr-side-brand-text">
            <span class="cr-side-brand-name">cloud-report</span>
            <span class="cr-side-brand-version">v1.0</span>
          </span>
        </NuxtLink>
      </div>

      <!-- Search button -->
      <button
        type="button"
        class="cr-side-search"
        :class="sidebar.collapsed.value ? 'cr-side-search--collapsed' : ''"
        @click="openPalette"
      >
        <UIcon name="i-lucide-search" class="w-[18px] h-[18px] shrink-0" :style="!sidebar.collapsed.value ? 'color: var(--cr-text-soft)' : ''" />
        <template v-if="!sidebar.collapsed.value">
          <span class="flex-1 text-left">{{ t('nav.search') }}</span>
          <kbd class="cr-side-kbd">{{ kbdCmd }}K</kbd>
        </template>
      </button>

      <!-- Nav -->
      <nav class="cr-side-nav">
        <div class="cr-side-section">
          <p v-if="!sidebar.collapsed.value" class="cr-side-section-title">{{ t('nav.workspace') }}</p>
          <ul class="space-y-0.5">
            <SidebarItem
              v-for="n in workspaceNav"
              :key="n.to"
              :to="n.to"
              :label="t(n.key)"
              :icon="n.icon"
              :badge="n.badge"
              :badge-tone="n.id === 'profiles' ? 'amber' : 'lime'"
              :collapsed="sidebar.collapsed.value"
            />
          </ul>
        </div>

        <div class="cr-side-divider" />

        <div class="cr-side-section">
          <p v-if="!sidebar.collapsed.value" class="cr-side-section-title">{{ t('nav.settings') }}</p>
          <ul class="space-y-0.5">
            <SidebarItem
              v-for="n in settingsNav"
              :key="n.to"
              :to="n.to"
              :label="t(n.key)"
              :icon="n.icon"
              :collapsed="sidebar.collapsed.value"
            />
          </ul>
        </div>
      </nav>

      <!-- Bottom: user card -->
      <div class="cr-side-bottom">
        <div class="cr-side-user" :class="sidebar.collapsed.value ? 'cr-side-user--collapsed' : ''">
          <span class="cr-side-avatar">
            {{ auth.user?.username?.[0]?.toUpperCase() ?? 'U' }}
          </span>
          <div v-if="!sidebar.collapsed.value" class="flex-1 min-w-0">
            <div class="text-[12.5px] font-semibold truncate" style="color: var(--cr-text)">
              {{ auth.user?.username ?? '—' }}
            </div>
            <div class="text-[10.5px] truncate" style="color: var(--cr-text-soft)">
              {{ auth.user?.isAdmin ? t('user.admin') : t('user.user') }}
            </div>
          </div>
          <button
            v-if="!sidebar.collapsed.value"
            type="button"
            class="cr-side-user-action"
            :title="t('nav.logout')"
            @click="onLogout"
          >
            <UIcon name="i-lucide-log-out" class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </aside>

    <!-- Main column -->
    <div class="flex-1 min-w-0 flex flex-col">
      <!-- Topbar -->
      <header
        class="sticky top-0 z-20 h-16 px-4 sm:px-6 flex items-center gap-3 border-b backdrop-blur"
        style="background: color-mix(in oklab, var(--cr-app-bg) 85%, transparent); border-color: var(--cr-border)"
      >
        <button
          type="button"
          class="lg:hidden cr-icon-btn !w-10 !h-10"
          aria-label="Abrir menú"
          @click="sidebarOpen = true"
        >
          <UIcon name="i-lucide-menu" class="w-5 h-5" />
        </button>

        <!-- Desktop sidebar toggle — replaces the in-sidebar collapse buttons -->
        <button
          type="button"
          class="hidden lg:inline-flex cr-topbar-toggle"
          :class="sidebar.collapsed.value ? 'cr-topbar-toggle--collapsed' : ''"
          :aria-label="sidebar.collapsed.value ? t('sidebar.expand') : t('sidebar.collapse')"
          :title="`${sidebar.collapsed.value ? t('sidebar.expand') : t('sidebar.collapse')} (${kbdCmd}+B)`"
          @click="sidebar.toggle()"
        >
          <Transition
            mode="out-in"
            enter-active-class="transition duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]"
            leave-active-class="transition duration-100"
            enter-from-class="opacity-0 scale-75 -rotate-12"
            leave-to-class="opacity-0 scale-75 rotate-12"
          >
            <UIcon
              v-if="sidebar.collapsed.value"
              key="open"
              name="i-lucide-panel-left-open"
              class="w-[18px] h-[18px]"
            />
            <UIcon
              v-else
              key="close"
              name="i-lucide-panel-left-close"
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
.cr-side-brand--collapsed {
  padding: 0;
  justify-content: center;
}
.cr-side-brand-link {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}
.cr-side-brand--collapsed .cr-side-brand-link {
  flex: 0;
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
  gap: 10px;
  margin: 12px 12px 8px;
  padding: 9px 12px;
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
.cr-side-search--collapsed {
  width: 44px;
  height: 44px;
  padding: 0;
  margin: 12px auto 8px;
  justify-content: center;
  border-radius: 10px;
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
  overflow-x: visible;             /* let the active rail extend past the items */
  padding: 4px 12px 12px;
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
  padding: 10px 12px 14px;
  border-top: 1px solid var(--cr-border);
  flex-shrink: 0;
}

.cr-side-user {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 12px;
  transition: background-color 140ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-side-user:hover {
  background: var(--cr-surface-soft);
}
.cr-side-user--collapsed {
  justify-content: center;
  padding: 8px 0;
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
