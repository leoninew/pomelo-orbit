import { describe, expect, it } from 'vitest';
import {
  environmentVariableRowsEqual,
  validateEnvironmentVariableRows,
  type EnvironmentVariableListRow,
} from './environmentVariableList';

const rows = (values: Array<[string, string]>): EnvironmentVariableListRow[] =>
  values.map(([key, value], index) => ({ id: `env-${index}`, key, value }));

describe('environmentVariableList', () => {
  it('drops a fully blank draft row before submission', () => {
    const result = validateEnvironmentVariableRows(
      rows([
        ['LOG_LEVEL', 'info'],
        ['', ''],
      ])
    );

    expect(result).toMatchObject({
      valid: true,
      entries: [{ key: 'LOG_LEVEL', value: 'info' }],
    });
  });

  it('keeps a partial row local and assigns its error to the key input', () => {
    const result = validateEnvironmentVariableRows(rows([['', 'value']]));

    expect(result).toMatchObject({ valid: false, errors: { 'env-0': 'required' } });
  });

  it('reports duplicate keys on every conflicting row', () => {
    const result = validateEnvironmentVariableRows(
      rows([
        ['TOKEN', 'one'],
        [' TOKEN ', 'two'],
      ])
    );

    expect(result).toMatchObject({
      valid: false,
      errors: { 'env-0': 'duplicate', 'env-1': 'duplicate' },
    });
  });

  it('applies a caller-provided key rule', () => {
    const result = validateEnvironmentVariableRows(rows([['invalid.key', 'value']]), (key) =>
      /^[A-Za-z_][A-Za-z0-9_]*$/.test(key) ? undefined : 'invalid'
    );

    expect(result).toMatchObject({ valid: false, errors: { 'env-0': 'invalid' } });
  });

  it('treats a fully blank draft row as unchanged', () => {
    expect(environmentVariableRowsEqual(rows([['', '']]), [])).toBe(true);
  });
});
