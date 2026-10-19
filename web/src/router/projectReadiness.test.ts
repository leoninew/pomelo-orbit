import type { Router } from 'vue-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { projectInitializationApi } from '@/api/project/initialization';
import type { ProjectInitializationStatusResp } from '@/gen/proto/orbit/v1/project_initialization/project_initialization';
import {
  DEFAULT_READY_REDIRECT,
  ensureProjectExecutionReady,
  isExecutionEnvironmentReady,
  isProjectInitializationPath,
  resolveInitializationCompletionRedirect,
} from './projectReadiness';

vi.mock('@/api/project/initialization', () => ({
  projectInitializationApi: { getStatus: vi.fn() },
}));

afterEach(() => vi.clearAllMocks());

describe('project readiness routing', () => {
  it('recognizes the wizard path', () => {
    expect(isProjectInitializationPath('/project/abc/initialization')).toBe(true);
    expect(isProjectInitializationPath('/gateway')).toBe(false);
  });

  it('opens the initialized gateway detail after setup completes', () => {
    expect(resolveInitializationCompletionRedirect()).toBe(DEFAULT_READY_REDIRECT);
  });

  it('requires an environment for CI and also a gateway for CD, regardless of probe history', () => {
    const unconfigured = { status: 'needs_environment' } as ProjectInitializationStatusResp;
    expect(isExecutionEnvironmentReady(unconfigured, 'ci')).toBe(false);
    expect(isExecutionEnvironmentReady(unconfigured, 'cd')).toBe(false);
    const configured = {
      status: 'needs_gateway',
      environment: { last_probe_status: 'failed' },
    } as ProjectInitializationStatusResp;
    expect(isExecutionEnvironmentReady(configured, 'ci')).toBe(true);
    expect(isExecutionEnvironmentReady(configured, 'cd')).toBe(false);
    const ready = { status: 'ready' } as ProjectInitializationStatusResp;
    expect(isExecutionEnvironmentReady(ready, 'ci')).toBe(true);
    expect(isExecutionEnvironmentReady(ready, 'cd')).toBe(true);
  });

  it('checks live status before every execution and redirects the correct project if unready', async () => {
    const router = {
      currentRoute: { value: { fullPath: '/pipeline' } },
      push: vi.fn().mockResolvedValue(undefined),
    } as unknown as Router;
    vi.mocked(projectInitializationApi.getStatus)
      .mockResolvedValueOnce({ status: 'needs_environment' } as ProjectInitializationStatusResp)
      .mockResolvedValueOnce({ status: 'needs_gateway' } as ProjectInitializationStatusResp);

    expect(await ensureProjectExecutionReady('project-a', router, 'ci')).toBe(false);
    expect(router.push).toHaveBeenCalledWith({
      name: 'ProjectInitialization',
      params: { id: 'project-a' },
      query: { purpose: 'ci', returnTo: '/pipeline' },
    });
    expect(await ensureProjectExecutionReady('project-a', router, 'ci')).toBe(true);
    expect(projectInitializationApi.getStatus).toHaveBeenCalledTimes(2);
    expect(router.push).toHaveBeenCalledOnce();
  });

  it('redirects CD to initialization when the gateway is missing', async () => {
    const router = {
      currentRoute: { value: { fullPath: '/service/42' } },
      push: vi.fn().mockResolvedValue(undefined),
    } as unknown as Router;
    vi.mocked(projectInitializationApi.getStatus).mockResolvedValue({
      status: 'needs_gateway',
    } as ProjectInitializationStatusResp);

    expect(await ensureProjectExecutionReady('project-b', router, 'cd')).toBe(false);
    expect(router.push).toHaveBeenCalledWith({
      name: 'ProjectInitialization',
      params: { id: 'project-b' },
      query: { purpose: 'cd', returnTo: '/service/42' },
    });
  });

  it('accepts only internal return routes outside the initialization wizard', () => {
    expect(resolveInitializationCompletionRedirect('/pipeline-run/42')).toBe('/pipeline-run/42');
    expect(resolveInitializationCompletionRedirect('//external.test')).toBe(DEFAULT_READY_REDIRECT);
    expect(resolveInitializationCompletionRedirect('https://external.test')).toBe(
      DEFAULT_READY_REDIRECT
    );
    expect(resolveInitializationCompletionRedirect('/project/a/initialization')).toBe(
      DEFAULT_READY_REDIRECT
    );
  });
});
