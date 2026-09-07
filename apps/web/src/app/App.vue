<script setup lang="ts">
import AppShell from '@/components/layout/AppShell.vue'
import DictImportIndicator from '@/features/meta/components/DictImportIndicator.vue'
</script>

<template>
  <AppShell>
    <template #header-actions>
      <DictImportIndicator />
    </template>
    <RouterView />
  </AppShell>
</template>
