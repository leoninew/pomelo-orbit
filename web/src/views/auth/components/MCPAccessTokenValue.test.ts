// @vitest-environment happy-dom
import { createApp, nextTick, type App } from 'vue';
import { createI18n } from 'vue-i18n';
import { afterEach, describe, expect, it, vi } from 'vitest';
import messages from '@/i18n/locales/en-US';
import MCPAccessTokenValue from './MCPAccessTokenValue.vue';

const toast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock('@/composables/useToast', () => ({ useToast: () => toast }));

let app: App | undefined;
let target: HTMLDivElement | undefined;

function mount(value?: string) {
  target = document.createElement('div');
  document.body.append(target);
  app = createApp(MCPAccessTokenValue, { value });
  app.use(createI18n({ legacy: false, locale: 'en-US', messages: { 'en-US': messages } }));
  app.mount(target);
  return target;
}

function button(root: HTMLElement, label: string) {
  const result = root.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`);
  if (!result) {
    throw new Error(`Missing button: ${label}`);
  }
  return result;
}

afterEach(() => {
  app?.unmount();
  target?.remove();
  app = undefined;
  target = undefined;
  vi.restoreAllMocks();
  vi.clearAllMocks();
});

describe('MCPAccessTokenValue', () => {
  it('masks by default, copies the full token, and toggles visibility', async () => {
    const value = 'orbit_mcp_pat_example';
    const copy = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue();
    const root = mount(value);
    expect(root.textContent?.trim()).toBe('orbi********mple');

    button(root, messages.mcpAccessTokens.copy).click();
    await nextTick();
    expect(copy).toHaveBeenCalledWith(value);
    expect(toast.success).toHaveBeenCalledWith(messages.mcpAccessTokens.copied);
    expect(root.textContent?.trim()).toBe('orbi********mple');

    button(root, messages.mcpAccessTokens.showToken).click();
    await nextTick();
    expect(root.textContent?.trim()).toBe(value);
    button(root, messages.mcpAccessTokens.hideToken).click();
    await nextTick();
    expect(root.textContent?.trim()).toBe('orbi********mple');
  });

  it('does not offer reveal or copy for an absent token', () => {
    const root = mount();
    expect(root.textContent?.trim()).toBe('-');
    expect(root.querySelectorAll('button')).toHaveLength(0);
  });

  it('reports clipboard failure without revealing the token', async () => {
    vi.spyOn(navigator.clipboard, 'writeText').mockRejectedValue(new Error('Clipboard denied'));
    const root = mount('orbit_mcp_pat_example');
    button(root, messages.mcpAccessTokens.copy).click();
    await nextTick();
    expect(toast.error).toHaveBeenCalledWith(messages.mcpAccessTokens.copyFailed);
    expect(toast.success).not.toHaveBeenCalled();
    expect(root.textContent?.trim()).toBe('orbi********mple');
  });
});
