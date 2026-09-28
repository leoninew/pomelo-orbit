// @vitest-environment happy-dom
import { createApp, defineComponent, h, nextTick, ref } from 'vue';
import { describe, expect, it } from 'vitest';
import ViewModeTabs from '@/components/ViewModeTabs.vue';

describe('ViewModeTabs', () => {
  it('keeps a view selected when its tab is clicked again', async () => {
    const mode = ref<'list' | 'dag'>('list');
    const target = document.createElement('div');
    const app = createApp(
      defineComponent({
        setup() {
          return () =>
            h(ViewModeTabs, {
              modelValue: mode.value,
              'onUpdate:modelValue': (value: 'list' | 'dag') => {
                mode.value = value;
              },
            });
        },
      })
    );
    document.body.append(target);
    app.mount(target);

    try {
      const listTab = target.querySelector<HTMLButtonElement>('[role="tab"]:first-child');
      const dagTab = target.querySelector<HTMLButtonElement>('[role="tab"]:last-child');

      listTab?.click();
      await nextTick();
      expect(mode.value).toBe('list');
      expect(listTab?.getAttribute('aria-selected')).toBe('true');

      dagTab?.click();
      await nextTick();
      expect(mode.value).toBe('dag');
      expect(dagTab?.getAttribute('aria-selected')).toBe('true');

      dagTab?.click();
      await nextTick();
      expect(mode.value).toBe('dag');
      expect(dagTab?.getAttribute('aria-selected')).toBe('true');
    } finally {
      app.unmount();
      target.remove();
    }
  });
});
