import type { VersionResp } from '@/gen/proto/orbit/v1/application/version';

export interface VersionTreeSelection {
  versionId: string;
  componentId?: string;
}

export type VersionTreeLoadState = 'loading' | 'loaded' | 'error';

export function resolveVersionTreeSelection(
  versions: VersionResp[],
  selection: VersionTreeSelection | undefined
): VersionTreeSelection | undefined {
  if (selection && versions.some((version) => version.id === selection.versionId)) {
    return selection;
  }
  return versions[0] ? { versionId: versions[0].id } : undefined;
}
