import { storeToRefs } from 'pinia'
import { reactive, toRef } from 'vue'
import { useAdminStore, type AdminIssueRow, type IssueRunRow } from '../stores/admin'
import { useSessionStore } from '../stores/session'
import { useProjectContext } from './useProjectContext'

export function useOpsActions() {
  const session = useSessionStore()
  const admin = useAdminStore()
  const { projects } = storeToRefs(admin)

  const opsForm = reactive({
    project_key: '',
    issue_id: 0,
    run_id: 0,
    cancel_reason: 'manual cancel',
  })

  const { ensureProjectSelected } = useProjectContext(toRef(opsForm, 'project_key'), projects)

  function onIssueRowClick(row: AdminIssueRow | null) {
    opsForm.issue_id = Number(row?.id ?? 0)
  }

  function onRunRowClick(row: IssueRunRow | null) {
    opsForm.run_id = row?.id ?? 0
  }

  async function onOpsProjectChange() {
    if (!session.token || !opsForm.project_key) {
      return
    }
    await admin.fetchProjectIssues(session.token, opsForm.project_key, 200)
  }

  async function loadOpsIssueRuns() {
    if (!session.token || !opsForm.issue_id) {
      ElMessage.warning('请先选择 Issue')
      return
    }
    await admin.fetchIssueRuns(session.token, opsForm.issue_id, 200)
  }

  async function loadOpsRunLogs() {
    if (!session.token || !opsForm.run_id) {
      ElMessage.warning('请先选择 Run')
      return
    }
    await admin.fetchRunLogs(session.token, opsForm.run_id, 1000)
  }

  async function retryOpsIssue() {
    if (!session.token || !opsForm.issue_id) {
      ElMessage.warning('请先选择 Issue')
      return
    }
    await admin.retryIssue(session.token, opsForm.issue_id)
    ElMessage.success('Issue 已重试入队')
    if (!opsForm.project_key) {
      return
    }
    await admin.fetchProjectIssues(session.token, opsForm.project_key, 200)
  }

  async function cancelOpsRun() {
    if (!session.token || !opsForm.run_id) {
      ElMessage.warning('请先选择 Run')
      return
    }
    await admin.cancelRun(session.token, opsForm.run_id, opsForm.cancel_reason)
    ElMessage.success('Run 已取消')
    if (opsForm.issue_id) {
      await admin.fetchIssueRuns(session.token, opsForm.issue_id, 200)
    }
  }

  async function resetOpsProjectSyncCursor() {
    if (!session.token || !opsForm.project_key) {
      ElMessage.warning('请先选择项目')
      return
    }
    await admin.resetProjectSyncCursor(session.token, opsForm.project_key)
    ElMessage.success('项目同步游标已重置')
  }

  async function refreshOpsMetrics() {
    if (!session.token) {
      return
    }
    await admin.fetchMetrics(session.token)
  }

  async function initOpsPanel() {
    if (!session.token) {
      return
    }

    await admin.fetchProjects(session.token)
    await admin.fetchMetrics(session.token)
    ensureProjectSelected()
    if (opsForm.project_key) {
      await admin.fetchProjectIssues(session.token, opsForm.project_key, 200)
    }
  }

  return {
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
  }
}
