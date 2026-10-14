// @vitest-environment happy-dom
import { createApp, h, nextTick, ref, type App } from 'vue';
import { afterEach, describe, expect, it, vi } from 'vitest';
import i18n from '@/i18n';
import AppInput from './AppInput.vue';
import AppTextarea from './AppTextarea.vue';
import ComboboxSelect from './ComboboxSelect.vue';
import ListPagination from './ListPagination.vue';
import SelectControl from './SelectControl.vue';
import SearchControl from './SearchControl.vue';

let app: App | undefined;
let target: HTMLDivElement | undefined;

async function render() {
  for (let index = 0; index < 4; index++) {
    await nextTick();
  }
}

async function mount(view: () => ReturnType<typeof h>) {
  target = document.createElement('div');
  document.body.append(target);
  app = createApp({ render: view }).use(i18n);
  app.mount(target);
  await render();
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
  it('clears search and runs the unfiltered query while restoring input focus', async () => {
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
    await render();
    expect(value.value).toBe('');
    expect(onSearch).toHaveBeenCalledTimes(1);
    expect(document.activeElement).toBe(element<HTMLInputElement>(root, 'input'));
  });
  it('clears a text field through its native input event, preserving validation attributes and focus', async () => {
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
    await render();
    expect(value.value).toBe('');
    expect(input.value).toBe('');
    expect(onInput).toHaveBeenCalledTimes(1);
    expect(onInput.mock.calls[0]?.[0].target).toBe(input);
    expect(document.activeElement).toBe(input);
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
    await render();
    expect(value.value).toBe(3);
    expect(input.min).toBe('0');
    expect(input.max).toBe('10');
    value.value = 0;
    await render();
    clearButton(root).click();
    await render();
    expect(value.value).toBe('');
    expect(input.value).toBe('');
  });

  it('waits for composition to finish and applies the trim modifier', async () => {
    const value = ref('');
    const root = await mount(() =>
      h(AppInput, {
        modelValue: value.value,
        modelModifiers: { trim: true },
        'onUpdate:modelValue': (next: string | number) => {
          value.value = String(next);
        },
      })
    );
    const input = element<HTMLInputElement>(root, 'input');
    input.dispatchEvent(new CompositionEvent('compositionstart', { bubbles: true }));
    input.value = ' 中文 ';
    input.dispatchEvent(new Event('input', { bubbles: true }));
    expect(value.value).toBe('');
    input.dispatchEvent(new CompositionEvent('compositionend', { bubbles: true }));
    await render();
    expect(value.value).toBe('中文');
  });

  it('clears multiline text and keeps rows and focus on the textarea', async () => {
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
    await render();
    expect(value.value).toBe('');
    expect(onInput).toHaveBeenCalledTimes(1);
    expect(document.activeElement).toBe(textarea);
  });

  it('restores an unselected select without adding a placeholder option', async () => {
    const value = ref<string | number>('running');
    const root = await mount(() =>
      h(SelectControl, {
        modelValue: value.value,
        options: ['running', 'stopped'],
        placeholder: '筛选状态',
        'onUpdate:modelValue': (next: string | number) => {
          value.value = next;
        },
      })
    );
    const trigger = element<HTMLButtonElement>(root, '[role="combobox"]');
    clearButton(root, 'common.clearSelection').click();
    await render();
    expect(value.value).toBe('');
    expect(trigger.textContent?.trim()).toBe('筛选状态');
    expect(document.activeElement).toBe(trigger);
    trigger.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
    await render();
    expect(
      [...document.querySelectorAll('[role="option"]')].map((option) => option.textContent?.trim())
    ).toEqual(['running', 'stopped']);
  });

  it('clears combobox selection and typed search without reopening the menu', async () => {
    const value = ref<string | number>('api');
    const onSearch = vi.fn();
    const root = await mount(() =>
      h(ComboboxSelect, {
        modelValue: value.value,
        options: [{ value: 'api', label: 'API service' }],
        onSearch,
        'onUpdate:modelValue': (next: string | number) => {
          value.value = next;
        },
      })
    );
    const input = element<HTMLInputElement>(root, 'input');
    input.value = 'other';
    input.dispatchEvent(new Event('input', { bubbles: true }));
    await render();
    clearButton(root, 'common.clearSelection').click();
    await render();
    expect(value.value).toBe('');
    expect(input.value).toBe('');
    expect(onSearch).toHaveBeenLastCalledWith('');
    expect(document.activeElement).toBe(input);
    expect(input.getAttribute('aria-expanded')).toBe('false');
    input.value = 'unselected search';
    input.dispatchEvent(new Event('input', { bubbles: true }));
    await render();
    clearButton(root, 'common.clearSelection').click();
    await render();
    expect(input.value).toBe('');
    expect(root.querySelector('.app-clear-button')).toBeNull();
  });

  it('resets pagination to a valid configured size', async () => {
    const onSize = vi.fn();
    const root = await mount(() =>
      h(ListPagination, {
        current: 1,
        pageSize: 20,
        total: 40,
        totalPages: 2,
        pageSizeOptions: [10, 20, 50],
        'onChange-page-size': onSize,
      })
    );
    clearButton(root, 'common.reset').click();
    await render();
    expect(onSize).toHaveBeenCalledWith(10);
  });
});
