import type { Router } from 'vue-router';
import { projectInitializationApi } from '@/api/project/initialization';
import type { ProjectInitializationStatusResp } from '@/gen/proto/orbit/v1/project_initialization/project_initialization';

export const DEFAULT_READY_REDIRECT = '/gateway';

export function isProjectInitializationPath(path: string): boolean {
  return /^\/project\/[^/]+\/initialization\/?$/.test(path);
}

/**
 * Resolve the destination after project initialization has completed.
 *
 * On normal setup, open the managed Gateway. A CI or CD action may return to
 * its originating page after initialization; external and wizard URLs are ignored.
 */
export function resolveInitializationCompletionRedirect(returnTo?: unknown): string {
  if (
    typeof returnTo === 'string' &&
    returnTo.startsWith('/') &&
    !returnTo.startsWith('//') &&
    !isProjectInitializationPath(returnTo.split('?')[0])
  ) {
    return returnTo;
  }
  return DEFAULT_READY_REDIRECT;
}

export type ExecutionPurpose = 'ci' | 'cd';

export function isExecutionEnvironmentReady(
  status: ProjectInitializationStatusResp,
  purpose: ExecutionPurpose
): boolean {
  return status.status === 'ready' || (purpose === 'ci' && status.status === 'needs_gateway');
}

export async function ensureProjectExecutionReady(
  projectId: string,
  router: Router,
  purpose: ExecutionPurpose
): Promise<boolean> {
  const status = await projectInitializationApi.getStatus(projectId);
  if (isExecutionEnvironmentReady(status, purpose)) {
    return true;
  }
  await router.push({
    name: 'ProjectInitialization',
    params: { id: projectId },
    query: { purpose, returnTo: router.currentRoute.value.fullPath },
  });
  return false;
}
