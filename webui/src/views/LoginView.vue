<template>
  <el-card class="login-card" shadow="hover">
    <template #header>
      <div class="login-header">
        <h1>Agent Coder</h1>
        <p>登录后进入控制台</p>
      </div>
    </template>
    <el-form label-position="top" @submit.prevent="onLogin">
      <el-form-item label="用户名">
        <el-input v-model="loginForm.username" placeholder="admin" />
      </el-form-item>
      <el-form-item label="密码">
        <el-input v-model="loginForm.password" type="password" placeholder="******" show-password />
      </el-form-item>
      <el-alert
        v-if="session.error"
        :closable="false"
        type="error"
        show-icon
        :title="session.error"
        class="mb-12"
      />
      <el-button type="primary" :loading="session.loading" @click="onLogin">登录</el-button>
    </el-form>
  </el-card>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSessionStore } from '../stores/session'

const session = useSessionStore()
const route = useRoute()
const router = useRouter()

const loginForm = reactive({
  username: 'admin',
  password: 'admin123',
})

async function onLogin() {
  await session.login(loginForm.username, loginForm.password)
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/board'
  await router.push(redirect)
}
</script>

<style scoped>
.login-card {
  width: min(460px, 100%);
  margin: 8vh auto 0;
}

.login-header h1 {
  margin: 0;
  font-size: 28px;
}

.login-header p {
  margin: 6px 0 0;
  color: #5f6b7a;
}

.mb-12 {
  margin-bottom: 12px;
}
</style>
