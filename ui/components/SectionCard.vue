<script setup lang="ts">
interface Props {
  title?: string
  description?: string
  icon?: string
}
defineProps<Props>()
</script>

<template>
  <section class="cr-card p-5 sm:p-6">
    <header v-if="title || $slots.headerActions" class="flex items-center justify-between gap-4 mb-4">
      <div class="flex items-center gap-3 min-w-0">
        <span
          v-if="icon"
          class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0"
          style="background: var(--color-wise-100); color: var(--color-wise-700)"
        >
          <UIcon :name="icon" class="w-4.5 h-4.5" />
        </span>
        <div class="min-w-0">
          <h2 v-if="title" class="text-[15px] font-bold tracking-tight" style="color: var(--cr-text)">
            {{ title }}
          </h2>
          <p v-if="description" class="text-[12.5px] mt-0.5" style="color: var(--cr-text-muted)">
            {{ description }}
          </p>
        </div>
      </div>
      <div class="shrink-0">
        <slot name="headerActions" />
      </div>
    </header>
    <slot />
  </section>
</template>
