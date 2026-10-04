<template>
  <span ref="container" class="block min-w-0">
    <AppTooltip :content="hasHiddenText ? text : undefined">
      <Primitive
        as="span"
        :as-child="asChild"
        class="block truncate"
        :class="{
          'cursor-help underline decoration-dashed underline-offset-2 focus-visible:outline focus-visible:outline-ring':
            hasHiddenText,
        }"
        :tabindex="hasHiddenText && !asChild ? 0 : undefined"
      >
        <slot>{{ displayText ?? text }}</slot>
      </Primitive>
    </AppTooltip>
  </span>
</template>

<script setup lang="ts">
  import { Primitive } from 'reka-ui';
  import { computed, onBeforeUnmount, onMounted, onUpdated, ref } from 'vue';
  import AppTooltip from '@/components/AppTooltip.vue';

  const props = defineProps<{ text?: string; displayText?: string; asChild?: boolean }>();
  const container = ref<HTMLSpanElement>();
  const isTruncated = ref(false);
  const hasHiddenText = computed(
    () => isTruncated.value || (props.displayText !== undefined && props.displayText !== props.text)
  );
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
