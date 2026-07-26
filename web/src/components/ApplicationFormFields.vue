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
        @input="updateField('code', ($event.target as HTMLInputElement).value)"
      />
      <p v-if="errors.code" class="app-field-error mt-1 text-xs">{{ errors.code }}</p>
      <p class="app-field-hint">{{ t('application.codeHint') }}</p>
    </div>

    <div class="space-y-1.5">
      <label class="app-field-label block">{{ t('application.kind') }}</label>
      <RawValueSelect
        :model-value="form.kind"
        :values="kindValues"
        :placeholder="t('application.kindPlaceholder')"
        @update:model-value="updateField('kind', String($event))"
      />
      <p class="app-field-hint">{{ t('application.kindHint') }}</p>
    </div>

    <div class="space-y-1.5">
      <label class="app-field-label block">{{ t('application.imagePullPolicy') }}</label>
      <RawValueSelect
        :model-value="form.image_pull_policy"
        :values="imagePullPolicyValues"
        :placeholder="t('application.imagePullPolicyPlaceholder')"
        @update:model-value="updateField('image_pull_policy', String($event))"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import type { ApplicationCreateReq } from '@/gen/proto/orbit/v1/application/application';

  const props = defineProps<{
    form: ApplicationCreateReq;
    errors: { name: string; code: string };
  }>();

  const emit = defineEmits<{
    'update:form': [value: ApplicationCreateReq];
  }>();

  const { t } = useI18n();

  function updateField<K extends keyof ApplicationCreateReq>(
    field: K,
    value: ApplicationCreateReq[K]
  ) {
    emit('update:form', { ...props.form, [field]: value });
  }

  const kindValues = ['standard', 'gateway'];

  const imagePullPolicyValues = ['missing', 'always', 'never'];
</script>
