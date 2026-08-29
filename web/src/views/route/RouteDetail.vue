<template>
  <div class="flex flex-col gap-4">
    <!-- Header -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <DetailPageHeader :items="[]" :title="routeData?.name ?? t('route.detailTitle')" />
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="routeData && !routeEnabled"
          class="app-button-primary h-9 px-3"
          :disabled="operating || isSyncDialogOpen"
          @click="handleEnable"
        >
          <Play class="size-4" />
          {{ t('route.status.enabled') }}
        </button>
        <button
          v-else-if="routeData"
          class="app-button-warning h-9 px-3"
          :disabled="operating || isSyncDialogOpen"
          @click="handleDisable"
        >
          <PowerOff class="size-4" />
          {{ t('route.status.disabled') }}
        </button>
        <button
          v-if="routeData"
          class="h-9 px-3"
          :class="hasPendingEnabledChange ? 'app-button-warning' : 'app-button'"
          :disabled="operating || isSyncDialogOpen"
          @click="openSyncModal"
        >
          <RefreshCw class="size-4" />
          {{ hasPendingEnabledChange ? t('route.syncPending', { count: 1 }) : t('route.syncAll') }}
        </button>
        <button
          v-if="gatewayRuntimeLogTarget"
          class="app-button h-9 px-3"
          :disabled="operating"
          @click="openGatewayLogs"
        >
          <ScrollText class="size-4" />
          {{ t('route.actions.logs') }}
        </button>
        <button
          v-if="routeData"
          class="app-button-danger h-9 px-3"
          :disabled="operating"
          @click="openDeleteModal"
        >
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/routes')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <!-- Loading State -->
    <AppLoadingState v-if="basicInfoLoading" size="section" />

    <!-- Content -->
    <template v-else-if="routeData">
      <!-- Basic Info Card -->
      <DetailInfoCard
        :title="t('route.basicInfo')"
        :editable="Boolean(routeData)"
        :disabled="operating"
        @edit="openEditModal"
      >
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>{{ t('route.fields.name') }}</dt>
            <dd class="text-foreground">{{ routeData.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('route.fields.domain') }}</dt>
            <dd>
              <a
                v-if="routeEnabled && routeData.protocol === 'http'"
                :href="`${routeData.https_enabled ? 'https' : 'http'}://${routeData.domain}`"
                target="_blank"
                class="app-link inline-flex items-center gap-1"
              >
                {{ routeData.domain }}
                <ExternalLink class="size-3" />
              </a>
              <span v-else class="text-foreground">{{ routeData.domain }}</span>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('route.fields.protocol') }}</dt>
            <dd class="text-foreground">{{ routeData.protocol.toUpperCase() }}</dd>
          </div>
          <div v-if="routeData.protocol === 'http'" class="flex gap-2">
            <dt>{{ t('route.fields.pathPrefix') }}</dt>
            <dd class="text-foreground">{{ routeData.path_prefix }}</dd>
          </div>
          <div v-if="routeData.protocol === 'tcp'" class="flex gap-2">
            <dt>{{ t('route.fields.listenPort') }}</dt>
            <dd class="text-foreground">{{ routeData.listen_port }}</dd>
          </div>
          <div v-if="routeData.target_url" class="flex gap-2">
            <dt>{{ t('route.fields.targetUrl') }}</dt>
            <dd class="text-foreground">{{ routeData.target_url }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.status') }}</dt>
            <dd>
              <AppBadge variant="status" :tone="routeEnabled ? 'success' : 'default'">
                {{ routeEnabled ? t('route.status.enabled') : t('route.status.disabled') }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.createdAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(routeData.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.updatedAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(routeData.updated_at) }}</dd>
          </div>
        </dl>
      </DetailInfoCard>

      <!-- HTTPS Config Card -->
      <DetailInfoCard v-if="routeData.protocol === 'http'" :title="t('route.httpsConfig')">
        <div class="space-y-5 p-5 sm:p-6">
          <div
            class="flex flex-wrap items-center justify-between gap-4 rounded-md border border-border bg-muted/20 px-4 py-3"
          >
            <div class="flex min-w-0 items-center gap-3">
              <span
                class="flex size-9 shrink-0 items-center justify-center rounded-md"
                :class="
                  routeData.https_enabled
                    ? 'bg-primary/10 text-primary'
                    : 'bg-muted text-muted-foreground'
                "
              >
                <ShieldCheck v-if="routeData.https_enabled" class="size-5" />
                <ShieldOff v-else class="size-5" />
              </span>
              <div>
                <p class="text-sm font-medium text-foreground">{{ t('route.currentStatus') }}</p>
                <p class="text-sm text-muted-foreground">
                  {{ routeData.https_enabled ? t('route.httpsEnabled') : t('route.httpsDisabled') }}
                </p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <AppBadge variant="status" :tone="routeData.https_enabled ? 'info' : 'default'">
                {{ routeData.https_enabled ? 'HTTPS' : 'HTTP' }}
              </AppBadge>
              <button
                v-if="routeData.https_enabled"
                class="app-button-danger h-9 px-3"
                :disabled="operating"
                @click="handleDisableHttps"
              >
                <X class="size-4" />
                {{ t('route.disableHttps') }}
              </button>
            </div>
          </div>

          <div
            v-if="!routeData.https_enabled"
            class="grid gap-3"
            :class="canUseLetsencrypt ? 'lg:grid-cols-3' : 'md:grid-cols-2'"
          >
            <button
              v-if="canUseLetsencrypt"
              class="app-action-item flex min-h-28 items-start gap-3 p-4"
              :disabled="operating"
              @click="openLetsEncryptDialog"
            >
              <span
                class="flex size-9 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary"
              >
                <ShieldCheck class="size-5" />
              </span>
              <span class="min-w-0 flex-1 text-left">
                <span class="block font-medium text-foreground">
                  {{ t('route.enableLetsencrypt') }}
                </span>
                <span class="mt-1 block text-muted-foreground">
                  {{ t('route.letsencryptHint') }}
                </span>
              </span>
              <ChevronRight class="mt-1 size-4 shrink-0 text-muted-foreground" />
            </button>
            <button
              class="app-action-item flex min-h-28 items-start gap-3 p-4"
              :disabled="operating"
              @click="handleEnableMkcert"
            >
              <span
                class="flex size-9 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary"
              >
                <Key class="size-5" />
              </span>
              <span class="min-w-0 flex-1 text-left">
                <span class="block font-medium text-foreground">{{ t('route.enableMkcert') }}</span>
                <span class="mt-1 block text-muted-foreground">{{ t('route.mkcertHint') }}</span>
              </span>
              <ChevronRight class="mt-1 size-4 shrink-0 text-muted-foreground" />
            </button>
            <div
              class="flex min-h-28 flex-col justify-between gap-3 rounded-md border border-border bg-muted/20 p-4"
            >
              <div class="flex items-start gap-3">
                <span
                  class="flex size-9 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary"
                >
                  <Upload class="size-5" />
                </span>
                <div class="min-w-0">
                  <p class="font-medium text-foreground">{{ t('route.uploadCustomCert') }}</p>
                  <p class="mt-1 text-sm text-muted-foreground">
                    {{ t('route.uploadCustomCertHint') }}
                  </p>
                </div>
              </div>
              <div
                class="flex flex-col items-start justify-between gap-3 sm:flex-row sm:items-center"
              >
                <p class="min-w-0 text-xs text-muted-foreground">{{ t('route.pemFileHint') }}</p>
                <button
                  type="button"
                  class="app-button h-9 shrink-0 px-3"
                  :disabled="operating"
                  @click="selectCertFile"
                >
                  <Upload class="size-4" />
                  {{ t('route.selectPemFile') }}
                </button>
              </div>
              <input
                ref="certFileInput"
                type="file"
                accept=".pem"
                class="hidden"
                @change="handleCertUpload"
              />
            </div>
          </div>
        </div>
      </DetailInfoCard>
      <p v-if="editSubmitError" class="app-field-error mt-3" role="alert">
        {{ editSubmitError }}
      </p>
    </template>

    <AppDialog v-model:open="isEditDialogOpen" :title="t('route.editRoute')">
      <div class="space-y-4">
        <div class="grid gap-4 sm:grid-cols-2">
          <div class="space-y-1.5">
            <label class="app-field-label block">{{ t('route.fields.protocol') }}</label>
            <SelectControl
              :model-value="form.protocol"
              :options="protocolOptions"
              @update:model-value="updateProtocol"
            />
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">
              {{ t('route.fields.name') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="form.name"
              type="text"
              class="app-input"
              :class="errors.name ? 'app-input-error' : ''"
              :placeholder="t('route.hints.name')"
              :aria-invalid="errors.name ? 'true' : undefined"
              @input="errors.name = ''"
            />
            <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
          </div>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('route.fields.domain') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.domain"
            type="text"
            class="app-input"
            :class="errors.domain ? 'app-input-error' : ''"
            placeholder="example.com"
            :aria-invalid="errors.domain ? 'true' : undefined"
            @input="errors.domain = ''"
          />
          <p v-if="errors.domain" class="app-field-error text-xs">{{ errors.domain }}</p>
        </div>
        <div v-if="form.protocol === 'http'" class="space-y-1.5">
          <label class="app-field-label block">{{ t('route.fields.pathPrefix') }}</label>
          <input
            v-model="form.path_prefix"
            type="text"
            class="app-input"
            :placeholder="t('route.hints.pathPrefix')"
          />
        </div>
        <template v-if="form.protocol === 'http'">
          <RouteManagedTargetSelect
            v-if="!form.custom_target"
            protocol="http"
            :services="targetServices"
            :service-id="form.service_id"
            :component-name="form.component_name"
            :endpoint-protocol="form.endpoint_protocol"
            :endpoint-container-port="form.endpoint_container_port"
            :service-error="errors.service_id"
            :component-error="errors.component_name"
            :endpoint-error="errors.endpoint_protocol || errors.endpoint_container_port"
            @update:service-id="form.service_id = $event"
            @update:component-name="form.component_name = $event"
            @update:endpoint-protocol="
              form.endpoint_protocol = $event;
              errors.endpoint_protocol = '';
            "
            @update:endpoint-container-port="
              form.endpoint_container_port = $event;
              errors.endpoint_container_port = '';
            "
          />
          <label class="flex cursor-pointer items-center gap-3">
            <SwitchRoot
              :model-value="form.custom_target"
              class="app-switch-root"
              @update:model-value="setCustomTarget($event)"
            >
              <SwitchThumb class="app-switch-thumb" />
            </SwitchRoot>
            <span class="text-sm text-foreground">{{ t('route.advancedCustomTarget') }}</span>
          </label>
        </template>
        <div v-if="form.protocol === 'http' && form.custom_target" class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('route.fields.targetUrl') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.target_url"
            type="text"
            class="app-input"
            :class="errors.target_url ? 'app-input-error' : ''"
            :placeholder="t('route.hints.targetUrl')"
            :aria-invalid="errors.target_url ? 'true' : undefined"
            @input="errors.target_url = ''"
          />
          <p v-if="errors.target_url" class="app-field-error text-xs">{{ errors.target_url }}</p>
        </div>
        <template v-if="form.protocol === 'tcp'">
          <RouteManagedTargetSelect
            protocol="tcp"
            :services="targetServices"
            :service-id="form.service_id"
            :component-name="form.component_name"
            :endpoint-protocol="form.endpoint_protocol"
            :endpoint-container-port="form.endpoint_container_port"
            :listen-port="form.listen_port"
            :service-error="errors.service_id"
            :component-error="errors.component_name"
            :endpoint-error="errors.endpoint_protocol || errors.endpoint_container_port"
            :listen-port-error="errors.listen_port"
            @update:service-id="form.service_id = $event"
            @update:component-name="form.component_name = $event"
            @update:endpoint-protocol="
              form.endpoint_protocol = $event;
              errors.endpoint_protocol = '';
            "
            @update:endpoint-container-port="
              form.endpoint_container_port = $event;
              errors.endpoint_container_port = '';
            "
            @update:listen-port="
              form.listen_port = $event;
              errors.listen_port = '';
            "
          />
        </template>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          @cancel="isEditDialogOpen = false"
          @confirm="handleSave"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('route.deleteRoute')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('route.deleteConfirm', { domain: routeData?.domain }) }}
      </p>
      <p v-if="deleteSubmitError" class="app-field-error mt-3" role="alert">
        {{ deleteSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          variant="destructive"
          @cancel="isDeleteDialogOpen = false"
          @confirm="handleDelete"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isLetsEncryptDialogOpen"
      :title="t('route.chooseLetsencryptChallenge')"
      width-class="w-[min(460px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <div>
          <p class="app-field-hint mb-2">{{ t('route.letsencryptChallengeHint') }}</p>
          <SelectControl
            id="route-letsencrypt-challenge"
            :model-value="letsEncryptChallenge"
            :options="letsEncryptChallengeOptions"
            :disabled="!hasAvailableLetsEncryptChallenge"
            :invalid="Boolean(letsEncryptChallengeError)"
            @update:model-value="updateLetsEncryptChallenge"
          />
          <p v-if="letsEncryptChallengeError" class="app-field-error mt-1" role="alert">
            {{ letsEncryptChallengeError }}
          </p>
        </div>
        <p v-if="routeData?.acme_challenge_hint" class="text-sm text-muted-foreground">
          {{ routeData.acme_challenge_hint }}
        </p>
        <p v-if="letsEncryptSubmitError" class="app-field-error" role="alert">
          {{ letsEncryptSubmitError }}
        </p>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-disabled="!hasAvailableLetsEncryptChallenge"
          @cancel="closeLetsEncryptDialog"
          @confirm="handleEnableLetsencrypt"
        />
      </template>
    </AppDialog>

    <RuntimeContainerLogsDrawer
      v-if="gatewayRuntimeLogTarget"
      v-model:open="isGatewayLogsDrawerOpen"
      :target="gatewayRuntimeLogTarget"
    />

    <RouteSyncDialog
      v-model:open="isSyncDialogOpen"
      :changes="syncChanges"
      @synced="handleSyncComplete"
    />
  </div>
</template>

<script setup lang="ts">
  import {
    ArrowLeft,
    ChevronRight,
    ExternalLink,
    Key,
    Play,
    PowerOff,
    RefreshCw,
    ScrollText,
    ShieldCheck,
    ShieldOff,
    Trash2,
    Upload,
    X,
  } from '@lucide/vue';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { SwitchRoot, SwitchThumb } from 'reka-ui';
  import { useRoute, useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import { gatewayApi } from '@/api/gateway/gateway';
  import type { RouteResp } from '@/gen/proto/orbit/v1/route/route';
  import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';
  import { routeApi } from '@/api/route/route';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import DetailPageHeader from '@/components/DetailPageHeader.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import RuntimeContainerLogsDrawer from '@/components/RuntimeContainerLogsDrawer.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import RouteManagedTargetSelect from '@/components/RouteManagedTargetSelect.vue';
  import RouteSyncDialog from '@/components/RouteSyncDialog.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useRouteTargetServices } from '@/composables/useRouteTargetServices';
  import { useProjectStore } from '@/stores/project';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { formatTime } from '@/utils/time';
  import type { RuntimeContainerLogTarget } from '@/components/runtimeContainerLogs';

  const currentRoute = useRoute();
  const router = useRouter();
  const routeId = currentRoute.params.id as string;
  const toast = useToast();
  const projectStore = useProjectStore();
  const { t } = useI18n();
  const targetUrlPattern = /^https?:\/\/[a-zA-Z0-9.-]+(?::\d+)?$/;

  const { loading: basicInfoLoading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { services: targetServices, load: loadTargetServices } = useRouteTargetServices();
  const protocolOptions = [
    { value: 'http', label: 'HTTP' },
    { value: 'tcp', label: 'TCP' },
  ];

  const routeData = ref<RouteResp>();
  const isEditDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isLetsEncryptDialogOpen = ref(false);
  const isSyncDialogOpen = ref(false);
  const pendingEnabled = ref<boolean>();
  const letsEncryptChallenge = ref<'http' | 'dns'>('http');
  const letsEncryptChallengeError = ref('');
  const letsEncryptSubmitError = ref('');
  const certFileInput = ref<HTMLInputElement>();
  const gatewayForLogs = ref<GatewayResp>();
  const isGatewayLogsDrawerOpen = ref(false);

  const form = reactive({
    name: '',
    protocol: 'http',
    domain: '',
    path_prefix: '/',
    target_url: '',
    custom_target: false,
    listen_port: undefined as number | undefined,
    service_id: '',
    component_name: '',
    endpoint_protocol: '',
    endpoint_container_port: undefined as number | undefined,
  });
  const errors = reactive({
    name: '',
    domain: '',
    target_url: '',
    listen_port: '',
    service_id: '',
    component_name: '',
    endpoint_protocol: '',
    endpoint_container_port: '',
  });
  const editSubmitError = ref('');
  const deleteSubmitError = ref('');

  const routeEnabled = computed(() => pendingEnabled.value ?? routeData.value?.enabled ?? false);
  const hasPendingEnabledChange = computed(
    () => routeData.value !== undefined && pendingEnabled.value !== undefined
  );
  const syncChanges = computed(() => {
    if (!routeData.value || pendingEnabled.value === undefined) {
      return [];
    }
    return [{ route_id: routeData.value.id, enabled: pendingEnabled.value }];
  });

  const canUseLetsencrypt = computed(() => {
    if (!routeData.value) {
      return false;
    }
    const d = routeData.value.domain;
    return (
      d !== 'localhost' &&
      !d.endsWith('.localhost') &&
      !d.endsWith('.lvh.me') &&
      !/^\d+\.\d+\.\d+\.\d+$/.test(d)
    );
  });
  const letsEncryptChallengeOptions = computed(() => [
    {
      value: 'http',
      label: t('route.letsencryptHttpChallenge'),
      disabled: !routeData.value?.http01_available,
    },
    {
      value: 'dns',
      label: t('route.letsencryptDnsChallenge'),
      disabled: !routeData.value?.dns01_available,
    },
  ]);
  const hasAvailableLetsEncryptChallenge = computed(() =>
    Boolean(routeData.value?.http01_available || routeData.value?.dns01_available)
  );
  const gatewayRuntimeLogTarget = computed<RuntimeContainerLogTarget | undefined>(() => {
    const current = gatewayForLogs.value;
    if (
      !current ||
      !current.default_service_id ||
      !current.default_service_instance_key ||
      !current.traefik_component_name
    ) {
      return undefined;
    }
    return {
      applicationId: current.id,
      serviceId: current.default_service_id,
      component: current.traefik_component_name,
      title: t('service.logs.titleWithComponent', {
        app: current.name,
        instance: current.default_service_instance_key,
        component: current.traefik_component_name,
      }),
    };
  });

  async function fetchRoute() {
    try {
      await execute(async () => {
        const data = await routeApi.get(routeId);
        routeData.value = data;
        Object.assign(form, {
          name: data.name,
          protocol: data.protocol,
          domain: data.domain,
          path_prefix: data.path_prefix,
          target_url: data.target_url,
          custom_target: data.protocol === 'http' && !data.service_id,
          listen_port: data.listen_port,
          service_id: data.service_id ?? '',
          component_name: data.component_name ?? '',
          endpoint_protocol: data.endpoint_protocol ?? '',
          endpoint_container_port: data.endpoint_container_port,
        });
        await loadGatewayForLogs(data.gateway_application_id);
      });
    } catch {
      toast.error(t('route.toast.loadDetailFailed'));
      router.push('/routes');
    }
  }

  async function loadGatewayForLogs(applicationId: string) {
    gatewayForLogs.value = undefined;
    if (!applicationId) return;
    try {
      gatewayForLogs.value = await gatewayApi.get(applicationId);
    } catch {
      // Route detail remains available when its Gateway runtime is unavailable.
    }
  }

  async function openEditModal() {
    if (!routeData.value) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('route.toast.selectProjectRequired'));
      return;
    }
    try {
      await loadTargetServices(projectId);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.loadDetailFailed'));
      return;
    }
    Object.assign(form, {
      name: routeData.value.name,
      protocol: routeData.value.protocol,
      domain: routeData.value.domain,
      path_prefix: routeData.value.path_prefix,
      target_url: routeData.value.target_url,
      custom_target: routeData.value.protocol === 'http' && !routeData.value.service_id,
      listen_port: routeData.value.listen_port,
      service_id: routeData.value.service_id ?? '',
      component_name: routeData.value.component_name ?? '',
      endpoint_protocol: routeData.value.endpoint_protocol ?? '',
      endpoint_container_port: routeData.value.endpoint_container_port,
    });
    Object.assign(errors, {
      name: '',
      domain: '',
      target_url: '',
      listen_port: '',
      service_id: '',
      component_name: '',
      endpoint_protocol: '',
      endpoint_container_port: '',
    });
    editSubmitError.value = '';
    isEditDialogOpen.value = true;
  }

  function openDeleteModal() {
    deleteSubmitError.value = '';
    isDeleteDialogOpen.value = true;
  }

  async function handleSave() {
    editSubmitError.value = '';
    errors.name = /^[a-z][a-z0-9._-]*$/.test(form.name) ? '' : t('route.validation.nameInvalid');
    errors.domain = form.domain.trim() ? '' : t('route.validation.domainRequired');
    errors.target_url = '';
    errors.listen_port = '';
    errors.service_id = '';
    errors.component_name = '';
    errors.endpoint_protocol = '';
    errors.endpoint_container_port = '';
    if (form.protocol === 'http') {
      if (form.custom_target) {
        errors.target_url = targetUrlPattern.test(form.target_url)
          ? ''
          : t('route.validation.targetUrlInvalid');
      } else {
        setManagedTargetErrors();
      }
    } else {
      errors.listen_port = isValidListenPort(form.listen_port)
        ? ''
        : t('route.validation.listenPortInvalid');
      errors.service_id = form.service_id.trim() ? '' : t('route.validation.serviceIdRequired');
      errors.component_name = form.component_name.trim()
        ? ''
        : t('route.validation.componentNameRequired');
      errors.endpoint_protocol = form.endpoint_protocol.trim()
        ? ''
        : t('route.validation.endpointNameRequired');
      errors.endpoint_container_port = isValidListenPort(form.endpoint_container_port)
        ? ''
        : t('route.validation.endpointNameRequired');
    }
    if (Object.values(errors).some(Boolean)) {
      return;
    }
    try {
      await executeOp(async () => {
        const updated = await routeApi.update(routeId, {
          name: form.name,
          protocol: form.protocol,
          domain: form.domain,
          path_prefix: form.protocol === 'http' ? form.path_prefix : '',
          target_url: form.protocol === 'http' && form.custom_target ? form.target_url : '',
          listen_port: form.protocol === 'tcp' ? form.listen_port : undefined,
          service_id: form.protocol === 'tcp' || !form.custom_target ? form.service_id.trim() : '',
          component_name:
            form.protocol === 'tcp' || !form.custom_target ? form.component_name.trim() : '',
          endpoint_protocol:
            form.protocol === 'tcp' || !form.custom_target ? form.endpoint_protocol.trim() : '',
          endpoint_container_port:
            form.protocol === 'tcp' || !form.custom_target
              ? form.endpoint_container_port
              : undefined,
        });
        routeData.value = updated;
        toast.success(t('route.toast.updateSuccess'));
        isEditDialogOpen.value = false;
        await fetchRoute();
        await openGatewayLogs();
      });
    } catch (error) {
      editSubmitError.value =
        error instanceof Error ? error.message : t('route.toast.updateFailed');
    }
  }

  function resetProtocolFields() {
    form.service_id = '';
    form.component_name = '';
    form.endpoint_protocol = '';
    form.endpoint_container_port = undefined;
    if (form.protocol === 'http') {
      form.path_prefix ||= '/';
      form.listen_port = undefined;
      return;
    }
    form.path_prefix = '';
    form.target_url = '';
    form.custom_target = false;
  }

  function updateProtocol(value: string | number) {
    if (value !== 'http' && value !== 'tcp') {
      return;
    }
    form.protocol = value;
    resetProtocolFields();
  }

  function setManagedTargetErrors() {
    const error = t('route.validation.managedTargetRequired');
    errors.service_id = form.service_id.trim() ? '' : error;
    errors.component_name = form.component_name.trim() ? '' : error;
    errors.endpoint_protocol = form.endpoint_protocol.trim() ? '' : error;
    errors.endpoint_container_port = isValidListenPort(form.endpoint_container_port) ? '' : error;
  }

  function setCustomTarget(value: boolean) {
    form.custom_target = value;
    if (value) {
      form.service_id = '';
      form.component_name = '';
      form.endpoint_protocol = '';
      form.endpoint_container_port = undefined;
      form.target_url ||= 'http://';
      return;
    }
    form.target_url = '';
  }

  function isValidListenPort(port: number | undefined): boolean {
    return port !== undefined && Number.isInteger(port) && port >= 1 && port <= 65535;
  }

  async function handleDelete() {
    deleteSubmitError.value = '';
    try {
      await executeOp(async () => {
        await routeApi.delete(routeId);
        toast.success(t('route.toast.deleteSuccess'));
        router.push('/routes');
      });
    } catch (error) {
      deleteSubmitError.value =
        error instanceof Error ? error.message : t('route.toast.deleteFailed');
    }
  }

  function handleEnable() {
    setPendingEnabled(true);
  }

  function handleDisable() {
    setPendingEnabled(false);
  }

  function setPendingEnabled(enabled: boolean) {
    if (!routeData.value || enabled === routeData.value.enabled) {
      pendingEnabled.value = undefined;
      return;
    }
    pendingEnabled.value = enabled;
  }

  function openSyncModal() {
    isSyncDialogOpen.value = true;
  }

  async function handleSyncComplete() {
    pendingEnabled.value = undefined;
    await fetchRoute();
  }

  async function handleCertUpload(event: Event) {
    const file = (event.target as HTMLInputElement).files?.[0];
    if (!file) {
      return;
    }
    try {
      await executeOp(async () => {
        await routeApi.uploadCert(routeId, file);
        toast.success(t('route.toast.certUploadSuccess'));
        await fetchRoute();
        await openGatewayLogs();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.certUploadFailed'));
    } finally {
      (event.target as HTMLInputElement).value = '';
    }
  }

  function selectCertFile() {
    certFileInput.value?.click();
  }

  async function handleDisableHttps() {
    try {
      await executeOp(async () => {
        await routeApi.disableHttps(routeId);
        toast.success(t('route.toast.httpsDisabled'));
        await fetchRoute();
        await openGatewayLogs();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.operationFailed'));
    }
  }

  function openLetsEncryptDialog() {
    const current = routeData.value;
    if (current?.acme_challenge === 'dns' && current.dns01_available) {
      letsEncryptChallenge.value = 'dns';
    } else if (current?.http01_available) {
      letsEncryptChallenge.value = 'http';
    } else if (current?.dns01_available) {
      letsEncryptChallenge.value = 'dns';
    }
    letsEncryptChallengeError.value = '';
    letsEncryptSubmitError.value = '';
    isLetsEncryptDialogOpen.value = true;
  }

  function closeLetsEncryptDialog() {
    letsEncryptChallengeError.value = '';
    letsEncryptSubmitError.value = '';
    isLetsEncryptDialogOpen.value = false;
  }

  function updateLetsEncryptChallenge(value: string | number) {
    if (value !== 'http' && value !== 'dns') {
      return;
    }
    letsEncryptChallenge.value = value;
    letsEncryptChallengeError.value = '';
    letsEncryptSubmitError.value = '';
  }

  async function openGatewayLogs() {
    if (!gatewayRuntimeLogTarget.value) return;
    isGatewayLogsDrawerOpen.value = true;
  }

  async function handleEnableLetsencrypt() {
    letsEncryptSubmitError.value = '';
    if (!hasAvailableLetsEncryptChallenge.value) {
      letsEncryptChallengeError.value = t('route.validation.letsencryptChallengeUnavailable');
      return;
    }
    try {
      await executeOp(async () => {
        await routeApi.enableLetsencrypt(routeId, { challenge: letsEncryptChallenge.value });
        toast.success(t('route.toast.letsencryptEnabled'));
        closeLetsEncryptDialog();
        await fetchRoute();
        await openGatewayLogs();
      });
    } catch (error) {
      letsEncryptSubmitError.value =
        error instanceof Error ? error.message : t('route.toast.operationFailed');
    }
  }

  async function handleEnableMkcert() {
    try {
      await executeOp(async () => {
        await routeApi.enableMkcert(routeId, {});
        toast.success(t('route.toast.mkcertEnabled'));
        await fetchRoute();
        await openGatewayLogs();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.operationFailed'));
    }
  }

  onMounted(fetchRoute);
</script>
