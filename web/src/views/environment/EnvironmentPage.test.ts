// @vitest-environment happy-dom
import { createPinia, setActivePinia } from 'pinia';
import { createApp, nextTick, type App } from 'vue';
import { createMemoryHistory, createRouter, RouterView } from 'vue-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { projectEnvironmentApi } from '@/api/project/environment';
import { useToast } from '@/composables/useToast';
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
    update: vi.fn(),
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
  useToast().toasts.value = [];
  vi.clearAllMocks();
});

describe('Environment page', () => {
  it.each([true, false])(
    'saves and closes the edit dialog with target warning=%s',
    async (shared) => {
      const pinia = createPinia();
      setActivePinia(pinia);
      useProjectStore().setActiveProject('project-1');
      const environment = {
        id: 'environment-1',
        project_id: 'project-1',
        code: 'project-1',
        target_type: 'local',
        target_may_be_shared: false,
        target_revision: 1,
        created_at: '',
        updated_at: '',
        ssh: undefined,
        local: {
          workspace_root: '~/orbit',
          platform: 'linux',
          host: 'orbit-host',
          username: 'orbit',
        },
      };
      vi.mocked(projectEnvironmentApi.get).mockResolvedValue(environment);
      vi.mocked(projectEnvironmentApi.update).mockResolvedValue({
        ...environment,
        target_may_be_shared: shared,
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
      mountedApp.use(pinia).use(router).use(i18n).mount(target);
      await flushRender();
      [...target.querySelectorAll('button')]
        .find((button) => button.textContent?.trim() === i18n.global.t('common.edit'))
        ?.click();
      await vi.waitFor(() => expect(document.querySelector('[role="dialog"] form')).not.toBeNull());
      document
        .querySelector('[role="dialog"] form')
        ?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
      await vi.waitFor(() => expect(projectEnvironmentApi.update).toHaveBeenCalledOnce());
      await flushRender();
      expect(projectEnvironmentApi.update).toHaveBeenCalledWith('project-1', {
        target_type: 'local',
        local: { workspace_root: '~/orbit' },
      });
      expect(document.querySelector('[role="dialog"]')).toBeNull();
      expect(useToast().toasts.value).toEqual([
        expect.objectContaining({
          type: shared ? 'warning' : 'success',
          text: i18n.global.t(
            shared ? 'project.environment.targetMayBeSharedWarning' : 'project.environment.updated'
          ),
        }),
      ]);
    }
  );

  it.each(['local', 'ssh'])(
    'shows the terminal entry for a configured environment (%s)',
    async (targetType) => {
      const pinia = createPinia();
      setActivePinia(pinia);
      useProjectStore().setActiveProject('project-1');
      vi.mocked(projectEnvironmentApi.get).mockResolvedValue({
        id: 'environment-1',
        project_id: 'project-1',
        code: 'project-1',
        target_type: targetType,
        target_may_be_shared: false,
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
      expect(terminalEntry).toBeDefined();
      const initializationEntry = [...target.querySelectorAll('button')].find((button) =>
        button.textContent?.includes(i18n.global.t('project.initialization.sshCommand'))
      );
      expect(!!initializationEntry).toBe(targetType === 'ssh');
      terminalEntry?.click();
      await flushRender();
      expect(target.querySelector('[aria-label="Close test terminal"]')).not.toBeNull();
    }
  );

  it.each([
    { name: 'target revision change', revision: 2, disposals: 1 },
    { name: 'failed Docker probe', revision: 1, disposals: 0 },
  ])('retains the drawer on close and handles $name', async ({ revision, disposals }) => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useProjectStore().setActiveProject('project-1');
    const environment = {
      id: 'environment-1',
      project_id: 'project-1',
      code: 'project-1',
      target_type: 'ssh',
      target_may_be_shared: false,
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
      target_revision: revision,
      last_probe_status: 'failed',
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
    expect(terminalLifecycle.dispose).toHaveBeenCalledTimes(disposals);
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
