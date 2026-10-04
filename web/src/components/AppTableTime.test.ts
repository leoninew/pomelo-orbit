// @vitest-environment happy-dom
import { createApp, h, nextTick, ref, type App } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import AppTableTime from '@/components/AppTableTime.vue';

const time = ref<string | null>('2026-10-03T18:22:30Z');
let app: App | undefined;
let target: HTMLDivElement | undefined;

async function mountTime() {
  target = document.createElement('div');
  document.body.append(target);
  app = createApp({ render: () => h(AppTableTime, { time: time.value }) });
  app.mount(target);
  await nextTick();
}

function trigger() {
  const element = target?.querySelector<HTMLElement>('.truncate');
  if (!element) {
    throw new Error('table time was not rendered');
  }
  return element;
}

beforeEach(() => {
  time.value = '2026-10-03T18:22:30Z';
  vi.stubEnv('TZ', 'Asia/Shanghai');
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      disconnect() {}
      unobserve() {}
    }
  );
});

afterEach(() => {
  app?.unmount();
  target?.remove();
  app = undefined;
  target = undefined;
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
});

describe('AppTableTime', () => {
  it('shows the local date and reveals the full local time on focus', async () => {
    await mountTime();
    expect(trigger().textContent?.trim()).toBe('2026-10-04');
    expect(trigger().classList.contains('decoration-dashed')).toBe(true);
    trigger().focus();
    await vi.waitFor(() => {
      expect(document.querySelector('.app-tooltip-content')?.textContent).toContain(
        '2026-10-04 02:22:30'
      );
    });
  });

  it('shows a missing time without a tooltip and updates when a time arrives', async () => {
    time.value = null;
    await mountTime();
    expect(trigger().textContent?.trim()).toBe('-');
    expect(trigger().classList.contains('underline')).toBe(false);
    expect(trigger().hasAttribute('tabindex')).toBe(false);
    expect(document.querySelector('.app-tooltip-content')).toBeNull();

    time.value = '2026-10-03T18:22:30Z';
    await nextTick();
    expect(trigger().textContent?.trim()).toBe('2026-10-04');
    expect(trigger().tabIndex).toBe(0);
  });
});
