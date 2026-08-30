// @vitest-environment happy-dom
import { createPinia, setActivePinia } from 'pinia';
import { createApp, nextTick, type App } from 'vue';
import { createMemoryHistory, createRouter } from 'vue-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { routeApi } from '@/api/route/route';
import { serviceApi } from '@/api/service/service';
import i18n from '@/i18n';
import type { RouteResp } from '@/gen/proto/orbit/v1/route/route';
import { useProjectStore } from '@/stores/project';
import RouteDetail from './RouteDetail.vue';

vi.mock('@/api/route/route', () => ({
  routeApi: {
    get: vi.fn(),
    update: vi.fn(),
    previewSync: vi.fn(),
    confirmSync: vi.fn(),
  },
}));

vi.mock('@/api/service/service', () => ({
  serviceApi: {
    list: vi.fn(),
  },
}));

const route: RouteResp = {
  id: 'route-1',
  name: 'api-route',
  domain: 'api.example.test',
  path_prefix: '/',
  target_url: 'http://host.docker.internal:8080',
  enabled: true,
  https_enabled: false,
  cert_type: 'manual',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  protocol: 'http',
  acme_challenge: 'http',
  gateway_application_id: '',
  http01_available: false,
  dns01_available: false,
  acme_challenge_hint: '',
};

let mountedApp: App | undefined;
let target: HTMLDivElement | undefined;
let pinia: ReturnType<typeof createPinia>;

async function flushRender() {
  await Promise.resolve();
  await nextTick();
  await Promise.resolve();
  await nextTick();
}

beforeEach(() => {
  pinia = createPinia();
  setActivePinia(pinia);
  useProjectStore().setActiveProject('project-1');
  vi.mocked(routeApi.get).mockResolvedValue(route);
  vi.mocked(routeApi.update).mockResolvedValue({ ...route, name: 'api-route-edited' });
  vi.mocked(routeApi.previewSync).mockResolvedValue({
    business_hash: 'business-hash',
    traefik_hash: 'traefik-hash',
    matched: false,
    differences: [],
  });
  vi.mocked(routeApi.confirmSync).mockResolvedValue({ message: 'Routes synced successfully' });
  vi.mocked(serviceApi.list).mockResolvedValue({ items: [], pages: 1 } as never);
});

afterEach(() => {
  mountedApp?.unmount();
  target?.remove();
  mountedApp = undefined;
  target = undefined;
  vi.clearAllMocks();
});

describe('Route detail synchronization', () => {
  it('marks an edit as pending and only confirms after manual sync', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/route/:id', component: RouteDetail }],
    });
    await router.push('/route/route-1');
    await router.isReady();

    target = document.createElement('div');
    document.body.append(target);
    mountedApp = createApp({ template: '<RouterView />' });
    mountedApp.use(pinia);
    mountedApp.use(router);
    mountedApp.use(i18n);
    mountedApp.mount(target);
    await flushRender();

    const editButton = [...target.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('common.edit')
    );
    expect(editButton).toBeDefined();
    editButton?.click();
    await flushRender();

    const nameInput = document.querySelector<HTMLInputElement>('.app-dialog-content input');
    expect(nameInput).not.toBeNull();
    if (!nameInput) throw new Error('Route name input is missing');
    nameInput.value = 'api-route-edited';
    nameInput.dispatchEvent(new Event('input', { bubbles: true }));
    await flushRender();

    const editConfirmButton = [...document.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('common.confirm')
    );
    expect(editConfirmButton).toBeDefined();
    editConfirmButton?.click();
    await flushRender();

    expect(routeApi.update).toHaveBeenCalledWith(
      'route-1',
      expect.objectContaining({
        name: 'api-route-edited',
      })
    );
    expect(routeApi.confirmSync).not.toHaveBeenCalled();
    expect(target.textContent).toContain(i18n.global.t('route.syncPending', { count: 1 }));

    const syncButton = [...target.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('route.syncPending', { count: 1 })
    );
    expect(syncButton).toBeDefined();
    await vi.waitFor(() => expect(syncButton?.disabled).toBe(false));
    syncButton?.click();
    await vi.waitFor(() =>
      expect(routeApi.previewSync).toHaveBeenCalledWith(
        { changes: [] },
        { project_id: 'project-1' }
      )
    );

    const syncConfirmButton = [...document.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('common.confirm')
    );
    expect(syncConfirmButton).toBeDefined();
    await vi.waitFor(() => expect(syncConfirmButton?.disabled).toBe(false));
    syncConfirmButton?.click();
    await vi.waitFor(() =>
      expect(routeApi.confirmSync).toHaveBeenCalledWith(
        {
          changes: [],
          business_hash: 'business-hash',
          traefik_hash: 'traefik-hash',
        },
        { project_id: 'project-1' }
      )
    );
  });
});
