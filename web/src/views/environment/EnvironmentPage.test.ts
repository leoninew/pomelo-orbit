// @vitest-environment happy-dom
import { createPinia, setActivePinia } from 'pinia';
import { createApp, nextTick, type App } from 'vue';
import { createMemoryHistory, createRouter, RouterView } from 'vue-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { projectEnvironmentApi } from '@/api/project/environment';
import i18n from '@/i18n';
import { useProjectStore } from '@/stores/project';
import { ApiError } from '@/utils/request';
import EnvironmentPage from './EnvironmentPage.vue';

vi.mock('@/api/project/environment', () => ({
  projectEnvironmentApi: {
    get: vi.fn(),
  },
}));

let mountedApp: App | undefined;
let target: HTMLDivElement | undefined;

async function flushRender() {
  await Promise.resolve();
  await nextTick();
  await Promise.resolve();
  await nextTick();
}

afterEach(() => {
  mountedApp?.unmount();
  target?.remove();
  mountedApp = undefined;
  target = undefined;
  vi.clearAllMocks();
});

describe('Environment page', () => {
  it('redirects to project initialization when the environment is not found', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useProjectStore().setActiveProject('project-1');
    vi.mocked(projectEnvironmentApi.get).mockRejectedValue(
      new ApiError('Project environment not found', 404, 'not_found', 'request-1')
    );

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/environment', component: EnvironmentPage },
        {
          path: '/project/:id/initialization',
          name: 'ProjectInitialization',
          component: { template: '<div>Project initialization</div>' },
        },
      ],
    });

    await router.push('/environment');
    await router.isReady();
    target = document.createElement('div');
    document.body.append(target);
    mountedApp = createApp(RouterView);
    mountedApp.use(pinia);
    mountedApp.use(router);
    mountedApp.use(i18n);
    mountedApp.mount(target);

    await vi.waitFor(() => {
      expect(router.currentRoute.value).toMatchObject({
        name: 'ProjectInitialization',
        params: { id: 'project-1' },
        query: { redirect: '/environment' },
      });
    });
    await flushRender();

    expect(projectEnvironmentApi.get).toHaveBeenCalledWith('project-1');
    expect(target.textContent).not.toContain('Project environment not found');
  });
});
