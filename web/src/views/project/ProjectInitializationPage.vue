<template>
  <div class="mx-auto max-w-6xl space-y-6">
    <div class="space-y-2">
      <DetailPageHeader :items="[]" :title="t('project.initialization.title')" />
      <p class="max-w-3xl text-sm text-muted-foreground">
        {{ t('project.initialization.description') }}
      </p>
    </div>

    <AppLoadingState v-if="loading && !status" size="section" />
    <section v-else-if="loadError" class="app-surface space-y-4 p-6" aria-live="polite">
      <p class="text-sm text-destructive">{{ loadError }}</p>
      <button type="button" class="app-button h-9 px-3" @click="reloadStatus">
        {{ t('common.retry') }}
      </button>
    </section>

    <section v-else-if="status" class="grid items-start gap-6 lg:grid-cols-[15rem_minmax(0,1fr)]">
      <aside class="app-surface p-4 lg:sticky lg:top-6">
        <StepperRoot
          :model-value="selectedStep"
          orientation="vertical"
          :linear="true"
          class="space-y-1"
          :aria-label="t('project.initialization.title')"
          @update:model-value="handleStepChange"
        >
          <StepperItem
            v-for="item in stepItems"
            :key="item.key"
            v-slot="{ state }"
            :step="item.step"
            :completed="item.step < progressStep"
            :disabled="
              item.step > progressStep || (item.step === 3 && environmentHasUnsavedChanges)
            "
            class="group"
          >
            <div class="flex min-w-0 items-start gap-3">
              <StepperTrigger
                class="mt-0.5 inline-flex size-9 shrink-0 items-center justify-center rounded-full border text-sm font-medium outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed"
                :class="[
                  state === 'active'
                    ? 'border-primary bg-primary text-primary-foreground'
                    : state === 'completed'
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-border bg-background text-muted-foreground',
                  item.step > progressStep || (item.step === 3 && environmentHasUnsavedChanges)
                    ? 'opacity-60'
                    : 'hover:border-primary/60',
                ]"
              >
                <StepperIndicator class="inline-flex size-4 items-center justify-center">
                  <Check v-if="state === 'completed'" class="size-4" aria-hidden="true" />
                  <span v-else>{{ item.step }}</span>
                </StepperIndicator>
              </StepperTrigger>
              <div class="min-w-0 pt-0.5">
                <StepperTitle
                  class="block text-sm font-medium text-foreground transition-colors group-data-[disabled]:text-muted-foreground"
                >
                  {{ item.title }}
                </StepperTitle>
                <StepperDescription class="mt-1 block text-xs leading-5 text-muted-foreground">
                  {{ item.hint }}
                </StepperDescription>
              </div>
            </div>
            <StepperSeparator
              v-if="item.step < stepItems.length"
              class="ml-[1.125rem] my-2 h-7 w-px"
              :class="item.step < progressStep ? 'bg-primary' : 'bg-border'"
            />
          </StepperItem>
        </StepperRoot>
      </aside>

      <div class="app-surface min-w-0">
        <div class="space-y-6 p-5 sm:p-6">
          <form
            v-if="currentStep === 'environment'"
            class="space-y-4"
            novalidate
            @submit.prevent="saveEnvironment"
          >
            <EnvironmentTargetFields
              :model-value="environmentForm"
              :errors="environmentErrors"
              :disabled="operating"
              :local-workspace-root="localWorkspaceRoot"
              :local-display="localDisplay"
              id-prefix="initialization"
            />

            <hr class="border-border" />
            <p v-if="environmentSubmitError" class="app-field-error" role="alert">
              {{ environmentSubmitError }}
            </p>
            <div class="flex flex-wrap justify-end gap-2">
              <button
                v-if="isRemoteSSH"
                type="button"
                class="app-button h-9 px-4"
                :disabled="operating || loadingSshCommand"
                :aria-busy="loadingSshCommand"
                @click="openSshCommand"
              >
                <LoaderCircle
                  v-if="loadingSshCommand"
                  class="size-4 animate-spin"
                  aria-hidden="true"
                />
                {{ t('project.initialization.sshCommand') }}
              </button>
              <button
                type="submit"
                class="app-button-primary h-9 px-4"
                :disabled="operating || loadingSshCommand"
                :aria-busy="activeOperation === 'saving'"
              >
                <LoaderCircle
                  v-if="activeOperation === 'saving'"
                  class="size-4 animate-spin"
                  aria-hidden="true"
                />
                {{ t('project.initialization.next') }}
                <ArrowRight v-if="activeOperation !== 'saving'" class="size-4" aria-hidden="true" />
              </button>
            </div>
          </form>

          <section
            v-else-if="currentStep === 'probe'"
            class="space-y-5"
            aria-labelledby="initialization-probe-checklist-heading"
          >
            <div class="space-y-3">
              <h2 id="initialization-probe-checklist-heading" class="app-detail-section-title">
                {{ t('project.initialization.probeChecklistTitle') }}
              </h2>
              <div class="overflow-hidden rounded-lg border border-border">
                <ul class="divide-y divide-border">
                  <li
                    v-for="(item, index) in probeChecks"
                    :key="item.key"
                    class="flex items-center gap-3 px-4 py-3 sm:px-5"
                  >
                    <div
                      class="flex size-7 shrink-0 items-center justify-center rounded-full"
                      :class="probeCheckIconClass(index)"
                    >
                      <LoaderCircle
                        v-if="probeCheckState(index) === 'checking'"
                        class="size-4 animate-spin"
                        aria-hidden="true"
                      />
                      <CheckCircle2
                        v-else-if="probeCheckState(index) === 'success'"
                        class="size-4"
                        aria-hidden="true"
                      />
                      <CircleDashed v-else class="size-4" aria-hidden="true" />
                    </div>
                    <span class="min-w-0 flex-1 text-sm text-foreground">
                      {{ item.label }}
                    </span>
                    <span class="text-xs text-muted-foreground">
                      {{ probeCheckStateLabel(index) }}
                    </span>
                  </li>
                </ul>
              </div>
            </div>

            <hr class="border-border" />

            <p
              v-if="probeStatus === 'failed' && probeDiagnostic"
              class="app-field-error"
              role="alert"
            >
              {{ probeDiagnostic }}
            </p>
            <p v-else-if="probeStatus === 'succeeded'" class="text-xs text-muted-foreground">
              {{ t('project.initialization.probeLastRun') }}:
              <span class="text-foreground">{{ formatTime(probeLastRun) }}</span>
            </p>
            <p v-if="environmentSubmitError" class="app-field-error" role="alert">
              {{ environmentSubmitError }}
            </p>

            <div class="flex flex-wrap justify-end gap-2">
              <button
                type="button"
                class="app-button h-9 px-4"
                :disabled="operating"
                :aria-busy="activeOperation === 'probing'"
                @click="probeEnvironment"
              >
                <LoaderCircle
                  v-if="activeOperation === 'probing'"
                  class="size-4 animate-spin"
                  aria-hidden="true"
                />
                {{
                  activeOperation === 'probing'
                    ? t('project.initialization.probing')
                    : t('project.initialization.probe')
                }}
              </button>
              <button
                type="button"
                class="app-button-primary h-9 px-4"
                :disabled="operating || !canContinueToGateway"
                @click="selectStep(3)"
              >
                {{ t('project.initialization.next') }}
                <ArrowRight class="size-4" aria-hidden="true" />
              </button>
            </div>
          </section>

          <form v-else class="space-y-8" novalidate @submit.prevent="createGateway">
            <section class="space-y-4" aria-labelledby="initialization-gateway-control-plane">
              <h2 id="initialization-gateway-control-plane" class="app-detail-section-title">
                {{ t('gateway.sections.controlPlane') }}
              </h2>
              <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div class="space-y-1.5">
                  <label class="app-field-label block" for="initialization-gateway-image">
                    {{ t('gateway.fields.image') }}
                    <span class="text-destructive">*</span>
                  </label>
                  <input
                    id="initialization-gateway-image"
                    :value="gatewayForm.initial_component_image"
                    type="text"
                    class="app-input"
                    :class="gatewayErrors.initial_component_image ? 'app-input-error' : ''"
                    :aria-invalid="gatewayErrors.initial_component_image ? 'true' : undefined"
                    @input="
                      updateGatewayField(
                        'initial_component_image',
                        ($event.target as HTMLInputElement).value
                      )
                    "
                  />
                  <p
                    v-if="gatewayErrors.initial_component_image"
                    class="app-field-error"
                    role="alert"
                  >
                    {{ gatewayValidationMessage('initial_component_image') }}
                  </p>
                </div>
                <div class="space-y-1.5">
                  <label class="app-field-label block" for="initialization-gateway-rest-api-url">
                    {{ t('gateway.fields.restApiUrl') }}
                    <span class="text-destructive">*</span>
                  </label>
                  <input
                    id="initialization-gateway-rest-api-url"
                    :value="gatewayForm.rest_api_url"
                    type="url"
                    class="app-input"
                    :class="gatewayErrors.rest_api_url ? 'app-input-error' : ''"
                    :aria-invalid="gatewayErrors.rest_api_url ? 'true' : undefined"
                    @input="
                      updateGatewayField('rest_api_url', ($event.target as HTMLInputElement).value)
                    "
                  />
                  <p v-if="gatewayErrors.rest_api_url" class="app-field-error" role="alert">
                    {{ gatewayValidationMessage('rest_api_url') }}
                  </p>
                </div>
                <div class="space-y-1.5">
                  <label
                    class="app-field-label block"
                    for="initialization-gateway-rest-api-host-url"
                  >
                    {{ t('gateway.fields.restApiHostUrl') }}
                    <span class="text-destructive">*</span>
                  </label>
                  <input
                    id="initialization-gateway-rest-api-host-url"
                    :value="gatewayForm.rest_api_host_url"
                    type="url"
                    class="app-input"
                    :class="gatewayErrors.rest_api_host_url ? 'app-input-error' : ''"
                    :aria-invalid="gatewayErrors.rest_api_host_url ? 'true' : undefined"
                    @input="
                      updateGatewayField(
                        'rest_api_host_url',
                        ($event.target as HTMLInputElement).value
                      )
                    "
                  />
                  <p v-if="gatewayErrors.rest_api_host_url" class="app-field-error" role="alert">
                    {{ gatewayValidationMessage('rest_api_host_url') }}
                  </p>
                </div>
                <div class="space-y-1.5">
                  <label class="app-field-label block" for="initialization-gateway-timeout">
                    {{ t('gateway.fields.restReadyTimeout') }}
                    <span class="text-destructive">*</span>
                  </label>
                  <input
                    id="initialization-gateway-timeout"
                    :value="gatewayForm.rest_ready_timeout_seconds"
                    type="number"
                    min="1"
                    max="300"
                    class="app-input"
                    :class="gatewayErrors.rest_ready_timeout_seconds ? 'app-input-error' : ''"
                    :aria-invalid="gatewayErrors.rest_ready_timeout_seconds ? 'true' : undefined"
                    @input="
                      updateGatewayField(
                        'rest_ready_timeout_seconds',
                        ($event.target as HTMLInputElement).value
                      )
                    "
                  />
                  <p
                    v-if="gatewayErrors.rest_ready_timeout_seconds"
                    class="app-field-error"
                    role="alert"
                  >
                    {{ gatewayValidationMessage('rest_ready_timeout_seconds') }}
                  </p>
                </div>
                <div class="space-y-1.5">
                  <label class="app-field-label block" for="initialization-gateway-base-domain">
                    {{ t('gateway.fields.baseDomain') }}
                    <span class="text-destructive">*</span>
                  </label>
                  <input
                    id="initialization-gateway-base-domain"
                    :value="gatewayForm.base_domain"
                    type="text"
                    class="app-input"
                    :class="gatewayErrors.base_domain ? 'app-input-error' : ''"
                    :aria-invalid="gatewayErrors.base_domain ? 'true' : undefined"
                    @input="
                      updateGatewayField('base_domain', ($event.target as HTMLInputElement).value)
                    "
                  />
                  <p v-if="gatewayErrors.base_domain" class="app-field-error" role="alert">
                    {{ gatewayValidationMessage('base_domain') }}
                  </p>
                </div>
              </div>
            </section>

            <section class="space-y-4" aria-labelledby="initialization-gateway-ingress">
              <h2 id="initialization-gateway-ingress" class="app-detail-section-title">
                {{ t('gateway.sections.ingressDefaults') }}
              </h2>
              <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div class="space-y-1.5">
                  <label class="app-field-label block" for="initialization-gateway-entrypoint">
                    {{ t('gateway.fields.defaultEntrypoint') }}
                    <span class="text-destructive">*</span>
                  </label>
                  <SelectControl
                    id="initialization-gateway-entrypoint"
                    :model-value="gatewayForm.default_entrypoint"
                    :options="entrypointOptions"
                    :invalid="Boolean(gatewayErrors.default_entrypoint)"
                    @update:model-value="updateGatewayField('default_entrypoint', String($event))"
                  />
                  <p v-if="gatewayErrors.default_entrypoint" class="app-field-error" role="alert">
                    {{ gatewayValidationMessage('default_entrypoint') }}
                  </p>
                </div>
                <div class="space-y-1.5">
                  <label class="app-field-label block" for="initialization-gateway-tls-mode">
                    {{ t('gateway.fields.tlsMode') }}
                    <span class="text-destructive">*</span>
                  </label>
                  <SelectControl
                    id="initialization-gateway-tls-mode"
                    :model-value="gatewayForm.tls_mode"
                    :options="tlsModeOptions"
                    :invalid="Boolean(gatewayErrors.tls_mode)"
                    @update:model-value="updateGatewayField('tls_mode', String($event))"
                  />
                  <p v-if="gatewayErrors.tls_mode" class="app-field-error" role="alert">
                    {{ gatewayValidationMessage('tls_mode') }}
                  </p>
                </div>
              </div>
            </section>

            <section class="space-y-4" aria-labelledby="initialization-gateway-certificates">
              <h2 id="initialization-gateway-certificates" class="app-detail-section-title">
                {{ t('gateway.sections.routeCertificates') }}
              </h2>
              <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div class="space-y-1.5">
                  <label class="app-field-label block" for="initialization-gateway-acme-profile">
                    {{ t('gateway.fields.acmeProfile') }}
                  </label>
                  <SelectControl
                    id="initialization-gateway-acme-profile"
                    :model-value="acmeProfileValue"
                    :options="acmeProfileOptions"
                    :invalid="Boolean(gatewayErrors.acme_profile)"
                    @update:model-value="handleAcmeProfileChange"
                  />
                  <p v-if="gatewayErrors.acme_profile" class="app-field-error" role="alert">
                    {{ gatewayValidationMessage('acme_profile') }}
                  </p>
                </div>
                <div v-if="gatewayForm.acme_profile" class="space-y-1.5">
                  <label class="app-field-label block" for="initialization-gateway-acme-email">
                    {{ t('gateway.fields.acmeEmail') }}
                    <span class="text-destructive">*</span>
                  </label>
                  <input
                    id="initialization-gateway-acme-email"
                    :value="gatewayForm.acme_email"
                    type="email"
                    class="app-input"
                    :class="gatewayErrors.acme_email ? 'app-input-error' : ''"
                    :aria-invalid="gatewayErrors.acme_email ? 'true' : undefined"
                    @input="
                      updateGatewayField('acme_email', ($event.target as HTMLInputElement).value)
                    "
                  />
                  <p v-if="gatewayErrors.acme_email" class="app-field-error" role="alert">
                    {{ gatewayValidationMessage('acme_email') }}
                  </p>
                </div>
                <div
                  v-if="usesDNSProfile(gatewayForm.acme_profile)"
                  class="space-y-1.5 md:col-span-2"
                >
                  <label class="app-field-label block" for="initialization-gateway-dns-token">
                    {{ t('gateway.fields.dnsApiToken') }}
                    <span class="text-destructive">*</span>
                  </label>
                  <input
                    id="initialization-gateway-dns-token"
                    :value="gatewayForm.dns_api_token"
                    type="password"
                    class="app-input"
                    :class="gatewayErrors.dns_api_token ? 'app-input-error' : ''"
                    :aria-invalid="gatewayErrors.dns_api_token ? 'true' : undefined"
                    @input="
                      updateGatewayField('dns_api_token', ($event.target as HTMLInputElement).value)
                    "
                  />
                  <p v-if="gatewayErrors.dns_api_token" class="app-field-error" role="alert">
                    {{ gatewayValidationMessage('dns_api_token') }}
                  </p>
                </div>
              </div>
            </section>

            <hr class="border-border" />
            <p v-if="gatewaySubmitError" class="app-field-error" role="alert">
              {{ gatewaySubmitError }}
            </p>

            <div class="flex flex-wrap items-center justify-between gap-3">
              <button
                type="button"
                class="app-button h-9 px-3"
                :disabled="operating"
                @click="selectStep(selectedStep - 1)"
              >
                <ChevronLeft class="size-4" aria-hidden="true" />
                {{ t('common.back') }}
              </button>
              <button
                type="submit"
                class="app-button-primary h-9 px-4"
                :disabled="operating"
                :aria-busy="activeOperation === 'creating'"
              >
                <LoaderCircle
                  v-if="activeOperation === 'creating'"
                  class="size-4 animate-spin"
                  aria-hidden="true"
                />
                {{
                  activeOperation === 'creating'
                    ? t('project.initialization.creating')
                    : t('project.initialization.createGateway')
                }}
                <ArrowRight
                  v-if="activeOperation !== 'creating'"
                  class="size-4"
                  aria-hidden="true"
                />
              </button>
            </div>
          </form>
        </div>
      </div>
    </section>

    <SshInitializationCommandDialog
      v-model:open="showSshCommandDialog"
      :command="sshCommand"
      :loading="loadingSshCommand"
      :error="sshCommandError"
    />
  </div>
</template>

<script setup lang="ts">
  import {
    ArrowRight,
    Check,
    CheckCircle2,
    ChevronLeft,
    CircleDashed,
    LoaderCircle,
  } from '@lucide/vue';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import {
    StepperDescription,
    StepperIndicator,
    StepperItem,
    StepperRoot,
    StepperSeparator,
    StepperTitle,
    StepperTrigger,
  } from 'reka-ui';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import SshInitializationCommandDialog from '@/components/SshInitializationCommandDialog.vue';
  import DetailPageHeader from '@/components/DetailPageHeader.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import {
    READY_INITIALIZATION_STATUS,
    useProjectInitializationStore,
  } from '@/stores/projectInitialization';
  import { useProjectStore } from '@/stores/project';
  import { resolveInitializationCompletionRedirect } from '@/router/projectReadiness';
  import { ApiError } from '@/utils/request';
  import { formatTime } from '@/utils/time';
  import { buildLinuxSshInitializationCommand } from '@/utils/linuxSshCommand';
  import { buildWindowsSshInitializationCommand } from '@/utils/windowsSshCommand';
  import EnvironmentTargetFields from '@/views/environment/EnvironmentTargetFields.vue';
  import {
    assignEnvironmentFormErrors,
    emptyEnvironmentForm,
    emptyEnvironmentFormErrors,
    environmentFormDirty,
    environmentFormHasErrors,
    hydrateEnvironmentForm,
    initializationEnvironmentRequestFromForm,
    validateEnvironmentForm,
  } from '@/views/environment/environmentForm';
  import {
    emptyGatewayConfigForm,
    gatewayConfigFormFromDefaults,
    gatewayInitializeRequestFromForm,
    usesDNSProfile,
    validateGatewayConfigForm,
    type GatewayAcmeProfile,
    type GatewayConfigForm,
    type GatewayConfigFormErrors,
  } from '@/views/gateway/gatewayConfigForm';

  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const toast = useToast();
  const projectStore = useProjectStore();
  const initializationStore = useProjectInitializationStore();
  const { loading: operating, execute: executeOperation } = useStatusAsync();

  const activeOperation = ref<'idle' | 'saving' | 'probing' | 'creating'>('idle');
  const loadError = ref('');
  const environmentSubmitError = ref('');
  const gatewaySubmitError = ref('');
  const gatewayHydrated = ref(false);
  const loadingSshCommand = ref(false);
  const showSshCommandDialog = ref(false);
  const sshCommand = ref('');
  const sshCommandError = ref('');
  const environmentForm = reactive(emptyEnvironmentForm());
  const environmentErrors = reactive(emptyEnvironmentFormErrors());
  const gatewayForm = reactive<GatewayConfigForm>(emptyGatewayConfigForm());
  const gatewayErrors = reactive<GatewayConfigFormErrors>({});
  const noAcmeProfileValue = '__acme_disabled__';
  const entrypointOptions = [
    { value: 'web', label: 'web' },
    { value: 'websecure', label: 'websecure' },
  ];
  const tlsModeOptions = [
    { value: 'none', label: 'none' },
    { value: 'tls', label: 'tls' },
    { value: 'letsencrypt', label: 'letsencrypt' },
  ];
  const acmeProfileOptions = computed(() => [
    { value: noAcmeProfileValue, label: t('gateway.acmeProfiles.none') },
    { value: 'http', label: t('gateway.acmeProfiles.http') },
    { value: 'dns', label: t('gateway.acmeProfiles.dns') },
    { value: 'http-dns', label: t('gateway.acmeProfiles.httpDns') },
  ]);

  const status = computed(() => initializationStore.statusFor(projectStore.activeProjectId ?? ''));
  const loading = computed(() => initializationStore.loading);
  const localWorkspaceRoot = computed(() => status.value?.defaults?.local_workspace_root || '');
  const isRemoteSSH = computed(() => environmentForm.targetType === 'ssh');
  const localDisplay = computed(() => {
    const defaults = status.value?.defaults;
    if (!defaults) {
      return status.value?.environment?.local;
    }
    return {
      workspace_root: defaults.local_workspace_root || '',
      platform: defaults.local_platform || '',
      host: defaults.local_host || '',
      username: defaults.local_username || '',
    };
  });
  const acmeProfileValue = computed(() => gatewayForm.acme_profile || noAcmeProfileValue);
  const stepItems = computed(() => [
    {
      step: 1,
      key: 'environment',
      title: t('project.initialization.steps.environment'),
      hint: t('project.initialization.steps.environmentHint'),
    },
    {
      step: 2,
      key: 'probe',
      title: t('project.initialization.steps.probe'),
      hint: t('project.initialization.steps.probeHint'),
    },
    {
      step: 3,
      key: 'gateway',
      title: t('project.initialization.steps.gateway'),
      hint: t('project.initialization.steps.gatewayHint'),
    },
  ]);
  const progressStep = computed(() => stepFromStatus(status.value?.status));
  const selectedStep = ref(1);
  const currentStep = computed(() => stepItems.value[selectedStep.value - 1]?.key || 'environment');
  const environmentHasUnsavedChanges = computed(() =>
    Boolean(
      status.value?.environment && environmentFormDirty(environmentForm, status.value.environment)
    )
  );
  const probeStatus = computed(() => {
    const environment = status.value?.environment;
    if (!environment || environment.last_probe_revision !== environment.target_revision) {
      return '';
    }
    return environment.last_probe_status || '';
  });
  const isProbing = computed(() => activeOperation.value === 'probing');
  const probeDiagnostic = computed(() => status.value?.environment?.last_probe_diagnostic || '');
  const probeLastRun = computed(() => status.value?.environment?.last_probe_at);
  const probeChecks = computed(() => {
    const items = [
      {
        key: environmentForm.targetType === 'local' ? 'workspace' : 'target',
        label: t(
          environmentForm.targetType === 'local'
            ? 'project.initialization.probeChecks.workspace'
            : 'project.initialization.probeChecks.target'
        ),
      },
      { key: 'docker', label: t('project.initialization.probeChecks.docker') },
      { key: 'compose', label: t('project.initialization.probeChecks.compose') },
    ];
    if (environmentForm.targetType === 'ssh' && environmentForm.platform === 'windows') {
      items.splice(1, 0, {
        key: 'wsl',
        label: t('project.initialization.probeChecks.wsl'),
      });
    }
    return items;
  });

  type ProbeCheckState = 'pending' | 'checking' | 'success';

  function probeCheckState(_index: number): ProbeCheckState {
    if (isProbing.value) {
      return 'checking';
    }
    if (probeStatus.value === 'succeeded') {
      return 'success';
    }
    return 'pending';
  }

  function probeCheckIconClass(index: number) {
    const state = probeCheckState(index);
    if (state === 'success') {
      return 'bg-green-500/10 text-green-600 dark:text-green-400';
    }
    if (state === 'checking') {
      return 'bg-primary/10 text-primary';
    }
    return 'bg-muted text-muted-foreground';
  }

  function probeCheckStateLabel(index: number) {
    return t(`project.initialization.probeCheckStates.${probeCheckState(index)}`);
  }

  const canContinueToGateway = computed(
    () =>
      status.value?.status === 'needs_gateway' &&
      probeStatus.value === 'succeeded' &&
      !environmentHasUnsavedChanges.value
  );

  async function openSshCommand() {
    sshCommandError.value = '';
    const projectId = projectStore.activeProjectId;
    if (!projectId || !isRemoteSSH.value) {
      return;
    }
    if (!validateEnvironment()) {
      return;
    }
    showSshCommandDialog.value = true;
    loadingSshCommand.value = true;
    sshCommand.value = '';
    try {
      const result = await initializationStore.prepareSSHEnvironment(
        projectId,
        initializationEnvironmentRequestFromForm(environmentForm)
      );
      if (!result.status || !result.public_key) {
        throw new Error(t('project.initialization.sshCommandFailed'));
      }
      hydrate(result.status);
      selectedStep.value = 1;
      sshCommand.value =
        environmentForm.platform === 'windows'
          ? buildWindowsSshInitializationCommand({
              host: environmentForm.host,
              port: environmentForm.port,
              username: environmentForm.username,
              workspaceRoot: environmentForm.workspaceRoot,
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

  function stepFromStatus(value?: string): number {
    if (value === 'needs_probe') {
      return 2;
    }
    if (value === 'needs_gateway' || value === READY_INITIALIZATION_STATUS) {
      return 3;
    }
    return 1;
  }

  function resetWorkspace() {
    gatewayHydrated.value = false;
    selectedStep.value = 1;
    Object.assign(environmentForm, emptyEnvironmentForm());
    assignEnvironmentFormErrors(environmentErrors, emptyEnvironmentFormErrors());
    Object.assign(gatewayForm, emptyGatewayConfigForm());
    showSshCommandDialog.value = false;
    sshCommand.value = '';
    sshCommandError.value = '';
    loadingSshCommand.value = false;
  }

  function routeProjectId(): string {
    const value = route.params.id;
    return (Array.isArray(value) ? value[0] : value)?.trim() ?? '';
  }

  function openProject(id: string) {
    const projectId = id.trim();
    if (!projectId) {
      void router.replace({ name: 'Projects' });
      return;
    }
    resetWorkspace();
    projectStore.setActiveProject(projectId);
    void loadStatus(projectId);
  }

  function selectStep(step: number) {
    if (step < 1 || step > progressStep.value) {
      return;
    }
    selectedStep.value = step;
  }

  function handleStepChange(step: number | undefined) {
    if (step !== undefined) {
      selectStep(step);
    }
  }

  async function loadStatus(id: string, force = false) {
    loadError.value = '';
    try {
      const view = force
        ? await initializationStore.fetchStatus(id)
        : await initializationStore.ensureStatus(id);
      if (id !== routeProjectId()) {
        return;
      }
      hydrate(view);
      if (view.status === READY_INITIALIZATION_STATUS) {
        await router.replace(resolveInitializationCompletionRedirect());
      }
    } catch (error: unknown) {
      if (id !== routeProjectId()) {
        return;
      }
      if (error instanceof ApiError && error.code === 'not_found') {
        await router.replace({ name: 'Projects' });
        return;
      }
      loadError.value =
        error instanceof Error ? error.message : t('project.initialization.loadFailed');
    }
  }

  function reloadStatus() {
    const projectId = routeProjectId();
    if (projectId) {
      void loadStatus(projectId, true);
    }
  }

  function hydrate(view: NonNullable<typeof status.value>) {
    if (!view) {
      return;
    }
    Object.assign(
      environmentForm,
      hydrateEnvironmentForm(view.environment, view.defaults?.local_workspace_root || '')
    );
    if (!gatewayHydrated.value && view.defaults) {
      Object.assign(gatewayForm, gatewayConfigFormFromDefaults(view.defaults));
      gatewayHydrated.value = true;
    }
    selectedStep.value = stepFromStatus(view.status);
  }

  function validateEnvironment() {
    assignEnvironmentFormErrors(
      environmentErrors,
      validateEnvironmentForm(environmentForm, {
        host: t('project.environment.validation.host'),
        port: t('project.environment.validation.port'),
        username: t('project.environment.validation.username'),
        workspaceRoot: t('project.environment.validation.workspaceRoot'),
      })
    );
    return !environmentFormHasErrors(environmentErrors);
  }

  async function saveEnvironment() {
    environmentSubmitError.value = '';
    if (!validateEnvironment()) {
      return;
    }
    const id = projectStore.activeProjectId;
    if (!id) {
      return;
    }
    activeOperation.value = 'saving';
    try {
      await executeOperation(async () => {
        const view = await initializationStore.saveEnvironment(
          id,
          initializationEnvironmentRequestFromForm(environmentForm)
        );
        hydrate(view);
        selectedStep.value = 2;
        toast.success(t('project.initialization.saved'));
      });
    } catch (error: unknown) {
      environmentSubmitError.value =
        error instanceof Error ? error.message : t('project.initialization.saveFailed');
    } finally {
      activeOperation.value = 'idle';
    }
  }

  async function probeEnvironment() {
    environmentSubmitError.value = '';
    const id = projectStore.activeProjectId;
    if (!id) {
      return;
    }
    if (environmentFormDirty(environmentForm, status.value?.environment)) {
      await saveEnvironment();
      if (environmentSubmitError.value) {
        return;
      }
    }
    activeOperation.value = 'probing';
    try {
      await executeOperation(async () => {
        const view = await initializationStore.probeEnvironment(id);
        hydrate(view);
        if (view.status === 'needs_gateway' || view.status === READY_INITIALIZATION_STATUS) {
          toast.success(t('project.initialization.probeSucceeded'));
        } else {
          toast.error(
            view.environment?.last_probe_diagnostic || t('project.initialization.probeFailed')
          );
        }
      });
    } catch (error: unknown) {
      environmentSubmitError.value =
        error instanceof Error ? error.message : t('project.initialization.probeFailed');
    } finally {
      activeOperation.value = 'idle';
    }
  }

  function replaceGatewayErrors(next: GatewayConfigFormErrors) {
    for (const field of Object.keys(gatewayErrors)) {
      delete gatewayErrors[field];
    }
    Object.assign(gatewayErrors, next);
  }

  function updateGatewayField<K extends keyof GatewayConfigForm>(
    field: K,
    value: GatewayConfigForm[K]
  ) {
    gatewayForm[field] = value;
    delete gatewayErrors[String(field)];
  }

  function handleAcmeProfileChange(value: string | number) {
    const profile = String(value);
    updateGatewayField(
      'acme_profile',
      profile === noAcmeProfileValue ? '' : (profile as GatewayAcmeProfile)
    );
  }

  function gatewayValidationMessage(field: string) {
    return t(`gateway.validation.${gatewayErrors[field]}`);
  }

  async function createGateway() {
    gatewaySubmitError.value = '';
    const errors = validateGatewayConfigForm(gatewayForm, 'initialize');
    replaceGatewayErrors(errors);
    if (Object.keys(errors).length > 0) {
      return;
    }
    const id = projectStore.activeProjectId;
    if (!id) {
      return;
    }
    activeOperation.value = 'creating';
    try {
      await executeOperation(async () => {
        const latest = await initializationStore.fetchStatus(id);
        if (latest.status !== 'needs_gateway') {
          hydrate(latest);
          if (latest.status === READY_INITIALIZATION_STATUS) {
            await router.replace(resolveInitializationCompletionRedirect());
          }
          return;
        }
        const view = await initializationStore.createGateway(
          id,
          gatewayInitializeRequestFromForm(gatewayForm)
        );
        hydrate(view);
        toast.success(t('project.initialization.created'));
        await router.replace(resolveInitializationCompletionRedirect());
      });
    } catch (error: unknown) {
      gatewaySubmitError.value =
        error instanceof Error ? error.message : t('project.initialization.createFailed');
    } finally {
      activeOperation.value = 'idle';
    }
  }

  onMounted(() => openProject(routeProjectId()));

  watch(
    () => route.params.id,
    () => {
      openProject(routeProjectId());
    }
  );
</script>
