<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <DetailPageHeader :items="[]" :title="gateway?.name || t('gateway.detailTitle')" />
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="gateway"
          class="app-button-primary h-9 px-3"
          :disabled="operating || isDeploying"
          @click="openDeployDialog"
        >
          <Rocket class="size-4" />
          {{ t('gateway.actions.deploy') }}
        </button>
        <button
          v-if="gateway"
          class="app-button h-9 px-3"
          :disabled="!gatewayRuntimeLogTarget"
          @click="openGatewayLogs"
        >
          <ScrollText class="size-4" />
          {{ t('gateway.actions.logs') }}
        </button>
        <button
          v-if="gateway"
          class="app-button-danger h-9 px-3"
          :disabled="operating || !canStop"
          @click="openStopDialog"
        >
          <Square class="size-4" />
          {{ t('gateway.actions.stop') }}
        </button>
        <button class="app-button h-9 px-4" @click="goBack">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppLoadingState v-if="status === 'loading'" size="section" />

    <template v-else-if="gateway">
      <DetailInfoCard
        :title="t('gateway.sections.controlPlane')"
        editable
        :disabled="operating"
        @edit="openControlPlaneEditDialog"
      >
        <template #actions>
          <button class="app-button h-9 px-3" @click="goWorkload">
            <Layers class="size-4" />
            {{ t('gateway.openWorkload') }}
          </button>
        </template>
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>{{ t('gateway.fields.name') }}</dt>
            <dd class="text-foreground">{{ gateway.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('gateway.fields.code') }}</dt>
            <dd class="text-foreground">{{ gateway.code }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>
              {{ t('gateway.fields.restApiUrl') }}
            </dt>
            <dd class="min-w-0 break-all">
              <a
                :href="gateway.rest_api_url"
                target="_blank"
                rel="noopener noreferrer"
                class="app-link inline-flex items-center gap-1"
              >
                {{ gateway.rest_api_url }}
                <ExternalLink class="size-3.5 shrink-0" />
              </a>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>
              {{ t('gateway.fields.baseDomain') }}
            </dt>
            <dd class="text-foreground">{{ gateway.base_domain }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('gateway.fields.traefikComponentName') }}</dt>
            <dd class="text-foreground">{{ gateway.traefik_component_name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('gateway.fields.restReadyTimeout') }}</dt>
            <dd class="text-foreground">{{ gateway.rest_ready_timeout_seconds }}s</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.createdAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(gateway.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.updatedAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(gateway.updated_at) }}</dd>
          </div>
        </dl>
      </DetailInfoCard>

      <DetailInfoCard
        :title="t('gateway.sections.ingressDefaults')"
        editable
        :disabled="operating"
        @edit="openIngressEditDialog"
      >
        <div class="space-y-3 px-5 py-4">
          <dl class="app-detail-info-grid">
            <div class="flex gap-2">
              <dt>{{ t('gateway.fields.defaultEntrypoint') }}</dt>
              <dd>
                <AppBadge variant="pill">{{ gateway.default_entrypoint }}</AppBadge>
              </dd>
            </div>
            <div class="flex gap-2">
              <dt>{{ t('gateway.fields.tlsMode') }}</dt>
              <dd>
                <AppBadge variant="pill">{{ gateway.tls_mode }}</AppBadge>
              </dd>
            </div>
          </dl>
        </div>
      </DetailInfoCard>

      <DetailInfoCard
        :title="t('gateway.sections.routeCertificates')"
        editable
        :disabled="operating"
        @edit="openCertificateEditDialog"
      >
        <div class="space-y-3 px-5 py-4">
          <dl class="app-detail-info-grid">
            <div class="flex gap-2">
              <dt>{{ t('gateway.fields.acmeProfile') }}</dt>
              <dd>
                <AppBadge variant="pill">{{ acmeProfileLabel(gateway.acme_profile) }}</AppBadge>
              </dd>
            </div>
            <div v-if="gateway.acme_profile" class="flex gap-2">
              <dt>{{ t('gateway.fields.acmeEmail') }}</dt>
              <dd class="text-foreground">{{ gateway.acme_email }}</dd>
            </div>
            <div v-if="usesDNSProfile(gateway.acme_profile)" class="flex gap-2">
              <dt>{{ t('gateway.fields.dnsApiToken') }}</dt>
              <dd class="min-w-0 flex-1">
                <SensitiveValue
                  :value="gateway.dns_api_token"
                  :label="t('gateway.fields.dnsApiToken')"
                  :show-label="t('common.showValue')"
                  :hide-label="t('common.hideValue')"
                />
              </dd>
            </div>
          </dl>
        </div>
      </DetailInfoCard>
    </template>

    <AppDialog
      v-model:open="isControlPlaneEditDialogOpen"
      :title="t('gateway.dialog.editControlPlane')"
      width-class="w-[min(640px,calc(100vw-32px))]"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div class="space-y-1.5">
          <label class="app-field-label block" for="gateway-edit-name">
            {{ t('gateway.fields.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="gateway-edit-name"
            v-model="controlPlaneForm.name"
            type="text"
            class="app-input"
            :class="controlPlaneErrors.name ? 'app-input-error' : ''"
            :aria-invalid="controlPlaneErrors.name ? 'true' : undefined"
            @input="delete controlPlaneErrors.name"
          />
          <p v-if="controlPlaneErrors.name" class="app-field-error" role="alert">
            {{ validationMessage(controlPlaneErrors.name) }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="gateway-edit-code">
            {{ t('gateway.fields.code') }}
          </label>
          <input
            id="gateway-edit-code"
            :value="gateway?.code || ''"
            type="text"
            class="app-input"
            readonly
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="gateway-edit-traefik-component-name">
            {{ t('gateway.fields.traefikComponentName') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="gateway-edit-traefik-component-name"
            v-model="controlPlaneForm.traefik_component_name"
            type="text"
            class="app-input"
            :class="controlPlaneErrors.traefik_component_name ? 'app-input-error' : ''"
            :aria-invalid="controlPlaneErrors.traefik_component_name ? 'true' : undefined"
            @input="delete controlPlaneErrors.traefik_component_name"
          />
          <p v-if="controlPlaneErrors.traefik_component_name" class="app-field-error" role="alert">
            {{ validationMessage(controlPlaneErrors.traefik_component_name) }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="gateway-edit-rest-api-url">
            {{ t('gateway.fields.restApiUrl') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="gateway-edit-rest-api-url"
            v-model="controlPlaneForm.rest_api_url"
            type="url"
            class="app-input"
            :class="controlPlaneErrors.rest_api_url ? 'app-input-error' : ''"
            :aria-invalid="controlPlaneErrors.rest_api_url ? 'true' : undefined"
            @input="delete controlPlaneErrors.rest_api_url"
          />
          <p v-if="controlPlaneErrors.rest_api_url" class="app-field-error" role="alert">
            {{ validationMessage(controlPlaneErrors.rest_api_url) }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="gateway-edit-rest-ready-timeout">
            {{ t('gateway.fields.restReadyTimeout') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="gateway-edit-rest-ready-timeout"
            v-model="controlPlaneForm.rest_ready_timeout_seconds"
            type="number"
            min="1"
            max="300"
            class="app-input"
            :class="controlPlaneErrors.rest_ready_timeout_seconds ? 'app-input-error' : ''"
            :aria-invalid="controlPlaneErrors.rest_ready_timeout_seconds ? 'true' : undefined"
            @input="delete controlPlaneErrors.rest_ready_timeout_seconds"
          />
          <p
            v-if="controlPlaneErrors.rest_ready_timeout_seconds"
            class="app-field-error"
            role="alert"
          >
            {{ validationMessage(controlPlaneErrors.rest_ready_timeout_seconds) }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="gateway-edit-base-domain">
            {{ t('gateway.fields.baseDomain') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="gateway-edit-base-domain"
            v-model="controlPlaneForm.base_domain"
            type="text"
            class="app-input"
            :class="controlPlaneErrors.base_domain ? 'app-input-error' : ''"
            :aria-invalid="controlPlaneErrors.base_domain ? 'true' : undefined"
            @input="delete controlPlaneErrors.base_domain"
          />
          <p v-if="controlPlaneErrors.base_domain" class="app-field-error" role="alert">
            {{ validationMessage(controlPlaneErrors.base_domain) }}
          </p>
        </div>
      </div>
      <p v-if="controlPlaneSubmitError" class="app-field-error mt-3" role="alert">
        {{ controlPlaneSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          @cancel="isControlPlaneEditDialogOpen = false"
          @confirm="saveControlPlane"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isIngressEditDialogOpen"
      :title="t('gateway.dialog.editIngressDefaults')"
      width-class="w-[min(520px,calc(100vw-32px))]"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div class="space-y-1.5">
          <label class="app-field-label block" for="gateway-edit-entrypoint">
            {{ t('gateway.fields.defaultEntrypoint') }}
          </label>
          <SelectControl
            id="gateway-edit-entrypoint"
            :model-value="ingressForm.default_entrypoint"
            :options="entrypointOptions"
            :invalid="Boolean(ingressErrors.default_entrypoint)"
            @update:model-value="
              ingressForm.default_entrypoint = String($event);
              delete ingressErrors.default_entrypoint;
            "
          />
          <p v-if="ingressErrors.default_entrypoint" class="app-field-error" role="alert">
            {{ validationMessage(ingressErrors.default_entrypoint) }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="gateway-edit-tls-mode">
            {{ t('gateway.fields.tlsMode') }}
          </label>
          <SelectControl
            id="gateway-edit-tls-mode"
            :model-value="ingressForm.tls_mode"
            :options="tlsModeOptions"
            :invalid="Boolean(ingressErrors.tls_mode)"
            @update:model-value="
              ingressForm.tls_mode = String($event);
              delete ingressErrors.tls_mode;
            "
          />
          <p v-if="ingressErrors.tls_mode" class="app-field-error" role="alert">
            {{ validationMessage(ingressErrors.tls_mode) }}
          </p>
        </div>
      </div>
      <p v-if="ingressSubmitError" class="app-field-error mt-3" role="alert">
        {{ ingressSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          @cancel="isIngressEditDialogOpen = false"
          @confirm="saveIngressDefaults"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isCertificateEditDialogOpen"
      :title="t('gateway.dialog.editRouteCertificates')"
      width-class="w-[min(640px,calc(100vw-32px))]"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div class="space-y-1.5">
          <label class="app-field-label block" for="gateway-edit-acme-profile">
            {{ t('gateway.fields.acmeProfile') }}
          </label>
          <SelectControl
            id="gateway-edit-acme-profile"
            :model-value="certificateAcmeProfileValue"
            :options="acmeProfileOptions"
            :invalid="Boolean(certificateErrors.acme_profile)"
            @update:model-value="handleCertificateAcmeProfileChange"
          />
          <p v-if="certificateErrors.acme_profile" class="app-field-error" role="alert">
            {{ validationMessage(certificateErrors.acme_profile) }}
          </p>
        </div>
        <div v-if="certificateForm.acme_profile" class="space-y-1.5">
          <label class="app-field-label block" for="gateway-edit-acme-email">
            {{ t('gateway.fields.acmeEmail') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="gateway-edit-acme-email"
            v-model="certificateForm.acme_email"
            type="email"
            class="app-input"
            :class="certificateErrors.acme_email ? 'app-input-error' : ''"
            :aria-invalid="certificateErrors.acme_email ? 'true' : undefined"
            @input="delete certificateErrors.acme_email"
          />
          <p v-if="certificateErrors.acme_email" class="app-field-error" role="alert">
            {{ validationMessage(certificateErrors.acme_email) }}
          </p>
        </div>
        <div v-if="certificateUsesDNSProfile" class="space-y-1.5">
          <label class="app-field-label block" for="gateway-edit-dns-api-token">
            {{ t('gateway.fields.dnsApiToken') }}
            <span class="text-destructive">*</span>
          </label>
          <div class="relative">
            <input
              id="gateway-edit-dns-api-token"
              v-model="certificateForm.dns_api_token"
              :type="isCertificateTokenVisible ? 'text' : 'password'"
              class="app-input pr-10"
              :class="certificateErrors.dns_api_token ? 'app-input-error' : ''"
              :aria-invalid="certificateErrors.dns_api_token ? 'true' : undefined"
              @input="delete certificateErrors.dns_api_token"
            />
            <button
              type="button"
              class="absolute right-1 top-1/2 inline-flex size-9 -translate-y-1/2 items-center justify-center text-muted-foreground hover:text-foreground"
              :aria-label="
                isCertificateTokenVisible ? t('common.hideValue') : t('common.showValue')
              "
              :title="isCertificateTokenVisible ? t('common.hideValue') : t('common.showValue')"
              @click="isCertificateTokenVisible = !isCertificateTokenVisible"
            >
              <EyeOff v-if="isCertificateTokenVisible" class="size-4" />
              <Eye v-else class="size-4" />
            </button>
          </div>
          <p v-if="certificateErrors.dns_api_token" class="app-field-error" role="alert">
            {{ validationMessage(certificateErrors.dns_api_token) }}
          </p>
        </div>
      </div>
      <p v-if="certificateSubmitError" class="app-field-error mt-3" role="alert">
        {{ certificateSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          @cancel="isCertificateEditDialogOpen = false"
          @confirm="saveRouteCertificates"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeployDialogOpen"
      :title="t('gateway.deploy.dialogTitle')"
      width-class="w-[min(480px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <p class="text-sm text-muted-foreground">{{ t('gateway.deploy.description') }}</p>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.deploy.service') }}
            <span class="text-destructive">*</span>
          </label>
          <SelectControl
            v-model="deployForm.service_id"
            :options="deployServiceSelectOptions"
            :placeholder="t('gateway.deploy.selectService')"
            :invalid="Boolean(deployErrors.service_id)"
            @update:model-value="handleDeployServiceChange"
          />
          <p v-if="deployErrors.service_id" class="app-field-error" role="alert">
            {{ deployErrors.service_id }}
          </p>
        </div>
        <label class="flex items-center gap-2">
          <input v-model="deployForm.force_recreate" type="checkbox" class="app-checkbox" />
          <span class="text-sm text-foreground">{{ t('gateway.deploy.forceRecreate') }}</span>
        </label>
      </div>
      <p v-if="deploySubmitError" class="app-field-error mt-3" role="alert">
        {{ deploySubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          @cancel="isDeployDialogOpen = false"
          @confirm="handleDeployOk"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isStopDialogOpen"
      :title="t('gateway.stop.dialogTitle')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <p class="text-sm text-muted-foreground">{{ t('gateway.stop.confirm') }}</p>
        <div v-if="stoppableServices.length > 1">
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.stop.service') }}
            <span class="text-destructive">*</span>
          </label>
          <SelectControl
            v-model="stopForm.service_id"
            :options="stopServiceSelectOptions"
            :placeholder="t('gateway.stop.selectService')"
            :invalid="Boolean(stopError)"
            @update:model-value="stopError = ''"
          />
          <p v-if="stopError" class="app-field-error" role="alert">{{ stopError }}</p>
        </div>
        <label class="flex items-center gap-2">
          <input v-model="stopForm.remove_volumes" type="checkbox" class="app-checkbox" />
          <span class="text-sm text-foreground">{{ t('gateway.stop.removeVolumes') }}</span>
        </label>
      </div>
      <p v-if="stopSubmitError" class="app-field-error mt-3" role="alert">
        {{ stopSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          variant="destructive"
          @cancel="isStopDialogOpen = false"
          @confirm="handleStopOk"
        />
      </template>
    </AppDialog>

    <RuntimeContainerLogsDrawer
      v-if="runtimeLogTarget"
      v-model:open="isGatewayLogsDrawerOpen"
      :target="runtimeLogTarget"
    />
  </div>
</template>

<script setup lang="ts">
  import {
    ArrowLeft,
    ExternalLink,
    Eye,
    EyeOff,
    Layers,
    Rocket,
    ScrollText,
    Square,
  } from '@lucide/vue';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';

  import { serviceApi } from '@/api/service/service';
  import { gatewayApi } from '@/api/gateway/gateway';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import DetailPageHeader from '@/components/DetailPageHeader.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import RuntimeContainerLogsDrawer from '@/components/RuntimeContainerLogsDrawer.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import SensitiveValue from '@/components/SensitiveValue.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
  import type { RuntimeContainerLogTarget } from '@/components/runtimeContainerLogs';
  import { formatTime } from '@/utils/time';
  import {
    gatewayConfigFormFromResponse,
    type GatewayAcmeProfile,
    type GatewayConfigForm,
    type GatewayConfigFormErrors,
    validateGatewayConfigForm,
  } from './gatewayConfigForm';

  const toast = useToast();
  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const { status, execute } = useStatusAsync();
  const { status: opStatus, execute: executeOp } = useStatusAsync();

  const gateway = ref<GatewayResp | null>(null);
  const services = ref<ServiceResp[]>([]);
  const isControlPlaneEditDialogOpen = ref(false);
  const controlPlaneForm = reactive<
    Pick<
      GatewayConfigForm,
      | 'name'
      | 'traefik_component_name'
      | 'rest_api_url'
      | 'rest_ready_timeout_seconds'
      | 'base_domain'
    >
  >({
    name: '',
    traefik_component_name: '',
    rest_api_url: '',
    rest_ready_timeout_seconds: '',
    base_domain: '',
  });
  const controlPlaneErrors = reactive<GatewayConfigFormErrors>({});
  const controlPlaneSubmitError = ref('');
  const isIngressEditDialogOpen = ref(false);
  const ingressForm = reactive<Pick<GatewayConfigForm, 'default_entrypoint' | 'tls_mode'>>({
    default_entrypoint: '',
    tls_mode: '',
  });
  const ingressErrors = reactive<GatewayConfigFormErrors>({});
  const ingressSubmitError = ref('');
  const isCertificateEditDialogOpen = ref(false);
  const certificateForm = reactive<
    Pick<GatewayConfigForm, 'acme_profile' | 'acme_email' | 'dns_api_token'>
  >({
    acme_profile: '',
    acme_email: '',
    dns_api_token: '',
  });
  const certificateErrors = reactive<GatewayConfigFormErrors>({});
  const certificateSubmitError = ref('');
  const isCertificateTokenVisible = ref(false);
  const isDeployDialogOpen = ref(false);
  const deployErrors = reactive({ service_id: '' });
  const deploySubmitError = ref('');
  const deployForm = reactive({
    service_id: '',
    force_recreate: false,
  });
  const runtimeLogTarget = ref<RuntimeContainerLogTarget>();
  const isGatewayLogsDrawerOpen = computed({
    get: () => runtimeLogTarget.value !== undefined,
    set: (open) => {
      if (!open) runtimeLogTarget.value = undefined;
    },
  });
  const gatewayRuntimeLogTarget = computed(() => {
    const current = gateway.value;
    if (
      !current ||
      !current.default_service_id ||
      !current.default_service_instance_key ||
      !current.traefik_component_name
    ) {
      return undefined;
    }
    return runtimeTargetForService(
      current,
      current.default_service_id,
      current.default_service_instance_key
    );
  });

  const isStopDialogOpen = ref(false);
  const stopError = ref('');
  const stopSubmitError = ref('');
  const stopForm = reactive({
    service_id: '',
    remove_volumes: false,
  });
  const gatewayId = () => String(route.params.id || '');
  const operating = computed(() => opStatus.value === 'loading');
  const isDeploying = computed(() => services.value.some((item) => item.active_deployment));
  const stoppableServices = computed(() =>
    services.value.filter(
      (item) => !item.active_deployment && (item.status === 'running' || item.status === 'faulted')
    )
  );
  const canStop = computed(() => stoppableServices.value.length > 0);
  const deployServiceSelectOptions = computed(() =>
    services.value.map((item) => ({ value: item.id, label: serviceOptionLabel(item) }))
  );

  const stopServiceSelectOptions = computed(() =>
    stoppableServices.value.map((item) => ({
      value: item.id,
      label: serviceOptionLabel(item),
    }))
  );
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
  const certificateAcmeProfileValue = computed(
    () => certificateForm.acme_profile || noAcmeProfileValue
  );
  const certificateUsesDNSProfile = computed(() => usesDNSProfile(certificateForm.acme_profile));

  function serviceOptionLabel(item: ServiceResp) {
    const instance = item.instance_key || 'default';
    return `${instance} (${item.status})`;
  }

  function replaceErrors(target: GatewayConfigFormErrors, next: GatewayConfigFormErrors) {
    for (const field of Object.keys(target)) delete target[field];
    Object.assign(target, next);
  }

  function keepSectionErrors(
    target: GatewayConfigFormErrors,
    errors: GatewayConfigFormErrors,
    fields: string[]
  ) {
    const next: GatewayConfigFormErrors = {};
    for (const field of fields) {
      if (errors[field]) next[field] = errors[field];
    }
    replaceErrors(target, next);
  }

  function validationMessage(error: string) {
    return t(`gateway.validation.${error}`);
  }

  function openControlPlaneEditDialog() {
    const current = gateway.value;
    if (!current) return;
    const form = gatewayConfigFormFromResponse(current);
    Object.assign(controlPlaneForm, {
      name: form.name,
      traefik_component_name: form.traefik_component_name,
      rest_api_url: form.rest_api_url,
      rest_ready_timeout_seconds: form.rest_ready_timeout_seconds,
      base_domain: form.base_domain,
    });
    replaceErrors(controlPlaneErrors, {});
    controlPlaneSubmitError.value = '';
    isControlPlaneEditDialogOpen.value = true;
  }

  async function saveControlPlane() {
    const current = gateway.value;
    if (!current) return;
    controlPlaneSubmitError.value = '';
    const form = gatewayConfigFormFromResponse(current);
    Object.assign(form, controlPlaneForm);
    const errors = validateGatewayConfigForm(form, 'edit');
    keepSectionErrors(controlPlaneErrors, errors, [
      'name',
      'traefik_component_name',
      'rest_api_url',
      'rest_ready_timeout_seconds',
      'base_domain',
    ]);
    if (Object.keys(controlPlaneErrors).length > 0) return;
    try {
      await executeOp(async () => {
        gateway.value = await gatewayApi.update(current.id, {
          name: controlPlaneForm.name.trim(),
          traefik_component_name: controlPlaneForm.traefik_component_name.trim(),
          rest_api_url: controlPlaneForm.rest_api_url.trim(),
          rest_ready_timeout_seconds: Number(controlPlaneForm.rest_ready_timeout_seconds),
          base_domain: controlPlaneForm.base_domain.trim(),
        });
        isControlPlaneEditDialogOpen.value = false;
        toast.success(t('gateway.toast.saveSuccess'));
      });
    } catch (error) {
      controlPlaneSubmitError.value =
        error instanceof Error ? error.message : t('gateway.toast.saveFailed');
    }
  }

  function openIngressEditDialog() {
    const current = gateway.value;
    if (!current) return;
    Object.assign(ingressForm, {
      default_entrypoint: current.default_entrypoint,
      tls_mode: current.tls_mode,
    });
    replaceErrors(ingressErrors, {});
    ingressSubmitError.value = '';
    isIngressEditDialogOpen.value = true;
  }

  async function saveIngressDefaults() {
    const current = gateway.value;
    if (!current) return;
    ingressSubmitError.value = '';
    const form = gatewayConfigFormFromResponse(current);
    Object.assign(form, ingressForm);
    const errors = validateGatewayConfigForm(form, 'edit');
    keepSectionErrors(ingressErrors, errors, ['default_entrypoint', 'tls_mode']);
    if (Object.keys(ingressErrors).length > 0) return;
    try {
      await executeOp(async () => {
        gateway.value = await gatewayApi.update(current.id, {
          default_entrypoint: ingressForm.default_entrypoint,
          tls_mode: ingressForm.tls_mode,
        });
        isIngressEditDialogOpen.value = false;
        toast.success(t('gateway.toast.saveSuccess'));
      });
    } catch (error) {
      ingressSubmitError.value =
        error instanceof Error ? error.message : t('gateway.toast.saveFailed');
    }
  }

  function openCertificateEditDialog() {
    const current = gateway.value;
    if (!current) return;
    Object.assign(certificateForm, {
      acme_profile: current.acme_profile as GatewayAcmeProfile,
      acme_email: current.acme_email,
      dns_api_token: current.dns_api_token,
    });
    replaceErrors(certificateErrors, {});
    certificateSubmitError.value = '';
    isCertificateTokenVisible.value = false;
    isCertificateEditDialogOpen.value = true;
  }

  function handleCertificateAcmeProfileChange(value: string | number) {
    const profile = String(value);
    certificateForm.acme_profile =
      profile === noAcmeProfileValue ? '' : (profile as GatewayAcmeProfile);
    delete certificateErrors.acme_profile;
  }

  async function saveRouteCertificates() {
    const current = gateway.value;
    if (!current) return;
    certificateSubmitError.value = '';
    const form = gatewayConfigFormFromResponse(current);
    Object.assign(form, certificateForm);
    const errors = validateGatewayConfigForm(form, 'edit');
    if (errors.tls_mode) errors.acme_profile = errors.tls_mode;
    keepSectionErrors(certificateErrors, errors, ['acme_profile', 'acme_email', 'dns_api_token']);
    if (Object.keys(certificateErrors).length > 0) return;
    try {
      await executeOp(async () => {
        gateway.value = await gatewayApi.update(current.id, {
          acme_profile: certificateForm.acme_profile,
          acme_email: certificateForm.acme_profile ? certificateForm.acme_email.trim() : '',
          dns_api_token: usesDNSProfile(certificateForm.acme_profile)
            ? certificateForm.dns_api_token.trim()
            : '',
        });
        isCertificateEditDialogOpen.value = false;
        toast.success(t('gateway.toast.saveSuccess'));
      });
    } catch (error) {
      certificateSubmitError.value =
        error instanceof Error ? error.message : t('gateway.toast.saveFailed');
    }
  }

  async function loadGateway() {
    const id = gatewayId();
    if (!id) {
      return;
    }
    try {
      await execute(async () => {
        gateway.value = await gatewayApi.get(id);
      });
      await loadRuntimeContext();
    } catch {
      toast.error(t('gateway.toast.loadDetailFailed'));
    }
  }

  async function loadRuntimeContext() {
    const id = gateway.value?.id;
    if (!id) {
      services.value = [];
      return;
    }
    try {
      const resp = await applicationApi.listServices(id);
      services.value = resp.items ?? [];
    } catch {
      services.value = [];
    }
  }

  async function openDeployDialog() {
    const current = gateway.value;
    if (!current) {
      return;
    }
    Object.assign(deployErrors, { service_id: '' });
    deploySubmitError.value = '';
    deployForm.force_recreate = false;
    if (services.value.length === 0) {
      toast.error(t('gateway.toast.noService'));
      return;
    }
    const serviceId =
      services.value.find((item) => item.id === current.default_service_id)?.id ||
      services.value[0].id;
    const selectedService = services.value.find((item) => item.id === serviceId);
    if (!selectedService) {
      return;
    }
    deployForm.service_id = selectedService.id;
    isDeployDialogOpen.value = true;
  }

  function handleDeployServiceChange(value: string | number) {
    const serviceId = String(value);
    deployForm.service_id = serviceId;
    deployErrors.service_id = '';
  }

  async function handleDeployOk() {
    const current = gateway.value;
    deploySubmitError.value = '';
    if (!current) {
      return;
    }
    if (!deployForm.service_id) {
      deployErrors.service_id = t('gateway.toast.deployServiceRequired');
      return;
    }
    deployErrors.service_id = '';
    try {
      await executeOp(async () => {
        const selectedService = services.value.find((item) => item.id === deployForm.service_id);
        if (!selectedService) {
          throw new Error(t('gateway.toast.deployServiceRequired'));
        }
        const result = await serviceApi.deploy(selectedService.id, {
          force_recreate: deployForm.force_recreate,
        });
        for (const warning of result.warnings) toast.error(warning);
        toast.success(t('gateway.toast.deployQueued'));
        isDeployDialogOpen.value = false;
        runtimeLogTarget.value = runtimeTargetForService(
          current,
          selectedService.id,
          selectedService.instance_key
        );
      });
    } catch (err: unknown) {
      deploySubmitError.value =
        err instanceof Error ? err.message : t('gateway.toast.deployFailed');
    }
  }

  function openGatewayLogs() {
    runtimeLogTarget.value = gatewayRuntimeLogTarget.value;
  }

  function runtimeTargetForService(
    current: GatewayResp,
    serviceId: string,
    instanceKey: string
  ): RuntimeContainerLogTarget {
    return {
      applicationId: current.id,
      serviceId,
      component: current.traefik_component_name,
      title: t('service.logs.titleWithComponent', {
        app: current.name,
        instance: instanceKey,
        component: current.traefik_component_name,
      }),
    };
  }

  function openStopDialog() {
    if (!canStop.value) {
      return;
    }
    stopError.value = '';
    stopSubmitError.value = '';
    stopForm.remove_volumes = false;
    stopForm.service_id = stoppableServices.value[0]?.id || '';
    isStopDialogOpen.value = true;
  }

  async function handleStopOk() {
    const current = gateway.value;
    stopSubmitError.value = '';
    if (!current) {
      return;
    }
    const targetId =
      stoppableServices.value.length === 1 ? stoppableServices.value[0].id : stopForm.service_id;
    if (!targetId) {
      stopError.value = t('gateway.toast.serviceRequired');
      return;
    }
    stopError.value = '';
    try {
      await executeOp(async () => {
        const result = await applicationApi.stop(current.id, {
          service_id: targetId,
          remove_volumes: stopForm.remove_volumes,
        });
        toast.success(t('gateway.toast.stopQueued'));
        isStopDialogOpen.value = false;
        if (result.deployment_id) {
          router.push(`/deployment/${result.deployment_id}`);
          return;
        }
        await loadRuntimeContext();
      });
    } catch (err: unknown) {
      stopSubmitError.value = err instanceof Error ? err.message : t('gateway.toast.stopFailed');
    }
  }

  function goWorkload() {
    if (!gateway.value) {
      return;
    }
    router.push(`/application/${gateway.value.id}`);
  }

  function goBack() {
    router.push('/gateways');
  }

  function usesDNSProfile(profile: string) {
    return profile === 'dns' || profile === 'http-dns';
  }

  function acmeProfileLabel(profile: string) {
    switch (profile) {
      case 'http':
        return t('gateway.acmeProfiles.http');
      case 'dns':
        return t('gateway.acmeProfiles.dns');
      case 'http-dns':
        return t('gateway.acmeProfiles.httpDns');
      case 'base':
      default:
        return t('gateway.acmeProfiles.none');
    }
  }

  watch(
    () => route.params.id,
    () => {
      loadGateway();
    }
  );

  onMounted(loadGateway);
</script>
