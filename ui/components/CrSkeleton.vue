<script setup lang="ts">
// Thin, ergonomic wrapper around the global `.cr-skeleton` utility so pages can
// drop shimmer placeholders without repeating inline styles.
//   <CrSkeleton w="60%" h="14px" />            → a text-line bar
//   <CrSkeleton w="40px" h="40px" rounded="10px" />  → an icon tile
//   <CrSkeleton class="aspect-[16/10] w-full" />     → a thumbnail (via classes)
withDefaults(defineProps<{
  /** width — any CSS length or % (default: fills container) */
  w?: string
  /** height — any CSS length (default: 1em-ish text line) */
  h?: string
  /** border-radius override; 'full' → pill */
  rounded?: string
}>(), {
  w: '100%',
  h: '0.9em',
  rounded: '',
})
</script>

<template>
  <span
    class="cr-skeleton block"
    :style="{
      width: w,
      height: h,
      borderRadius: rounded === 'full' ? '9999px' : (rounded || undefined),
    }"
    aria-hidden="true"
  />
</template>
