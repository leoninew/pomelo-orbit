// @vitest-environment happy-dom
/* eslint-disable vue/one-component-per-file -- The test uses lightweight child component doubles. */
import { createApp, h, nextTick } from 'vue';
import { createI18n } from 'vue-i18n';
import { describe, expect, it, vi } from 'vitest';

const { getLogs, getDeployment } = vi.hoisted(() => ({
  getLogs: vi.fn(),
  getDeployment: vi.fn(),
}));

vi.mock('@/api/application/application', () => ({
  applicationApi: { getLogs },
}));

vi.mock('@/api/deployment/deployment', () => ({
  deploymentApi: { get: getDeployment },
}));

vi.mock('@/components/AppDrawer.vue', async () => {
  const { defineComponent, h: render } = await import('vue');
  return {
    default: defineComponent({
      props: { open: Boolean },
      setup(_, { slots }) {
        return () => render('section', [slots.default?.(), slots.footer?.()]);
      },
    }),
  };
});

vi.mock('@/components/ContainerLogView.vue', async () => {
  const { defineComponent, h: render } = await import('vue');
  return {
    default: defineComponent({
      props: { logs: String, autoRefreshing: Boolean },
      setup(props) {
        return () =>
          render('div', {
            'data-auto-refreshing': String(props.autoRefreshing),
            'data-logs': props.logs,
          });
      },
    }),
  };
});

import RuntimeContainerLogsDrawer from './RuntimeContainerLogsDrawer.vue';

async function flushAsyncWork() {
  await Promise.resolve();
  await nextTick();
  await Promise.resolve();
  await nextTick();
}

describe('RuntimeContainerLogsDrawer', () => {
  it('stops automatic refresh after the associated deployment completes', async () => {
    getLogs.mockResolvedValue({ logs: 'container started' });
    getDeployment.mockResolvedValue({ status: 'ran_to_completion' });
    const target = document.createElement('div');
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      messages: { en: { common: { cancel: 'Cancel', refresh: 'Refresh' } } },
    });
    const app = createApp({
      render: () =>
        h(RuntimeContainerLogsDrawer, {
          open: true,
          target: {
            applicationId: 'application-1',
            serviceId: 'service-1',
            component: 'web',
            title: 'Application / default · web',
            deploymentId: 'deployment-1',
          },
        }),
    });
    app.use(i18n);
    document.body.append(target);
    app.mount(target);

    await flushAsyncWork();

    expect(getLogs).toHaveBeenCalledOnce();
    expect(getDeployment).toHaveBeenCalledWith('deployment-1', expect.any(Object));
    expect(target.querySelector('[data-logs]')?.getAttribute('data-logs')).toBe(
      'container started'
    );
    expect(
      target.querySelector('[data-auto-refreshing]')?.getAttribute('data-auto-refreshing')
    ).toBe('false');

    app.unmount();
    target.remove();
  });
});
