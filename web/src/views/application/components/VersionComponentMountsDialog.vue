<template>
  <AppDialog
    v-model:open="isOpen"
    :title="t('application.versionDetail.dialog.componentMounts')"
    width-class="w-[min(720px,calc(100vw-32px))]"
  >
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <label class="app-field-label">{{ t('application.detail.fields.mounts') }}</label>
        <button type="button" class="app-link text-sm" @click="addMount">
          {{ t('application.detail.actions.addMount') }}
        </button>
      </div>
      <div
        v-for="(mountRow, mountIndex) in mounts"
        :key="'mnt-' + mountIndex"
        class="space-y-2 rounded border border-dashed border-border p-2"
      >
        <div class="grid grid-cols-1 gap-2 sm:grid-cols-[0.9fr_1fr_1fr_auto_auto]">
          <RawValueSelect
            v-model="mountRow.source_type"
            :values="mountSourceTypeValues"
            :placeholder="t('application.detail.placeholders.mountSourceType')"
          />
          <input
            v-model="mountRow.source"
            type="text"
            class="app-input text-xs"
            :placeholder="t('application.detail.placeholders.mountSource')"
          />
          <input
            v-model="mountRow.target"
            type="text"
            class="app-input text-xs"
            :placeholder="t('application.detail.placeholders.mountTarget')"
          />
          <label class="flex items-center gap-1 text-xs text-muted-foreground">
            <input v-model="mountRow.read_only" type="checkbox" class="app-checkbox" />
            ro
          </label>
          <button type="button" class="app-link-danger" @click="mounts.splice(mountIndex, 1)">
            {{ t('common.delete') }}
          </button>
        </div>
        <div v-if="isFileMountRow(mountRow)" class="space-y-2 border-t border-border pt-2">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-xs text-muted-foreground">
              {{ t('application.detail.fields.mountContent') }}
            </span>
            <RawValueSelect
              v-model="mountRow.content_mode"
              :values="mountContentModeValues"
              :placeholder="t('application.detail.placeholders.mountContentMode')"
              class="w-36"
            />
          </div>
          <textarea
            v-model="mountRow.content"
            rows="5"
            class="app-input min-h-[6rem] w-full text-xs"
            :placeholder="t('application.detail.placeholders.mountContent')"
          />
        </div>
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
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import {
    isFileMountRow,
    parseMountsJson,
    serializeMountRows,
    type MountFormRow,
  } from '@/utils/versionComponentForm';

  const props = withDefaults(
    defineProps<{
      open: boolean;
      mountsJson?: string;
      saving?: boolean;
    }>(),
    { mountsJson: undefined, saving: false }
  );

  const emit = defineEmits<{
    'update:open': [open: boolean];
    save: [mountsJson: string | undefined];
  }>();

  const { t } = useI18n();
  const mounts = ref<MountFormRow[]>([]);
  const mountSourceTypeValues = ['logical', 'volume', 'special'];
  const mountContentModeValues = ['seed', 'sync'];
  const isOpen = computed({
    get: () => props.open,
    set: (open) => emit('update:open', open),
  });

  watch(
    () => props.open,
    (open) => {
      if (open) {
        mounts.value = parseMountsJson(props.mountsJson);
      }
    }
  );

  function addMount() {
    mounts.value.push({
      source_type: 'logical',
      source: '',
      target: '',
      read_only: false,
      content: '',
      content_mode: 'seed',
    });
  }

  function save() {
    emit('save', serializeMountRows(mounts.value));
  }
</script>
