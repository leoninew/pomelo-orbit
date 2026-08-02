export interface EnvironmentVariableEntry {
  key: string;
  value: string;
}

export interface EnvironmentVariableListRow extends EnvironmentVariableEntry {
  id: string;
}

export type EnvironmentVariableRowErrors = Record<string, string>;

export type EnvironmentVariableKeyValidator = (key: string) => string | undefined;

export interface EnvironmentVariableListValidation {
  entries: EnvironmentVariableEntry[];
  errors: EnvironmentVariableRowErrors;
  rows: EnvironmentVariableListRow[];
  valid: boolean;
}

export function environmentVariableRowsFromEntries(
  entries: EnvironmentVariableEntry[],
  prefix: string
): EnvironmentVariableListRow[] {
  return entries.map((entry, index) => ({
    id: `${prefix}-${index}`,
    key: entry.key,
    value: entry.value,
  }));
}

export function cloneEnvironmentVariableRows(
  rows: EnvironmentVariableListRow[]
): EnvironmentVariableListRow[] {
  return rows.map((row) => ({ ...row }));
}

export function validateEnvironmentVariableRows(
  rows: EnvironmentVariableListRow[],
  validateKey?: EnvironmentVariableKeyValidator
): EnvironmentVariableListValidation {
  const activeRows = rows
    .filter((row) => row.key.trim() !== '' || row.value !== '')
    .map((row) => ({ ...row, key: row.key.trim() }));
  const errors: EnvironmentVariableRowErrors = {};
  const keys = new Map<string, EnvironmentVariableListRow[]>();

  for (const row of activeRows) {
    if (!row.key) {
      errors[row.id] = 'required';
      continue;
    }
    const keyError = validateKey?.(row.key);
    if (keyError) {
      errors[row.id] = keyError;
      continue;
    }
    const items = keys.get(row.key) ?? [];
    items.push(row);
    keys.set(row.key, items);
  }

  for (const items of keys.values()) {
    if (items.length > 1) {
      for (const row of items) {
        errors[row.id] = 'duplicate';
      }
    }
  }

  return {
    rows: activeRows,
    entries: activeRows.map(({ key, value }) => ({ key, value })),
    errors,
    valid: Object.keys(errors).length === 0,
  };
}

export function environmentVariableRowsEqual(
  left: EnvironmentVariableListRow[],
  right: EnvironmentVariableListRow[]
): boolean {
  const normalize = (rows: EnvironmentVariableListRow[]) =>
    rows
      .filter((row) => row.key.trim() !== '' || row.value !== '')
      .map((row) => ({ key: row.key.trim(), value: row.value }));
  return JSON.stringify(normalize(left)) === JSON.stringify(normalize(right));
}
