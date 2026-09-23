import { createMemoryHistory, createRouter } from 'vue-router';
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

  it('only requires a fresh environment probe for CI, but also requires a gateway for CD', () => {
    for (const status of ['needs_environment', 'needs_probe']) {
      const view = { status } as ProjectInitializationStatusResp;
      expect(isExecutionEnvironmentReady(view, 'ci')).toBe(false);
      expect(isExecutionEnvironmentReady(view, 'cd')).toBe(false);
    }
    const probed = { status: 'needs_gateway' } as ProjectInitializationStatusResp;
    expect(isExecutionEnvironmentReady(probed, 'ci')).toBe(true);
    expect(isExecutionEnvironmentReady(probed, 'cd')).toBe(false);
    const ready = { status: 'ready' } as ProjectInitializationStatusResp;
    expect(isExecutionEnvironmentReady(ready, 'ci')).toBe(true);
    expect(isExecutionEnvironmentReady(ready, 'cd')).toBe(true);
  });

  it('checks live status before every execution and redirects the correct project if unready', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/pipeline', component: {} },
        { path: '/gateway', component: {} },
        { path: '/project/:id/initialization', name: 'ProjectInitialization', component: {} },
      ],
    });
    await router.push('/pipeline');
    await router.isReady();
    vi.mocked(projectInitializationApi.getStatus)
      .mockResolvedValueOnce({ status: 'needs_probe' } as ProjectInitializationStatusResp)
      .mockResolvedValueOnce({ status: 'needs_gateway' } as ProjectInitializationStatusResp);

    expect(await ensureProjectExecutionReady('project-a', router, 'ci')).toBe(false);
    expect(router.currentRoute.value).toMatchObject({
      name: 'ProjectInitialization',
      params: { id: 'project-a' },
      query: { purpose: 'ci', returnTo: '/pipeline' },
    });
    expect(await ensureProjectExecutionReady('project-a', router, 'ci')).toBe(true);
    expect(projectInitializationApi.getStatus).toHaveBeenCalledTimes(2);
  });

  it('redirects CD to initialization when the environment probe passed but gateway is missing', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/service/42', component: {} },
        { path: '/project/:id/initialization', name: 'ProjectInitialization', component: {} },
      ],
    });
    await router.push('/service/42');
    await router.isReady();
    vi.mocked(projectInitializationApi.getStatus).mockResolvedValue({
      status: 'needs_gateway',
    } as ProjectInitializationStatusResp);

    expect(await ensureProjectExecutionReady('project-b', router, 'cd')).toBe(false);
    expect(router.currentRoute.value).toMatchObject({
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
