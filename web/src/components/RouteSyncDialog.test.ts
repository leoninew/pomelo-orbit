// @vitest-environment happy-dom
import { createApp, type App } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { routeApi } from '@/api/route/route';
import RouteSyncDialog from '@/components/RouteSyncDialog.vue';
import type { RouteResp } from '@/gen/proto/orbit/v1/route/route';
import i18n from '@/i18n';
import { useProjectStore } from '@/stores/project';
import { ApiError } from '@/utils/request';

vi.mock('@/api/route/route', () => ({
  routeApi: {
    previewSync: vi.fn(),
    confirmSync: vi.fn(),
  },
}));

vi.mock('@/components/SelectControl.vue', async () => {
  const { h } = await import('vue');
  return {
    default: {
      props: ['id', 'modelValue', 'options', 'disabled'],
      emits: ['update:modelValue'],
      setup(
        props: {
          id: string;
          modelValue: string;
          options: { value: string; label: string; disabled?: boolean }[];
          disabled: boolean;
        },
        { emit }: { emit: (event: string, value: string) => void }
      ) {
        return () =>
          h(
            'select',
            {
              id: props.id,
              value: props.modelValue,
              disabled: props.disabled,
              onChange: (event: Event) =>
                emit('update:modelValue', (event.target as HTMLSelectElement).value),
            },
            props.options.map((option) =>
              h('option', { value: option.value, disabled: option.disabled }, option.label)
            )
          );
      },
    },
  };
});

let app: App | undefined;
let target: HTMLDivElement | undefined;

function mountDialog(props: Record<string, unknown>) {
  target = document.createElement('div');
  document.body.append(target);
  app = createApp(RouteSyncDialog, props);
  app.use(i18n);
  app.mount(target);
}

beforeEach(() => {
  setActivePinia(createPinia());
  useProjectStore().setActiveProject('project-1');
  vi.mocked(routeApi.previewSync).mockResolvedValue({
    business_hash: 'business-hash',
    traefik_hash: 'traefik-hash',
    matched: false,
    pending: [],
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
  it('keeps sync available when changes were saved but publishing failed', async () => {
    vi.mocked(routeApi.confirmSync).mockRejectedValueOnce(
      new ApiError(
        'Route changes were saved, but Traefik could not be updated.',
        503,
        'route_sync_publish_failed'
      )
    );
    const saved = vi.fn();
    mountDialog({
      open: true,
      changes: [{ route_id: 'route-1', enabled: true }],
      onSaved: saved,
    });
    await vi.waitFor(() => expect(routeApi.previewSync).toHaveBeenCalled());
    const confirm = [...document.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('route.syncAll')
    );
    await vi.waitFor(() => expect(confirm?.disabled).toBe(false));
    confirm?.click();
    await vi.waitFor(() => expect(saved).toHaveBeenCalledOnce());
    await vi.waitFor(() => expect(routeApi.previewSync).toHaveBeenCalledTimes(2));
    expect(routeApi.previewSync).toHaveBeenLastCalledWith('project-1', { changes: [] });
    expect(document.body.textContent).toContain('Route changes were saved');
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

  it('previews certificate and enablement together before confirming', async () => {
    const route: RouteResp = {
      id: 'route-1',
      name: 'api',
      domain: 'api.example.test',
      path_prefix: '/',
      target_url: 'http://api:8080',
      enabled: false,
      https_enabled: false,
      cert_type: 'manual',
      created_at: '',
      updated_at: '',
      protocol: 'http',
      acme_challenge: 'http',
      gateway_application_id: '',
      http01_available: true,
      dns01_available: false,
      acme_challenge_hint: '',
    };
    mountDialog({
      open: true,
      changes: [{ route_id: route.id, enabled: true }],
      route,
    });
    await vi.waitFor(() => expect(routeApi.previewSync).toHaveBeenCalled());

    const select = document.querySelector<HTMLSelectElement>('#route-sync-certificate');
    expect(select).not.toBeNull();
    await vi.waitFor(() => expect(select?.disabled).toBe(false));
    if (!select) {
      throw new Error('Certificate selector is missing');
    }
    select.value = 'mkcert';
    select.dispatchEvent(new Event('change', { bubbles: true }));

    await vi.waitFor(() =>
      expect(routeApi.previewSync).toHaveBeenLastCalledWith('project-1', {
        changes: [
          {
            route_id: route.id,
            enabled: true,
            certificate: { mode: 'mkcert', challenge: '', pem: '' },
          },
        ],
      })
    );
    expect(routeApi.confirmSync).not.toHaveBeenCalled();

    const confirm = [...document.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('route.syncAll')
    );
    await vi.waitFor(() => expect(confirm?.disabled).toBe(false));
    confirm?.click();
    await vi.waitFor(() =>
      expect(routeApi.confirmSync).toHaveBeenCalledWith('project-1', {
        changes: [
          {
            route_id: route.id,
            enabled: true,
            certificate: { mode: 'mkcert', challenge: '', pem: '' },
          },
        ],
        business_hash: 'business-hash',
        traefik_hash: 'traefik-hash',
      })
    );
  });

  it('keeps each source aligned with its rule', async () => {
    mountDialog({ open: true, changes: [] });
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
