<script setup lang="ts">
// On-brand empty state with optional inline SVG illustration. The `art` prop
// chooses one of our hand-drawn artworks. Fallback to the lucide icon if no
// art is named or the prop is omitted.

interface Props {
  icon?: string
  art?: 'templates' | 'reports' | 'assets' | 'inbox' | 'search' | 'profiles'
  title: string
  description?: string
  cta?: { label: string; to?: string; onClick?: () => void }
}
defineProps<Props>()
</script>

<template>
  <div class="cr-card p-10 sm:p-14 text-center flex flex-col items-center">
    <!-- Illustration / icon -->
    <div class="mb-5">
      <!-- Templates art -->
      <svg v-if="art === 'templates'" width="160" height="120" viewBox="0 0 160 120" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <rect x="32" y="14" width="76" height="96" rx="6" fill="white" stroke="#0e0f0c" stroke-opacity="0.10" stroke-width="1.5"/>
        <rect x="42" y="26" width="42" height="4" rx="2" fill="#0e0f0c" fill-opacity="0.80"/>
        <rect x="42" y="36" width="28" height="2.5" rx="1.25" fill="#0e0f0c" fill-opacity="0.30"/>
        <rect x="42" y="50" width="56" height="2" rx="1" fill="#0e0f0c" fill-opacity="0.16"/>
        <rect x="42" y="56" width="48" height="2" rx="1" fill="#0e0f0c" fill-opacity="0.16"/>
        <rect x="42" y="62" width="50" height="2" rx="1" fill="#0e0f0c" fill-opacity="0.16"/>
        <rect x="42" y="80" width="22" height="8" rx="3" fill="#9fe870"/>
        <!-- second page peeking behind -->
        <rect x="48" y="22" width="76" height="96" rx="6" fill="#f6f8f4" stroke="#0e0f0c" stroke-opacity="0.06" transform="rotate(4 86 70)"/>
        <!-- decorative star -->
        <circle cx="124" cy="22" r="6" fill="#9fe870"/>
        <path d="M124 18l1 3 3 1-3 1-1 3-1-3-3-1 3-1z" fill="#0e0f0c"/>
      </svg>

      <!-- Reports art -->
      <svg v-else-if="art === 'reports'" width="160" height="120" viewBox="0 0 160 120" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <rect x="30" y="20" width="100" height="80" rx="6" fill="white" stroke="#0e0f0c" stroke-opacity="0.10" stroke-width="1.5"/>
        <rect x="42" y="32" width="46" height="3" rx="1.5" fill="#0e0f0c"/>
        <rect x="42" y="70" width="14" height="20" rx="2" fill="#9fe870" fill-opacity="0.5"/>
        <rect x="60" y="60" width="14" height="30" rx="2" fill="#9fe870" fill-opacity="0.75"/>
        <rect x="78" y="50" width="14" height="40" rx="2" fill="#9fe870"/>
        <rect x="96" y="64" width="14" height="26" rx="2" fill="#9fe870" fill-opacity="0.7"/>
        <circle cx="130" cy="32" r="14" fill="#9fe870" fill-opacity="0.20"/>
        <path d="M124 32l4 4 8-8" stroke="#0e0f0c" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
      </svg>

      <!-- Assets art -->
      <svg v-else-if="art === 'assets'" width="160" height="120" viewBox="0 0 160 120" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <rect x="20" y="22" width="58" height="58" rx="6" fill="#9fe870" fill-opacity="0.25"/>
        <circle cx="36" cy="40" r="6" fill="#9fe870"/>
        <path d="M22 76l16-16 16 14 8-6 14 12" stroke="#0e0f0c" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
        <rect x="86" y="34" width="58" height="58" rx="6" fill="white" stroke="#0e0f0c" stroke-opacity="0.10"/>
        <rect x="94" y="46" width="42" height="3" rx="1.5" fill="#0e0f0c" fill-opacity="0.4"/>
        <rect x="94" y="56" width="34" height="2" rx="1" fill="#0e0f0c" fill-opacity="0.20"/>
        <rect x="94" y="62" width="38" height="2" rx="1" fill="#0e0f0c" fill-opacity="0.20"/>
        <rect x="94" y="68" width="28" height="2" rx="1" fill="#0e0f0c" fill-opacity="0.20"/>
      </svg>

      <!-- Profiles / activity art -->
      <svg v-else-if="art === 'profiles'" width="160" height="120" viewBox="0 0 160 120" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <rect x="22" y="32" width="116" height="56" rx="8" fill="white" stroke="#0e0f0c" stroke-opacity="0.10" stroke-width="1.5"/>
        <path d="M30 70 L52 50 L70 60 L90 38 L112 56 L130 46" stroke="#9fe870" stroke-width="2.5" fill="none" stroke-linecap="round" stroke-linejoin="round"/>
        <circle cx="52" cy="50" r="3" fill="#9fe870"/>
        <circle cx="70" cy="60" r="3" fill="#9fe870"/>
        <circle cx="90" cy="38" r="3" fill="#9fe870"/>
        <circle cx="112" cy="56" r="3" fill="#9fe870"/>
        <circle cx="130" cy="46" r="3" fill="#9fe870"/>
      </svg>

      <!-- Search empty -->
      <svg v-else-if="art === 'search'" width="140" height="120" viewBox="0 0 140 120" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <circle cx="56" cy="56" r="32" stroke="#0e0f0c" stroke-opacity="0.20" stroke-width="3" fill="white"/>
        <line x1="80" y1="80" x2="106" y2="106" stroke="#0e0f0c" stroke-opacity="0.35" stroke-width="6" stroke-linecap="round"/>
        <path d="M44 52a12 12 0 0 1 24 0" stroke="#9fe870" stroke-width="3" stroke-linecap="round" fill="none"/>
        <circle cx="48" cy="60" r="2" fill="#0e0f0c"/>
        <circle cx="64" cy="60" r="2" fill="#0e0f0c"/>
      </svg>

      <!-- Generic icon fallback -->
      <span
        v-else
        class="w-16 h-16 rounded-2xl flex items-center justify-center"
        style="background: var(--color-wise-100); color: var(--color-wise-700)"
      >
        <UIcon :name="icon ?? 'i-lucide-inbox'" class="w-8 h-8" />
      </span>
    </div>

    <h3 class="text-[18px] font-bold tracking-tight" style="color: var(--cr-text)">{{ title }}</h3>
    <p v-if="description" class="text-[14px] mt-2 max-w-md leading-relaxed" style="color: var(--cr-text-muted)">
      {{ description }}
    </p>
    <div v-if="cta" class="mt-6">
      <NuxtLink
        v-if="cta.to"
        :to="cta.to"
        class="cr-btn-primary !w-auto"
      >
        <UIcon name="i-lucide-plus" class="w-4 h-4" />
        <span>{{ cta.label }}</span>
      </NuxtLink>
      <button v-else type="button" class="cr-btn-primary !w-auto" @click="cta.onClick?.()">
        <UIcon name="i-lucide-plus" class="w-4 h-4" />
        <span>{{ cta.label }}</span>
      </button>
    </div>
  </div>
</template>
