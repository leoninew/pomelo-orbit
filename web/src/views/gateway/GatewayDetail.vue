<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="min-w-0">
        <h1 class="truncate text-xl font-semibold text-foreground">
          {{ gateway?.name || t('gateway.detailTitle') }}
        </h1>
        <p v-if="gateway" class="mt-1 text-sm text-muted-foreground">
          {{ gateway.code }}
          <span v-if="primaryService" class="text-muted-foreground">
            · {{ serviceStatusLabel(primaryService.status) }}
          </span>
        </p>
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
        <button
          v-if="gateway"
          class="app-button h-9 px-3"
          @click="router.push(`/gateway/${gateway.id}/edit`)"
        >
          <Pencil class="size-4" />
          {{ t('common.edit') }}
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
      <div class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">{{ t('gateway.sections.config') }}</h2>
        </div>
        <dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('gateway.fields.name') }}</dt>
            <dd class="text-foreground">{{ gateway.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('gateway.fields.code') }}</dt>
            <dd class="text-foreground">{{ gateway.code }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">
              {{ t('gateway.fields.restApiUrl') }}
            </dt>
            <dd class="min-w-0 break-all text-foreground">{{ gateway.rest_api_url }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">
              {{ t('gateway.fields.baseDomain') }}
            </dt>
            <dd class="text-foreground">{{ gateway.base_domain }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">
              {{ t('gateway.fields.defaultEntrypoint') }}
            </dt>
            <dd class="text-foreground">{{ gateway.default_entrypoint || '—' }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('gateway.fields.tlsMode') }}</dt>
            <dd class="text-foreground">{{ gateway.tls_mode || 'none' }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">
              {{ t('gateway.fields.imagePullPolicy') }}
            </dt>
            <dd class="text-foreground">{{ imagePullPolicyLabel(gateway.image_pull_policy) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('gateway.fields.image') }}</dt>
            <dd class="min-w-0 break-all text-foreground">{{ gateway.image || '—' }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.createdAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(gateway.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.updatedAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(gateway.updated_at) }}</dd>
          </div>
        </dl>
      </div>

      <div class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">{{ t('gateway.exposures.title') }}</h2>
        </div>
        <div class="px-5 py-4">
          <p class="mb-4 text-sm text-muted-foreground">{{ t('gateway.exposures.hint') }}</p>
          <div v-if="!(gateway.exposures || []).length" class="text-sm text-muted-foreground">
            {{ t('gateway.exposures.empty') }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="app-table-list min-w-[720px]">
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
                  <td class="text-foreground">{{ row.protocol }}</td>
                  <td class="text-foreground">{{ row.access }}</td>
                  <td class="text-foreground">{{ row.listen_port }} → {{ row.container_port }}</td>
                  <td class="text-foreground">{{ row.internal_dns }}</td>
                  <td class="text-foreground">{{ row.client_hint }}</td>
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
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.deploy.environment') }}
            <span class="text-destructive">*</span>
          </label>
          <SelectControl
            v-model="deployForm.environment_id"
            :options="environmentSelectOptions"
            :placeholder="t('gateway.deploy.selectEnvironment')"
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('gateway.deploy.instanceKey') }}
          </label>
          <input
            v-model="deployForm.instance_key"
            type="text"
            class="app-input"
            :placeholder="t('gateway.deploy.instanceKeyPlaceholder')"
          />
        </div>
        <label class="flex items-center gap-2">
          <input v-model="deployForm.force_recreate" type="checkbox" class="app-checkbox" />
          <span class="text-sm text-foreground">{{ t('gateway.deploy.forceRecreate') }}</span>
        </label>
        <p v-if="deployError" class="app-field-error text-xs">{{ deployError }}</p>
      </div>
      <template #footer>
        <button class="app-button" @click="isDeployDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="handleDeployOk">
          {{ t('gateway.actions.deploy') }}
        </button>
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
          />
          <p v-if="stopError" class="app-field-error mt-1 text-xs">{{ stopError }}</p>
        </div>
        <label class="flex items-center gap-2">
          <input v-model="stopForm.remove_volumes" type="checkbox" class="app-checkbox" />
          <span class="text-sm text-foreground">{{ t('gateway.stop.removeVolumes') }}</span>
        </label>
      </div>
      <template #footer>
        <button class="app-button" @click="isStopDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-danger" :disabled="operating" @click="handleStopOk">
          {{ t('gateway.actions.stop') }}
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Layers, Pencil, Rocket, Square } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import { environmentApi } from '@/api/environment/environment';
  import { gatewayApi } from '@/api/gateway/gateway';
  import AppDialog from '@/components/AppDialog.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { EnvironmentResp } from '@/gen/proto/orbit/v1/environment/environment';
  import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';
  import type { VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
  import { formatTime } from '@/utils/time';

  const toast = useToast();
  const { t, te } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const { status, execute } = useStatusAsync();
  const { status: opStatus, execute: executeOp } = useStatusAsync();

  const gateway = ref<GatewayResp | null>(null);
  const services = ref<ServiceResp[]>([]);
  const versions = ref<VersionResp[]>([]);
  const environments = ref<EnvironmentResp[]>([]);

  const isDeployDialogOpen = ref(false);
  const deployError = ref('');
  const deployForm = reactive({
    version_id: '',
    environment_id: '',
    instance_key: 'default',
    force_recreate: false,
  });

  const isStopDialogOpen = ref(false);
  const stopError = ref('');
  const stopForm = reactive({
    service_id: '',
    remove_volumes: false,
  });

  const gatewayId = () => String(route.params.id || '');
  const operating = computed(() => opStatus.value === 'loading');
  const isDeploying = computed(() => services.value.some((item) => item.status === 'deploying'));
  const stoppableServices = computed(() =>
    services.value.filter((item) => item.status === 'running' || item.status === 'faulted')
  );
  const canStop = computed(() => stoppableServices.value.length > 0);
  const primaryService = computed(() => services.value[0] ?? null);

  const versionSelectOptions = computed(() =>
    versions.value.map((item) => ({
      value: item.id,
      label:
        item.status === 'published'
          ? item.label
          : `${item.label} (${serviceStatusLabel(item.status)})`,
    }))
  );

  const environmentSelectOptions = computed(() =>
    environments.value.map((item) => ({
      value: item.id,
      label: item.name || item.code || item.id,
    }))
  );

  const stopServiceSelectOptions = computed(() =>
    stoppableServices.value.map((item) => ({
      value: item.id,
      label: serviceOptionLabel(item),
    }))
  );

  function imagePullPolicyLabel(policy: string) {
    const key = `application.imagePullPolicyLabels.${policy}`;
    return te(key) ? t(key) : policy || '—';
  }

  function serviceStatusLabel(value: string) {
    const key = `status.${value}`;
    return te(key) ? t(key) : value;
  }

  function serviceOptionLabel(item: ServiceResp) {
    const env = item.environment_code || item.environment_name || item.environment_id;
    const instance = item.instance_key || 'default';
    const statusText = serviceStatusLabel(item.status);
    return `${env} / ${instance} (${statusText})`;
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
      environments.value = [];
      return;
    }
    const [versionResp, envResp] = await Promise.all([
      applicationApi.listVersions(current.id, { per_page: 100 }),
      current.project_id
        ? environmentApi.list({ project_id: current.project_id, per_page: 100 })
        : Promise.resolve({ items: [] as EnvironmentResp[] }),
    ]);
    versions.value = selectDeployableVersions(versionResp.items ?? []);
    environments.value = envResp.items ?? [];
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

  function defaultEnvironmentId() {
    const local = environments.value.find((item) => item.code === 'local');
    return local?.id || environments.value[0]?.id || '';
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
    deployError.value = '';
    deployForm.force_recreate = false;
    deployForm.instance_key = primaryService.value?.instance_key || 'default';
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
    deployForm.environment_id = primaryService.value?.environment_id || defaultEnvironmentId();
    if (!deployForm.environment_id) {
      toast.error(t('gateway.toast.environmentRequired'));
      return;
    }
    isDeployDialogOpen.value = true;
  }

  async function handleDeployOk() {
    const current = gateway.value;
    if (!current) {
      return;
    }
    if (!deployForm.version_id) {
      deployError.value = t('gateway.toast.versionRequired');
      return;
    }
    if (!deployForm.environment_id) {
      deployError.value = t('gateway.toast.environmentRequired');
      return;
    }
    deployError.value = '';
    try {
      await executeOp(async () => {
        const result = await applicationApi.deploy(current.id, {
          version_id: deployForm.version_id,
          environment_id: deployForm.environment_id,
          instance_key: deployForm.instance_key.trim() || 'default',
          force_recreate: deployForm.force_recreate,
          runtime_config: {},
        });
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
