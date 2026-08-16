// @vitest-environment happy-dom
import { describe, expect, it } from 'vitest';
import router from './index';

describe('domain routes', () => {
  it.each([
    ['/pipeline', 'Pipelines'],
    ['/pipeline-run/42', 'PipelineRunDetail'],
    ['/pipeline-run/artifact/42', 'ArtifactDetail'],
    ['/application/42', 'ApplicationDetail'],
    ['/gateway/edit/42', 'GatewayEdit'],
    ['/routes', 'Route'],
  ])('resolves %s to %s', (path, name) => {
    expect(router.resolve(path).name).toBe(name);
  });
});
