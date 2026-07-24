import { describe, expect, it } from 'vitest';
import { getNavigationScope, primaryNavigation, secondaryNavigation } from './navigation';

describe('domain navigation declarations', () => {
  it('keeps primary navigation scoped to continuous integration, deployment, and settings', () => {
    expect(primaryNavigation).toEqual([
      expect.objectContaining({ key: 'pipeline', path: '/repository' }),
      expect.objectContaining({ key: 'deployment', path: '/applications' }),
      expect.objectContaining({ key: 'settings', path: '/users' }),
    ]);
  });

  it.each([
    ['/', 'home'],
    ['/projects', 'home'],
    ['/users', 'settings'],
    ['/settings', 'settings'],
    ['/pipeline/template', 'pipeline'],
    ['/pipeline-run/artifact', 'pipeline'],
    ['/application/example', 'deployment'],
    ['/gateway/example/edit', 'deployment'],
    ['/route/traefik', 'deployment'],
  ] as const)('classifies %s as the %s scope', (path, scope) => {
    expect(getNavigationScope(path)).toBe(scope);
  });

  it.each(['/ci/repository', '/cd/applications'])('does not classify legacy path %s', (path) => {
    expect(getNavigationScope(path)).toBeNull();
  });

  it('uses only domain paths in secondary navigation', () => {
    const paths = Object.values(secondaryNavigation)
      .flat()
      .flatMap((branch) => branch.children.map((entry) => entry.path));

    expect(paths).not.toContainEqual(expect.stringMatching(/^\/(?:ci|cd)(?:\/|$)/));
  });
});
