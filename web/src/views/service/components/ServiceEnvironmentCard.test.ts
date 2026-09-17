// @vitest-environment happy-dom
import { createApp, nextTick, ref, type Ref } from 'vue';
import { createI18n } from 'vue-i18n';
import { describe, expect, it } from 'vitest';
import type {
  EnvironmentVariableEntry,
  EnvironmentVariableListRow,
} from '@/components/environmentVariableList';
import ServiceEnvironmentCard from './ServiceEnvironmentCard.vue';

function createCardApp(
  rows: Ref<EnvironmentVariableListRow[]>,
  onSave?: (entries: EnvironmentVariableEntry[]) => void
) {
  return createApp(ServiceEnvironmentCard, {
    rows: rows.value,
    savedRows: rows.value.map((row) => ({ ...row })),
    allowAdd: false,
    allowRemove: false,
    defaultValues: { LOG_LEVEL: 'info' },
    defaultValueLabel: 'Default',
    valueLabel: 'Current',
    'onUpdate:rows': (updatedRows: EnvironmentVariableListRow[]) => {
      rows.value = updatedRows;
    },
    onSave,
  });
}

describe('ServiceEnvironmentCard', () => {
  it('opens the text field when editing an unchanged component environment variable', async () => {
    const rows = ref<EnvironmentVariableListRow[]>([
      { id: 'LOG_LEVEL', key: 'LOG_LEVEL', value: 'info' },
    ]);
    const target = document.createElement('div');
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      messages: {
        en: {
          common: { cancel: 'Cancel', edit: 'Edit', operation: 'Operation', save: 'Save' },
          environment: {
            fields: { key: 'Key' },
            searchPlaceholder: 'Search environment variables',
            title: 'Environment variables',
          },
          service: { componentDetail: { currentValue: 'Current', defaultValue: 'Default' } },
        },
      },
    });
    document.body.append(target);
    const app = createCardApp(rows);
    app.use(i18n);
    app.mount(target);

    const editButton = Array.from(target.querySelectorAll('button')).find(
      (button) => button.textContent?.trim() === 'Edit'
    );
    expect(editButton).toBeDefined();

    editButton?.click();
    await nextTick();

    expect(target.querySelector('input.app-input')).not.toBeNull();
    app.unmount();
    target.remove();
  });

  it('saves an edited row from the card action', async () => {
    const rows = ref<EnvironmentVariableListRow[]>([
      { id: 'LOG_LEVEL', key: 'LOG_LEVEL', value: 'info' },
    ]);
    const savedEntries = ref<EnvironmentVariableEntry[]>([]);
    const target = document.createElement('div');
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      messages: {
        en: {
          common: {
            cancel: 'Cancel',
            clearSearch: 'Clear search',
            delete: 'Delete',
            edit: 'Edit',
            operation: 'Operation',
            save: 'Save',
          },
          environment: {
            actions: { hideValue: 'Hide value', showValue: 'Show value' },
            fields: { key: 'Key', value: 'Value' },
            noResults: 'No matching environment variables',
            searchPlaceholder: 'Search environment variables',
            title: 'Environment variables',
            validation: { duplicate: 'Duplicate', required: 'Required' },
          },
          service: { componentDetail: { currentValue: 'Current', defaultValue: 'Default' } },
        },
      },
    });
    document.body.append(target);
    const app = createCardApp(rows, (entries) => {
      savedEntries.value = entries;
    });
    app.use(i18n);
    app.mount(target);

    const editButton = Array.from(target.querySelectorAll('button')).find(
      (button) => button.textContent?.trim() === 'Edit'
    );
    editButton?.click();
    await nextTick();

    const valueInput = target.querySelector<HTMLInputElement>('input.app-input');
    expect(valueInput).not.toBeNull();
    if (!valueInput) {
      throw new Error('environment value input was not rendered');
    }
    valueInput.value = 'debug';
    valueInput.dispatchEvent(new Event('input', { bubbles: true }));
    await nextTick();

    const saveButtons = Array.from(target.querySelectorAll('button')).filter(
      (button) => button.textContent?.trim() === 'Save'
    );
    expect(saveButtons).toHaveLength(1);
    saveButtons[0]?.click();
    await nextTick();

    expect(savedEntries.value).toEqual([{ key: 'LOG_LEVEL', value: 'debug' }]);
    app.unmount();
    target.remove();
  });
});
