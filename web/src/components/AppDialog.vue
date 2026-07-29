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
        <div class="border-b border-border px-6 py-4">
          <DialogTitle class="text-base font-semibold text-foreground">
            {{ title }}
          </DialogTitle>
          <DialogDescription v-if="description" class="mt-1 text-sm text-muted-foreground">
            {{ description }}
          </DialogDescription>
          <slot name="description" />
        </div>
        <div :class="bodyClass">
          <slot />
        </div>
        <div v-if="$slots.footer" class="flex justify-end gap-2 border-t border-border px-6 py-4">
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
      bodyClass: 'space-y-4 px-6 py-4',
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
