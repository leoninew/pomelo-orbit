import { defineStore } from 'pinia';
import { ref } from 'vue';
import { projectInitializationApi } from '@/api/project/initialization';
import type {
  ProjectInitializationBootstrapReq,
  ProjectInitializationEnvironmentReq,
  ProjectInitializationStatusResp,
} from '@/gen/proto/orbit/v1/project_initialization/project_initialization';

export const READY_INITIALIZATION_STATUS = 'ready';

export const useProjectInitializationStore = defineStore('projectInitialization', () => {
  const statuses = ref<Record<string, ProjectInitializationStatusResp>>({});
  const loading = ref(false);
  const inflight = new Map<string, Promise<ProjectInitializationStatusResp>>();

  function statusFor(projectId: string) {
    return statuses.value[projectId];
  }

  function clear(projectId?: string) {
    if (!projectId) {
      statuses.value = {};
      inflight.clear();
      return;
    }
    delete statuses.value[projectId];
    inflight.delete(projectId);
  }

  async function fetchStatus(projectId: string) {
    const pending = inflight.get(projectId);
    if (pending) {
      return pending;
    }
    loading.value = true;
    const request = projectInitializationApi
      .getStatus(projectId)
      .then((status) => {
        statuses.value = { ...statuses.value, [projectId]: status };
        return status;
      })
      .finally(() => {
        inflight.delete(projectId);
        loading.value = false;
      });
    inflight.set(projectId, request);
    return request;
  }

  async function ensureStatus(projectId: string) {
    return statuses.value[projectId] ?? fetchStatus(projectId);
  }

  async function testEnvironment(projectId: string, input: ProjectInitializationEnvironmentReq) {
    return projectInitializationApi.testEnvironment(projectId, input);
  }

  async function saveEnvironment(projectId: string, input: ProjectInitializationEnvironmentReq) {
    const status = await projectInitializationApi.saveEnvironment(projectId, input);
    statuses.value = { ...statuses.value, [projectId]: status };
    return status;
  }

  async function getDeploymentPublicKey(projectId: string) {
    return projectInitializationApi.getDeploymentPublicKey(projectId);
  }

  async function bootstrapEnvironment(projectId: string, input: ProjectInitializationBootstrapReq) {
    const status = await projectInitializationApi.bootstrapEnvironment(projectId, input);
    statuses.value = { ...statuses.value, [projectId]: status };
    return status;
  }

  async function probeEnvironment(projectId: string) {
    const status = await projectInitializationApi.probeEnvironment(projectId);
    statuses.value = { ...statuses.value, [projectId]: status };
    return status;
  }

  async function createGateway(
    projectId: string,
    input: Parameters<typeof projectInitializationApi.createGateway>[1]
  ) {
    const status = await projectInitializationApi.createGateway(projectId, input);
    statuses.value = { ...statuses.value, [projectId]: status };
    return status;
  }

  return {
    statuses,
    loading,
    statusFor,
    clear,
    fetchStatus,
    ensureStatus,
    testEnvironment,
    saveEnvironment,
    getDeploymentPublicKey,
    bootstrapEnvironment,
    probeEnvironment,
    createGateway,
  };
});
