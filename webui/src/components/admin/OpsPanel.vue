<template>
  <div class="form-inline-row mb-12">
    <ProjectSelect
      v-model="opsForm.project_key"
      :projects="projects"
      width="260px"
      @change="emit('projectChange')"
    />
    <el-button :loading="loading" @click="emit('refreshMetrics')">刷新指标</el-button>
    <el-button :loading="loading" @click="emit('resetSync')">重置同步游标</el-button>
    <el-input-number v-model="opsForm.issue_id" :min="1" placeholder="issue id" />
    <el-button :loading="loading" @click="emit('retryIssue')">重试 Issue</el-button>
    <el-button :loading="loading" @click="emit('loadRuns')">加载 Runs</el-button>
    <el-input-number v-model="opsForm.run_id" :min="1" placeholder="run id" />
    <el-button :loading="loading" @click="emit('loadLogs')">加载 Logs</el-button>
  </div>

  <div class="form-inline-row mb-12">
    <el-input
      v-model="opsForm.cancel_reason"
      style="width: 360px"
      placeholder="取消 run 的原因（可选）"
    />
    <el-button type="warning" :loading="loading" @click="emit('cancelRun')">取消 Run</el-button>
  </div>

  <el-row :gutter="16">
    <el-col :xs="24" :lg="8">
      <el-card shadow="never" class="ops-sub-card">
        <template #header>
          <div class="panel-title">项目 Issues</div>
        </template>
        <el-table :data="projectIssues" height="260" @row-click="emit('issueRowClick', $event)">
          <el-table-column label="ID" prop="id" width="80" />
          <el-table-column label="IID" prop="issue_iid" width="90" />
          <el-table-column label="Lifecycle" prop="lifecycle_status" min-width="140" />
        </el-table>
      </el-card>
    </el-col>
    <el-col :xs="24" :lg="8">
      <el-card shadow="never" class="ops-sub-card">
        <template #header>
          <div class="panel-title">Issue Runs</div>
        </template>
        <el-table :data="issueRuns" height="260" @row-click="emit('runRowClick', $event)">
          <el-table-column label="RunID" prop="id" width="90" />
          <el-table-column label="No" prop="run_no" width="70" />
          <el-table-column label="Kind" prop="run_kind" width="90" />
          <el-table-column label="Status" prop="status" min-width="120" />
        </el-table>
      </el-card>
    </el-col>
    <el-col :xs="24" :lg="8">
      <el-card shadow="never" class="ops-sub-card">
        <template #header>
          <div class="panel-title">运行指标</div>
        </template>
        <el-descriptions v-if="metrics" :column="1" size="small" border>
          <el-descriptions-item label="项目">{{ metrics.projects.total }} / enabled {{ metrics.projects.enabled }}</el-descriptions-item>
          <el-descriptions-item label="Issues">{{ metrics.issues.total }}</el-descriptions-item>
          <el-descriptions-item label="Runs">{{ metrics.runs.total }}</el-descriptions-item>
        </el-descriptions>
        <pre v-if="metrics" class="metrics-json">{{ JSON.stringify(metrics, null, 2) }}</pre>
      </el-card>
    </el-col>
  </el-row>

  <el-table :data="runLogs" height="300" class="mt-12">
    <el-table-column label="Seq" prop="seq" width="80" />
    <el-table-column label="At" width="190">
      <template #default="scope">
        {{ formatLocalDateTime(scope.row.at) }}
      </template>
    </el-table-column>
    <el-table-column label="Level" prop="level" width="90" />
    <el-table-column label="Stage" prop="stage" width="120" />
    <el-table-column label="Event" prop="event_type" width="130" />
    <el-table-column label="Message" prop="message" min-width="260" />
  </el-table>
</template>

<script setup lang="ts">
import type { AdminIssueRow, AdminProjectRow, IssueRunRow, OpsMetrics, RunLogRow } from '../../stores/admin'
import ProjectSelect from '../common/ProjectSelect.vue'
import { formatLocalDateTime } from '../../utils/format'

defineProps<{
  projects: AdminProjectRow[]
  projectIssues: AdminIssueRow[]
  issueRuns: IssueRunRow[]
  runLogs: RunLogRow[]
  metrics: OpsMetrics | null
  opsForm: {
    project_key: string
    issue_id: number
    run_id: number
    cancel_reason: string
  }
  loading: boolean
}>()

const emit = defineEmits<{
  projectChange: []
  refreshMetrics: []
  resetSync: []
  retryIssue: []
  loadRuns: []
  loadLogs: []
  cancelRun: []
  issueRowClick: [row: AdminIssueRow]
  runRowClick: [row: IssueRunRow]
}>()
</script>

<style scoped>
.panel-title {
  font-weight: 600;
}

.ops-sub-card {
  margin-bottom: 12px;
}

.form-inline-row {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.metrics-json {
  margin: 12px 0 0;
  padding: 10px;
  max-height: 220px;
  overflow: auto;
  border-radius: 6px;
  background: #f6f8fb;
  border: 1px solid #e3e8f0;
  color: #3b4556;
  font-size: 12px;
  line-height: 1.45;
}

.mb-12 {
  margin-bottom: 12px;
}

.mt-12 {
  margin-top: 12px;
}
</style>
