import { describe, expect, it } from 'vitest';
import { parseVersionEnvJson, serializeVersionEnvRows } from './versionEnvForm';

describe('versionEnvForm', () => {
  it('preserves environment text exactly', () => {
    const rows = parseVersionEnvJson('[{"key":" KEY ","value":" value "}]');

    expect(rows).toEqual([{ key: ' KEY ', value: ' value ' }]);
    expect(serializeVersionEnvRows(rows)).toBe('[{"key":" KEY ","value":" value "}]');
  });

  it('rejects malformed or incomplete JSON instead of discarding it', () => {
    expect(() => parseVersionEnvJson('{"KEY":"value"}')).toThrow('JSON array');
    expect(() => parseVersionEnvJson('[{"key":"KEY"}]')).toThrow('key and value');
    expect(() => serializeVersionEnvRows([{ key: '', value: 'value' }])).toThrow(
      'keys are required'
    );
  });
});
