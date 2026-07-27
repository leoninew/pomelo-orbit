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
          <button v-if="isEditable" class="app-button h-8 px-3" @click="openBasicModal">
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
            <dd class="text-foreground">{{ version.note || '—' }}</dd>
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
          <button
            v-if="isEditable"
            class="app-button-primary h-8 px-3"
            @click="openComponentModal()"
          >
            <Plus class="size-4" />
            {{ t('application.detail.actions.addComponent') }}
          </button>
        </div>
        <div class="overflow-x-auto">
          <table class="app-table-detail min-w-[960px]">
            <thead>
              <tr>
                <th>{{ t('application.detail.fields.component') }}</th>
                <th>{{ t('application.detail.fields.image') }}</th>
                <th>{{ t('application.detail.fields.ports') }}</th>
                <th>{{ t('application.detail.fields.runtimeConfig') }}</th>
                <th v-if="isEditable">{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="(version.components ?? []).length === 0">
                <td :colspan="isEditable ? 5 : 4" class="text-center text-muted-foreground">
                  {{ t('application.versionDetail.empty.components') }}
                </td>
              </tr>
              <tr v-for="(comp, index) in version.components" :key="comp.id || index">
                <td class="text-foreground">{{ comp.name }}</td>
                <td class="max-w-xs truncate text-xs text-muted-foreground" :title="comp.image">
                  {{ comp.image }}
                </td>
                <td class="text-muted-foreground">{{ portsSummary(comp.ports_json) }}</td>
                <td>
                  <button class="app-link text-left" @click="openComponentRuntimeDialog(index)">
                    {{ runtimeSummary(comp) }}
                  </button>
                </td>
                <td v-if="isEditable">
                  <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
                    <button class="app-link" @click="openComponentBaseDialog(index)">
                      {{ t('application.versionDetail.actions.componentBase') }}
                    </button>
                    <button class="app-link" @click="openComponentPortsDialog(index)">
                      {{ t('application.versionDetail.actions.componentPorts') }}
                    </button>
                    <button class="app-link" @click="openComponentEnvDialog(index)">
                      {{ t('application.versionDetail.actions.componentEnv') }}
                    </button>
                    <button class="app-link" @click="openComponentMountsDialog(index)">
                      {{ t('application.versionDetail.actions.componentMounts') }}
                    </button>
                    <DropdownMenuRoot>
                      <DropdownMenuTrigger
                        class="app-link inline-flex items-center gap-1"
                        :aria-label="t('application.versionDetail.actions.componentMore')"
                      >
                        {{ t('application.versionDetail.actions.componentMore') }}
                        <ChevronDown class="size-3" />
                      </DropdownMenuTrigger>
                      <DropdownMenuPortal>
                        <DropdownMenuContent
                          class="z-50 min-w-28 rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-lg outline-none"
                          align="end"
                          :side-offset="6"
                        >
                          <DropdownMenuItem
                            class="flex cursor-pointer rounded px-3 py-2 text-sm text-destructive outline-none transition-colors data-[highlighted]:bg-destructive/10"
                            :disabled="operating"
                            @select="openDeleteComponentDialog(index)"
                          >
                            {{ t('common.delete') }}
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenuPortal>
                    </DropdownMenuRoot>
                  </div>
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
            class="app-button-primary h-8 px-3"
            :disabled="(version.components ?? []).length === 0"
            @click="openExposeModal()"
          >
            <Plus class="size-4" />
            {{ t('application.detail.actions.addExpose') }}
          </button>
        </div>
        <div class="overflow-x-auto">
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
              <tr v-if="(version.exposes ?? []).length === 0">
                <td :colspan="isEditable ? 6 : 5" class="text-center text-muted-foreground">
                  {{ t('application.versionDetail.empty.exposes') }}
                </td>
              </tr>
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

    <VersionComponentBaseDialog
      v-model:open="isComponentBaseDialogOpen"
      :component="editingComponent"
      :component-names="componentNames"
      :saving="operating"
      @save="saveComponentBase"
    />

    <VersionComponentPortsDialog
      v-model:open="isComponentPortsDialogOpen"
      :ports-json="editingComponent?.ports_json"
      :saving="operating"
      @save="saveComponentPorts"
    />

    <VersionComponentEnvDialog
      v-model:open="isComponentEnvDialogOpen"
      :env-json="editingComponent?.env_json"
      :saving="operating"
      @save="saveComponentEnv"
    />

    <VersionComponentMountsDialog
      v-model:open="isComponentMountsDialogOpen"
      :mounts-json="editingComponent?.mounts_json"
      :saving="operating"
      @save="saveComponentMounts"
    />

    <VersionComponentRuntimeDialog
      v-model:open="isComponentRuntimeDialogOpen"
      :component="editingComponent"
      :readonly="!isEditable"
      :saving="operating"
      @save="saveComponentRuntime"
    />

    <VersionComponentDeleteDialog
      v-model:open="isDeleteComponentDialogOpen"
      :component-name="pendingDeleteComponent?.name"
      :saving="operating"
      @confirm="confirmDeleteComponent"
    />

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
          </label>
          <ComboboxSelect
            :model-value="deployForm.service_id"
            :options="deployServiceOptions"
            :placeholder="t('service.empty')"
            width-class="w-full"
            @update:model-value="deployForm.service_id = String($event || '')"
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
  import { ArrowLeft, ChevronDown, Pencil, Plus, Rocket, RotateCcw, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import VersionComponentBaseDialog from '@/views/application/components/VersionComponentBaseDialog.vue';
  import VersionComponentDeleteDialog from '@/views/application/components/VersionComponentDeleteDialog.vue';
  import VersionComponentEnvDialog from '@/views/application/components/VersionComponentEnvDialog.vue';
  import VersionComponentMountsDialog from '@/views/application/components/VersionComponentMountsDialog.vue';
  import VersionComponentPortsDialog from '@/views/application/components/VersionComponentPortsDialog.vue';
  import VersionComponentRuntimeDialog from '@/views/application/components/VersionComponentRuntimeDialog.vue';
  import {
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuPortal,
    DropdownMenuRoot,
    DropdownMenuTrigger,
  } from 'reka-ui';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type {
    VersionComponentReq,
    VersionComponentResp,
    VersionExposeReq,
    VersionResp,
  } from '@/gen/proto/orbit/v1/application/version';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
  import { versionStatusTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';
  import {
    parseEnvJson,
    parsePortsJson,
    serializeEnvRows,
    type EnvFormRow,
  } from '@/utils/versionComponentForm';

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
  const isComponentBaseDialogOpen = ref(false);
  const isComponentPortsDialogOpen = ref(false);
  const isComponentEnvDialogOpen = ref(false);
  const isComponentMountsDialogOpen = ref(false);
  const isComponentRuntimeDialogOpen = ref(false);
  const isDeleteComponentDialogOpen = ref(false);
  const isExposeDialogOpen = ref(false);
  const isForkDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isDeployDialogOpen = ref(false);
  const composePreviewDrawerOpen = ref(false);
  const composePreviewYaml = ref('');
  const composePreviewError = ref('');
  const editingComponentIndex = ref<number | null>(null);
  const editingExposeIndex = ref<number | null>(null);
  const pendingDeleteComponentIndex = ref<number | null>(null);
  const forkLabel = ref('');
  const forkLabelError = ref('');
  const basicFormError = ref('');
  const exposeFormError = ref('');
  const deployError = ref('');
  const deployForm = reactive({
    service_id: '',
    force_recreate: false,
  });
  const deployServices = ref<ServiceResp[]>([]);
  const deployServiceOptions = computed(() =>
    deployServices.value.map((service) => ({
      value: service.id,
      label: service.instance_key || 'default',
      description: service.version_label || service.version_id,
    }))
  );

  const basicForm = reactive({
    label: '',
    note: '',
    env: [] as EnvFormRow[],
  });
  const exposeForm = reactive({
    component_name: '',
    protocol: 'http',
    container_port: 80,
    access: 'local',
    listen_port: 0,
  });

  const exposeProtocolValues = ['http', 'tcp'];
  const exposeAccessValues = ['local', 'public'];
  const isEditable = computed(() => version.value?.status === 'unpublished');
  const isPublished = computed(() => version.value?.status === 'published');
  const isDeployable = computed(() => Boolean(version.value));
  const hasComponents = computed(() => (version.value?.components ?? []).length > 0);
  const envRows = computed(() => parseEnvJson(version.value?.env_json));
  const componentNameValues = computed(() => (version.value?.components ?? []).map((c) => c.name));
  const componentNames = computed(() =>
    (version.value?.components ?? []).map((component) => component.name)
  );
  const editingComponent = computed(() => {
    const index = editingComponentIndex.value;
    return index === null ? undefined : version.value?.components?.[index];
  });
  const pendingDeleteComponent = computed(() => {
    const index = pendingDeleteComponentIndex.value;
    return index === null ? undefined : version.value?.components?.[index];
  });

  function portsSummary(raw?: string) {
    const rows = parsePortsJson(raw);
    if (rows.length === 0) {
      return '—';
    }
    return rows.map((r) => `${r.host_port}:${r.container_port}`).join(', ');
  }

  function runtimeSummary(component: VersionComponentResp) {
    const parts: string[] = [];
    if (component.command_json) parts.push(t('application.runtime.summary.command'));
    if (component.args_json) parts.push(t('application.runtime.summary.args'));
    if (component.healthcheck_json) parts.push(t('application.runtime.summary.healthcheck'));
    if (component.resources_json) parts.push(t('application.runtime.summary.resources'));
    if (component.restart_policy) parts.push(t('application.runtime.summary.restart'));
    if (component.tmpfs_json) parts.push(t('application.runtime.summary.tmpfs'));
    if (component.ulimits_json) parts.push(t('application.runtime.summary.ulimits'));
    return parts.length > 0 ? parts.join(' · ') : t('application.runtime.summary.empty');
  }

  function componentPayload(component: VersionComponentResp): VersionComponentReq {
    return {
      name: component.name,
      image: component.image,
      command_json: component.command_json,
      args_json: component.args_json,
      env_json: component.env_json,
      ports_json: component.ports_json,
      mounts_json: component.mounts_json,
      networks_json: component.networks_json,
      depends_on_json: component.depends_on_json,
      healthcheck_json: component.healthcheck_json,
      resources_json: component.resources_json,
      pull_policy: component.pull_policy,
      restart_policy: component.restart_policy,
      tmpfs_json: component.tmpfs_json,
      ulimits_json: component.ulimits_json,
    };
  }

  function componentsPayload(): VersionComponentReq[] {
    return (version.value?.components ?? []).map(componentPayload);
  }

  function replaceComponent(
    index: number,
    patch: Partial<VersionComponentReq>
  ): VersionComponentReq[] | undefined {
    const next = componentsPayload();
    if (!next[index]) {
      return undefined;
    }
    next[index] = { ...next[index], ...patch };
    return next;
  }

  function renameDependencyReferences(
    components: VersionComponentReq[],
    oldName: string,
    newName: string
  ): VersionComponentReq[] {
    if (oldName === newName) {
      return components;
    }
    return components.map((component) => {
      if (!component.depends_on_json?.trim()) {
        return component;
      }
      try {
        const dependencies = JSON.parse(component.depends_on_json) as unknown;
        if (
          !Array.isArray(dependencies) ||
          !dependencies.every((item) => typeof item === 'string')
        ) {
          return component;
        }
        return {
          ...component,
          depends_on_json: JSON.stringify(
            dependencies.map((dependency) => (dependency === oldName ? newName : dependency))
          ),
        };
      } catch {
        return component;
      }
    });
  }

  function exposesPayload(): VersionExposeReq[] {
    return (version.value?.exposes ?? []).map((item) => ({
      component_name: item.component_name,
      protocol: item.protocol || 'http',
      container_port: item.container_port,
      path_prefix: item.path_prefix,
      access: item.access || 'public',
      listen_port: item.listen_port || undefined,
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
    basicForm.note = version.value.note || '';
    basicForm.env = parseEnvJson(version.value.env_json);
    basicFormError.value = '';
    isBasicDialogOpen.value = true;
  }

  async function saveBasic() {
    basicFormError.value = basicForm.label.trim()
      ? ''
      : t('application.validation.versionLabelRequired');
    if (basicFormError.value) {
      return;
    }
    try {
      await executeOp(async () => {
        version.value = await applicationApi.updateVersion(versionId, {
          label: basicForm.label.trim(),
          note: basicForm.note.trim(),
          env_json: serializeEnvRows(basicForm.env) ?? '',
          components: componentsPayload(),
          exposes: exposesPayload(),
        });
        toast.success(t('application.toast.updateSuccess'));
        isBasicDialogOpen.value = false;
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  function openComponentModal() {
    editingComponentIndex.value = null;
    isComponentBaseDialogOpen.value = true;
  }

  function openComponentBaseDialog(index: number) {
    const component = version.value?.components?.[index];
    if (!component) {
      return;
    }
    editingComponentIndex.value = index;
    isComponentBaseDialogOpen.value = true;
  }

  function openComponentPortsDialog(index: number) {
    const component = version.value?.components?.[index];
    if (!component) {
      return;
    }
    editingComponentIndex.value = index;
    isComponentPortsDialogOpen.value = true;
  }

  function openComponentEnvDialog(index: number) {
    const component = version.value?.components?.[index];
    if (!component) {
      return;
    }
    editingComponentIndex.value = index;
    isComponentEnvDialogOpen.value = true;
  }

  function openComponentMountsDialog(index: number) {
    const component = version.value?.components?.[index];
    if (!component) {
      return;
    }
    editingComponentIndex.value = index;
    isComponentMountsDialogOpen.value = true;
  }

  function openComponentRuntimeDialog(index: number) {
    const component = version.value?.components?.[index];
    if (!component) {
      return;
    }
    editingComponentIndex.value = index;
	 isComponentRuntimeDialogOpen.value = true;
  }

  async function updateComponents(next: VersionComponentReq[], exposes = exposesPayload()) {
    await executeOp(async () => {
      version.value = await applicationApi.updateVersion(versionId, { components: next, exposes });
      toast.success(t('application.toast.updateSuccess'));
    });
  }

  async function saveComponentBase({ name, image }: Pick<VersionComponentReq, 'name' | 'image'>) {
    let next = componentsPayload();
    const editingIndex = editingComponentIndex.value;
    if (editingIndex === null) {
      next.push({ name, image });
    } else {
      const oldName = version.value?.components?.[editingIndex]?.name;
      if (!oldName) {
        return;
      }
      next = replaceComponent(editingIndex, { name, image }) ?? next;
      next = renameDependencyReferences(next, oldName, name);
      const exposes = exposesPayload().map((expose) =>
        expose.component_name === oldName ? { ...expose, component_name: name } : expose
      );
      try {
        await updateComponents(next, exposes);
        isComponentBaseDialogOpen.value = false;
      } catch (error) {
        toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
      }
      return;
    }

    try {
      await updateComponents(next);
      isComponentBaseDialogOpen.value = false;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  async function saveComponentPorts(portsJson: string | undefined) {
    const index = editingComponentIndex.value;
    if (index === null) {
      return;
    }
    const next = replaceComponent(index, { ports_json: portsJson });
    if (!next) {
      return;
    }
    try {
      await updateComponents(next);
      isComponentPortsDialogOpen.value = false;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  async function saveComponentEnv(envJson: string | undefined) {
    const index = editingComponentIndex.value;
    if (index === null) {
      return;
    }
    const next = replaceComponent(index, { env_json: envJson });
    if (!next) {
      return;
    }
    try {
      await updateComponents(next);
      isComponentEnvDialogOpen.value = false;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  async function saveComponentMounts(mountsJson: string | undefined) {
    const index = editingComponentIndex.value;
    if (index === null) {
      return;
    }
    const next = replaceComponent(index, { mounts_json: mountsJson });
    if (!next) {
      return;
    }
    try {
      await updateComponents(next);
      isComponentMountsDialogOpen.value = false;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  async function saveComponentRuntime(
    config: Pick<
      VersionComponentReq,
      | 'command_json'
      | 'args_json'
      | 'healthcheck_json'
      | 'resources_json'
      | 'restart_policy'
      | 'tmpfs_json'
      | 'ulimits_json'
    >
  ) {
    const index = editingComponentIndex.value;
    if (index === null) {
      return;
    }
    const next = replaceComponent(index, config);
    if (!next) {
      return;
    }
    try {
      await updateComponents(next);
      isComponentRuntimeDialogOpen.value = false;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  function openDeleteComponentDialog(index: number) {
    if (!version.value?.components?.[index]) {
      return;
    }
    pendingDeleteComponentIndex.value = index;
    isDeleteComponentDialogOpen.value = true;
  }

  async function confirmDeleteComponent() {
    const index = pendingDeleteComponentIndex.value;
    const name = pendingDeleteComponent.value?.name;
    if (index === null || !name) {
      return;
    }
    if ((version.value?.exposes ?? []).some((item) => item.component_name === name)) {
      toast.error(t('application.versionDetail.validation.componentHasExposes'));
      return;
    }
    try {
      await updateComponents(
        componentsPayload().filter((_, componentIndex) => componentIndex !== index)
      );
      isDeleteComponentDialogOpen.value = false;
      pendingDeleteComponentIndex.value = null;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  function openExposeModal(index?: number) {
    if ((version.value?.components ?? []).length === 0) {
      toast.error(t('application.validation.exposeNeedsComponent'));
      return;
    }
    editingExposeIndex.value = index ?? null;
    exposeFormError.value = '';
    if (index === undefined || !version.value?.exposes?.[index]) {
      exposeForm.component_name = version.value?.components?.[0]?.name || '';
      exposeForm.protocol = 'http';
      exposeForm.container_port = 80;
      exposeForm.access = 'local';
      exposeForm.listen_port = 0;
    } else {
      const row = version.value.exposes[index];
      exposeForm.component_name = row.component_name;
      exposeForm.protocol = row.protocol || 'http';
      exposeForm.container_port = row.container_port;
      exposeForm.access = row.access || 'public';
      exposeForm.listen_port = row.listen_port || 0;
    }
    isExposeDialogOpen.value = true;
  }

  async function saveExpose() {
    const componentName = exposeForm.component_name.trim();
    const containerPort = Number(exposeForm.container_port);
    if (!componentName) {
      exposeFormError.value = t('application.validation.exposeComponentRequired');
      return;
    }
    if (!(version.value?.components ?? []).some((c) => c.name === componentName)) {
      exposeFormError.value = t('application.validation.exposeComponentNotFound');
      return;
    }
    if (!Number.isInteger(containerPort) || containerPort < 1 || containerPort > 65535) {
      exposeFormError.value = t('application.validation.portRange');
      return;
    }
    const listen = Number(exposeForm.listen_port) || 0;
    if (listen !== 0 && (listen < 1 || listen > 65535)) {
      exposeFormError.value = t('application.validation.portRange');
      return;
    }
    const req: VersionExposeReq = {
      component_name: componentName,
      protocol: exposeForm.protocol || 'http',
      container_port: containerPort,
      access: exposeForm.access || 'public',
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
          components: componentsPayload(),
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
          components: componentsPayload(),
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
    forkLabel.value = `${version.value?.label || 'v'}-copy`;
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

  async function openDeployModal() {
    if (!version.value || !isDeployable.value) {
      return;
    }
    if (!hasComponents.value) {
      toast.error(t('application.validation.componentRequired'));
      return;
    }
    deployError.value = '';
    deployForm.force_recreate = false;
    try {
      const services = await applicationApi.listServices(version.value.application_id);
      deployServices.value = services.items ?? [];
      deployForm.service_id = deployServices.value[0]?.id ?? '';
      if (!deployForm.service_id) {
        deployError.value = t('service.empty');
      }
      isDeployDialogOpen.value = true;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.deployFailed'));
    }
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
	if (!deployForm.service_id) {
		deployError.value = t('service.empty');
		return;
	}
    deployError.value = '';
    try {
      await executeOp(async () => {
        const result = await applicationApi.deploy(current.application_id, {
          version_id: current.id,
          service_id: deployForm.service_id,
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
