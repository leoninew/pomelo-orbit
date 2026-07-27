import { describe, expect, it } from 'vitest';
import { parseRuntimeEnvRows, serializeRuntimeEnvRows, validateRuntimeEnvRows } from './runtimeEnv';

describe('runtimeEnv', () => {
  it('parses and serializes runtime environment values in a stable order', () => {
    const rows = parseRuntimeEnvRows('{"Z_TOKEN":"z","API_TOKEN":"a"}');

    expect(rows).toEqual([
      { key: 'API_TOKEN', value: 'a' },
      { key: 'Z_TOKEN', value: 'z' },
    ]);
    expect(serializeRuntimeEnvRows(rows)).toBe('{"API_TOKEN":"a","Z_TOKEN":"z"}');
  });

  it('rejects invalid, duplicate, and empty environment values', () => {
    expect(validateRuntimeEnvRows([])).toBeTruthy();
    expect(validateRuntimeEnvRows([{ key: '1INVALID', value: 'value' }])).toBeTruthy();
    expect(
      validateRuntimeEnvRows([
        { key: 'TOKEN', value: 'one' },
        { key: 'TOKEN', value: 'two' },
      ])
    ).toBeTruthy();
    expect(validateRuntimeEnvRows([{ key: 'TOKEN', value: '' }])).toBeTruthy();
  });
});
