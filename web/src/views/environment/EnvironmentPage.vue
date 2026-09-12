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
            v-if="isWindowsSSH"
            class="app-button h-9 px-3"
            :disabled="operating"
            @click="openWindowsCommand"
          >
            <KeyRound class="size-4" aria-hidden="true" />
            {{ t('project.environment.windowsCommand') }}
          </button>
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
              <router-link v-if="environment.gateway_application_id" to="/gateway" class="app-link">
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
      <form novalidate @submit.prevent="initialize">
        <EnvironmentBootstrapFields
          :model-value="initializeForm"
          :errors="initializeErrors"
          :disabled="operating"
          id-prefix="environment-bootstrap"
        />
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
      <form novalidate @submit.prevent="save">
        <EnvironmentTargetFields
          :model-value="form"
          :errors="errors"
          show-state
          :disabled="operating"
          :local-workspace-root="environment?.local?.workspace_root || ''"
          :local-display="environment?.local"
          id-prefix="environment-edit"
        />
        <button type="submit" class="sr-only" tabindex="-1" aria-hidden="true"></button>
      </form>
      <p v-if="submitError" class="app-field-error mt-3" role="alert">{{ submitError }}</p>
      <template #footer>
        <AppDialogActions :busy="operating" @cancel="isEditDialogOpen = false" @confirm="save" />
      </template>
    </AppDialog>

    <WindowsSshInitializationDialog
      v-model:open="showWindowsCommandDialog"
      :command="windowsCommand"
      :loading="loadingWindowsCommand"
      :error="windowsCommandError"
    />
  </div>
</template>

<script setup lang="ts">
  import { KeyRound, RefreshCw } from '@lucide/vue';
  import { computed, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { projectEnvironmentApi } from '@/api/project/environment';
  import { projectInitializationApi } from '@/api/project/initialization';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import WindowsSshInitializationDialog from '@/components/WindowsSshInitializationDialog.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import DetailPageHeader from '@/components/DetailPageHeader.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { EnvironmentResp } from '@/gen/proto/orbit/v1/environment/environment';
  import { formatTime } from '@/utils/time';
  import { buildWindowsSshInitializationCommand } from '@/utils/windowsSshCommand';
  import EnvironmentBootstrapFields from '@/views/environment/EnvironmentBootstrapFields.vue';
  import EnvironmentTargetFields from '@/views/environment/EnvironmentTargetFields.vue';
  import {
    emptyEnvironmentBootstrapForm,
    emptyEnvironmentBootstrapFormErrors,
    environmentBootstrapRequestFromForm,
    validateEnvironmentBootstrapForm,
  } from '@/views/environment/environmentBootstrapForm';
  import {
    assignEnvironmentFormErrors as assignTargetFormErrors,
    emptyEnvironmentForm,
    emptyEnvironmentFormErrors,
    environmentFormHasErrors,
    environmentUpdateRequestFromForm,
    hydrateEnvironmentForm,
    validateEnvironmentForm,
  } from '@/views/environment/environmentForm';

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
  const showWindowsCommandDialog = ref(false);
  const loadingWindowsCommand = ref(false);
  const windowsCommand = ref('');
  const windowsCommandError = ref('');
  const form = reactive(emptyEnvironmentForm());
  const errors = reactive(emptyEnvironmentFormErrors());
  const initializeForm = reactive(emptyEnvironmentBootstrapForm());
  const initializeErrors = reactive(emptyEnvironmentBootstrapFormErrors());
  const isLocal = computed(() => environment.value?.target_type === 'local');
  const isSSH = computed(() => environment.value?.target_type === 'ssh' && !!environment.value.ssh);
  const canInitialize = computed(
    () => environment.value?.state === 'active' && environment.value.ssh?.platform === 'linux'
  );
  const isWindowsSSH = computed(
    () => environment.value?.target_type === 'ssh' && environment.value.ssh?.platform === 'windows'
  );

  function formatSSHAddress(host: string, port: number) {
    const displayHost = host.includes(':') && !host.startsWith('[') ? `[${host}]` : host;
    return `${displayHost}:${port}`;
  }

  function resetForm() {
    if (!environment.value) return;
    Object.assign(form, hydrateEnvironmentForm(environment.value));
    assignTargetFormErrors(errors, emptyEnvironmentFormErrors());
    submitError.value = '';
  }

  function validate() {
    assignTargetFormErrors(
      errors,
      validateEnvironmentForm(form, {
        host: t('project.environment.validation.host'),
        port: t('project.environment.validation.port'),
        username: t('project.environment.validation.username'),
        workspaceRoot: t('project.environment.validation.workspaceRoot'),
      })
    );
    return !environmentFormHasErrors(errors);
  }

  async function fetchEnvironment() {
    const projectId = projectStore.activeProjectId;
    showWindowsCommandDialog.value = false;
    windowsCommand.value = '';
    windowsCommandError.value = '';
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
    Object.assign(initializeForm, emptyEnvironmentBootstrapForm());
    Object.assign(initializeErrors, emptyEnvironmentBootstrapFormErrors());
    initializeSubmitError.value = '';
  }

  function openInitializeDialog() {
    resetInitializeForm();
    initializeForm.username = environment.value?.ssh?.username || '';
    isInitializeDialogOpen.value = true;
  }

  async function openWindowsCommand() {
    const projectId = projectStore.activeProjectId;
    const target = environment.value?.ssh;
    if (!projectId || !isWindowsSSH.value || !target) return;
    showWindowsCommandDialog.value = true;
    loadingWindowsCommand.value = true;
    windowsCommand.value = '';
    windowsCommandError.value = '';
    try {
      const { public_key: publicKey } =
        await projectInitializationApi.getDeploymentPublicKey(projectId);
      windowsCommand.value = buildWindowsSshInitializationCommand({
        host: target.host,
        port: target.port,
        username: target.username,
        workspaceRoot: target.workspace_root,
        publicKey,
      });
    } catch (error: unknown) {
      windowsCommandError.value =
        error instanceof Error
          ? error.message
          : t('project.initialization.windowsTargetKeyLoadFailed');
    } finally {
      loadingWindowsCommand.value = false;
    }
  }

  function closeInitializeDialog() {
    isInitializeDialogOpen.value = false;
  }

  function validateInitialize() {
    Object.assign(
      initializeErrors,
      validateEnvironmentBootstrapForm(initializeForm, {
        username: t('project.environment.validation.bootstrapUsername'),
        password: t('project.environment.validation.bootstrapPassword'),
        privateKey: t('project.environment.validation.bootstrapPrivateKey'),
      })
    );
    return !initializeErrors.username && !initializeErrors.credential;
  }

  async function initialize() {
    initializeSubmitError.value = '';
    if (!validateInitialize()) return;
    const projectId = projectStore.activeProjectId;
    if (!projectId) return;
    const input = environmentBootstrapRequestFromForm(initializeForm);
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
    const input = environmentUpdateRequestFromForm(form);
    try {
      await executeOperation(async () => {
        environment.value = await projectEnvironmentApi.update(projectId, input);
        showWindowsCommandDialog.value = false;
        windowsCommand.value = '';
        windowsCommandError.value = '';
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
