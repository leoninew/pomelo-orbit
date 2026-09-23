import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { projectApi } from '@/api/project/project';
import { ACTIVE_PROJECT_ID_KEY, ACTIVE_PROJECT_SUMMARY_KEY } from '@/constants/project';
import { useStorageStore } from '@/stores/storage';
import type {
  ProjectCreateReq,
  ProjectResp,
  ProjectSaveReq,
} from '@/gen/proto/orbit/v1/project/project';

type ProjectSummary = Pick<ProjectResp, 'id' | 'name'>;

export const useProjectStore = defineStore('project', () => {
  const storageStore = useStorageStore();
  const projects = ref<ProjectResp[]>([]);
  const activeProjectId = ref<string | null>(storageStore.getItem<string>(ACTIVE_PROJECT_ID_KEY));
  const activeProjectSummary = ref<ProjectSummary | null>(
    storageStore.getItem<ProjectSummary>(ACTIVE_PROJECT_SUMMARY_KEY)
  );
  const loading = ref(false);

  const activeProject = computed(() =>
    projects.value.find((project) => project.id === activeProjectId.value)
  );
  const activeProjectName = computed(
    () =>
      activeProject.value?.name ??
      (activeProjectSummary.value?.id === activeProjectId.value
        ? activeProjectSummary.value.name
        : null)
  );
  const activeProjects = computed(() => projects.value.filter((project) => project.is_active));

  function setActiveProject(project_id: string) {
    activeProjectId.value = project_id;
    storageStore.setItem(ACTIVE_PROJECT_ID_KEY, project_id);
    const project = projects.value.find((item) => item.id === project_id);
    activeProjectSummary.value = project ? { id: project.id, name: project.name } : null;
    if (activeProjectSummary.value) {
      storageStore.setItem(ACTIVE_PROJECT_SUMMARY_KEY, activeProjectSummary.value);
    } else {
      storageStore.removeItem(ACTIVE_PROJECT_SUMMARY_KEY);
    }
  }

  function clearProjects() {
    projects.value = [];
    activeProjectId.value = null;
    activeProjectSummary.value = null;
    storageStore.removeItem(ACTIVE_PROJECT_ID_KEY);
    storageStore.removeItem(ACTIVE_PROJECT_SUMMARY_KEY);
  }

  function selectFallbackProject(items: ProjectResp[]) {
    const selectableProjects = items.filter((project) => project.is_active);
    if (selectableProjects.length === 0) {
      clearProjects();
      return;
    }
    if (
      activeProjectId.value &&
      selectableProjects.some((project) => project.id === activeProjectId.value)
    ) {
      setActiveProject(activeProjectId.value);
      return;
    }
    setActiveProject(selectableProjects[0].id);
  }

  async function fetchProjects() {
    loading.value = true;
    try {
      const resp = await projectApi.list();
      projects.value = resp.items;
      selectFallbackProject(resp.items);
      return resp.items;
    } finally {
      loading.value = false;
    }
  }

  async function createProject(data: ProjectCreateReq) {
    const project = await projectApi.create(data);
    await fetchProjects();
    return project;
  }

  async function updateProject(id: string, data: ProjectSaveReq) {
    const project = await projectApi.update(id, data);
    await fetchProjects();
    return project;
  }

  async function deprecateProject(id: string) {
    await projectApi.deprecate(id, {});
    await fetchProjects();
  }

  return {
    projects,
    activeProjectId,
    activeProject,
    activeProjectName,
    activeProjects,
    loading,
    fetchProjects,
    setActiveProject,
    clearProjects,
    createProject,
    updateProject,
    deprecateProject,
  };
});
