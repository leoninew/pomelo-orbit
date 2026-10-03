// @vitest-environment happy-dom
import { createApp, nextTick, type App } from 'vue';
import { createMemoryHistory, createRouter } from 'vue-router';
import { createPinia } from 'pinia';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { applicationApi } from '@/api/application/application';
import { projectEnvironmentApi } from '@/api/project/environment';
import { serviceApi } from '@/api/service/service';
import i18n from '@/i18n';
import { useProjectStore } from '@/stores/project';
import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
import type { EnvironmentResp } from '@/gen/proto/orbit/v1/environment/environment';
import ServicePage from './ServicePage.vue';
import ServiceDetail from './ServiceDetail.vue';

vi.mock('@/api/service/service', () => ({
  serviceApi: { list: vi.fn(), get: vi.fn(), deploy: vi.fn(), updateBasic: vi.fn() },
}));
vi.mock('@/api/application/application', () => ({
  applicationApi: { listVersions: vi.fn() },
}));
vi.mock('@/api/project/environment', () => ({ projectEnvironmentApi: { get: vi.fn() } }));
vi.mock('@/router/projectReadiness', () => ({
  ensureProjectExecutionReady: vi.fn().mockResolvedValue(true),
}));
vi.mock('@/components/MonacoEditor.vue', () => ({ default: { template: '<div />' } }));

const service: ServiceResp = {
  id: 'service-1',
  application_id: 'app-1',
  version_id: 'version-1',
  code: 'api',
  application_name: 'Traefik',
  application_code: 'traefik',
  application_kind: 'standard',
  version_label: 'v1',
  status: 'stopped',
  created_at: '',
  updated_at: '',
  components: [],
  pending_deploy: false,
  effective_plan_hash: '',
  effective_error: '',
  active_deployment: false,
  env: [],
  deployment_directory: '/custom/api',
  directory_target_revision: 2,
  runtime_directory: '/old/api',
  runtime_target_revision: 2,
};

let app: App | undefined;
let target: HTMLDivElement | undefined;

async function flushRender() {
  for (let index = 0; index < 4; index++) {
    await Promise.resolve();
    await nextTick();
  }
}

async function mountService(view: 'list' | 'detail', kind: string) {
  const snapshot = { ...service, application_kind: kind };
  vi.mocked(serviceApi.list).mockResolvedValue({
    items: [snapshot],
    total: 1,
    page: 1,
    per_page: 10,
    pages: 1,
  });
  vi.mocked(serviceApi.get).mockResolvedValue(snapshot);
  vi.mocked(applicationApi.listVersions).mockResolvedValue({
    items: [
      {
        id: 'version-1',
        application_id: 'app-1',
        label: 'v1',
        status: 'published',
        created_at: '',
        updated_at: '',
        components: [],
        component_summary: '',
      },
    ],
    total: 1,
    page: 1,
    per_page: 100,
    pages: 1,
  });
  vi.mocked(projectEnvironmentApi.get).mockResolvedValue({
    target_type: 'local',
    target_revision: 2,
    local: { platform: 'linux', workspace_root: '/workspace' },
  } as EnvironmentResp);
  vi.mocked(serviceApi.deploy).mockResolvedValue({ deployment_id: '', warnings: [] });
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/service', component: ServicePage },
      { path: '/service/:id', component: ServiceDetail },
      { path: '/application/:id', component: ServicePage },
      { path: '/version/:id', component: ServicePage },
    ],
  });
  await router.push(view === 'list' ? '/service' : '/service/service-1');
  const pinia = createPinia();
  useProjectStore(pinia).setActiveProject('project-1');
  target = document.createElement('div');
  document.body.append(target);
  app = createApp(view === 'list' ? ServicePage : ServiceDetail);
  app.use(pinia).use(router).use(i18n);
  app.mount(target);
  await flushRender();
  return target;
}

function deployButton(root: ParentNode) {
  const button = [...root.querySelectorAll<HTMLButtonElement>('button')].find(
    (item) => item.textContent?.trim() === i18n.global.t('service.actions.deploy')
  );
  if (!button) {
    throw new Error('Service deploy button is missing');
  }
  return button;
}

afterEach(() => {
  app?.unmount();
  target?.remove();
  app = undefined;
  target = undefined;
  vi.clearAllMocks();
});

describe('service deployment entry', () => {
  it.each(['list', 'detail'] as const)('disables Gateway deployment in %s', async (view) => {
    const root = await mountService(view, 'gateway');
    expect(deployButton(root).disabled).toBe(true);
    deployButton(root).click();
    await flushRender();
    expect(projectEnvironmentApi.get).not.toHaveBeenCalled();
    if (view === 'list') {
      const cardView = root.querySelector<HTMLButtonElement>(
        `button[aria-label="${i18n.global.t('service.cardView')}"]`
      );
      if (!cardView) {
        throw new Error('Service card view button is missing');
      }
      cardView.click();
      await flushRender();
      expect(deployButton(root).disabled).toBe(true);
    }
  });

  it.each(['list', 'detail'] as const)(
    'submits version and edited directory together from %s',
    async (view) => {
      const root = await mountService(view, 'standard');
      expect(deployButton(root).disabled).toBe(false);
      deployButton(root).click();
      await flushRender();
      const dialog = document.querySelector('[role="dialog"]');
      if (!dialog) {
        throw new Error('Service deployment dialog is missing');
      }
      const label = [...dialog.querySelectorAll('label')].find((item) =>
        item.textContent?.includes(i18n.global.t('service.deploy.directory'))
      );
      if (!label) {
        throw new Error('Service deployment directory field is missing');
      }
      const input = document.getElementById(label.htmlFor) as HTMLInputElement;
      expect(input.value).toBe('/custom/api');
      input.value = '/new/api';
      input.dispatchEvent(new Event('input', { bubbles: true }));
      await flushRender();
      expect(dialog.querySelector('[role="status"]')?.textContent?.trim()).toBe(
        i18n.global.t('service.deploy.directoryWarning')
      );
      deployButton(dialog).click();
      await flushRender();
      expect(serviceApi.deploy).toHaveBeenCalledWith('project-1', 'service-1', {
        version_id: 'version-1',
        deployment_directory: '/new/api',
        environment_target_revision: 2,
        force_recreate: false,
        join_traefik_network: true,
      });
      expect(serviceApi.updateBasic).not.toHaveBeenCalled();
    }
  );
});
