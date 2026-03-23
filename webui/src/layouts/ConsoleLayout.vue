<template>
  <div class="console-shell">
    <el-container class="console-frame">
      <el-aside v-if="!isMobile" class="console-aside" :width="isCollapsed ? '72px' : '240px'">
        <div class="aside-brand">
          <el-icon class="brand-icon"><Cpu /></el-icon>
          <span v-show="!isCollapsed" class="brand-text">Agent Coder</span>
        </div>

        <el-scrollbar class="aside-scroll">
          <el-menu
            class="aside-menu"
            :default-active="activeIndex"
            :collapse="isCollapsed"
            :collapse-transition="false"
            @select="onSelect"
          >
            <el-menu-item index="/board">
              <el-icon><Monitor /></el-icon>
              <span>展板</span>
            </el-menu-item>

            <el-sub-menu v-if="isAdmin" index="admin">
              <template #title>
                <el-icon><Setting /></el-icon>
                <span>管理台</span>
              </template>
              <el-menu-item index="/admin/overview">
                <el-icon><DataAnalysis /></el-icon>
                <span>概览</span>
              </el-menu-item>
              <el-menu-item index="/admin/users">
                <el-icon><User /></el-icon>
                <span>用户</span>
              </el-menu-item>
              <el-menu-item index="/admin/projects">
                <el-icon><FolderOpened /></el-icon>
                <span>项目</span>
              </el-menu-item>
              <el-menu-item index="/admin/prompts">
                <el-icon><ChatDotRound /></el-icon>
                <span>Prompt</span>
              </el-menu-item>
              <el-menu-item index="/admin/ops">
                <el-icon><Tools /></el-icon>
                <span>运维</span>
              </el-menu-item>
            </el-sub-menu>
          </el-menu>
        </el-scrollbar>
      </el-aside>

      <el-container class="console-main-wrap">
        <el-header class="console-header">
          <div class="header-left">
            <el-button class="menu-toggle" text circle @click="toggleSidebar">
              <el-icon>
                <component :is="toggleIcon" />
              </el-icon>
            </el-button>
            <span class="header-title">{{ currentTitle }}</span>
          </div>

          <div class="header-right">
            <el-dropdown trigger="click" @command="onUserCommand">
              <div class="user-trigger">
                <el-avatar :size="28" class="user-avatar">{{ userInitial }}</el-avatar>
                <span class="username">{{ session.user?.username ?? 'Guest' }}</span>
                <el-icon class="dropdown-icon"><ArrowDown /></el-icon>
              </div>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item disabled>
                    {{ isAdmin ? '管理员' : '普通用户' }}
                  </el-dropdown-item>
                  <el-dropdown-item command="logout" divided>退出</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </el-header>

        <el-main class="console-content">
          <router-view />
        </el-main>
      </el-container>
    </el-container>

    <el-drawer
      v-model="mobileMenuOpen"
      direction="ltr"
      size="260px"
      :with-header="false"
      class="mobile-nav-drawer"
    >
      <div class="drawer-brand">
        <el-icon class="brand-icon"><Cpu /></el-icon>
        <span class="brand-text">Agent Coder</span>
      </div>

      <el-menu class="drawer-menu" :default-active="activeIndex" @select="onSelect">
        <el-menu-item index="/board">
          <el-icon><Monitor /></el-icon>
          <span>展板</span>
        </el-menu-item>

        <el-sub-menu v-if="isAdmin" index="admin-drawer">
          <template #title>
            <el-icon><Setting /></el-icon>
            <span>管理台</span>
          </template>
          <el-menu-item index="/admin/overview">概览</el-menu-item>
          <el-menu-item index="/admin/users">用户</el-menu-item>
          <el-menu-item index="/admin/projects">项目</el-menu-item>
          <el-menu-item index="/admin/prompts">Prompt</el-menu-item>
          <el-menu-item index="/admin/ops">运维</el-menu-item>
        </el-sub-menu>
      </el-menu>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import {
  ArrowDown,
  ChatDotRound,
  Cpu,
  DataAnalysis,
  Expand,
  Fold,
  FolderOpened,
  Monitor,
  Setting,
  Tools,
  User,
} from '@element-plus/icons-vue'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSessionStore } from '../stores/session'

const session = useSessionStore()
const route = useRoute()
const router = useRouter()

const isMobile = ref(false)
const isCollapsed = ref(false)
const mobileMenuOpen = ref(false)

const isAdmin = computed(() => !!session.user?.is_admin)

const activeIndex = computed(() => {
  if (route.path.startsWith('/admin/')) {
    return route.path
  }
  if (route.path.startsWith('/board')) {
    return '/board'
  }
  return '/board'
})

const routeTitleMap: Record<string, string> = {
  '/board': '展板',
  '/admin/overview': '管理台 / 概览',
  '/admin/users': '管理台 / 用户',
  '/admin/projects': '管理台 / 项目',
  '/admin/prompts': '管理台 / Prompt',
  '/admin/ops': '管理台 / 运维',
}

const currentTitle = computed(() => routeTitleMap[activeIndex.value] ?? '控制台')

const userInitial = computed(() => {
  const name = session.user?.username?.trim()
  if (!name) {
    return 'U'
  }
  return name.slice(0, 1).toUpperCase()
})

const toggleIcon = computed(() => {
  if (isMobile.value) {
    return Expand
  }
  return isCollapsed.value ? Expand : Fold
})

function syncViewport() {
  isMobile.value = window.innerWidth < 992
  if (!isMobile.value) {
    mobileMenuOpen.value = false
  }
}

function toggleSidebar() {
  if (isMobile.value) {
    mobileMenuOpen.value = !mobileMenuOpen.value
    return
  }
  isCollapsed.value = !isCollapsed.value
}

function onSelect(index: string) {
  if (index !== route.path) {
    void router.push(index)
  }
  if (isMobile.value) {
    mobileMenuOpen.value = false
  }
}

function onUserCommand(command: string | number | object) {
  if (command !== 'logout') {
    return
  }
  session.logout()
  void router.push('/login')
}

watch(
  () => route.path,
  () => {
    if (isMobile.value) {
      mobileMenuOpen.value = false
    }
  },
)

onMounted(() => {
  syncViewport()
  window.addEventListener('resize', syncViewport)
})

onUnmounted(() => {
  window.removeEventListener('resize', syncViewport)
})
</script>

<style scoped>
.console-shell {
  --surface: rgba(255, 255, 255, 0.78);
  --surface-strong: rgba(255, 255, 255, 0.94);
  --line: rgba(18, 53, 36, 0.09);
  min-height: 100vh;
  background:
    radial-gradient(circle at 12% 8%, #dceeff 0, transparent 36%),
    radial-gradient(circle at 85% 16%, #d9f8e9 0, transparent 34%),
    linear-gradient(140deg, #f1f6fb 0%, #e8f0f8 100%);
}

.console-frame {
  min-height: 100vh;
}

.console-aside {
  border-right: 1px solid var(--line);
  background: var(--surface-strong);
  backdrop-filter: blur(6px);
}

.aside-brand {
  height: 60px;
  padding: 0 16px;
  display: flex;
  align-items: center;
  gap: 10px;
  border-bottom: 1px solid var(--line);
}

.brand-icon {
  font-size: 18px;
  color: #1f7a5a;
}

.brand-text {
  font-weight: 700;
  letter-spacing: 0.02em;
  color: #1f2937;
}

.aside-scroll {
  height: calc(100vh - 60px);
}

.aside-menu,
.drawer-menu {
  border-right: none;
  background: transparent;
}

.console-main-wrap {
  min-width: 0;
}

.console-header {
  height: 60px;
  padding: 0 16px;
  border-bottom: 1px solid var(--line);
  background: var(--surface-strong);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.header-left {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.header-title {
  font-size: 15px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.header-right {
  display: flex;
  align-items: center;
}

.menu-toggle {
  color: #1f7a5a;
}

.user-trigger {
  height: 36px;
  border: 1px solid var(--line);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.9);
  padding: 4px 8px 4px 4px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
}

.user-avatar {
  background: linear-gradient(120deg, #1f7a5a, #248a7a);
  color: #fff;
}

.username {
  font-size: 13px;
  max-width: 160px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dropdown-icon {
  color: #6b7280;
}

.console-content {
  padding: 16px;
}

.drawer-brand {
  height: 52px;
  display: flex;
  align-items: center;
  gap: 10px;
  border-bottom: 1px solid var(--line);
  margin-bottom: 6px;
}

@media (max-width: 991px) {
  .console-header {
    padding: 0 12px;
  }

  .console-content {
    padding: 12px;
  }

  .username {
    max-width: 94px;
  }
}
</style>
