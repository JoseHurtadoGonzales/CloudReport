<script setup lang="ts">
// Light/Dark toggle. The actual state lives in `useTheme()` which persists to
// a cookie (so SSR + CSR agree → no FOUC on reload).

const theme = useTheme()
</script>

<template>
  <button
    type="button"
    class="cr-icon-btn relative overflow-hidden"
    :aria-label="theme.mode.value === 'dark' ? 'Cambiar a tema claro' : 'Cambiar a tema oscuro'"
    :title="theme.mode.value === 'dark' ? 'Tema claro' : 'Tema oscuro'"
    @click="theme.toggle()"
  >
    <Transition
      enter-active-class="transition duration-300 ease-[cubic-bezier(0.23,1,0.32,1)]"
      leave-active-class="transition duration-200 ease-[cubic-bezier(0.23,1,0.32,1)]"
      enter-from-class="opacity-0 scale-50 rotate-90"
      leave-to-class="opacity-0 scale-50 -rotate-90"
      mode="out-in"
    >
      <UIcon
        v-if="theme.mode.value === 'dark'"
        key="sun"
        name="i-lucide-sun"
        class="w-[22px] h-[22px] text-wise-400"
      />
      <UIcon
        v-else
        key="moon"
        name="i-lucide-moon"
        class="w-[22px] h-[22px]"
      />
    </Transition>
  </button>
</template>
