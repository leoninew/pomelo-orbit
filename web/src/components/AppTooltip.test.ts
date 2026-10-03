// @vitest-environment happy-dom
import { createApp, h, type App } from 'vue';
import { afterEach, describe, expect, it, vi } from 'vitest';
import AppTooltip from '@/components/AppTooltip.vue';

let app: App | undefined;
let target: HTMLDivElement | undefined;

function mountTooltip(content?: string) {
  target = document.createElement('div');
  document.body.append(target);
  app = createApp({
    render: () => h(AppTooltip, { content }, () => h('span', { tabindex: 0 }, 'HTTPS')),
  });
  app.mount(target);
  return target.querySelector<HTMLSpanElement>('span');
}

afterEach(() => {
  app?.unmount();
  target?.remove();
  app = undefined;
  target = undefined;
});

describe('AppTooltip', () => {
  it.each(['hover', 'focus'] as const)('shows portaled content on %s', async (interaction) => {
    const trigger = mountTooltip('PEM');
    if (interaction === 'hover') {
      trigger?.dispatchEvent(new PointerEvent('pointermove', { pointerType: 'mouse' }));
    } else {
      trigger?.focus();
    }
    await vi.waitFor(() => {
      const tooltip = document.querySelector<HTMLElement>('.app-tooltip-content');
      const description = document.querySelector<HTMLElement>('[role="tooltip"]');
      expect(tooltip?.textContent).toContain('PEM');
      expect(description?.textContent).toBe('PEM');
      expect(trigger?.getAttribute('aria-describedby')).toBe(description?.id);
      expect(target?.contains(tooltip)).toBe(false);
    });
  });

  it('renders only the trigger when there is no supplementary content', () => {
    const trigger = mountTooltip();
    trigger?.focus();
    expect(trigger?.textContent).toBe('HTTPS');
    expect(trigger?.hasAttribute('aria-describedby')).toBe(false);
    expect(document.querySelector('.app-tooltip-content')).toBeNull();
  });
});
