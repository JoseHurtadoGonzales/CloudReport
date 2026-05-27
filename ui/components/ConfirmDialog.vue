<script setup lang="ts">
interface Props {
  modelValue: boolean
  title: string
  description?: string
  confirmLabel?: string
  cancelLabel?: string
  destructive?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  confirmLabel: 'Confirmar',
  cancelLabel: 'Cancelar',
  destructive: false,
})

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  confirm: []
  cancel: []
}>()

function onCancel() {
  emit('update:modelValue', false)
  emit('cancel')
}
function onConfirm() {
  emit('confirm')
  emit('update:modelValue', false)
}
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-200"
      leave-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0" leave-to-class="opacity-0"
    >
      <div
        v-if="modelValue"
        class="fixed inset-0 z-50 flex items-center justify-center px-4 bg-black/50 backdrop-blur-sm"
        @click.self="onCancel"
      >
        <Transition
          appear
          enter-active-class="transition duration-200 ease-[cubic-bezier(0.22,1,0.36,1)]"
          enter-from-class="opacity-0 scale-95 -translate-y-2"
          leave-active-class="transition duration-150"
          leave-to-class="opacity-0 scale-95"
        >
          <div class="cr-card max-w-md w-full p-6">
            <div class="flex items-start gap-4">
              <span
                class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0"
                :style="destructive
                  ? 'background: rgb(208 50 56 / 0.10); color: #a72027'
                  : 'background: var(--color-wise-100); color: var(--color-wise-700)'"
              >
                <UIcon :name="destructive ? 'i-lucide-triangle-alert' : 'i-lucide-help-circle'" class="w-5 h-5" />
              </span>
              <div class="flex-1 min-w-0">
                <h3 class="text-[17px] font-bold tracking-tight" style="color: var(--cr-text)">{{ title }}</h3>
                <p v-if="description" class="text-[14px] mt-1.5" style="color: var(--cr-text-muted)">
                  {{ description }}
                </p>
              </div>
            </div>
            <div class="mt-6 flex items-center justify-end gap-2">
              <button
                type="button"
                class="px-4 py-2 rounded-xl text-[13.5px] font-semibold transition-colors duration-150"
                style="color: var(--cr-text-muted)"
                @click="onCancel"
              >
                {{ cancelLabel }}
              </button>
              <button
                type="button"
                class="cr-btn-primary !w-auto"
                :style="destructive ? 'background: #d03238; color: #fff' : ''"
                @click="onConfirm"
              >
                {{ confirmLabel }}
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
