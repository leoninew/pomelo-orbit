import type { EnvironmentVariableListRow } from '@/components/environmentVariableList';
import type { ServiceComponentEnvOverlay } from '@/gen/proto/orbit/v1/service/service';

export interface ServiceComponentEnvironmentRow {
  key: string;
  base: string;
  inheritedValue: string;
  value: string;
  deleted: boolean;
  overridden: boolean;
}

export function componentEnvironmentListRows(
  rows: ServiceComponentEnvironmentRow[]
): EnvironmentVariableListRow[] {
  return rows.map((row) => ({ id: row.key, key: row.key, value: row.value }));
}

export function componentEnvironmentRowsEqual(
  left: ServiceComponentEnvironmentRow[],
  right: ServiceComponentEnvironmentRow[]
): boolean {
  const rightByKey = new Map(right.map((row) => [row.key, row]));
  return (
    left.length === right.length &&
    left.every((row) => {
      const saved = rightByKey.get(row.key);
      return (
        saved !== undefined &&
        row.value === saved.value &&
        row.deleted === saved.deleted &&
        row.overridden === saved.overridden
      );
    })
  );
}

export function updateComponentEnvironmentRows(
  currentRows: ServiceComponentEnvironmentRow[],
  editorRows: EnvironmentVariableListRow[]
): ServiceComponentEnvironmentRow[] {
  const valuesByKey = new Map(editorRows.map((row) => [row.id, row.value]));
  return currentRows.map((row) => {
    const value = valuesByKey.get(row.key);
    if (value === undefined || value === row.value) return row;
    return {
      ...row,
      value,
      deleted: false,
      overridden: value !== row.inheritedValue,
    };
  });
}

export function resetComponentEnvironmentRow(
  currentRows: ServiceComponentEnvironmentRow[],
  key: string
): ServiceComponentEnvironmentRow[] {
  return currentRows.map((row) =>
    row.key === key
      ? {
          ...row,
          value: row.inheritedValue,
          deleted: false,
          overridden: false,
        }
      : row
  );
}

export function componentEnvironmentOverlays(
  rows: ServiceComponentEnvironmentRow[]
): ServiceComponentEnvOverlay[] {
  return rows.flatMap((row) => {
    if (row.deleted) return [{ key: row.key, state: 'deleted' }];
    return row.overridden ? [{ key: row.key, value: row.value, state: 'override' }] : [];
  });
}
