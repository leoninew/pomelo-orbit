// @vitest-environment happy-dom
import { createApp, h, nextTick, ref, type App, type VNode } from 'vue';
import { createMemoryHistory, createRouter, RouterLink, type Router } from 'vue-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import AppTruncatedText from '@/components/AppTruncatedText.vue';

const fullText = 'registry.example.com/project/component:20261003-build-123456789';
const text = ref(fullText);
const displayText = ref<string>();
const resizeCallbacks = new Set<() => void>();
let visibleWidth: number;
let contentWidth: number;
let app: App | undefined;
let target: HTMLDivElement | undefined;

async function mountText(child?: () => VNode, router?: Router) {
  target = document.createElement('div');
  document.body.append(target);
  app = createApp({
    render: () =>
      h(
        AppTruncatedText,
        { text: text.value, displayText: displayText.value, asChild: Boolean(child) },
        child
      ),
  });
  if (router) {
    app.use(router);
  }
  app.mount(target);
  await nextTick();
}

function trigger() {
  const element = target?.querySelector<HTMLElement>('.truncate');
  if (!element) {
    throw new Error('truncated text was not rendered');
  }
  return element;
}

beforeEach(() => {
  text.value = fullText;
  displayText.value = undefined;
  visibleWidth = 160;
  contentWidth = 400;
  vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockImplementation(function (
    this: HTMLElement
  ) {
    return this.classList.contains('truncate') ? visibleWidth : 0;
  });
  vi.spyOn(HTMLElement.prototype, 'scrollWidth', 'get').mockImplementation(function (
    this: HTMLElement
  ) {
    return this.classList.contains('truncate') ? contentWidth : 0;
  });
  vi.stubGlobal(
    'ResizeObserver',
    class {
      constructor(private callback: () => void) {}
      observe() {
        resizeCallbacks.add(this.callback);
      }
      disconnect() {
        resizeCallbacks.delete(this.callback);
      }
      unobserve() {}
    }
  );
});

afterEach(() => {
  app?.unmount();
  target?.remove();
  app = undefined;
  target = undefined;
  resizeCallbacks.clear();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('AppTruncatedText', () => {
  it.each(['hover', 'focus'] as const)(
    'shows the full text on %s when the display text is shortened without overflowing',
    async (interaction) => {
      contentWidth = 100;
      text.value = '2026-10-04 13:22:30';
      displayText.value = '2026-10-04';
      await mountText();
      expect(trigger().textContent?.trim()).toBe(displayText.value);
      expect(trigger().classList.contains('decoration-dashed')).toBe(true);
      expect(trigger().tabIndex).toBe(0);
      if (interaction === 'hover') {
        trigger().dispatchEvent(new PointerEvent('pointermove', { pointerType: 'mouse' }));
      } else {
        trigger().focus();
      }
      await vi.waitFor(() => {
        expect(document.querySelector('.app-tooltip-content')?.textContent).toContain(text.value);
      });

      trigger().blur();
      text.value = '-';
      displayText.value = '-';
      await nextTick();
      expect(trigger().textContent?.trim()).toBe('-');
      expect(trigger().classList.contains('underline')).toBe(false);
      expect(trigger().hasAttribute('tabindex')).toBe(false);
      expect(document.querySelector('.app-tooltip-content')).toBeNull();
    }
  );

  it.each(['hover', 'focus'] as const)(
    'shows the full truncated text on %s',
    async (interaction) => {
      await mountText();
      expect(trigger().classList.contains('decoration-dashed')).toBe(true);
      expect(trigger().tabIndex).toBe(0);
      if (interaction === 'hover') {
        trigger().dispatchEvent(new PointerEvent('pointermove', { pointerType: 'mouse' }));
      } else {
        trigger().focus();
      }
      await vi.waitFor(() => {
        const tooltip = document.querySelector<HTMLElement>('.app-tooltip-content');
        expect(tooltip?.textContent).toContain(fullText);
        expect(target?.contains(tooltip)).toBe(false);
      });
    }
  );

  it('renders fitting text without an underline or tooltip', async () => {
    contentWidth = 160;
    await mountText();
    expect(trigger().textContent?.trim()).toBe(fullText);
    expect(trigger().classList.contains('underline')).toBe(false);
    expect(trigger().hasAttribute('tabindex')).toBe(false);
    trigger().dispatchEvent(new PointerEvent('pointermove', { pointerType: 'mouse' }));
    await new Promise((resolve) => setTimeout(resolve, 200));
    expect(document.querySelector('.app-tooltip-content')).toBeNull();
  });

  it('updates the truncation indicator when the container or text changes', async () => {
    await mountText();
    visibleWidth = 500;
    resizeCallbacks.forEach((callback) => callback());
    await nextTick();
    expect(trigger().classList.contains('underline')).toBe(false);
    expect(trigger().hasAttribute('tabindex')).toBe(false);

    visibleWidth = 160;
    resizeCallbacks.forEach((callback) => callback());
    await nextTick();
    expect(trigger().classList.contains('decoration-dashed')).toBe(true);

    contentWidth = 70;
    text.value = 'nginx:1.28';
    await nextTick();
    expect(trigger().textContent?.trim()).toBe('nginx:1.28');
    expect(trigger().classList.contains('underline')).toBe(false);
    expect(trigger().hasAttribute('tabindex')).toBe(false);
  });

  it('keeps the child link as the focus and click target', async () => {
    const onClick = vi.fn((event: MouseEvent) => event.preventDefault());
    await mountText(() => h('a', { href: '/version/version-id', onClick }, text.value));
    expect(trigger().tagName).toBe('A');
    expect(trigger().getAttribute('href')).toBe('/version/version-id');
    expect(trigger().hasAttribute('tabindex')).toBe(false);
    trigger().focus();
    await vi.waitFor(() => {
      expect(document.querySelector('.app-tooltip-content')?.textContent).toContain(fullText);
    });
    expect(document.activeElement).toBe(trigger());
    trigger().click();
    expect(onClick).toHaveBeenCalledOnce();
    expect(target?.querySelectorAll('a')).toHaveLength(1);
  });

  it('adds hover feedback to a child label without adding a tab stop', async () => {
    await mountText(() => h('span', text.value));
    expect(trigger().classList.contains('decoration-dashed')).toBe(true);
    expect(target?.querySelector('[tabindex]')).toBeNull();
    trigger().dispatchEvent(new PointerEvent('pointermove', { pointerType: 'mouse' }));
    await vi.waitFor(() => {
      expect(document.querySelector('.app-tooltip-content')?.textContent).toContain(fullText);
    });
  });

  it('preserves navigation and focus on a truncated router link', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/version/:id', component: () => h('div') }],
    });
    await router.push('/version/initial');
    await mountText(
      () => h(RouterLink, { to: '/version/target', class: 'app-link' }, () => text.value),
      router
    );
    expect(trigger().getAttribute('href')).toBe('/version/target');
    expect(trigger().hasAttribute('tabindex')).toBe(false);
    trigger().focus();
    await vi.waitFor(() => {
      expect(document.querySelector('.app-tooltip-content')?.textContent).toContain(fullText);
    });
    trigger().click();
    await vi.waitFor(() => expect(router.currentRoute.value.path).toBe('/version/target'));
  });
});
