<template>
  <AppDialog
    v-model:open="isOpen"
    :title="t('application.versionDetail.dialog.componentEnv')"
    width-class="w-[min(560px,calc(100vw-32px))]"
  >
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <label class="app-field-label">{{ t('application.detail.fields.componentEnv') }}</label>
        <button type="button" class="app-link text-sm" @click="addEnv">
          {{ t('application.detail.actions.addEnv') }}
        </button>
      </div>
      <div
        v-for="(envRow, envIndex) in env"
        :key="'cenv-' + envIndex"
        class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1.2fr_auto]"
      >
        <input v-model="envRow.key" type="text" class="app-input font-mono text-xs" placeholder="KEY" />
        <input
          v-model="envRow.value"
          type="text"
          class="app-input font-mono text-xs"
          :placeholder="t('application.detail.placeholders.envValue')"
        />
        <button type="button" class="app-link-danger" @click="env.splice(envIndex, 1)">
          {{ t('common.delete') }}
        </button>
      </div>
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
  import { computed, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppDialog from '@/components/AppDialog.vue';
  import {
    parseEnvJson,
    serializeEnvRows,
    type EnvFormRow,
  } from '@/utils/versionComponentForm';

  const props = withDefaults(
    defineProps<{
      open: boolean;
      envJson?: string;
      saving?: boolean;
    }>(),
    { envJson: undefined, saving: false }
  );

  const emit = defineEmits<{
    'update:open': [open: boolean];
    save: [envJson: string | undefined];
  }>();

  const { t } = useI18n();
  const env = ref<EnvFormRow[]>([]);
  const isOpen = computed({
    get: () => props.open,
    set: (open) => emit('update:open', open),
  });

  watch(
    () => props.open,
    (open) => {
      if (open) {
        env.value = parseEnvJson(props.envJson);
      }
    }
  );

  function addEnv() {
    env.value.push({ key: '', value: '' });
  }

  function save() {
    emit('save', serializeEnvRows(env.value));
  }
</script>
