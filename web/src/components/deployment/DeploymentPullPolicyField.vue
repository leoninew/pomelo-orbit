<template>
  <div class="space-y-1.5">
    <label :for="inputId" class="app-field-label block">
      {{ t('deployment.fields.pullPolicy') }}
    </label>
    <SelectControl :id="inputId" v-model="modelValue" :options="options" reset-value="missing" />
  </div>
</template>

<script setup lang="ts">
  import { useId } from 'vue';
  import { useI18n } from 'vue-i18n';
  import SelectControl from '@/components/ui/SelectControl.vue';

  const modelValue = defineModel<string>({ required: true });
  const { t } = useI18n();
  const inputId = useId();
  const options = ['missing', 'always', 'never'].map((value) => ({ value, label: value }));
</script>
