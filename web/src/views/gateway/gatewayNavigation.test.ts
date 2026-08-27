// @vitest-environment happy-dom
import { createApp, nextTick, type App } from 'vue';
import { createMemoryHistory, createRouter } from 'vue-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { applicationApi } from '@/api/application/application';
import { gatewayApi } from '@/api/gateway/gateway';
import i18n from '@/i18n';
import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';
import GatewayDetail from './GatewayDetail.vue';

vi.mock('@/api/gateway/gateway', () => ({
  gatewayApi: {
    get: vi.fn(),
    update: vi.fn(),
  },
}));

vi.mock('@/api/application/application', () => ({
  applicationApi: {
    listServices: vi.fn(),
    stop: vi.fn(),
  },
}));

vi.mock('@/api/service/service', () => ({
  serviceApi: {
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
  base_domain: 'example.com',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  config_updated_at: '2026-01-01T00:00:00Z',
  default_entrypoint: 'web',
  tls_mode: 'none',
  exposures: [],
  default_service_id: '',
  default_service_instance_key: '',
  default_service_code: '',
  default_service_status: '',
  traefik_component_name: 'traefik',
  rest_ready_timeout_seconds: 30,
  acme_profile: '',
  acme_email: '',
  dns_api_token: '',
  version_bindings: [],
};

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

describe('Gateway detail editing', () => {
  it('saves control-plane fields in the detail dialog without a render error', async () => {
    vi.mocked(gatewayApi.get).mockResolvedValue(gateway);
    vi.mocked(gatewayApi.update).mockResolvedValue({ ...gateway, name: 'Traefik edge' });
    vi.mocked(applicationApi.listServices).mockResolvedValue({ items: [] });
    const renderErrors: unknown[] = [];
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/gateway/:id', component: GatewayDetail }],
    });

    await router.push('/gateway/gateway-1');
    target = document.createElement('div');
    document.body.append(target);
    mountedApp = createApp({ template: '<RouterView />' });
    mountedApp.config.errorHandler = (error) => renderErrors.push(error);
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
    if (!nameInput) throw new Error('Gateway name input is missing');
    nameInput.value = 'Traefik edge';
    nameInput.dispatchEvent(new Event('input', { bubbles: true }));
    await flushRender();

    const confirmButton = [...document.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('common.confirm')
    );
    expect(confirmButton).toBeDefined();
    confirmButton?.click();
    await flushRender();

    expect(gatewayApi.update).toHaveBeenCalledWith('gateway-1', {
      name: 'Traefik edge',
      traefik_component_name: gateway.traefik_component_name,
      rest_api_url: gateway.rest_api_url,
      rest_ready_timeout_seconds: gateway.rest_ready_timeout_seconds,
      base_domain: gateway.base_domain,
    });
    expect(target.textContent).toContain('Traefik edge');
    expect(renderErrors).toEqual([]);
  });
});
