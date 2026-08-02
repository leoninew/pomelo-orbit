<template>
  <div class="space-y-4">
    <div class="space-y-1.5">
      <label class="app-field-label block">
        {{ t('application.name') }}
        <span class="text-destructive">*</span>
      </label>
      <input
        :value="form.name"
        type="text"
        :placeholder="t('application.namePlaceholder')"
        class="app-input"
        :class="errors.name ? 'app-input-error' : ''"
        :aria-invalid="errors.name ? 'true' : undefined"
        @input="updateField('name', ($event.target as HTMLInputElement).value)"
      />
      <p v-if="errors.name" class="app-field-error mt-1 text-xs">{{ errors.name }}</p>
    </div>

    <div class="space-y-1.5">
      <label class="app-field-label block">
        {{ t('application.code') }}
        <span class="text-destructive">*</span>
      </label>
      <input
        :value="form.code"
        type="text"
        :placeholder="t('application.codePlaceholder')"
        class="app-input"
        :class="errors.code ? 'app-input-error' : ''"
        :aria-invalid="errors.code ? 'true' : undefined"
        @input="updateField('code', ($event.target as HTMLInputElement).value)"
      />
      <p v-if="errors.code" class="app-field-error mt-1 text-xs">{{ errors.code }}</p>
      <p class="app-field-hint">{{ t('application.codeHint') }}</p>
    </div>

    <div class="space-y-1.5">
      <label class="app-field-label block">{{ t('application.kind') }}</label>
      <input value="standard" type="text" class="app-input" disabled />
    </div>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n';
  import type { ApplicationCreateReq } from '@/gen/proto/orbit/v1/application/application';

  const props = defineProps<{
    form: ApplicationCreateReq;
    errors: { name: string; code: string };
  }>();

  const emit = defineEmits<{
    'update:form': [value: ApplicationCreateReq];
    'clear-error': [field: 'name' | 'code'];
  }>();

  const { t } = useI18n();

  function updateField<K extends keyof ApplicationCreateReq>(
    field: K,
    value: ApplicationCreateReq[K]
  ) {
    if (field === 'name' || field === 'code') {
      emit('clear-error', field);
    }
    emit('update:form', { ...props.form, [field]: value });
  }
</script>
