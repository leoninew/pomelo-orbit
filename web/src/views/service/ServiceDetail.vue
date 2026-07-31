<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">{{ t('service.detail.title') }}</h1>
        <DetailHeaderMeta v-if="service">
          <AppBadge variant="status" :tone="appStatusTone(service.status)">
            {{ service.status }}
          </AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="service"
          class="app-button h-9 px-3"
          :disabled="operating"
          @click="openPreview"
        >
          <FileCode2 class="size-4" />
          {{ t('application.detail.actions.preview') }}
        </button>
        <button
          v-if="service"
          class="app-button-primary h-9 px-3"
          :disabled="operating || isDeploying"
          @click="openDeployDialog"
        >
          <Rocket class="size-4" />
          {{ t('service.actions.deploy') }}
        </button>
        <button
          v-if="service"
          class="app-button-danger h-9 px-3"
          :disabled="operating || !canStop"
          @click="isStopDialogOpen = true"
        >
          <Square class="size-4" />
          {{ t('service.actions.stop') }}
        </button>
        <button
          v-if="service"
          class="app-button h-9 px-3"
          :disabled="operating"
          @click="openLogsDrawer"
        >
          <ScrollText class="size-4" />
          {{ t('service.actions.logsAll') }}
        </button>
        <button
          v-if="canDelete"
          class="app-button-danger h-9 px-3"
          :disabled="operating"
          @click="isDeleteDialogOpen = true"
        >
          <Trash2 class="size-4" />
          {{ t('service.actions.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/services')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppSpinner v-if="loading && !service" class="py-12" />
    <AppEmptyState v-else-if="!service" :message="t('service.detail.notFound')" />

    <template v-else>
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">{{ t('service.detail.sections.basic') }}</h2>
          <button
            class="app-button-primary h-9 px-3"
            :disabled="operating"
            @click="openBasicDialog"
          >
            <Pencil class="size-4" />
            {{ t('common.edit') }}
          </button>
        </div>
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>{{ t('service.fields.application') }}</dt>
            <dd>
              <router-link :to="`/application/${service.application_id}`" class="app-link">
                {{ service.application_name || service.application_id }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('service.fields.instanceKey') }}</dt>
            <dd class="text-foreground">{{ service.instance_key }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('service.fields.version') }}</dt>
            <dd>
              <router-link :to="`/version/${service.version_id}`" class="app-link">
                {{ service.version_label || service.version_id }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.status') }}</dt>
            <dd>
              <AppBadge variant="status" :tone="appStatusTone(service.status)">
                {{ service.status }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.updatedAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(service.updated_at) }}</dd>
          </div>
        </dl>
      </div>

      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">{{ t('service.runtimeConfig.title') }}</h2>
          <button
            class="app-button-primary h-9 px-3"
            :disabled="operating"
            @click="openRuntimeConfigDialog()"
          >
            <Plus class="size-4" />
            {{ t('common.add') }}
          </button>
        </div>

        <AppEmptyState v-if="runtimeConfigEntries.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[640px] table-fixed">
            <colgroup>
              <col class="w-[36%]" />
              <col />
              <col class="w-24" />
            </colgroup>
            <thead>
              <tr>
                <th>{{ t('service.runtimeConfig.key') }}</th>
                <th>{{ t('service.runtimeConfig.value') }}</th>
                <th class="w-24">{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="entry in runtimeConfigEntries" :key="entry.key">
                <td class="break-all text-foreground">{{ entry.key }}</td>
                <td>
                  <SensitiveValue
                    class="w-full"
                    :value="entry.value"
                    :label="t('service.runtimeConfig.value')"
                    :show-label="t('service.runtimeConfig.showValue')"
                    :hide-label="t('service.runtimeConfig.hideValue')"
                  />
                </td>
                <td class="w-24">
                  <div class="flex items-center gap-1">
                    <button
                      class="app-icon-button"
                      :aria-label="t('common.edit')"
                      :disabled="operating"
                      :title="t('common.edit')"
                      @click="openRuntimeConfigDialog(entry.key)"
                    >
                      <Pencil class="size-4" />
                    </button>
                    <button
                      class="app-icon-button"
                      :aria-label="t('common.delete')"
                      :disabled="operating"
                      :title="t('common.delete')"
                      @click="removeRuntimeConfig(entry.key)"
                    >
                      <Trash2 class="size-4" />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">{{ t('service.exposes.title') }}</h2>
          <button
            class="app-button-primary h-9 px-3"
            :disabled="operating || selectedComponents.length === 0"
            @click="openExposeDialog()"
          >
            <Plus class="size-4" />
            {{ t('application.detail.actions.addExpose') }}
          </button>
        </div>

        <AppEmptyState v-if="service.exposes.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[760px]">
            <thead>
              <tr>
                <th>{{ t('application.detail.fields.component') }}</th>
                <th>{{ t('application.detail.placeholders.exposeProtocol') }}</th>
                <th>{{ t('application.detail.placeholders.exposeAccess') }}</th>
                <th>{{ t('application.detail.placeholders.containerPort') }}</th>
                <th>{{ t('application.detail.placeholders.listenPort') }}</th>
                <th class="w-24">{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(expose, index) in service.exposes" :key="expose.id">
                <td>{{ expose.component_name }}</td>
                <td>{{ expose.protocol }}</td>
                <td>{{ expose.access }}</td>
                <td>{{ expose.container_port }}</td>
                <td>{{ expose.listen_port || expose.container_port }}</td>
                <td class="w-24">
                  <div class="flex items-center gap-1">
                    <button
                      class="app-icon-button"
                      :aria-label="t('common.edit')"
                      :disabled="operating"
                      :title="t('common.edit')"
                      @click="openExposeDialog(index)"
                    >
                      <Pencil class="size-4" />
                    </button>
                    <button
                      class="app-icon-button"
                      :aria-label="t('common.delete')"
                      :disabled="operating"
                      :title="t('common.delete')"
                      @click="removeExpose(index)"
                    >
                      <Trash2 class="size-4" />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">{{ t('service.detail.sections.components') }}</h2>
          <button class="app-button h-9 px-3" :disabled="containersLoading" @click="loadContainers">
            <RefreshCw class="size-4" :class="{ 'animate-spin': containersLoading }" />
            {{ t('common.refresh') }}
          </button>
        </div>
        <AppSpinner v-if="containersLoading && containers.length === 0" class="py-8" />
        <p v-else-if="containersError" class="px-5 py-4 text-sm text-destructive">
          {{ containersError }}
        </p>
        <AppEmptyState v-else-if="containers.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[720px]">
            <thead>
              <tr>
                <th>{{ t('service.fields.component') }}</th>
                <th>{{ t('common.status') }}</th>
                <th>{{ t('service.fields.health') }}</th>
                <th>{{ t('service.fields.image') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="container in containers" :key="container.id || container.name">
                <td>{{ container.service }}</td>
                <td>
                  <AppBadge
                    v-if="container.state"
                    variant="pill"
                    :tone="containerStateTone(container.state)"
                  >
                    {{ container.state }}
                  </AppBadge>
                </td>
                <td>{{ container.health }}</td>
                <td class="max-w-sm truncate" :title="container.image">{{ container.image }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <AppDialog
      v-if="service"
      v-model:open="isBasicDialogOpen"
      :title="t('service.detail.dialog.editBasic')"
    >
      <div>
        <label class="app-field-label mb-1.5 block">{{ t('service.fields.application') }}</label>
        <input
          :value="service.application_name || service.application_id"
          type="text"
          disabled
          class="app-input"
        />
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('service.fields.version') }}
          <span class="text-destructive">*</span>
        </label>
        <select
          v-model="basicForm.version_id"
          class="app-input"
          :class="basicErrors.version_id ? 'app-input-error' : ''"
          :disabled="operating"
          :aria-invalid="basicErrors.version_id ? 'true' : undefined"
          @change="basicErrors.version_id = ''"
        >
          <option value="" disabled>{{ t('service.create.selectVersion') }}</option>
          <option v-for="version in versions" :key="version.id" :value="version.id">
            {{ version.label }}
          </option>
        </select>
        <p v-if="basicErrors.version_id" class="app-field-error" role="alert">
          {{ basicErrors.version_id }}
        </p>
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('service.fields.instanceKey') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="basicForm.instance_key"
          type="text"
          class="app-input"
          :class="basicErrors.instance_key ? 'app-input-error' : ''"
          :disabled="operating"
          :aria-invalid="basicErrors.instance_key ? 'true' : undefined"
          @input="basicErrors.instance_key = ''"
        />
        <p v-if="basicErrors.instance_key" class="app-field-error" role="alert">
          {{ basicErrors.instance_key }}
        </p>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="isBasicDialogOpen = false"
          @confirm="saveBasic"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isRuntimeConfigDialogOpen"
      :title="editingRuntimeConfigKey === null ? t('common.add') : t('common.edit')"
    >
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('service.runtimeConfig.key') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="runtimeConfigForm.key"
          type="text"
          class="app-input"
          :class="runtimeConfigFormErrors.key ? 'app-input-error' : ''"
          :disabled="operating"
          :aria-invalid="runtimeConfigFormErrors.key ? 'true' : undefined"
          @input="runtimeConfigFormErrors.key = ''"
        />
        <p v-if="runtimeConfigFormErrors.key" class="app-field-error" role="alert">
          {{ runtimeConfigFormErrors.key }}
        </p>
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">{{ t('service.runtimeConfig.value') }}</label>
        <div class="relative">
          <input
            v-model="runtimeConfigForm.value"
            :type="isRuntimeConfigValueVisible ? 'text' : 'password'"
            class="app-input pr-10"
            :disabled="operating"
          />
          <button
            type="button"
            class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground transition-colors hover:text-foreground"
            :aria-label="
              isRuntimeConfigValueVisible
                ? t('service.runtimeConfig.hideValue')
                : t('service.runtimeConfig.showValue')
            "
            :disabled="operating"
            :title="
              isRuntimeConfigValueVisible
                ? t('service.runtimeConfig.hideValue')
                : t('service.runtimeConfig.showValue')
            "
            @click="isRuntimeConfigValueVisible = !isRuntimeConfigValueVisible"
          >
            <EyeOff v-if="isRuntimeConfigValueVisible" class="size-4" />
            <Eye v-else class="size-4" />
          </button>
        </div>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="closeRuntimeConfigDialog"
          @confirm="saveRuntimeConfig"
        />
      </template>
    </AppDialog>

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
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('application.detail.fields.component') }}
            <span class="text-destructive">*</span>
          </label>
          <select
            v-model="exposeForm.component_name"
            class="app-input"
            :class="exposeErrors.component_name ? 'app-input-error' : ''"
            :aria-invalid="exposeErrors.component_name ? 'true' : undefined"
            @change="exposeErrors.component_name = ''"
          >
            <option
              v-for="component in selectedComponents"
              :key="component.id"
              :value="component.name"
            >
              {{ component.name }}
            </option>
          </select>
          <p v-if="exposeErrors.component_name" class="app-field-error" role="alert">
            {{ exposeErrors.component_name }}
          </p>
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div class="space-y-1.5">
            <label class="app-field-label block">
              {{ t('application.detail.placeholders.exposeProtocol') }}
              <span class="text-destructive">*</span>
            </label>
            <select
              v-model="exposeForm.protocol"
              class="app-input"
              :class="exposeErrors.protocol ? 'app-input-error' : ''"
              :aria-invalid="exposeErrors.protocol ? 'true' : undefined"
              @change="
                exposeErrors.protocol = '';
                exposeErrors.path_prefix = '';
                if (exposeForm.protocol === 'tcp') exposeForm.path_prefix = '';
              "
            >
              <option value="" disabled>
                {{ t('application.detail.placeholders.exposeProtocol') }}
              </option>
              <option value="http">http</option>
              <option value="tcp">tcp</option>
            </select>
            <p v-if="exposeErrors.protocol" class="app-field-error" role="alert">
              {{ exposeErrors.protocol }}
            </p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">
              {{ t('application.detail.placeholders.exposeAccess') }}
              <span class="text-destructive">*</span>
            </label>
            <select
              v-model="exposeForm.access"
              class="app-input"
              :class="exposeErrors.access ? 'app-input-error' : ''"
              :aria-invalid="exposeErrors.access ? 'true' : undefined"
              @change="exposeErrors.access = ''"
            >
              <option value="local">local</option>
              <option value="public">public</option>
            </select>
            <p v-if="exposeErrors.access" class="app-field-error" role="alert">
              {{ exposeErrors.access }}
            </p>
          </div>
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div class="space-y-1.5">
            <label class="app-field-label block">
              {{ t('application.detail.placeholders.containerPort') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="exposeForm.container_port"
              type="number"
              min="1"
              max="65535"
              class="app-input"
              :class="exposeErrors.container_port ? 'app-input-error' : ''"
              :aria-invalid="exposeErrors.container_port ? 'true' : undefined"
              @input="exposeErrors.container_port = ''"
            />
            <p v-if="exposeErrors.container_port" class="app-field-error" role="alert">
              {{ exposeErrors.container_port }}
            </p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">
              {{ t('application.detail.placeholders.listenPort') }}
            </label>
            <input
              v-model="exposeForm.listen_port"
              type="number"
              min="1"
              max="65535"
              class="app-input"
              :class="exposeErrors.listen_port ? 'app-input-error' : ''"
              :aria-invalid="exposeErrors.listen_port ? 'true' : undefined"
              @input="exposeErrors.listen_port = ''"
            />
            <p v-if="exposeErrors.listen_port" class="app-field-error" role="alert">
              {{ exposeErrors.listen_port }}
            </p>
          </div>
        </div>
        <div v-if="exposeForm.protocol === 'http'" class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('application.detail.fields.pathPrefix') }}
          </label>
          <input
            v-model="exposeForm.path_prefix"
            class="app-input"
            :class="exposeErrors.path_prefix ? 'app-input-error' : ''"
            :aria-invalid="exposeErrors.path_prefix ? 'true' : undefined"
            @input="exposeErrors.path_prefix = ''"
          />
          <p v-if="exposeErrors.path_prefix" class="app-field-error" role="alert">
            {{ exposeErrors.path_prefix }}
          </p>
        </div>
        <p v-if="exposeFormError" class="app-field-error text-xs">{{ exposeFormError }}</p>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="closeExposeDialog"
          @confirm="saveExpose"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeployDialogOpen"
      :title="t('service.deploy.dialogTitle')"
      width-class="w-[min(440px,calc(100vw-32px))]"
    >
      <label class="flex items-center gap-2">
        <input v-model="forceRecreate" type="checkbox" class="app-checkbox" />
        <span class="text-sm text-foreground">{{ t('service.deploy.forceRecreate') }}</span>
      </label>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.deploy')"
          @cancel="isDeployDialogOpen = false"
          @confirm="deploy"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isStopDialogOpen"
      :title="t('service.stop.dialogTitle')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="mb-4 text-sm text-muted-foreground">{{ t('service.stop.confirm') }}</p>
      <label class="flex items-center gap-2">
        <input v-model="stopRemoveVolumes" type="checkbox" class="app-checkbox" />
        <span class="text-sm text-foreground">{{ t('service.stop.removeVolumes') }}</span>
      </label>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.stop')"
          variant="destructive"
          @cancel="isStopDialogOpen = false"
          @confirm="stop"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('service.delete.dialogTitle')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('service.delete.confirm', { instance: service?.instance_key || '' }) }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="isDeleteDialogOpen = false"
          @confirm="remove"
        />
      </template>
    </AppDialog>

    <AppDrawer
      :open="previewDrawerOpen"
      :title="t('application.detail.drawer.composePreview')"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="(open) => (previewDrawerOpen = open)"
    >
      <div class="flex h-full min-h-[420px] flex-col p-6">
        <AppSpinner v-if="previewLoading" class="py-8" />
        <p v-else-if="previewError" class="text-sm text-destructive">{{ previewError }}</p>
        <MonacoEditor
          v-else
          :model-value="previewYaml"
          language="yaml"
          height="100%"
          :readonly="true"
        />
      </div>
    </AppDrawer>
    <AppDrawer
      :open="logsDrawerOpen"
      :title="t('service.logs.title')"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="(open) => (logsDrawerOpen = open)"
    >
      <div class="flex h-full min-h-[420px] flex-col gap-3 p-6">
        <button class="app-button h-9 w-fit px-3" :disabled="logsLoading" @click="loadLogs">
          <RefreshCw class="size-4" :class="{ 'animate-spin': logsLoading }" />
          {{ t('common.refresh') }}
        </button>
        <AppSpinner v-if="logsLoading" class="py-8" />
        <p v-else-if="logsError" class="text-sm text-destructive">{{ logsError }}</p>
        <pre
          v-else
          class="min-h-0 flex-1 overflow-auto whitespace-pre-wrap break-words rounded border border-border bg-muted/30 p-3 text-xs text-foreground"
          >{{ logs || t('service.logs.empty') }}</pre>
      </div>
    </AppDrawer>
  </div>
</template>

<script setup lang="ts">
  import {
    ArrowLeft,
    Eye,
    EyeOff,
    FileCode2,
    Pencil,
    Plus,
    RefreshCw,
    Rocket,
    ScrollText,
    Square,
    Trash2,
  } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import { serviceApi } from '@/api/service/service';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import SensitiveValue from '@/components/SensitiveValue.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ApplicationContainerStatusResp } from '@/gen/proto/orbit/v1/application/application';
  import type { VersionComponentResp, VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import type { ServiceExposeReq, ServiceResp } from '@/gen/proto/orbit/v1/service/service';
  import { appStatusTone, containerStateTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';

  type ExposeForm = {
    component_name: string;
    protocol: string;
    container_port: string;
    listen_port: string;
    access: string;
    path_prefix: string;
  };

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const { loading, execute } = useStatusAsync();
  const { loading: opLoading, execute: executeOp } = useStatusAsync();
  const serviceId = computed(() => String(route.params.id || ''));
  const service = ref<ServiceResp>();
  const versions = ref<VersionResp[]>([]);
  const selectedComponents = ref<VersionComponentResp[]>([]);
  const containers = ref<ApplicationContainerStatusResp[]>([]);
  const containersLoading = ref(false);
  const containersError = ref('');
  const isBasicDialogOpen = ref(false);
  const basicErrors = reactive({ version_id: '', instance_key: '' });
  const basicForm = reactive({ version_id: '', instance_key: '' });
  const isRuntimeConfigDialogOpen = ref(false);
  const editingRuntimeConfigKey = ref<string | null>(null);
  const isRuntimeConfigValueVisible = ref(false);
  const runtimeConfigFormErrors = reactive({ key: '' });
  const runtimeConfigForm = reactive({ key: '', value: '' });
  const isExposeDialogOpen = ref(false);
  const editingExposeIndex = ref<number | null>(null);
  const exposeFormError = ref('');
  const exposeErrors = reactive({
    component_name: '',
    protocol: '',
    access: '',
    container_port: '',
    listen_port: '',
    path_prefix: '',
  });
  const exposeForm = reactive<ExposeForm>({
    component_name: '',
    protocol: '',
    container_port: '',
    listen_port: '',
    access: '',
    path_prefix: '',
  });
  const isDeployDialogOpen = ref(false);
  const isStopDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const forceRecreate = ref(false);
  const stopRemoveVolumes = ref(false);
  const previewDrawerOpen = ref(false);
  const previewLoading = ref(false);
  const previewYaml = ref('');
  const previewError = ref('');
  const logsDrawerOpen = ref(false);
  const logsLoading = ref(false);
  const logs = ref('');
  const logsError = ref('');
  const operating = computed(() => opLoading.value);
  const isDeploying = computed(() => service.value?.status === 'deploying');
  const canStop = computed(
    () => service.value?.status === 'running' || service.value?.status === 'faulted'
  );
  const canDelete = computed(() => service.value?.status === 'stopped');
  const runtimeConfigEntries = computed(() =>
    Object.entries(service.value?.runtime_config ?? {})
      .map(([key, value]) => ({ key, value }))
      .sort((a, b) => a.key.localeCompare(b.key))
  );

  function cloneExposes(exposes: ServiceResp['exposes']): ServiceExposeReq[] {
    return exposes.map((expose) => ({
      component_name: expose.component_name,
      protocol: expose.protocol,
      container_port: expose.container_port,
      listen_port: expose.listen_port,
      access: expose.access,
      path_prefix: expose.path_prefix,
    }));
  }

  async function fetchService() {
    if (!serviceId.value) return;
    try {
      await execute(async () => {
        service.value = await serviceApi.get(serviceId.value);
      });
      if (service.value)
        await Promise.all([loadVersions(), loadSelectedVersion(), loadContainers()]);
    } catch (error) {
      service.value = undefined;
      toast.error(error instanceof Error ? error.message : t('service.toast.loadDetailFailed'));
    }
  }

  async function loadVersions() {
    if (!service.value) return;
    const page = await applicationApi.listVersions(service.value.application_id, { per_page: 100 });
    versions.value = page.items ?? [];
  }

  async function loadSelectedVersion(versionId = service.value?.version_id ?? '') {
    if (!versionId) {
      selectedComponents.value = [];
      return;
    }
    const version = await applicationApi.getVersion(versionId);
    selectedComponents.value = version.components ?? [];
  }

  async function loadContainers() {
    if (!service.value) return;
    containersLoading.value = true;
    containersError.value = '';
    try {
      containers.value = (
        await applicationApi.getStatus(service.value.application_id, {
          service_id: service.value.id,
        })
      ).containers;
    } catch (error) {
      containers.value = [];
      containersError.value =
        error instanceof Error ? error.message : t('service.containers.loadFailed');
    } finally {
      containersLoading.value = false;
    }
  }

  function openBasicDialog() {
    if (!service.value) return;
    Object.assign(basicForm, {
      version_id: service.value.version_id,
      instance_key: service.value.instance_key,
    });
    Object.assign(basicErrors, { version_id: '', instance_key: '' });
    isBasicDialogOpen.value = true;
  }
  async function saveBasic() {
    const current = service.value;
    if (!current) return;
    basicErrors.version_id = basicForm.version_id ? '' : t('service.create.versionRequired');
    basicErrors.instance_key = basicForm.instance_key.trim()
      ? ''
      : t('service.create.instanceKeyRequired');
    if (basicErrors.version_id || basicErrors.instance_key) {
      return;
    }
    try {
      await executeOp(async () => {
        service.value = await serviceApi.updateBasic(current.id, {
          version_id: basicForm.version_id,
          instance_key: basicForm.instance_key.trim(),
        });
      });
      isBasicDialogOpen.value = false;
      toast.success(t('service.detail.saved'));
      await Promise.all([loadSelectedVersion(), loadContainers()]);
    } catch (error) {
      basicErrors.instance_key =
        error instanceof Error ? error.message : t('service.toast.loadDetailFailed');
    }
  }

  function openRuntimeConfigDialog(key?: string) {
    if (!service.value) return;
    editingRuntimeConfigKey.value = key ?? null;
    Object.assign(runtimeConfigForm, {
      key: key ?? '',
      value: key === undefined ? '' : service.value.runtime_config[key],
    });
    runtimeConfigFormErrors.key = '';
    isRuntimeConfigValueVisible.value = false;
    isRuntimeConfigDialogOpen.value = true;
  }

  function closeRuntimeConfigDialog() {
    isRuntimeConfigDialogOpen.value = false;
    editingRuntimeConfigKey.value = null;
    runtimeConfigFormErrors.key = '';
    isRuntimeConfigValueVisible.value = false;
  }

  function resetExposeErrors() {
    Object.assign(exposeErrors, {
      component_name: '',
      protocol: '',
      access: '',
      container_port: '',
      listen_port: '',
      path_prefix: '',
    });
  }

  function closeExposeDialog() {
    isExposeDialogOpen.value = false;
    editingExposeIndex.value = null;
    exposeFormError.value = '';
    resetExposeErrors();
  }

  function openExposeDialog(index?: number) {
    if (!service.value || !selectedComponents.value.length) return;
    editingExposeIndex.value = index ?? null;
    exposeFormError.value = '';
    resetExposeErrors();
    const current = index === undefined ? undefined : service.value.exposes[index];
    if (current)
      Object.assign(exposeForm, {
        component_name: current.component_name,
        protocol: current.protocol,
        container_port: String(current.container_port),
        listen_port: current.listen_port === undefined ? '' : String(current.listen_port),
        access: current.access,
        path_prefix: current.path_prefix ?? '',
      });
    else
      Object.assign(exposeForm, {
        component_name: selectedComponents.value[0].name,
        protocol: '',
        container_port: '',
        listen_port: '',
        access: 'local',
        path_prefix: '',
      });
    isExposeDialogOpen.value = true;
  }

  async function saveExpose() {
    const current = service.value;
    if (!current) return;
    const containerPort = Number(exposeForm.container_port);
    const listenText = exposeForm.listen_port.trim();
    const listenPort = listenText === '' ? undefined : Number(listenText);
    const pathPrefix = exposeForm.path_prefix.trim();
    const componentMessage = t('application.validation.exposeComponentRequired');
    const protocolAccessMessage = t('application.validation.exposeFieldsInvalid');
    const portMessage = t('application.validation.portRange');
    exposeErrors.component_name = selectedComponents.value.some(
      (component) => component.name === exposeForm.component_name
    )
      ? ''
      : componentMessage;
    exposeErrors.protocol = ['http', 'tcp'].includes(exposeForm.protocol)
      ? ''
      : protocolAccessMessage;
    exposeErrors.access = ['local', 'public'].includes(exposeForm.access)
      ? ''
      : protocolAccessMessage;
    exposeErrors.container_port =
      Number.isInteger(containerPort) && containerPort >= 1 && containerPort <= 65535
        ? ''
        : portMessage;
    exposeErrors.listen_port =
      listenPort === undefined ||
      (Number.isInteger(listenPort) && listenPort >= 1 && listenPort <= 65535)
        ? ''
        : portMessage;
    exposeErrors.path_prefix =
      exposeForm.protocol === 'tcp' && pathPrefix ? protocolAccessMessage : '';
    if (Object.values(exposeErrors).some(Boolean)) {
      return;
    }
    const expose: ServiceExposeReq = {
      component_name: exposeForm.component_name,
      protocol: exposeForm.protocol,
      container_port: containerPort,
      listen_port: listenPort,
      access: exposeForm.access,
      path_prefix: exposeForm.protocol === 'http' && pathPrefix ? pathPrefix : undefined,
    };
    const exposes = cloneExposes(current.exposes);
    if (editingExposeIndex.value === null) exposes.push(expose);
    else exposes[editingExposeIndex.value] = expose;
    try {
      await executeOp(async () => {
        service.value = await serviceApi.updateConfiguration(current.id, {
          runtime_config: current.runtime_config,
          exposes,
        });
      });
      closeExposeDialog();
      toast.success(t('service.exposes.saved'));
    } catch (error) {
      exposeFormError.value =
        error instanceof Error ? error.message : t('service.exposes.loadFailed');
    }
  }

  async function saveRuntimeConfig() {
    const current = service.value;
    if (!current) return;
    const key = runtimeConfigForm.key.trim();
    if (
      !key ||
      (editingRuntimeConfigKey.value !== key &&
        Object.prototype.hasOwnProperty.call(current.runtime_config, key))
    ) {
      runtimeConfigFormErrors.key = t('service.runtimeConfig.invalid');
      return;
    }
    const runtimeConfig = { ...current.runtime_config };
    if (editingRuntimeConfigKey.value !== null) delete runtimeConfig[editingRuntimeConfigKey.value];
    runtimeConfig[key] = runtimeConfigForm.value;
    try {
      await executeOp(async () => {
        service.value = await serviceApi.updateConfiguration(current.id, {
          runtime_config: runtimeConfig,
          exposes: cloneExposes(current.exposes),
        });
      });
      closeRuntimeConfigDialog();
      toast.success(t('service.runtimeConfig.saved'));
    } catch (error) {
      runtimeConfigFormErrors.key =
        error instanceof Error ? error.message : t('service.runtimeConfig.loadFailed');
    }
  }

  async function removeRuntimeConfig(key: string) {
    const current = service.value;
    if (!current) return;
    const runtimeConfig = { ...current.runtime_config };
    delete runtimeConfig[key];
    try {
      await executeOp(async () => {
        service.value = await serviceApi.updateConfiguration(current.id, {
          runtime_config: runtimeConfig,
          exposes: cloneExposes(current.exposes),
        });
      });
      toast.success(t('service.runtimeConfig.saved'));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.runtimeConfig.loadFailed'));
    }
  }

  async function removeExpose(index: number) {
    const current = service.value;
    if (!current) return;
    const exposes = cloneExposes(current.exposes).filter((_, exposeIndex) => exposeIndex !== index);
    try {
      await executeOp(async () => {
        service.value = await serviceApi.updateConfiguration(current.id, {
          runtime_config: current.runtime_config,
          exposes,
        });
      });
      toast.success(t('service.exposes.saved'));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.exposes.loadFailed'));
    }
  }

  async function openPreview() {
    if (!service.value) return;
    previewDrawerOpen.value = true;
    previewLoading.value = true;
    previewYaml.value = '';
    previewError.value = '';
    try {
      previewYaml.value = (await serviceApi.preview(service.value.id)).compose_yaml;
    } catch (error) {
      previewError.value =
        error instanceof Error ? error.message : t('application.toast.loadPreviewFailed');
    } finally {
      previewLoading.value = false;
    }
  }
  function openDeployDialog() {
    forceRecreate.value = false;
    isDeployDialogOpen.value = true;
  }
  async function deploy() {
    const current = service.value;
    if (!current) return;
    try {
      await executeOp(async () => {
        const result = await serviceApi.deploy(current.id, {
          force_recreate: forceRecreate.value,
        });
        for (const warning of result.warnings) toast.error(warning);
        isDeployDialogOpen.value = false;
        if (result.deployment_id) await router.push(`/deployment/${result.deployment_id}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.deployFailed'));
    }
  }
  async function stop() {
    const current = service.value;
    if (!current) return;
    try {
      await executeOp(async () => {
        const result = await applicationApi.stop(current.application_id, {
          service_id: current.id,
          remove_volumes: stopRemoveVolumes.value,
        });
        isStopDialogOpen.value = false;
        if (result.deployment_id) await router.push(`/deployment/${result.deployment_id}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.stopFailed'));
    }
  }
  async function remove() {
    const current = service.value;
    if (!current) return;
    try {
      await executeOp(async () => {
        await serviceApi.remove(current.id);
        await router.push('/services');
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.deleteFailed'));
    }
  }
  function openLogsDrawer() {
    logsDrawerOpen.value = true;
    void loadLogs();
  }
  async function loadLogs() {
    if (!service.value) return;
    logsLoading.value = true;
    logsError.value = '';
    try {
      logs.value = (
        await applicationApi.getLogs(service.value.application_id, {
          service_id: service.value.id,
          tail: 200,
        })
      ).logs;
    } catch (error) {
      logsError.value = error instanceof Error ? error.message : t('service.logs.loadFailed');
    } finally {
      logsLoading.value = false;
    }
  }

  watch(serviceId, () => {
    service.value = undefined;
    void fetchService();
  });
  onMounted(() => {
    void fetchService();
  });
</script>
