// @vitest-environment happy-dom
import { createApp, h, nextTick, ref, type Slots } from 'vue';
import { expect, it, vi } from 'vitest';
import { createI18n } from 'vue-i18n';
import messages from '@/i18n/locales/zh-CN';
import type { RouteResp } from '@/gen/proto/orbit/v1/route/route';
import RouteTargetServiceLink from './RouteTargetServiceLink.vue';

vi.mock('@/components/ui/AppTruncatedText.vue', () => ({
  default: {
    props: ['text', 'asChild'],
    setup:
      (_props: unknown, { slots }: { slots: Slots }) =>
      () =>
        slots.default?.(),
  },
}));

it('links a managed service and renders custom or deleted targets without a link', async () => {
  const route = ref<Pick<RouteResp, 'service_id' | 'service_code'>>({
    service_id: 'service-1',
    service_code: 'api-main',
  });
  const target = document.createElement('div');
  const app = createApp({ render: () => h(RouteTargetServiceLink, { route: route.value }) });
  app.component('RouterLink', (props: { to: string }, { slots }: { slots: Slots }) =>
    h('a', { href: props.to }, slots.default?.())
  );
  app.use(createI18n({ legacy: false, locale: 'zh-CN', messages: { 'zh-CN': messages } }));
  app.mount(target);
  try {
    expect(target.querySelector('a')?.getAttribute('href')).toBe('/service/service-1');
    expect(target.textContent).toBe('api-main');

    route.value = { service_id: 'service-2', service_code: 'api-worker' };
    await nextTick();
    expect(target.querySelector('a')?.getAttribute('href')).toBe('/service/service-2');
    expect(target.textContent).toBe('api-worker');

    route.value = { service_id: 'service-2' };
    await nextTick();
    expect(target.querySelector('a')).toBeNull();
    expect(target.textContent?.trim()).toBe('服务已删除');

    route.value = {};
    await nextTick();
    expect(target.querySelector('a')).toBeNull();
    expect(target.textContent?.trim()).toBe('—');
  } finally {
    app.unmount();
    target.remove();
  }
});
