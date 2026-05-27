<script setup lang="ts">
interface Props {
  title: string
  description?: string
  icon?: string
}
defineProps<Props>()
</script>

<template>
  <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
    <div class="min-w-0">
      <div class="flex items-center gap-3">
        <span
          v-if="icon"
          class="w-10 h-10 rounded-xl flex items-center justify-center"
          style="background: var(--color-wise-100); color: var(--color-wise-700)"
        >
          <UIcon :name="icon" class="w-5 h-5" />
        </span>
        <h1 class="text-[26px] font-black tracking-tight" style="color: var(--cr-text)">{{ title }}</h1>
      </div>
      <p v-if="description" class="text-[14px] mt-1.5 max-w-2xl" style="color: var(--cr-text-muted)">
        {{ description }}
      </p>
    </div>
    <div class="flex items-center gap-2 shrink-0">
      <slot name="actions" />
    </div>
  </div>
</template>
