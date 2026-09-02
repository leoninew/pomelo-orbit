import type { Router } from 'vue-router';

const REDIRECT_BASE_URL = 'http://pomelo-orbit.local';

function localPath(redirect: unknown): string | undefined {
  if (typeof redirect !== 'string' || !redirect.startsWith('/')) {
    return undefined;
  }

  try {
    const target = new URL(redirect, REDIRECT_BASE_URL);
    if (target.origin !== REDIRECT_BASE_URL) {
      return undefined;
    }
    return `${target.pathname}${target.search}${target.hash}`;
  } catch {
    return undefined;
  }
}

/**
 * Resolves the destination after authentication. Only registered protected routes
 * are valid destinations, and nested login redirects are unwrapped once per level.
 */
export function resolveLoginRedirect(router: Pick<Router, 'resolve'>, redirect: unknown): string {
  const visited = new Set<string>();
  let target = localPath(redirect);

  while (target && !visited.has(target)) {
    visited.add(target);
    const location = router.resolve(target);

    if (location.matched.length === 0) {
      return '/';
    }
    if (location.name === 'Login') {
      target = localPath(location.query.redirect);
      continue;
    }
    if (location.matched.some((record) => record.meta.public === true)) {
      return '/';
    }
    return location.fullPath;
  }

  return '/';
}
