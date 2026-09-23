import { describe, expect, it } from 'vitest';
import { PERMISSIONS } from '@/constants/permissions';
import {
  getNavigationScope,
  getSecondaryNavigationTitle,
  primaryNavigation,
  secondaryNavigation,
} from './navigation';

describe('domain navigation declarations', () => {
  it('keeps primary navigation scoped to continuous integration, deployment, and system management', () => {
    expect(primaryNavigation).toEqual([
      expect.objectContaining({ key: 'pipeline', path: '/repository' }),
      expect.objectContaining({ key: 'deployment', path: '/dialogue' }),
      expect.objectContaining({
        key: 'settings',
        labelKey: 'nav.systemManagement',
        path: '/projects',
      }),
    ]);
  });

  it.each([
    ['/', 'home'],
    ['/projects', 'settings'],
    ['/users', 'settings'],
    ['/mcp-access-token', 'settings'],
    ['/settings', 'settings'],
    ['/environment', 'settings'],
    ['/pipeline/example', 'pipeline'],
    ['/pipeline-run/artifact', 'pipeline'],
    ['/application/example', 'deployment'],
    ['/gateway', 'deployment'],
    ['/route/traefik', 'deployment'],
  ] as const)('classifies %s as the %s scope', (path, scope) => {
    expect(getNavigationScope(path)).toBe(scope);
  });

  it('groups continuous integration entries into code and pipeline', () => {
    expect(secondaryNavigation.pipeline).toEqual([
      expect.objectContaining({
        key: 'code',
        children: [
          expect.objectContaining({ key: 'repository', path: '/repository' }),
          expect.objectContaining({ key: 'credentials', path: '/repository-credential' }),
        ],
      }),
      expect.objectContaining({
        key: 'pipeline',
        children: [
          expect.objectContaining({ key: 'pipelines', path: '/pipeline' }),
          expect.objectContaining({ key: 'pipelinestages', path: '/pipeline-stage' }),
          expect.objectContaining({ key: 'pipelineruns', path: '/pipeline-run' }),
          expect.objectContaining({ key: 'artifacts', path: '/pipeline-run/artifact' }),
        ],
      }),
    ]);
  });

  it('groups continuous deployment entries into delivery and ingress flows', () => {
    expect(secondaryNavigation.deployment).toEqual([
      expect.objectContaining({
        key: 'delivery',
        children: [
          expect.objectContaining({ key: 'deployment-dialogue', path: '/dialogue' }),
          expect.objectContaining({ key: 'applications', path: '/applications' }),
          expect.objectContaining({ key: 'services', path: '/services' }),
          expect.objectContaining({ key: 'deployments', path: '/deployments' }),
        ],
      }),
      expect.objectContaining({
        key: 'ingress',
        children: [
          expect.objectContaining({ key: 'gateway', path: '/gateway' }),
          expect.objectContaining({ key: 'route', path: '/routes' }),
        ],
      }),
    ]);
  });

  it('builds continuous integration and deployment titles from the selected menu path', () => {
    const labels: Record<string, string> = {
      'nav.groups.code': '代码仓库',
      'nav.repositories': '仓库',
      'nav.groups.ingress': '网络接入',
      'nav.gateway': '网关',
      'nav.groups.project': '项目',
      'nav.groups.admin': '系统管理',
      'nav.settings': '系统设置',
      'nav.environment': '环境',
    };
    const t = (key: string) => labels[key] || key;

    expect(getSecondaryNavigationTitle('pipeline', 'repository', t)).toBe('代码仓库 仓库');
    expect(getSecondaryNavigationTitle('deployment', 'gateway', t)).toBe('网络接入 网关');
    expect(getSecondaryNavigationTitle('settings', 'settings', t)).toBe('系统管理 系统设置');
    expect(getSecondaryNavigationTitle('settings', 'environment', t)).toBe('项目 环境');
    expect(getSecondaryNavigationTitle('pipeline', 'missing', t)).toBeNull();
  });

  it('groups project resources before system management entries', () => {
    expect(secondaryNavigation.settings).toEqual([
      expect.objectContaining({
        key: 'project',
        children: [
          expect.objectContaining({ key: 'projects', path: '/projects' }),
          expect.objectContaining({ key: 'environment', path: '/environment' }),
        ],
      }),
      expect.objectContaining({
        key: 'admin',
        children: [
          expect.objectContaining({ key: 'users', path: '/users' }),
          expect.objectContaining({ key: 'roles', path: '/roles' }),
          expect.objectContaining({ key: 'loginhistory', path: '/login-history' }),
          expect.objectContaining({ key: 'mcpAccessTokens', path: '/mcp-access-token' }),
          expect.objectContaining({ key: 'settings', path: '/settings' }),
        ],
      }),
    ]);
  });

  it('keeps project environment accessible without system settings permission', () => {
    const projectNavigation = secondaryNavigation.settings.find((entry) => entry.key === 'project');
    const adminNavigation = secondaryNavigation.settings.find((entry) => entry.key === 'admin');
    const environment = projectNavigation?.children.find((entry) => entry.key === 'environment');
    const settings = adminNavigation?.children.find((entry) => entry.key === 'settings');
    expect(environment?.permission).toBeUndefined();
    expect(settings?.permission).toBe(PERMISSIONS.SETTING_READ);
  });
});
