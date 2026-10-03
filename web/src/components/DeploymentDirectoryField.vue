<template>
  <div class="app-form-field">
    <label :for="id" class="app-field-label">
      {{ t('service.deploy.directory') }}
      <span class="app-required-marker" aria-hidden="true">*</span>
    </label>
    <input
      :id="id"
      :value="modelValue"
      type="text"
      class="app-input w-full"
      :class="{ 'app-input-error': error }"
      :aria-invalid="Boolean(error)"
      :aria-describedby="error ? id + '-error' : undefined"
      @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
    <p v-if="error" :id="id + '-error'" class="app-field-error" role="alert">{{ error }}</p>
    <p v-if="directoryChanged" class="app-field-warning" role="status">
      {{
        t(gateway ? 'service.deploy.gatewayDirectoryWarning' : 'service.deploy.directoryWarning')
      }}
    </p>
  </div>
</template>

<script setup lang="ts">
  import { computed, useId } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { normalizeDeploymentDirectory } from '@/composables/useDeploymentDirectory';
  const props = defineProps<{
    modelValue: string;
    platform: string;
    runtimeDirectory: string;
    error?: string;
    gateway?: boolean;
  }>();
  const emit = defineEmits<{ 'update:modelValue': [value: string] }>();
  const { t } = useI18n();
  const id = useId();
  const directoryChanged = computed(
    () =>
      Boolean(props.runtimeDirectory) &&
      normalizeDeploymentDirectory(props.modelValue, props.platform) !==
        normalizeDeploymentDirectory(props.runtimeDirectory, props.platform)
  );
</script>
