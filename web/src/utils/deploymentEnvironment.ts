const defaultWorkspaceRoot = '~/.pomelo-orbit';

export function defaultDeploymentWorkspaceRoot(): string {
  return defaultWorkspaceRoot;
}

export function isPlatformWorkspaceRoot(platform: string, workspaceRoot: string): boolean {
  const value = workspaceRoot.trim();
  if (isHomeWorkspaceRoot(value)) return true;
  return isAbsoluteWorkspaceRoot(value, platform);
}

export function isLocalWorkspaceRoot(workspaceRoot: string): boolean {
  return (
    isPlatformWorkspaceRoot('linux', workspaceRoot) ||
    isPlatformWorkspaceRoot('windows', workspaceRoot)
  );
}

function isAbsoluteWorkspaceRoot(workspaceRoot: string, platform: string): boolean {
  if (platform === 'linux') return workspaceRoot.startsWith('/');
  if (platform === 'windows') return /^[A-Za-z]:\\/.test(workspaceRoot);
  return false;
}

function isHomeWorkspaceRoot(workspaceRoot: string): boolean {
  return workspaceRoot === '~' || workspaceRoot.startsWith('~/');
}
