<script setup lang="ts">
const cfg = useRuntimeConfig()
const auth = useAuthStore()
const toasts = useToasts()
const { t } = useI18n()

const importing = ref(false)
const exportLoading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

async function doExport() {
  exportLoading.value = true
  try {
    const res = await fetch(`${cfg.public.apiBase}/api/export`, {
      method: 'POST',
      headers: { ...(auth.token ? { Authorization: `Bearer ${auth.token}` } : {}) },
    })
    if (!res.ok) throw new Error(await res.text())
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `cloud-report-export-${new Date().toISOString().slice(0, 10)}.zip`
    a.click()
    URL.revokeObjectURL(url)
    toasts.success(t('settings.importExport.exported'))
  } catch (err: any) {
    toasts.error(t('settings.importExport.couldNotExport'), err.message)
  } finally { exportLoading.value = false }
}

async function doImport(file: File) {
  importing.value = true
  try {
    const buffer = await file.arrayBuffer()
    const res = await fetch(`${cfg.public.apiBase}/api/import`, {
      method: 'POST',
      headers: {
        'content-type': 'application/zip',
        ...(auth.token ? { Authorization: `Bearer ${auth.token}` } : {}),
      },
      body: buffer,
    })
    if (!res.ok) throw new Error(await res.text())
    const stats = await res.json()
    const summary = Object.entries(stats).map(([k, v]) => `${k}: ${v}`).join(', ')
    toasts.success(t('settings.importExport.imported'), summary)
  } catch (err: any) {
    toasts.error(t('settings.importExport.couldNotImport'), err.message)
  } finally { importing.value = false }
}

function onFile(e: Event) {
  const t = e.target as HTMLInputElement
  if (t.files && t.files[0]) doImport(t.files[0])
  t.value = ''
}
</script>

<template>
  <div class="cr-anim-fade-up">
    <PageHeader
      icon="i-lucide-package"
      :title="t('settings.importExport.title')"
      :description="t('settings.importExport.description')"
    />

    <div class="grid md:grid-cols-2 gap-4">
      <SectionCard :title="t('settings.importExport.exportTitle')" icon="i-lucide-package-open" :description="t('settings.importExport.exportDesc')">
        <p class="text-[13px] mb-4" style="color: var(--cr-text-muted)">
          {{ t('settings.importExport.exportBody') }}
          (<code>templates/</code>, <code>assets/</code>, <code>scripts/</code>, …) {{ t('settings.importExport.exportBody2') }}
        </p>
        <button class="cr-btn-primary !w-auto" :disabled="exportLoading" @click="doExport">
          <UIcon v-if="!exportLoading" name="i-lucide-download" class="w-4 h-4" />
          <span v-else class="w-4 h-4 border-2 rounded-full cr-anim-spin" style="border-color: #0e0f0c; border-top-color: transparent" />
          <span>{{ exportLoading ? t('common.generating') : t('settings.importExport.exportBtn') }}</span>
        </button>
      </SectionCard>

      <SectionCard :title="t('settings.importExport.importTitle')" icon="i-lucide-package" :description="t('settings.importExport.importDesc')">
        <p class="text-[13px] mb-4" style="color: var(--cr-text-muted)">
          {{ t('settings.importExport.importBody') }} <code>shortid</code> {{ t('settings.importExport.importBody2') }}
        </p>
        <input ref="fileInput" type="file" accept=".zip,application/zip" class="hidden" @change="onFile" />
        <button class="cr-btn-primary !w-auto" :disabled="importing" @click="fileInput?.click()">
          <UIcon v-if="!importing" name="i-lucide-upload" class="w-4 h-4" />
          <span v-else class="w-4 h-4 border-2 rounded-full cr-anim-spin" style="border-color: #0e0f0c; border-top-color: transparent" />
          <span>{{ importing ? t('common.importing') : t('settings.importExport.importBtn') }}</span>
        </button>
      </SectionCard>
    </div>
  </div>
</template>
