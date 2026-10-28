<template>
  <DialogRoot v-model:open="openModel">
    <DialogPortal>
      <DialogOverlay class="app-dialog-overlay" />
      <DialogContent
        v-bind="contentA11yAttrs"
        :class="['app-dialog-content', widthClass, contentClass]"
        @open-auto-focus.prevent
        @close-auto-focus.prevent
      >
        <div class="shrink-0 px-6 pb-2 pt-6">
          <div class="flex min-w-0 items-center justify-between gap-3">
            <DialogTitle class="min-w-0 break-words text-base font-normal text-foreground">
              {{ title }}
            </DialogTitle>
            <slot name="title-actions" />
          </div>
          <DialogDescription v-if="description" class="mt-1 text-sm text-muted-foreground">
            {{ description }}
          </DialogDescription>
          <slot name="description" />
        </div>
        <div :class="bodyClass">
          <slot />
        </div>
        <div v-if="$slots.footer" class="flex shrink-0 flex-wrap justify-end gap-2 px-6 pb-6 pt-4">
          <slot name="footer" />
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<script setup lang="ts">
  import { computed, useSlots } from 'vue';
  import {
    DialogContent,
    DialogDescription,
    DialogOverlay,
    DialogPortal,
    DialogRoot,
    DialogTitle,
  } from 'reka-ui';

  const props = withDefaults(
    defineProps<{
      open: boolean;
      title: string;
      description?: string;
      widthClass?: string;
      bodyClass?: string;
      contentClass?: string;
    }>(),
    {
      description: '',
      widthClass: 'w-[min(520px,calc(100vw-32px))]',
      bodyClass: 'min-h-0 flex-1 space-y-4 overflow-y-auto px-6 py-4',
      contentClass: '',
    }
  );

  const emit = defineEmits<{
    'update:open': [open: boolean];
  }>();

  const slots = useSlots();
  const openModel = computed({
    get: () => props.open,
    set: (value) => emit('update:open', value),
  });
  const contentA11yAttrs = computed(() =>
    props.description || slots.description ? {} : { 'aria-describedby': undefined }
  );
</script>
