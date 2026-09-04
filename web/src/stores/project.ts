import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { projectApi } from '@/api/project/project';
import { ACTIVE_PROJECT_ID_KEY } from '@/constants/project';
import { useStorageStore } from '@/stores/storage';
import type {
  ProjectCreateReq,
  ProjectResp,
  ProjectSaveReq,
} from '@/gen/proto/orbit/v1/project/project';

export const useProjectStore = defineStore('project', () => {
  const storageStore = useStorageStore();
  const projects = ref<ProjectResp[]>([]);
  const activeProjectId = ref<string | null>(storageStore.getItem<string>(ACTIVE_PROJECT_ID_KEY));
  const loading = ref(false);

  const activeProject = computed(() =>
    projects.value.find((project) => project.id === activeProjectId.value)
  );

  function setActiveProject(project_id: string) {
    activeProjectId.value = project_id;
    storageStore.setItem(ACTIVE_PROJECT_ID_KEY, project_id);
  }

  function clearProjects() {
    projects.value = [];
    activeProjectId.value = null;
    storageStore.removeItem(ACTIVE_PROJECT_ID_KEY);
  }

  function selectFallbackProject(items: ProjectResp[]) {
    if (items.length === 0) {
      clearProjects();
      return;
    }
    if (activeProjectId.value && items.some((project) => project.id === activeProjectId.value)) {
      return;
    }
    setActiveProject(items[0].id);
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
    loading,
    fetchProjects,
    setActiveProject,
    clearProjects,
    createProject,
    updateProject,
    deprecateProject,
  };
});
