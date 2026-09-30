// @vitest-environment happy-dom
import { createPinia, setActivePinia } from 'pinia';
import { createApp, nextTick, type App } from 'vue';
import { createMemoryHistory, createRouter, RouterView } from 'vue-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { projectEnvironmentApi } from '@/api/project/environment';
import i18n from '@/i18n';
import { useProjectStore } from '@/stores/project';
import { ApiError } from '@/utils/request';
import EnvironmentPage from './EnvironmentPage.vue';

const terminalLifecycle = vi.hoisted(() => ({ dispose: vi.fn() }));

vi.mock('./EnvironmentTerminalDrawer.vue', async () => {
  const { defineComponent, h, onBeforeUnmount } = await import('vue');
  return {
    default: defineComponent({
      props: { open: Boolean },
      emits: ['update:open'],
      setup(props, { emit }) {
        onBeforeUnmount(terminalLifecycle.dispose);
        return () =>
          props.open
            ? h('button', {
                'aria-label': 'Close test terminal',
                onClick: () => emit('update:open', false),
              })
            : null;
      },
    }),
  };
});

vi.mock('@/api/project/environment', () => ({
  projectEnvironmentApi: {
    get: vi.fn(),
    probe: vi.fn(),
  },
}));

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

describe('Environment page', () => {
  it.each(['local', 'ssh'])(
    'shows the terminal entry only for an SSH environment (%s)',
    async (targetType) => {
      const pinia = createPinia();
      setActivePinia(pinia);
      useProjectStore().setActiveProject('project-1');
      vi.mocked(projectEnvironmentApi.get).mockResolvedValue({
        id: 'environment-1',
        project_id: 'project-1',
        code: 'project-1',
        target_type: targetType,
        target_revision: 1,
        created_at: '',
        updated_at: '',
        ssh:
          targetType === 'ssh'
            ? {
                host: 'remote-host',
                port: 22,
                username: 'managed-user',
                platform: 'linux',
                workspace_root: '/srv/orbit',
                host_key_fingerprint: '',
              }
            : undefined,
        local:
          targetType === 'local'
            ? {
                workspace_root: '/srv/orbit',
                platform: 'linux',
                host: 'local-host',
                username: 'local-user',
              }
            : undefined,
      });
      const router = createRouter({
        history: createMemoryHistory(),
        routes: [{ path: '/environment', component: EnvironmentPage }],
      });
      await router.push('/environment');
      await router.isReady();
      target = document.createElement('div');
      document.body.append(target);
      mountedApp = createApp(RouterView);
      mountedApp.use(pinia);
      mountedApp.use(router);
      mountedApp.use(i18n);
      mountedApp.mount(target);
      await flushRender();
      const terminalEntry = [...target.querySelectorAll('button')].find((button) =>
        button.textContent?.includes(i18n.global.t('project.environment.terminal.title'))
      );
      expect(!!terminalEntry).toBe(targetType === 'ssh');
    }
  );

  it('retains the drawer on close and disposes it when the environment revision changes', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useProjectStore().setActiveProject('project-1');
    const environment = {
      id: 'environment-1',
      project_id: 'project-1',
      code: 'project-1',
      target_type: 'ssh',
      target_revision: 1,
      created_at: '',
      updated_at: '',
      local: undefined,
      ssh: {
        host: 'remote-host',
        port: 22,
        username: 'managed-user',
        platform: 'linux',
        workspace_root: '/srv/orbit',
        host_key_fingerprint: '',
      },
    };
    vi.mocked(projectEnvironmentApi.get).mockResolvedValue(environment);
    vi.mocked(projectEnvironmentApi.probe).mockResolvedValue({
      ...environment,
      target_revision: 2,
      ssh: { ...environment.ssh, port: 2222 },
      last_probe_status: 'succeeded',
    });
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/environment', component: EnvironmentPage }],
    });
    await router.push('/environment');
    await router.isReady();
    target = document.createElement('div');
    document.body.append(target);
    mountedApp = createApp(RouterView);
    mountedApp.use(pinia);
    mountedApp.use(router);
    mountedApp.use(i18n);
    mountedApp.mount(target);
    await flushRender();
    [...target.querySelectorAll('button')]
      .find((button) =>
        button.textContent?.includes(i18n.global.t('project.environment.terminal.title'))
      )
      ?.click();
    await flushRender();
    target.querySelector<HTMLButtonElement>('[aria-label="Close test terminal"]')?.click();
    await flushRender();
    expect(terminalLifecycle.dispose).not.toHaveBeenCalled();
    [...target.querySelectorAll('button')]
      .find((button) => button.textContent?.includes(i18n.global.t('project.environment.probe')))
      ?.click();
    await flushRender();
    expect(projectEnvironmentApi.probe).toHaveBeenCalledWith('project-1');
    expect(terminalLifecycle.dispose).toHaveBeenCalledOnce();
  });

  it('redirects to project initialization when the environment is not found', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useProjectStore().setActiveProject('project-1');
    vi.mocked(projectEnvironmentApi.get).mockRejectedValue(
      new ApiError('Project environment not found', 404, 'not_found', 'request-1')
    );

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/environment', component: EnvironmentPage },
        {
          path: '/project/:id/initialization',
          name: 'ProjectInitialization',
          component: { template: '<div>Project initialization</div>' },
        },
      ],
    });

    await router.push('/environment');
    await router.isReady();
    target = document.createElement('div');
    document.body.append(target);
    mountedApp = createApp(RouterView);
    mountedApp.use(pinia);
    mountedApp.use(router);
    mountedApp.use(i18n);
    mountedApp.mount(target);

    await vi.waitFor(() => {
      expect(router.currentRoute.value).toMatchObject({
        name: 'ProjectInitialization',
        params: { id: 'project-1' },
        query: { redirect: '/environment' },
      });
    });
    await flushRender();

    expect(projectEnvironmentApi.get).toHaveBeenCalledWith('project-1');
    expect(target.textContent).not.toContain('Project environment not found');
  });
});
