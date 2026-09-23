// @vitest-environment happy-dom
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { projectApi } from '@/api/project/project';
import { ACTIVE_PROJECT_ID_KEY, ACTIVE_PROJECT_SUMMARY_KEY } from '@/constants/project';
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
    expect(store.activeProjectName).toBe('Active Project');
  });

  it('shows the selected project name immediately after a reload and refreshes it from the API', async () => {
    const firstProject = {
      id: 'first-project',
      name: 'First Project',
      code: 'first',
      is_active: true,
      created_at: '',
      updated_at: '',
    };
    const secondProject = {
      ...firstProject,
      id: 'second-project',
      name: 'Second Project',
      code: 'second',
    };
    vi.mocked(projectApi.list).mockResolvedValueOnce({ items: [firstProject, secondProject] });
    const store = useProjectStore();
    await store.fetchProjects();
    store.setActiveProject(secondProject.id);

    expect(JSON.parse(localStorage.getItem(ACTIVE_PROJECT_ID_KEY) || 'null')).toBe(
      secondProject.id
    );
    expect(JSON.parse(localStorage.getItem(ACTIVE_PROJECT_SUMMARY_KEY) || 'null')).toEqual({
      id: secondProject.id,
      name: secondProject.name,
    });

    setActivePinia(createPinia());
    const restored = useProjectStore();
    expect(restored.activeProject).toBeUndefined();
    expect(restored.activeProjectId).toBe(secondProject.id);
    expect(restored.activeProjectName).toBe('Second Project');

    vi.mocked(projectApi.list).mockResolvedValueOnce({
      items: [firstProject, { ...secondProject, name: 'Renamed Project' }],
    });
    await restored.fetchProjects();
    expect(restored.activeProjectName).toBe('Renamed Project');
    expect(JSON.parse(localStorage.getItem(ACTIVE_PROJECT_SUMMARY_KEY) || 'null')).toEqual({
      id: secondProject.id,
      name: 'Renamed Project',
    });
  });
});
