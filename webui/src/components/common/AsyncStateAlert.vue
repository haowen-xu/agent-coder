<template>
  <el-alert
    v-if="error"
    :closable="false"
    type="error"
    show-icon
    :title="error"
    class="mb-12"
  />
</template>

<script setup lang="ts">
defineProps<{
  error?: string
}>()
</script>

<style scoped>
.mb-12 {
  margin-bottom: 12px;
}
</style>
