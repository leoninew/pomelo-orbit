<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="min-w-0 break-words text-xl font-semibold text-foreground">
          {{ version?.label || t('application.versionDetail.title') }}
        </h1>
        <DetailHeaderMeta v-if="version">
          <AppBadge variant="status" :tone="versionStatusTone(version.status)">
            {{ version.status }}
          </AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="version && isEditable"
          class="app-button-primary h-9 px-3"
          :disabled="operating"
          @click="handlePublish"
        >
          {{ t('application.detail.actions.publish') }}
        </button>
        <button
          v-else-if="version && isPublished"
          class="app-button h-9 px-3"
          :disabled="operating"
          @click="handleUnpublish"
        >
          <RotateCcw class="size-4" />
          {{ t('application.detail.actions.unpublish') }}
        </button>
        <button
          v-if="version && isDeployable"
          class="app-button-primary h-9 px-3"
          :disabled="operating"
          @click="openDeployModal"
        >
          <Rocket class="size-4" />
          {{ t('application.detail.actions.deploy') }}
        </button>
        <button v-if="version" class="app-button h-9 px-3" @click="openPreview">
          {{ t('application.detail.actions.preview') }}
        </button>
        <button v-if="version" class="app-button h-9 px-3" @click="openForkModal">
          {{ t('application.detail.actions.fork') }}
        </button>
        <button
          v-if="version && isEditable"
          :disabled="operating"
          class="app-button-danger h-9 px-3"
          @click="isDeleteDialogOpen = true"
        >
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="goBack">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppSpinner v-if="loading" class="py-12" />

    <template v-else-if="version">
      <!-- 基本信息 -->
      <div class="app-surface">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="font-semibold text-foreground">
            {{ t('application.detail.sections.basicInfo') }}
          </h2>
          <button v-if="isEditable" class="app-button h-9 px-3" @click="openBasicModal">
            <Pencil class="size-4" />
            {{ t('common.edit') }}
          </button>
        </div>
        <dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.detail.fields.versionLabel') }}
            </dt>
            <dd class="text-foreground">{{ version.label }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('common.status') }}
            </dt>
            <dd>
              <AppBadge variant="pill" :tone="versionStatusTone(version.status)">
                {{ version.status }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.detail.fields.note') }}
            </dt>
            <dd class="text-foreground">{{ version.note === undefined ? '—' : version.note }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('common.createdAt') }}
            </dt>
            <dd class="text-muted-foreground">{{ formatTime(version.created_at) }}</dd>
          </div>
          <div class="flex gap-2 sm:col-span-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.detail.fields.envVars') }}
            </dt>
            <dd class="min-w-0 flex-1">
              <div v-if="envRows.length === 0" class="text-muted-foreground">—</div>
              <div v-else class="space-y-1 text-xs text-foreground">
                <div v-for="row in envRows" :key="row.key">{{ row.key }}={{ row.value }}</div>
              </div>
            </dd>
          </div>
        </dl>
      </div>

      <!-- 组件 -->
      <div class="app-surface">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="font-semibold text-foreground">
            {{ t('application.detail.fields.components') }}
          </h2>
          <button v-if="isEditable" class="app-button h-9 px-3" @click="openComponentPage()">
            <Plus class="size-4" />
            {{ t('application.detail.actions.addComponent') }}
          </button>
        </div>
        <AppEmptyState v-if="(version.components ?? []).length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-table-detail min-w-[640px]">
            <thead>
              <tr>
                <th>{{ t('application.detail.fields.component') }}</th>
                <th>{{ t('application.detail.fields.image') }}</th>
                <th class="w-28">{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="comp in version.components" :key="comp.id">
                <td class="text-foreground">{{ comp.name }}</td>
                <td class="max-w-xs truncate text-xs text-muted-foreground" :title="comp.image">
                  {{ comp.image }}
                </td>
                <td>
                  <button class="app-link" @click="openComponentPage(comp.id)">
                    {{ t('application.view') }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 暴露 -->
      <div class="app-surface">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="font-semibold text-foreground">
            {{ t('application.detail.fields.exposes') }}
          </h2>
          <button
            v-if="isEditable"
            class="app-button-primary h-9 px-3"
            :disabled="(version.components ?? []).length === 0"
            @click="openExposeModal()"
          >
            <Plus class="size-4" />
            {{ t('application.detail.actions.addExpose') }}
          </button>
        </div>
        <AppEmptyState v-if="(version.exposes ?? []).length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-table-detail min-w-[720px]">
            <thead>
              <tr>
                <th>{{ t('application.detail.fields.component') }}</th>
                <th>{{ t('application.detail.placeholders.exposeProtocol') }}</th>
                <th>{{ t('application.detail.placeholders.exposeAccess') }}</th>
                <th>{{ t('application.detail.placeholders.containerPort') }}</th>
                <th>{{ t('application.detail.placeholders.listenPort') }}</th>
                <th v-if="isEditable">{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, index) in version.exposes" :key="row.id || index">
                <td class="text-foreground">{{ row.component_name }}</td>
                <td>
                  <AppBadge variant="pill">{{ row.protocol }}</AppBadge>
                </td>
                <td>
                  <AppBadge variant="pill">{{ row.access }}</AppBadge>
                </td>
                <td class="text-muted-foreground">{{ row.container_port }}</td>
                <td class="text-muted-foreground">{{ row.listen_port || row.container_port }}</td>
                <td v-if="isEditable">
                  <div class="flex items-center gap-3">
                    <button class="app-link" @click="openExposeModal(index)">
                      {{ t('common.edit') }}
                    </button>
                    <button
                      class="app-link-danger"
                      :disabled="operating"
                      @click="removeExpose(index)"
                    >
                      {{ t('common.delete') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <!-- 编辑基本信息 -->
    <AppDialog
      v-model:open="isBasicDialogOpen"
      :title="t('application.versionDetail.dialog.editBasic')"
      width-class="w-[min(480px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.versionLabel') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="basicForm.label"
            type="text"
            class="app-input"
            :class="basicFormError ? 'app-input-error' : ''"
            :placeholder="t('application.detail.placeholders.versionLabel')"
          />
          <p v-if="basicFormError" class="app-field-error mt-1 text-xs">{{ basicFormError }}</p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.note') }}
          </label>
          <input
            v-model="basicForm.note"
            type="text"
            class="app-input"
            :placeholder="t('application.detail.placeholders.note')"
          />
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="app-field-label">{{ t('application.detail.fields.envVars') }}</label>
            <button
              type="button"
              class="app-link text-sm"
              @click="basicForm.env.push({ key: '', value: '' })"
            >
              {{ t('application.detail.actions.addEnv') }}
            </button>
          </div>
          <div
            v-for="(envRow, envIndex) in basicForm.env"
            :key="'venv-' + envIndex"
            class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1.2fr_auto]"
          >
            <input v-model="envRow.key" type="text" class="app-input text-xs" placeholder="KEY" />
            <input
              v-model="envRow.value"
              type="text"
              class="app-input text-xs"
              :placeholder="t('application.detail.placeholders.envValue')"
            />
            <button
              type="button"
              class="app-link-danger"
              @click="basicForm.env.splice(envIndex, 1)"
            >
              {{ t('common.delete') }}
            </button>
          </div>
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isBasicDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-primary" @click="saveBasic">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <!-- 暴露表单 -->
    <AppDialog
      v-model:open="isExposeDialogOpen"
      :title="
        editingExposeIndex === null
          ? t('application.versionDetail.dialog.addExpose')
          : t('application.versionDetail.dialog.editExpose')
      "
      width-class="w-[min(560px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.component') }}
            <span class="text-destructive">*</span>
          </label>
          <RawValueSelect
            v-model="exposeForm.component_name"
            :values="componentNameValues"
            :placeholder="t('application.detail.placeholders.exposeComponent')"
          />
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.detail.placeholders.exposeProtocol') }}
            </label>
            <RawValueSelect
              v-model="exposeForm.protocol"
              :values="exposeProtocolValues"
              :placeholder="t('application.detail.placeholders.exposeProtocol')"
            />
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.detail.placeholders.exposeAccess') }}
            </label>
            <RawValueSelect
              v-model="exposeForm.access"
              :values="exposeAccessValues"
              :placeholder="t('application.detail.placeholders.exposeAccess')"
            />
          </div>
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.detail.placeholders.containerPort') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model.number="exposeForm.container_port"
              type="number"
              min="1"
              max="65535"
              class="app-input"
            />
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.detail.placeholders.listenPort') }}
            </label>
            <input
              v-model.number="exposeForm.listen_port"
              type="number"
              min="0"
              max="65535"
              class="app-input"
            />
          </div>
        </div>
        <p v-if="exposeFormError" class="app-field-error text-xs">{{ exposeFormError }}</p>
      </div>
      <template #footer>
        <button class="app-button" @click="isExposeDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-primary" @click="saveExpose">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <!-- Fork -->
    <AppDialog
      v-model:open="isForkDialogOpen"
      :title="t('application.detail.dialog.forkVersion')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('application.detail.fields.versionLabel') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="forkLabel"
          type="text"
          class="app-input"
          :class="forkLabelError ? 'app-input-error' : ''"
          :placeholder="t('application.detail.placeholders.versionLabel')"
        />
        <p v-if="forkLabelError" class="app-field-error mt-1 text-xs">{{ forkLabelError }}</p>
      </div>
      <template #footer>
        <button class="app-button" @click="isForkDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-primary" @click="handleForkOk">
          {{ t('application.detail.actions.fork') }}
        </button>
      </template>
    </AppDialog>

    <!-- Delete -->
    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('application.detail.dialog.deleteVersion')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{
          t('application.detail.dialog.deleteVersionConfirm', {
            label: version?.label || '-',
          })
        }}
      </p>
      <template #footer>
        <button class="app-button" @click="isDeleteDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-destructive" @click="handleDeleteOk">
          {{ t('common.delete') }}
        </button>
      </template>
    </AppDialog>

    <!-- Deploy -->
    <AppDialog
      v-model:open="isDeployDialogOpen"
      :title="t('application.detail.dialog.deploy')"
      width-class="w-[min(480px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <p class="text-sm text-muted-foreground">
          {{ t('application.versionDetail.deployDescription', { label: version?.label || '-' }) }}
        </p>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('service.fields.instanceKey') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="deployForm.instance_key"
            type="text"
            required
            class="app-input"
            :placeholder="t('application.versionDetail.instanceKeyPlaceholder')"
          />
        </div>
        <label class="flex items-center gap-2">
          <input v-model="deployForm.force_recreate" type="checkbox" class="app-checkbox" />
          <span class="text-sm text-foreground">
            {{ t('application.detail.fields.forceRecreate') }}
          </span>
        </label>
      </div>
      <template #footer>
        <button class="app-button" @click="isDeployDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-primary" @click="handleDeployOk">
          {{ t('application.detail.actions.deploy') }}
        </button>
      </template>
    </AppDialog>

    <!-- Compose 预览 -->
    <AppDrawer
      :open="composePreviewDrawerOpen"
      :title="t('application.detail.drawer.composePreview')"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="handleComposePreviewDrawerOpenChange"
    >
      <div class="flex h-full flex-col gap-3 p-6">
        <p class="text-sm text-muted-foreground">
          {{ t('application.detail.drawer.composePreviewDescription') }}
        </p>
        <div v-if="composePreviewLoading" class="flex flex-1 items-center justify-center">
          <AppSpinner />
        </div>
        <div
          v-else-if="composePreviewError"
          class="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
        >
          {{ composePreviewError }}
        </div>
        <div v-else class="min-h-0 flex-1">
          <MonacoEditor
            :model-value="composePreviewYaml"
            language="yaml"
            height="100%"
            :readonly="true"
          />
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="composePreviewDrawerOpen = false">
          {{ t('application.detail.actions.close') }}
        </button>
      </template>
    </AppDrawer>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Pencil, Plus, Rocket, RotateCcw, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { VersionExposeReq, VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import { versionStatusTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';
  import {
    parseVersionEnvJson,
    serializeVersionEnvRows,
    type VersionEnvFormRow,
  } from '@/utils/versionEnvForm';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const versionId = route.params.id as string;

  const { loading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { loading: composePreviewLoading, execute: executeComposePreview } = useStatusAsync();

  const version = ref<VersionResp>();

  const isBasicDialogOpen = ref(false);
  const isExposeDialogOpen = ref(false);
  const isForkDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isDeployDialogOpen = ref(false);
  const composePreviewDrawerOpen = ref(false);
  const composePreviewYaml = ref('');
  const composePreviewError = ref('');
  const editingExposeIndex = ref<number | null>(null);
  const forkLabel = ref('');
  const forkLabelError = ref('');
  const basicFormError = ref('');
  const exposeFormError = ref('');
  const deployError = ref('');
  const deployForm = reactive({
    instance_key: 'default',
    force_recreate: false,
  });

  const basicForm = reactive({
    label: '',
    note: '',
    env: [] as VersionEnvFormRow[],
  });
  const exposeForm = reactive({
    component_name: '',
    protocol: '',
    container_port: 0,
    access: '',
    listen_port: 0,
  });

  const exposeProtocolValues = ['http', 'tcp'];
  const exposeAccessValues = ['local', 'public'];
  const isEditable = computed(() => version.value?.status === 'unpublished');
  const isPublished = computed(() => version.value?.status === 'published');
  const isDeployable = computed(() => Boolean(version.value));
  const hasComponents = computed(() => (version.value?.components ?? []).length > 0);
  const envRows = computed(() => parseVersionEnvJson(version.value?.env_json));
  const componentNameValues = computed(() => (version.value?.components ?? []).map((c) => c.name));

  function exposesPayload(): VersionExposeReq[] {
    return (version.value?.exposes ?? []).map((item) => ({
      component_name: item.component_name,
      protocol: item.protocol,
      container_port: item.container_port,
      path_prefix: item.path_prefix,
      access: item.access,
      listen_port: item.listen_port,
    }));
  }

  async function fetchVersion() {
    try {
      await execute(async () => {
        version.value = await applicationApi.getVersion(versionId);
      });
    } catch {
      toast.error(t('application.toast.loadVersionsFailed'));
      router.push('/versions');
    }
  }

  function goBack() {
    const appId = version.value?.application_id;
    if (appId) {
      router.push(`/application/${appId}`);
      return;
    }
    router.push('/applications');
  }

  function openBasicModal() {
    if (!version.value) {
      return;
    }
    basicForm.label = version.value.label;
    basicForm.note = version.value.note === undefined ? '' : version.value.note;
    try {
      basicForm.env = parseVersionEnvJson(version.value.env_json);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
      return;
    }
    basicFormError.value = '';
    isBasicDialogOpen.value = true;
  }

  async function saveBasic() {
    basicFormError.value =
      basicForm.label === '' ? t('application.validation.versionLabelRequired') : '';
    if (basicFormError.value) {
      return;
    }
    try {
      await executeOp(async () => {
        version.value = await applicationApi.updateVersion(versionId, {
          label: basicForm.label,
          note: basicForm.note,
          env_json: serializeVersionEnvRows(basicForm.env),
          exposes: exposesPayload(),
        });
        toast.success(t('application.toast.updateSuccess'));
        isBasicDialogOpen.value = false;
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  function openComponentPage(id = 'new') {
    router.push(`/version/${versionId}/component/${id}`);
  }

  function openExposeModal(index?: number) {
    if ((version.value?.components ?? []).length === 0) {
      toast.error(t('application.validation.exposeNeedsComponent'));
      return;
    }
    editingExposeIndex.value = index ?? null;
    exposeFormError.value = '';
    if (index === undefined || !version.value?.exposes?.[index]) {
      exposeForm.component_name = '';
      exposeForm.protocol = '';
      exposeForm.container_port = 0;
      exposeForm.access = '';
      exposeForm.listen_port = 0;
    } else {
      const row = version.value.exposes[index];
      exposeForm.component_name = row.component_name;
      exposeForm.protocol = row.protocol;
      exposeForm.container_port = row.container_port;
      exposeForm.access = row.access;
      exposeForm.listen_port = row.listen_port === undefined ? 0 : row.listen_port;
    }
    isExposeDialogOpen.value = true;
  }

  async function saveExpose() {
    const componentName = exposeForm.component_name;
    const containerPort = Number(exposeForm.container_port);
    if (!componentName) {
      exposeFormError.value = t('application.validation.exposeComponentRequired');
      return;
    }
    if (!(version.value?.components ?? []).some((c) => c.name === componentName)) {
      exposeFormError.value = t('application.validation.exposeComponentNotFound');
      return;
    }
    if (
      (exposeForm.protocol !== 'http' && exposeForm.protocol !== 'tcp') ||
      (exposeForm.access !== 'local' && exposeForm.access !== 'public')
    ) {
      exposeFormError.value = t('application.validation.exposeFieldsInvalid');
      return;
    }
    if (!Number.isInteger(containerPort) || containerPort < 1 || containerPort > 65535) {
      exposeFormError.value = t('application.validation.portRange');
      return;
    }
    const listen = Number(exposeForm.listen_port);
    if (!Number.isInteger(listen) || (listen !== 0 && (listen < 1 || listen > 65535))) {
      exposeFormError.value = t('application.validation.portRange');
      return;
    }
    const req: VersionExposeReq = {
      component_name: componentName,
      protocol: exposeForm.protocol,
      container_port: containerPort,
      access: exposeForm.access,
      listen_port: listen > 0 ? listen : undefined,
    };
    const next = exposesPayload();
    if (editingExposeIndex.value === null) {
      next.push(req);
    } else {
      next[editingExposeIndex.value] = req;
    }
    try {
      await executeOp(async () => {
        version.value = await applicationApi.updateVersion(versionId, {
          exposes: next,
        });
        toast.success(t('application.toast.updateSuccess'));
        isExposeDialogOpen.value = false;
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  async function removeExpose(index: number) {
    const next = exposesPayload().filter((_, i) => i !== index);
    try {
      await executeOp(async () => {
        version.value = await applicationApi.updateVersion(versionId, {
          exposes: next,
        });
        toast.success(t('application.toast.updateSuccess'));
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  async function handlePublish() {
    try {
      await executeOp(async () => {
        version.value = await applicationApi.publishVersion(versionId);
        toast.success(t('application.toast.publishSuccess'));
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.publishFailed'));
    }
  }

  async function handleUnpublish() {
    try {
      await executeOp(async () => {
        version.value = await applicationApi.unpublishVersion(versionId);
        toast.success(t('application.toast.unpublishSuccess'));
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.unpublishFailed'));
    }
  }

  function openForkModal() {
    forkLabel.value = version.value ? `${version.value.label}-copy` : '';
    forkLabelError.value = '';
    isForkDialogOpen.value = true;
  }

  async function handleForkOk() {
    forkLabelError.value = forkLabel.value.trim()
      ? ''
      : t('application.validation.versionLabelRequired');
    if (forkLabelError.value) {
      return;
    }
    try {
      await executeOp(async () => {
        const created = await applicationApi.forkVersion(versionId, {
          label: forkLabel.value.trim(),
        });
        toast.success(t('application.toast.forkSuccess'));
        isForkDialogOpen.value = false;
        router.push(`/version/${created.id}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.forkFailed'));
    }
  }

  async function handleDeleteOk() {
    try {
      await executeOp(async () => {
        await applicationApi.deleteVersion(versionId);
        toast.success(t('application.toast.deleteVersionSuccess'));
        isDeleteDialogOpen.value = false;
        goBack();
      });
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('application.toast.deleteVersionFailed')
      );
    }
  }

  async function openPreview() {
    composePreviewYaml.value = '';
    composePreviewError.value = '';
    composePreviewDrawerOpen.value = true;
    try {
      await executeComposePreview(async () => {
        const { compose_yaml } = await applicationApi.previewVersion(versionId, {
          instance_key: 'default',
        });
        composePreviewYaml.value = compose_yaml;
      });
    } catch (error) {
      composePreviewError.value =
        error instanceof Error ? error.message : t('application.toast.loadPreviewFailed');
    }
  }

  function openDeployModal() {
    if (!version.value || !isDeployable.value) {
      return;
    }
    if (!hasComponents.value) {
      toast.error(t('application.validation.componentRequired'));
      return;
    }
    deployError.value = '';
    deployForm.force_recreate = false;
    deployForm.instance_key = 'default';
    isDeployDialogOpen.value = true;
  }

  async function handleDeployOk() {
    const current = version.value;
    if (!current) {
      return;
    }
    if (!hasComponents.value) {
      toast.error(t('application.validation.componentRequired'));
      isDeployDialogOpen.value = false;
      return;
    }
    deployError.value = '';
    try {
      await executeOp(async () => {
        const result = await applicationApi.deploy(current.application_id, {
          version_id: current.id,
          instance_key: deployForm.instance_key,
          force_recreate: deployForm.force_recreate,
        });
        toast.success(t('application.toast.deployTriggeredDetail'));
        isDeployDialogOpen.value = false;
        if (result.deployment_id) {
          router.push(`/deployment/${result.deployment_id}`);
        }
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.deployFailed'));
    }
  }

  function handleComposePreviewDrawerOpenChange(open: boolean) {
    composePreviewDrawerOpen.value = open;
    if (!open) {
      composePreviewYaml.value = '';
      composePreviewError.value = '';
    }
  }

  onMounted(async () => {
    await fetchVersion();
  });
</script>
