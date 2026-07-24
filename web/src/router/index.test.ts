// @vitest-environment happy-dom
import { describe, expect, it } from 'vitest';
import { getNavigationScope } from '@/navigation';
import router from './index';

describe('domain route cutover', () => {
  it.each([
    ['/pipeline/stage', 'PipelineStagePage'],
    ['/pipeline-run/42', 'PipelineRunDetail'],
    ['/application/42', 'ApplicationDetail'],
    ['/gateway/42/edit', 'GatewayEdit'],
    ['/route/traefik', 'TraefikRoute'],
  ])('resolves %s to %s', (path, name) => {
    expect(router.resolve(path).name).toBe(name);
  });

  it.each(['/ci/repository', '/cd/application'])('does not resolve legacy path %s', (path) => {
    expect(router.resolve(path).matched).toHaveLength(0);
    expect(getNavigationScope(path)).toBeNull();
  });
});
