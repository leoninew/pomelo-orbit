// @vitest-environment happy-dom
import { createApp, h, nextTick, ref } from 'vue';
import { describe, expect, it } from 'vitest';
import { createI18n } from 'vue-i18n';
import ComponentIdentityFields from './ComponentIdentityFields.vue';
import messages from '@/i18n/locales/zh-CN';

describe('ComponentIdentityFields', () => {
  it('associates identity errors with controls and clears only the corrected field', async () => {
    const user = ref('1000:');
    const groups = ref('988,,docker');
    const target = document.createElement('div');
    const app = createApp({
      render: () =>
        h(ComponentIdentityFields, {
          user: user.value,
          groups: groups.value,
          'onUpdate:user': (value: string) => {
            user.value = value;
          },
          'onUpdate:groups': (value: string) => {
            groups.value = value;
          },
        }),
    });
    app.use(createI18n({ legacy: false, locale: 'zh-CN', messages: { 'zh-CN': messages } }));
    document.body.append(target);
    app.mount(target);
    try {
      const inputs = target.querySelectorAll('input');
      expect(inputs).toHaveLength(2);
      const [input, groupsInput] = inputs;
      if (!input || !groupsInput) {
        throw new Error('Identity controls are missing');
      }
      expect(input.getAttribute('aria-invalid')).toBe('true');
      expect(groupsInput.getAttribute('aria-invalid')).toBe('true');
      expect(target.querySelectorAll('[role="alert"]')).toHaveLength(2);
      input.value = '1000:1000';
      input.dispatchEvent(new Event('input', { bubbles: true }));
      await nextTick();
      expect(user.value).toBe('1000:1000');
      expect(input.hasAttribute('aria-invalid')).toBe(false);
      expect(groupsInput.getAttribute('aria-invalid')).toBe('true');
      expect(target.querySelectorAll('[role="alert"]')).toHaveLength(1);
      groupsInput.value = '988, docker';
      groupsInput.dispatchEvent(new Event('input', { bubbles: true }));
      await nextTick();
      expect(groups.value).toBe('988, docker');
      expect(target.querySelectorAll('[role="alert"]')).toHaveLength(0);
    } finally {
      app.unmount();
      target.remove();
    }
  });
});
