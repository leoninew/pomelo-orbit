<template>
  <div class="flex flex-col gap-4">
    <!-- Header -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">
          {{ routeData?.name ?? t('route.detailTitle') }}
        </h1>
        <DetailHeaderMeta v-if="routeData">
          <AppBadge variant="status" :tone="routeData.enabled ? 'success' : 'default'">
            {{ routeData.enabled ? t('route.status.enabled') : t('route.status.disabled') }}
          </AppBadge>
          <AppBadge
            variant="status"
            :tone="
              routeData.protocol === 'tcp'
                ? 'warning'
                : routeData.https_enabled
                  ? 'info'
                  : 'default'
            "
          >
            {{ routeData.protocol === 'tcp' ? 'TCP' : routeData.https_enabled ? 'HTTPS' : 'HTTP' }}
          </AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
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
                v-if="routeData.protocol === 'http'"
                :href="`${routeData.https_enabled ? 'https' : 'http'}://${routeData.domain}`"
                target="_blank"
                class="app-link inline-flex items-center gap-1"
              >
                {{ routeData.domain }}
                <ExternalLink class="size-3" />
              </a>
              <span v-else class="text-foreground">
                {{ routeData.domain }}:{{ routeData.listen_port }}
              </span>
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
          <div v-if="routeData.target_url" class="flex gap-2">
            <dt>{{ t('route.fields.targetUrl') }}</dt>
            <dd class="text-foreground">{{ routeData.target_url }}</dd>
          </div>
          <template v-if="routeData.service_id">
            <div class="flex gap-2">
              <dt>{{ t('route.fields.listenPort') }}</dt>
              <dd class="text-foreground">{{ routeData.listen_port }}</dd>
            </div>
            <div class="flex gap-2">
              <dt>{{ t('route.fields.serviceId') }}</dt>
              <dd class="text-foreground">{{ routeData.service_id }}</dd>
            </div>
            <div class="flex gap-2">
              <dt>{{ t('route.fields.componentName') }}</dt>
              <dd class="text-foreground">{{ routeData.component_name }}</dd>
            </div>
            <div class="flex gap-2">
              <dt>{{ t('route.fields.endpointName') }}</dt>
              <dd class="text-foreground">
                {{ routeData.endpoint_protocol }}{{ routeData.endpoint_container_port }}
              </dd>
            </div>
          </template>
          <div class="flex gap-2">
            <dt>{{ t('common.status') }}</dt>
            <dd>
              <AppBadge variant="status" :tone="routeData.enabled ? 'success' : 'default'">
                {{ routeData.enabled ? t('route.status.enabled') : t('route.status.disabled') }}
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
        <div class="space-y-4 px-5 py-4">
          <div class="flex items-center justify-between rounded-md bg-muted/30 p-3">
            <div>
              <p class="text-sm text-foreground">{{ t('route.currentStatus') }}</p>
              <p class="text-sm text-muted-foreground">
                {{ routeData.https_enabled ? t('route.httpsEnabled') : t('route.httpsDisabled') }}
              </p>
            </div>
            <AppBadge variant="status" :tone="routeData.https_enabled ? 'info' : 'default'">
              {{ routeData.https_enabled ? 'HTTPS' : 'HTTP' }}
            </AppBadge>
          </div>

          <div v-if="!routeData.https_enabled" class="space-y-2">
            <button
              v-if="canUseLetsencrypt"
              class="app-action-item"
              :disabled="operating"
              @click="handleEnableLetsencrypt"
            >
              <p class="text-sm text-foreground">{{ t('route.enableLetsencrypt') }}</p>
              <p class="text-sm text-muted-foreground">{{ t('route.letsencryptHint') }}</p>
            </button>
            <button class="app-action-item" :disabled="operating" @click="handleEnableMkcert">
              <p class="text-sm text-foreground">{{ t('route.enableMkcert') }}</p>
              <p class="text-sm text-muted-foreground">{{ t('route.mkcertHint') }}</p>
            </button>
            <label class="app-action-item cursor-pointer">
              <p class="text-sm text-foreground">{{ t('route.uploadCustomCert') }}</p>
              <p class="text-sm text-muted-foreground">{{ t('route.uploadCustomCertHint') }}</p>
              <input type="file" accept=".pem" class="hidden" @change="handleCertUpload" />
            </label>
          </div>

          <div v-else class="flex justify-end">
            <button class="app-button-danger" :disabled="operating" @click="handleDisableHttps">
              {{ t('route.disableHttps') }}
            </button>
          </div>
        </div>
      </DetailInfoCard>
      <p v-if="editSubmitError" class="app-field-error mt-3" role="alert">
        {{ editSubmitError }}
      </p>
    </template>

    <AppDialog v-model:open="isEditDialogOpen" :title="t('route.editRoute')">
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('route.fields.protocol') }}</label>
          <select v-model="form.protocol" class="app-input" @change="resetProtocolFields">
            <option value="http">HTTP</option>
            <option value="tcp">TCP</option>
          </select>
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
        <template v-else>
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
        <label class="flex cursor-pointer items-center gap-3">
          <SwitchRoot v-model="form.enabled" class="app-switch-root">
            <SwitchThumb class="app-switch-thumb" />
          </SwitchRoot>
          <span class="text-sm text-foreground">{{ t('route.status.enabled') }}</span>
        </label>
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
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, ExternalLink, Trash2 } from '@lucide/vue';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { SwitchRoot, SwitchThumb } from 'reka-ui';
  import { useRoute, useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import type { RouteResp } from '@/gen/proto/orbit/v1/route/route';
  import { routeApi } from '@/api/route/route';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import RouteManagedTargetSelect from '@/components/RouteManagedTargetSelect.vue';
  import { usePageBreadcrumbs } from '@/composables/useBreadcrumbs';
  import { useRouteTargetServices } from '@/composables/useRouteTargetServices';
  import { useProjectStore } from '@/stores/project';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { formatTime } from '@/utils/time';

  const currentRoute = useRoute();
  const router = useRouter();
  const routeId = currentRoute.params.id as string;
  const toast = useToast();
  const projectStore = useProjectStore();
  const { t } = useI18n();
  usePageBreadcrumbs([]);
  const targetUrlPattern = /^https?:\/\/[a-zA-Z0-9.-]+(?::\d+)?$/;

  const { loading: basicInfoLoading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { services: targetServices, load: loadTargetServices } = useRouteTargetServices();

  const routeData = ref<RouteResp>();
  const isEditDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);

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
    enabled: false,
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
          custom_target: Boolean(data.target_url),
          listen_port: data.listen_port,
          service_id: data.service_id ?? '',
          component_name: data.component_name ?? '',
          endpoint_protocol: data.endpoint_protocol ?? '',
          endpoint_container_port: data.endpoint_container_port,
          enabled: data.enabled,
        });
      });
    } catch {
      toast.error(t('route.toast.loadDetailFailed'));
      router.push('/routes');
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
      custom_target: Boolean(routeData.value.target_url),
      listen_port: routeData.value.listen_port,
      service_id: routeData.value.service_id ?? '',
      component_name: routeData.value.component_name ?? '',
      endpoint_protocol: routeData.value.endpoint_protocol ?? '',
      endpoint_container_port: routeData.value.endpoint_container_port,
      enabled: routeData.value.enabled,
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
          enabled: form.enabled,
        });
        routeData.value = updated;
        toast.success(t('route.toast.updateSuccess'));
        isEditDialogOpen.value = false;
        await fetchRoute();
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

  async function handleCertUpload(event: Event) {
    const file = (event.target as HTMLInputElement).files?.[0];
    if (!file) {
      return;
    }
    try {
      await executeOp(async () => {
        await routeApi.uploadCert(routeId, file);
        toast.success(t('route.toast.certUploadSuccess'));
        fetchRoute();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.certUploadFailed'));
    }
  }

  async function handleDisableHttps() {
    try {
      await executeOp(async () => {
        await routeApi.disableHttps(routeId);
        toast.success(t('route.toast.httpsDisabled'));
        fetchRoute();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.operationFailed'));
    }
  }

  async function handleEnableLetsencrypt() {
    try {
      await executeOp(async () => {
        await routeApi.enableLetsencrypt(routeId, {});
        toast.success(t('route.toast.letsencryptEnabled'));
        fetchRoute();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.operationFailed'));
    }
  }

  async function handleEnableMkcert() {
    try {
      await executeOp(async () => {
        await routeApi.enableMkcert(routeId, {});
        toast.success(t('route.toast.mkcertEnabled'));
        fetchRoute();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.operationFailed'));
    }
  }

  onMounted(fetchRoute);
</script>
