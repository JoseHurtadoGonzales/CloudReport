<script setup lang="ts">
interface Props {
  label: string
  value: string | number
  icon: string
  trend?: number
  hint?: string
}
defineProps<Props>()
</script>

<template>
  <div class="cr-card p-5 flex items-start gap-4">
    <span
      class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0"
      style="background: var(--color-wise-100); color: var(--color-wise-700)"
    >
      <UIcon :name="icon" class="w-5 h-5" />
    </span>
    <div class="min-w-0 flex-1">
      <p class="text-[12px] font-semibold uppercase tracking-wider" style="color: var(--cr-text-muted)">
        {{ label }}
      </p>
      <p class="text-[24px] font-bold mt-1 tabular-nums leading-tight" style="color: var(--cr-text)">
        {{ value }}
      </p>
      <p v-if="hint" class="text-[12px] mt-1" style="color: var(--cr-text-soft)">
        {{ hint }}
      </p>
      <p v-if="trend != null" class="text-[12px] mt-1 font-semibold inline-flex items-center gap-1"
         :style="{ color: trend >= 0 ? 'var(--color-positive)' : '#d03238' }">
        <UIcon :name="trend >= 0 ? 'i-lucide-trending-up' : 'i-lucide-trending-down'" class="w-3.5 h-3.5" />
        {{ trend > 0 ? '+' : '' }}{{ trend }}%
      </p>
    </div>
  </div>
</template>
