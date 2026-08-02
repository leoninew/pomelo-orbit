<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 items-center gap-2">
        <h1 class="app-detail-page-title break-words">
          {{ detail?.declaration?.name || '组件配置' }}
        </h1>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button class="app-button-primary h-9 px-3" :disabled="operating || !detail" @click="save">
          <Save class="size-4" />
          {{ t('common.save') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push(`/service/${serviceId}`)">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppSpinner v-if="loading" class="py-12" />

    <template v-else-if="detail && draft">
      <section class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">{{ t('environment.title') }}</h2>
        </div>
        <AppEmptyState v-if="environmentRows.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table table-fixed min-w-[760px]">
            <colgroup>
              <col class="w-[25%]" />
              <col class="w-[30%]" />
              <col class="w-[30%]" />
              <col class="w-[15%]" />
            </colgroup>
            <thead>
              <tr>
                <th>{{ t('environment.fields.key') }}</th>
                <th>{{ t('service.componentDetail.defaultValue') }}</th>
                <th>{{ t('service.componentDetail.currentValue') }}</th>
                <th>{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="entry in environmentRows" :key="entry.key">
                <td class="max-w-0 text-foreground">
                  <span class="block truncate" :title="entry.key">{{ entry.key }}</span>
                </td>
                <td class="max-w-0 text-muted-foreground">
                  <span class="block truncate" :title="entry.base || '-'">
                    {{ entry.base || '-' }}
                  </span>
                </td>
                <td class="max-w-0">
                  <span
                    v-if="entry.deleted"
                    class="block py-2 text-sm text-amber-700 dark:text-amber-300"
                  >
                    {{ t('common.remove') }}
                  </span>
                  <div v-else>
                    <input
                      :value="entry.value"
                      type="text"
                      class="app-input h-9"
                      :class="entry.overridden ? 'border-amber-500' : ''"
                      :aria-label="t('service.componentDetail.currentValue')"
                      :disabled="operating"
                      @input="updateEnvironmentValue(entry, $event)"
                    />
                  </div>
                </td>
                <td class="align-top whitespace-nowrap">
                  <button
                    v-if="entry.deleted || entry.overridden"
                    class="app-link inline-flex h-9 items-center"
                    :disabled="operating"
                    @click="entry.deleted ? restoreEnvironment(entry) : resetEnvironment(entry)"
                  >
                    {{ entry.deleted ? t('common.restore') : t('common.reset') }}
                  </button>
                  <button
                    v-else
                    class="app-link-danger inline-flex h-9 items-center"
                    :disabled="operating"
                    @click="removeEnvironment(entry)"
                  >
                    {{ t('common.delete') }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">资源配额</h2>
          <button
            v-if="resourceFields.length"
            :class="resourceDeleted ? 'app-link' : 'app-link-danger'"
            @click="resourceDeleted ? restoreResources() : removeResources()"
          >
            {{ resourceDeleted ? t('common.restore') : t('common.remove') }}
          </button>
        </div>
        <AppEmptyState v-if="resourceFields.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table table-fixed min-w-[760px]">
            <colgroup>
              <col class="w-[25%]" />
              <col class="w-[30%]" />
              <col class="w-[30%]" />
              <col class="w-[15%]" />
            </colgroup>
            <thead>
              <tr>
                <th>配置</th>
                <th>{{ t('service.componentDetail.defaultValue') }}</th>
                <th>{{ t('service.componentDetail.currentValue') }}</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="field in resourceFields" :key="field.key">
                <td class="text-foreground">{{ field.label }}</td>
                <td class="max-w-0 text-muted-foreground">
                  <span class="block truncate" :title="displayValue(field.base)">
                    {{ displayValue(field.base) }}
                  </span>
                </td>
                <td class="max-w-0">
                  <div v-if="editingResourceKey === field.key">
                    <input v-model="editingResourceValue" class="app-input h-9" />
                  </div>
                  <span
                    v-else
                    class="block truncate"
                    :class="
                      resourceDeleted || field.value !== field.base
                        ? 'text-amber-600 dark:text-amber-400'
                        : 'text-foreground'
                    "
                    :title="resourceDeleted ? '-' : displayValue(field.value)"
                  >
                    {{ resourceDeleted ? '-' : displayValue(field.value) }}
                  </span>
                </td>
                <td>
                  <div v-if="editingResourceKey === field.key" class="flex gap-2">
                    <button class="app-link" @click="applyResourceEdit(field)">
                      {{ t('common.save') }}
                    </button>
                    <button
                      class="text-muted-foreground hover:text-foreground"
                      @click="cancelResourceEdit"
                    >
                      {{ t('common.cancel') }}
                    </button>
                  </div>
                  <div v-else-if="!resourceDeleted" class="flex gap-2">
                    <button class="app-link" @click="startResourceEdit(field)">
                      {{ t('common.edit') }}
                    </button>
                    <button
                      v-if="field.value !== field.base"
                      class="text-muted-foreground hover:text-foreground"
                      @click="resetResource(field)"
                    >
                      {{ t('common.reset') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">Endpoint</h2>
        </div>
        <AppEmptyState v-if="endpointRows.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table table-fixed min-w-[1080px]">
            <colgroup>
              <col class="w-[12%]" />
              <col class="w-[13%]" />
              <col class="w-[25%]" />
              <col class="w-[35%]" />
              <col class="w-[15%]" />
            </colgroup>
            <thead>
              <tr>
                <th>接口</th>
                <th>契约</th>
                <th>{{ t('service.componentDetail.defaultValue') }}</th>
                <th>{{ t('service.componentDetail.currentValue') }}</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="endpoint in endpointRows" :key="endpoint.name">
                <td class="max-w-0 text-foreground">
                  <span class="block truncate" :title="endpoint.name">{{ endpoint.name }}</span>
                </td>
                <td>
                  <AppBadge variant="pill">{{ endpoint.protocol }}</AppBadge>
                  / {{ endpoint.container_port }}
                </td>
                <td class="max-w-0 text-muted-foreground">
                  <dl class="space-y-1">
                    <div
                      v-for="field in endpointFields"
                      :key="field.key"
                      class="flex min-w-0 gap-2"
                    >
                      <dt class="shrink-0">{{ field.label }}</dt>
                      <dd class="min-w-0 truncate" :title="displayValue(endpoint.base[field.key])">
                        {{ displayValue(endpoint.base[field.key]) }}
                      </dd>
                    </div>
                  </dl>
                </td>
                <td class="max-w-0">
                  <div v-if="editingEndpointName === endpoint.name" class="space-y-2">
                    <label
                      v-for="field in endpointFields"
                      :key="field.key"
                      class="grid grid-cols-[72px_minmax(0,1fr)] items-center gap-2"
                    >
                      <span class="text-muted-foreground">{{ field.label }}</span>
                      <RawValueSelect
                        v-if="field.key === 'mode'"
                        v-model="editingEndpoint[field.key]"
                        :values="endpointModes"
                        width-class="w-full"
                      />
                      <input v-else v-model="editingEndpoint[field.key]" class="app-input h-9" />
                    </label>
                  </div>
                  <span v-else-if="endpoint.deleted" class="text-amber-600 dark:text-amber-400">
                    -
                  </span>
                  <dl v-else class="space-y-1">
                    <div
                      v-for="field in endpointFields"
                      :key="field.key"
                      class="flex min-w-0 gap-2"
                    >
                      <dt class="shrink-0 text-muted-foreground">{{ field.label }}</dt>
                      <dd
                        class="min-w-0 truncate"
                        :class="
                          endpoint.value[field.key] !== endpoint.base[field.key]
                            ? 'text-amber-600 dark:text-amber-400'
                            : 'text-foreground'
                        "
                        :title="displayValue(endpoint.value[field.key])"
                      >
                        {{ displayValue(endpoint.value[field.key]) }}
                      </dd>
                    </div>
                  </dl>
                </td>
                <td>
                  <div v-if="editingEndpointName === endpoint.name" class="flex gap-2">
                    <button class="app-link" @click="applyEndpointEdit(endpoint)">
                      {{ t('common.save') }}
                    </button>
                    <button
                      class="text-muted-foreground hover:text-foreground"
                      @click="cancelEndpointEdit"
                    >
                      {{ t('common.cancel') }}
                    </button>
                  </div>
                  <div v-else class="flex gap-2">
                    <button
                      v-if="!endpoint.deleted"
                      class="app-link"
                      @click="startEndpointEdit(endpoint)"
                    >
                      {{ t('common.edit') }}
                    </button>
                    <button
                      v-if="endpoint.deleted"
                      class="app-link"
                      @click="restoreEndpoint(endpoint)"
                    >
                      {{ t('common.restore') }}
                    </button>
                    <button
                      v-else-if="endpointHasChanges(endpoint)"
                      class="text-muted-foreground hover:text-foreground"
                      @click="resetEndpoint(endpoint)"
                    >
                      {{ t('common.reset') }}
                    </button>
                    <button v-else class="app-link-danger" @click="removeEndpoint(endpoint)">
                      {{ t('common.remove') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">挂载</h2>
        </div>
        <AppEmptyState v-if="mountRows.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table table-fixed min-w-[880px]">
            <colgroup>
              <col class="w-[18%]" />
              <col class="w-[15%]" />
              <col class="w-[26%]" />
              <col class="w-[26%]" />
              <col class="w-[15%]" />
            </colgroup>
            <thead>
              <tr>
                <th>目标</th>
                <th>类型</th>
                <th>{{ t('service.componentDetail.defaultValue') }}</th>
                <th>{{ t('service.componentDetail.currentValue') }}</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="mount in mountRows" :key="mount.target">
                <td class="max-w-0 text-foreground">
                  <span class="block truncate" :title="mount.target">{{ mount.target }}</span>
                </td>
                <td>
                  <AppBadge variant="pill">{{ mount.source_type }}</AppBadge>
                </td>
                <td class="max-w-0 text-muted-foreground">
                  <span class="block truncate" :title="displayValue(mount.base)">
                    {{ displayValue(mount.base) }}
                  </span>
                </td>
                <td class="max-w-0">
                  <div v-if="editingMountTarget === mount.target">
                    <input v-model="editingMountSource" class="app-input h-9" />
                  </div>
                  <span
                    v-else
                    class="block truncate"
                    :class="
                      mount.deleted || mount.source !== mount.base
                        ? 'text-amber-600 dark:text-amber-400'
                        : 'text-foreground'
                    "
                    :title="mount.deleted ? '-' : displayValue(mount.source)"
                  >
                    {{ mount.deleted ? '-' : displayValue(mount.source) }}
                  </span>
                </td>
                <td>
                  <div v-if="editingMountTarget === mount.target" class="flex gap-2">
                    <button class="app-link" @click="applyMountEdit(mount)">
                      {{ t('common.save') }}
                    </button>
                    <button
                      class="text-muted-foreground hover:text-foreground"
                      @click="cancelMountEdit"
                    >
                      {{ t('common.cancel') }}
                    </button>
                  </div>
                  <div v-else class="flex gap-2">
                    <button v-if="!mount.deleted" class="app-link" @click="startMountEdit(mount)">
                      {{ t('common.edit') }}
                    </button>
                    <button v-if="mount.deleted" class="app-link" @click="restoreMount(mount)">
                      {{ t('common.restore') }}
                    </button>
                    <button
                      v-else-if="mount.source !== mount.base"
                      class="text-muted-foreground hover:text-foreground"
                      @click="resetMount(mount)"
                    >
                      {{ t('common.reset') }}
                    </button>
                    <button v-else class="app-link-danger" @click="removeMount(mount)">
                      {{ t('common.remove') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Save } from 'lucide-vue-next';
  import { computed, onMounted, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { serviceApi } from '@/api/service/service';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type {
    ServiceComponentDetailResp,
    ServiceComponentOverlayUpdateReq,
  } from '@/gen/proto/orbit/v1/service/service';

  type EnvRow = {
    key: string;
    base: string;
    inheritedValue: string;
    value: string;
    deleted: boolean;
    overridden: boolean;
  };
  type MountRow = {
    target: string;
    source_type: string;
    base: string;
    source: string;
    deleted: boolean;
  };
  type EndpointValues = {
    mode: string;
    bind_address: string;
    listen_port: string;
    entrypoint: string;
    path_prefix: string;
  };
  type EndpointField = {
    key: keyof EndpointValues;
    label: string;
  };
  type EndpointRow = {
    name: string;
    protocol: string;
    container_port: number;
    base: EndpointValues;
    value: EndpointValues;
    deleted: boolean;
  };
  type ResourceField = {
    key: 'limit_cpus' | 'limit_memory' | 'reservation_cpus' | 'reservation_memory';
    label: string;
    base: string;
    value: string;
  };

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const { loading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOperation } = useStatusAsync();
  const serviceId = String(route.params.id || '');
  const componentId = String(route.params.componentId || '');
  const detail = ref<ServiceComponentDetailResp>();
  const environmentRows = ref<EnvRow[]>([]);
  const mountRows = ref<MountRow[]>([]);
  const editingMountTarget = ref<string | null>(null);
  const editingMountSource = ref('');
  const endpointRows = ref<EndpointRow[]>([]);
  const editingEndpointName = ref<string | null>(null);
  const editingEndpoint = ref<EndpointValues>(emptyEndpointValues());
  const resourceDeleted = ref(false);
  const resourceFields = ref<ResourceField[]>([]);
  const editingResourceKey = ref<ResourceField['key'] | null>(null);
  const editingResourceValue = ref('');
  const endpointModes = ['internal', 'local', 'host', 'gateway_http', 'gateway_tcp'];
  const endpointFields: EndpointField[] = [
    { key: 'mode', label: '模式' },
    { key: 'bind_address', label: '监听地址' },
    { key: 'listen_port', label: '监听端口' },
    { key: 'entrypoint', label: '入口' },
    { key: 'path_prefix', label: '路径' },
  ];

  function emptyEndpointValues(): EndpointValues {
    return { mode: '', bind_address: '', listen_port: '', entrypoint: '', path_prefix: '' };
  }

  function overlayByKey<T extends { state: string }>(
    items: T[],
    key: string,
    select: (item: T) => string
  ) {
    return items.find((item) => select(item) === key);
  }
  function draftFromResponse(value: ServiceComponentDetailResp) {
    if (!value.declaration || !value.component)
      throw new Error('service component detail is incomplete');
    const declaration = value.declaration;
    const component = value.component;
    const effectiveEnv = value.effective?.env ?? declaration.env;
    environmentRows.value = declaration.env.map((item) => {
      const overlay = overlayByKey(component.env, item.key, (candidate) => candidate.key);
      const effective = effectiveEnv.find((candidate) => candidate.key === item.key);
      const inheritedValue = effective?.value ?? item.value;
      return {
        key: item.key,
        base: item.value,
        inheritedValue,
        value: overlay?.value ?? inheritedValue,
        deleted: overlay?.state === 'deleted',
        overridden: overlay?.state === 'override',
      };
    });
    mountRows.value = declaration.mounts.map((item) => {
      const overlay = component.mounts.find((candidate) => candidate.target === item.target);
      return {
        target: item.target,
        source_type: item.source_type,
        base: item.source,
        source: overlay?.source ?? item.source,
        deleted: overlay?.state === 'deleted',
      };
    });
    endpointRows.value = declaration.endpoints.map((item) => {
      const overlay = overlayByKey(component.endpoints, item.name, (candidate) => candidate.name);
      const base = {
        mode: item.mode,
        bind_address: item.bind_address ?? '',
        listen_port: String(item.listen_port ?? ''),
        entrypoint: item.entrypoint ?? '',
        path_prefix: item.path_prefix ?? '',
      };
      return {
        name: item.name,
        protocol: item.protocol,
        container_port: item.container_port,
        base,
        value: {
          mode: overlay?.mode ?? base.mode,
          bind_address: overlay?.bind_address ?? base.bind_address,
          listen_port: String(overlay?.listen_port ?? base.listen_port),
          entrypoint: overlay?.entrypoint ?? base.entrypoint,
          path_prefix: overlay?.path_prefix ?? base.path_prefix,
        },
        deleted: overlay?.state === 'deleted',
      };
    });
    const resources = declaration.resources;
    const overlay = component.resources;
    resourceDeleted.value = overlay?.state === 'deleted';
    resourceFields.value = resources
      ? [
          {
            key: 'limit_cpus',
            label: 'CPU Limit',
            base: resources.limit_cpus ?? '',
            value: overlay?.limit_cpus ?? resources.limit_cpus ?? '',
          },
          {
            key: 'limit_memory',
            label: 'Memory Limit',
            base: resources.limit_memory ?? '',
            value: overlay?.limit_memory ?? resources.limit_memory ?? '',
          },
          {
            key: 'reservation_cpus',
            label: 'CPU Reservation',
            base: resources.reservation_cpus ?? '',
            value: overlay?.reservation_cpus ?? resources.reservation_cpus ?? '',
          },
          {
            key: 'reservation_memory',
            label: 'Memory Reservation',
            base: resources.reservation_memory ?? '',
            value: overlay?.reservation_memory ?? resources.reservation_memory ?? '',
          },
        ]
      : [];
  }
  function updateEnvironmentValue(row: EnvRow, event: Event) {
    row.value = (event.target as HTMLInputElement).value;
    row.deleted = false;
    row.overridden = row.value !== row.inheritedValue;
  }
  function resetEnvironment(row: EnvRow) {
    row.value = row.inheritedValue;
    row.deleted = false;
    row.overridden = false;
  }
  function removeEnvironment(row: EnvRow) {
    row.deleted = true;
    row.overridden = false;
  }
  function restoreEnvironment(row: EnvRow) {
    row.value = row.inheritedValue;
    row.deleted = false;
    row.overridden = false;
  }
  function startResourceEdit(field: ResourceField) {
    editingResourceKey.value = field.key;
    editingResourceValue.value = field.value;
  }
  function applyResourceEdit(field: ResourceField) {
    field.value = editingResourceValue.value;
    cancelResourceEdit();
  }
  function cancelResourceEdit() {
    editingResourceKey.value = null;
    editingResourceValue.value = '';
  }
  function resetResource(field: ResourceField) {
    field.value = field.base;
  }
  function removeResources() {
    resourceDeleted.value = true;
    cancelResourceEdit();
  }
  function restoreResources() {
    resourceDeleted.value = false;
    resourceFields.value.forEach((field) => {
      field.value = field.base;
    });
  }
  function startMountEdit(row: MountRow) {
    editingMountTarget.value = row.target;
    editingMountSource.value = row.source;
  }
  function applyMountEdit(row: MountRow) {
    row.source = editingMountSource.value;
    row.deleted = false;
    cancelMountEdit();
  }
  function cancelMountEdit() {
    editingMountTarget.value = null;
    editingMountSource.value = '';
  }
  function resetMount(row: MountRow) {
    row.source = row.base;
    row.deleted = false;
  }
  function removeMount(row: MountRow) {
    row.deleted = true;
    cancelMountEdit();
  }
  function restoreMount(row: MountRow) {
    row.source = row.base;
    row.deleted = false;
  }
  function startEndpointEdit(row: EndpointRow) {
    editingEndpointName.value = row.name;
    editingEndpoint.value = { ...row.value };
  }
  function applyEndpointEdit(row: EndpointRow) {
    row.value = { ...editingEndpoint.value };
    row.deleted = false;
    cancelEndpointEdit();
  }
  function cancelEndpointEdit() {
    editingEndpointName.value = null;
    editingEndpoint.value = emptyEndpointValues();
  }
  function endpointHasChanges(row: EndpointRow) {
    return endpointFields.some((field) => row.value[field.key] !== row.base[field.key]);
  }
  function resetEndpoint(row: EndpointRow) {
    row.value = { ...row.base };
    row.deleted = false;
  }
  function removeEndpoint(row: EndpointRow) {
    row.deleted = true;
    cancelEndpointEdit();
  }
  function restoreEndpoint(row: EndpointRow) {
    row.value = { ...row.base };
    row.deleted = false;
  }
  function displayValue(value: string) {
    return value || '-';
  }
  const draft = computed(() => (detail.value?.declaration ? true : false));
  function optional(value: string) {
    return value === '' ? undefined : value;
  }
  function resourceOverride(key: ResourceField['key']) {
    const field = resourceFields.value.find((item) => item.key === key);
    if (!field || field.value === field.base) {
      return undefined;
    }
    return optional(field.value);
  }
  function payload(): ServiceComponentOverlayUpdateReq {
    return {
      env: environmentRows.value.flatMap((row) => {
        if (row.deleted) return [{ key: row.key, state: 'deleted' }];
        return row.overridden ? [{ key: row.key, value: row.value, state: 'override' }] : [];
      }),
      mounts: mountRows.value.map((row) =>
        row.deleted
          ? { target: row.target, state: 'deleted' }
          : { target: row.target, source: row.source, state: 'override' }
      ),
      resources:
        resourceFields.value.length === 0
          ? undefined
          : resourceDeleted.value
            ? { state: 'deleted' }
            : {
                state: 'override',
                limit_cpus: resourceOverride('limit_cpus'),
                limit_memory: resourceOverride('limit_memory'),
                reservation_cpus: resourceOverride('reservation_cpus'),
                reservation_memory: resourceOverride('reservation_memory'),
              },
      endpoints: endpointRows.value.map((row) =>
        row.deleted
          ? { name: row.name, state: 'deleted' }
          : {
              name: row.name,
              state: 'override',
              mode: row.value.mode,
              bind_address: optional(row.value.bind_address),
              listen_port: row.value.listen_port === '' ? undefined : Number(row.value.listen_port),
              entrypoint: optional(row.value.entrypoint),
              path_prefix: optional(row.value.path_prefix),
            }
      ),
    };
  }
  async function load() {
    try {
      await execute(async () => {
        const value = await serviceApi.getComponent(serviceId, componentId);
        detail.value = value;
        draftFromResponse(value);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '加载组件配置失败');
      await router.push(`/service/${serviceId}`);
    }
  }
  async function save() {
    try {
      await executeOperation(async () => {
        await serviceApi.updateComponent(serviceId, componentId, payload());
        await load();
        toast.success('已保存，等待显式部署');
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '保存失败');
    }
  }
  onMounted(load);
</script>
