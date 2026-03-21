<template>
  <el-select
    :model-value="modelValue"
    placeholder="选择项目"
    filterable
    :style="{ width }"
    @update:model-value="onUpdate"
  >
    <el-option
      v-for="row in projects"
      :key="row.project_key"
      :label="`${row.project_key} (${row.name})`"
      :value="row.project_key"
    />
  </el-select>
</template>

<script setup lang="ts">
interface ProjectOption {
  project_key: string
  name: string
}

withDefaults(
  defineProps<{
    modelValue: string
    projects: ProjectOption[]
    width?: string
  }>(),
  {
    width: '240px',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [value: string]
}>()

function onUpdate(value: string) {
  emit('update:modelValue', value)
  emit('change', value)
}
</script>
