<template>
  <AppDialog :open="open" :title="t('route.configureCertificate')" @update:open="handleOpenChange">
    <div class="space-y-2">
      <label for="route-certificate-mode" class="app-field-label block">
        {{ t('route.syncCertificateMode') }}
      </label>
      <SelectControl
        id="route-certificate-mode"
        :model-value="selectedMode"
        :options="certificateOptions"
        :disabled="saving"
        :invalid="Boolean(error)"
        @update:model-value="updateMode"
      />
      <div v-if="selectedMode === 'manual'" class="flex flex-wrap items-center gap-3">
        <button
          type="button"
          class="app-button h-9 px-3"
          :disabled="saving"
          @click="fileInput?.click()"
        >
          <Upload class="size-4" />
          {{ t('route.selectPemFile') }}
        </button>
        <span class="text-sm text-muted-foreground">
          {{
            file?.name || t(currentMode === 'manual' ? 'route.savedPemHint' : 'route.pemFileHint')
          }}
        </span>
        <input ref="fileInput" type="file" accept=".pem" class="hidden" @change="selectFile" />
      </div>
      <p v-if="error" class="app-field-error text-xs" role="alert">{{ error }}</p>
    </div>
    <template #footer>
      <AppDialogActions
        :busy="saving"
        :confirm-disabled="selectedMode === currentMode && !file"
        :confirm-label="t('route.saveConfiguration')"
        @cancel="close"
        @confirm="save"
      />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';
  import { Upload } from '@lucide/vue';
  import { useI18n } from 'vue-i18n';
  import { routeApi } from '@/api/route/route';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useProjectStore } from '@/stores/project';
  import type { RouteResp } from '@/gen/proto/orbit/v1/route/route';

  type CertificateMode = 'http' | 'letsencrypt-http' | 'letsencrypt-dns' | 'mkcert' | 'manual';
  const props = defineProps<{ open: boolean; route: RouteResp }>();
  const emit = defineEmits<{ 'update:open': [open: boolean]; saved: [route: RouteResp] }>();
  const { t } = useI18n();
  const projectStore = useProjectStore();
  const selectedMode = ref<CertificateMode>('http');
  const fileInput = ref<HTMLInputElement>();
  const file = ref<File>();
  const error = ref('');
  const saving = ref(false);
  const currentMode = computed<CertificateMode>(() => {
    if (!props.route.https_enabled) {
      return 'http';
    }
    if (props.route.cert_type === 'letsencrypt') {
      return props.route.acme_challenge === 'dns' ? 'letsencrypt-dns' : 'letsencrypt-http';
    }
    return props.route.cert_type === 'mkcert' ? 'mkcert' : 'manual';
  });
  const canUseLetsEncrypt = computed(() => {
    const domain = props.route.domain;
    return (
      domain !== 'localhost' &&
      !domain.endsWith('.localhost') &&
      !domain.endsWith('.lvh.me') &&
      !/^\d+\.\d+\.\d+\.\d+$/.test(domain)
    );
  });
  const certificateOptions = computed(() => [
    { value: 'http', label: 'HTTP' },
    {
      value: 'letsencrypt-http',
      label: "HTTPS · Let's Encrypt · HTTP-01",
      disabled: !canUseLetsEncrypt.value || !props.route.http01_available,
    },
    {
      value: 'letsencrypt-dns',
      label: "HTTPS · Let's Encrypt · DNS-01",
      disabled: !canUseLetsEncrypt.value || !props.route.dns01_available,
    },
    { value: 'mkcert', label: 'HTTPS · mkcert' },
    { value: 'manual', label: 'HTTPS · PEM' },
  ]);

  watch(
    () => props.open,
    (open) => {
      if (open) {
        selectedMode.value = currentMode.value;
        file.value = undefined;
        error.value = '';
      }
    },
    { immediate: true }
  );

  function close() {
    emit('update:open', false);
  }
  function handleOpenChange(open: boolean) {
    if (!open) {
      close();
    }
  }
  function updateMode(value: string | number) {
    if (!certificateOptions.value.some((option) => option.value === value && !option.disabled)) {
      return;
    }
    selectedMode.value = value as CertificateMode;
    file.value = undefined;
    error.value = '';
  }
  function selectFile(event: Event) {
    const input = event.target as HTMLInputElement;
    file.value = input.files?.[0];
    error.value = '';
  }
  async function save() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      error.value = t('route.toast.selectProjectRequired');
      return;
    }
    if (selectedMode.value === 'manual' && !file.value) {
      error.value = t('route.pemRequired');
      return;
    }
    saving.value = true;
    error.value = '';
    try {
      const routeId = props.route.id;
      let updated: RouteResp;
      switch (selectedMode.value) {
        case 'http':
          updated = await routeApi.disableHTTPS(projectId, routeId);
          break;
        case 'letsencrypt-http':
          updated = await routeApi.enableLetsEncrypt(projectId, routeId, 'http');
          break;
        case 'letsencrypt-dns':
          updated = await routeApi.enableLetsEncrypt(projectId, routeId, 'dns');
          break;
        case 'mkcert':
          updated = await routeApi.enableMkcert(projectId, routeId);
          break;
        case 'manual': {
          const selectedFile = file.value;
          if (!selectedFile) {
            return;
          }
          updated = await routeApi.uploadCert(projectId, routeId, selectedFile);
          break;
        }
      }
      emit('saved', updated);
      close();
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : t('route.toast.updateFailed');
    } finally {
      saving.value = false;
    }
  }
</script>
