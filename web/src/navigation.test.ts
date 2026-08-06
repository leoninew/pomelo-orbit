import { describe, expect, it } from 'vitest';
import { getNavigationScope, primaryNavigation, secondaryNavigation } from './navigation';

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
    ['/settings', 'settings'],
    ['/pipeline/template', 'pipeline'],
    ['/pipeline-run/artifact', 'pipeline'],
    ['/application/example', 'deployment'],
    ['/gateway/edit/example', 'deployment'],
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
          expect.objectContaining({ key: 'credentials', path: '/credential' }),
        ],
      }),
      expect.objectContaining({
        key: 'pipeline',
        children: [
          expect.objectContaining({ key: 'pipelinetemplates', path: '/pipeline/template' }),
          expect.objectContaining({ key: 'buildstages', path: '/pipeline/stage' }),
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
          expect.objectContaining({ key: 'gateways', path: '/gateways' }),
          expect.objectContaining({ key: 'route', path: '/routes' }),
        ],
      }),
    ]);
  });

  it('groups system management entries in the requested order', () => {
    expect(secondaryNavigation.settings).toEqual([
      expect.objectContaining({
        key: 'admin',
        children: [
          expect.objectContaining({ key: 'projects', path: '/projects' }),
          expect.objectContaining({ key: 'users', path: '/users' }),
          expect.objectContaining({ key: 'roles', path: '/roles' }),
          expect.objectContaining({ key: 'loginhistory', path: '/login-history' }),
          expect.objectContaining({ key: 'settings', path: '/settings' }),
        ],
      }),
    ]);
  });

  it('uses only domain paths in secondary navigation', () => {
    const paths = Object.values(secondaryNavigation)
      .flat()
      .flatMap((branch) => branch.children.map((entry) => entry.path));

    expect(paths).not.toContainEqual(expect.stringMatching(/^\/(?:ci|cd)(?:\/|$)/));
  });
});
