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
            v-if="isSSH"
            class="app-button h-9 px-3"
            :disabled="operating"
            @click="openSshCommand"
          >
            <KeyRound class="size-4" aria-hidden="true" />
            {{ t('project.initialization.sshCommand') }}
          </button>
        </template>
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>{{ t('project.environment.targetType') }}</dt>
            <dd class="text-foreground">
              {{ t(`project.environment.targetTypes.${environment.target_type}`) }}
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

    <SshInitializationCommandDialog
      v-model:open="showSshCommandDialog"
      :command="sshCommand"
      :loading="loadingSshCommand"
      :error="sshCommandError"
    />
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
  import SshInitializationCommandDialog from '@/components/SshInitializationCommandDialog.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import DetailPageHeader from '@/components/DetailPageHeader.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { EnvironmentResp } from '@/gen/proto/orbit/v1/environment/environment';
  import { formatTime } from '@/utils/time';
  import { buildLinuxSshInitializationCommand } from '@/utils/linuxSshCommand';
  import { buildWindowsSshInitializationCommand } from '@/utils/windowsSshCommand';
  import EnvironmentTargetFields from '@/views/environment/EnvironmentTargetFields.vue';
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
  const showSshCommandDialog = ref(false);
  const loadingSshCommand = ref(false);
  const sshCommand = ref('');
  const sshCommandError = ref('');
  const form = reactive(emptyEnvironmentForm());
  const errors = reactive(emptyEnvironmentFormErrors());
  const isLocal = computed(() => environment.value?.target_type === 'local');
  const isSSH = computed(() => environment.value?.target_type === 'ssh' && !!environment.value.ssh);

  function formatSSHAddress(host: string, port: number) {
    const displayHost = host.includes(':') && !host.startsWith('[') ? `[${host}]` : host;
    return `${displayHost}:${port}`;
  }

  function resetForm() {
    if (!environment.value) {
      return;
    }
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
    showSshCommandDialog.value = false;
    sshCommand.value = '';
    sshCommandError.value = '';
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

  async function openSshCommand() {
    const projectId = projectStore.activeProjectId;
    const target = environment.value?.ssh;
    if (!projectId || !isSSH.value || !target) {
      return;
    }
    showSshCommandDialog.value = true;
    loadingSshCommand.value = true;
    sshCommand.value = '';
    sshCommandError.value = '';
    try {
      const result = await projectEnvironmentApi.prepareSSHCommand(projectId, {
        target_type: 'ssh',
        ssh: {
          platform: target.platform,
          host: target.host,
          port: target.port,
          username: target.username,
          workspace_root: target.workspace_root,
        },
      });
      if (!result.environment || !result.public_key) {
        throw new Error(t('project.initialization.sshCommandFailed'));
      }
      environment.value = result.environment;
      sshCommand.value =
        target.platform === 'windows'
          ? buildWindowsSshInitializationCommand({
              host: target.host,
              port: target.port,
              username: target.username,
              workspaceRoot: target.workspace_root,
              publicKey: result.public_key,
            })
          : buildLinuxSshInitializationCommand({ publicKey: result.public_key });
    } catch (error: unknown) {
      sshCommandError.value =
        error instanceof Error ? error.message : t('project.initialization.sshCommandFailed');
    } finally {
      loadingSshCommand.value = false;
    }
  }

  async function save() {
    submitError.value = '';
    if (!validate()) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    const input = environmentUpdateRequestFromForm(form);
    try {
      await executeOperation(async () => {
        environment.value = await projectEnvironmentApi.update(projectId, input);
        showSshCommandDialog.value = false;
        sshCommand.value = '';
        sshCommandError.value = '';
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
    if (!environment.value || !projectId) {
      return;
    }
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
</script>
