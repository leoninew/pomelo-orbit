import { describe, expect, it } from 'vitest';
import {
  DEFAULT_READY_REDIRECT,
  isProjectInitializationPath,
  isProjectReadinessExemptPath,
  isProjectReadinessGuardedPath,
  resolveInitializationCompletionRedirect,
  resolveInitializationRedirect,
} from './projectReadiness';

describe('project readiness routing', () => {
  it('treats project management, login, and wizard routes as exempt', () => {
    expect(isProjectReadinessExemptPath('/login')).toBe(true);
    expect(isProjectReadinessExemptPath('/projects')).toBe(true);
    expect(isProjectReadinessExemptPath('/project/abc')).toBe(true);
    expect(isProjectReadinessExemptPath('/project/abc/initialization')).toBe(true);
    expect(isProjectReadinessExemptPath('/settings')).toBe(true);
    expect(isProjectReadinessExemptPath('/users')).toBe(true);
  });

  it('guards home, pipeline, and deployment routes', () => {
    expect(isProjectReadinessGuardedPath('/')).toBe(true);
    expect(isProjectReadinessGuardedPath('/applications')).toBe(true);
    expect(isProjectReadinessGuardedPath('/environment')).toBe(true);
    expect(isProjectReadinessGuardedPath('/pipeline')).toBe(true);
    expect(isProjectReadinessGuardedPath('/gateways')).toBe(true);
    expect(isProjectReadinessGuardedPath('/projects')).toBe(false);
    expect(isProjectReadinessGuardedPath('/project/abc/initialization')).toBe(false);
  });

  it('recognizes the wizard path and sanitizes return URLs', () => {
    expect(isProjectInitializationPath('/project/abc/initialization')).toBe(true);
    expect(resolveInitializationRedirect('/environment')).toBe('/environment');
    expect(resolveInitializationRedirect('https://example.test')).toBe(DEFAULT_READY_REDIRECT);
    expect(resolveInitializationRedirect('/project/abc/initialization')).toBe(
      DEFAULT_READY_REDIRECT
    );
    expect(resolveInitializationRedirect(undefined)).toBe(DEFAULT_READY_REDIRECT);
  });

  it('opens the initialized gateway detail before falling back to the return URL', () => {
    expect(resolveInitializationCompletionRedirect('gateway-42', '/')).toBe('/gateway/gateway-42');
    expect(resolveInitializationCompletionRedirect(' gateway/42 ', '/')).toBe(
      '/gateway/gateway%2F42'
    );
    expect(resolveInitializationCompletionRedirect(undefined, '/environment')).toBe('/environment');
  });
});
