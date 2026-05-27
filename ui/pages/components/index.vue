<script setup lang="ts">
const { t } = useI18n()
</script>

<template>
  <EntityList
    entity-set="components"
    icon="i-lucide-blocks"
    :title="t('componentsList.title')"
    :description="t('componentsList.description')"
    new-path="/components/new"
    sub-key="engine"
  />
</template>
