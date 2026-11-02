<template>
  <AppDialog
    :open="open"
    :title="t('service.componentDetail.mountEditor.title')"
    :width-class="
      draft?.source_type === 'controlled_file'
        ? 'w-[min(900px,calc(100vw-32px))]'
        : 'w-[min(640px,calc(100vw-32px))]'
    "
    @update:open="emit('update:open', $event)"
  >
    <div v-if="draft" class="space-y-4">
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.sourceType') }}
          </label>
          <AppInput
            :model-value="draft.source_type"
            readonly
            :aria-label="t('application.componentDetail.fields.sourceType')"
          />
        </div>
        <div>
          <label for="service-mount-source" class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.source') }}
            <span class="text-destructive">*</span>
          </label>
          <AppInput
            id="service-mount-source"
            :model-value="draft.values.source"
            :class="errors.source ? 'app-input-error' : ''"
            :aria-invalid="errors.source ? 'true' : undefined"
            :aria-describedby="errors.source ? 'service-mount-source-error' : undefined"
            @update:model-value="updateField('source', $event)"
          />
          <p
            v-if="errors.source"
            id="service-mount-source-error"
            class="app-field-error"
            role="alert"
          >
            {{ t(errors.source) }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.target') }}
          </label>
          <AppInput
            :model-value="draft.target"
            readonly
            :aria-label="t('application.componentDetail.fields.target')"
          />
        </div>
        <label
          v-if="draft.source_type !== 'controlled_file'"
          class="flex items-center gap-2 self-end pb-2"
        >
          <input
            v-model="draft.values.read_only"
            type="checkbox"
            class="app-checkbox"
            @change="updateField('read_only', draft.values.read_only)"
          />
          <span class="text-sm text-foreground">
            {{ t('application.componentDetail.fields.readOnly') }}
          </span>
        </label>
        <label
          v-if="['directory', 'file'].includes(draft.source_type)"
          class="flex items-center gap-2 self-end pb-2"
        >
          <input
            v-model="draft.values.source_is_host_path"
            type="checkbox"
            class="app-checkbox"
            @change="updateField('source_is_host_path', draft.values.source_is_host_path)"
          />
          <span class="text-sm text-foreground">
            {{ t('application.componentDetail.fields.sourceIsHostPath') }}
          </span>
        </label>
        <template v-if="draft.source_type === 'controlled_file'">
          <div>
            <label for="service-mount-mode" class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.mode') }}
              <span class="text-destructive">*</span>
            </label>
            <AppInput
              id="service-mount-mode"
              :model-value="draft.values.mode"
              :class="errors.mode ? 'app-input-error' : ''"
              :aria-invalid="errors.mode ? 'true' : undefined"
              :aria-describedby="errors.mode ? 'service-mount-mode-error' : undefined"
              @update:model-value="updateField('mode', $event)"
            />
            <p
              v-if="errors.mode"
              id="service-mount-mode-error"
              class="app-field-error"
              role="alert"
            >
              {{ t(errors.mode) }}
            </p>
          </div>
          <ControlledFileWritePolicy
            id="service-mount-policy"
            :model-value="draft.values.ignore_if_exists"
            @update:model-value="updateField('ignore_if_exists', $event)"
          />
        </template>
      </div>
      <template v-if="draft.source_type === 'controlled_file'">
        <span id="service-mount-content-label" class="app-field-label block">
          {{ t('service.componentDetail.mountEditor.content') }}
        </span>
        <div
          role="group"
          aria-labelledby="service-mount-content-label"
          :aria-invalid="errors.content ? 'true' : undefined"
          :aria-describedby="errors.content ? 'service-mount-content-error' : undefined"
        >
          <MonacoEditor
            :model-value="draft.values.content"
            language="plaintext"
            height="280px"
            :has-error="Boolean(errors.content)"
            @update:model-value="updateField('content', $event)"
          />
        </div>
        <p
          v-if="errors.content"
          id="service-mount-content-error"
          class="app-field-error"
          role="alert"
        >
          {{ t(errors.content) }}
        </p>
      </template>
    </div>
    <template #footer>
      <AppDialogActions
        :busy="busy"
        :confirm-label="t('common.confirm')"
        @cancel="emit('update:open', false)"
        @confirm="confirm"
      />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import ControlledFileWritePolicy from '@/components/application/ControlledFileWritePolicy.vue';
  import AppDialog from '@/components/ui/AppDialog.vue';
  import AppDialogActions from '@/components/ui/AppDialogActions.vue';
  import AppInput from '@/components/ui/AppInput.vue';
  import MonacoEditor from '@/components/ui/MonacoEditor.vue';
  import {
    cloneComponentMount,
    setMountFieldValue,
    validateComponentMount,
    type MountErrors,
    type MountField,
    type ServiceComponentMountRow,
  } from './serviceComponentMount';

  const props = defineProps<{
    open: boolean;
    mount: ServiceComponentMountRow | undefined;
    busy: boolean;
  }>();
  const emit = defineEmits<{
    'update:open': [value: boolean];
    confirm: [value: ServiceComponentMountRow];
  }>();
  const { t } = useI18n();
  const draft = ref<ServiceComponentMountRow>();
  const errors = ref<MountErrors>({});
  watch(
    () => [props.open, props.mount] as const,
    ([open, mount]) => {
      draft.value = open && mount ? cloneComponentMount(mount) : undefined;
      errors.value = {};
    },
    { immediate: true }
  );
  function updateField<Field extends MountField>(
    field: Field,
    value: ServiceComponentMountRow['values'][Field]
  ) {
    if (!draft.value) {
      return;
    }
    setMountFieldValue(draft.value, field, value);
    if (field === 'mode') {
      delete errors.value.mode;
    }
    if (field === 'content') {
      delete errors.value.content;
    }
    if (field === 'source' || field === 'source_is_host_path') {
      delete errors.value.source;
    }
  }
  function confirm() {
    if (!draft.value || props.busy) {
      return;
    }
    if (draft.value.overridden.source) {
      draft.value.values.source = draft.value.values.source.trim();
    }
    errors.value = validateComponentMount(draft.value);
    if (Object.keys(errors.value).length) {
      return;
    }
    draft.value.deleted = false;
    emit('confirm', cloneComponentMount(draft.value));
  }
</script>
