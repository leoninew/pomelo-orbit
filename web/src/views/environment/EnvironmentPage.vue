<template>
  <div class="space-y-4">
    <DetailPageHeader :items="[]" :title="t('project.environment.title')" />

    <AppLoadingState v-if="loading" size="section" />
    <AppEmptyState v-else-if="!projectStore.activeProjectId" />
    <p v-else-if="loadError" class="py-4 text-sm text-destructive">{{ loadError }}</p>

    <template v-else-if="environment">
      <DetailInfoCard
        :title="t('project.environment.sections.basicInfo')"
        editable
        :disabled="operating"
        @edit="openEditDialog"
      >
        <template #actions>
          <button
            v-if="canInitialize"
            class="app-button h-9 px-3"
            :disabled="operating"
            @click="openInitializeDialog"
          >
            <KeyRound class="size-4" />
            {{ t('project.environment.initialize') }}
          </button>
        </template>
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>{{ t('project.environment.targetType') }}</dt>
            <dd class="text-foreground">
              {{ t(`project.environment.targetTypes.${environment.target_type}`) }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('project.environment.state') }}</dt>
            <dd>
              <AppBadge
                variant="status"
                :tone="environment.state === 'active' ? 'success' : 'default'"
              >
                {{ t(`project.environment.states.${environment.state}`) }}
              </AppBadge>
            </dd>
          </div>
          <div v-if="isLocal" class="flex gap-2 sm:col-span-2">
            <dt>{{ t('project.environment.workspaceRoot') }}</dt>
            <dd class="min-w-0 break-all text-foreground">
              {{ environment.local?.workspace_root || t('common.notSet') }}
            </dd>
          </div>
          <div v-if="isSSH && environment.ssh" class="flex gap-2">
            <dt>{{ t('project.environment.platform') }}</dt>
            <dd class="text-foreground">
              {{ t(`project.environment.platforms.${environment.ssh.platform}`) }}
            </dd>
          </div>
          <div v-if="isSSH && environment.ssh" class="flex gap-2">
            <dt>{{ t('project.environment.sshTarget') }}</dt>
            <dd class="min-w-0 break-all text-foreground">
              {{ formatSSHAddress(environment.ssh.host, environment.ssh.port) }}
            </dd>
          </div>
          <div v-if="isSSH && environment.ssh" class="flex gap-2">
            <dt>{{ t('project.environment.username') }}</dt>
            <dd class="text-foreground">{{ environment.ssh.username }}</dd>
          </div>
          <div v-if="isSSH && environment.ssh" class="flex gap-2 sm:col-span-2">
            <dt>{{ t('project.environment.workspaceRoot') }}</dt>
            <dd class="min-w-0 break-all text-foreground">
              {{ environment.ssh.workspace_root }}
            </dd>
          </div>
        </dl>
      </DetailInfoCard>

      <DetailInfoCard :title="t('project.environment.sections.readiness')">
        <template #actions>
          <button
            class="app-button h-9 px-3"
            :disabled="operating"
            :aria-busy="operating"
            @click="probe"
          >
            <RefreshCw class="size-4" :class="{ 'animate-spin': operating }" />
            {{ operating ? t('project.environment.probing') : t('project.environment.probe') }}
          </button>
        </template>
        <dl class="app-detail-info-grid">
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
          <div class="flex gap-2">
            <dt>{{ t('project.environment.lastProbeAt') }}</dt>
            <dd class="text-muted-foreground">
              {{
                environment.last_probe_at
                  ? formatTime(environment.last_probe_at)
                  : t('common.notSet')
              }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('project.environment.gateway') }}</dt>
            <dd class="text-foreground">
              <router-link
                v-if="environment.gateway_application_id"
                :to="`/gateway/${environment.gateway_application_id}`"
                class="app-link"
              >
                {{ t('project.environment.openGateway') }}
              </router-link>
              <span v-else class="text-muted-foreground">{{ t('common.notSet') }}</span>
            </dd>
          </div>
          <div v-if="isSSH && environment.ssh" class="flex gap-2 sm:col-span-2">
            <dt>{{ t('project.environment.hostKeyFingerprint') }}</dt>
            <dd class="min-w-0 break-all text-foreground">
              {{ environment.ssh.host_key_fingerprint || t('common.notSet') }}
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
    </template>

    <AppDialog
      v-model:open="isInitializeDialogOpen"
      :title="t('project.environment.initializeTitle')"
      width-class="w-[min(720px,calc(100vw-32px))]"
    >
      <form class="grid gap-4 sm:grid-cols-2" novalidate @submit.prevent="initialize">
        <div class="space-y-1.5 sm:col-span-2">
          <label class="app-field-label block">
            {{ t('project.environment.bootstrapUsername') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="initializeForm.username"
            class="app-input"
            :class="initializeErrors.username ? 'app-input-error' : ''"
            :disabled="operating"
            :aria-invalid="initializeErrors.username ? 'true' : undefined"
            :aria-describedby="initializeErrors.username ? 'initialize-username-error' : undefined"
            @input="initializeErrors.username = ''"
          />
          <p
            v-if="initializeErrors.username"
            id="initialize-username-error"
            class="app-field-error text-xs"
            role="alert"
          >
            {{ initializeErrors.username }}
          </p>
        </div>
        <div class="space-y-1.5 sm:col-span-2">
          <label class="app-field-label block">{{ t('project.environment.authentication') }}</label>
          <div class="flex h-10 overflow-hidden rounded-md border border-border bg-background">
            <button
              type="button"
              class="flex-1 px-3 text-sm text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
              :class="initializeForm.authType === 'password' ? 'bg-primary/10 text-primary' : ''"
              :disabled="operating"
              :aria-pressed="initializeForm.authType === 'password'"
              @click="selectInitializeAuth('password')"
            >
              {{ t('project.environment.authenticationModes.password') }}
            </button>
            <button
              type="button"
              class="flex-1 border-l border-border px-3 text-sm text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
              :class="initializeForm.authType === 'privateKey' ? 'bg-primary/10 text-primary' : ''"
              :disabled="operating"
              :aria-pressed="initializeForm.authType === 'privateKey'"
              @click="selectInitializeAuth('privateKey')"
            >
              {{ t('project.environment.authenticationModes.privateKey') }}
            </button>
          </div>
        </div>
        <template v-if="initializeForm.authType === 'password'">
          <div class="space-y-1.5 sm:col-span-2">
            <label class="app-field-label block">
              {{ t('project.environment.bootstrapPassword') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="initializeForm.password"
              type="password"
              autocomplete="current-password"
              class="app-input"
              :class="initializeErrors.credential ? 'app-input-error' : ''"
              :disabled="operating"
              :aria-invalid="initializeErrors.credential ? 'true' : undefined"
              :aria-describedby="
                initializeErrors.credential ? 'initialize-credential-error' : undefined
              "
              @input="initializeErrors.credential = ''"
            />
            <p
              v-if="initializeErrors.credential"
              id="initialize-credential-error"
              class="app-field-error text-xs"
              role="alert"
            >
              {{ initializeErrors.credential }}
            </p>
          </div>
        </template>
        <template v-else>
          <div class="space-y-1.5 sm:col-span-2">
            <label class="app-field-label block">
              {{ t('project.environment.bootstrapPrivateKey') }}
              <span class="text-destructive">*</span>
            </label>
            <textarea
              v-model="initializeForm.privateKey"
              rows="8"
              class="app-textarea w-full font-mono text-xs"
              :class="initializeErrors.credential ? 'app-input-error' : ''"
              :disabled="operating"
              :aria-invalid="initializeErrors.credential ? 'true' : undefined"
              :aria-describedby="
                initializeErrors.credential ? 'initialize-credential-error' : undefined
              "
              @input="initializeErrors.credential = ''"
            />
            <p
              v-if="initializeErrors.credential"
              id="initialize-credential-error"
              class="app-field-error text-xs"
              role="alert"
            >
              {{ initializeErrors.credential }}
            </p>
          </div>
          <div class="space-y-1.5 sm:col-span-2">
            <label class="app-field-label block">
              {{ t('project.environment.bootstrapPrivateKeyPassphrase') }}
            </label>
            <input
              v-model="initializeForm.privateKeyPassphrase"
              type="password"
              autocomplete="off"
              class="app-input"
              :disabled="operating"
            />
          </div>
        </template>
        <button type="submit" class="sr-only" tabindex="-1" aria-hidden="true"></button>
      </form>
      <p v-if="initializeSubmitError" class="app-field-error mt-3" role="alert">
        {{ initializeSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('project.environment.initialize')"
          @cancel="closeInitializeDialog"
          @confirm="initialize"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isEditDialogOpen"
      :title="t('project.environment.editTitle')"
      width-class="w-[min(720px,calc(100vw-32px))]"
      body-class="min-h-0 flex-1 space-y-4 overflow-y-auto px-6 py-4"
      content-class="max-h-[calc(100vh-32px)] flex flex-col"
    >
      <form class="grid gap-4 sm:grid-cols-2" novalidate @submit.prevent="save">
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('project.environment.targetType') }}</label>
          <SelectControl
            :model-value="form.targetType"
            :options="targetTypeOptions"
            :disabled="operating"
            @update:model-value="updateTargetType"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('project.environment.state') }}</label>
          <SelectControl v-model="form.state" :options="stateOptions" :disabled="operating" />
        </div>
        <template v-if="form.targetType === 'ssh'">
          <div class="space-y-1.5">
            <label class="app-field-label block">{{ t('project.environment.platform') }}</label>
            <SelectControl
              :model-value="form.platform"
              :options="platformOptions"
              :disabled="operating"
              @update:model-value="updatePlatform"
            />
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
        </template>
        <button type="submit" class="sr-only" tabindex="-1" aria-hidden="true"></button>
      </form>
      <p v-if="submitError" class="app-field-error mt-3" role="alert">{{ submitError }}</p>
      <template #footer>
        <AppDialogActions :busy="operating" @cancel="isEditDialogOpen = false" @confirm="save" />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { KeyRound, RefreshCw } from '@lucide/vue';
  import { computed, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { projectEnvironmentApi } from '@/api/project/environment';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import DetailPageHeader from '@/components/DetailPageHeader.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type {
    EnvironmentResp,
    ProjectEnvironmentInitializeReq,
    ProjectEnvironmentUpdateReq,
  } from '@/gen/proto/orbit/v1/environment/environment';
  import {
    defaultDeploymentWorkspaceRoot,
    isPlatformWorkspaceRoot,
  } from '@/utils/deploymentEnvironment';
  import { formatTime } from '@/utils/time';

  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { loading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOperation } = useStatusAsync();
  const environment = ref<EnvironmentResp>();
  const loadError = ref('');
  const submitError = ref('');
  const isEditDialogOpen = ref(false);
  const isInitializeDialogOpen = ref(false);
  const initializeSubmitError = ref('');
  const form = reactive({
    state: 'active',
    targetType: 'local',
    platform: 'linux',
    host: '',
    port: 22,
    username: '',
    workspaceRoot: '',
  });
  const errors = reactive({
    host: '',
    port: '',
    username: '',
    workspaceRoot: '',
  });
  const initializeForm = reactive({
    username: '',
    authType: 'password' as 'password' | 'privateKey',
    password: '',
    privateKey: '',
    privateKeyPassphrase: '',
  });
  const initializeErrors = reactive({
    username: '',
    credential: '',
  });
  const stateOptions = computed(() => [
    { value: 'active', label: t('project.environment.states.active') },
    { value: 'disabled', label: t('project.environment.states.disabled') },
  ]);
  const platformOptions = computed(() => [
    { value: 'linux', label: t('project.environment.platforms.linux') },
    { value: 'windows', label: t('project.environment.platforms.windows') },
  ]);
  const targetTypeOptions = computed(() => [
    { value: 'local', label: t('project.environment.targetTypes.local') },
    { value: 'ssh', label: t('project.environment.targetTypes.ssh') },
  ]);
  const isLocal = computed(() => environment.value?.target_type === 'local');
  const isSSH = computed(() => environment.value?.target_type === 'ssh' && !!environment.value.ssh);
  const canInitialize = computed(
    () => environment.value?.state === 'active' && environment.value.ssh?.platform === 'linux'
  );
  const workspaceRootPlaceholder = defaultDeploymentWorkspaceRoot();

  function formatSSHAddress(host: string, port: number) {
    const displayHost = host.includes(':') && !host.startsWith('[') ? `[${host}]` : host;
    return `${displayHost}:${port}`;
  }

  function resetForm() {
    if (!environment.value) return;
    form.state = environment.value.state;
    form.targetType = environment.value.target_type;
    const ssh = environment.value.ssh;
    form.platform = ssh?.platform || 'linux';
    form.host = ssh?.host || '';
    form.port = ssh?.port || 22;
    form.username = ssh?.username || '';
    form.workspaceRoot = ssh?.workspace_root || defaultDeploymentWorkspaceRoot();
    Object.keys(errors).forEach((key) => {
      errors[key as keyof typeof errors] = '';
    });
    submitError.value = '';
  }

  function applyLocalSSHPreset() {
    const local = environment.value?.local;
    if (
      environment.value?.target_type !== 'local' ||
      form.targetType !== 'ssh' ||
      !local ||
      form.platform !== local.platform
    ) {
      return;
    }
    form.host = local.host;
    form.username = local.username;
    errors.host = '';
    errors.username = '';
  }

  function updateTargetType(value: string | number) {
    form.targetType = String(value);
    if (form.targetType === 'ssh') {
      applyLocalSSHPreset();
    }
  }

  function updatePlatform(value: string | number) {
    const platform = String(value);
    if (platform === form.platform) return;
    form.platform = platform;
    if (!isPlatformWorkspaceRoot(platform, form.workspaceRoot)) {
      form.workspaceRoot = defaultDeploymentWorkspaceRoot();
    }
    applyLocalSSHPreset();
  }

  function validate() {
    if (form.targetType === 'local') {
      Object.keys(errors).forEach((key) => {
        errors[key as keyof typeof errors] = '';
      });
      return true;
    }
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
    errors.workspaceRoot = isPlatformWorkspaceRoot(form.platform, form.workspaceRoot)
      ? ''
      : t('project.environment.validation.workspaceRoot');
    return !Object.values(errors).some(Boolean);
  }

  async function fetchEnvironment() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      environment.value = undefined;
      loadError.value = '';
      return;
    }
    loadError.value = '';
    try {
      await execute(async () => {
        environment.value = await projectEnvironmentApi.get(projectId);
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

  function resetInitializeForm() {
    initializeForm.username = '';
    initializeForm.authType = 'password';
    initializeForm.password = '';
    initializeForm.privateKey = '';
    initializeForm.privateKeyPassphrase = '';
    initializeErrors.username = '';
    initializeErrors.credential = '';
    initializeSubmitError.value = '';
  }

  function openInitializeDialog() {
    resetInitializeForm();
    initializeForm.username = environment.value?.ssh?.username || '';
    isInitializeDialogOpen.value = true;
  }

  function closeInitializeDialog() {
    isInitializeDialogOpen.value = false;
  }

  function selectInitializeAuth(authType: 'password' | 'privateKey') {
    if (initializeForm.authType === authType) return;
    initializeForm.authType = authType;
    initializeForm.password = '';
    initializeForm.privateKey = '';
    initializeForm.privateKeyPassphrase = '';
    initializeErrors.credential = '';
  }

  function validateInitialize() {
    initializeErrors.username =
      initializeForm.username.trim() && !/[\r\n]/.test(initializeForm.username)
        ? ''
        : t('project.environment.validation.bootstrapUsername');
    initializeErrors.credential =
      initializeForm.authType === 'password'
        ? initializeForm.password
          ? ''
          : t('project.environment.validation.bootstrapPassword')
        : initializeForm.privateKey.trim()
          ? ''
          : t('project.environment.validation.bootstrapPrivateKey');
    return !initializeErrors.username && !initializeErrors.credential;
  }

  async function initialize() {
    initializeSubmitError.value = '';
    if (!validateInitialize()) return;
    const projectId = projectStore.activeProjectId;
    if (!projectId) return;
    const input: ProjectEnvironmentInitializeReq = {
      username: initializeForm.username.trim(),
      password: initializeForm.authType === 'password' ? initializeForm.password : '',
      private_key: initializeForm.authType === 'privateKey' ? initializeForm.privateKey : '',
      private_key_passphrase:
        initializeForm.authType === 'privateKey' ? initializeForm.privateKeyPassphrase : '',
    };
    try {
      await executeOperation(async () => {
        environment.value = await projectEnvironmentApi.initialize(projectId, input);
        toast.success(t('project.environment.initialized'));
        closeInitializeDialog();
      });
    } catch (error: unknown) {
      initializeSubmitError.value =
        error instanceof Error ? error.message : t('project.environment.initializeFailed');
    }
  }

  async function save() {
    submitError.value = '';
    if (!validate()) return;
    const projectId = projectStore.activeProjectId;
    if (!projectId) return;
    const input: ProjectEnvironmentUpdateReq = {
      state: form.state,
      target_type: form.targetType,
      ssh:
        form.targetType === 'ssh'
          ? {
              platform: form.platform,
              host: form.host.trim(),
              port: form.port,
              username: form.username.trim(),
              workspace_root: form.workspaceRoot.trim(),
            }
          : undefined,
    };
    try {
      await executeOperation(async () => {
        environment.value = await projectEnvironmentApi.update(projectId, input);
        toast.success(t('project.environment.updated'));
        isEditDialogOpen.value = false;
      });
    } catch (error: unknown) {
      submitError.value =
        error instanceof Error ? error.message : t('project.environment.saveFailed');
    }
  }

  async function probe() {
    const projectId = projectStore.activeProjectId;
    if (!environment.value || !projectId) return;
    try {
      await executeOperation(async () => {
        const result = await projectEnvironmentApi.probe(projectId);
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
    () => projectStore.activeProjectId,
    () => {
      void fetchEnvironment();
    },
    { immediate: true }
  );

  watch(isInitializeDialogOpen, (open) => {
    if (!open) resetInitializeForm();
  });
</script>
