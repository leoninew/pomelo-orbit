<template>
  <AppDialog
    v-model:open="isOpen"
    :title="t('application.versionDetail.dialog.componentPorts')"
    width-class="w-[min(560px,calc(100vw-32px))]"
  >
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <label class="app-field-label">{{ t('application.detail.fields.ports') }}</label>
        <button type="button" class="app-link text-sm" @click="addPort">
          {{ t('application.detail.actions.addPort') }}
        </button>
      </div>
      <div
        v-for="(portRow, portIndex) in ports"
        :key="'port-' + portIndex"
        class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1fr_auto]"
      >
        <input
          v-model.number="portRow.host_port"
          type="number"
          min="1"
          max="65535"
          class="app-input text-xs"
          :placeholder="t('application.detail.placeholders.hostPort')"
        />
        <input
          v-model.number="portRow.container_port"
          type="number"
          min="1"
          max="65535"
          class="app-input text-xs"
          :placeholder="t('application.detail.placeholders.containerPort')"
        />
        <button type="button" class="app-link-danger" @click="ports.splice(portIndex, 1)">
          {{ t('common.delete') }}
        </button>
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
  import { computed, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppDialog from '@/components/AppDialog.vue';
  import {
    isValidPort,
    parsePortsJson,
    serializePortsRows,
    type PortFormRow,
  } from '@/utils/versionComponentForm';

  const props = withDefaults(
    defineProps<{
      open: boolean;
      portsJson?: string;
      saving?: boolean;
    }>(),
    { portsJson: undefined, saving: false }
  );

  const emit = defineEmits<{
    'update:open': [open: boolean];
    save: [portsJson: string | undefined];
  }>();

  const { t } = useI18n();
  const ports = ref<PortFormRow[]>([]);
  const formError = ref('');
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
      ports.value = parsePortsJson(props.portsJson);
      formError.value = '';
    }
  );

  function addPort() {
    ports.value.push({ host_port: 80, container_port: 80 });
  }

  function save() {
    if (
      ports.value.some(
        (port) => !isValidPort(Number(port.host_port)) || !isValidPort(Number(port.container_port))
      )
    ) {
      formError.value = t('application.validation.portRange');
      return;
    }
    formError.value = '';
    emit('save', serializePortsRows(ports.value));
  }
</script>
