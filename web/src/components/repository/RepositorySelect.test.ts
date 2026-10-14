// @vitest-environment happy-dom
/* eslint-disable vue/one-component-per-file -- The test mounts the subject and a lightweight child double. */
import { createApp, type App, type PropType } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { repositoryApi } from '@/api/repository/repository';
import type { RepositoryResp } from '@/gen/proto/orbit/v1/repository/repository';
import RepositorySelect from '@/components/repository/RepositorySelect.vue';
import type { ComboboxOption } from '@/components/ui/ComboboxSelect.vue';

vi.mock('@/api/repository/repository', () => ({
  repositoryApi: { list: vi.fn(), get: vi.fn() },
}));
vi.mock('@/components/ui/ComboboxSelect.vue', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    default: defineComponent({
      props: {
        options: { type: Array as PropType<ComboboxOption[]>, required: true },
        filterOptions: { type: Boolean, required: true },
      },
      setup(props) {
        return () =>
          h(
            'div',
            { 'data-filter-options': String(props.filterOptions) },
            props.options.map((option) => option.label).join(',')
          );
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

function getTarget() {
  if (!target) {
    throw new Error('Test mount target is not initialized');
  }
  return target;
}

describe('RepositorySelect', () => {
  it('keeps a selected repository visible when it is outside the first page', async () => {
    vi.mocked(repositoryApi.get).mockResolvedValue(repository('selected', '已选仓库'));
    vi.mocked(repositoryApi.list).mockResolvedValue(page([repository('first', '第一页仓库')], 101));

    app = createApp(RepositorySelect, {
      modelValue: 'selected',
      projectId: 'project-a',
    });
    app.mount(getTarget());
    await vi.waitFor(() => {
      expect(target?.textContent).toContain('已选仓库');
      expect(target?.textContent).toContain('第一页仓库');
    });
    expect(
      target?.querySelector('[data-filter-options]')?.getAttribute('data-filter-options')
    ).toBe('false');
  });
});
