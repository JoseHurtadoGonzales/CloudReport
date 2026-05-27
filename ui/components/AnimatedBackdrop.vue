<script setup lang="ts">
// Decorative animated background. Two drifting blob gradients + a grain
// overlay. CSS keyframes (GPU-friendly: translate3d + scale only). Honors
// prefers-reduced-motion via the base CSS layer.
//
// In dark mode the blobs reduce opacity so they don't blow out the surface;
// the radial base becomes a deep wash with a hint of lime.
</script>

<template>
  <div aria-hidden="true" class="absolute inset-0 overflow-hidden pointer-events-none cr-backdrop">
    <!-- Light-mode base wash -->
    <div class="cr-backdrop-base-light" />
    <!-- Dark-mode base wash -->
    <div class="cr-backdrop-base-dark" />

    <!-- Primary drifting blob (lime) -->
    <div class="cr-backdrop-blob cr-backdrop-blob-1 cr-anim-drift-1" />
    <!-- Secondary blob (cyan accent) -->
    <div class="cr-backdrop-blob cr-backdrop-blob-2 cr-anim-drift-2" />

    <!-- Texture -->
    <div class="absolute inset-0 cr-bg-grain opacity-60" />

    <!-- Inner ring -->
    <div class="absolute inset-0 ring-1 ring-inset" style="--tw-ring-color: rgb(0 0 0 / 0.03)" />
  </div>
</template>

<style>
.cr-backdrop-base-light {
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse at top left, #e2f6d5 0%, #f6f8f4 55%, #ffffff 100%);
  transition: opacity 250ms cubic-bezier(0.22, 1, 0.36, 1);
}
html.dark .cr-backdrop-base-light { opacity: 0; }

.cr-backdrop-base-dark {
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse at top left, #1d2218 0%, #0e120c 60%, #0a0c08 100%);
  opacity: 0;
  transition: opacity 250ms cubic-bezier(0.22, 1, 0.36, 1);
}
html.dark .cr-backdrop-base-dark { opacity: 1; }

.cr-backdrop-blob {
  position: absolute;
  border-radius: 9999px;
  mix-blend-mode: multiply;
  filter: blur(90px);
  transition: opacity 250ms cubic-bezier(0.22, 1, 0.36, 1);
}
html.dark .cr-backdrop-blob { mix-blend-mode: screen; }

.cr-backdrop-blob-1 {
  top: -8rem; left: -8rem;
  width: 680px; height: 680px;
  background: radial-gradient(circle at 30% 30%, #9fe870 0%, rgba(159, 232, 112, 0) 70%);
  opacity: 0.7;
}
html.dark .cr-backdrop-blob-1 { opacity: 0.35; }

.cr-backdrop-blob-2 {
  bottom: -10rem; right: -8rem;
  width: 760px; height: 760px;
  background: radial-gradient(circle at 70% 70%, #38c8ff 0%, rgba(56, 200, 255, 0) 70%);
  opacity: 0.6;
  filter: blur(110px);
}
html.dark .cr-backdrop-blob-2 { opacity: 0.22; }
</style>
