// @vitest-environment happy-dom
import { createApp, h, nextTick, ref } from 'vue';
import { describe, expect, it } from 'vitest';
import { createI18n } from 'vue-i18n';
import DeploymentDirectoryField from './DeploymentDirectoryField.vue';
import AppDialogActions from './AppDialogActions.vue';
import messages from '@/i18n/locales/zh-CN';

describe('DeploymentDirectoryField', () => {
  it.each([false, true])(
    'warns for a stopped deployed service and still allows confirmation (gateway=%s)',
    async (gateway) => {
      const directory = ref('/old/api');
      let confirmed = false;
      const target = document.createElement('div');
      const app = createApp({
        render: () =>
          h('div', [
            h(DeploymentDirectoryField, {
              modelValue: directory.value,
              platform: 'linux',
              runtimeDirectory: '/old/api',
              gateway,
              'onUpdate:modelValue': (value: string) => {
                directory.value = value;
              },
            }),
            h(AppDialogActions, {
              confirmLabel: '部署',
              onConfirm: () => {
                confirmed = true;
              },
            }),
          ]),
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
        expect(target.querySelector('[role="status"]')?.textContent).toContain(
          '旧数据不会自动迁移'
        );
        const confirm = target.querySelector<HTMLButtonElement>('button:last-child');
        if (!confirm) {
          throw new Error('Deployment confirmation button is missing');
        }
        expect(confirm.disabled).toBe(false);
        confirm.click();
        expect(confirmed).toBe(true);
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
