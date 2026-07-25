<template>
  <AppDialog
    v-model:open="isOpen"
    :title="
      isEditing
        ? t('application.versionDetail.dialog.editComponent')
        : t('application.versionDetail.dialog.addComponent')
    "
    width-class="w-[min(480px,calc(100vw-32px))]"
  >
    <div class="space-y-4">
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('application.detail.fields.component') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.name"
          type="text"
          class="app-input"
          :placeholder="t('application.detail.placeholders.componentName')"
        />
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('application.detail.fields.image') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.image"
          type="text"
          class="app-input"
          :placeholder="t('application.detail.placeholders.componentImage')"
        />
      </div>
      <p v-if="formError" class="app-field-error text-xs">{{ formError }}</p>
    </div>
    <template #footer>
      <button class="app-button" @click="isOpen = false">{{ t('common.cancel') }}</button>
      <button :disabled="saving" class="app-button-primary" @click="save">
        {{ t('common.save') }}
      </button>
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppDialog from '@/components/AppDialog.vue';

  interface ComponentBase {
    name: string;
    image: string;
  }

  const props = withDefaults(
    defineProps<{
      open: boolean;
      component?: ComponentBase;
      componentNames: string[];
      saving?: boolean;
    }>(),
    { saving: false }
  );

  const emit = defineEmits<{
    'update:open': [open: boolean];
    save: [component: ComponentBase];
  }>();

  const { t } = useI18n();
  const form = reactive<ComponentBase>({ name: '', image: '' });
  const formError = ref('');
  const isEditing = computed(() => Boolean(props.component));
  const isOpen = computed({
    get: () => props.open,
    set: (open) => emit('update:open', open),
  });

  watch(
    () => props.open,
    (open) => {
      if (!open) {
        return;
      }
      form.name = props.component?.name ?? '';
      form.image = props.component?.image ?? '';
      formError.value = '';
    }
  );

  function save() {
    const name = form.name.trim();
    const image = form.image.trim();
    if (!name || !image) {
      formError.value = t('application.validation.componentNameImageRequired');
      return;
    }
    if (!/^[a-z][a-z0-9-]*$/.test(name)) {
      formError.value = t('application.validation.componentNameInvalid');
      return;
    }
    if (props.componentNames.some((componentName) => componentName === name && name !== props.component?.name)) {
      formError.value = t('application.versionDetail.validation.duplicateComponent');
      return;
    }
    formError.value = '';
    emit('save', { name, image });
  }
</script>
