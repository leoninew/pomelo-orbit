// @vitest-environment happy-dom
import { createApp, h, nextTick, ref } from 'vue';
import { createI18n } from 'vue-i18n';
import { describe, expect, it } from 'vitest';
import type { EnvironmentVariableListRow } from '@/components/environmentVariableList';
import ServiceEnvironmentCard from './ServiceEnvironmentCard.vue';

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
    const app = createApp({
      render: () =>
        h(ServiceEnvironmentCard, {
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
        }),
    });
    app.use(i18n);
    document.body.append(target);
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
});
