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
      <SelectControl
        :model-value="form.kind || 'standard'"
        :options="kindOptions"
        :placeholder="t('application.kindPlaceholder')"
        @update:model-value="updateField('kind', String($event))"
      />
      <p class="app-field-hint">{{ t('application.kindHint') }}</p>
    </div>

    <div class="space-y-1.5">
      <label class="app-field-label block">{{ t('application.imagePullPolicy') }}</label>
      <SelectControl
        :model-value="form.image_pull_policy"
        :options="imagePullPolicyOptions"
        :placeholder="t('application.imagePullPolicyPlaceholder')"
        @update:model-value="updateField('image_pull_policy', String($event))"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import SelectControl from '@/components/SelectControl.vue';
  import type { ApplicationCreateReq } from '@/gen/proto/orbit/v1/application';

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

  const kindOptions = computed(() => [
    { value: 'standard', label: t('application.kindOptions.standard') },
    { value: 'gateway', label: t('application.kindOptions.gateway') },
  ]);

  const imagePullPolicyOptions = computed(() => [
    { value: 'missing', label: t('application.imagePullPolicyOptions.missing') },
    { value: 'always', label: t('application.imagePullPolicyOptions.always') },
    { value: 'never', label: t('application.imagePullPolicyOptions.never') },
  ]);
</script>
