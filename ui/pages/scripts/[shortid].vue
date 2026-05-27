<script setup lang="ts">
const route = useRoute()
const router = useRouter()
const toasts = useToasts()
const entity = useEntity('scripts')
const { t } = useI18n()

const isNew = computed(() => route.params.shortid === 'new')

const form = ref({
  shortid: '',
  name: 'sin nombre',
  content: 'function beforeRender(req, res) {\n  // req.data.now = new Date().toISOString()\n}\n',
  scope: 'template',
  isGlobal: false,
})
const saving = ref(false)

onMounted(async () => {
  if (isNew.value) return
  try {
    const s = await entity.get(route.params.shortid as string)
    form.value = {
      shortid: s.shortid, name: s.name, content: s.content,
      scope: s.scope, isGlobal: !!s.isGlobal,
    }
  } catch (err: any) {
    toasts.error(t('common.couldNotLoad'), extractError(err))
    router.push('/scripts')
  }
})

async function save() {
  saving.value = true
  try {
    const payload = { name: form.value.name, content: form.value.content, scope: form.value.scope, isGlobal: form.value.isGlobal }
    if (isNew.value) {
      const s = await entity.create(payload)
      toasts.success(t('scripts.created'))
      router.push(`/scripts/${s.shortid}`)
    } else {
      await entity.update(form.value.shortid, payload)
      toasts.success(t('common.saved'))
    }
  } catch (err: any) {
    toasts.error(t('common.couldNotSave'), extractError(err))
  } finally { saving.value = false }
}
</script>

<template>
  <div class="cr-anim-fade-up">
    <div class="flex items-center justify-between gap-3 mb-5 flex-wrap">
      <div class="flex items-center gap-3 min-w-0">
        <NuxtLink to="/scripts" class="cr-row-action !w-9 !h-9"><UIcon name="i-lucide-arrow-left" class="w-4 h-4" /></NuxtLink>
        <div class="min-w-0">
          <input v-model="form.name" type="text" :placeholder="t('scripts.namePlaceholder')"
                 class="text-[20px] font-bold tracking-tight bg-transparent border-0 outline-none w-full" style="color: var(--cr-text)" />
          <div class="text-[11.5px] font-mono" style="color: var(--cr-text-soft)">{{ form.shortid || t('common.new') }}</div>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <select v-model="form.scope" class="cr-input !w-auto !py-2 !text-[13px] !pl-3" style="padding-top:8px; padding-bottom:8px">
          <option value="template">{{ t('scripts.scopeTemplate') }}</option>
          <option value="folder">{{ t('scripts.scopeFolder') }}</option>
          <option value="global">{{ t('scripts.scopeGlobal') }}</option>
        </select>
        <button class="cr-btn-primary !w-auto" :disabled="saving" @click="save">
          <UIcon name="i-lucide-save" class="w-4 h-4" />
          <span>{{ isNew ? t('header.create') : t('header.save') }}</span>
        </button>
      </div>
    </div>
    <CodeEditor v-model="form.content" language="javascript" height="640px" :placeholder="t('scripts.placeholder')" />
    <p class="text-[12px] mt-3" style="color: var(--cr-text-muted)">
      {{ t('scripts.scopeHintLead') }} <code class="px-1 py-0.5 rounded" style="background: var(--cr-surface-soft)">template</code> {{ t('scripts.scopeHintTemplate') }}
      <code class="px-1 py-0.5 rounded" style="background: var(--cr-surface-soft)">global</code> {{ t('scripts.scopeHintGlobal') }}
    </p>
  </div>
</template>
