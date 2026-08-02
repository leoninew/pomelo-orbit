<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="min-w-0">
        <h1 class="app-detail-page-title break-words">
          {{ gateway?.name || t('gateway.detailTitle') }}
        </h1>
      </div>
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
        <button v-if="gateway" class="app-button h-9 px-3" @click="goWorkload">
          <Layers class="size-4" />
          {{ t('gateway.openWorkload') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/gateways')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppSpinner v-if="status === 'loading'" class="py-12" />

    <template v-else-if="gateway">
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">
            {{ t('gateway.sections.config') }}
          </h2>
          <button class="app-button-primary h-9 px-3" :disabled="operating" @click="openEditDialog">
            <Pencil class="size-4" />
            {{ t('common.edit') }}
          </button>
        </div>
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
      </div>

      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">
            {{ t('gateway.exposures.title') }}
          </h2>
        </div>
        <div class="px-5 py-4">
          <AppEmptyState v-if="!(gateway.exposures || []).length" size="compact" />
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
                <tr v-for="(row, idx) in gateway.exposures" :key="idx">
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
      </div>
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
            {{ t('gateway.deploy.version') }}
            <span class="text-destructive">*</span>
          </label>
          <SelectControl
            v-model="deployForm.version_id"
            :options="versionSelectOptions"
            :placeholder="t('gateway.deploy.selectVersion')"
            :invalid="Boolean(deployErrors.version_id)"
            @update:model-value="deployErrors.version_id = ''"
          />
          <p v-if="deployErrors.version_id" class="app-field-error" role="alert">
            {{ deployErrors.version_id }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.deploy.instanceKey') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="deployForm.instance_key"
            type="text"
            required
            class="app-input"
            :class="deployErrors.instance_key ? 'app-input-error' : ''"
            :placeholder="t('gateway.deploy.instanceKeyPlaceholder')"
            :aria-invalid="deployErrors.instance_key ? 'true' : undefined"
            @input="deployErrors.instance_key = ''"
          />
          <p v-if="deployErrors.instance_key" class="app-field-error" role="alert">
            {{ deployErrors.instance_key }}
          </p>
        </div>
        <label class="flex items-center gap-2">
          <input v-model="deployForm.force_recreate" type="checkbox" class="app-checkbox" />
          <span class="text-sm text-foreground">{{ t('gateway.deploy.forceRecreate') }}</span>
        </label>
      </div>
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
  import { ArrowLeft, ExternalLink, Layers, Pencil, Rocket, Square } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';

  import { serviceApi } from '@/api/service/service';
  import { gatewayApi } from '@/api/gateway/gateway';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';
  import type { VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
  import { formatTime } from '@/utils/time';

  const toast = useToast();
  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const { status, execute } = useStatusAsync();
  const { status: opStatus, execute: executeOp } = useStatusAsync();

  const gateway = ref<GatewayResp | null>(null);
  const services = ref<ServiceResp[]>([]);
  const versions = ref<VersionResp[]>([]);

  const isDeployDialogOpen = ref(false);
  const deployErrors = reactive({ version_id: '', instance_key: '' });
  const deployForm = reactive({
    version_id: '',
    instance_key: 'default',
    force_recreate: false,
  });

  const isStopDialogOpen = ref(false);
  const stopError = ref('');
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

  const gatewayId = () => String(route.params.id || '');
  const operating = computed(() => opStatus.value === 'loading');
  const isDeploying = computed(() => services.value.some((item) => item.status === 'deploying'));
  const stoppableServices = computed(() =>
    services.value.filter((item) => item.status === 'running' || item.status === 'faulted')
  );
  const canStop = computed(() => stoppableServices.value.length > 0);
  const primaryService = computed(() => services.value[0] ?? null);
  const entrypointValues = ['web', 'websecure'];
  const tlsModeValues = ['none', 'letsencrypt', 'tls'];

  const versionSelectOptions = computed(() =>
    versions.value.map((item) => ({
      value: item.id,
      label: item.status === 'published' ? item.label : `${item.label} (${item.status})`,
    }))
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
      toast.error(error instanceof Error ? error.message : t('gateway.toast.saveFailed'));
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

  async function loadDeployOptions() {
    const current = gateway.value;
    if (!current) {
      versions.value = [];
      return;
    }
    const versionResp = await applicationApi.listVersions(current.id, { per_page: 100 });
    versions.value = selectDeployableVersions(versionResp.items ?? []);
  }

  /** Prefer published versions; if none, fall back to all versions newest-first. */
  function selectDeployableVersions(items: VersionResp[]): VersionResp[] {
    const published = items.filter((item) => item.status === 'published');
    return sortVersionsByNewest(published.length > 0 ? published : items);
  }

  function sortVersionsByNewest(items: VersionResp[]): VersionResp[] {
    return [...items].sort((a, b) => {
      const ta = Date.parse(a.created_at) || 0;
      const tb = Date.parse(b.created_at) || 0;
      if (tb !== ta) {
        return tb - ta;
      }
      return b.id.localeCompare(a.id);
    });
  }

  function defaultVersionId() {
    const bound = primaryService.value?.version_id;
    if (bound && versions.value.some((item) => item.id === bound)) {
      return bound;
    }
    // versions already newest-first when falling back to unpublished.
    return versions.value[0]?.id || '';
  }

  async function openDeployDialog() {
    if (!gateway.value) {
      return;
    }
    Object.assign(deployErrors, { version_id: '', instance_key: '' });
    deployForm.force_recreate = false;
    deployForm.instance_key = 'default';
    try {
      await loadDeployOptions();
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : t('gateway.toast.loadDeployOptionsFailed'));
      return;
    }
    if (versions.value.length === 0) {
      toast.error(t('gateway.toast.noVersion'));
      return;
    }
    deployForm.version_id = defaultVersionId();
    isDeployDialogOpen.value = true;
  }

  async function handleDeployOk() {
    const current = gateway.value;
    if (!current) {
      return;
    }
    if (!deployForm.version_id) {
      deployErrors.version_id = t('gateway.toast.versionRequired');
      return;
    }
    deployErrors.version_id = '';
    deployErrors.instance_key = '';
    const instanceKey = deployForm.instance_key.trim();
    if (!instanceKey) {
      deployErrors.instance_key = t('gateway.toast.instanceKeyRequired');
      return;
    }
    try {
      await executeOp(async () => {
        const existing = services.value.find((item) => item.instance_key === instanceKey);
        let serviceId = existing?.id;
        if (serviceId) {
          const detail = await serviceApi.get(serviceId);
          if (detail.version_id !== deployForm.version_id) {
            await serviceApi.updateBasic(serviceId, {
              version_id: deployForm.version_id,
              instance_key: detail.instance_key,
            });
          }
        } else {
          const created = await serviceApi.create({
            application_id: current.id,
            version_id: deployForm.version_id,
            instance_key: instanceKey,
          });
          serviceId = created.id;
        }
        const result = await serviceApi.deploy(serviceId, {
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
      toast.error(err instanceof Error ? err.message : t('gateway.toast.deployFailed'));
    }
  }

  function openStopDialog() {
    if (!canStop.value) {
      return;
    }
    stopError.value = '';
    stopForm.remove_volumes = false;
    stopForm.service_id = stoppableServices.value[0]?.id || '';
    isStopDialogOpen.value = true;
  }

  async function handleStopOk() {
    const current = gateway.value;
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
      toast.error(err instanceof Error ? err.message : t('gateway.toast.stopFailed'));
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
