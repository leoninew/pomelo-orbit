<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <DetailPageHeader
        :items="breadcrumbs"
        :title="detail?.version_component?.name || '组件配置'"
      />
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

    <AppLoadingState v-if="loading" size="section" />

    <template v-else-if="detail && draft">
      <DetailInfoCard class="order-0" :title="t('service.componentDetail.runtimeTitle')">
        <div class="overflow-x-auto">
          <table class="app-data-table table-fixed min-w-[760px]">
            <colgroup>
              <col class="w-[24%]" />
              <col class="w-[28%]" />
              <col class="w-[33%]" />
              <col class="w-[15%]" />
            </colgroup>
            <thead>
              <tr>
                <th>配置</th>
                <th>{{ t('service.componentDetail.defaultValue') }}</th>
                <th>{{ t('service.componentDetail.currentValue') }}</th>
                <th>{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="field in runtimeRows" :key="field.key">
                <td class="text-foreground">{{ field.label }}</td>
                <td class="max-w-0 text-muted-foreground">
                  <span class="block truncate" :title="displayValue(field.state.base)">
                    {{ displayValue(field.state.base) }}
                  </span>
                </td>
                <td class="max-w-0">
                  <RawValueSelect
                    v-if="editingRuntimeKey === field.key && field.kind === 'select'"
                    v-model="editingRuntimeValue"
                    :values="field.values"
                    width-class="h-9 w-full"
                  />
                  <textarea
                    v-else-if="editingRuntimeKey === field.key"
                    v-model="editingRuntimeValue"
                    class="app-textarea"
                    rows="3"
                  />
                  <span
                    v-else
                    class="block h-9 truncate leading-9"
                    :class="
                      runtimeFieldDiffers(field)
                        ? 'text-amber-600 dark:text-amber-400'
                        : 'text-foreground'
                    "
                    :title="displayValue(field.state.value)"
                  >
                    {{ displayValue(field.state.value) }}
                  </span>
                </td>
                <td class="whitespace-nowrap">
                  <div v-if="editingRuntimeKey === field.key" class="flex h-9 items-center gap-2">
                    <button class="app-link" :disabled="operating" @click="applyRuntimeEdit(field)">
                      {{ t('common.save') }}
                    </button>
                    <button
                      class="text-muted-foreground hover:text-foreground"
                      :disabled="operating"
                      @click="cancelRuntimeEdit"
                    >
                      {{ t('common.cancel') }}
                    </button>
                  </div>
                  <div v-else class="flex h-9 items-center gap-2">
                    <button class="app-link" :disabled="operating" @click="startRuntimeEdit(field)">
                      {{ t('common.edit') }}
                    </button>
                    <button
                      v-if="runtimeFieldDiffers(field)"
                      class="text-muted-foreground hover:text-foreground"
                      :disabled="operating"
                      @click="resetRuntime(field)"
                    >
                      {{ t('service.componentDetail.reset') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </DetailInfoCard>

      <DetailInfoCard class="order-2" :title="t('environment.title')">
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
                  <div v-if="editingEnvironmentKey === entry.key">
                    <input
                      v-model="editingEnvironmentValue"
                      type="text"
                      class="app-input h-9"
                      :aria-label="t('service.componentDetail.currentValue')"
                      :disabled="operating"
                    />
                  </div>
                  <span
                    v-else
                    class="block h-9 truncate leading-9"
                    :class="
                      entry.deleted || entry.overridden
                        ? 'text-amber-600 dark:text-amber-400'
                        : 'text-foreground'
                    "
                    :title="entry.deleted ? '-' : displayValue(entry.value)"
                  >
                    {{ entry.deleted ? '-' : displayValue(entry.value) }}
                  </span>
                </td>
                <td class="whitespace-nowrap">
                  <div
                    v-if="editingEnvironmentKey === entry.key"
                    class="flex h-9 items-center gap-2"
                  >
                    <button
                      class="app-link"
                      :disabled="operating"
                      @click="applyEnvironmentEdit(entry)"
                    >
                      {{ t('common.save') }}
                    </button>
                    <button
                      class="text-muted-foreground hover:text-foreground"
                      :disabled="operating"
                      @click="cancelEnvironmentEdit"
                    >
                      {{ t('common.cancel') }}
                    </button>
                  </div>
                  <div v-else class="flex h-9 items-center gap-2">
                    <button
                      v-if="!entry.deleted"
                      class="app-link"
                      :disabled="operating"
                      @click="startEnvironmentEdit(entry)"
                    >
                      {{ t('common.edit') }}
                    </button>
                    <button
                      v-if="entry.deleted"
                      class="app-link"
                      :disabled="operating"
                      @click="resetEnvironmentToVersion(entry)"
                    >
                      {{ t('service.componentDetail.reset') }}
                    </button>
                    <button
                      v-else-if="entry.overridden"
                      class="text-muted-foreground hover:text-foreground"
                      :disabled="operating"
                      @click="resetEnvironmentToVersion(entry)"
                    >
                      {{ t('service.componentDetail.reset') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </DetailInfoCard>

      <DetailInfoCard class="order-4" title="资源配额">
        <template #actions>
          <button
            v-if="resourceDeleted"
            class="app-link"
            :disabled="operating"
            @click="resetResourcesToVersion()"
          >
            {{ t('service.componentDetail.reset') }}
          </button>
        </template>
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
                    class="block h-9 truncate leading-9"
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
                  <div v-if="editingResourceKey === field.key" class="flex h-9 items-center gap-2">
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
                  <div v-else-if="!resourceDeleted" class="flex h-9 items-center gap-2">
                    <button class="app-link" @click="startResourceEdit(field)">
                      {{ t('common.edit') }}
                    </button>
                    <button
                      v-if="field.value !== field.base"
                      class="text-muted-foreground hover:text-foreground"
                      @click="resetResourceToVersion(field)"
                    >
                      {{ t('service.componentDetail.reset') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </DetailInfoCard>

      <DetailInfoCard class="order-1" title="Endpoint">
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
              <tr v-for="endpoint in endpointRows" :key="endpoint.identity">
                <td class="max-w-0 text-foreground">
                  <span class="block truncate" :title="endpoint.identity">
                    {{ endpoint.identity }}
                  </span>
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
                  <div v-if="editingEndpointIdentity === endpoint.identity" class="space-y-2">
                    <label
                      v-for="field in endpointFields"
                      :key="field.key"
                      class="grid grid-cols-[72px_minmax(0,1fr)] items-center gap-2"
                    >
                      <span class="text-muted-foreground">{{ field.label }}</span>
                      <RawValueSelect
                        v-if="field.key === 'mode'"
                        v-model="editingEndpoint[field.key]"
                        :values="endpointModeValues(endpoint.protocol)"
                        width-class="h-9 w-full"
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
                  <div
                    v-if="editingEndpointIdentity === endpoint.identity"
                    class="flex h-9 items-center gap-2"
                  >
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
                  <div v-else class="flex h-9 items-center gap-2">
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
                      @click="resetEndpointToVersion(endpoint)"
                    >
                      {{ t('service.componentDetail.reset') }}
                    </button>
                    <button
                      v-else-if="endpointHasChanges(endpoint)"
                      class="text-muted-foreground hover:text-foreground"
                      @click="resetEndpointToVersion(endpoint)"
                    >
                      {{ t('service.componentDetail.reset') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </DetailInfoCard>

      <DetailInfoCard class="order-3" title="挂载">
        <AppEmptyState v-if="mountRows.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[760px]">
            <thead>
              <tr>
                <th>{{ t('application.componentDetail.fields.sourceType') }}</th>
                <th>{{ t('application.componentDetail.fields.source') }}</th>
                <th>{{ t('application.componentDetail.fields.target') }}</th>
                <th>{{ t('application.componentDetail.fields.readOnly') }}</th>
                <th class="w-48">{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="mount in mountRows" :key="mount.target">
                <td class="min-w-28 text-foreground">
                  {{ mountTypeLabel(mount.source_type) }}
                </td>
                <td
                  class="min-w-56 break-all"
                  :class="
                    mount.deleted || mountHasChanges(mount)
                      ? 'text-amber-600 dark:text-amber-400'
                      : 'text-foreground'
                  "
                >
                  {{ mount.deleted ? '-' : displayValue(mount.source) }}
                </td>
                <td class="min-w-56 break-all text-foreground">
                  {{ mount.target }}
                </td>
                <td class="min-w-20 text-foreground">
                  {{ mount.read_only ? t('common.yes') : t('common.no') }}
                </td>
                <td class="w-36">
                  <div class="flex items-center gap-3">
                    <button
                      v-if="!mount.deleted"
                      class="app-link"
                      :disabled="operating"
                      @click="openMountDialog(mount)"
                    >
                      {{ t('common.edit') }}
                    </button>
                    <button
                      v-if="mount.deleted"
                      class="app-link"
                      :disabled="operating"
                      @click="resetMountToVersion(mount)"
                    >
                      {{ t('service.componentDetail.reset') }}
                    </button>
                    <button
                      v-else-if="mountHasChanges(mount)"
                      class="text-muted-foreground hover:text-foreground"
                      :disabled="operating"
                      @click="resetMountToVersion(mount)"
                    >
                      {{ t('service.componentDetail.reset') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </DetailInfoCard>
    </template>

    <AppDialog
      :open="mountDialogOpen"
      :title="t('common.edit')"
      width-class="w-[min(640px,calc(100vw-32px))]"
      body-class="space-y-4 px-6 py-4 text-sm"
      @update:open="setMountDialogOpen"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.sourceType') }}
          </label>
          <input
            :value="editingMount?.source_type ?? ''"
            class="app-input bg-muted"
            :aria-label="t('application.componentDetail.fields.sourceType')"
            readonly
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.source') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="editingMountSource"
            class="app-input"
            :class="mountSourceError ? 'app-input-error' : ''"
            type="text"
            :aria-invalid="mountSourceError ? 'true' : undefined"
            :aria-describedby="mountSourceError ? 'service-mount-source-error' : undefined"
            @input="clearMountSourceError"
          />
          <p
            v-if="mountSourceError"
            id="service-mount-source-error"
            class="app-field-error"
            role="alert"
          >
            {{ mountSourceError }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.target') }}
          </label>
          <input
            :value="editingMount?.target ?? ''"
            class="app-input bg-muted"
            :aria-label="t('application.componentDetail.fields.target')"
            readonly
          />
        </div>
        <label class="flex items-center gap-2 self-end pb-2">
          <input
            :checked="editingMount?.read_only ?? false"
            class="app-checkbox"
            type="checkbox"
            disabled
          />
          <span class="text-sm text-foreground">
            {{ t('application.componentDetail.fields.readOnly') }}
          </span>
        </label>
        <label
          v-if="editingMount?.source_type === 'directory' || editingMount?.source_type === 'file'"
          class="flex items-center gap-2 sm:col-span-2"
        >
          <input v-model="editingMountSourceIsHostPath" class="app-checkbox" type="checkbox" />
          <span class="text-sm text-foreground">
            {{ t('application.componentDetail.fields.sourceIsHostPath') }}
          </span>
        </label>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.confirm')"
          @cancel="closeMountDialog"
          @confirm="saveMountDialog"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Save } from '@lucide/vue';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { serviceApi } from '@/api/service/service';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import DetailPageHeader from '@/components/DetailPageHeader.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type {
    ServiceComponentDetailResp,
    ServiceComponentOverlayUpdateReq,
    ServiceResp,
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
    read_only: boolean;
    base: string;
    base_source_is_host_path: boolean;
    source: string;
    source_is_host_path: boolean;
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
    identity: string;
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
  type RuntimeFieldKey = 'entrypoint' | 'command' | 'pull_policy' | 'restart_policy';
  type RuntimeFieldState = {
    base: string;
    value: string;
    overridden: boolean;
  };
  type RuntimeOverlayDraft = Record<RuntimeFieldKey, RuntimeFieldState>;
  type RuntimeFieldDefinition = {
    key: RuntimeFieldKey;
    label: string;
    kind: 'text' | 'select';
    values: string[];
  };
  type RuntimeField = {
    key: RuntimeFieldKey;
    label: string;
    state: RuntimeFieldState;
    kind: 'text' | 'select';
    values: string[];
  };

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const { loading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOperation } = useStatusAsync();
  const serviceId = String(route.params.id || '');
  const componentId = String(route.params.componentId || '');
  const service = ref<ServiceResp>();
  const detail = ref<ServiceComponentDetailResp>();
  const breadcrumbs = computed(() => {
    const currentService = service.value;
    if (!currentService || !detail.value) {
      return [];
    }
    return [
      {
        label: `${currentService.application_name} / ${currentService.instance_key || 'default'}`,
        to: `/service/${currentService.id}`,
      },
    ];
  });
  const environmentRows = ref<EnvRow[]>([]);
  const editingEnvironmentKey = ref<string | null>(null);
  const editingEnvironmentValue = ref('');
  const mountRows = ref<MountRow[]>([]);
  const mountDialogOpen = ref(false);
  const editingMountTarget = ref<string | null>(null);
  const editingMountSource = ref('');
  const editingMountSourceIsHostPath = ref(false);
  const mountSourceError = ref('');
  const endpointRows = ref<EndpointRow[]>([]);
  const runtimeDraft = reactive<RuntimeOverlayDraft>(emptyRuntimeOverlayDraft());
  const editingRuntimeKey = ref<RuntimeFieldKey | null>(null);
  const editingRuntimeValue = ref('');
  const editingEndpointIdentity = ref<string | null>(null);
  const editingEndpoint = ref<EndpointValues>(emptyEndpointValues());
  const resourceDeleted = ref(false);
  const resourceFields = ref<ResourceField[]>([]);
  const editingResourceKey = ref<ResourceField['key'] | null>(null);
  const editingResourceValue = ref('');
  function endpointModeValues(protocol: string) {
    return protocol === 'http'
      ? ['internal', 'local', 'host', 'gateway']
      : ['internal', 'local', 'host'];
  }
  const pullPolicyValues = ['always', 'missing', 'never'];
  const restartPolicyValues = ['no', 'unless-stopped'];
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
  function emptyRuntimeOverlayDraft(): RuntimeOverlayDraft {
    return {
      entrypoint: { base: '', value: '', overridden: false },
      command: { base: '', value: '', overridden: false },
      pull_policy: { base: '', value: '', overridden: false },
      restart_policy: { base: '', value: '', overridden: false },
    };
  }

  function overlayByKey<T extends { state: string }>(
    items: T[],
    key: string,
    select: (item: T) => string
  ) {
    return items.find((item) => select(item) === key);
  }
  function draftFromResponse(value: ServiceComponentDetailResp) {
    if (!value.version_component || !value.service_component)
      throw new Error('service component detail is incomplete');
    const declaration = value.version_component;
    const component = value.service_component;
    Object.assign(runtimeDraft.entrypoint, {
      base: declaration.entrypoint,
      value: component.entrypoint ?? '',
      overridden: component.entrypoint !== undefined,
    });
    Object.assign(runtimeDraft.command, {
      base: declaration.command,
      value: component.command ?? '',
      overridden: component.command !== undefined,
    });
    Object.assign(runtimeDraft.pull_policy, {
      base: declaration.pull_policy,
      value: component.pull_policy ?? '',
      overridden: component.pull_policy !== undefined,
    });
    Object.assign(runtimeDraft.restart_policy, {
      base: declaration.restart_policy ?? '',
      value: component.restart_policy ?? '',
      overridden: component.restart_policy !== undefined,
    });
    environmentRows.value = declaration.env.map((item) => {
      const overlay = overlayByKey(component.env, item.key, (candidate) => candidate.key);
      const inheritedValue = item.value;
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
        read_only: item.read_only,
        base: item.source,
        base_source_is_host_path: item.source_is_host_path,
        source: overlay?.source ?? item.source,
        source_is_host_path: overlay?.source_is_host_path ?? item.source_is_host_path,
        deleted: overlay?.state === 'deleted',
      };
    });
    endpointRows.value = declaration.endpoints.map((item) => {
      const overlay = component.endpoints.find(
        (candidate) =>
          candidate.protocol === item.protocol && candidate.container_port === item.container_port
      );
      const base = {
        mode: item.mode,
        bind_address: item.bind_address ?? '',
        listen_port: String(item.listen_port ?? ''),
        entrypoint: item.entrypoint ?? '',
        path_prefix: item.path_prefix ?? '',
      };
      return {
        identity: `${item.protocol}${item.container_port}`,
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
  function startEnvironmentEdit(row: EnvRow) {
    editingEnvironmentKey.value = row.key;
    editingEnvironmentValue.value = row.value;
  }
  function applyEnvironmentEdit(row: EnvRow) {
    row.value = editingEnvironmentValue.value;
    row.deleted = false;
    row.overridden = row.value !== row.inheritedValue;
    cancelEnvironmentEdit();
  }
  function cancelEnvironmentEdit() {
    editingEnvironmentKey.value = null;
    editingEnvironmentValue.value = '';
  }
  function resetEnvironmentToVersion(row: EnvRow) {
    row.value = row.inheritedValue;
    row.deleted = false;
    row.overridden = false;
    cancelEnvironmentEdit();
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
  function resetResourceToVersion(field: ResourceField) {
    field.value = field.base;
    cancelResourceEdit();
  }
  function resetResourcesToVersion() {
    resourceDeleted.value = false;
    resourceFields.value.forEach((field) => {
      field.value = field.base;
    });
    cancelResourceEdit();
  }
  const editingMount = computed(() =>
    mountRows.value.find((row) => row.target === editingMountTarget.value)
  );
  function openMountDialog(row: MountRow) {
    editingMountTarget.value = row.target;
    editingMountSource.value = row.source;
    editingMountSourceIsHostPath.value = row.source_is_host_path;
    mountSourceError.value = '';
    mountDialogOpen.value = true;
  }
  function closeMountDialog() {
    mountDialogOpen.value = false;
    editingMountTarget.value = null;
    editingMountSource.value = '';
    editingMountSourceIsHostPath.value = false;
    mountSourceError.value = '';
  }
  function setMountDialogOpen(open: boolean) {
    if (open) {
      mountDialogOpen.value = true;
      return;
    }
    closeMountDialog();
  }
  function clearMountSourceError() {
    mountSourceError.value = '';
  }
  function saveMountDialog() {
    const row = editingMount.value;
    if (!row) {
      return;
    }
    if (!editingMountSource.value.trim()) {
      mountSourceError.value = t('service.componentDetail.validation.sourceRequired');
      return;
    }
    row.source = editingMountSource.value;
    row.source_is_host_path = editingMountSourceIsHostPath.value;
    row.deleted = false;
    closeMountDialog();
  }
  function resetMountToVersion(row: MountRow) {
    row.source = row.base;
    row.source_is_host_path = row.base_source_is_host_path;
    row.deleted = false;
    closeMountDialog();
  }
  function mountHasChanges(row: MountRow) {
    return row.source !== row.base || row.source_is_host_path !== row.base_source_is_host_path;
  }
  function startEndpointEdit(row: EndpointRow) {
    editingEndpointIdentity.value = row.identity;
    editingEndpoint.value = { ...row.value };
  }
  function applyEndpointEdit(row: EndpointRow) {
    row.value = { ...editingEndpoint.value };
    row.deleted = false;
    cancelEndpointEdit();
  }
  function cancelEndpointEdit() {
    editingEndpointIdentity.value = null;
    editingEndpoint.value = emptyEndpointValues();
  }
  function endpointHasChanges(row: EndpointRow) {
    return endpointFields.some((field) => row.value[field.key] !== row.base[field.key]);
  }
  function resetEndpointToVersion(row: EndpointRow) {
    row.value = { ...row.base };
    row.deleted = false;
    cancelEndpointEdit();
  }
  function displayValue(value: string) {
    return value || '-';
  }
  const runtimeFieldDefinitions = computed<RuntimeFieldDefinition[]>(() => [
    {
      key: 'pull_policy',
      label: t('application.componentDetail.fields.pullPolicy'),
      kind: 'select',
      values: pullPolicyValues,
    },
    {
      key: 'restart_policy',
      label: t('application.componentDetail.fields.restartPolicy'),
      kind: 'select',
      values: restartPolicyValues,
    },
    {
      key: 'entrypoint',
      label: t('application.componentDetail.fields.containerEntrypoint'),
      kind: 'text',
      values: [],
    },
    {
      key: 'command',
      label: t('application.componentDetail.fields.command'),
      kind: 'text',
      values: [],
    },
  ]);
  const runtimeRows = computed<RuntimeField[]>(() =>
    runtimeFieldDefinitions.value.map((field) => ({ ...field, state: runtimeDraft[field.key] }))
  );
  function runtimeFieldDiffers(field: RuntimeField) {
    return field.state.overridden && field.state.value !== field.state.base;
  }
  function startRuntimeEdit(field: RuntimeField) {
    editingRuntimeKey.value = field.key;
    editingRuntimeValue.value = field.state.overridden ? field.state.value : field.state.base;
  }
  function cancelRuntimeEdit() {
    editingRuntimeKey.value = null;
    editingRuntimeValue.value = '';
  }
  function applyRuntimeEdit(field: RuntimeField) {
    const state = runtimeDraft[field.key];
    state.value = editingRuntimeValue.value;
    state.overridden = true;
    cancelRuntimeEdit();
  }
  function resetRuntime(field: RuntimeField) {
    const state = runtimeDraft[field.key];
    state.value = '';
    state.overridden = false;
    cancelRuntimeEdit();
  }
  function mountTypeLabel(value: string): string {
    const labels: Record<string, string> = {
      directory: t('application.componentDetail.mountTypes.directory'),
      file: t('application.componentDetail.mountTypes.file'),
      named_volume: t('application.componentDetail.mountTypes.namedVolume'),
      controlled_file: t('application.componentDetail.mountTypes.controlledFile'),
    };
    return labels[value] ?? value;
  }
  const draft = computed(() => (detail.value?.version_component ? true : false));
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
      entrypoint: runtimeDraft.entrypoint.overridden ? runtimeDraft.entrypoint.value : undefined,
      command: runtimeDraft.command.overridden ? runtimeDraft.command.value : undefined,
      pull_policy: runtimeDraft.pull_policy.overridden
        ? optional(runtimeDraft.pull_policy.value)
        : undefined,
      restart_policy: runtimeDraft.restart_policy.overridden
        ? optional(runtimeDraft.restart_policy.value)
        : undefined,
      env: environmentRows.value.flatMap((row) => {
        if (row.deleted) return [{ key: row.key, state: 'deleted' }];
        return row.overridden ? [{ key: row.key, value: row.value, state: 'override' }] : [];
      }),
      mounts: mountRows.value.map((row) =>
        row.deleted
          ? { target: row.target, state: 'deleted' }
          : {
              target: row.target,
              source: row.source,
              source_is_host_path: row.source_is_host_path,
              state: 'override',
            }
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
          ? { protocol: row.protocol, container_port: row.container_port, state: 'deleted' }
          : {
              protocol: row.protocol,
              container_port: row.container_port,
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
        const [serviceValue, value] = await Promise.all([
          serviceApi.get(serviceId),
          serviceApi.getComponent(serviceId, componentId),
        ]);
        service.value = serviceValue;
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
