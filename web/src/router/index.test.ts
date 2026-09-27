// @vitest-environment happy-dom
import { describe, expect, it } from 'vitest';
import { getNavigationScope } from '@/navigation';
import router from './index';

describe('domain routes', () => {
  it.each([
    ['/pipeline', 'Pipelines'],
    ['/pipeline-run/42', 'PipelineRunDetail'],
    ['/pipeline-run/artifact/42', 'ArtifactDetail'],
    ['/application/42', 'ApplicationDetail'],
    ['/gateway', 'Gateway'],
    ['/routes', 'Route'],
    ['/route/traefik', 'TraefikRoutes'],
    ['/route/42', 'RouteDetail'],
    ['/environment', 'Environment'],
    ['/project/abc/initialization', 'ProjectInitialization'],
  ])('resolves %s to %s', (path, name) => {
    expect(router.resolve(path).name).toBe(name);
  });

  it('selects the matching sidebar entry for each route list', () => {
    expect(router.resolve('/routes').meta.menuKey).toBe('route');
    expect(router.resolve('/route/traefik').meta.menuKey).toBe('traefikroutes');
  });

  it('places environment in the system management navigation scope', () => {
    expect(getNavigationScope('/environment')).toBe('settings');
  });
});
