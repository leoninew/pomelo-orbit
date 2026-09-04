<template>
  <DetailInfoCard
    :title="t('project.environment.title')"
    :loading="loading"
    :editable="Boolean(environment)"
    :disabled="operating"
    @edit="openEditDialog"
  >
    <template #actions>
      <button class="app-button h-9 px-3" :disabled="operating || !environment" @click="probe">
        <RefreshCw class="size-4" />
        {{ t('project.environment.probe') }}
      </button>
    </template>

    <p v-if="loadError" class="px-5 py-4 text-sm text-destructive">{{ loadError }}</p>
    <dl v-else-if="environment" class="app-detail-info-grid">
      <div class="flex gap-2">
        <dt>{{ t('project.environment.state') }}</dt>
        <dd>
          <AppBadge variant="status" :tone="environment.state === 'active' ? 'success' : 'default'">
            {{ t(`project.environment.states.${environment.state}`) }}
          </AppBadge>
        </dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('project.environment.platform') }}</dt>
        <dd class="text-foreground">
          {{ t(`project.environment.platforms.${environment.platform}`) }}
        </dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('project.environment.host') }}</dt>
        <dd class="text-foreground">{{ environment.host }}</dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('project.environment.port') }}</dt>
        <dd class="text-foreground">{{ environment.port }}</dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('project.environment.username') }}</dt>
        <dd class="text-foreground">{{ environment.username }}</dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('project.environment.targetRevision') }}</dt>
        <dd class="text-foreground">{{ environment.target_revision }}</dd>
      </div>
      <div class="flex gap-2 sm:col-span-2">
        <dt>{{ t('project.environment.workspaceRoot') }}</dt>
        <dd class="min-w-0 break-all text-foreground">{{ environment.workspace_root }}</dd>
      </div>
      <div class="flex gap-2 sm:col-span-2">
        <dt>{{ t('project.environment.hostKeyFingerprint') }}</dt>
        <dd class="min-w-0 break-all font-mono text-xs text-foreground">
          {{ environment.host_key_fingerprint }}
        </dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('project.environment.gatewayApplication') }}</dt>
        <dd class="min-w-0 truncate text-foreground">
          {{ environment.gateway_application_id || t('common.notSet') }}
        </dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('project.environment.lastProbeAt') }}</dt>
        <dd class="text-muted-foreground">
          {{
            environment.last_probe_at ? formatTime(environment.last_probe_at) : t('common.notSet')
          }}
        </dd>
      </div>
      <div class="flex gap-2">
        <dt>{{ t('project.environment.lastProbeStatus') }}</dt>
        <dd>
          <AppBadge
            v-if="environment.last_probe_status"
            variant="status"
            :tone="environment.last_probe_status === 'succeeded' ? 'success' : 'error'"
          >
            {{ t(`project.environment.probeStates.${environment.last_probe_status}`) }}
          </AppBadge>
          <span v-else class="text-muted-foreground">{{ t('common.notSet') }}</span>
        </dd>
      </div>
      <div class="flex gap-2 sm:col-span-2">
        <dt>{{ t('project.environment.lastProbeDiagnostic') }}</dt>
        <dd class="min-w-0 break-words text-muted-foreground">
          {{ environment.last_probe_diagnostic || t('common.notSet') }}
        </dd>
      </div>
    </dl>
  </DetailInfoCard>

  <AppDialog
    v-model:open="isEditDialogOpen"
    :title="t('project.environment.editTitle')"
    width-class="w-[min(720px,calc(100vw-32px))]"
    body-class="min-h-0 flex-1 space-y-4 overflow-y-auto px-6 py-4"
    content-class="max-h-[calc(100vh-32px)] flex flex-col"
  >
    <form class="grid gap-4 sm:grid-cols-2" novalidate @submit.prevent="save">
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('project.environment.state') }}</label>
        <SelectControl v-model="form.state" :options="stateOptions" :disabled="operating" />
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('project.environment.platform') }}</label>
        <SelectControl v-model="form.platform" :options="platformOptions" :disabled="operating" />
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('project.environment.host') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.host"
          class="app-input"
          :class="errors.host ? 'app-input-error' : ''"
          :disabled="operating"
          @input="errors.host = ''"
        />
        <p v-if="errors.host" class="app-field-error text-xs">{{ errors.host }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('project.environment.port') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model.number="form.port"
          type="number"
          min="1"
          max="65535"
          class="app-input"
          :class="errors.port ? 'app-input-error' : ''"
          :disabled="operating"
          @input="errors.port = ''"
        />
        <p v-if="errors.port" class="app-field-error text-xs">{{ errors.port }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('project.environment.username') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.username"
          class="app-input"
          :class="errors.username ? 'app-input-error' : ''"
          :disabled="operating"
          @input="errors.username = ''"
        />
        <p v-if="errors.username" class="app-field-error text-xs">{{ errors.username }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('project.environment.workspaceRoot') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.workspaceRoot"
          class="app-input"
          :class="errors.workspaceRoot ? 'app-input-error' : ''"
          :placeholder="workspaceRootPlaceholder"
          :disabled="operating"
          @input="errors.workspaceRoot = ''"
        />
        <p v-if="errors.workspaceRoot" class="app-field-error text-xs">
          {{ errors.workspaceRoot }}
        </p>
      </div>
      <div class="space-y-1.5 sm:col-span-2">
        <label class="app-field-label block">
          {{ t('project.environment.hostKeyFingerprint') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.hostKeyFingerprint"
          class="app-input"
          :class="errors.hostKeyFingerprint ? 'app-input-error' : ''"
          :disabled="operating"
          @input="errors.hostKeyFingerprint = ''"
        />
        <p v-if="errors.hostKeyFingerprint" class="app-field-error text-xs">
          {{ errors.hostKeyFingerprint }}
        </p>
      </div>
      <div class="space-y-1.5 sm:col-span-2">
        <label class="app-field-label block">
          {{ t('project.environment.deploymentSSHPrivateKey') }}
        </label>
        <textarea
          v-model="form.privateKey"
          class="app-textarea min-h-36 font-mono text-xs"
          :disabled="operating"
        />
        <p class="app-field-hint">
          {{ t('project.environment.deploymentSSHPrivateKeyHint') }}
        </p>
      </div>
      <div class="space-y-1.5 sm:col-span-2">
        <label class="app-field-label block">
          {{ t('project.environment.deploymentSSHKeyPassphrase') }}
        </label>
        <input
          v-model="form.passphrase"
          type="password"
          autocomplete="new-password"
          class="app-input"
          :disabled="operating || !form.privateKey.trim()"
        />
        <p class="app-field-hint">
          {{ t('project.environment.deploymentSSHKeyPassphraseHint') }}
        </p>
      </div>
      <button type="submit" class="sr-only" tabindex="-1" aria-hidden="true"></button>
    </form>
    <p v-if="submitError" class="app-field-error mt-3" role="alert">{{ submitError }}</p>
    <template #footer>
      <AppDialogActions :busy="operating" @cancel="isEditDialogOpen = false" @confirm="save" />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { RefreshCw } from '@lucide/vue';
  import { computed, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { projectEnvironmentApi } from '@/api/project/environment';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type {
    EnvironmentResp,
    ProjectEnvironmentUpdateReq,
  } from '@/gen/proto/orbit/v1/environment/environment';
  import { formatTime } from '@/utils/time';

  const props = defineProps<{ projectId: string }>();
  const { t } = useI18n();
  const toast = useToast();
  const { loading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOperation } = useStatusAsync();
  const environment = ref<EnvironmentResp>();
  const loadError = ref('');
  const submitError = ref('');
  const isEditDialogOpen = ref(false);
  const form = reactive({
    state: 'active',
    platform: 'linux',
    host: '',
    port: 22,
    username: '',
    workspaceRoot: '',
    hostKeyFingerprint: '',
    privateKey: '',
    passphrase: '',
  });
  const errors = reactive({
    host: '',
    port: '',
    username: '',
    workspaceRoot: '',
    hostKeyFingerprint: '',
  });
  const stateOptions = computed(() => [
    { value: 'active', label: t('project.environment.states.active') },
    { value: 'disabled', label: t('project.environment.states.disabled') },
  ]);
  const platformOptions = computed(() => [
    { value: 'linux', label: t('project.environment.platforms.linux') },
    { value: 'windows', label: t('project.environment.platforms.windows') },
  ]);
  const workspaceRootPlaceholder = computed(() =>
    form.platform === 'windows'
      ? t('project.environment.windowsWorkspacePlaceholder')
      : t('project.environment.linuxWorkspacePlaceholder')
  );

  function resetForm() {
    if (!environment.value) return;
    form.state = environment.value.state;
    form.platform = environment.value.platform;
    form.host = environment.value.host;
    form.port = environment.value.port;
    form.username = environment.value.username;
    form.workspaceRoot = environment.value.workspace_root;
    form.hostKeyFingerprint = environment.value.host_key_fingerprint;
    form.privateKey = '';
    form.passphrase = '';
    Object.keys(errors).forEach((key) => {
      errors[key as keyof typeof errors] = '';
    });
    submitError.value = '';
  }

  function validate() {
    errors.host =
      form.host.trim() && !/\s/.test(form.host) ? '' : t('project.environment.validation.host');
    errors.port =
      Number.isInteger(form.port) && form.port >= 1 && form.port <= 65535
        ? ''
        : t('project.environment.validation.port');
    errors.username =
      form.username.trim() && !/[\r\n]/.test(form.username)
        ? ''
        : t('project.environment.validation.username');
    const validWorkspace =
      form.platform === 'linux'
        ? form.workspaceRoot.trim().startsWith('/')
        : /^[A-Za-z]:\\/.test(form.workspaceRoot.trim());
    errors.workspaceRoot = validWorkspace ? '' : t('project.environment.validation.workspaceRoot');
    errors.hostKeyFingerprint = /^SHA256:[A-Za-z0-9+/]+={0,2}$/.test(form.hostKeyFingerprint.trim())
      ? ''
      : t('project.environment.validation.hostKeyFingerprint');
    return !Object.values(errors).some(Boolean);
  }

  async function fetchEnvironment() {
    loadError.value = '';
    try {
      await execute(async () => {
        environment.value = await projectEnvironmentApi.get(props.projectId);
      });
    } catch (error: unknown) {
      loadError.value =
        error instanceof Error ? error.message : t('project.environment.loadFailed');
    }
  }

  function openEditDialog() {
    resetForm();
    isEditDialogOpen.value = true;
  }

  async function save() {
    submitError.value = '';
    if (!validate()) return;
    const input: ProjectEnvironmentUpdateReq = {
      state: form.state,
      platform: form.platform,
      host: form.host.trim(),
      port: form.port,
      username: form.username.trim(),
      workspace_root: form.workspaceRoot.trim(),
      host_key_fingerprint: form.hostKeyFingerprint.trim(),
    };
    if (form.privateKey.trim()) {
      input.deployment_ssh_private_key = form.privateKey.trim();
      if (form.passphrase) input.deployment_ssh_key_passphrase = form.passphrase;
    }
    try {
      await executeOperation(async () => {
        environment.value = await projectEnvironmentApi.update(props.projectId, input);
        toast.success(t('project.environment.updated'));
        isEditDialogOpen.value = false;
      });
    } catch (error: unknown) {
      submitError.value =
        error instanceof Error ? error.message : t('project.environment.saveFailed');
    }
  }

  async function probe() {
    if (!environment.value) return;
    try {
      await executeOperation(async () => {
        const result = await projectEnvironmentApi.probe(props.projectId);
        environment.value = result;
        if (result.last_probe_status === 'succeeded') {
          toast.success(t('project.environment.probeSucceeded'));
          return;
        }
        toast.error(result.last_probe_diagnostic || t('project.environment.probeFailed'));
      });
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('project.environment.probeFailed'));
    }
  }

  watch(
    () => form.privateKey,
    (privateKey) => {
      if (!privateKey.trim()) form.passphrase = '';
    }
  );
  watch(
    () => props.projectId,
    () => {
      void fetchEnvironment();
    },
    { immediate: true }
  );
</script>
