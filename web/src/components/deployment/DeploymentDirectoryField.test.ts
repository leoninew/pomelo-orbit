// @vitest-environment happy-dom
import { createApp, h, nextTick, ref } from 'vue';
import { describe, expect, it } from 'vitest';
import { createI18n } from 'vue-i18n';
import DeploymentDirectoryField from '@/components/deployment/DeploymentDirectoryField.vue';
import messages from '@/i18n/locales/zh-CN';

describe('DeploymentDirectoryField', () => {
  it.each([false, true])(
    'warns when editing a deployed directory (gateway=%s)',
    async (gateway) => {
      const directory = ref('/old/api');
      const target = document.createElement('div');
      const app = createApp({
        render: () =>
          h(DeploymentDirectoryField, {
            modelValue: directory.value,
            platform: 'linux',
            runtimeDirectory: '/old/api',
            gateway,
            'onUpdate:modelValue': (value: string) => {
              directory.value = value;
            },
          }),
      });
      app.use(createI18n({ legacy: false, locale: 'zh-CN', messages: { 'zh-CN': messages } }));
      document.body.append(target);
      app.mount(target);
      try {
        expect(target.querySelector('[role="status"]')).toBeNull();
        const input = target.querySelector('input');
        if (!input) {
          throw new Error('Deployment directory input is missing');
        }
        input.value = '/new/api';
        input.dispatchEvent(new Event('input', { bubbles: true }));
        await nextTick();
        expect(directory.value).toBe('/new/api');
        expect(target.querySelector('[role="status"]')?.textContent).toContain(
          '旧数据不会自动迁移'
        );
        directory.value = '/old//./api/';
        await nextTick();
        expect(target.querySelector('[role="status"]')).toBeNull();
      } finally {
        app.unmount();
        target.remove();
      }
    }
  );
});
