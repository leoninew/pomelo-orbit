// @vitest-environment happy-dom
import { createPinia, setActivePinia } from 'pinia';
import { createApp, nextTick, type App } from 'vue';
import { createMemoryHistory, createRouter, RouterView } from 'vue-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { projectInitializationApi } from '@/api/project/initialization';
import { routeApi } from '@/api/route/route';
import { traefikRouteApi } from '@/api/route/traefik';
import { serviceApi } from '@/api/service/service';
import i18n from '@/i18n';
import type { RouteResp } from '@/gen/proto/orbit/v1/route/route';
import { useProjectStore } from '@/stores/project';
import RouteDetail from './RouteDetail.vue';
import RoutePage from './RoutePage.vue';

vi.mock('@/api/project/initialization', () => ({
  projectInitializationApi: { getStatus: vi.fn() },
}));

vi.mock('@/api/route/route', () => ({
  routeApi: {
    get: vi.fn(),
    list: vi.fn(),
    update: vi.fn(),
    previewSync: vi.fn(),
    confirmSync: vi.fn(),
  },
}));

vi.mock('@/api/route/traefik', () => ({
  traefikRouteApi: { getConfig: vi.fn() },
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
  vi.mocked(projectInitializationApi.getStatus).mockResolvedValue({ status: 'ready' } as never);
  vi.mocked(routeApi.get).mockResolvedValue(route);
  vi.mocked(traefikRouteApi.getConfig).mockResolvedValue({
    dashboard_domain: '',
    https_enabled: false,
    internal_domain: 'internal.example.test',
    external_domain: '',
  });
  vi.mocked(routeApi.list).mockResolvedValue({ items: [route], total: 1 } as never);
  vi.mocked(routeApi.update).mockResolvedValue({ ...route, name: 'api-route-edited' });
  vi.mocked(routeApi.previewSync).mockResolvedValue({
    business_hash: 'business-hash',
    route_ids: ['route-1'],
    publication_hash: 'publication-hash',
    items: [
      {
        route_id: 'route-1',
        route_name: 'api',
        action: 'publish',
        rule: { protocol: 'http', match: 'Host(`api.example.test`)', target: 'http://api:8080' },
        cert_type: '',
        acme_challenge: '',
        business_hash: 'business-hash',
        publication_hash: 'publication-hash',
      },
    ],
  });
  vi.mocked(routeApi.confirmSync).mockResolvedValue({
    message: '',
    code: 'route_sync_completed',
    request_id: '',
    results: [
      {
        route_id: route.id,
        route_name: route.name,
        operation_id: 'operation-1',
        code: 'route_sync_completed',
        error: '',
        business_save: 'unchanged',
        file_commit: 'committed',
        configuration_match: 'matched',
        certificate_verification: 'not_applicable',
        recovery: 'not_needed',
        cleanup: 'completed',
      },
    ],
  });
  vi.mocked(serviceApi.list).mockResolvedValue({ items: [], pages: 1 } as never);
});

afterEach(() => {
  mountedApp?.unmount();
  target?.remove();
  mountedApp = undefined;
  target = undefined;
  vi.clearAllMocks();
});

describe('Route synchronization', () => {
  it.each([
    { path: '/routes', component: RoutePage, scope: 'project', routeIds: [] },
    { path: '/route/route-1', component: RouteDetail, scope: 'selected', routeIds: ['route-1'] },
  ] as const)(
    'retains an unsaved enable draft after closing failed sync on $path',
    async ({ path, component, scope, routeIds }) => {
      vi.mocked(routeApi.confirmSync).mockResolvedValueOnce({
        message: '',
        code: 'route_sync_incomplete',
        request_id: 'request-1',
        results: [
          {
            route_id: route.id,
            route_name: route.name,
            operation_id: '',
            code: 'route_sync_skipped',
            error: '',
            business_save: 'unchanged',
            file_commit: 'not_attempted',
            configuration_match: 'unverified',
            certificate_verification: 'not_applicable',
            recovery: 'not_needed',
            cleanup: 'not_attempted',
          },
        ],
      });
      const router = createRouter({
        history: createMemoryHistory(),
        routes: [{ path, component }],
      });
      await router.push(path);
      await router.isReady();
      target = document.createElement('div');
      document.body.append(target);
      mountedApp = createApp(RouterView);
      mountedApp.use(pinia);
      mountedApp.use(router);
      mountedApp.use(i18n);
      mountedApp.mount(target);
      await vi.waitFor(() => expect(target?.textContent).toContain(route.name));
      await flushRender();
      const disable = [...target.querySelectorAll<HTMLButtonElement>('button')].find(
        (button) => button.textContent?.trim() === i18n.global.t('route.status.disabled')
      );
      expect(disable).toBeDefined();
      disable?.click();
      await flushRender();
      const openSync = () =>
        [...(target?.querySelectorAll<HTMLButtonElement>('button') ?? [])].find((button) =>
          button.textContent?.trim().startsWith(i18n.global.t('route.syncAll'))
        );
      await vi.waitFor(() => expect(openSync()?.disabled).toBe(false));
      openSync()?.click();
      await vi.waitFor(() => expect(routeApi.previewSync).toHaveBeenCalledOnce());
      const dialogButton = (key: string) =>
        [...document.querySelectorAll<HTMLButtonElement>('.app-dialog-content button')].find(
          (button) => button.textContent?.trim() === i18n.global.t(key)
        );
      await vi.waitFor(() => expect(dialogButton('route.syncAll')?.disabled).toBe(false));
      dialogButton('route.syncAll')?.click();
      await vi.waitFor(() => {
        const status = document.querySelector<HTMLElement>(
          '.app-dialog-content tbody td:last-child span'
        );
        expect(status?.textContent?.trim()).toBe(i18n.global.t('route.syncStatuses.failed'));
        expect(status?.title).toBe(i18n.global.t('route.syncErrorCodes.route_sync_skipped'));
      });
      await vi.waitFor(() => expect(dialogButton('common.cancel')?.disabled).toBe(false));
      dialogButton('common.cancel')?.click();
      await flushRender();
      openSync()?.click();
      await vi.waitFor(() =>
        expect(routeApi.previewSync).toHaveBeenLastCalledWith('project-1', {
          scope,
          route_ids: [...routeIds],
          changes: [{ route_id: route.id, enabled: false }],
        })
      );
      expect(routeApi.previewSync).toHaveBeenCalledTimes(2);
    }
  );

  it('marks an edit as pending and only confirms after manual sync', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/route/:id', component: RouteDetail }],
    });
    await router.push('/route/route-1');
    await router.isReady();

    target = document.createElement('div');
    document.body.append(target);
    mountedApp = createApp(RouterView);
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
    if (!nameInput) {
      throw new Error('Route name input is missing');
    }
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
      'project-1',
      'route-1',
      expect.objectContaining({
        name: 'api-route-edited',
      })
    );
    expect(routeApi.previewSync).not.toHaveBeenCalled();
    expect(routeApi.confirmSync).not.toHaveBeenCalled();
    expect(target.textContent).toContain(i18n.global.t('route.syncPending', { count: 1 }));

    const syncButton = [...target.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('route.syncPending', { count: 1 })
    );
    expect(syncButton).toBeDefined();
    await vi.waitFor(() => expect(syncButton?.disabled).toBe(false));
    syncButton?.click();
    await vi.waitFor(() =>
      expect(routeApi.previewSync).toHaveBeenCalledWith('project-1', {
        scope: 'selected',
        route_ids: ['route-1'],
        changes: [],
      })
    );

    const syncConfirmButton = [...document.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('route.syncAll')
    );
    expect(syncConfirmButton).toBeDefined();
    await vi.waitFor(() => expect(syncConfirmButton?.disabled).toBe(false));
    syncConfirmButton?.click();
    await vi.waitFor(() =>
      expect(routeApi.confirmSync).toHaveBeenCalledWith('project-1', {
        changes: [],
        business_hash: 'business-hash',
        route_ids: ['route-1'],
        publication_hash: 'publication-hash',
      })
    );
  });

  it('keeps list changes pending until the user opens and confirms sync', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/routes', component: RoutePage }],
    });
    await router.push('/routes?pending_sync=1');
    await router.isReady();

    target = document.createElement('div');
    document.body.append(target);
    mountedApp = createApp(RouterView);
    mountedApp.use(pinia);
    mountedApp.use(router);
    mountedApp.use(i18n);
    mountedApp.mount(target);
    await vi.waitFor(() => expect(routeApi.list).toHaveBeenCalled());
    await flushRender();

    expect(routeApi.previewSync).not.toHaveBeenCalled();
    expect(routeApi.confirmSync).not.toHaveBeenCalled();

    const disableButton = [...target.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('route.status.disabled')
    );
    expect(disableButton).toBeDefined();
    disableButton?.click();
    await flushRender();

    expect(routeApi.previewSync).not.toHaveBeenCalled();
    expect(routeApi.confirmSync).not.toHaveBeenCalled();

    const syncButton = [...target.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('route.syncPending', { count: 2 })
    );
    expect(syncButton).toBeDefined();
    syncButton?.click();
    await vi.waitFor(() =>
      expect(routeApi.previewSync).toHaveBeenCalledWith('project-1', {
        scope: 'project',
        route_ids: [],
        changes: [{ route_id: route.id, enabled: false }],
      })
    );
    expect(routeApi.confirmSync).not.toHaveBeenCalled();

    const confirmButton = [...document.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('route.syncAll')
    );
    expect(confirmButton).toBeDefined();
    await vi.waitFor(() => expect(confirmButton?.disabled).toBe(false));
    confirmButton?.click();
    await vi.waitFor(() =>
      expect(routeApi.confirmSync).toHaveBeenCalledWith('project-1', {
        changes: [{ route_id: route.id, enabled: false }],
        business_hash: 'business-hash',
        route_ids: ['route-1'],
        publication_hash: 'publication-hash',
      })
    );
  });
});
