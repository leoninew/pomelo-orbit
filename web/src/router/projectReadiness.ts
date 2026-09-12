import { getNavigationScope } from '@/navigation';

export const PROJECT_INITIALIZATION_ROUTE = 'ProjectInitialization';
export const DEFAULT_READY_REDIRECT = '/gateways';

export function isProjectInitializationPath(path: string): boolean {
  return /^\/project\/[^/]+\/initialization\/?$/.test(path);
}

export function isProjectReadinessExemptPath(path: string): boolean {
  if (
    path === '/login' ||
    path.startsWith('/google/') ||
    path === '/403' ||
    path === '/projects' ||
    isProjectInitializationPath(path)
  ) {
    return true;
  }
  if (/^\/project\/[^/]+$/.test(path)) {
    return true;
  }
  return getNavigationScope(path) === 'settings';
}

export function isProjectReadinessGuardedPath(path: string): boolean {
  if (isProjectReadinessExemptPath(path)) {
    return false;
  }
  const scope = getNavigationScope(path);
  return scope === 'home' || scope === 'pipeline' || scope === 'deployment';
}

export function initializationPath(projectId: string): string {
  return `/project/${projectId}/initialization`;
}

export function resolveInitializationRedirect(raw: unknown): string {
  const value = Array.isArray(raw) ? raw[0] : raw;
  if (typeof value !== 'string') {
    return DEFAULT_READY_REDIRECT;
  }
  if (!value.startsWith('/') || value.startsWith('//') || value.includes('://')) {
    return DEFAULT_READY_REDIRECT;
  }
  if (isProjectInitializationPath(value.split('?')[0] ?? value)) {
    return DEFAULT_READY_REDIRECT;
  }
  return value;
}

/**
 * Resolve the destination after a project initialization has completed.
 *
 * A completed initialization always has a managed Gateway. Opening its detail
 * page gives the user the expected post-setup view (including the default
 * Service's deploy action), while the sanitized return URL remains a fallback
 * for older/incomplete responses.
 */
export function resolveInitializationCompletionRedirect(
  gatewayId: unknown,
  rawRedirect: unknown
): string {
  if (typeof gatewayId === 'string' && gatewayId.trim()) {
    return `/gateway/${encodeURIComponent(gatewayId.trim())}`;
  }
  return resolveInitializationRedirect(rawRedirect);
}
