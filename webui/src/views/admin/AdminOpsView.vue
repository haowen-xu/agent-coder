<template>
  <el-row :gutter="16" class="panel-row">
    <el-col :span="24">
      <el-card class="admin-card" shadow="never">
        <template #header>
          <div class="panel-title">运维与日志</div>
        </template>
        <OpsPanel
          :projects="projects"
          :project-issues="admin.projectIssues"
          :issue-runs="admin.issueRuns"
          :run-logs="admin.runLogs"
          :metrics="admin.metrics"
          :ops-form="opsForm"
          :loading="loading"
          @project-change="onOpsProjectChange"
          @refresh-metrics="refreshOpsMetrics"
          @reset-sync="resetOpsProjectSyncCursor"
          @retry-issue="retryOpsIssue"
          @load-runs="loadOpsIssueRuns"
          @load-logs="loadOpsRunLogs"
          @cancel-run="cancelOpsRun"
          @issue-row-click="onIssueRowClick"
          @run-row-click="onRunRowClick"
        />
      </el-card>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { onMounted } from 'vue'
import { useAdminStore } from '../../stores/admin'
import { useOpsActions } from '../../composables/useOpsActions'
import OpsPanel from '../../components/admin/OpsPanel.vue'

const admin = useAdminStore()
const { loading } = storeToRefs(admin)
const {
  projects,
  opsForm,
  onIssueRowClick,
  onRunRowClick,
  onOpsProjectChange,
  loadOpsIssueRuns,
  loadOpsRunLogs,
  retryOpsIssue,
  cancelOpsRun,
  resetOpsProjectSyncCursor,
  refreshOpsMetrics,
  initOpsPanel,
} = useOpsActions()

onMounted(async () => {
  await initOpsPanel()
})
</script>

<style scoped>
.panel-row {
  margin-top: 4px;
}

.panel-title {
  font-weight: 600;
}
</style>
