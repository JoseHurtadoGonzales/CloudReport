<script setup lang="ts">
const route = useRoute()
const router = useRouter()
const toasts = useToasts()
const entity = useEntity('data')
const { t } = useI18n()

const isNew = computed(() => route.params.shortid === 'new')

const form = ref({
  shortid: '',
  name: 'sin nombre',
  dataJson: '{\n  "title": "Mi reporte",\n  "items": []\n}',
})
const saving = ref(false)

const isValid = computed(() => {
  try { JSON.parse(form.value.dataJson); return true } catch { return false }
})

onMounted(async () => {
  if (isNew.value) return
  try {
    const d = await entity.get(route.params.shortid as string)
    let raw = d.dataJson
    if (typeof raw === 'object') raw = JSON.stringify(raw, null, 2)
    form.value = { shortid: d.shortid, name: d.name, dataJson: raw ?? '{}' }
  } catch (err: any) {
    toasts.error(t('common.couldNotLoad'), extractError(err))
    router.push('/data')
  }
})

async function save() {
  if (!isValid.value) { toasts.error(t('data.invalid')); return }
  saving.value = true
  try {
    const payload = { name: form.value.name, dataJson: JSON.parse(form.value.dataJson) }
    if (isNew.value) {
      const d = await entity.create(payload)
      toasts.success(t('data.created'))
      router.push(`/data/${d.shortid}`)
    } else {
      await entity.update(form.value.shortid, payload)
      toasts.success(t('common.saved'))
    }
  } catch (err: any) { toasts.error(t('common.couldNotSave'), extractError(err)) }
  finally { saving.value = false }
}
</script>

<template>
  <div class="cr-anim-fade-up">
    <div class="flex items-center justify-between gap-3 mb-5 flex-wrap">
      <div class="flex items-center gap-3 min-w-0">
        <NuxtLink to="/data" class="cr-row-action !w-9 !h-9"><UIcon name="i-lucide-arrow-left" class="w-4 h-4" /></NuxtLink>
        <div class="min-w-0">
          <input v-model="form.name" type="text" :placeholder="t('data.namePlaceholder')"
                 class="text-[20px] font-bold tracking-tight bg-transparent border-0 outline-none w-full" style="color: var(--cr-text)" />
          <div class="text-[11.5px] font-mono" style="color: var(--cr-text-soft)">{{ form.shortid || t('common.new') }}</div>
        </div>
      </div>
      <div class="flex items-center gap-3">
        <p class="text-[12px] inline-flex items-center gap-1.5" :style="{ color: isValid ? 'var(--color-positive)' : '#d03238' }">
          <UIcon :name="isValid ? 'i-lucide-check' : 'i-lucide-circle-alert'" class="w-3.5 h-3.5" />
          {{ isValid ? t('data.valid') : t('data.invalid') }}
        </p>
        <button class="cr-btn-primary !w-auto" :disabled="saving || !isValid" @click="save">
          <UIcon name="i-lucide-save" class="w-4 h-4" />
          <span>{{ isNew ? t('header.create') : t('header.save') }}</span>
        </button>
      </div>
    </div>
    <CodeEditor v-model="form.dataJson" language="json" height="640px" />
  </div>
</template>
