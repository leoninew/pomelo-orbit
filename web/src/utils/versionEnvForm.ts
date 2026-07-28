export interface VersionEnvFormRow {
  key: string;
  value: string;
}

export function parseVersionEnvJson(raw?: string): VersionEnvFormRow[] {
  if (raw === undefined || raw === '') {
    return [];
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    throw new Error('Version environment variables must be valid JSON');
  }
  if (!Array.isArray(parsed)) {
    throw new Error('Version environment variables must be a JSON array');
  }
  return parsed.map((item) => {
    if (
      item === null ||
      typeof item !== 'object' ||
      typeof (item as { key?: unknown }).key !== 'string' ||
      typeof (item as { value?: unknown }).value !== 'string'
    ) {
      throw new Error('Version environment variable rows require string key and value fields');
    }
    const row = item as { key: string; value: string };
    return { key: row.key, value: row.value };
  });
}

export function serializeVersionEnvRows(rows: VersionEnvFormRow[]): string {
  if (rows.some((row) => row.key === '')) {
    throw new Error('Version environment variable keys are required');
  }
  return JSON.stringify(rows);
}
