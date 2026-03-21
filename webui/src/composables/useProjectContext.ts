import { computed, watch, type Ref } from 'vue'

interface ProjectLike {
  project_key: string
}

export function useProjectContext(projectKey: Ref<string>, projects: Ref<ProjectLike[]>) {
  const hasProjects = computed(() => projects.value.length > 0)
  const hasProjectSelection = computed(() => !!projectKey.value)

  function ensureProjectSelected() {
    if (projects.value.length === 0) {
      projectKey.value = ''
      return
    }

    const exists = projects.value.some((item) => item.project_key === projectKey.value)
    if (!exists) {
      projectKey.value = projects.value[0].project_key
    }
  }

  watch(projects, ensureProjectSelected, { deep: true })

  return {
    hasProjects,
    hasProjectSelection,
    ensureProjectSelected,
  }
}
