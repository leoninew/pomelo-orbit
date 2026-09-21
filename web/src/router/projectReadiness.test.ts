import { describe, expect, it } from 'vitest';
import {
  DEFAULT_READY_REDIRECT,
  isProjectInitializationPath,
  resolveInitializationCompletionRedirect,
} from './projectReadiness';

describe('project readiness routing', () => {
  it('recognizes the wizard path', () => {
    expect(isProjectInitializationPath('/project/abc/initialization')).toBe(true);
    expect(isProjectInitializationPath('/gateway')).toBe(false);
  });

  it('opens the initialized gateway detail after setup completes', () => {
    expect(resolveInitializationCompletionRedirect()).toBe(DEFAULT_READY_REDIRECT);
  });
});
