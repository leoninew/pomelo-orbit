// @vitest-environment happy-dom
/* eslint-disable vue/one-component-per-file */
import { createApp, nextTick, type App } from 'vue';
import { createMemoryHistory, createRouter } from 'vue-router';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { gatewayApi } from '@/api/gateway/gateway';
import { applicationApi } from '@/api/application/application';
import { projectEnvironmentApi } from '@/api/project/environment';
import { serviceApi } from '@/api/service/service';
import i18n from '@/i18n';
import { useProjectStore } from '@/stores/project';
import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';
import type { EnvironmentResp } from '@/gen/proto/orbit/v1/environment/environment';
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

vi.mock('@/api/project/environment', () => ({ projectEnvironmentApi: { get: vi.fn() } }));
vi.mock('@/router/projectReadiness', () => ({
  ensureProjectExecutionReady: vi.fn().mockResolvedValue(true),
}));
vi.mock('@/components/RuntimeContainerLogsDrawer.vue', () => ({
  default: { template: '<div />' },
}));

const gateway: GatewayResp = {
  deployment_directory: '',
  directory_target_revision: 0,
  runtime_directory: '',
  runtime_target_revision: 0,
  id: 'gateway-1',
  project_id: 'project-1',
  code: 'traefik',
  name: 'Traefik',
  kind: 'gateway',
  rest_api_url: 'http://traefik:8080',
  rest_api_host_url: 'http://127.0.0.1:8080',
  internal_domain: 'example.com',
  external_domain: '',
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
  it.each(['stopped', 'running', 'faulted'])(
    'allows Gateway operations during an active deployment with stored status %s',
    async (status) => {
      pinia = createPinia();
      useProjectStore(pinia).setActiveProject('project-1');
      vi.mocked(gatewayApi.list).mockResolvedValue({
        items: [
          {
            ...gateway,
            service_id: 'gateway-service',
            service_code: 'traefik-default',
            service_status: status,
            active_deployment: true,
            deployment_directory: '/custom/gateway',
            directory_target_revision: 2,
            runtime_directory: '/old/gateway',
            runtime_target_revision: 2,
          },
        ],
        total: 1,
        page: 1,
        per_page: 1,
        pages: 1,
      });
      vi.mocked(projectEnvironmentApi.get).mockResolvedValue({
        target_type: 'local',
        target_revision: 2,
        local: { platform: 'linux', workspace_root: '/workspace' },
      } as EnvironmentResp);
      vi.mocked(serviceApi.deploy).mockResolvedValue({ deployment_id: '', warnings: [] });
      vi.mocked(applicationApi.stop).mockResolvedValue({ deployment_id: '' });
      const router = createRouter({
        history: createMemoryHistory(),
        routes: [{ path: '/gateway', component: GatewayDetail }],
      });
      await router.push('/gateway');
      target = document.createElement('div');
      document.body.append(target);
      mountedApp = createApp(GatewayDetail);
      mountedApp.use(pinia).use(router).use(i18n);
      mountedApp.mount(target);
      await flushRender();
      const deployButton = [...target.querySelectorAll<HTMLButtonElement>('button')].find(
        (button) => button.textContent?.trim() === i18n.global.t('gateway.actions.deploy')
      );
      if (!deployButton) {
        throw new Error('Gateway deploy button is missing');
      }
      expect(deployButton.disabled).toBe(false);
      deployButton.click();
      await flushRender();
      const dialog = document.querySelector('[role="dialog"]');
      const input = dialog?.querySelector<HTMLInputElement>('input[type="text"]');
      if (!dialog || !input) {
        throw new Error('Gateway deployment directory input is missing');
      }
      expect(input.value).toBe('/custom/gateway');
      input.value = '/new/gateway';
      input.dispatchEvent(new Event('input', { bubbles: true }));
      await flushRender();
      expect(dialog.querySelector('[role="status"]')?.textContent?.trim()).toBe(
        i18n.global.t('service.deploy.gatewayDirectoryWarning')
      );
      const confirm = [...dialog.querySelectorAll<HTMLButtonElement>('button')].find(
        (button) => button.textContent?.trim() === i18n.global.t('common.confirm')
      );
      if (!confirm) {
        throw new Error('Gateway deployment confirmation button is missing');
      }
      expect(confirm.disabled).toBe(false);
      confirm.click();
      await flushRender();
      expect(serviceApi.deploy).toHaveBeenCalledWith('project-1', 'gateway-service', {
        deployment_directory: '/new/gateway',
        environment_target_revision: 2,
        force_recreate: false,
      });
      for (let index = 0; index < 2; index++) {
        const stopButton = [...target.querySelectorAll<HTMLButtonElement>('button')].find(
          (button) => button.textContent?.trim() === i18n.global.t('gateway.actions.stop')
        );
        if (!stopButton) {
          throw new Error('Gateway stop button is missing');
        }
        await vi.waitFor(() => expect(stopButton.disabled).toBe(false));
        stopButton.click();
        await flushRender();
        const stopDialog = document.querySelector('[role="dialog"]');
        const confirmStop = [
          ...(stopDialog?.querySelectorAll<HTMLButtonElement>('button') ?? []),
        ].find((button) => button.textContent?.trim() === i18n.global.t('common.confirm'));
        if (!confirmStop) {
          throw new Error('Gateway stop confirmation is missing');
        }
        confirmStop.click();
        await flushRender();
      }
      expect(applicationApi.stop).toHaveBeenCalledTimes(2);
    }
  );

  it('keeps an unconfigured gateway page open with the shared empty state', async () => {
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
      routes: [{ path: '/gateway', component: GatewayDetail }],
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
    expect(gatewayApi.list).toHaveBeenCalledWith('project-1');
    expect(target.textContent).toContain(i18n.global.t('common.noData'));
  });

  it('saves basic information independently of Traefik API settings', async () => {
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
      internal_domain: gateway.internal_domain,
      external_domain: gateway.external_domain,
    });
    expect(gatewayApi.list).toHaveBeenCalledWith('project-1');
    expect(target.textContent).toContain('Traefik edge');
    expect(renderErrors).toEqual([]);
  });

  it('validates and saves Traefik API settings in their own card and dialog', async () => {
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
    vi.mocked(gatewayApi.update).mockResolvedValue({
      ...gateway,
      rest_api_url: 'http://edge:8080',
      rest_api_host_url: 'http://127.0.0.1:9090',
      rest_ready_timeout_seconds: 45,
    });
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/gateway', component: GatewayDetail }],
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

    const cards = [...target.querySelectorAll<HTMLElement>('.app-detail-card')];
    const basicCard = cards.find(
      (card) =>
        card.querySelector('h2')?.textContent === i18n.global.t('gateway.sections.basicInfo')
    );
    const apiCard = cards.find(
      (card) =>
        card.querySelector('h2')?.textContent === i18n.global.t('gateway.sections.traefikApi')
    );
    expect(basicCard).toBeDefined();
    expect(apiCard).toBeDefined();
    expect(basicCard?.textContent).not.toContain(gateway.rest_api_url);
    expect(basicCard?.textContent).not.toContain(gateway.rest_api_host_url);
    expect(apiCard?.textContent).toContain(gateway.rest_api_url);
    expect(apiCard?.textContent).toContain(gateway.rest_api_host_url);
    expect(apiCard?.textContent).toContain(i18n.global.t('gateway.fields.restReadyTimeout'));
    apiCard?.querySelector<HTMLButtonElement>('.app-button-primary')?.click();
    await flushRender();

    expect(document.querySelector('#gateway-edit-name')).toBeNull();
    expect(document.querySelector('#gateway-edit-internal-domain')).toBeNull();
    const urlInput = document.querySelector<HTMLInputElement>('#gateway-edit-rest-api-url');
    const hostInput = document.querySelector<HTMLInputElement>('#gateway-edit-rest-api-host-url');
    const timeoutInput = document.querySelector<HTMLInputElement>(
      '#gateway-edit-rest-ready-timeout'
    );
    if (!urlInput || !hostInput || !timeoutInput) {
      throw new Error('Traefik API inputs are missing');
    }
    const confirmButton = [...document.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('common.confirm')
    );
    expect(confirmButton).toBeDefined();
    hostInput.value = 'invalid';
    hostInput.dispatchEvent(new Event('input', { bubbles: true }));
    timeoutInput.value = '0';
    timeoutInput.dispatchEvent(new Event('input', { bubbles: true }));
    confirmButton?.click();
    await flushRender();

    expect(gatewayApi.update).not.toHaveBeenCalled();
    expect(hostInput.getAttribute('aria-invalid')).toBe('true');
    expect(hostInput.classList.contains('app-input-error')).toBe(true);
    expect(document.querySelector('#gateway-edit-rest-api-host-url-error')?.textContent).toContain(
      i18n.global.t('gateway.validation.restApiHostUrlInvalid')
    );
    expect(timeoutInput.getAttribute('aria-invalid')).toBe('true');

    hostInput.value = 'http://127.0.0.1:9090';
    hostInput.dispatchEvent(new Event('input', { bubbles: true }));
    await flushRender();
    expect(hostInput.hasAttribute('aria-invalid')).toBe(false);
    expect(timeoutInput.getAttribute('aria-invalid')).toBe('true');
    urlInput.value = 'http://edge:8080';
    urlInput.dispatchEvent(new Event('input', { bubbles: true }));
    timeoutInput.value = '45';
    timeoutInput.dispatchEvent(new Event('input', { bubbles: true }));
    confirmButton?.click();
    await flushRender();

    expect(gatewayApi.update).toHaveBeenCalledWith('project-1', 'gateway-1', {
      rest_api_url: 'http://edge:8080',
      rest_api_host_url: 'http://127.0.0.1:9090',
      rest_ready_timeout_seconds: 45,
    });
    expect(target.textContent).toContain('http://edge:8080');
    expect(target.textContent).toContain(gateway.name);
    expect(document.querySelector('#gateway-edit-rest-api-url')).toBeNull();
  });
});
