import { describe, expect, it } from 'vitest';
import {
  componentEnvironmentOverlays,
  componentEnvironmentRowsEqual,
  resetComponentEnvironmentRow,
  updateComponentEnvironmentRows,
  type ServiceComponentEnvironmentRow,
} from './serviceComponentEnvironment';

function environmentRow(
  key: string,
  overrides: Partial<ServiceComponentEnvironmentRow> = {}
): ServiceComponentEnvironmentRow {
  return {
    key,
    base: 'base',
    inheritedValue: 'base',
    value: 'base',
    deleted: false,
    overridden: false,
    ...overrides,
  };
}

describe('service component environment state', () => {
  it('updates only the edited row and preserves unrelated deleted overlays', () => {
    const rows = [environmentRow('LOG_LEVEL'), environmentRow('LEGACY_FLAG', { deleted: true })];

    expect(
      updateComponentEnvironmentRows(rows, [
        { id: 'LOG_LEVEL', key: 'LOG_LEVEL', value: 'debug' },
        { id: 'LEGACY_FLAG', key: 'LEGACY_FLAG', value: 'base' },
      ])
    ).toEqual([
      environmentRow('LOG_LEVEL', { value: 'debug', overridden: true }),
      environmentRow('LEGACY_FLAG', { deleted: true }),
    ]);
  });

  it('resets an override to its inherited value', () => {
    expect(
      resetComponentEnvironmentRow(
        [environmentRow('LOG_LEVEL', { value: 'debug', overridden: true })],
        'LOG_LEVEL'
      )
    ).toEqual([environmentRow('LOG_LEVEL')]);
  });

  it('includes overlay state when comparing saved rows', () => {
    expect(
      componentEnvironmentRowsEqual(
        [environmentRow('LOG_LEVEL')],
        [environmentRow('LOG_LEVEL', { deleted: true })]
      )
    ).toBe(false);
  });

  it('builds only deleted and overridden API overlays', () => {
    expect(
      componentEnvironmentOverlays([
        environmentRow('UNCHANGED'),
        environmentRow('LOG_LEVEL', { value: 'debug', overridden: true }),
        environmentRow('LEGACY_FLAG', { deleted: true }),
      ])
    ).toEqual([
      { key: 'LOG_LEVEL', value: 'debug', state: 'override' },
      { key: 'LEGACY_FLAG', state: 'deleted' },
    ]);
  });
});
