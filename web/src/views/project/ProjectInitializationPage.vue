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
            <p
              v-else-if="isRemoteSSH && sshTestPassed"
              class="text-sm text-green-600 dark:text-green-400"
            >
              {{ t('project.initialization.testSSHPassed') }}
            </p>
            <div class="flex flex-wrap justify-end gap-2">
              <button
                v-if="isRemoteSSH"
                type="button"
                class="app-button h-9 px-4"
                :disabled="operating"
                :aria-busy="activeOperation === 'testing'"
                @click="testSSH"
              >
                <LoaderCircle
                  v-if="activeOperation === 'testing'"
                  class="size-4 animate-spin"
                  aria-hidden="true"
                />
                {{
                  activeOperation === 'testing'
                    ? t('project.initialization.testingSSH')
                    : t('project.initialization.testSSH')
                }}
              </button>
              <button
                v-if="isWindowsSSH"
                type="button"
                class="app-button h-9 px-4"
                :disabled="operating || loadingDeploymentPublicKey"
                :aria-busy="loadingDeploymentPublicKey"
                @click="openWindowsCommand"
              >
                <LoaderCircle
                  v-if="loadingDeploymentPublicKey"
                  class="size-4 animate-spin"
                  aria-hidden="true"
                />
                {{ t('project.initialization.windowsTargetCommand') }}
              </button>
              <span
                class="inline-flex"
                :title="
                  isRemoteSSH && !sshTestPassed
                    ? t('project.initialization.saveRequiresSSHTest')
                    : undefined
                "
              >
                <button
                  type="submit"
                  class="app-button-primary h-9 px-4"
                  :disabled="operating || (isRemoteSSH && !sshTestPassed)"
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
              </span>
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
                    :class="probeCheckState(index) === 'error' ? 'bg-destructive/5' : ''"
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
                      <XCircle
                        v-else-if="probeCheckState(index) === 'error'"
                        class="size-4"
                        aria-hidden="true"
                      />
                      <CircleDashed v-else class="size-4" aria-hidden="true" />
                    </div>
                    <span
                      class="min-w-0 flex-1 text-sm"
                      :class="
                        probeCheckState(index) === 'error' ? 'text-destructive' : 'text-foreground'
                      "
                    >
                      {{ item.label }}
                    </span>
                    <span class="text-xs text-muted-foreground">
                      {{ probeCheckStateLabel(index) }}
                    </span>
                  </li>
                </ul>
              </div>
            </div>

            <div
              v-if="canBootstrapHost && probeStatus !== 'succeeded'"
              class="space-y-3 rounded-lg border border-border p-4"
            >
              <div>
                <h3 class="text-sm font-medium text-foreground">
                  {{ t('project.environment.initializeTitle') }}
                </h3>
                <p class="mt-1 text-xs text-muted-foreground">
                  {{ t('project.initialization.bootstrapHint') }}
                </p>
              </div>
              <EnvironmentBootstrapFields
                :model-value="bootstrapForm"
                :errors="bootstrapErrors"
                :disabled="operating"
                id-prefix="initialization-bootstrap"
              />
              <button
                type="button"
                class="app-button h-9 px-4"
                :disabled="operating"
                :aria-busy="activeOperation === 'bootstrapping'"
                @click="bootstrapEnvironment"
              >
                <LoaderCircle
                  v-if="activeOperation === 'bootstrapping'"
                  class="size-4 animate-spin"
                  aria-hidden="true"
                />
                {{ t('project.environment.initialize') }}
              </button>
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

    <WindowsSshInitializationDialog
      v-model:open="showWindowsCommandDialog"
      :command="windowsCommand"
      :loading="loadingDeploymentPublicKey"
      :error="windowsCommandError"
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
    XCircle,
  } from '@lucide/vue';
  import { computed, onMounted, onUnmounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { onBeforeRouteUpdate, useRoute, useRouter } from 'vue-router';
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
  import WindowsSshInitializationDialog from '@/components/WindowsSshInitializationDialog.vue';
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

  const props = defineProps<{ id?: string }>();
  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const toast = useToast();
  const projectStore = useProjectStore();
  const initializationStore = useProjectInitializationStore();
  const { loading: operating, execute: executeOperation } = useStatusAsync();

  const activeOperation = ref<
    'idle' | 'testing' | 'saving' | 'probing' | 'bootstrapping' | 'creating'
  >('idle');
  const sshTestSignature = ref('');
  const loadError = ref('');
  const environmentSubmitError = ref('');
  const gatewaySubmitError = ref('');
  const gatewayHydrated = ref(false);
  const probeCheckIndex = ref(-1);
  let probeCheckTimer: ReturnType<typeof setInterval> | undefined;
  const deploymentPublicKey = ref('');
  const loadingDeploymentPublicKey = ref(false);
  const showWindowsCommandDialog = ref(false);
  const windowsCommand = ref('');
  const windowsCommandError = ref('');
  const environmentForm = reactive(emptyEnvironmentForm());
  const environmentErrors = reactive(emptyEnvironmentFormErrors());
  const bootstrapForm = reactive(emptyEnvironmentBootstrapForm());
  const bootstrapErrors = reactive(emptyEnvironmentBootstrapFormErrors());
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

  const projectId = computed(() =>
    String(props.id || route.params.id || projectStore.activeProjectId || '')
  );
  const status = computed(() => initializationStore.statusFor(projectId.value));
  const loading = computed(() => initializationStore.loading);
  const localWorkspaceRoot = computed(() => status.value?.defaults?.local_workspace_root || '');
  const isRemoteSSH = computed(() => environmentForm.targetType === 'ssh');
  const isWindowsSSH = computed(
    () => environmentForm.targetType === 'ssh' && environmentForm.platform === 'windows'
  );
  const sshTargetSignature = computed(() => {
    if (!isRemoteSSH.value) {
      return '';
    }
    return [
      environmentForm.platform,
      environmentForm.host.trim(),
      String(environmentForm.port),
      environmentForm.username.trim(),
      environmentForm.workspaceRoot.trim(),
    ].join('\0');
  });
  const sshTestPassed = computed(
    () => isRemoteSSH.value && sshTestSignature.value === sshTargetSignature.value
  );
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
  const canBootstrapHost = computed(
    () =>
      status.value?.environment?.target_type === 'ssh' &&
      status.value.environment.ssh?.platform === 'linux' &&
      !status.value.environment.ssh?.host_key_fingerprint &&
      status.value.environment.last_probe_status !== 'succeeded'
  );
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

  type ProbeCheckState = 'pending' | 'checking' | 'success' | 'error';

  function probeCheckState(index: number): ProbeCheckState {
    if (isProbing.value) {
      if (index < probeCheckIndex.value) {
        return 'success';
      }
      return index === probeCheckIndex.value ? 'checking' : 'pending';
    }
    if (probeStatus.value === 'succeeded') {
      return 'success';
    }
    if (probeStatus.value === 'failed') {
      const failedIndex = failedProbeCheckIndex();
      if (index < failedIndex) {
        return 'success';
      }
      return index === failedIndex ? 'error' : 'pending';
    }
    return 'pending';
  }

  function probeCheckIconClass(index: number) {
    const state = probeCheckState(index);
    if (state === 'success') {
      return 'bg-green-500/10 text-green-600 dark:text-green-400';
    }
    if (state === 'error') {
      return 'bg-destructive/10 text-destructive';
    }
    if (state === 'checking') {
      return 'bg-primary/10 text-primary';
    }
    return 'bg-muted text-muted-foreground';
  }

  function probeCheckStateLabel(index: number) {
    return t(`project.initialization.probeCheckStates.${probeCheckState(index)}`);
  }

  function failedProbeCheckIndex() {
    const diagnostic = probeDiagnostic.value.toLowerCase();
    const key = diagnostic.includes('compose')
      ? 'compose'
      : diagnostic.includes('wsl') || diagnostic.includes('docker desktop')
        ? 'wsl'
        : diagnostic.includes('workspace')
          ? 'workspace'
          : diagnostic.includes('docker')
            ? 'docker'
            : 'target';
    const index = probeChecks.value.findIndex((item) => item.key === key);
    return index >= 0 ? index : Math.max(0, probeCheckIndex.value);
  }

  function clearProbeCheckTimer() {
    if (probeCheckTimer) {
      clearInterval(probeCheckTimer);
      probeCheckTimer = undefined;
    }
  }

  function startProbeChecks() {
    clearProbeCheckTimer();
    probeCheckIndex.value = 0;
    probeCheckTimer = setInterval(() => {
      if (probeCheckIndex.value < probeChecks.value.length - 1) {
        probeCheckIndex.value += 1;
      }
    }, 700);
  }
  const canContinueToGateway = computed(
    () =>
      status.value?.status === 'needs_gateway' &&
      probeStatus.value === 'succeeded' &&
      !environmentHasUnsavedChanges.value
  );

  async function openWindowsCommand() {
    windowsCommandError.value = '';
    if (!projectId.value || !isWindowsSSH.value) {
      return;
    }
    if (!validateEnvironment()) {
      return;
    }
    showWindowsCommandDialog.value = true;
    loadingDeploymentPublicKey.value = true;
    try {
      if (!deploymentPublicKey.value) {
        const result = await initializationStore.getDeploymentPublicKey(projectId.value);
        deploymentPublicKey.value = result.public_key;
      }
      windowsCommand.value = buildWindowsSshInitializationCommand({
        host: environmentForm.host,
        port: environmentForm.port,
        username: environmentForm.username,
        workspaceRoot: environmentForm.workspaceRoot,
        publicKey: deploymentPublicKey.value,
      });
    } catch {
      windowsCommandError.value = t('project.initialization.windowsTargetKeyLoadFailed');
    } finally {
      loadingDeploymentPublicKey.value = false;
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
    clearProbeCheckTimer();
    probeCheckIndex.value = -1;
    gatewayHydrated.value = false;
    sshTestSignature.value = '';
    selectedStep.value = 1;
    Object.assign(environmentForm, emptyEnvironmentForm());
    assignEnvironmentFormErrors(environmentErrors, emptyEnvironmentFormErrors());
    Object.assign(bootstrapForm, emptyEnvironmentBootstrapForm());
    Object.assign(bootstrapErrors, emptyEnvironmentBootstrapFormErrors());
    Object.assign(gatewayForm, emptyGatewayConfigForm());
    deploymentPublicKey.value = '';
    windowsCommand.value = '';
    windowsCommandError.value = '';
    loadingDeploymentPublicKey.value = false;
  }

  function openProject(id: string) {
    if (!id) {
      return;
    }
    projectStore.setActiveProject(id);
    void loadStatus(id);
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

  async function loadStatus(id = projectId.value, force = false) {
    if (!id) {
      return;
    }
    loadError.value = '';
    try {
      const view = force
        ? await initializationStore.fetchStatus(id)
        : await initializationStore.ensureStatus(id);
      if (id !== projectId.value) {
        return;
      }
      hydrate(view);
      if (view.status === READY_INITIALIZATION_STATUS) {
        await router.replace(resolveInitializationCompletionRedirect());
      }
    } catch (error: unknown) {
      loadError.value =
        error instanceof Error ? error.message : t('project.initialization.loadFailed');
    }
  }

  function reloadStatus() {
    void loadStatus(projectId.value, true);
  }

  function hydrate(view: NonNullable<typeof status.value>) {
    if (!view) {
      return;
    }
    Object.assign(
      environmentForm,
      hydrateEnvironmentForm(view.environment, view.defaults?.local_workspace_root || '')
    );
    if (view.environment?.ssh?.username && !bootstrapForm.username) {
      bootstrapForm.username = view.environment.ssh.username;
    }
    if (!gatewayHydrated.value && view.defaults) {
      Object.assign(gatewayForm, gatewayConfigFormFromDefaults(view.defaults));
      gatewayHydrated.value = true;
    }
    selectedStep.value = stepFromStatus(view.status);
    if (environmentForm.targetType === 'ssh' && view.environment?.ssh) {
      sshTestSignature.value = sshTargetSignature.value;
    } else {
      sshTestSignature.value = '';
    }
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

  async function testSSH() {
    environmentSubmitError.value = '';
    if (!validateEnvironment()) {
      return;
    }
    const id = projectId.value;
    if (!id) {
      return;
    }
    activeOperation.value = 'testing';
    try {
      await executeOperation(async () => {
        await initializationStore.testEnvironment(
          id,
          initializationEnvironmentRequestFromForm(environmentForm)
        );
        sshTestSignature.value = sshTargetSignature.value;
        toast.success(t('project.initialization.testSSHPassed'));
      });
    } catch (error: unknown) {
      sshTestSignature.value = '';
      environmentSubmitError.value =
        error instanceof Error ? error.message : t('project.initialization.testSSHFailed');
    } finally {
      activeOperation.value = 'idle';
    }
  }

  async function saveEnvironment() {
    environmentSubmitError.value = '';
    if (!validateEnvironment()) {
      return;
    }
    const id = projectId.value;
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
        toast.success(t('project.initialization.saved'));
      });
    } catch (error: unknown) {
      environmentSubmitError.value =
        error instanceof Error ? error.message : t('project.initialization.saveFailed');
    } finally {
      activeOperation.value = 'idle';
    }
  }

  async function bootstrapEnvironment() {
    environmentSubmitError.value = '';
    Object.assign(
      bootstrapErrors,
      validateEnvironmentBootstrapForm(bootstrapForm, {
        username: t('project.environment.validation.bootstrapUsername'),
        password: t('project.environment.validation.bootstrapPassword'),
        privateKey: t('project.environment.validation.bootstrapPrivateKey'),
      })
    );
    if (bootstrapErrors.username || bootstrapErrors.credential) {
      return;
    }
    const id = projectId.value;
    if (!id) {
      return;
    }
    if (environmentFormDirty(environmentForm, status.value?.environment)) {
      await saveEnvironment();
      if (environmentSubmitError.value) {
        return;
      }
    }
    activeOperation.value = 'bootstrapping';
    try {
      await executeOperation(async () => {
        const view = await initializationStore.bootstrapEnvironment(
          id,
          environmentBootstrapRequestFromForm(bootstrapForm)
        );
        hydrate(view);
        toast.success(t('project.environment.initialized'));
      });
    } catch (error: unknown) {
      environmentSubmitError.value =
        error instanceof Error ? error.message : t('project.environment.initializeFailed');
    } finally {
      activeOperation.value = 'idle';
    }
  }

  async function probeEnvironment() {
    environmentSubmitError.value = '';
    const id = projectId.value;
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
    startProbeChecks();
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
      clearProbeCheckTimer();
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
    const id = projectId.value;
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

  onMounted(() => openProject(projectId.value));

  onUnmounted(clearProbeCheckTimer);

  onBeforeRouteUpdate((to) => {
    const nextId = String(to.params.id || '');
    if (nextId === projectId.value) {
      return;
    }
    resetWorkspace();
    openProject(nextId);
  });
</script>
