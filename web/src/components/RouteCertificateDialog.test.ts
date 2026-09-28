// @vitest-environment happy-dom
import { createApp, type App } from 'vue';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { routeApi } from '@/api/route/route';
import RouteCertificateDialog from '@/components/RouteCertificateDialog.vue';
import type { RouteResp } from '@/gen/proto/orbit/v1/route/route';
import i18n from '@/i18n';
import { useProjectStore } from '@/stores/project';

vi.mock('@/api/route/route', () => ({
  routeApi: { enableMkcert: vi.fn(), previewSync: vi.fn(), confirmSync: vi.fn() },
}));

vi.mock('@/components/SelectControl.vue', async () => {
  const { h } = await import('vue');
  return {
    default: {
      props: ['id', 'modelValue', 'options', 'disabled'],
      emits: ['update:modelValue'],
      setup(
        props: Record<string, unknown>,
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
            (props.options as Array<{ value: string; label: string }>).map((option) =>
              h('option', { value: option.value }, option.label)
            )
          );
      },
    },
  };
});

const route: RouteResp = {
  id: 'route-1',
  name: 'live',
  domain: 'live.example.test',
  path_prefix: '/',
  target_url: 'http://app:8080',
  enabled: true,
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
let app: App | undefined;
let target: HTMLDivElement | undefined;

beforeEach(() => {
  setActivePinia(createPinia());
  useProjectStore().setActiveProject('project-1');
  vi.mocked(routeApi.enableMkcert).mockResolvedValue({
    ...route,
    https_enabled: true,
    cert_type: 'mkcert',
  });
});

afterEach(() => {
  app?.unmount();
  target?.remove();
  app = undefined;
  target = undefined;
  vi.clearAllMocks();
});

it('saves a certificate on an enabled route without syncing', async () => {
  const saved = vi.fn();
  target = document.createElement('div');
  document.body.append(target);
  app = createApp(RouteCertificateDialog, { open: true, route, onSaved: saved });
  app.use(i18n);
  app.mount(target);

  const select = await vi.waitFor(() => {
    const item = document.querySelector<HTMLSelectElement>('#route-certificate-mode');
    expect(item).not.toBeNull();
    return item;
  });
  if (!select) {
    throw new Error('Certificate mode selector is missing');
  }
  select.value = 'mkcert';
  select.dispatchEvent(new Event('change', { bubbles: true }));
  const save = await vi.waitFor(() => {
    const item = [...document.querySelectorAll<HTMLButtonElement>('button')].find(
      (button) => button.textContent?.trim() === i18n.global.t('route.saveConfiguration')
    );
    expect(item?.disabled).toBe(false);
    return item;
  });
  save?.click();
  await vi.waitFor(() => expect(saved).toHaveBeenCalledOnce());
  expect(routeApi.enableMkcert).toHaveBeenCalledWith('project-1', 'route-1');
  expect(routeApi.previewSync).not.toHaveBeenCalled();
  expect(routeApi.confirmSync).not.toHaveBeenCalled();
});
