// @vitest-environment happy-dom
import { createApp, h, nextTick, ref, type App } from 'vue';
import { afterEach, describe, expect, it, vi } from 'vitest';
import i18n from '@/i18n';
import AppInput from './AppInput.vue';
import AppTextarea from './AppTextarea.vue';
import SearchControl from './SearchControl.vue';

let app: App | undefined;
let target: HTMLDivElement | undefined;

async function mount(view: () => ReturnType<typeof h>) {
  target = document.createElement('div');
  document.body.append(target);
  app = createApp({ render: view }).use(i18n);
  app.mount(target);
  await nextTick();
  return target;
}

function clearButton(root: ParentNode, key = 'common.clearInput') {
  const button = root.querySelector<HTMLButtonElement>(
    `button[aria-label="${i18n.global.t(key)}"]`
  );
  if (!button) {
    throw new Error(`Clear or reset button is missing: ${key}`);
  }
  return button;
}

function element<T extends Element>(root: ParentNode, selector: string): T {
  const result = root.querySelector<T>(selector);
  if (!result) {
    throw new Error(`Field element is missing: ${selector}`);
  }
  return result;
}

afterEach(() => {
  app?.unmount();
  target?.remove();
  app = undefined;
  target = undefined;
});

describe('shared field interactions', () => {
  it('clears search and runs the unfiltered query', async () => {
    const value = ref('api');
    const onSearch = vi.fn();
    const root = await mount(() =>
      h(SearchControl, {
        modelValue: value.value,
        onSearch,
        'onUpdate:modelValue': (next: string) => {
          value.value = next;
        },
      })
    );
    clearButton(root, 'common.clearSearch').click();
    await nextTick();
    expect(value.value).toBe('');
    expect(onSearch).toHaveBeenCalledTimes(1);
  });
  it('clears a text field without submitting its form', async () => {
    const value = ref('api');
    const onInput = vi.fn();
    const onSubmit = vi.fn();
    const root = await mount(() =>
      h('form', { onSubmit }, [
        h(AppInput, {
          id: 'name',
          modelValue: value.value,
          'onUpdate:modelValue': (next: string | number) => {
            value.value = String(next);
          },
          onInput,
          required: true,
          maxlength: 30,
          'aria-invalid': 'true',
          'aria-describedby': 'name-error',
        }),
      ])
    );
    const input = element<HTMLInputElement>(root, 'input');
    expect(input.id).toBe('name');
    expect(input.required).toBe(true);
    expect(input.maxLength).toBe(30);
    expect(input.getAttribute('aria-invalid')).toBe('true');
    expect(input.getAttribute('aria-describedby')).toBe('name-error');
    clearButton(root).click();
    await nextTick();
    expect(value.value).toBe('');
    expect(input.value).toBe('');
    expect(onInput).toHaveBeenCalledTimes(1);
    expect(onInput.mock.calls[0]?.[0].target).toBe(input);
    expect(onSubmit).not.toHaveBeenCalled();
    expect(root.querySelector('.app-clear-button')).toBeNull();
  });

  it('preserves number input conversion and clears zero as an empty value', async () => {
    const value = ref<string | number>(0);
    const root = await mount(() =>
      h(AppInput, {
        type: 'number',
        modelValue: value.value,
        modelModifiers: { number: true },
        min: 0,
        max: 10,
        'onUpdate:modelValue': (next: string | number) => {
          value.value = next;
        },
      })
    );
    const input = element<HTMLInputElement>(root, 'input');
    input.value = '3';
    input.dispatchEvent(new Event('input', { bubbles: true }));
    await nextTick();
    expect(value.value).toBe(3);
    expect(input.min).toBe('0');
    expect(input.max).toBe('10');
    value.value = 0;
    await nextTick();
    clearButton(root).click();
    await nextTick();
    expect(value.value).toBe('');
    expect(input.value).toBe('');
  });

  it('clears multiline text', async () => {
    const value = ref('line 1\nline 2');
    const onInput = vi.fn();
    const root = await mount(() =>
      h(AppTextarea, {
        modelValue: value.value,
        rows: 5,
        onInput,
        'onUpdate:modelValue': (next: string) => {
          value.value = next;
        },
      })
    );
    const textarea = element<HTMLTextAreaElement>(root, 'textarea');
    expect(textarea.getAttribute('rows')).toBe('5');
    clearButton(root).click();
    await nextTick();
    expect(value.value).toBe('');
    expect(onInput).toHaveBeenCalledTimes(1);
  });
});
