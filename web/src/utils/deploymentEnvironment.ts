const defaultWorkspaceRoot = '~/.pomelo-orbit';

export function defaultDeploymentWorkspaceRoot(): string {
  return defaultWorkspaceRoot;
}

export function isPlatformWorkspaceRoot(platform: string, workspaceRoot: string): boolean {
  const value = workspaceRoot.trim();
  if (isHomeWorkspaceRoot(value)) return true;
  if (platform === 'linux') return value.startsWith('/');
  if (platform === 'windows') return /^[A-Za-z]:\\/.test(value);
  return false;
}

function isHomeWorkspaceRoot(workspaceRoot: string): boolean {
  return workspaceRoot === '~' || workspaceRoot.startsWith('~/');
}
