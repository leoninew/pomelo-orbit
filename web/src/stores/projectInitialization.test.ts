import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { projectInitializationApi } from '@/api/project/initialization';
import { useProjectInitializationStore } from './projectInitialization';

vi.mock('@/api/project/initialization', () => ({
  projectInitializationApi: {
    getStatus: vi.fn(),
  },
}));

const status = {
  status: 'needs_environment',
  defaults: { local_workspace_root: '~/.pomelo-orbit' },
};

describe('projectInitialization store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.mocked(projectInitializationApi.getStatus).mockReset();
    vi.mocked(projectInitializationApi.getStatus).mockResolvedValue(status as never);
  });

  it('reuses a cached status instead of fetching again', async () => {
    const store = useProjectInitializationStore();
    await store.fetchStatus('project-1');
    await store.ensureStatus('project-1');
    expect(projectInitializationApi.getStatus).toHaveBeenCalledTimes(1);
  });

  it('shares an in-flight fetch for the same project', async () => {
    const store = useProjectInitializationStore();
    let resolveStatus: ((value: typeof status) => void) | undefined;
    vi.mocked(projectInitializationApi.getStatus).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveStatus = resolve;
        }) as never
    );

    const first = store.fetchStatus('project-1');
    const second = store.fetchStatus('project-1');
    resolveStatus?.(status);
    await Promise.all([first, second]);

    expect(projectInitializationApi.getStatus).toHaveBeenCalledTimes(1);
  });
});
