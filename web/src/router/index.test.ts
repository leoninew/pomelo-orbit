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
    ['/gateway/42', 'GatewayDetail'],
    ['/routes', 'Route'],
    ['/environment', 'Environment'],
    ['/project/abc/initialization', 'ProjectInitialization'],
  ])('resolves %s to %s', (path, name) => {
    expect(router.resolve(path).name).toBe(name);
  });

  it('places environment in the deployment navigation scope', () => {
    expect(getNavigationScope('/environment')).toBe('deployment');
  });
});
