// @vitest-environment happy-dom
import { createApp, type App } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { routeApi } from '@/api/route/route';
import RouteSyncDialog from '@/components/RouteSyncDialog.vue';
import i18n from '@/i18n';
import { useProjectStore } from '@/stores/project';
import { ApiError } from '@/utils/request';

vi.mock('@/api/route/route', () => ({
  routeApi: { previewSync: vi.fn(), confirmSync: vi.fn() },
}));

let app: App | undefined;
let target: HTMLDivElement | undefined;

function mountDialog(props: Record<string, unknown>) {
  target = document.createElement('div');
  document.body.append(target);
  app = createApp(RouteSyncDialog, props);
  app.use(i18n);
  app.mount(target);
}

function button(label: string) {
  return [...document.querySelectorAll<HTMLButtonElement>('button')].find(
    (item) => item.textContent?.trim() === label
  );
}

beforeEach(() => {
  setActivePinia(createPinia());
  useProjectStore().setActiveProject('project-1');
  vi.mocked(routeApi.previewSync).mockResolvedValue({
    business_hash: 'business-hash',
    traefik_hash: 'traefik-hash',
    matched: false,
    differences: [
      {
        action: 'modified',
        route_name: 'api',
        field: 'route',
        business: { match: 'HTTP Host(`api.example.test`)', target: 'http://api:8080' },
        traefik: { match: 'HTTP Host(`api.example.test`)', target: 'http://old-api:8080' },
      },
    ],
  });
  vi.mocked(routeApi.confirmSync).mockResolvedValue({ message: 'Routes synced successfully' });
});

afterEach(() => {
  app?.unmount();
  target?.remove();
  app = undefined;
  target = undefined;
  vi.clearAllMocks();
});

describe('RouteSyncDialog', () => {
  it('retries publication from saved state after a failure', async () => {
    vi.mocked(routeApi.confirmSync).mockRejectedValueOnce(
      new ApiError('Publication failed', 503, 'route_sync_publish_failed')
    );
    const saved = vi.fn();
    mountDialog({ open: true, changes: [{ route_id: 'route-1', enabled: true }], onSaved: saved });
    await vi.waitFor(() => expect(routeApi.previewSync).toHaveBeenCalled());
    const confirm = button(i18n.global.t('route.syncAll'));
    await vi.waitFor(() => expect(confirm?.disabled).toBe(false));
    confirm?.click();
    await vi.waitFor(() => expect(saved).toHaveBeenCalledOnce());
    expect(document.body.textContent).toContain(i18n.global.t('route.syncPublishFailed'));
    expect(confirm?.disabled).toBe(true);
    button(i18n.global.t('route.retryPreview'))?.click();
    await vi.waitFor(() =>
      expect(routeApi.previewSync).toHaveBeenLastCalledWith('project-1', { changes: [] })
    );
    await vi.waitFor(() => expect(confirm?.disabled).toBe(false));
    confirm?.click();
    await vi.waitFor(() =>
      expect(routeApi.confirmSync).toHaveBeenLastCalledWith('project-1', {
        changes: [],
        business_hash: 'business-hash',
        traefik_hash: 'traefik-hash',
      })
    );
  });

  it('shows the permission error and request ID', async () => {
    vi.mocked(routeApi.confirmSync).mockRejectedValueOnce(
      new ApiError('Permission denied', 503, 'route_sync_publish_permission_denied', 'request-1')
    );
    mountDialog({ open: true, changes: [] });
    await vi.waitFor(() => expect(button(i18n.global.t('route.syncAll'))?.disabled).toBe(false));
    button(i18n.global.t('route.syncAll'))?.click();
    await vi.waitFor(() =>
      expect(document.body.textContent).toContain(i18n.global.t('route.syncPermissionDenied'))
    );
    expect(document.body.textContent).toContain('request-1');
  });

  it('identifies a saved certificate change when route rules still match', async () => {
    vi.mocked(routeApi.previewSync).mockResolvedValueOnce({
      business_hash: 'business-hash',
      traefik_hash: 'traefik-hash',
      matched: true,
      differences: [],
    });
    mountDialog({ open: true, changes: [], certificatePending: true });
    await vi.waitFor(() =>
      expect(document.body.textContent).toContain(i18n.global.t('route.syncCertificatePending'))
    );
    expect(document.body.textContent).not.toContain(i18n.global.t('route.syncMatched'));
    expect(document.querySelector('.app-dialog-content table')).toBeNull();
    expect(button(i18n.global.t('route.syncAll'))?.disabled).toBe(false);
  });

  it('shows route rules without certificate or enablement controls', async () => {
    mountDialog({ open: true, changes: [{ route_id: 'route-1', enabled: true }] });
    await vi.waitFor(() =>
      expect(document.body.textContent).toContain(i18n.global.t('route.syncSource'))
    );
    expect(document.body.textContent).toContain(i18n.global.t('route.syncSources.customRoute'));
    expect(document.body.textContent).toContain(i18n.global.t('route.syncSources.dockerLabel'));
    expect(document.querySelector('.app-dialog-content details')).toBeNull();
    expect(document.querySelector('#route-sync-certificate')).toBeNull();
    expect(document.body.textContent).not.toContain(i18n.global.t('route.status.enabled'));
    const rows = [
      ...document.querySelectorAll<HTMLTableRowElement>('.app-dialog-content tbody tr'),
    ];
    expect(rows).toHaveLength(2);
    expect(rows[0]?.cells.item(0)?.rowSpan).toBe(2);
    const businessRuleLines = rows[0]?.cells.item(2)?.querySelectorAll('p');
    expect(businessRuleLines).toHaveLength(2);
    expect(businessRuleLines?.[0]?.textContent).toBe('HTTP Host(`api.example.test`)');
    expect(businessRuleLines?.[1]?.textContent).toBe('-> http://api:8080');
    expect(rows[1]?.cells.item(0)?.textContent).toContain(
      i18n.global.t('route.syncSources.dockerLabel')
    );
  });
});
