import { reactive, watch } from 'vue';
import { projectEnvironmentApi } from '@/api/project/environment';
import type { EnvironmentResp } from '@/gen/proto/orbit/v1/environment/environment';

export interface DeploymentDirectorySnapshot {
  deployment_directory: string;
  directory_target_revision: number;
  runtime_directory: string;
  runtime_target_revision: number;
}

export function normalizeDeploymentDirectory(value: string, platform: string) {
  const path = value.trim().replace(/\\/g, '/');
  const prefix = path.startsWith('//') ? '//' : path.startsWith('/') ? '/' : '';
  const normalized =
    prefix +
    path
      .split('/')
      .filter((part) => part && part !== '.')
      .join('/');
  return platform === 'windows' ? normalized.toLowerCase() : normalized;
}

export function validDeploymentDirectory(value: string, platform: string) {
  const directory = value.trim();
  if (
    !directory ||
    directory.length > 2048 ||
    directory.includes('\0') ||
    /[\r\n]/.test(directory)
  ) {
    return false;
  }
  const path = platform === 'windows' ? directory.replace(/\\/g, '/') : directory;
  if (path.includes('\\') || path.split('/').includes('..')) {
    return false;
  }
  const normalized = normalizeDeploymentDirectory(path, platform);
  if (normalized === '/' || /^[a-z]:$/i.test(normalized)) {
    return false;
  }
  return (
    path === '~' ||
    path.startsWith('~/') ||
    (platform === 'windows'
      ? /^[a-z]:\//i.test(path) || /^\/\/[^/]+\/[^/]+\//.test(path)
      : path.startsWith('/'))
  );
}

export function initialDeploymentDirectory(
  environment: EnvironmentResp,
  snapshot: DeploymentDirectorySnapshot,
  code: string
) {
  if (
    snapshot.deployment_directory &&
    snapshot.directory_target_revision === environment.target_revision
  ) {
    return snapshot.deployment_directory;
  }
  const root =
    environment.target_type === 'ssh'
      ? environment.ssh?.workspace_root
      : environment.local?.workspace_root;
  if (!root) {
    throw new Error('Environment workspace is unavailable');
  }
  return root.replace(/\\/g, '/').replace(/\/+$/, '') + '/deployment/' + code;
}

export function useDeploymentDirectory(currentProjectId: () => string | null) {
  const state = reactive({
    directory: '',
    platform: '',
    runtimeDirectory: '',
    error: '',
    targetRevision: 0,
    projectId: '',
    ready: false,
  });
  let sequence = 0;
  function reset() {
    sequence++;
    Object.assign(state, {
      directory: '',
      runtimeDirectory: '',
      error: '',
      targetRevision: 0,
      projectId: '',
      ready: false,
    });
  }
  watch(currentProjectId, reset);
  async function load(projectId: string, snapshot: DeploymentDirectorySnapshot, code: string) {
    reset();
    const request = sequence;
    const environment = await projectEnvironmentApi.get(projectId);
    if (request !== sequence || currentProjectId() !== projectId) {
      return false;
    }
    Object.assign(state, {
      directory: initialDeploymentDirectory(environment, snapshot, code),
      platform:
        environment.target_type === 'ssh'
          ? (environment.ssh?.platform ?? '')
          : (environment.local?.platform ?? ''),
      runtimeDirectory: snapshot.runtime_directory,
      targetRevision: environment.target_revision,
      projectId,
      ready: true,
    });
    return true;
  }
  function validate(message: string) {
    state.error = validDeploymentDirectory(state.directory, state.platform) ? '' : message;
    return state.ready && state.projectId === currentProjectId() && !state.error;
  }
  function payload() {
    return {
      deployment_directory: state.directory.trim(),
      environment_target_revision: state.targetRevision,
    };
  }
  return { state, load, validate, payload, reset };
}
