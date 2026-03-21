import { storeToRefs } from 'pinia'
import { computed, reactive, toRef } from 'vue'
import { useAdminStore, type PromptTemplate } from '../stores/admin'
import { useSessionStore } from '../stores/session'
import { useProjectContext } from './useProjectContext'

export function usePromptEditor() {
  const session = useSessionStore()
  const admin = useAdminStore()
  const { projects, projectPrompts, defaultPrompts } = storeToRefs(admin)

  const promptRunKindOptions = ['dev', 'merge']
  const promptForm = reactive({
    project_key: '',
    run_kind: 'dev',
    agent_role: 'dev',
    content: '',
  })

  const { ensureProjectSelected } = useProjectContext(toRef(promptForm, 'project_key'), projects)

  const promptAgentRoleOptions = computed(() => {
    if (promptForm.run_kind === 'merge') {
      return ['merge', 'review']
    }
    return ['dev', 'review']
  })

  const activePrompt = computed(() =>
    projectPrompts.value.find(
      (item) => item.run_kind === promptForm.run_kind && item.agent_role === promptForm.agent_role,
    ),
  )
  const promptHasOverride = computed(() => activePrompt.value?.source === 'project_override')

  function ensurePromptRole() {
    if (!promptAgentRoleOptions.value.includes(promptForm.agent_role)) {
      promptForm.agent_role = promptAgentRoleOptions.value[0]
    }
  }

  function onPromptRunKindChange() {
    ensurePromptRole()
  }

  function fillPromptFromTemplate(row: PromptTemplate) {
    promptForm.run_kind = row.run_kind
    promptForm.agent_role = row.agent_role
    promptForm.content = row.content
    ensurePromptRole()
  }

  async function loadProjectPrompts() {
    if (!session.token || !promptForm.project_key) {
      admin.projectPrompts = []
      return
    }

    await admin.fetchProjectPrompts(session.token, promptForm.project_key)
    const exact = admin.projectPrompts.find(
      (item) => item.run_kind === promptForm.run_kind && item.agent_role === promptForm.agent_role,
    )

    if (exact) {
      promptForm.content = exact.content
      return
    }

    if (admin.projectPrompts.length > 0) {
      fillPromptFromTemplate(admin.projectPrompts[0])
      return
    }

    promptForm.content = ''
  }

  async function savePromptOverride() {
    if (!session.token || !promptForm.project_key) {
      ElMessage.warning('请先选择项目')
      return
    }
    if (!promptForm.content.trim()) {
      ElMessage.warning('Prompt 内容不能为空')
      return
    }

    await admin.upsertProjectPrompt(
      session.token,
      promptForm.project_key,
      promptForm.run_kind,
      promptForm.agent_role,
      promptForm.content,
    )
    await loadProjectPrompts()
    ElMessage.success('Prompt 覆盖已保存')
  }

  async function resetPromptOverride() {
    if (!session.token || !promptForm.project_key) {
      ElMessage.warning('请先选择项目')
      return
    }
    if (!promptHasOverride.value) {
      ElMessage.warning('当前组合没有项目覆盖，已是默认模板')
      return
    }

    await admin.deleteProjectPrompt(
      session.token,
      promptForm.project_key,
      promptForm.run_kind,
      promptForm.agent_role,
    )
    await loadProjectPrompts()
    ElMessage.success('已回退到默认模板')
  }

  async function initPromptEditor() {
    if (!session.token) {
      return
    }

    await admin.fetchProjects(session.token)
    await admin.fetchDefaultPrompts(session.token)
    ensureProjectSelected()
    await loadProjectPrompts()
  }

  return {
    projects,
    projectPrompts,
    defaultPrompts,
    promptRunKindOptions,
    promptForm,
    promptAgentRoleOptions,
    promptHasOverride,
    fillPromptFromTemplate,
    onPromptRunKindChange,
    loadProjectPrompts,
    savePromptOverride,
    resetPromptOverride,
    initPromptEditor,
  }
}
