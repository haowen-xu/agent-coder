<template>
  <el-container class="console-page">
    <el-main class="console-content">
      <el-card class="topbar" shadow="never">
        <div class="topbar-inner">
          <PageHeader
            title="Agent Coder"
            :subtitle="`当前用户: ${session.user?.username ?? '-'} (admin: ${isAdmin ? 'yes' : 'no'})`"
          />
          <div class="actions">
            <el-button @click="refreshPage">刷新</el-button>
            <el-button type="danger" plain @click="onLogout">退出</el-button>
          </div>
        </div>
      </el-card>

      <el-card class="nav-card" shadow="never">
        <el-menu :default-active="activeIndex" mode="horizontal" @select="onSelect">
          <el-menu-item index="/board">展板</el-menu-item>
          <el-sub-menu v-if="isAdmin" index="admin">
            <template #title>管理台</template>
            <el-menu-item index="/admin/overview">概览</el-menu-item>
            <el-menu-item index="/admin/users">用户</el-menu-item>
            <el-menu-item index="/admin/projects">项目</el-menu-item>
            <el-menu-item index="/admin/prompts">Prompt</el-menu-item>
            <el-menu-item index="/admin/ops">运维</el-menu-item>
          </el-sub-menu>
        </el-menu>
      </el-card>

      <router-view />
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSessionStore } from '../stores/session'
import PageHeader from '../components/common/PageHeader.vue'

const session = useSessionStore()
const route = useRoute()
const router = useRouter()

const isAdmin = computed(() => !!session.user?.is_admin)
const activeIndex = computed(() => {
  if (route.path.startsWith('/admin/')) {
    return route.path
  }
  return '/board'
})

function onSelect(index: string) {
  if (index !== route.path) {
    void router.push(index)
  }
}

function onLogout() {
  session.logout()
  void router.push('/login')
}

function refreshPage() {
  window.location.reload()
}
</script>

<style scoped>
.console-page {
  min-height: 100vh;
  background:
    radial-gradient(circle at 15% 20%, #edf5ff 0, transparent 42%),
    radial-gradient(circle at 85% 0, #e9f8f1 0, transparent 30%),
    linear-gradient(140deg, #f6f9fc 0%, #eef3f9 100%);
}

.console-content {
  padding: 20px;
}

.topbar {
  margin-bottom: 12px;
}

.topbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.nav-card {
  margin-bottom: 12px;
}

.actions {
  display: flex;
  gap: 8px;
}
</style>
