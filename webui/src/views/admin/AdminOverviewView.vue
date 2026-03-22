<template>
  <el-row :gutter="16" class="panel-row">
    <el-col :xs="24" :md="12" :xl="8">
      <el-card class="panel" shadow="never">
        <template #header>
          <div class="panel-title">管理入口</div>
        </template>
        <el-space direction="vertical" alignment="start">
          <el-button text @click="router.push('/admin/users')">用户管理</el-button>
          <el-button text @click="router.push('/admin/projects')">项目管理</el-button>
          <el-button text @click="router.push('/admin/prompts')">Prompt 管理</el-button>
          <el-button text @click="router.push('/admin/ops')">运维与日志</el-button>
        </el-space>
      </el-card>
    </el-col>

    <el-col :xs="24" :md="12" :xl="8">
      <el-card class="panel" shadow="never">
        <template #header>
          <div class="panel-title">系统健康</div>
        </template>
        <el-descriptions :column="1" size="small" border>
          <el-descriptions-item label="Health">{{ health.status }}</el-descriptions-item>
          <el-descriptions-item label="DB">{{ health.db }}</el-descriptions-item>
          <el-descriptions-item label="Env">{{ health.meta?.app?.env ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="Now">{{ formatLocalDateTime(health.meta?.now) }}</el-descriptions-item>
        </el-descriptions>
      </el-card>
    </el-col>

    <el-col :xs="24" :xl="8">
      <el-card class="panel" shadow="never">
        <template #header>
          <div class="panel-title">运行指标</div>
        </template>
        <el-descriptions v-if="admin.metrics" :column="1" size="small" border>
          <el-descriptions-item label="项目">{{ admin.metrics.projects.total }} / enabled {{ admin.metrics.projects.enabled }}</el-descriptions-item>
          <el-descriptions-item label="Issues">{{ admin.metrics.issues.total }}</el-descriptions-item>
          <el-descriptions-item label="Runs">{{ admin.metrics.runs.total }}</el-descriptions-item>
        </el-descriptions>
      </el-card>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAdminStore } from '../../stores/admin'
import { useHealthStore } from '../../stores/health'
import { useSessionStore } from '../../stores/session'
import { formatLocalDateTime } from '../../utils/format'

const router = useRouter()
const session = useSessionStore()
const admin = useAdminStore()
const health = useHealthStore()

onMounted(async () => {
  await health.refresh()
  if (session.token) {
    await admin.fetchMetrics(session.token)
  }
})
</script>

<style scoped>
.panel-row {
  margin-top: 4px;
}

.panel {
  min-height: 220px;
}

.panel-title {
  font-weight: 600;
}
</style>
