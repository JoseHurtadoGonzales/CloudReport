<script setup lang="ts">
// Sidebar nav row with:
//   • origin-right tooltip when sidebar is collapsed (Teleported to <body>
//     with position:fixed so it can never be clipped by sidebar overflow)
//   • optional notification badge
//   • spring-feel hover scale on the icon
//   • :active scale(0.96) on the row (Emil's tactile rule)
//   • lateral lime indicator on the active state
//
// Tooltip skip-delay: after the first one opens, neighbours open instantly
// (Emil rule #7). We use a module-level boolean flag to coordinate across
// sibling instances.

interface Props {
  to: string
  label: string
  icon: string
  collapsed: boolean
  badge?: number | string
  badgeTone?: 'lime' | 'amber' | 'red' | 'mute'
}

const props = withDefaults(defineProps<Props>(), {
  badge: undefined,
  badgeTone: 'lime',
})

const route = useRoute()

const active = computed(() => {
  if (props.to === '/') return route.path === '/'
  return route.path === props.to || route.path.startsWith(props.to + '/')
})

const hostRef = ref<HTMLElement | null>(null)
const tooltipVisible = ref(false)
const tipPos = ref({ top: 0, left: 0 })

let hoverTimer: ReturnType<typeof setTimeout> | null = null

// Module-level coordination flag (shared by all SidebarItem instances on the
// page). Lives on window so it survives HMR. Resets after a short idle period.
const RECENT_KEY = '__cr_sidebar_tip_recent'
const RECENT_TTL = 250
function anyRecentlyOpen(): boolean {
  if (typeof window === 'undefined') return false
  return !!(window as any)[RECENT_KEY]
}
function markRecent() {
  if (typeof window === 'undefined') return
  ;(window as any)[RECENT_KEY] = true
  setTimeout(() => { (window as any)[RECENT_KEY] = false }, RECENT_TTL)
}

function recomputePosition() {
  if (!hostRef.value) return
  const rect = (hostRef.value as HTMLElement).getBoundingClientRect()
  tipPos.value = {
    top:  rect.top + rect.height / 2,
    left: rect.right + 12,
  }
}

function showTooltip() {
  if (!props.collapsed) return
  recomputePosition()
  if (anyRecentlyOpen()) {
    tooltipVisible.value = true
    markRecent()
    return
  }
  hoverTimer = setTimeout(() => {
    recomputePosition()
    tooltipVisible.value = true
    markRecent()
  }, 350)
}
function hideTooltip() {
  if (hoverTimer) { clearTimeout(hoverTimer); hoverTimer = null }
  tooltipVisible.value = false
}

// Keep position in sync if the page scrolls or the window resizes while the
// tooltip is open.
let raf = 0
function onWinChange() {
  if (!tooltipVisible.value) return
  cancelAnimationFrame(raf)
  raf = requestAnimationFrame(recomputePosition)
}
onMounted(() => {
  window.addEventListener('scroll',  onWinChange, true)
  window.addEventListener('resize',  onWinChange)
})
onBeforeUnmount(() => {
  window.removeEventListener('scroll', onWinChange, true)
  window.removeEventListener('resize', onWinChange)
  if (hoverTimer) clearTimeout(hoverTimer)
  cancelAnimationFrame(raf)
})

// If collapsed flips while tooltip is open → close it.
watch(() => props.collapsed, (v) => { if (!v) hideTooltip() })

const badgeStyle = computed(() => {
  return ({
    lime:  'background: var(--color-wise-400); color: #0e0f0c',
    amber: 'background: #ffd11a; color: #4a3b1c',
    red:   'background: #d03238; color: #fff',
    mute:  'background: var(--cr-border-strong); color: var(--cr-text)',
  })[props.badgeTone]
})
</script>

<template>
  <li ref="hostRef" class="cr-side-item-host">
    <NuxtLink
      :to="to"
      class="cr-side-item"
      :class="[active ? 'cr-side-item--active' : '', collapsed ? 'cr-side-item--collapsed' : '']"
      @mouseenter="showTooltip"
      @mouseleave="hideTooltip"
      @focus="showTooltip"
      @blur="hideTooltip"
    >
      <span class="cr-side-rail" aria-hidden="true" />
      <span class="cr-side-ico-wrap">
        <UIcon :name="icon" class="cr-side-ico" />
      </span>
      <span v-if="!collapsed" class="cr-side-label whitespace-nowrap">{{ label }}</span>

      <span
        v-if="badge != null && badge !== '' && badge !== 0 && !collapsed"
        class="cr-side-badge"
        :style="badgeStyle"
      >
        {{ badge }}
      </span>

      <span
        v-if="badge != null && badge !== '' && badge !== 0 && collapsed"
        class="cr-side-badge-dot"
        :style="badgeStyle"
        aria-hidden="true"
      />
    </NuxtLink>

    <!-- Teleported tooltip — never clipped by sidebar overflow / z-index trap -->
    <Teleport to="body">
      <Transition
        enter-active-class="cr-tip-enter"
        leave-active-class="cr-tip-leave"
        enter-from-class="cr-tip-from"
        leave-to-class="cr-tip-to"
      >
        <div
          v-if="collapsed && tooltipVisible"
          class="cr-tip"
          :style="{ top: tipPos.top + 'px', left: tipPos.left + 'px' }"
          role="tooltip"
        >
          <span class="cr-tip-arrow" aria-hidden="true" />
          {{ label }}
          <span
            v-if="badge != null && badge !== '' && badge !== 0"
            class="cr-tip-badge"
            :style="badgeStyle"
          >{{ badge }}</span>
        </div>
      </Transition>
    </Teleport>
  </li>
</template>

<style>
.cr-side-item-host {
  position: relative;
}

.cr-side-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 9px 10px;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 500;
  color: var(--cr-text-muted);
  position: relative;
  transition:
    background-color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    color 140ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-side-item:hover { background: var(--cr-surface-soft); color: var(--cr-text); }
.cr-side-item:active { transform: scale(0.97); }

.cr-side-rail {
  position: absolute;
  left: -14px;
  top: 50%;
  width: 3px;
  height: 0;
  border-radius: 0 3px 3px 0;
  background: var(--color-wise-500);
  transform: translateY(-50%);
  transition: height 220ms cubic-bezier(0.34, 1.56, 0.64, 1);
}
.cr-side-item--active .cr-side-rail { height: 22px; }

.cr-side-item--active {
  background: var(--color-wise-100);
  color: var(--color-wise-800);
  font-weight: 600;
}
html.dark .cr-side-item--active {
  background: rgb(159 232 112 / 0.14);
  color: var(--color-wise-300);
}
.cr-side-item--collapsed.cr-side-item--active {
  box-shadow: inset 0 0 0 1px rgb(159 232 112 / 0.35);
}
html.dark .cr-side-item--collapsed.cr-side-item--active {
  box-shadow: inset 0 0 0 1px rgb(159 232 112 / 0.28);
}

.cr-side-ico-wrap {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  transition: transform 200ms cubic-bezier(0.34, 1.56, 0.64, 1);
}
.cr-side-item:hover .cr-side-ico-wrap { transform: scale(1.10); }
.cr-side-ico { width: 18px; height: 18px; }

.cr-side-label { flex: 1; min-width: 0; }

.cr-side-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 18px;
  min-width: 18px;
  padding: 0 6px;
  border-radius: 9999px;
  font-size: 10.5px;
  font-weight: 700;
  letter-spacing: 0.02em;
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}
.cr-side-badge-dot {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 6px;
  height: 6px;
  border-radius: 9999px;
}

.cr-side-item--collapsed {
  width: 44px;
  height: 44px;
  padding: 0;
  margin: 0 auto;
  justify-content: center;
  border-radius: 10px;
}
.cr-side-item--collapsed .cr-side-rail { left: -8px; }
.cr-side-item--collapsed .cr-side-ico-wrap { width: 100%; height: 100%; }

/* ─────────────────────────────────────────────────────────────────────
   Tooltip (Teleported to <body>, position: fixed → never clipped)
   ───────────────────────────────────────────────────────────────────── */
.cr-tip {
  position: fixed;
  transform: translateY(-50%);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: var(--cr-text);
  color: var(--cr-app-bg);
  font-size: 12px;
  font-weight: 600;
  border-radius: 8px;
  white-space: nowrap;
  z-index: 9999;
  box-shadow:
    0 10px 30px -10px rgb(0 0 0 / 0.30),
    0 2px 6px rgb(0 0 0 / 0.10);
  pointer-events: none;
  transform-origin: left center;
}
html.dark .cr-tip {
  background: #f4f6f1;
  color: #0e0f0c;
}
.cr-tip-arrow {
  position: absolute;
  left: -4px;
  top: 50%;
  width: 8px;
  height: 8px;
  background: inherit;
  transform: translateY(-50%) rotate(45deg);
  border-radius: 2px;
  z-index: -1;
}
.cr-tip-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 16px;
  min-width: 16px;
  padding: 0 5px;
  border-radius: 9999px;
  font-size: 10px;
  font-weight: 700;
}

.cr-tip-enter {
  transition:
    opacity 160ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 200ms cubic-bezier(0.34, 1.56, 0.64, 1);
}
.cr-tip-leave {
  transition:
    opacity 100ms cubic-bezier(0.23, 1, 0.32, 1),
    transform 100ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-tip-from { opacity: 0; transform: translateY(-50%) translateX(-6px) scale(0.92); }
.cr-tip-to   { opacity: 0; transform: translateY(-50%) translateX(-4px) scale(0.96); }
</style>
