// @vitest-environment happy-dom
import { createApp, h, nextTick, ref, type App } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { repositoryApi } from '@/api/repository/repository';
import type { RepositoryResp } from '@/gen/proto/orbit/v1/repository/repository';
import RepositorySelect from '@/components/RepositorySelect.vue';

vi.mock('@/api/repository/repository', () => ({
  repositoryApi: { list: vi.fn(), get: vi.fn() },
}));
vi.mock('@/components/ComboboxSelect.vue', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    default: defineComponent({
      props: ['options', 'filterOptions'],
      emits: ['search', 'load-more', 'update:modelValue'],
      setup(props, { emit }) {
        return () =>
          h('div', { 'data-filter-options': String(props.filterOptions) }, [
            h(
              'div',
              { id: 'options' },
              (props.options as { label: string }[]).map((option) => option.label).join(',')
            ),
            h('button', { id: 'search', onClick: () => emit('search', 'code-only') }),
            h('button', { id: 'more', onClick: () => emit('load-more') }),
          ]);
      },
    }),
  };
});

const repository = (id: string, name: string, code = id) => ({ id, name, code }) as RepositoryResp;
const page = (items: RepositoryResp[], total: number) => ({
  items,
  total,
  page: 1,
  per_page: 100,
  pages: Math.ceil(total / 100),
});

let app: App | undefined;
let target: HTMLDivElement | undefined;

beforeEach(() => {
  target = document.createElement('div');
  document.body.append(target);
});

afterEach(() => {
  app?.unmount();
  target?.remove();
  app = undefined;
  target = undefined;
  vi.clearAllMocks();
});

describe('RepositorySelect', () => {
  it('searches beyond the first page and keeps a selected repository visible', async () => {
    vi.mocked(repositoryApi.get).mockResolvedValue(repository('selected', '已选仓库'));
    vi.mocked(repositoryApi.list).mockImplementation(async (_projectId, params) => {
      if (params?.page === 2) {
        return page([repository('later', '第二页仓库')], 101);
      }
      if (params?.search === 'code-only') {
        return page([repository('match', '显示名称不含搜索词', 'code-only')], 101);
      }
      return page([repository('first', '第一页仓库')], 101);
    });

    app = createApp(RepositorySelect, {
      modelValue: 'selected',
      projectId: 'project-a',
    });
    app.mount(target!);
    await vi.waitFor(() => expect(target?.textContent).toContain('已选仓库'));
    expect(
      target?.querySelector('[data-filter-options]')?.getAttribute('data-filter-options')
    ).toBe('false');

    target?.querySelector<HTMLButtonElement>('#search')?.click();
    await vi.waitFor(() =>
      expect(repositoryApi.list).toHaveBeenCalledWith(
        'project-a',
        expect.objectContaining({ search: 'code-only', page: 1 })
      )
    );
    await vi.waitFor(() => expect(target?.textContent).toContain('显示名称不含搜索词'));
    target?.querySelector<HTMLButtonElement>('#more')?.click();
    await vi.waitFor(() => expect(target?.textContent).toContain('第二页仓库'));
    expect(target?.textContent).toContain('已选仓库');
  });

  it('discards a response from the previous project', async () => {
    let finishOld: ((value: ReturnType<typeof page>) => void) | undefined;
    vi.mocked(repositoryApi.list).mockImplementation((projectId) => {
      if (projectId === 'project-a') {
        return new Promise((resolve) => {
          finishOld = resolve;
        });
      }
      return Promise.resolve(page([repository('new', '新项目仓库')], 1));
    });
    const projectId = ref('project-a');
    app = createApp({
      setup: () => () =>
        h(RepositorySelect, {
          modelValue: '',
          projectId: projectId.value,
        }),
    });
    app.mount(target!);
    await vi.waitFor(() => expect(finishOld).toBeDefined());
    projectId.value = 'project-b';
    await nextTick();
    await vi.waitFor(() => expect(target?.textContent).toContain('新项目仓库'));
    finishOld?.(page([repository('old', '旧项目仓库')], 1));
    await nextTick();
    expect(target?.textContent).not.toContain('旧项目仓库');
  });
});
