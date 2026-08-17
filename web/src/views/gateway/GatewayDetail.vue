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
          class="app-button-danger h-9 px-3"
          :disabled="operating || !canStop"
          @click="openStopDialog"
        >
          <Square class="size-4" />
          {{ t('gateway.actions.stop') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/gateways')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppLoadingState v-if="status === 'loading'" size="section" />

    <template v-else-if="gateway">
      <DetailInfoCard
        :title="t('gateway.sections.config')"
        editable
        :disabled="operating"
        @edit="openEditDialog"
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
            <dt>
              {{ t('gateway.fields.defaultEntrypoint') }}
            </dt>
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

      <DetailInfoCard :title="t('gateway.exposures.title')">
        <template #actions>
          <SearchControl
            v-model="exposureSearchText"
            :placeholder="t('gateway.exposures.searchPlaceholder')"
            :loading="loading"
            class="shrink-0"
            @search="handleExposureSearch"
          />
        </template>
        <div class="px-5 py-4">
          <AppEmptyState v-if="filteredExposures.length === 0" size="compact" />
          <div v-else class="overflow-x-auto">
            <table class="app-data-table min-w-[720px]">
              <thead>
                <tr>
                  <th>{{ t('gateway.exposures.app') }}</th>
                  <th>{{ t('gateway.exposures.component') }}</th>
                  <th>{{ t('gateway.exposures.protocol') }}</th>
                  <th>{{ t('gateway.exposures.access') }}</th>
                  <th>{{ t('gateway.exposures.listen') }}</th>
                  <th>{{ t('gateway.exposures.internalDns') }}</th>
                  <th>{{ t('gateway.exposures.clientHint') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, idx) in filteredExposures" :key="idx">
                  <td class="text-foreground">{{ row.application_code }}</td>
                  <td class="text-foreground">{{ row.component_name }}</td>
                  <td>
                    <AppBadge variant="pill">{{ row.protocol }}</AppBadge>
                  </td>
                  <td>
                    <AppBadge variant="pill">{{ row.access }}</AppBadge>
                  </td>
                  <td class="text-foreground">{{ row.listen_port }}:{{ row.container_port }}</td>
                  <td class="text-foreground">{{ row.internal_dns }}</td>
                  <td class="text-foreground">
                    <a
                      v-if="isHttpAddress(row.client_hint)"
                      :href="row.client_hint"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="app-link inline-flex items-center gap-1"
                    >
                      {{ row.client_hint }}
                      <ExternalLink class="size-3.5 shrink-0" />
                    </a>
                    <template v-else>{{ row.client_hint }}</template>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </DetailInfoCard>
    </template>

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
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.deploy.version') }}
            <span class="text-destructive">*</span>
          </label>
          <SelectControl
            v-model="deployForm.version_id"
            :options="deployVersionSelectOptions"
            :placeholder="t('gateway.deploy.selectVersion')"
            :invalid="Boolean(deployErrors.version_id)"
            @update:model-value="deployErrors.version_id = ''"
          />
          <p v-if="deployErrors.version_id" class="app-field-error" role="alert">
            {{ deployErrors.version_id }}
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

    <AppDialog
      v-model:open="isEditDialogOpen"
      :title="t('gateway.dialog.edit')"
      width-class="w-[min(720px,calc(100vw-32px))]"
      body-class="space-y-4 px-6 py-4 text-sm"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.fields.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="editForm.name"
            type="text"
            class="app-input"
            :class="editErrors.name ? 'app-input-error' : ''"
            :placeholder="t('gateway.placeholders.name')"
            :aria-invalid="editErrors.name ? 'true' : undefined"
            @input="editErrors.name = ''"
          />
          <p v-if="editErrors.name" class="app-field-error mt-1 text-xs">{{ editErrors.name }}</p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">{{ t('gateway.fields.code') }}</label>
          <input :value="gateway?.code" type="text" class="app-input" disabled />
          <p class="app-field-hint mt-1">{{ t('gateway.hints.code') }}</p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.fields.restApiUrl') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="editForm.rest_api_url"
            type="text"
            class="app-input"
            :class="editErrors.rest_api_url ? 'app-input-error' : ''"
            :placeholder="t('gateway.placeholders.restApiUrl')"
            :aria-invalid="editErrors.rest_api_url ? 'true' : undefined"
            @input="editErrors.rest_api_url = ''"
          />
          <p v-if="editErrors.rest_api_url" class="app-field-error mt-1 text-xs">
            {{ editErrors.rest_api_url }}
          </p>
          <p v-else class="app-field-hint mt-1">{{ t('gateway.hints.restApiUrl') }}</p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.fields.baseDomain') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="editForm.base_domain"
            type="text"
            class="app-input"
            :class="editErrors.base_domain ? 'app-input-error' : ''"
            :placeholder="t('gateway.placeholders.baseDomain')"
            :aria-invalid="editErrors.base_domain ? 'true' : undefined"
            @input="editErrors.base_domain = ''"
          />
          <p v-if="editErrors.base_domain" class="app-field-error mt-1 text-xs">
            {{ editErrors.base_domain }}
          </p>
          <p v-else class="app-field-hint mt-1">{{ t('gateway.hints.baseDomain') }}</p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.fields.defaultEntrypoint') }}
          </label>
          <RawValueSelect
            v-model="editForm.default_entrypoint"
            :values="entrypointValues"
            :placeholder="t('gateway.placeholders.defaultEntrypoint')"
          />
          <p class="app-field-hint mt-1">{{ t('gateway.hints.defaultEntrypoint') }}</p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">{{ t('gateway.fields.tlsMode') }}</label>
          <RawValueSelect
            v-model="editForm.tls_mode"
            :values="tlsModeValues"
            :placeholder="t('gateway.placeholders.tlsMode')"
          />
        </div>
      </div>
      <p class="text-sm text-muted-foreground">{{ t('gateway.hints.compileOnSave') }}</p>
      <p v-if="editSubmitError" class="app-field-error" role="alert">
        {{ editSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          @cancel="isEditDialogOpen = false"
          @confirm="saveGateway"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, ExternalLink, Layers, Rocket, Square } from '@lucide/vue';
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
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
  import { formatTime } from '@/utils/time';

  const toast = useToast();
  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const { status, loading, execute } = useStatusAsync();
  const { status: opStatus, execute: executeOp } = useStatusAsync();

  const gateway = ref<GatewayResp | null>(null);
  const services = ref<ServiceResp[]>([]);
  const deployVersions = ref<VersionResp[]>([]);
  const exposureSearchText = ref('');
  const appliedExposureSearch = ref('');

  const filteredExposures = computed(() => {
    const exposures = gateway.value?.exposures ?? [];
    const keyword = appliedExposureSearch.value.trim().toLowerCase();
    if (!keyword) {
      return exposures;
    }
    return exposures.filter(
      (row) =>
        row.application_code.toLowerCase().includes(keyword) ||
        row.component_name.toLowerCase().includes(keyword) ||
        row.protocol.toLowerCase().includes(keyword) ||
        row.access.toLowerCase().includes(keyword) ||
        String(row.listen_port).includes(keyword) ||
        row.internal_dns.toLowerCase().includes(keyword) ||
        row.client_hint.toLowerCase().includes(keyword)
    );
  });

  const isDeployDialogOpen = ref(false);
  const deployErrors = reactive({ service_id: '', version_id: '' });
  const deploySubmitError = ref('');
  const deployForm = reactive({
    service_id: '',
    version_id: '',
    force_recreate: false,
  });

  const isStopDialogOpen = ref(false);
  const stopError = ref('');
  const stopSubmitError = ref('');
  const stopForm = reactive({
    service_id: '',
    remove_volumes: false,
  });
  const isEditDialogOpen = ref(false);
  const editForm = reactive({
    name: '',
    rest_api_url: '',
    base_domain: '',
    default_entrypoint: 'web',
    tls_mode: 'none',
  });
  const editErrors = reactive({
    name: '',
    rest_api_url: '',
    base_domain: '',
  });
  const editSubmitError = ref('');

  const gatewayId = () => String(route.params.id || '');
  const operating = computed(() => opStatus.value === 'loading');
  const isDeploying = computed(() => services.value.some((item) => item.active_deployment));
  const stoppableServices = computed(() =>
    services.value.filter(
      (item) => !item.active_deployment && (item.status === 'running' || item.status === 'faulted')
    )
  );
  const canStop = computed(() => stoppableServices.value.length > 0);
  const entrypointValues = ['web', 'websecure'];
  const tlsModeValues = ['none', 'letsencrypt', 'tls'];

  const deployServiceSelectOptions = computed(() =>
    services.value.map((item) => ({ value: item.id, label: serviceOptionLabel(item) }))
  );
  const deployVersionSelectOptions = computed(() =>
    deployVersions.value.map((item) => ({ value: item.id, label: item.label }))
  );

  const stopServiceSelectOptions = computed(() =>
    stoppableServices.value.map((item) => ({
      value: item.id,
      label: serviceOptionLabel(item),
    }))
  );

  function serviceOptionLabel(item: ServiceResp) {
    const instance = item.instance_key || 'default';
    return `${instance} (${item.status})`;
  }

  function isHttpAddress(value: string) {
    return /^https?:\/\//i.test(value);
  }

  function openEditDialog() {
    const current = gateway.value;
    if (!current) {
      return;
    }
    Object.assign(editForm, {
      name: current.name,
      rest_api_url: current.rest_api_url || '',
      base_domain: current.base_domain || '',
      default_entrypoint: current.default_entrypoint,
      tls_mode: current.tls_mode,
    });
    Object.assign(editErrors, { name: '', rest_api_url: '', base_domain: '' });
    editSubmitError.value = '';
    isEditDialogOpen.value = true;
  }

  function validateEditForm() {
    editErrors.name = editForm.name.trim() ? '' : t('gateway.validation.nameRequired');
    editErrors.rest_api_url = editForm.rest_api_url.trim()
      ? ''
      : t('gateway.validation.restApiUrlRequired');
    editErrors.base_domain = editForm.base_domain.trim()
      ? ''
      : t('gateway.validation.baseDomainRequired');
    return !editErrors.name && !editErrors.rest_api_url && !editErrors.base_domain;
  }

  async function saveGateway() {
    const current = gateway.value;
    editSubmitError.value = '';
    if (!current || !validateEditForm()) {
      return;
    }
    try {
      await executeOp(async () => {
        gateway.value = await gatewayApi.update(current.id, {
          name: editForm.name.trim(),
          rest_api_url: editForm.rest_api_url.trim(),
          base_domain: editForm.base_domain.trim(),
          default_entrypoint: editForm.default_entrypoint,
          tls_mode: editForm.tls_mode,
        });
        isEditDialogOpen.value = false;
        toast.success(t('gateway.toast.saveCompiled'));
        await loadRuntimeContext();
      });
    } catch (error) {
      editSubmitError.value =
        error instanceof Error ? error.message : t('gateway.toast.saveFailed');
    }
  }

  function handleExposureSearch() {
    appliedExposureSearch.value = exposureSearchText.value;
    void loadGateway();
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
    Object.assign(deployErrors, { service_id: '', version_id: '' });
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
    try {
      const page = await applicationApi.listVersions(current.id, { per_page: 100 });
      deployVersions.value = page.items ?? [];
      deployForm.service_id = selectedService.id;
      deployForm.version_id = selectedService.version_id;
      isDeployDialogOpen.value = true;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.loadFailed'));
    }
  }

  function handleDeployServiceChange(value: string | number) {
    const serviceId = String(value);
    deployForm.service_id = serviceId;
    deployErrors.service_id = '';
    const selectedService = services.value.find((item) => item.id === serviceId);
    if (selectedService) {
      deployForm.version_id = selectedService.version_id;
      deployErrors.version_id = '';
    }
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
    if (!deployForm.version_id) {
      deployErrors.version_id = t('gateway.deploy.versionRequired');
      return;
    }
    deployErrors.service_id = '';
    deployErrors.version_id = '';
    try {
      await executeOp(async () => {
        const selectedService = services.value.find((item) => item.id === deployForm.service_id);
        if (!selectedService) {
          throw new Error(t('gateway.toast.deployServiceRequired'));
        }
        let serviceForDeploy = selectedService;
        if (deployForm.version_id !== selectedService.version_id) {
          serviceForDeploy = await serviceApi.updateBasic(selectedService.id, {
            version_id: deployForm.version_id,
            instance_key: selectedService.instance_key,
          });
          const index = services.value.findIndex((item) => item.id === serviceForDeploy.id);
          if (index >= 0) {
            services.value[index] = serviceForDeploy;
          }
        }
        const result = await serviceApi.deploy(serviceForDeploy.id, {
          force_recreate: deployForm.force_recreate,
        });
        for (const warning of result.warnings) toast.error(warning);
        toast.success(t('gateway.toast.deployQueued'));
        isDeployDialogOpen.value = false;
        if (result.deployment_id) {
          router.push(`/deployment/${result.deployment_id}`);
          return;
        }
        await loadRuntimeContext();
      });
    } catch (err: unknown) {
      deploySubmitError.value =
        err instanceof Error ? err.message : t('gateway.toast.deployFailed');
    }
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

  watch(
    () => route.params.id,
    () => {
      loadGateway();
    }
  );

  onMounted(loadGateway);
</script>
