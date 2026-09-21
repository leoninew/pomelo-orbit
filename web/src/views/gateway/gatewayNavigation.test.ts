// @vitest-environment happy-dom
/* eslint-disable vue/one-component-per-file */
import { createApp, nextTick, type App } from 'vue';
import { createMemoryHistory, createRouter } from 'vue-router';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { gatewayApi } from '@/api/gateway/gateway';
import i18n from '@/i18n';
import { useProjectStore } from '@/stores/project';
import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';
import GatewayDetail from './GatewayDetail.vue';

vi.mock('@/api/gateway/gateway', () => ({
  gatewayApi: {
    list: vi.fn(),
    update: vi.fn(),
  },
}));

vi.mock('@/api/application/application', () => ({
  applicationApi: {
    stop: vi.fn(),
  },
}));

vi.mock('@/api/service/service', () => ({
  serviceApi: {
    get: vi.fn(),
    deploy: vi.fn(),
  },
}));

const gateway: GatewayResp = {
  id: 'gateway-1',
  project_id: 'project-1',
  code: 'traefik',
  name: 'Traefik',
  kind: 'gateway',
  rest_api_url: 'http://traefik:8080',
  rest_api_host_url: 'http://127.0.0.1:8080',
  base_domain: 'example.com',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  config_updated_at: '2026-01-01T00:00:00Z',
  default_entrypoint: 'web',
  tls_mode: 'none',
  exposures: [],
  service_id: '',
  service_code: '',
  service_status: '',
  active_deployment: false,
  rest_ready_timeout_seconds: 30,
  acme_profile: '',
  acme_email: '',
  dns_api_token: '',
  version_bindings: [],
};

const ProjectInitializationStub = { template: '<div>Project initialization</div>' };

let mountedApp: App | undefined;
let target: HTMLDivElement | undefined;
let pinia: ReturnType<typeof createPinia> | undefined;

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
  pinia = undefined;
  vi.clearAllMocks();
});

describe('Gateway detail editing', () => {
  it('keeps an unconfigured gateway page open until initialization is requested', async () => {
    pinia = createPinia();
    setActivePinia(pinia);
    useProjectStore().setActiveProject('project-1');
    vi.mocked(gatewayApi.list).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      per_page: 1,
      pages: 0,
    });
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/gateway', component: GatewayDetail },
        {
          path: '/project/:id/initialization',
          name: 'ProjectInitialization',
          component: ProjectInitializationStub,
        },
      ],
    });

    await router.push('/gateway');
    target = document.createElement('div');
    document.body.append(target);
    mountedApp = createApp({ template: '<RouterView />' });
    mountedApp.use(pinia);
    mountedApp.use(router);
    mountedApp.use(i18n);
    mountedApp.mount(target);
    await flushRender();

    expect(router.currentRoute.value.path).toBe('/gateway');
    expect(target.textContent).toContain(i18n.global.t('gateway.empty'));

    const initializeButton = [...target.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('project.initialization.title')
    );
    expect(initializeButton).toBeDefined();
    initializeButton?.click();

    await vi.waitFor(() =>
      expect(router.currentRoute.value).toMatchObject({
        name: 'ProjectInitialization',
        params: { id: 'project-1' },
      })
    );
  });

  it('saves control-plane fields in the detail dialog without a render error', async () => {
    pinia = createPinia();
    setActivePinia(pinia);
    useProjectStore().setActiveProject('project-1');
    vi.mocked(gatewayApi.list).mockResolvedValue({
      items: [gateway],
      total: 1,
      page: 1,
      per_page: 1,
      pages: 1,
    });
    vi.mocked(gatewayApi.update).mockResolvedValue({ ...gateway, name: 'Traefik edge' });
    const renderErrors: unknown[] = [];
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/gateway', component: GatewayDetail }],
    });

    await router.push('/gateway');
    target = document.createElement('div');
    document.body.append(target);
    mountedApp = createApp({ template: '<RouterView />' });
    mountedApp.config.errorHandler = (error) => renderErrors.push(error);
    mountedApp.use(pinia);
    mountedApp.use(router);
    mountedApp.use(i18n);
    mountedApp.mount(target);
    await flushRender();

    const editButton = target.querySelector<HTMLButtonElement>(
      '.app-detail-card .app-button-primary'
    );
    expect(editButton).not.toBeNull();
    editButton?.click();
    await flushRender();

    const nameInput = document.querySelector<HTMLInputElement>('#gateway-edit-name');
    expect(nameInput).not.toBeNull();
    if (!nameInput) {
      throw new Error('Gateway name input is missing');
    }
    nameInput.value = 'Traefik edge';
    nameInput.dispatchEvent(new Event('input', { bubbles: true }));
    await flushRender();

    const confirmButton = [...document.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('common.confirm')
    );
    expect(confirmButton).toBeDefined();
    confirmButton?.click();
    await flushRender();

    expect(gatewayApi.update).toHaveBeenCalledWith('project-1', 'gateway-1', {
      name: 'Traefik edge',
      rest_api_url: gateway.rest_api_url,
      rest_api_host_url: gateway.rest_api_host_url,
      rest_ready_timeout_seconds: gateway.rest_ready_timeout_seconds,
      base_domain: gateway.base_domain,
    });
    expect(gatewayApi.list).toHaveBeenCalledWith('project-1');
    expect(target.textContent).toContain('Traefik edge');
    expect(renderErrors).toEqual([]);
  });
});
