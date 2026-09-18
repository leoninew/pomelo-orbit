import { describe, expect, it } from 'vitest';
import { isLocalWorkspaceRoot, isPlatformWorkspaceRoot } from './deploymentEnvironment';

describe('deploymentEnvironment workspace roots', () => {
  it('allows SSH home directories and platform-absolute paths', () => {
    expect(isPlatformWorkspaceRoot('linux', '~/.pomelo-orbit')).toBe(true);
    expect(isPlatformWorkspaceRoot('windows', '~/.pomelo-orbit')).toBe(true);
    expect(isPlatformWorkspaceRoot('linux', '/srv/orbit')).toBe(true);
    expect(isPlatformWorkspaceRoot('windows', 'C:\\orbit-workspace')).toBe(true);
    expect(isPlatformWorkspaceRoot('linux', 'C:\\orbit-workspace')).toBe(false);
  });

  it('allows local home directories and absolute paths', () => {
    expect(isLocalWorkspaceRoot('~/.pomelo-orbit')).toBe(true);
    expect(isLocalWorkspaceRoot('~')).toBe(true);
    expect(isLocalWorkspaceRoot('relative/path')).toBe(false);
    expect(isLocalWorkspaceRoot('.pomelo-orbit')).toBe(false);
    expect(isLocalWorkspaceRoot('/tmp/orbit-workspace')).toBe(true);
    expect(isLocalWorkspaceRoot('C:\\orbit-workspace')).toBe(true);
  });
});
