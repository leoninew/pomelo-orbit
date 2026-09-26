<template>
  <ComboboxSelect
    :model-value="modelValue"
    :options="options"
    :placeholder="placeholder"
    :width-class="widthClass"
    :invalid="invalid"
    :loading="loading"
    :has-more="hasMore"
    :empty-text="loadError || '暂无数据'"
    :filter-options="false"
    @update:model-value="selectValue"
    @search="search"
    @load-more="loadMore"
  />
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue';
  import { repositoryApi } from '@/api/repository/repository';
  import ComboboxSelect, { type ComboboxOption } from '@/components/ComboboxSelect.vue';

  const props = withDefaults(
    defineProps<{
      modelValue: string;
      projectId: string | null;
      selectedLabel?: string;
      placeholder?: string;
      widthClass?: string;
      invalid?: boolean;
    }>(),
    { placeholder: '请选择', widthClass: 'w-full', invalid: false }
  );
  const emit = defineEmits<{ 'update:modelValue': [value: string] }>();
  const rows = ref<ComboboxOption[]>([]);
  const selected = ref<ComboboxOption>();
  const loading = ref(false);
  const hasMore = ref(false);
  const loadError = ref('');
  const query = ref('');
  const page = ref(0);
  let generation = 0;
  let timer: ReturnType<typeof setTimeout> | undefined;

  const options = computed(() => {
    if (!selected.value || rows.value.some((row) => row.value === selected.value?.value)) {
      return rows.value;
    }
    return [selected.value, ...rows.value];
  });

  async function load(nextPage: number) {
    const projectId = props.projectId;
    if (!projectId || loading.value) {
      return;
    }
    const requestGeneration = generation;
    loading.value = true;
    loadError.value = '';
    try {
      const params = { page: nextPage, per_page: 100, search: query.value || undefined };
      const result = await repositoryApi.list(projectId, params);
      if (requestGeneration !== generation || props.projectId !== projectId) {
        return;
      }
      const items: ComboboxOption[] = result.items.map((item) => ({
        value: item.id,
        label: item.name,
        description: item.code,
      }));
      rows.value = nextPage === 1 ? items : [...rows.value, ...items];
      page.value = nextPage;
      hasMore.value = nextPage * 100 < result.total;
    } catch (reason) {
      if (requestGeneration === generation) {
        loadError.value = reason instanceof Error ? reason.message : '加载失败';
      }
    } finally {
      if (requestGeneration === generation) {
        loading.value = false;
      }
    }
  }

  function refresh() {
    generation += 1;
    loading.value = false;
    rows.value = [];
    page.value = 0;
    hasMore.value = false;
    void load(1);
  }

  function search(value: string) {
    if (timer) {
      clearTimeout(timer);
    }
    timer = setTimeout(() => {
      if (query.value === value.trim()) {
        return;
      }
      query.value = value.trim();
      refresh();
    }, 250);
  }

  function loadMore() {
    if (hasMore.value) {
      void load(page.value + 1);
    }
  }

  function selectValue(value: string | number) {
    const id = String(value);
    selected.value = options.value.find((row) => row.value === id);
    emit('update:modelValue', id);
  }

  watch(
    () => props.projectId,
    (projectId, previous) => {
      if (timer) {
        clearTimeout(timer);
      }
      query.value = '';
      selected.value = undefined;
      if (previous && previous !== projectId) {
        emit('update:modelValue', '');
      }
      refresh();
    },
    { immediate: true }
  );

  watch(
    () => [props.modelValue, props.selectedLabel, props.projectId] as const,
    async ([id, label, projectId]) => {
      if (!id || !projectId) {
        selected.value = undefined;
        return;
      }
      const known = rows.value.find((row) => row.value === id);
      if (known) {
        selected.value = known;
        return;
      }
      if (label) {
        selected.value = { value: id, label };
        return;
      }
      const requestGeneration = generation;
      try {
        const item = await repositoryApi.get(projectId, id);
        if (requestGeneration === generation && props.modelValue === id) {
          selected.value = { value: id, label: item.name, description: item.code };
        }
      } catch {
        if (requestGeneration === generation && props.modelValue === id) {
          selected.value = undefined;
        }
      }
    },
    { immediate: true }
  );

  onBeforeUnmount(() => {
    generation += 1;
    if (timer) {
      clearTimeout(timer);
    }
  });
</script>
