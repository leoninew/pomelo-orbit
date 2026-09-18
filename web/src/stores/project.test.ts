// @vitest-environment happy-dom
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { projectApi } from '@/api/project/project';
import { useProjectStore } from './project';

vi.mock('@/api/project/project', () => ({
  projectApi: {
    list: vi.fn(),
  },
}));

describe('project store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
    vi.mocked(projectApi.list).mockReset();
  });

  it('only exposes active projects for selection and replaces a deprecated selection', async () => {
    vi.mocked(projectApi.list).mockResolvedValue({
      items: [
        {
          id: 'deprecated-project',
          name: 'Deprecated Project',
          code: 'deprecated',
          is_active: false,
          created_at: '',
          updated_at: '',
        },
        {
          id: 'active-project',
          name: 'Active Project',
          code: 'active',
          is_active: true,
          created_at: '',
          updated_at: '',
        },
      ],
    });
    const store = useProjectStore();
    store.setActiveProject('deprecated-project');

    await store.fetchProjects();

    expect(store.activeProjects.map((project) => project.id)).toEqual(['active-project']);
    expect(store.activeProjectId).toBe('active-project');
  });
});
