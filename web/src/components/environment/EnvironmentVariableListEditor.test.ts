// @vitest-environment happy-dom
import { createApp, nextTick, ref, type Ref } from 'vue';
import { createI18n } from 'vue-i18n';
import { describe, expect, it } from 'vitest';
import messages from '@/i18n/locales/en-US';
import type {
  EnvironmentVariableEntry,
  EnvironmentVariableListRow,
} from '@/components/environment/environmentVariableList';
import EnvironmentVariableListEditor from './EnvironmentVariableListEditor.vue';

function createEditorApp(
  rows: Ref<EnvironmentVariableListRow[]>,
  onSave?: (entries: EnvironmentVariableEntry[]) => void
) {
  return createApp(EnvironmentVariableListEditor, {
    title: 'Environment variables',
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

describe('EnvironmentVariableListEditor', () => {
  it('saves an edited row from the save action', async () => {
    const rows = ref<EnvironmentVariableListRow[]>([
      { id: 'LOG_LEVEL', key: 'LOG_LEVEL', value: 'info' },
    ]);
    const savedEntries = ref<EnvironmentVariableEntry[]>([]);
    const target = document.createElement('div');
    const i18n = createI18n({
      legacy: false,
      locale: 'en-US',
      messages: { 'en-US': messages },
    });
    document.body.append(target);
    const app = createEditorApp(rows, (entries) => {
      savedEntries.value = entries;
    });
    app.use(i18n);
    app.mount(target);

    try {
      const editButton = Array.from(target.querySelectorAll('button')).find(
        (button) => button.textContent?.trim() === 'Edit'
      );
      editButton?.click();
      await nextTick();

      const valueInput = target.querySelector<HTMLInputElement>('input[aria-label="Value"]');
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
    } finally {
      app.unmount();
      target.remove();
    }
  });
});
