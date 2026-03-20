<template>
  <el-container class="page">
    <el-main class="content">
      <el-card class="panel" shadow="hover">
        <template #header>
          <div class="panel-header">
            <div>
              <h1>Agent Coder</h1>
              <p>Go + Hertz + GORM + Vue3 初始化模板</p>
            </div>
            <el-button type="primary" :loading="store.loading" @click="store.refresh">
              刷新状态
            </el-button>
          </div>
        </template>

        <el-alert
          v-if="store.error"
          type="error"
          show-icon
          :closable="false"
          :title="`请求失败: ${store.error}`"
        />

        <el-descriptions v-else :column="1" border>
          <el-descriptions-item label="服务状态">{{ store.status }}</el-descriptions-item>
          <el-descriptions-item label="数据库状态">{{ store.db }}</el-descriptions-item>
          <el-descriptions-item label="应用名">{{ store.meta?.app.name ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="运行环境">{{ store.meta?.app.env ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="监听地址">{{ store.meta?.server.addr ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="数据库方言">{{ store.meta?.db.dialect ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="服务时间">{{ store.meta?.now ?? '-' }}</el-descriptions-item>
        </el-descriptions>
      </el-card>
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useHealthStore } from './stores/health'

const store = useHealthStore()

onMounted(() => {
  void store.refresh()
})
</script>

<style scoped>
.page {
  min-height: 100vh;
  background: linear-gradient(120deg, #f6f9fc 0%, #eef3f9 100%);
}

.content {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.panel {
  width: min(920px, 100%);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.panel-header h1 {
  margin: 0;
  font-size: 24px;
}

.panel-header p {
  margin: 4px 0 0;
  color: #5f6b7a;
}
</style>
