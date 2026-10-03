<template>
  <span ref="container" class="block min-w-0">
    <AppTooltip :content="isTruncated ? text : undefined">
      <Primitive
        as="span"
        :as-child="asChild"
        class="block truncate"
        :class="{
          'cursor-help underline decoration-dashed underline-offset-2 focus-visible:outline focus-visible:outline-ring':
            isTruncated,
        }"
        :tabindex="isTruncated && !asChild ? 0 : undefined"
      >
        <slot>{{ text }}</slot>
      </Primitive>
    </AppTooltip>
  </span>
</template>

<script setup lang="ts">
  import { Primitive } from 'reka-ui';
  import { onBeforeUnmount, onMounted, onUpdated, ref } from 'vue';
  import AppTooltip from '@/components/AppTooltip.vue';

  defineProps<{ text?: string; asChild?: boolean }>();
  const container = ref<HTMLSpanElement>();
  const isTruncated = ref(false);
  let observer: ResizeObserver | undefined;

  function updateTruncation() {
    const element = container.value?.firstElementChild;
    isTruncated.value = Boolean(element && element.scrollWidth > element.clientWidth);
  }

  onUpdated(updateTruncation);

  onMounted(() => {
    observer = new ResizeObserver(updateTruncation);
    if (container.value) {
      observer.observe(container.value);
    }
    updateTruncation();
  });

  onBeforeUnmount(() => observer?.disconnect());
</script>
