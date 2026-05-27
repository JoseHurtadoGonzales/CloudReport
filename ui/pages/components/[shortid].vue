<script setup lang="ts">
const route = useRoute()
const router = useRouter()
const toasts = useToasts()
const entity = useEntity('components')
const { t } = useI18n()

const isNew = computed(() => route.params.shortid === 'new')

const form = ref({
  shortid: '', name: 'sin nombre',
  content: '<div class="card">\n  <h2>{{title}}</h2>\n  <p>{{body}}</p>\n</div>',
  engine: 'handlebars',
  helpers: '',
})
const saving = ref(false)

onMounted(async () => {
  if (isNew.value) return
  try {
    const c = await entity.get(route.params.shortid as string)
    form.value = { shortid: c.shortid, name: c.name, content: c.content, engine: c.engine, helpers: c.helpers ?? '' }
  } catch (err: any) {
    toasts.error(t('common.couldNotLoad'), extractError(err))
    router.push('/components')
  }
})

async function save() {
  saving.value = true
  try {
    const p = { name: form.value.name, content: form.value.content, engine: form.value.engine, helpers: form.value.helpers }
    if (isNew.value) {
      const c = await entity.create(p)
      toasts.success(t('components.created'))
      router.push(`/components/${c.shortid}`)
    } else {
      await entity.update(form.value.shortid, p)
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
        <NuxtLink to="/components" class="cr-row-action !w-9 !h-9"><UIcon name="i-lucide-arrow-left" class="w-4 h-4" /></NuxtLink>
        <div class="min-w-0">
          <input v-model="form.name" type="text" :placeholder="t('components.namePlaceholder')"
                 class="text-[20px] font-bold tracking-tight bg-transparent border-0 outline-none w-full" style="color: var(--cr-text)" />
          <div class="text-[11.5px] font-mono" style="color: var(--cr-text-soft)">
            {{ form.shortid || t('common.new') }} — {{ t('components.invokeWith') }} <code>&#123;&#123;&gt; {{ form.name }}&#125;&#125;</code>
          </div>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <select v-model="form.engine" class="cr-input !w-auto !py-2 !text-[13px] !pl-3" style="padding-top:8px; padding-bottom:8px">
          <option value="handlebars">handlebars</option>
          <option value="none">none</option>
        </select>
        <button class="cr-btn-primary !w-auto" :disabled="saving" @click="save">
          <UIcon name="i-lucide-save" class="w-4 h-4" /><span>{{ isNew ? t('header.create') : t('header.save') }}</span>
        </button>
      </div>
    </div>
    <CodeEditor v-model="form.content" language="html" height="640px" />
  </div>
</template>
