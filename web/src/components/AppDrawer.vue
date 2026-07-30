<template>
  <DialogRoot v-model:open="openModel">
    <DialogPortal>
      <DialogOverlay class="app-dialog-overlay" />
      <DialogContent
        v-bind="contentA11yAttrs"
        :class="['app-drawer-content', widthClass, contentClass]"
        @open-auto-focus.prevent
        @close-auto-focus.prevent
      >
        <div class="flex items-start justify-between gap-4 border-b border-border px-6 py-4">
          <div class="min-w-0">
            <DialogTitle class="text-base font-semibold text-foreground">
              {{ title }}
            </DialogTitle>
            <DialogDescription v-if="description" class="mt-1 text-sm text-muted-foreground">
              {{ description }}
            </DialogDescription>
            <slot name="description" />
          </div>
          <DialogClose as-child>
            <button
              type="button"
              class="app-icon-button shrink-0"
              :aria-label="t('common.close')"
              :title="t('common.close')"
            >
              <X class="size-4" />
            </button>
          </DialogClose>
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
  import { X } from 'lucide-vue-next';
  import { computed, useSlots } from 'vue';
  import { useI18n } from 'vue-i18n';
  import {
    DialogClose,
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
      widthClass: 'w-[min(720px,100vw)]',
      bodyClass: 'min-h-0 flex-1 overflow-y-auto px-6 py-4',
      contentClass: '',
    }
  );

  const emit = defineEmits<{
    'update:open': [open: boolean];
  }>();

  const slots = useSlots();
  const { t } = useI18n({ useScope: 'global' });
  const openModel = computed({
    get: () => props.open,
    set: (value) => emit('update:open', value),
  });
  const contentA11yAttrs = computed(() =>
    props.description || slots.description ? {} : { 'aria-describedby': undefined }
  );
</script>
