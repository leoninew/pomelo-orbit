<template>
  <div
    v-if="totalPages > 0"
    class="overflow-x-auto"
    :class="standalone ? '' : 'border-t border-border px-5 py-3 sm:px-6'"
  >
    <div class="flex min-w-max items-center justify-end gap-3">
      <SelectControl
        :model-value="pageSize"
        :options="pageSizeSelectOptions"
        width-class="h-9 w-28"
        @update:model-value="emit('change-page-size', Number($event))"
      />
      <div class="flex items-center gap-1">
        <button
          class="flex size-9 items-center justify-center rounded-lg border border-input bg-background text-muted-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="current <= 1"
          :aria-label="t('common.previousPage')"
          :title="t('common.previousPage')"
          @click="goPage(current - 1)"
        >
          <ChevronLeft class="size-4" />
        </button>
        <template v-if="totalPages <= 7">
          <button
            v-for="page in totalPages"
            :key="page"
            class="flex size-9 items-center justify-center rounded-lg border text-sm transition-colors"
            :class="
              page === current
                ? 'border-primary bg-primary text-primary-foreground'
                : 'border-input bg-background text-foreground hover:bg-muted/50'
            "
            :aria-current="page === current ? 'page' : undefined"
            @click="goPage(page)"
          >
            {{ page }}
          </button>
        </template>
        <template v-else>
          <button
            v-for="(page, index) in visiblePages"
            :key="page === -1 ? `ellipsis-${index}` : page"
            class="flex size-9 items-center justify-center rounded-lg border text-sm transition-colors"
            :class="
              page === current
                ? 'border-primary bg-primary text-primary-foreground'
                : page === -1
                  ? 'border-transparent bg-transparent text-muted-foreground cursor-default'
                  : 'border-input bg-background text-foreground hover:bg-muted/50'
            "
            :disabled="page === -1"
            :aria-current="page === current ? 'page' : undefined"
            @click="page !== -1 && goPage(page)"
          >
            {{ page === -1 ? '...' : page }}
          </button>
        </template>
        <button
          class="flex size-9 items-center justify-center rounded-lg border border-input bg-background text-muted-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="current >= totalPages"
          :aria-label="t('common.nextPage')"
          :title="t('common.nextPage')"
          @click="goPage(current + 1)"
        >
          <ChevronRight class="size-4" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ChevronLeft, ChevronRight } from '@lucide/vue';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import SelectControl from '@/components/SelectControl.vue';

  const { t } = useI18n();

  const props = withDefaults(
    defineProps<{
      current: number;
      pageSize: number;
      total: number;
      totalPages: number;
      pageSizeOptions?: number[];
      standalone?: boolean;
    }>(),
    {
      pageSizeOptions: () => [10, 20, 50],
      standalone: false,
    }
  );

  const emit = defineEmits<{
    'change-page': [page: number];
    'change-page-size': [pageSize: number];
  }>();

  function goPage(page: number) {
    if (page < 1 || page > props.totalPages || page === props.current) {
      return;
    }
    emit('change-page', page);
  }

  const pageSizeSelectOptions = computed(() =>
    props.pageSizeOptions.map((size) => ({
      value: size,
      label: t('common.perPage', { size }),
    }))
  );

  // 计算可见页码，支持省略号显示
  const visiblePages = computed(() => {
    const pages: number[] = [];
    const { current, totalPages } = props;

    if (totalPages <= 7) {
      // 7页以内全部显示
      for (let i = 1; i <= totalPages; i++) {
        pages.push(i);
      }
    } else {
      // 总是显示第一页
      pages.push(1);

      if (current <= 3) {
        // 当前页在前面
        pages.push(2, 3, 4, -1, totalPages);
      } else if (current >= totalPages - 2) {
        // 当前页在后面
        pages.push(-1, totalPages - 3, totalPages - 2, totalPages - 1, totalPages);
      } else {
        // 当前页在中间
        pages.push(-1, current - 1, current, current + 1, -1, totalPages);
      }
    }

    return pages;
  });
</script>
