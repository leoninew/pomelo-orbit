import { describe, expect, it } from 'vitest';
import { buildSshKeyComment } from '@/utils/sshKeyComment';

describe('SSH key attribution', () => {
  it('identifies the key creation time, site, runtime host, project, and environment', () => {
    const comment = buildSshKeyComment({
      keyCreatedAt: '2026-10-15T01:02:03Z',
      site: 'https://orbit.example.com',
      sourceHost: 'orbit-control',
      projectName: 'Production',
      projectId: 'project-1',
      environmentId: 'environment-1',
    });

    expect(comment).toBe(
      '# Pomelo Orbit | key_created_at="2026-10-15T01:02:03Z" | site="https://orbit.example.com" | source_host="orbit-control" | project="Production" | project_id="project-1" | environment_id="environment-1"'
    );
  });

  it('keeps project names on one comment line', () => {
    const comment = buildSshKeyComment({
      keyCreatedAt: '2026-10-15T01:02:03Z',
      site: 'https://orbit.example.com',
      sourceHost: 'orbit-control',
      projectName: 'Team\r\n"deploy"',
      projectId: 'project-1',
      environmentId: 'environment-1',
    });

    expect(comment?.split(/\r?\n/)).toHaveLength(1);
    expect(comment).toContain('project="Team\\r\\n\\"deploy\\""');
  });

  it.each([
    { keyCreatedAt: undefined, sourceHost: 'orbit-control' },
    { keyCreatedAt: '2026-10-15T01:02:03Z', sourceHost: undefined },
  ])('does not create a comment when the backend omits attribution metadata: %j', (metadata) => {
    expect(
      buildSshKeyComment({
        ...metadata,
        site: 'https://orbit.example.com',
        projectName: 'Production',
        projectId: 'project-1',
        environmentId: 'environment-1',
      })
    ).toBeNull();
  });
});
