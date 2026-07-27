export interface RuntimeEnvRow {
  key: string;
  value: string;
}

const runtimeEnvKeyPattern = /^[A-Za-z_][A-Za-z0-9_]*$/;

export function parseRuntimeEnvRows(raw?: string): RuntimeEnvRow[] {
  if (!raw?.trim()) {
    return [];
  }
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
      return [];
    }
    return Object.entries(parsed as Record<string, unknown>)
      .filter(([, value]) => typeof value === 'string')
      .map(([key, value]) => ({ key, value: String(value) }))
      .sort((left, right) => left.key.localeCompare(right.key));
  } catch {
    return [];
  }
}

export function validateRuntimeEnvRows(rows: RuntimeEnvRow[]): string | undefined {
  if (rows.length === 0) {
    return '至少添加一个环境变量';
  }
  const keys = new Set<string>();
  for (const row of rows) {
    const key = row.key.trim();
    if (!runtimeEnvKeyPattern.test(key)) {
      return '环境变量名必须以字母或下划线开头，只能包含字母、数字和下划线';
    }
    if (keys.has(key)) {
      return '环境变量名不能重复';
    }
    if (!row.value) {
      return '环境变量值不能为空';
    }
    keys.add(key);
  }
  return undefined;
}

export function serializeRuntimeEnvRows(rows: RuntimeEnvRow[]): string | undefined {
  const values: Record<string, string> = {};
  for (const row of rows) {
    const key = row.key.trim();
    if (key) {
      values[key] = row.value;
    }
  }
  return Object.keys(values).length > 0 ? JSON.stringify(values) : undefined;
}
