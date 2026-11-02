<template>
  <div class="space-y-1.5">
    <label :for="id" class="app-field-label block">
      {{ t('controlledFile.writePolicy') }}
    </label>
    <SelectControl
      :id="id"
      :model-value="String(modelValue)"
      :options="[
        { value: 'false', label: t('controlledFile.overwrite') },
        { value: 'true', label: t('controlledFile.preserve') },
      ]"
      reset-value="false"
      @update:model-value="emit('update:modelValue', $event === 'true')"
    />
    <p v-if="modelValue" class="text-sm text-muted-foreground">
      {{ t('controlledFile.preserveHelp') }}
    </p>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n';
  import SelectControl from '@/components/ui/SelectControl.vue';

  defineProps<{
    id: string;
    modelValue: boolean;
  }>();
  const emit = defineEmits<{
    'update:modelValue': [value: boolean];
  }>();
  const { t } = useI18n();
</script>
