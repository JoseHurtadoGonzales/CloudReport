<script setup lang="ts">
const { t } = useI18n()
</script>

<template>
  <EntityList
    entity-set="data"
    icon="i-lucide-database"
    :title="t('dataList.title')"
    :description="t('dataList.description')"
    new-path="/data/new"
  />
</template>
