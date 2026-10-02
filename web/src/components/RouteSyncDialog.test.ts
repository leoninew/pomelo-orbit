// @vitest-environment happy-dom
import { createApp, nextTick, type App } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { routeApi } from '@/api/route/route';
import RouteSyncDialog from '@/components/RouteSyncDialog.vue';
import i18n from '@/i18n';
import { useProjectStore } from '@/stores/project';
import type {
  RouteSyncConfirmResp,
  RouteSyncPlanItemResp,
  RouteSyncPreviewResp,
  RouteSyncResultResp,
} from '@/gen/proto/orbit/v1/route/route';
import { ApiError } from '@/utils/request';

vi.mock('@/api/route/route', () => ({
  routeApi: { previewSync: vi.fn(), confirmSync: vi.fn() },
}));

let app: App | undefined;
let target: HTMLDivElement | undefined;

function mountDialog(props: Record<string, unknown>) {
  target = document.createElement('div');
  document.body.append(target);
  app = createApp(RouteSyncDialog, { scope: 'selected', routeIds: ['route-1'], ...props });
  app.use(i18n);
  app.mount(target);
}

function button(label: string) {
  return [...document.querySelectorAll<HTMLButtonElement>('button')].find(
    (item) => item.textContent?.trim() === label
  );
}

function status(label: string) {
  return [
    ...document.querySelectorAll<HTMLElement>('.app-dialog-content tbody td:last-child span'),
  ].find((item) => item.textContent?.trim() === label);
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

function planItem(routeId: string, action = 'publish'): RouteSyncPlanItemResp {
  return {
    route_id: routeId,
    route_name: routeId,
    action,
    rule: undefined,
    cert_type: '',
    acme_challenge: '',
    business_hash: `${routeId}-business`,
    publication_hash: `${routeId}-publication`,
  };
}

function rowStatus(routeId: string) {
  const row = [...document.querySelectorAll('.app-dialog-content tbody tr')].find(
    (item) => item.querySelector('td')?.textContent?.trim() === routeId
  );
  return row?.querySelector<HTMLElement>('td:last-child span');
}

beforeEach(() => {
  setActivePinia(createPinia());
  useProjectStore().setActiveProject('project-1');
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
    results: [itemResult()],
  });
});

function itemResult(overrides: Partial<RouteSyncResultResp> = {}): RouteSyncResultResp {
  return {
    route_id: 'route-1',
    route_name: 'api',
    operation_id: 'operation-1',
    code: 'route_sync_completed',
    error: '',
    business_save: 'unchanged',
    file_commit: 'committed',
    configuration_match: 'matched',
    certificate_verification: 'not_applicable',
    recovery: 'not_needed',
    cleanup: 'completed',
    ...overrides,
  };
}

afterEach(() => {
  app?.unmount();
  target?.remove();
  app = undefined;
  target = undefined;
  vi.clearAllMocks();
});

describe('RouteSyncDialog', () => {
  it('updates each row as sequential confirmations return and retries only the failure', async () => {
    const items = [planItem('a', 'withdraw'), planItem('b'), planItem('c', 'skip')];
    items[1].rule = { protocol: 'https', match: 'Host(`b.example.test`)', target: 'http://b:8080' };
    items[1].cert_type = 'manual';
    vi.mocked(routeApi.previewSync).mockResolvedValueOnce({
      business_hash: 'batch-business',
      publication_hash: 'batch-publication',
      route_ids: ['a', 'b', 'c'],
      items,
    });
    const a = deferred<RouteSyncConfirmResp>();
    const b = deferred<RouteSyncConfirmResp>();
    const c = deferred<RouteSyncConfirmResp>();
    vi.mocked(routeApi.confirmSync)
      .mockReturnValueOnce(a.promise)
      .mockReturnValueOnce(b.promise)
      .mockReturnValueOnce(c.promise);
    const saved = vi.fn();
    const synced = vi.fn();
    mountDialog({
      open: true,
      scope: 'project',
      changes: [
        { route_id: 'a', enabled: false },
        { route_id: 'b', enabled: true },
        { route_id: 'c', enabled: false },
      ],
      onSaved: saved,
      onSynced: synced,
    });
    await vi.waitFor(() => expect(button(i18n.global.t('route.syncAll'))?.disabled).toBe(false));
    const dialog = document.querySelector('.app-dialog-content');
    const table = dialog?.querySelector('table');
    const originalRows = [...document.querySelectorAll('.app-dialog-content tbody tr')];
    expect(dialog?.querySelectorAll('table')).toHaveLength(1);
    expect(originalRows).toHaveLength(3);
    for (const item of items) {
      expect(rowStatus(item.route_id)?.textContent?.trim()).toBe(
        i18n.global.t('route.syncStatuses.pending')
      );
    }
    button(i18n.global.t('route.syncAll'))?.click();
    await nextTick();
    expect(dialog?.querySelector('table')).toBe(table);
    expect([...document.querySelectorAll('.app-dialog-content tbody tr')]).toEqual(originalRows);
    expect(originalRows[1].textContent).toContain(i18n.global.t('route.syncActions.publish'));
    expect(originalRows[1].children[2].textContent).toContain('HTTPS');
    expect(originalRows[1].children[2].textContent).toContain('PEM');
    expect(originalRows[1].textContent).toContain('Host(`b.example.test`)');
    expect(originalRows[1].textContent).toContain('http://b:8080');
    expect(originalRows[1].textContent).toContain('PEM');
    expect(routeApi.confirmSync).toHaveBeenCalledTimes(1);
    expect(rowStatus('a')?.textContent?.trim()).toBe(
      i18n.global.t('route.syncStatuses.processing')
    );
    expect(rowStatus('b')?.textContent?.trim()).toBe(i18n.global.t('route.syncStatuses.pending'));
    expect(rowStatus('c')?.textContent?.trim()).toBe(i18n.global.t('route.syncStatuses.pending'));
    const resultA = itemResult({ route_id: 'a', business_save: 'saved' });
    a.resolve({ message: '', code: 'route_sync_completed', request_id: '', results: [resultA] });
    await vi.waitFor(() => expect(routeApi.confirmSync).toHaveBeenCalledTimes(2));
    expect(rowStatus('a')?.textContent?.trim()).toBe(i18n.global.t('route.syncStatuses.completed'));
    expect(rowStatus('b')?.textContent?.trim()).toBe(
      i18n.global.t('route.syncStatuses.processing')
    );
    expect(saved).toHaveBeenLastCalledWith([resultA]);
    expect(synced).not.toHaveBeenCalled();
    b.resolve({
      message: '',
      code: 'route_sync_incomplete',
      request_id: 'request-b',
      results: [
        itemResult({
          route_id: 'b',
          code: 'route_sync_publish_permission_denied',
          business_save: 'saved',
        }),
      ],
    });
    await vi.waitFor(() => expect(routeApi.confirmSync).toHaveBeenCalledTimes(3));
    expect(rowStatus('b')?.textContent?.trim()).toBe(i18n.global.t('route.syncStatuses.failed'));
    expect(rowStatus('b')?.title).toBe(
      i18n.global.t('route.syncErrorCodes.route_sync_publish_permission_denied')
    );
    expect(rowStatus('c')?.textContent?.trim()).toBe(
      i18n.global.t('route.syncStatuses.processing')
    );
    c.resolve({
      message: '',
      code: 'route_sync_completed',
      request_id: '',
      results: [
        itemResult({ route_id: 'c', business_save: 'saved', file_commit: 'not_applicable' }),
      ],
    });
    await vi.waitFor(() => expect(button(i18n.global.t('route.retryPreview'))).toBeDefined());
    expect(saved).toHaveBeenCalledTimes(3);
    expect(synced).not.toHaveBeenCalled();
    for (const [index, item] of items.entries()) {
      expect(routeApi.confirmSync).toHaveBeenNthCalledWith(index + 1, 'project-1', {
        route_ids: [item.route_id],
        changes: [{ route_id: item.route_id, enabled: item.route_id === 'b' }],
        business_hash: item.business_hash,
        publication_hash: item.publication_hash,
      });
    }
    const retryPreview = deferred<RouteSyncPreviewResp>();
    vi.mocked(routeApi.previewSync).mockReturnValueOnce(retryPreview.promise);
    button(i18n.global.t('route.retryPreview'))?.click();
    await nextTick();
    expect(document.querySelector('.app-dialog-content')).toBe(dialog);
    expect(dialog?.querySelectorAll('table')).toHaveLength(1);
    expect(dialog?.querySelector('table')).toBe(table);
    expect([...document.querySelectorAll('.app-dialog-content tbody tr')]).toEqual(originalRows);
    expect(rowStatus('a')?.textContent?.trim()).toBe(i18n.global.t('route.syncStatuses.completed'));
    expect(rowStatus('b')?.textContent?.trim()).toBe(i18n.global.t('route.syncStatuses.failed'));
    expect(rowStatus('c')?.textContent?.trim()).toBe(i18n.global.t('route.syncStatuses.completed'));
    retryPreview.resolve({
      business_hash: 'retry-business',
      publication_hash: 'retry-publication',
      route_ids: ['b'],
      items: [
        {
          ...planItem('b'),
          rule: { protocol: 'http', match: 'Host(`b.example.test`)', target: 'http://b:9090' },
          business_hash: 'retry-item-business',
          publication_hash: 'retry-item-publication',
        },
      ],
    });
    await vi.waitFor(() =>
      expect(routeApi.previewSync).toHaveBeenLastCalledWith('project-1', {
        scope: 'selected',
        route_ids: ['b'],
        changes: [],
      })
    );
    vi.mocked(routeApi.confirmSync).mockResolvedValueOnce({
      message: '',
      code: 'route_sync_completed',
      request_id: '',
      results: [itemResult({ route_id: 'b' })],
    });
    await vi.waitFor(() => expect(button(i18n.global.t('route.syncAll'))?.disabled).toBe(false));
    expect(dialog?.querySelector('table')).toBe(table);
    expect(originalRows[1].textContent).toContain('http://b:9090');
    button(i18n.global.t('route.syncAll'))?.click();
    await vi.waitFor(() => expect(synced).toHaveBeenCalledOnce());
    expect(routeApi.confirmSync).toHaveBeenCalledTimes(4);
    expect(routeApi.confirmSync).toHaveBeenLastCalledWith('project-1', {
      route_ids: ['b'],
      changes: [],
      business_hash: 'retry-item-business',
      publication_hash: 'retry-item-publication',
    });
    expect(status(i18n.global.t('route.syncStatuses.failed'))).toBeUndefined();
    expect(dialog?.querySelector('table')).toBe(table);
    expect([...document.querySelectorAll('.app-dialog-content tbody tr')]).toEqual(originalRows);
  });

  it.each([
    {
      error: new ApiError('Preview expired', 409, 'route_sync_preview_expired'),
      reason: 'route.syncErrorCodes.route_sync_preview_expired',
    },
    {
      error: new ApiError('Connection failed', undefined, undefined, undefined, 'network'),
      reason: 'Connection failed',
    },
  ])(
    'continues after a request failure and keeps the failed draft: $reason',
    async ({ error, reason }) => {
      vi.mocked(routeApi.previewSync).mockResolvedValueOnce({
        business_hash: 'batch-business',
        publication_hash: 'batch-publication',
        route_ids: ['a', 'b'],
        items: [planItem('a'), planItem('b')],
      });
      vi.mocked(routeApi.confirmSync)
        .mockRejectedValueOnce(error)
        .mockResolvedValueOnce({
          message: '',
          code: 'route_sync_completed',
          request_id: '',
          results: [itemResult({ route_id: 'b' })],
        });
      const saved = vi.fn();
      mountDialog({
        open: true,
        scope: 'project',
        changes: [{ route_id: 'a', enabled: true }],
        onSaved: saved,
      });
      await vi.waitFor(() => expect(button(i18n.global.t('route.syncAll'))?.disabled).toBe(false));
      button(i18n.global.t('route.syncAll'))?.click();
      await vi.waitFor(() => expect(button(i18n.global.t('route.retryPreview'))).toBeDefined());
      expect(routeApi.confirmSync).toHaveBeenCalledTimes(2);
      expect(rowStatus('a')?.title).toBe(
        reason.startsWith('route.') ? i18n.global.t(reason) : reason
      );
      expect(rowStatus('b')?.textContent?.trim()).toBe(
        i18n.global.t('route.syncStatuses.completed')
      );
      expect(saved).toHaveBeenCalledOnce();
      expect(saved.mock.calls[0]?.[0]?.[0]?.route_id).toBe('b');
      expect(routeApi.previewSync).toHaveBeenCalledOnce();
      button(i18n.global.t('route.retryPreview'))?.click();
      await vi.waitFor(() =>
        expect(routeApi.previewSync).toHaveBeenLastCalledWith('project-1', {
          scope: 'selected',
          route_ids: ['a'],
          changes: [{ route_id: 'a', enabled: true }],
        })
      );
    }
  );

  it('retries publication from saved state after a failure', async () => {
    vi.mocked(routeApi.confirmSync).mockResolvedValueOnce({
      message: '',
      code: 'route_sync_incomplete',
      request_id: 'request-1',
      results: [
        itemResult({
          code: 'route_sync_publish_failed',
          business_save: 'saved',
          configuration_match: 'unverified',
          recovery: 'restored',
        }),
      ],
    });
    const saved = vi.fn();
    mountDialog({ open: true, changes: [{ route_id: 'route-1', enabled: true }], onSaved: saved });
    await vi.waitFor(() => expect(routeApi.previewSync).toHaveBeenCalled());
    const confirm = button(i18n.global.t('route.syncAll'));
    await vi.waitFor(() => expect(confirm?.disabled).toBe(false));
    confirm?.click();
    await vi.waitFor(() => expect(saved).toHaveBeenCalledOnce());
    expect(status(i18n.global.t('route.syncStatuses.failed'))?.title).toBe(
      i18n.global.t('route.syncErrorCodes.route_sync_publish_failed')
    );
    expect(confirm?.disabled).toBe(true);
    button(i18n.global.t('route.retryPreview'))?.click();
    await vi.waitFor(() =>
      expect(routeApi.previewSync).toHaveBeenLastCalledWith('project-1', {
        scope: 'selected',
        route_ids: ['route-1'],
        changes: [],
      })
    );
    await vi.waitFor(() => expect(confirm?.disabled).toBe(false));
    confirm?.click();
    await vi.waitFor(() =>
      expect(routeApi.confirmSync).toHaveBeenLastCalledWith('project-1', {
        changes: [],
        business_hash: 'business-hash',
        route_ids: ['route-1'],
        publication_hash: 'publication-hash',
      })
    );
  });

  it.each([
    {
      code: 'route_sync_publish_permission_denied',
      cleanup: 'not_attempted',
      reason: 'route.syncErrorCodes.route_sync_publish_permission_denied',
    },
    {
      code: 'route_sync_reload_failed',
      cleanup: 'not_attempted',
      reason: 'route.syncErrorCodes.route_sync_reload_failed',
    },
    {
      code: 'route_sync_publish_failed',
      cleanup: 'failed',
      reason: 'route.syncCleanupFailed',
    },
  ])(
    'shows only failure status with a reason tooltip for $code/$cleanup',
    async ({ code, cleanup, reason }) => {
      vi.mocked(routeApi.confirmSync).mockResolvedValueOnce({
        message: '',
        code: 'route_sync_incomplete',
        request_id: 'request-1',
        results: [
          itemResult({
            code,
            cleanup,
            configuration_match: 'unverified',
          }),
        ],
      });
      mountDialog({ open: true, changes: [] });
      await vi.waitFor(() => expect(button(i18n.global.t('route.syncAll'))?.disabled).toBe(false));
      button(i18n.global.t('route.syncAll'))?.click();
      await vi.waitFor(() =>
        expect(status(i18n.global.t('route.syncStatuses.failed'))).toBeDefined()
      );
      const failure = status(i18n.global.t('route.syncStatuses.failed'));
      expect(failure?.title).toBe(i18n.global.t(reason));
      expect(failure?.tabIndex).toBe(0);
      expect(document.body.textContent).not.toContain(i18n.global.t(reason));
      expect(document.body.textContent).not.toContain('request-1');
      expect(document.body.textContent).not.toContain('operation-1');
    }
  );

  it.each([
    {
      protocol: 'https',
      targetUrl: 'http://api:8080',
      certType: 'manual',
      challenge: '',
      certificate: 'PEM',
    },
    {
      protocol: 'https',
      targetUrl: 'http://api:8080',
      certType: 'letsencrypt',
      challenge: 'dns',
      certificate: "Let's Encrypt · DNS-01",
    },
    {
      protocol: 'http',
      targetUrl: 'https://api:8443',
      certType: '',
      challenge: '',
      certificate: '',
    },
  ])(
    'shows $protocol separately from the upstream protocol',
    async ({ protocol, targetUrl, certType, challenge, certificate }) => {
      vi.mocked(routeApi.previewSync).mockResolvedValueOnce({
        business_hash: 'business-hash',
        route_ids: ['route-1'],
        publication_hash: 'publication-hash',
        items: [
          {
            route_id: 'route-1',
            route_name: 'api',
            action: 'publish',
            rule: { protocol, match: 'Host(`api.example.test`)', target: targetUrl },
            cert_type: certType,
            acme_challenge: challenge,
            business_hash: 'business-hash',
            publication_hash: 'publication-hash',
          },
        ],
      });
      mountDialog({ open: true, changes: [] });
      await vi.waitFor(() =>
        expect(document.body.textContent).toContain(i18n.global.t('route.syncActions.publish'))
      );
      const cells = document.querySelectorAll('.app-dialog-content tbody td');
      expect(cells[2].querySelector('p')?.textContent).toBe(protocol.toUpperCase());
      expect(cells[3].textContent).toContain('Host(`api.example.test`)');
      expect(cells[3].textContent).toContain(targetUrl);
      expect(cells[3].textContent).not.toContain('HTTPS');
      expect(cells[3].textContent).not.toContain('HTTP');
      if (certType) {
        expect(cells[2].textContent).toContain(certificate);
        expect(cells[3].textContent).not.toContain(certificate);
      }
      expect(document.querySelectorAll('.app-dialog-content tbody tr')).toHaveLength(1);
      expect(button(i18n.global.t('route.syncAll'))?.disabled).toBe(false);
    }
  );

  it('disables confirmation for an empty publication list', async () => {
    vi.mocked(routeApi.previewSync).mockResolvedValueOnce({
      business_hash: 'business-hash',
      route_ids: [],
      publication_hash: 'publication-hash',
      items: [],
    });
    mountDialog({ open: true, changes: [] });
    await vi.waitFor(() =>
      expect(document.body.textContent).toContain(i18n.global.t('route.syncEmpty'))
    );
    expect(button(i18n.global.t('route.syncAll'))?.disabled).toBe(true);
  });

  it('keeps the original table open through synchronization and completion', async () => {
    const response = deferred<RouteSyncConfirmResp>();
    vi.mocked(routeApi.confirmSync).mockReturnValueOnce(response.promise);
    const openChange = vi.fn();
    mountDialog({ open: true, changes: [], 'onUpdate:open': openChange });
    await vi.waitFor(() => expect(button(i18n.global.t('route.syncAll'))?.disabled).toBe(false));
    const dialog = document.querySelector('.app-dialog-content');
    const table = dialog?.querySelector('table');
    const row = table?.querySelector('tbody tr');
    button(i18n.global.t('route.syncAll'))?.click();
    await nextTick();
    expect(status(i18n.global.t('route.syncStatuses.processing'))).toBeDefined();
    expect(button(i18n.global.t('common.cancel'))?.disabled).toBe(true);
    dialog?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
    await nextTick();
    expect(openChange).not.toHaveBeenCalled();
    expect(dialog?.querySelector('table')).toBe(table);
    response.resolve({
      message: '',
      code: 'route_sync_completed',
      request_id: 'request-1',
      results: [itemResult({ certificate_verification: 'unverified' })],
    });
    await vi.waitFor(() =>
      expect(status(i18n.global.t('route.syncStatuses.completed'))).toBeDefined()
    );
    expect(status(i18n.global.t('route.syncStatuses.completed'))?.hasAttribute('title')).toBe(
      false
    );
    expect(
      [...document.querySelectorAll('.app-dialog-content th')].map((item) =>
        item.textContent?.trim()
      )
    ).toEqual([
      i18n.global.t('route.fields.name'),
      i18n.global.t('route.syncAction'),
      i18n.global.t('route.fields.protocol'),
      i18n.global.t('route.syncRule'),
      i18n.global.t('route.syncStatus'),
    ]);
    expect(document.querySelector('.app-dialog-content')).toBe(dialog);
    expect(dialog?.querySelectorAll('table')).toHaveLength(1);
    expect(dialog?.querySelector('table')).toBe(table);
    expect(table?.querySelector('tbody tr')).toBe(row);
    expect(row?.querySelectorAll('td')).toHaveLength(5);
    expect(row?.textContent).toContain(i18n.global.t('route.syncActions.publish'));
    expect(row?.children[2].textContent).toBe('HTTP');
    expect(row?.textContent).toContain('Host(`api.example.test`)');
    expect(row?.textContent).toContain('http://api:8080');
    expect(document.body.textContent).not.toContain('operation-1');
    expect(document.body.textContent).not.toContain('request-1');
    expect(button(i18n.global.t('common.close'))).toBeDefined();
    expect(openChange).not.toHaveBeenCalled();
    button(i18n.global.t('common.close'))?.click();
    expect(openChange).toHaveBeenCalledWith(false);
  });
});
