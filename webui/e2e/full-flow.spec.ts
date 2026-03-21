import { readFile } from 'node:fs/promises'
import { expect, test } from '@playwright/test'
import {
  closeGitLabIssue,
  createGitLabIssue,
  listAdminProjectIssues,
  listAdminProjects,
  listGitLabIssueNotes,
  listIssueRuns,
  listRunLogs,
  loginAdmin,
  loginFromUI,
  pollUntil,
  readRuntimeState,
  selectProjectByKey,
  setElSwitchState,
  type AdminIssueItem,
  type AdminRunItem,
} from './helpers'

test.describe.configure({ mode: 'serial' })

async function readTail(filePath: string, maxChars = 4_000) {
  try {
    const text = await readFile(filePath, 'utf-8')
    if (text.length <= maxChars) {
      return text
    }
    return text.slice(text.length - maxChars)
  } catch {
    return ''
  }
}

test('webui + gitlab 全链路流程', async ({ page, request }) => {
  const state = await readRuntimeState()
  const adminToken = await loginAdmin(request, state)

  let issueIID: number | null = null

  try {
    const projectsBefore = await listAdminProjects(request, state, adminToken)
    expect(projectsBefore.items.some((it) => it.project_key === state.e2eProjectKey)).toBeTruthy()

    await loginFromUI(page, state)
    await expect(page).toHaveURL(/\/board/)
    await expect(page.getByText('Agent Coder')).toBeVisible()

    await page.goto(`${state.baseURL}/admin/projects`)
    await expect(page.getByText('项目编辑')).toBeVisible()

    const projectRow = page.locator('.el-table__body tr').filter({ hasText: state.e2eProjectKey }).first()
    await expect(projectRow).toBeVisible()
    await projectRow.click()
    await expect(page.getByText(`当前编辑: ${state.e2eProjectKey}`)).toBeVisible()

    const sandboxSwitch = page
      .locator('.project-form .el-form-item')
      .filter({ hasText: 'sandbox_plan_review' })
      .locator('.el-switch')
      .first()
    await setElSwitchState(sandboxSwitch, true)
    await page.getByRole('button', { name: '更新项目' }).click()
    await expect(page.locator('.el-message').filter({ hasText: '项目已更新' })).toBeVisible()

    const projectsWithSandboxOn = await listAdminProjects(request, state, adminToken)
    const projectSandboxOn = projectsWithSandboxOn.items.find((it) => it.project_key === state.e2eProjectKey)
    expect(projectSandboxOn?.sandbox_plan_review).toBe(true)

    await setElSwitchState(sandboxSwitch, false)
    await page.getByRole('button', { name: '更新项目' }).click()
    await expect(page.locator('.el-message').filter({ hasText: '项目已更新' })).toBeVisible()

    const projectsWithSandboxOff = await listAdminProjects(request, state, adminToken)
    const projectSandboxOff = projectsWithSandboxOff.items.find((it) => it.project_key === state.e2eProjectKey)
    expect(projectSandboxOff?.sandbox_plan_review).toBe(false)

    await page.goto(`${state.baseURL}/admin/users`)
    await expect(page.getByText('用户管理')).toBeVisible()

    const newUsername = `e2e-ui-${Date.now()}`
    await page.getByPlaceholder('用户名').fill(newUsername)
    await page.getByPlaceholder('密码').fill('pass123456')
    await page.getByRole('button', { name: '创建用户' }).click()
    await expect(page.locator('.el-message').filter({ hasText: '用户已创建' })).toBeVisible()

    const userRow = page.locator('.el-table__body tr').filter({ hasText: newUsername }).first()
    await expect(userRow).toBeVisible()
    await userRow.getByRole('button', { name: '设为管理员' }).click()
    await expect(userRow.getByRole('button', { name: '取消管理员' })).toBeVisible()

    await userRow.getByRole('button', { name: '禁用' }).click()
    await expect(userRow.getByRole('button', { name: '启用' })).toBeVisible()

    await page.goto(`${state.baseURL}/admin/prompts`)
    await expect(page.getByText('Prompt 管理')).toBeVisible()

    const promptProjectSelect = page.locator('.form-inline-row').first().locator('.el-select').first()
    await selectProjectByKey(promptProjectSelect, state.e2eProjectKey)

    await page.getByRole('button', { name: '加载模板' }).click()
    const promptRows = page.locator('.el-table').first().locator('.el-table__body tr')
    await expect(promptRows.first()).toBeVisible()

    const promptTextarea = page.locator('.el-textarea__inner').first()
    const originPrompt = await promptTextarea.inputValue()
    await promptTextarea.fill(`${originPrompt}\n<!-- e2e-marker-${Date.now()} -->`)
    await page.getByRole('button', { name: '保存覆盖' }).click()
    await expect(page.locator('.el-message').filter({ hasText: 'Prompt 覆盖已保存' })).toBeVisible()

    await page.getByRole('button', { name: '回退默认' }).click()
    await expect(page.locator('.el-message').filter({ hasText: '已回退到默认模板' })).toBeVisible()

    const marker = `playwright-e2e-${Date.now()}`
    const gitlabIssue = await createGitLabIssue(
      state,
      `[E2E] ${marker}`,
      `temporary issue for webui full flow (${marker})`,
      ['Agent Ready', marker],
    )
    issueIID = gitlabIssue.iid

    await page.goto(`${state.baseURL}/board`)

    const boardProjectRow = page
      .locator('.panel')
      .first()
      .locator('.el-table__body tr')
      .filter({ hasText: state.e2eProjectKey })
      .first()
    await expect(boardProjectRow).toBeVisible()
    await boardProjectRow.click()

    const localIssue = (await pollUntil(
      async () => {
        const resp = await listAdminProjectIssues(request, state, adminToken, state.e2eProjectKey)
        return resp.items.find((it) => it.issue_iid === gitlabIssue.iid) ?? null
      },
      (value) => value !== null,
      {
        timeoutMs: 5 * 60_000,
        intervalMs: 5_000,
        description: 'issue synced from gitlab',
      },
    )) as AdminIssueItem

    await page.goto(`${state.baseURL}/board`)
    const boardProjectRowAfterSync = page
      .locator('.panel')
      .first()
      .locator('.el-table__body tr')
      .filter({ hasText: state.e2eProjectKey })
      .first()
    await expect(boardProjectRowAfterSync).toBeVisible()
    await boardProjectRowAfterSync.click()
    await expect(
      page
        .locator('.panel')
        .nth(1)
        .locator('.el-table__body tr')
        .filter({ hasText: marker })
        .first(),
    ).toBeVisible({ timeout: 60_000 })

    const firstRun = (await pollUntil(
      async () => {
        const resp = await listIssueRuns(request, state, adminToken, localIssue.id)
        return resp.items[0] ?? null
      },
      (value) => value !== null,
      {
        timeoutMs: 10 * 60_000,
        intervalMs: 8_000,
        description: 'first run created',
      },
    )) as AdminRunItem

    const finalRun = (await pollUntil(
      async () => {
        const resp = await listIssueRuns(request, state, adminToken, localIssue.id)
        return resp.items.find((it) => it.id === firstRun.id) ?? null
      },
      (value) => value !== null && ['succeeded', 'failed', 'canceled'].includes(value.status),
      {
        timeoutMs: 16 * 60_000,
        intervalMs: 10_000,
        description: 'run reaches terminal status',
      },
    )) as AdminRunItem

    const runLogsResp = await listRunLogs(request, state, adminToken, firstRun.id)

    if (finalRun.status !== 'succeeded') {
      const workerTail = await readTail(state.workerLogPath)
      const serverTail = await readTail(state.serverLogPath)
      const logTail = runLogsResp.items
        .slice(-10)
        .map((it) => `[${it.seq}] ${it.message}`)
        .join('\n')

      throw new Error(
        [
          `auto dev run failed: run_id=${firstRun.id} status=${finalRun.status} error=${finalRun.error_summary ?? ''}`,
          `run logs:\n${logTail}`,
          `worker log tail:\n${workerTail}`,
          `server log tail:\n${serverTail}`,
        ].join('\n\n'),
      )
    }

    const issueAfter = (await pollUntil(
      async () => {
        const resp = await listAdminProjectIssues(request, state, adminToken, state.e2eProjectKey)
        return resp.items.find((it) => it.id === localIssue.id) ?? null
      },
      (value) => value !== null && ['human_review', 'closed'].includes(value.lifecycle_status),
      {
        timeoutMs: 2 * 60_000,
        intervalMs: 5_000,
        description: 'issue lifecycle after dev run',
      },
    )) as AdminIssueItem

    expect(issueAfter.lifecycle_status).toBe('human_review')
    expect((issueAfter.mr_iid ?? 0) > 0).toBeTruthy()
    expect(runLogsResp.items.length).toBeGreaterThan(0)

    const notes = await pollUntil(
      async () => listGitLabIssueNotes(state, gitlabIssue.iid),
      (rows) => rows.some((note) => note.body.includes('MR is ready for human review')),
      {
        timeoutMs: 2 * 60_000,
        intervalMs: 5_000,
        description: 'issue note generated by worker',
      },
    )
    expect(notes.some((note) => note.body.includes('MR is ready for human review'))).toBeTruthy()

    await page.goto(`${state.baseURL}/admin/ops`)
    await expect(page.getByText('运维与日志')).toBeVisible()

    const opsProjectSelect = page.locator('.form-inline-row').first().locator('.el-select').first()
    await selectProjectByKey(opsProjectSelect, state.e2eProjectKey)

    const issuesCard = page.locator('.ops-sub-card').filter({ hasText: '项目 Issues' }).first()
    const issueRow = issuesCard.locator('.el-table__body tr').filter({ hasText: String(gitlabIssue.iid) }).first()
    await expect(issueRow).toBeVisible({ timeout: 2 * 60_000 })
    await issueRow.click()

    await page.getByRole('button', { name: '加载 Runs' }).click()

    const runsCard = page.locator('.ops-sub-card').filter({ hasText: 'Issue Runs' }).first()
    const runRow = runsCard.locator('.el-table__body tr').filter({ hasText: String(firstRun.id) }).first()
    await expect(runRow).toBeVisible()
    await expect(runRow).toContainText('succeeded')
    await runRow.click()

    await page.getByRole('button', { name: '加载 Logs' }).click()
    const logsTable = page.locator('.el-table').filter({ hasText: 'Message' }).first()
    await expect(logsTable.locator('.el-table__body tr').first()).toBeVisible()
  } finally {
    if (issueIID !== null) {
      try {
        await closeGitLabIssue(state, issueIID)
      } catch {
        // ignore cleanup failure
      }
    }
  }
})
