<template>
  <el-form label-position="top" class="compact-form" @submit.prevent="onSubmit">
    <el-form-item label="用户名">
      <el-input v-model="model.username" placeholder="用户名" />
    </el-form-item>
    <el-form-item label="密码">
      <el-input v-model="model.password" type="password" show-password placeholder="密码" />
    </el-form-item>
    <div class="form-inline-row">
      <el-checkbox v-model="model.is_admin">管理员</el-checkbox>
      <el-checkbox v-model="model.enabled">启用</el-checkbox>
    </div>
    <el-button type="primary" :loading="loading" @click="onSubmit">创建用户</el-button>
  </el-form>
</template>

<script setup lang="ts">
defineProps<{
  model: {
    username: string
    password: string
    is_admin: boolean
    enabled: boolean
  }
  loading: boolean
}>()

const emit = defineEmits<{
  submit: []
}>()

function onSubmit() {
  emit('submit')
}
</script>

<style scoped>
.compact-form {
  margin-bottom: 12px;
}

.form-inline-row {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}
</style>
