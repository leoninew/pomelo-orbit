export const DEFAULT_READY_REDIRECT = '/gateway';

export function isProjectInitializationPath(path: string): boolean {
  return /^\/project\/[^/]+\/initialization\/?$/.test(path);
}

/**
 * Resolve the destination after project initialization has completed.
 *
 * A completed initialization always has a managed Gateway. Opening its detail
 * page gives the user the expected post-setup view, including the default
 * Service's deploy action. The return URL used to enter the wizard is not the
 * completion destination.
 */
export function resolveInitializationCompletionRedirect(): string {
  return DEFAULT_READY_REDIRECT;
}
