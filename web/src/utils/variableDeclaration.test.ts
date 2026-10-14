import { describe, expect, it } from 'vitest';
import { effectiveVariableValue } from './variableDeclaration';

describe('effectiveVariableValue', () => {
  it('uses the default when the current value is blank', () => {
    expect(effectiveVariableValue({ value: '  ', default: 'default-value' })).toBe('default-value');
  });

  it('keeps the current value when it is set', () => {
    expect(effectiveVariableValue({ value: 'current-value', default: 'default-value' })).toBe(
      'current-value'
    );
  });

  it('returns undefined when no configuration is supplied', () => {
    expect(effectiveVariableValue()).toBeUndefined();
  });
});
