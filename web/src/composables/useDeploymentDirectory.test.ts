// @vitest-environment happy-dom
import { effectScope, nextTick, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';
import { projectEnvironmentApi } from '@/api/project/environment';
import {
  initialDeploymentDirectory,
  useDeploymentDirectory,
  validDeploymentDirectory,
} from './useDeploymentDirectory';
import type { EnvironmentResp } from '@/gen/proto/orbit/v1/environment/environment';

vi.mock('@/api/project/environment', () => ({ projectEnvironmentApi: { get: vi.fn() } }));

const snapshot = {
  deployment_directory: '',
  directory_target_revision: 0,
  runtime_directory: '/old/api',
  runtime_target_revision: 1,
};
const environment = {
  target_type: 'ssh',
  target_revision: 2,
  ssh: { platform: 'windows', workspace_root: 'D:/Orbit Workspace' },
} as EnvironmentResp;

describe('deployment directory', () => {
  it('builds a default from the target and restores only a matching confirmed revision', () => {
    expect(initialDeploymentDirectory(environment, snapshot, 'api')).toBe(
      'D:/Orbit Workspace/deployment/api'
    );
    const configured = {
      ...snapshot,
      deployment_directory: 'D:/custom/api',
      directory_target_revision: 2,
    };
    expect(initialDeploymentDirectory(environment, configured, 'api')).toBe('D:/custom/api');
    expect(
      initialDeploymentDirectory({ ...environment, target_revision: 3 }, configured, 'api')
    ).toBe('D:/Orbit Workspace/deployment/api');
  });
  it('accepts target paths outside the workspace and rejects ambiguous relative paths', () => {
    expect(validDeploymentDirectory('/custom/api', 'linux')).toBe(true);
    expect(validDeploymentDirectory('D:\\Custom Space\\api', 'windows')).toBe(true);
    expect(validDeploymentDirectory('~/api', 'linux')).toBe(true);
    expect(validDeploymentDirectory('D:/api', 'linux')).toBe(false);
    expect(validDeploymentDirectory('api', 'windows')).toBe(false);
  });
  it('submits an edited directory without blocking for an existing deployment', async () => {
    vi.mocked(projectEnvironmentApi.get).mockResolvedValue(environment);
    const project = ref<string | null>('project-1');
    const scope = effectScope();
    const form = scope.run(() => useDeploymentDirectory(() => project.value));
    if (!form) {
      throw new Error('Deployment directory scope is inactive');
    }
    try {
      expect(await form.load('project-1', snapshot, 'api')).toBe(true);
      form.state.directory = 'D:/new/api';
      expect(form.validate('invalid')).toBe(true);
      expect(form.payload()).toEqual({
        deployment_directory: 'D:/new/api',
        environment_target_revision: 2,
      });
      project.value = 'project-2';
      await nextTick();
      expect(form.state.ready).toBe(false);
      expect(form.validate('invalid')).toBe(false);
    } finally {
      scope.stop();
    }
  });
  it('discards an environment response after the current project changes', async () => {
    let resolve!: (value: EnvironmentResp) => void;
    vi.mocked(projectEnvironmentApi.get).mockReturnValue(
      new Promise((done) => {
        resolve = done;
      })
    );
    const project = ref<string | null>('project-1');
    const scope = effectScope();
    const form = scope.run(() => useDeploymentDirectory(() => project.value));
    if (!form) {
      throw new Error('Deployment directory scope is inactive');
    }
    try {
      const load = form.load('project-1', snapshot, 'api');
      project.value = 'project-2';
      await nextTick();
      resolve(environment);
      expect(await load).toBe(false);
      expect(form.state.ready).toBe(false);
    } finally {
      scope.stop();
    }
  });
});
