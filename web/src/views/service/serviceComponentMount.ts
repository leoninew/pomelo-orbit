import type {
  ServiceComponentDeclaredMount,
  ServiceComponentMountOverlay,
} from '@/gen/proto/orbit/v1/service/service';

export const mountFields = [
  'source',
  'source_is_host_path',
  'read_only',
  'content',
  'mode',
  'ignore_if_exists',
] as const;

export type MountField = (typeof mountFields)[number];
type MountValues = Pick<ServiceComponentDeclaredMount, MountField>;

export interface ServiceComponentMountRow {
  target: string;
  source_type: string;
  shared: boolean;
  base: MountValues;
  values: MountValues;
  overridden: Record<MountField, boolean>;
  deleted: boolean;
}

export function componentMountRows(
  declarations: ServiceComponentDeclaredMount[],
  overlays: ServiceComponentMountOverlay[]
): ServiceComponentMountRow[] {
  return declarations.map((declaration) => {
    const overlay = overlays.find((item) => item.target === declaration.target);
    const base: MountValues = {
      source: declaration.source,
      source_is_host_path: declaration.source_is_host_path,
      read_only: declaration.read_only,
      content: declaration.content,
      mode: declaration.mode,
      ignore_if_exists: declaration.ignore_if_exists,
    };
    return {
      target: declaration.target,
      source_type: declaration.source_type,
      shared: declaration.shared,
      base,
      values: {
        source: overlay?.source ?? base.source,
        source_is_host_path: overlay?.source_is_host_path ?? base.source_is_host_path,
        read_only: overlay?.read_only ?? base.read_only,
        content: overlay?.content ?? base.content,
        mode: overlay?.mode ?? base.mode,
        ignore_if_exists: overlay?.ignore_if_exists ?? base.ignore_if_exists,
      },
      overridden: Object.fromEntries(
        mountFields.map((field) => [field, overlay?.[field] !== undefined])
      ) as Record<MountField, boolean>,
      deleted: overlay?.state === 'deleted',
    };
  });
}

export function cloneComponentMount(row: ServiceComponentMountRow): ServiceComponentMountRow {
  return {
    ...row,
    base: { ...row.base },
    values: { ...row.values },
    overridden: { ...row.overridden },
  };
}

export function setMountFieldValue<Field extends MountField>(
  row: ServiceComponentMountRow,
  field: Field,
  value: MountValues[Field]
): void {
  row.values[field] = value;
  row.overridden[field] = true;
}

export function resetComponentMount(row: ServiceComponentMountRow): void {
  row.values = { ...row.base };
  mountFields.forEach((field) => {
    row.overridden[field] = false;
  });
  row.deleted = false;
}

export function componentMountHasOverrides(row: ServiceComponentMountRow): boolean {
  return mountFields.some((field) => row.overridden[field]);
}

export function componentMountOverlays(
  rows: ServiceComponentMountRow[]
): ServiceComponentMountOverlay[] {
  return rows.flatMap((row) => {
    if (row.deleted) {
      return [{ target: row.target, state: 'deleted' }];
    }
    if (!componentMountHasOverrides(row)) {
      return [];
    }
    return [
      {
        target: row.target,
        state: 'override',
        ...Object.fromEntries(
          mountFields
            .filter((field) => row.overridden[field])
            .map((field) => [field, row.values[field]])
        ),
      },
    ];
  });
}

export type MountErrors = Partial<Record<'source' | 'mode' | 'content', string>>;

export function validateComponentMount(row: ServiceComponentMountRow): MountErrors {
  const errors: MountErrors = {};
  const source = row.values.source;
  const absolute = source.startsWith('/') || /^[A-Za-z]:\//.test(source);
  if (!source.trim()) {
    errors.source = 'service.componentDetail.validation.sourceRequired';
  } else if (row.source_type === 'named_volume') {
    if (/[\\/]/.test(source)) {
      errors.source = 'service.componentDetail.validation.volumeName';
    }
  } else if (
    (!absolute &&
      !(source.startsWith('./') && !source.includes('\\') && !source.split('/').includes('..'))) ||
    (row.values.source_is_host_path && !absolute)
  ) {
    errors.source = 'service.componentDetail.validation.sourceMustStartWithDotSlash';
  } else if (row.shared && !absolute) {
    errors.source = 'application.componentDetail.validation.sharedSource';
  }
  if (row.source_type === 'controlled_file') {
    if (!/^0[0-7]{3}$/.test(row.values.mode)) {
      errors.mode = 'service.componentDetail.validation.fileMode';
    }
    if (new TextEncoder().encode(row.values.content).length > 256 * 1024) {
      errors.content = 'service.componentDetail.validation.fileSize';
    }
  }
  return errors;
}
