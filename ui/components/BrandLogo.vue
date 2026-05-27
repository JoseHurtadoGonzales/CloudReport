<script setup lang="ts">
// Custom brand mark. Lime rounded square with:
//   • subtle top→bottom gradient for depth
//   • thin white highlight line at the top edge (glass / 3D feel)
//   • a rising-line chart with three nodes (data points) + a flagship dot
// Sized by the `size` prop (px). Stays crisp at any size because everything
// is in vector + relative units.

interface Props {
  size?: number
}
const props = withDefaults(defineProps<Props>(), { size: 36 })

// Unique gradient IDs per instance so multiple logos on a page don't collide.
const uid = useId()
const gradId = `cr-brand-grad-${uid}`
const shineId = `cr-brand-shine-${uid}`
</script>

<template>
  <span class="cr-brand" :style="{ width: size + 'px', height: size + 'px' }">
    <svg
      :width="size"
      :height="size"
      viewBox="0 0 40 40"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      <defs>
        <linearGradient :id="gradId" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%"   stop-color="#b7f283" />
          <stop offset="55%"  stop-color="#9fe870" />
          <stop offset="100%" stop-color="#7dcf4d" />
        </linearGradient>
        <linearGradient :id="shineId" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%"   stop-color="rgba(255,255,255,0.55)" />
          <stop offset="100%" stop-color="rgba(255,255,255,0)" />
        </linearGradient>
      </defs>

      <!-- Rounded square w/ gradient + 1px stroke for crispness -->
      <rect width="40" height="40" rx="10" :fill="`url(#${gradId})`" />
      <!-- Top highlight line -->
      <rect x="1" y="1" width="38" height="14" rx="9" :fill="`url(#${shineId})`" />
      <!-- Subtle inner stroke -->
      <rect
        x="0.5" y="0.5" width="39" height="39" rx="9.5"
        fill="none"
        stroke="rgba(14,15,12,0.10)" stroke-width="1"
      />

      <!-- Rising chart polyline -->
      <path
        d="M 9 28 L 16 22 L 22 25 L 31 13"
        fill="none"
        stroke="#0e0f0c"
        stroke-width="2.6"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <!-- Nodes -->
      <circle cx="16" cy="22" r="1.5" fill="#0e0f0c" />
      <circle cx="22" cy="25" r="1.5" fill="#0e0f0c" />
      <!-- Flagship dot at the peak -->
      <circle cx="31" cy="13" r="2.8" fill="#0e0f0c" />
      <circle cx="31" cy="13" r="1"   fill="#9fe870" />
    </svg>
  </span>
</template>

<style>
.cr-brand {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  /* Glow + lift */
  box-shadow:
    0 8px 22px -6px rgb(159 232 112 / 0.55),
    0 2px 6px rgb(14 15 12 / 0.10);
  transition:
    transform 220ms cubic-bezier(0.34, 1.56, 0.64, 1),
    box-shadow 220ms cubic-bezier(0.23, 1, 0.32, 1);
}
.cr-brand:hover {
  transform: rotate(-6deg) scale(1.06);
  box-shadow:
    0 14px 30px -6px rgb(159 232 112 / 0.65),
    0 3px 10px rgb(14 15 12 / 0.12);
}
</style>
