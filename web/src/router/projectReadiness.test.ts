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

  it('guards service and gateway routes only', () => {
    expect(isProjectReadinessGuardedPath('/services')).toBe(true);
    expect(isProjectReadinessGuardedPath('/service/42')).toBe(true);
    expect(isProjectReadinessGuardedPath('/service/42/component/7')).toBe(true);
    expect(isProjectReadinessGuardedPath('/gateway')).toBe(true);
    expect(isProjectReadinessGuardedPath('/')).toBe(false);
    expect(isProjectReadinessGuardedPath('/applications')).toBe(false);
    expect(isProjectReadinessGuardedPath('/environment')).toBe(false);
    expect(isProjectReadinessGuardedPath('/pipeline')).toBe(false);
    expect(isProjectReadinessGuardedPath('/routes')).toBe(false);
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

  it('opens the initialized gateway detail after setup completes', () => {
    expect(resolveInitializationCompletionRedirect()).toBe(DEFAULT_READY_REDIRECT);
  });
});
