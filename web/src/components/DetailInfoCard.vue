<template>
  <section class="app-surface app-detail-card">
    <div class="app-section-header app-detail-section-header shrink-0">
      <slot name="header">
        <h2 v-if="title" class="app-detail-section-title">{{ title }}</h2>
        <div
          v-if="editable || hasActionsSlot"
          class="flex flex-wrap items-center gap-2"
          :class="actionsClass"
        >
          <button
            v-if="editable"
            class="app-button-primary h-9 px-3"
            :disabled="disabled"
            @click="emit('edit')"
          >
            <Pencil class="size-4" />
            {{ t('common.edit') }}
          </button>
          <slot name="actions" />
        </div>
      </slot>
    </div>
    <AppLoadingState v-if="loading" size="section" />
    <slot v-else />
  </section>
</template>

<script setup lang="ts">
  import { Pencil } from '@lucide/vue';
  import { computed, useSlots } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppLoadingState from '@/components/AppLoadingState.vue';

  withDefaults(
    defineProps<{
      title?: string;
      editable?: boolean;
      disabled?: boolean;
      loading?: boolean;
      actionsClass?: string;
    }>(),
    {
      editable: false,
      disabled: false,
      loading: false,
    }
  );

  const emit = defineEmits<{ edit: [] }>();
  const { t } = useI18n();
  const slots = useSlots();
  const hasActionsSlot = computed(() => Boolean(slots.actions));
</script>
