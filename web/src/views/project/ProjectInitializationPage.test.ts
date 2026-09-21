// @vitest-environment happy-dom
import { createPinia, setActivePinia } from 'pinia';
import { createApp, nextTick, type App } from 'vue';
import { createMemoryHistory, createRouter, RouterView } from 'vue-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { projectInitializationApi } from '@/api/project/initialization';
import i18n from '@/i18n';
import { useProjectStore } from '@/stores/project';
import { ApiError } from '@/utils/request';
import ProjectInitializationPage from './ProjectInitializationPage.vue';

vi.mock('@/api/project/initialization', () => ({
  projectInitializationApi: {
    getStatus: vi.fn(),
  },
}));

const incompleteStatus = {
  status: 'needs_environment',
  defaults: {
    local_workspace_root: '~/.pomelo-orbit',
    image: 'traefik:3.6',
    rest_api_url: 'http://traefik:8080',
    rest_ready_timeout_seconds: 30,
    base_domain: 'example.test',
    default_entrypoint: 'web',
    tls_mode: 'none',
    acme_profile: '',
    acme_email: '',
    dns_api_token: '',
    local_platform: 'linux',
    local_host: 'localhost',
    local_username: 'orbit',
    rest_api_host_url: 'http://127.0.0.1:8080',
  },
  environment: undefined,
  gateway: undefined,
};

let mountedApp: App | undefined;
let target: HTMLDivElement | undefined;

async function flushRender() {
  await Promise.resolve();
  await nextTick();
  await Promise.resolve();
  await nextTick();
}

function initializationRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/project/:id/initialization',
        name: 'ProjectInitialization',
        component: ProjectInitializationPage,
      },
      { path: '/projects', name: 'Projects', component: { template: '<div>Projects</div>' } },
      { path: '/gateway', name: 'Gateway', component: { template: '<div>Gateway</div>' } },
    ],
  });
}

afterEach(() => {
  mountedApp?.unmount();
  target?.remove();
  mountedApp = undefined;
  target = undefined;
  vi.clearAllMocks();
});

describe('Project initialization page', () => {
  it('uses the route project when opening the wizard', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    vi.mocked(projectInitializationApi.getStatus).mockResolvedValue(incompleteStatus as never);
    const router = initializationRouter();

    await router.push('/project/project-1/initialization');
    await router.isReady();
    target = document.createElement('div');
    document.body.append(target);
    mountedApp = createApp(RouterView);
    mountedApp.use(pinia);
    mountedApp.use(router);
    mountedApp.use(i18n);
    mountedApp.mount(target);

    await vi.waitFor(() =>
      expect(projectInitializationApi.getStatus).toHaveBeenCalledWith('project-1')
    );
    expect(useProjectStore().activeProjectId).toBe('project-1');

    await router.push('/project/project-2/initialization');
    await vi.waitFor(() =>
      expect(projectInitializationApi.getStatus).toHaveBeenCalledWith('project-2')
    );
    await flushRender();

    expect(useProjectStore().activeProjectId).toBe('project-2');
  });

  it('returns a missing project to the project list', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    vi.mocked(projectInitializationApi.getStatus).mockRejectedValue(
      new ApiError('Project project-1 not found', 404, 'not_found', 'request-1')
    );
    const router = initializationRouter();

    await router.push('/project/project-1/initialization');
    await router.isReady();
    target = document.createElement('div');
    document.body.append(target);
    mountedApp = createApp(RouterView);
    mountedApp.use(pinia);
    mountedApp.use(router);
    mountedApp.use(i18n);
    mountedApp.mount(target);

    await vi.waitFor(() => expect(router.currentRoute.value.name).toBe('Projects'));
  });
});
