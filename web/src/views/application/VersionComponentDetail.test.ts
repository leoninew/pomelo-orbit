// @vitest-environment happy-dom
import { createApp, nextTick } from 'vue';
import { createPinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import { createMemoryHistory, createRouter } from 'vue-router';
import { describe, expect, it, vi } from 'vitest';
import { applicationApi } from '@/api/application/application';
import type { VersionComponentResp } from '@/gen/proto/orbit/v1/application/version';
import messages from '@/i18n/locales/zh-CN';
import { useProjectStore } from '@/stores/project';
import VersionComponentDetail from './VersionComponentDetail.vue';

vi.mock('@/api/application/application', () => ({
  applicationApi: {
    getVersion: vi.fn(),
    get: vi.fn(),
    getVersionComponent: vi.fn(),
    updateVersionComponentBasic: vi.fn(),
    updateVersionComponentIdentity: vi.fn(),
  },
}));

vi.mock('@/components/MonacoEditor.vue', () => ({ default: { render: () => null } }));
vi.mock('@/components/AppDialog.vue', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    default: defineComponent({
      props: { open: Boolean, title: String },
      setup(props, { slots }) {
        return () =>
          props.open
            ? h('div', { 'data-dialog-title': props.title }, [slots.default?.(), slots.footer?.()])
            : null;
      },
    }),
  };
});

describe('VersionComponentDetail identity', () => {
  it('edits identity in its own section and saves only identity', async () => {
    const component: VersionComponentResp = {
      id: 'component-1',
      version_id: 'version-1',
      name: 'orbit',
      image: 'orbit:1',
      entrypoint: '',
      command: '',
      env: [],
      endpoints: [],
      mounts: [],
      dependencies: [],
      healthcheck: undefined,
      resources: undefined,
      pull_policy: 'missing',
      restart_policy: 'unless-stopped',
      tmpfs: [],
      ulimits: [],
      devices: [],
      user: '1000:1000',
      group_add: ['988', 'docker'],
      created_at: '',
      updated_at: '',
    };
    vi.mocked(applicationApi.getVersion).mockResolvedValue({
      id: 'version-1',
      application_id: 'application-1',
      label: '1',
      status: 'unpublished',
      components: [component],
      component_summary: '',
      created_at: '',
      updated_at: '',
    });
    vi.mocked(applicationApi.get).mockResolvedValue({
      id: 'application-1',
      name: 'Orbit',
      code: 'orbit',
      kind: 'standard',
      created_at: '',
      updated_at: '',
    });
    vi.mocked(applicationApi.getVersionComponent).mockResolvedValue(component);
    vi.mocked(applicationApi.updateVersionComponentIdentity).mockImplementation(
      async (_projectId, _versionId, _componentId, identity) => ({ ...component, ...identity })
    );
    const pinia = createPinia();
    useProjectStore(pinia).setActiveProject('project-1');
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/version/:versionId/component/:componentId', component: VersionComponentDetail },
      ],
    });
    const app = createApp(VersionComponentDetail);
    app.use(pinia);
    app.use(router);
    app.use(createI18n({ legacy: false, locale: 'zh-CN', messages: { 'zh-CN': messages } }));
    await router.push('/version/version-1/component/component-1');
    await router.isReady();
    const target = document.createElement('div');
    document.body.append(target);
    app.mount(target);
    try {
      const section = (title: string) =>
        Array.from(target.querySelectorAll('section')).find(
          (item) => item.querySelector('h2')?.textContent === title
        );
      await vi.waitFor(() => expect(section('运行身份')).toBeDefined());
      const basic = section('基础信息');
      expect(basic?.textContent).not.toContain('运行用户');
      expect(basic?.textContent).not.toContain('附加组');
      section('运行身份')?.querySelector<HTMLButtonElement>('button')?.click();
      await nextTick();
      const dialog = target.querySelector('[data-dialog-title="运行身份"]');
      const inputs = dialog?.querySelectorAll<HTMLInputElement>('input');
      expect(inputs).toHaveLength(2);
      const [user, groups] = Array.from(inputs ?? []);
      if (!dialog || !user || !groups) {
        throw new Error('Identity editor is missing');
      }
      expect(groups.value).toBe('988, docker');
      user.value = '2000:2000';
      user.dispatchEvent(new Event('input', { bubbles: true }));
      groups.value = '999, docker';
      groups.dispatchEvent(new Event('input', { bubbles: true }));
      await nextTick();
      Array.from(dialog.querySelectorAll<HTMLButtonElement>('button'))
        .find((button) => button.textContent?.includes('保存'))
        ?.click();
      await vi.waitFor(() =>
        expect(applicationApi.updateVersionComponentIdentity).toHaveBeenCalledWith(
          'project-1',
          'version-1',
          'component-1',
          { user: '2000:2000', group_add: ['999', 'docker'] }
        )
      );
      expect(applicationApi.updateVersionComponentBasic).not.toHaveBeenCalled();
      await vi.waitFor(() => expect(section('运行身份')?.textContent).toContain('2000:2000'));
      expect(basic?.textContent).toContain('orbit:1');
    } finally {
      app.unmount();
      target.remove();
    }
  });
});
