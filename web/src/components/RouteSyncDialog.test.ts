// @vitest-environment happy-dom
import { createApp, type App } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { routeApi } from '@/api/route/route';
import RouteSyncDialog from '@/components/RouteSyncDialog.vue';
import i18n from '@/i18n';
import { useProjectStore } from '@/stores/project';

vi.mock('@/api/route/route', () => ({
  routeApi: {
    previewSync: vi.fn(),
  },
}));

let app: App | undefined;
let target: HTMLDivElement | undefined;

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
        business_value: 'HTTP Host(`api.example.test`) -> http://api:8080',
        traefik_value: 'HTTP Host(`api.example.test`) -> http://old-api:8080',
      },
    ],
  });
});

afterEach(() => {
  app?.unmount();
  target?.remove();
  app = undefined;
  target = undefined;
  vi.clearAllMocks();
});

describe('RouteSyncDialog', () => {
  it('keeps each source aligned with its rule', async () => {
    target = document.createElement('div');
    document.body.append(target);
    app = createApp(RouteSyncDialog, { open: true, changes: [] });
    app.use(i18n);
    app.mount(target);
    await vi.waitFor(() => expect(routeApi.previewSync).toHaveBeenCalled());
    await vi.waitFor(() =>
      expect(document.body.textContent).toContain(i18n.global.t('route.syncSource'))
    );

    expect(document.body.textContent).toContain(i18n.global.t('route.syncSources.customRoute'));
    expect(document.body.textContent).toContain(i18n.global.t('route.syncSources.dockerLabel'));
    expect(document.body.textContent).toContain('HTTP Host(`api.example.test`) -> http://api:8080');
    expect(document.body.textContent).not.toContain(i18n.global.t('route.syncBusinessValue'));
    expect(document.body.textContent).not.toContain(i18n.global.t('route.syncTraefikValue'));

    const rows = [
      ...document.querySelectorAll<HTMLTableRowElement>('.app-dialog-content tbody tr'),
    ];
    expect(rows).toHaveLength(2);
    const [customRouteRow, dockerLabelRow] = rows;
    if (!customRouteRow || !dockerLabelRow) {
      throw new Error('Sync preview rows are missing');
    }

    expect(customRouteRow.cells).toHaveLength(3);
    expect(customRouteRow.cells.item(0)?.rowSpan).toBe(2);
    expect(customRouteRow.cells.item(1)?.textContent).toContain(
      i18n.global.t('route.syncSources.customRoute')
    );
    expect(customRouteRow.cells.item(2)?.classList.contains('break-words')).toBe(true);
    expect(dockerLabelRow.cells).toHaveLength(2);
    expect(dockerLabelRow.cells.item(0)?.textContent).toContain(
      i18n.global.t('route.syncSources.dockerLabel')
    );
    expect(dockerLabelRow.cells.item(1)?.textContent).toContain(
      'HTTP Host(`api.example.test`) -> http://old-api:8080'
    );
  });
});
