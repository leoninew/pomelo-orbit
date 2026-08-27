<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('gateway.toolbar')">
      <SearchControl
        v-model="searchText"
        class="shrink-0"
        :placeholder="t('gateway.searchPlaceholder')"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <div class="flex items-center gap-3">
        <button class="app-button-primary px-5" @click="openCreateDialog">
          <Plus class="size-4" />
          {{ t('gateway.create') }}
        </button>
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || t('gateway.toast.loadFailed') }}</p>
      </div>
      <AppEmptyState v-else-if="gateways.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[1040px]">
          <thead>
            <tr>
              <th>{{ t('gateway.fields.name') }}</th>
              <th>{{ t('gateway.fields.code') }}</th>
              <th>{{ t('gateway.fields.traefikComponentName') }}</th>
              <th>{{ t('gateway.sections.ingressDefaults') }}</th>
              <th>{{ t('gateway.sections.routeCertificates') }}</th>
              <th>{{ t('gateway.fields.baseDomain') }}</th>
              <th>{{ t('common.updatedAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in gateways" :key="item.id">
              <td>
                <router-link :to="`/gateway/${item.id}`" class="app-link">
                  {{ item.name }}
                </router-link>
              </td>
              <td class="text-foreground">{{ item.code }}</td>
              <td class="text-foreground">{{ item.traefik_component_name }}</td>
              <td>
                <div class="flex flex-wrap gap-1.5">
                  <AppBadge variant="pill">{{ item.default_entrypoint }}</AppBadge>
                  <AppBadge variant="pill">{{ item.tls_mode }}</AppBadge>
                </div>
              </td>
              <td>
                <div class="flex flex-wrap gap-1.5">
                  <AppBadge variant="pill">{{ acmeProfileLabel(item.acme_profile) }}</AppBadge>
                </div>
              </td>
              <td class="text-foreground">{{ item.base_domain }}</td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(item.config_updated_at) }}
              </td>
              <td class="whitespace-nowrap">
                <button
                  class="app-link-danger"
                  :disabled="operating"
                  @click="openDeleteDialog(item)"
                >
                  {{ t('common.delete') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <ListPagination
        :current="pagination.current"
        :page-size="pagination.pageSize"
        :total="pagination.total"
        :total-pages="totalPages"
        @change-page="goPage"
        @change-page-size="handlePageSizeChange"
      />
    </div>

    <GatewayExposuresCard
      v-if="exposureGateway"
      :gateway="exposureGateway"
      :gateway-options="exposureGatewayOptions"
      :selected-gateway-id="exposureGatewayId"
      :loading="exposureLoading"
      @update:selected-gateway-id="selectExposureGateway"
    />

    <AppDialog
      v-model:open="isCreateDialogOpen"
      :title="t('gateway.dialog.create')"
      width-class="w-[min(920px,calc(100vw-32px))]"
      content-class="max-h-[calc(100vh-32px)] overflow-y-auto"
      body-class="px-6 py-5 text-sm"
    >
      <div class="space-y-8">
        <section class="space-y-4" aria-labelledby="gateway-create-control-plane">
          <h2 id="gateway-create-control-plane" class="app-detail-section-title">
            {{ t('gateway.sections.controlPlane') }}
          </h2>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div class="space-y-1.5">
              <label class="app-field-label block" for="gateway-create-name">
                {{ t('gateway.fields.name') }}
                <span class="text-destructive">*</span>
              </label>
              <input
                id="gateway-create-name"
                :value="createForm.name"
                type="text"
                class="app-input"
                :class="createErrors.name ? 'app-input-error' : ''"
                :aria-invalid="createErrors.name ? 'true' : undefined"
                @input="updateCreateField('name', ($event.target as HTMLInputElement).value)"
              />
              <p v-if="createErrors.name" class="app-field-error" role="alert">
                {{ createValidationMessage('name') }}
              </p>
            </div>
            <div class="space-y-1.5">
              <label class="app-field-label block" for="gateway-create-code">
                {{ t('gateway.fields.code') }}
                <span class="text-destructive">*</span>
              </label>
              <input
                id="gateway-create-code"
                :value="createForm.code"
                type="text"
                class="app-input"
                :class="createErrors.code ? 'app-input-error' : ''"
                :aria-invalid="createErrors.code ? 'true' : undefined"
                @input="updateCreateField('code', ($event.target as HTMLInputElement).value)"
              />
              <p v-if="createErrors.code" class="app-field-error" role="alert">
                {{ createValidationMessage('code') }}
              </p>
            </div>
            <div class="space-y-1.5">
              <label class="app-field-label block" for="gateway-create-traefik-component-name">
                {{ t('gateway.fields.traefikComponentName') }}
                <span class="text-destructive">*</span>
              </label>
              <input
                id="gateway-create-traefik-component-name"
                :value="createForm.traefik_component_name"
                type="text"
                class="app-input"
                :class="createErrors.traefik_component_name ? 'app-input-error' : ''"
                :aria-invalid="createErrors.traefik_component_name ? 'true' : undefined"
                @input="
                  updateCreateField(
                    'traefik_component_name',
                    ($event.target as HTMLInputElement).value
                  )
                "
              />
              <p v-if="createErrors.traefik_component_name" class="app-field-error" role="alert">
                {{ createValidationMessage('traefik_component_name') }}
              </p>
            </div>
            <div class="space-y-1.5">
              <label class="app-field-label block" for="gateway-create-rest-api-url">
                {{ t('gateway.fields.restApiUrl') }}
                <span class="text-destructive">*</span>
              </label>
              <input
                id="gateway-create-rest-api-url"
                :value="createForm.rest_api_url"
                type="url"
                class="app-input"
                :class="createErrors.rest_api_url ? 'app-input-error' : ''"
                :aria-invalid="createErrors.rest_api_url ? 'true' : undefined"
                @input="
                  updateCreateField('rest_api_url', ($event.target as HTMLInputElement).value)
                "
              />
              <p v-if="createErrors.rest_api_url" class="app-field-error" role="alert">
                {{ createValidationMessage('rest_api_url') }}
              </p>
            </div>
            <div class="space-y-1.5">
              <label class="app-field-label block" for="gateway-create-rest-ready-timeout">
                {{ t('gateway.fields.restReadyTimeout') }}
                <span class="text-destructive">*</span>
              </label>
              <input
                id="gateway-create-rest-ready-timeout"
                :value="createForm.rest_ready_timeout_seconds"
                type="number"
                min="1"
                max="300"
                class="app-input"
                :class="createErrors.rest_ready_timeout_seconds ? 'app-input-error' : ''"
                :aria-invalid="createErrors.rest_ready_timeout_seconds ? 'true' : undefined"
                @input="
                  updateCreateField(
                    'rest_ready_timeout_seconds',
                    ($event.target as HTMLInputElement).value
                  )
                "
              />
              <p
                v-if="createErrors.rest_ready_timeout_seconds"
                class="app-field-error"
                role="alert"
              >
                {{ createValidationMessage('rest_ready_timeout_seconds') }}
              </p>
            </div>
            <div class="space-y-1.5">
              <label class="app-field-label block" for="gateway-create-base-domain">
                {{ t('gateway.fields.baseDomain') }}
                <span class="text-destructive">*</span>
              </label>
              <input
                id="gateway-create-base-domain"
                :value="createForm.base_domain"
                type="text"
                class="app-input"
                :class="createErrors.base_domain ? 'app-input-error' : ''"
                :aria-invalid="createErrors.base_domain ? 'true' : undefined"
                @input="updateCreateField('base_domain', ($event.target as HTMLInputElement).value)"
              />
              <p v-if="createErrors.base_domain" class="app-field-error" role="alert">
                {{ createValidationMessage('base_domain') }}
              </p>
            </div>
            <div class="space-y-1.5">
              <label class="app-field-label block" for="gateway-create-initial-component-image">
                {{ t('gateway.fields.image') }}
                <span class="text-destructive">*</span>
              </label>
              <input
                id="gateway-create-initial-component-image"
                :value="createForm.initial_component_image"
                type="text"
                class="app-input"
                :class="createErrors.initial_component_image ? 'app-input-error' : ''"
                :aria-invalid="createErrors.initial_component_image ? 'true' : undefined"
                @input="
                  updateCreateField(
                    'initial_component_image',
                    ($event.target as HTMLInputElement).value
                  )
                "
              />
              <p v-if="createErrors.initial_component_image" class="app-field-error" role="alert">
                {{ createValidationMessage('initial_component_image') }}
              </p>
            </div>
            <div class="space-y-1.5">
              <label class="app-field-label block" for="gateway-create-pull-policy">
                {{ t('gateway.fields.imagePullPolicy') }}
              </label>
              <SelectControl
                id="gateway-create-pull-policy"
                :model-value="createForm.initial_component_pull_policy"
                :options="pullPolicyOptions"
                :invalid="Boolean(createErrors.initial_component_pull_policy)"
                @update:model-value="
                  updateCreateField('initial_component_pull_policy', String($event))
                "
              />
              <p
                v-if="createErrors.initial_component_pull_policy"
                class="app-field-error"
                role="alert"
              >
                {{ createValidationMessage('initial_component_pull_policy') }}
              </p>
            </div>
          </div>
        </section>

        <section class="space-y-4" aria-labelledby="gateway-create-ingress">
          <h2 id="gateway-create-ingress" class="app-detail-section-title">
            {{ t('gateway.sections.ingressDefaults') }}
          </h2>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div class="space-y-1.5">
              <label class="app-field-label block" for="gateway-create-entrypoint">
                {{ t('gateway.fields.defaultEntrypoint') }}
              </label>
              <SelectControl
                id="gateway-create-entrypoint"
                :model-value="createForm.default_entrypoint"
                :options="entrypointOptions"
                :invalid="Boolean(createErrors.default_entrypoint)"
                @update:model-value="updateCreateField('default_entrypoint', String($event))"
              />
              <p v-if="createErrors.default_entrypoint" class="app-field-error" role="alert">
                {{ createValidationMessage('default_entrypoint') }}
              </p>
            </div>
            <div class="space-y-1.5">
              <label class="app-field-label block" for="gateway-create-tls-mode">
                {{ t('gateway.fields.tlsMode') }}
              </label>
              <SelectControl
                id="gateway-create-tls-mode"
                :model-value="createForm.tls_mode"
                :options="tlsModeOptions"
                :invalid="Boolean(createErrors.tls_mode)"
                @update:model-value="updateCreateField('tls_mode', String($event))"
              />
              <p v-if="createErrors.tls_mode" class="app-field-error" role="alert">
                {{ createValidationMessage('tls_mode') }}
              </p>
            </div>
          </div>
        </section>

        <section class="space-y-4" aria-labelledby="gateway-create-certificates">
          <h2 id="gateway-create-certificates" class="app-detail-section-title">
            {{ t('gateway.sections.routeCertificates') }}
          </h2>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div class="space-y-1.5">
              <label class="app-field-label block" for="gateway-create-acme-profile">
                {{ t('gateway.fields.acmeProfile') }}
              </label>
              <SelectControl
                id="gateway-create-acme-profile"
                :model-value="createAcmeProfileValue"
                :options="acmeProfileOptions"
                :invalid="Boolean(createErrors.acme_profile)"
                @update:model-value="handleCreateAcmeProfileChange"
              />
              <p v-if="createErrors.acme_profile" class="app-field-error" role="alert">
                {{ createValidationMessage('acme_profile') }}
              </p>
            </div>
            <div v-if="createForm.acme_profile" class="space-y-1.5">
              <label class="app-field-label block" for="gateway-create-acme-email">
                {{ t('gateway.fields.acmeEmail') }}
                <span class="text-destructive">*</span>
              </label>
              <input
                id="gateway-create-acme-email"
                :value="createForm.acme_email"
                type="email"
                class="app-input"
                :class="createErrors.acme_email ? 'app-input-error' : ''"
                :aria-invalid="createErrors.acme_email ? 'true' : undefined"
                @input="updateCreateField('acme_email', ($event.target as HTMLInputElement).value)"
              />
              <p v-if="createErrors.acme_email" class="app-field-error" role="alert">
                {{ createValidationMessage('acme_email') }}
              </p>
            </div>
            <div v-if="createUsesDNSProfile" class="space-y-1.5">
              <label class="app-field-label block" for="gateway-create-dns-api-token">
                {{ t('gateway.fields.dnsApiToken') }}
                <span class="text-destructive">*</span>
              </label>
              <div class="relative">
                <input
                  id="gateway-create-dns-api-token"
                  :value="createForm.dns_api_token"
                  :type="isCreateTokenVisible ? 'text' : 'password'"
                  class="app-input pr-10"
                  :class="createErrors.dns_api_token ? 'app-input-error' : ''"
                  :aria-invalid="createErrors.dns_api_token ? 'true' : undefined"
                  @input="
                    updateCreateField('dns_api_token', ($event.target as HTMLInputElement).value)
                  "
                />
                <button
                  type="button"
                  class="absolute right-1 top-1/2 inline-flex size-9 -translate-y-1/2 items-center justify-center text-muted-foreground hover:text-foreground"
                  :aria-label="isCreateTokenVisible ? t('common.hideValue') : t('common.showValue')"
                  :title="isCreateTokenVisible ? t('common.hideValue') : t('common.showValue')"
                  @click="isCreateTokenVisible = !isCreateTokenVisible"
                >
                  <EyeOff v-if="isCreateTokenVisible" class="size-4" />
                  <Eye v-else class="size-4" />
                </button>
              </div>
              <p v-if="createErrors.dns_api_token" class="app-field-error" role="alert">
                {{ createValidationMessage('dns_api_token') }}
              </p>
            </div>
          </div>
        </section>
      </div>
      <p v-if="createSubmitError" class="app-field-error mt-4" role="alert">
        {{ createSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.create')"
          @cancel="closeCreateDialog"
          @confirm="handleCreate"
        />
      </template>
    </AppDialog>

    <AppDialog v-model:open="isDeleteDialogOpen" :title="t('gateway.dialog.delete')">
      <p class="text-sm text-muted-foreground">
        {{ t('gateway.dialog.deleteConfirm', { name: pendingDelete?.name || '' }) }}
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
  import { Eye, EyeOff, Plus } from '@lucide/vue';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { ToolbarRoot } from 'reka-ui';
  import { gatewayApi } from '@/api/gateway/gateway';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';
  import { useProjectStore } from '@/stores/project';
  import { formatTime } from '@/utils/time';
  import {
    emptyGatewayConfigForm,
    gatewayCreateRequestFromForm,
    type GatewayAcmeProfile,
    type GatewayConfigForm,
    type GatewayConfigFormErrors,
    validateGatewayConfigForm,
  } from './gatewayConfigForm';
  import GatewayExposuresCard from './components/GatewayExposuresCard.vue';

  const { t } = useI18n();
  const toast = useToast();
  const router = useRouter();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { loading: exposureLoading, execute: executeExposure } = useStatusAsync();
  const gateways = ref<GatewayResp[]>([]);
  const exposureGateway = ref<GatewayResp>();
  const exposureGatewayId = ref('');
  const searchText = ref('');
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize) || 1);
  const exposureGatewayOptions = computed(() =>
    gateways.value.map((gateway) => ({
      value: gateway.id,
      label: `${gateway.name} (${gateway.code})`,
    }))
  );
  const isCreateDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const pendingDelete = ref<GatewayResp>();
  const createForm = reactive<GatewayConfigForm>(emptyGatewayConfigForm());
  const createErrors = reactive<GatewayConfigFormErrors>({});
  const createSubmitError = ref('');
  const deleteSubmitError = ref('');
  const isCreateTokenVisible = ref(false);
  const noAcmeProfileValue = '__acme_disabled__';
  const pullPolicyOptions = [
    { value: 'missing', label: 'missing' },
    { value: 'always', label: 'always' },
    { value: 'never', label: 'never' },
  ];
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
  const createAcmeProfileValue = computed(() => createForm.acme_profile || noAcmeProfileValue);
  const createUsesDNSProfile = computed(
    () => createForm.acme_profile === 'dns' || createForm.acme_profile === 'http-dns'
  );
  let exposureRequest = 0;

  watch(
    () => projectStore.activeProjectId,
    () => {
      pagination.current = 1;
      clearExposureGateway();
      void fetchData();
    }
  );
  watch(isCreateDialogOpen, (open) => {
    if (!open) resetCreateState();
  });

  async function fetchData() {
    const projectID = projectStore.activeProjectId;
    if (!projectID) {
      gateways.value = [];
      pagination.total = 0;
      clearExposureGateway();
      return;
    }
    try {
      await execute(async () => {
        const result = await gatewayApi.list({
          project_id: projectID,
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
        });
        gateways.value = result.items ?? [];
        pagination.total = result.total ?? 0;
      });
      syncExposureGateway();
    } catch {
      toast.error(t('gateway.toast.loadFailed'));
    }
  }

  function replaceErrors(target: GatewayConfigFormErrors, next: GatewayConfigFormErrors) {
    for (const field of Object.keys(target)) delete target[field];
    Object.assign(target, next);
  }

  function resetCreateState() {
    Object.assign(createForm, emptyGatewayConfigForm());
    replaceErrors(createErrors, {});
    createSubmitError.value = '';
    isCreateTokenVisible.value = false;
  }

  function openCreateDialog() {
    resetCreateState();
    isCreateDialogOpen.value = true;
  }

  function closeCreateDialog() {
    isCreateDialogOpen.value = false;
  }

  function clearCreateError(field: string) {
    delete createErrors[field];
  }

  function updateCreateField<K extends keyof GatewayConfigForm>(
    field: K,
    value: GatewayConfigForm[K]
  ) {
    createForm[field] = value;
    clearCreateError(String(field));
  }

  function handleCreateAcmeProfileChange(value: string | number) {
    const profile = String(value);
    updateCreateField(
      'acme_profile',
      profile === noAcmeProfileValue ? '' : (profile as GatewayAcmeProfile)
    );
  }

  function createValidationMessage(field: string) {
    return t(`gateway.validation.${createErrors[field]}`);
  }

  function handleSearch() {
    pagination.current = 1;
    void fetchData();
  }

  function acmeProfileLabel(profile: string) {
    switch (profile) {
      case 'http':
        return t('gateway.acmeProfiles.http');
      case 'dns':
        return t('gateway.acmeProfiles.dns');
      case 'http-dns':
        return t('gateway.acmeProfiles.httpDns');
      default:
        return t('gateway.acmeProfiles.none');
    }
  }

  function clearExposureGateway() {
    exposureRequest += 1;
    exposureGatewayId.value = '';
    exposureGateway.value = undefined;
  }

  function syncExposureGateway() {
    const selected = gateways.value.find((item) => item.id === exposureGatewayId.value);
    const id = selected?.id || gateways.value[0]?.id || '';
    if (id === exposureGatewayId.value && exposureGateway.value) {
      return;
    }
    void selectExposureGateway(id);
  }

  async function selectExposureGateway(id: string) {
    exposureGatewayId.value = id;
    const request = ++exposureRequest;
    if (!id) {
      exposureGateway.value = undefined;
      return;
    }
    if (exposureGateway.value?.id !== id) {
      exposureGateway.value = undefined;
    }
    try {
      const detail = await executeExposure(() => gatewayApi.get(id));
      if (request === exposureRequest && detail) {
        exposureGateway.value = detail;
      }
    } catch {
      if (request === exposureRequest) {
        exposureGateway.value = undefined;
        toast.error(t('gateway.toast.loadDetailFailed'));
      }
    }
  }

  function goPage(page: number) {
    pagination.current = page;
    void fetchData();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    void fetchData();
  }

  async function handleCreate() {
    createSubmitError.value = '';
    const errors = validateGatewayConfigForm(createForm, 'create');
    replaceErrors(createErrors, errors);
    if (Object.keys(errors).length > 0) return;
    const projectID = projectStore.activeProjectId;
    if (!projectID) {
      createSubmitError.value = t('gateway.toast.selectProjectRequired');
      return;
    }
    try {
      await executeOp(async () => {
        const created = await gatewayApi.create(
          gatewayCreateRequestFromForm(createForm, projectID)
        );
        closeCreateDialog();
        await router.push(`/gateway/${created.id}`);
      });
    } catch (error) {
      createSubmitError.value =
        error instanceof Error ? error.message : t('gateway.toast.createFailed');
    }
  }

  function openDeleteDialog(value: GatewayResp) {
    pendingDelete.value = value;
    deleteSubmitError.value = '';
    isDeleteDialogOpen.value = true;
  }

  async function handleDelete() {
    const item = pendingDelete.value;
    if (!item) return;
    deleteSubmitError.value = '';
    try {
      await executeOp(async () => {
        await gatewayApi.delete(item.id);
        toast.success(t('gateway.toast.deleteSuccess'));
        isDeleteDialogOpen.value = false;
        await fetchData();
      });
    } catch (error) {
      deleteSubmitError.value =
        error instanceof Error ? error.message : t('gateway.toast.deleteFailed');
    }
  }

  onMounted(fetchData);
</script>
