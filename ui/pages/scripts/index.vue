<script setup lang="ts">
const { t } = useI18n()
</script>

<template>
  <EntityList
    entity-set="scripts"
    icon="i-lucide-braces"
    :title="t('scriptsList.title')"
    :description="t('scriptsList.description')"
    new-path="/scripts/new"
    sub-key="scope"
  />
</template>
