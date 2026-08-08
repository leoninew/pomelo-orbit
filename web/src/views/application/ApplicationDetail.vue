<template>
  <div class="flex flex-col gap-4">
    <div v-if="!versionsOnly" class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">
          {{ application?.name || t('application.detail.title') }}
        </h1>
        <DetailHeaderMeta v-if="application">
          <AppBadge :tone="applicationKindTone(application.kind)">{{ application.kind }}</AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button v-if="application" class="app-button h-9 px-3" @click="handleExport">
          <Download class="size-4" />
          {{ t('application.detail.actions.export') }}
        </button>
        <button
          v-if="application"
          :disabled="operating"
          class="app-button-danger h-9 px-3"
          @click="openDeleteModal"
        >
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/applications')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppLoadingState v-if="basicInfoLoading" size="section" />

    <div v-else-if="application" class="flex flex-col gap-4">
      <DetailInfoCard
        v-if="!versionsOnly"
        :title="t('application.detail.sections.basicInfo')"
        editable
        :disabled="operating"
        @edit="openEditModal"
      >
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt class="whitespace-nowrap">{{ t('common.name') }}</dt>
            <dd class="text-foreground">{{ application.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="whitespace-nowrap">{{ t('application.code') }}</dt>
            <dd class="text-foreground">{{ application.code }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="whitespace-nowrap">{{ t('application.kind') }}</dt>
            <dd>
              <AppBadge variant="pill" :tone="applicationKindTone(application.kind)">
                {{ application.kind }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="whitespace-nowrap">{{ t('common.createdAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(application.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="whitespace-nowrap">{{ t('common.updatedAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(application.updated_at) }}</dd>
          </div>
        </dl>
      </DetailInfoCard>

      <!-- 版本 -->
      <DetailInfoCard :title="t('application.detail.sections.versions')">
        <template #actions>
          <button class="app-button-primary h-9 px-3" @click="openCreateVersionModal">
            <Plus class="size-4" />
            {{ t('application.detail.actions.createVersion') }}
          </button>
        </template>
        <AppLoadingState v-if="versionListLoading" size="compact" />
        <AppEmptyState v-else-if="versions.length === 0" size="compact" />
        <div v-else-if="!versionsOnly" class="overflow-x-auto">
          <table class="app-data-table min-w-[1000px]">
            <thead>
              <tr>
                <th class="w-12">
                  <span class="sr-only">
                    {{ t('application.detail.actions.expandComponents') }}
                  </span>
                </th>
                <th>{{ t('application.detail.fields.versionLabel') }}</th>
                <th>{{ t('common.status') }}</th>
                <th>{{ t('application.detail.fields.components') }}</th>
                <th>{{ t('common.createdAt') }}</th>
                <th>{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="version in versions" :key="version.id">
                <tr>
                  <td class="w-12">
                    <button
                      type="button"
                      class="app-icon-button size-7"
                      :title="
                        isVersionExpanded(version.id)
                          ? t('application.detail.actions.collapseComponents')
                          : t('application.detail.actions.expandComponents')
                      "
                      :aria-label="
                        isVersionExpanded(version.id)
                          ? t('application.detail.actions.collapseComponents')
                          : t('application.detail.actions.expandComponents')
                      "
                      :aria-expanded="isVersionExpanded(version.id)"
                      @click="toggleVersionComponents(version.id)"
                    >
                      <ChevronDown
                        class="size-4 transition-transform"
                        :class="isVersionExpanded(version.id) ? 'rotate-180' : ''"
                      />
                    </button>
                  </td>
                  <td class="text-foreground">
                    <router-link :to="`/version/${version.id}`" class="app-link">
                      {{ version.label }}
                    </router-link>
                  </td>
                  <td>
                    <AppBadge variant="pill" :tone="versionStatusTone(version.status)">
                      {{ version.status }}
                    </AppBadge>
                  </td>
                  <td
                    class="max-w-xs truncate text-muted-foreground"
                    :title="version.component_summary || ''"
                  >
                    {{ version.component_summary }}
                  </td>
                  <td class="whitespace-nowrap text-muted-foreground">
                    {{ formatTime(version.created_at) }}
                  </td>
                  <td>
                    <div class="flex flex-wrap items-center gap-3">
                      <router-link :to="`/version/${version.id}`" class="app-link">
                        {{ t('application.view') }}
                      </router-link>
                      <button
                        class="app-link"
                        :disabled="operating"
                        @click="previewVersion(version.id)"
                      >
                        {{ t('application.detail.actions.preview') }}
                      </button>
                      <button class="app-link" @click="openForkModal(version)">
                        {{ t('application.detail.actions.fork') }}
                      </button>
                      <button
                        class="app-link-danger"
                        :disabled="operating"
                        @click="openDeleteVersionModal(version)"
                      >
                        {{ t('common.delete') }}
                      </button>
                    </div>
                  </td>
                </tr>
                <tr v-if="isVersionExpanded(version.id)" class="!hover:bg-transparent">
                  <td :colspan="6" class="!h-auto bg-muted/20 p-0">
                    <div class="px-5 py-4">
                      <AppLoadingState
                        v-if="versionComponentLoadState[version.id] === 'loading'"
                        size="compact"
                      />
                      <p
                        v-else-if="versionComponentLoadState[version.id] === 'error'"
                        class="py-2 text-sm text-destructive"
                      >
                        {{ t('application.versionDetail.componentsLoadFailed') }}
                      </p>
                      <AppEmptyState
                        v-else-if="versionComponents[version.id]?.length === 0"
                        size="compact"
                      />
                      <div v-else-if="versionComponents[version.id]" class="overflow-x-auto">
                        <table class="app-data-table min-w-[840px]">
                          <thead>
                            <tr>
                              <th>{{ t('application.detail.fields.component') }}</th>
                              <th>{{ t('application.detail.fields.image') }}</th>
                              <th>{{ t('application.componentDetail.fields.pullPolicy') }}</th>
                              <th>{{ t('application.componentDetail.fields.restartPolicy') }}</th>
                              <th>{{ t('common.operation') }}</th>
                            </tr>
                          </thead>
                          <tbody>
                            <tr
                              v-for="component in versionComponents[version.id]"
                              :key="component.id"
                            >
                              <td>
                                <router-link
                                  :to="`/version/${version.id}/component/${component.id}`"
                                  class="app-link"
                                >
                                  {{ component.name }}
                                </router-link>
                              </td>
                              <td
                                class="max-w-md whitespace-normal break-all text-muted-foreground"
                              >
                                {{ component.image }}
                              </td>
                              <td class="text-muted-foreground">{{ component.pull_policy }}</td>
                              <td class="text-muted-foreground">{{ component.restart_policy }}</td>
                              <td>
                                <div
                                  v-if="version.status === 'unpublished'"
                                  class="flex items-center gap-2"
                                >
                                  <button
                                    class="app-link"
                                    :disabled="operating"
                                    @click="openComponentEditDialog(version.id, component)"
                                  >
                                    {{ t('common.edit') }}
                                  </button>
                                  <button
                                    class="app-link-danger"
                                    :disabled="operating"
                                    @click="openComponentDeleteDialog(version.id, component)"
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
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[880px]">
            <thead>
              <tr>
                <th>{{ t('application.detail.fields.versionLabel') }}</th>
                <th>{{ t('common.status') }}</th>
                <th>{{ t('application.detail.fields.components') }}</th>
                <th>{{ t('application.detail.fields.note') }}</th>
                <th>{{ t('common.createdAt') }}</th>
                <th>{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="version in versions" :key="version.id">
                <td class="text-foreground">
                  <router-link :to="`/version/${version.id}`" class="app-link">
                    {{ version.label }}
                  </router-link>
                </td>
                <td>
                  <AppBadge variant="pill" :tone="versionStatusTone(version.status)">
                    {{ version.status }}
                  </AppBadge>
                </td>
                <td
                  class="max-w-xs truncate text-muted-foreground"
                  :title="version.component_summary || ''"
                >
                  {{ version.component_summary }}
                </td>
                <td class="max-w-xs truncate text-muted-foreground" :title="version.note || ''">
                  {{ version.note }}
                </td>
                <td class="text-muted-foreground">{{ formatTime(version.created_at) }}</td>
                <td>
                  <div class="flex flex-wrap items-center gap-3">
                    <button
                      v-if="versionsOnly && version.status === 'unpublished'"
                      class="app-link"
                      :disabled="operating"
                      @click="handlePublish(version.id)"
                    >
                      {{ t('application.detail.actions.publish') }}
                    </button>
                    <button
                      v-else-if="versionsOnly && version.status === 'published'"
                      class="app-link"
                      :disabled="operating"
                      @click="handleUnpublish(version.id)"
                    >
                      {{ t('application.detail.actions.unpublish') }}
                    </button>
                    <button
                      class="app-link"
                      :disabled="operating"
                      @click="previewVersion(version.id)"
                    >
                      {{ t('application.detail.actions.preview') }}
                    </button>
                    <button class="app-link" @click="openForkModal(version)">
                      {{ t('application.detail.actions.fork') }}
                    </button>
                    <button
                      v-if="versionsOnly"
                      class="app-link-danger"
                      :disabled="operating"
                      @click="openDeleteVersionModal(version)"
                    >
                      {{ t('common.delete') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </DetailInfoCard>
      <ListPagination
        v-if="versionsOnly"
        :current="versionPagination.current"
        :page-size="versionPagination.pageSize"
        :total="versionPagination.total"
        :total-pages="versionTotalPages"
        @change-page="handleVersionPageChange"
        @change-page-size="handleVersionPageSizeChange"
      />
    </div>

    <!-- 编辑应用 -->
    <AppDialog
      v-if="!versionsOnly"
      v-model:open="isEditDialogOpen"
      :title="t('application.detail.dialog.editApplication')"
    >
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('common.name') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="editForm.name"
          type="text"
          class="app-input"
          :class="editErrors.name ? 'app-input-error' : ''"
          :aria-invalid="editErrors.name ? 'true' : undefined"
          @input="editErrors.name = ''"
        />
        <p v-if="editErrors.name" class="app-field-error mt-1 text-xs">{{ editErrors.name }}</p>
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">{{ t('application.code') }}</label>
        <input v-model="editForm.code" type="text" disabled class="app-input" />
      </div>
      <p v-if="editSubmitError" class="app-field-error mt-3" role="alert">
        {{ editSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="isEditDialogOpen = false"
          @confirm="handleEditOk"
        />
      </template>
    </AppDialog>

    <!-- 删除应用 -->
    <AppDialog
      v-if="!versionsOnly"
      v-model:open="isDeleteDialogOpen"
      :title="t('application.detail.dialog.confirmDelete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="mb-4 text-sm text-muted-foreground">
        {{
          t('application.detail.dialog.deleteApplicationConfirm', {
            name: application?.name || '-',
          })
        }}
      </p>
      <label class="flex items-center gap-2">
        <input v-model="deleteDir" type="checkbox" class="app-checkbox" />
        <span class="text-sm text-foreground">
          {{ t('application.detail.dialog.deleteWorkDir', { code: application?.code || '-' }) }}
        </span>
      </label>
      <p v-if="deleteApplicationError" class="app-field-error mt-3" role="alert">
        {{ deleteApplicationError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="isDeleteDialogOpen = false"
          @confirm="handleDeleteOk"
        />
      </template>
    </AppDialog>

    <!-- 删除版本 -->
    <AppDialog
      :open="isDeleteVersionDialogOpen"
      :title="t('application.detail.dialog.deleteVersion')"
      width-class="w-[min(420px,calc(100vw-32px))]"
      @update:open="setDeleteVersionDialogOpen"
    >
      <p class="text-sm text-muted-foreground">
        {{
          t('application.detail.dialog.deleteVersionConfirm', {
            label: pendingDeleteVersion?.label || '-',
          })
        }}
      </p>
      <p v-if="deleteVersionError" class="app-field-error" role="alert">
        {{ deleteVersionError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="setDeleteVersionDialogOpen(false)"
          @confirm="handleDeleteVersionOk"
        />
      </template>
    </AppDialog>

    <!-- 编辑版本组件基本信息 -->
    <AppDialog
      :open="isComponentEditDialogOpen"
      :title="t('application.componentDetail.sections.basic')"
      width-class="w-[min(640px,calc(100vw-32px))]"
      @update:open="setComponentEditDialogOpen"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.component') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="componentForm.name"
            class="app-input"
            :class="componentErrors.name ? 'app-input-error' : ''"
            type="text"
            :aria-invalid="componentErrors.name ? 'true' : undefined"
            @input="componentErrors.name = ''"
          />
          <p v-if="componentErrors.name" class="app-field-error" role="alert">
            {{ componentErrors.name }}
          </p>
        </div>
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.image') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="componentForm.image"
            class="app-input"
            :class="componentErrors.image ? 'app-input-error' : ''"
            type="text"
            :aria-invalid="componentErrors.image ? 'true' : undefined"
            @input="componentErrors.image = ''"
          />
          <p v-if="componentErrors.image" class="app-field-error" role="alert">
            {{ componentErrors.image }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.pullPolicy') }}
          </label>
          <RawValueSelect
            v-model="componentForm.pull_policy"
            :placeholder="t('common.notSet')"
            :values="componentPullPolicyValues"
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.restartPolicy') }}
          </label>
          <RawValueSelect
            v-model="componentForm.restart_policy"
            :placeholder="t('common.notSet')"
            :values="componentRestartPolicyValues"
          />
        </div>
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.command') }}
          </label>
          <textarea v-model="componentForm.command" class="app-textarea" rows="3" />
        </div>
      </div>
      <p v-if="componentEditError" class="app-field-error mt-3" role="alert">
        {{ componentEditError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="cancelComponentEditing"
          @confirm="saveComponentBasic"
        />
      </template>
    </AppDialog>

    <!-- 删除版本组件 -->
    <AppDialog
      v-model:open="isComponentDeleteDialogOpen"
      :title="t('application.componentDetail.actions.delete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{
          t('application.componentDetail.deleteDescription', {
            name: pendingComponent?.name || '-',
          })
        }}
      </p>
      <p v-if="componentDeleteError" class="app-field-error mt-3" role="alert">
        {{ componentDeleteError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="cancelComponentDeletion"
          @confirm="deleteComponent"
        />
      </template>
    </AppDialog>

    <!-- 创建版本（仅基本信息） -->
    <AppDialog
      v-model:open="isVersionDialogOpen"
      :title="t('application.detail.dialog.createVersion')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.versionLabel') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="versionForm.label"
            type="text"
            class="app-input"
            :class="versionFormErrors.label ? 'app-input-error' : ''"
            :placeholder="t('application.detail.placeholders.versionLabel')"
            :aria-invalid="versionFormErrors.label ? 'true' : undefined"
            @input="versionFormErrors.label = ''"
          />
          <p v-if="versionFormErrors.label" class="app-field-error mt-1 text-xs">
            {{ versionFormErrors.label }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.note') }}
          </label>
          <input
            v-model="versionForm.note"
            type="text"
            class="app-input"
            :placeholder="t('application.detail.placeholders.note')"
          />
        </div>
      </div>
      <p v-if="versionCreateError" class="app-field-error mt-3" role="alert">
        {{ versionCreateError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.create')"
          @cancel="isVersionDialogOpen = false"
          @confirm="handleVersionCreate"
        />
      </template>
    </AppDialog>

    <!-- Fork 版本 -->
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
          :aria-invalid="forkLabelError ? 'true' : undefined"
          @input="forkLabelError = ''"
        />
        <p v-if="forkLabelError" class="app-field-error mt-1 text-xs">{{ forkLabelError }}</p>
      </div>
      <p v-if="forkSubmitError" class="app-field-error mt-3" role="alert">
        {{ forkSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.copy')"
          @cancel="isForkDialogOpen = false"
          @confirm="handleForkOk"
        />
      </template>
    </AppDialog>

    <AppDrawer
      :open="previewOpen"
      :title="t('application.detail.drawer.composePreview')"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="setPreviewOpen"
    >
      <div class="flex h-full flex-col gap-3 p-6">
        <AppLoadingState v-if="previewLoading" class="flex-1 items-center" size="section" />
        <div
          v-else-if="previewError"
          class="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
        >
          {{ previewError }}
        </div>
        <div v-else class="min-h-0 flex-1">
          <MonacoEditor
            :model-value="previewContent"
            language="yaml"
            height="100%"
            :readonly="true"
          />
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="setPreviewOpen(false)">
          {{ t('application.detail.actions.close') }}
        </button>
      </template>
    </AppDrawer>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, ChevronDown, Download, Plus, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import type { VersionComponentResp, VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import { applicationKindTone, versionStatusTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import {
    componentBasicRequestFromForm,
    componentFormFromResponse,
    emptyComponentForm,
  } from './componentForm';

  const {
    versionsOnly = false,
    applicationId: applicationIdProp = '',
    versionSearch = '',
  } = defineProps<{
    versionsOnly?: boolean;
    applicationId?: string;
    versionSearch?: string;
  }>();

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const applicationId = applicationIdProp || (route.params.id as string);
  const toast = useToast();

  const { loading: basicInfoLoading, execute: executeBasicInfo } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { loading: previewLoading, execute: executePreview } = useStatusAsync();
  const { loading: versionListLoading, execute: executeVersionList } = useStatusAsync();

  const application = ref<ApplicationResp>();
  const versions = ref<VersionResp[]>([]);
  const expandedVersionIds = ref<string[]>([]);
  const versionComponents = reactive<Record<string, VersionComponentResp[]>>({});
  const versionComponentLoadState = reactive<Record<string, 'loading' | 'loaded' | 'error'>>({});
  const versionPagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const versionTotalPages = computed(() =>
    Math.ceil(versionPagination.total / versionPagination.pageSize)
  );

  const previewOpen = ref(false);
  const previewContent = ref('');
  const previewError = ref('');

  const isEditDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isDeleteVersionDialogOpen = ref(false);
  const pendingDeleteVersion = ref<VersionResp | null>(null);
  const deleteVersionError = ref('');
  const editSubmitError = ref('');
  const deleteApplicationError = ref('');
  const versionCreateError = ref('');
  const componentEditError = ref('');
  const componentDeleteError = ref('');
  const isVersionDialogOpen = ref(false);
  const isForkDialogOpen = ref(false);
  const isComponentEditDialogOpen = ref(false);
  const isComponentDeleteDialogOpen = ref(false);
  const pendingComponentVersionId = ref('');
  const pendingComponent = ref<VersionComponentResp>();
  const forkingVersionId = ref('');
  const forkLabel = ref('');
  const forkLabelError = ref('');
  const forkSubmitError = ref('');
  const deleteDir = ref(false);

  const editForm = reactive({
    name: '',
    code: '',
  });
  const editErrors = reactive({ name: '' });

  const versionForm = reactive({
    label: '',
    note: '',
  });
  const versionFormErrors = reactive({ label: '' });
  const componentForm = reactive(emptyComponentForm());
  const componentErrors = reactive({ name: '', image: '' });
  const componentPullPolicyValues = ['always', 'missing', 'never'];
  const componentRestartPolicyValues = ['no', 'unless-stopped'];

  async function fetchApplication() {
    try {
      await executeBasicInfo(async () => {
        const data = await applicationApi.get(applicationId);
        application.value = data;
        Object.assign(editForm, {
          name: data.name,
          code: data.code,
        });
      });
    } catch {
      toast.error(t('application.toast.loadDetailFailed'));
      router.push('/applications');
    }
  }

  async function loadVersions() {
    try {
      await executeVersionList(async () => {
        const resp = await applicationApi.listVersions(applicationId, {
          page: versionsOnly ? versionPagination.current : 1,
          per_page: versionsOnly ? versionPagination.pageSize : 100,
          search: versionsOnly ? versionSearch : '',
        });
        versions.value = resp.items ?? [];
        if (versionsOnly) {
          versionPagination.total = resp.total;
          versionPagination.current = resp.page;
          versionPagination.pageSize = resp.per_page;
        }
      });
    } catch {
      toast.error(t('application.toast.loadVersionsFailed'));
    }
  }

  async function loadVersionComponents(versionId: string) {
    const state = versionComponentLoadState[versionId];
    if (state === 'loading' || state === 'loaded') {
      return;
    }
    versionComponentLoadState[versionId] = 'loading';
    try {
      const version = await applicationApi.getVersion(versionId);
      versionComponents[versionId] = version.components ?? [];
      versionComponentLoadState[versionId] = 'loaded';
    } catch {
      versionComponentLoadState[versionId] = 'error';
      toast.error(t('application.versionDetail.componentsLoadFailed'));
    }
  }

  function isVersionExpanded(versionId: string) {
    return expandedVersionIds.value.includes(versionId);
  }

  function toggleVersionComponents(versionId: string) {
    if (isVersionExpanded(versionId)) {
      expandedVersionIds.value = expandedVersionIds.value.filter((id) => id !== versionId);
      return;
    }
    expandedVersionIds.value = [...expandedVersionIds.value, versionId];
    void loadVersionComponents(versionId);
  }

  function handleVersionPageChange(page: number) {
    versionPagination.current = page;
    void loadVersions();
  }

  function handleVersionPageSizeChange(pageSize: number) {
    versionPagination.pageSize = pageSize;
    versionPagination.current = 1;
    void loadVersions();
  }

  async function handleExport() {
    try {
      const data = await applicationApi.exportApplication(applicationId);
      const blob = new Blob([JSON.stringify(data, null, 2)], {
        type: 'application/json',
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${data.code || 'application'}.json`;
      a.click();
      URL.revokeObjectURL(url);
      toast.success(t('application.toast.exportSuccess'));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.exportFailed'));
    }
  }

  async function previewVersion(versionId: string) {
    previewContent.value = '';
    previewError.value = '';
    previewOpen.value = true;
    try {
      await executePreview(async () => {
        const result = await applicationApi.previewVersion(versionId, {});
        previewContent.value = result.compose_yaml;
      });
    } catch (error) {
      previewError.value =
        error instanceof Error ? error.message : t('application.toast.loadPreviewFailed');
    }
  }

  function setPreviewOpen(open: boolean) {
    previewOpen.value = open;
    if (!open) {
      previewContent.value = '';
      previewError.value = '';
    }
  }

  function openEditModal() {
    editErrors.name = '';
    editSubmitError.value = '';
    if (application.value) {
      Object.assign(editForm, {
        name: application.value.name,
        code: application.value.code,
      });
    }
    isEditDialogOpen.value = true;
  }

  async function handleEditOk() {
    editSubmitError.value = '';
    editErrors.name = editForm.name.trim() ? '' : t('application.validation.nameRequired');
    if (editErrors.name) {
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.update(applicationId, {
          name: editForm.name,
        });
        toast.success(t('application.toast.updateSuccess'));
        isEditDialogOpen.value = false;
        await fetchApplication();
      });
    } catch (error) {
      editSubmitError.value =
        error instanceof Error ? error.message : t('application.toast.updateFailed');
    }
  }

  function openDeleteModal() {
    deleteDir.value = false;
    deleteApplicationError.value = '';
    isDeleteDialogOpen.value = true;
  }

  async function handleDeleteOk() {
    deleteApplicationError.value = '';
    try {
      await executeOp(async () => {
        await applicationApi.delete(applicationId, deleteDir.value);
        toast.success(t('application.toast.deleteSuccess'));
        router.push('/applications');
      });
    } catch (error) {
      deleteApplicationError.value =
        error instanceof Error ? error.message : t('application.toast.deleteFailed');
    }
  }

  function openCreateVersionModal() {
    versionForm.label = '';
    versionForm.note = '';
    versionFormErrors.label = '';
    versionCreateError.value = '';
    isVersionDialogOpen.value = true;
  }

  async function handleVersionCreate() {
    versionCreateError.value = '';
    versionFormErrors.label = versionForm.label.trim()
      ? ''
      : t('application.validation.versionLabelRequired');
    if (versionFormErrors.label) {
      return;
    }
    const note = versionForm.note.trim() || undefined;
    try {
      await executeOp(async () => {
        const created = await applicationApi.createVersion(applicationId, {
          application_id: applicationId,
          label: versionForm.label.trim(),
          note,
          components: [],
        });
        toast.success(t('application.toast.createVersionSuccess'));
        isVersionDialogOpen.value = false;
        router.push(`/version/${created.id}`);
      });
    } catch (error) {
      versionCreateError.value =
        error instanceof Error ? error.message : t('application.toast.saveFailed');
    }
  }

  async function handlePublish(versionId: string) {
    try {
      await executeOp(async () => {
        await applicationApi.publishVersion(versionId);
        toast.success(t('application.toast.publishSuccess'));
        await loadVersions();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.publishFailed'));
    }
  }

  async function handleUnpublish(versionId: string) {
    try {
      await executeOp(async () => {
        await applicationApi.unpublishVersion(versionId);
        toast.success(t('application.toast.unpublishSuccess'));
        await loadVersions();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.unpublishFailed'));
    }
  }

  function openDeleteVersionModal(version: VersionResp) {
    pendingDeleteVersion.value = version;
    deleteVersionError.value = '';
    isDeleteVersionDialogOpen.value = true;
  }

  function setDeleteVersionDialogOpen(open: boolean) {
    isDeleteVersionDialogOpen.value = open;
    if (!open) {
      pendingDeleteVersion.value = null;
      deleteVersionError.value = '';
    }
  }

  async function handleDeleteVersionOk() {
    const target = pendingDeleteVersion.value;
    if (!target) {
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.deleteVersion(target.id);
        toast.success(t('application.toast.deleteVersionSuccess'));
        setDeleteVersionDialogOpen(false);
        await loadVersions();
      });
    } catch (error) {
      deleteVersionError.value =
        error instanceof Error ? error.message : t('application.toast.deleteVersionFailed');
    }
  }

  function resetComponentErrors() {
    Object.assign(componentErrors, { name: '', image: '' });
  }

  function resetComponentForm() {
    Object.assign(componentForm, emptyComponentForm());
  }

  function openComponentEditDialog(versionId: string, component: VersionComponentResp) {
    pendingComponentVersionId.value = versionId;
    pendingComponent.value = component;
    Object.assign(componentForm, componentFormFromResponse(component));
    resetComponentErrors();
    componentEditError.value = '';
    isComponentEditDialogOpen.value = true;
  }

  function cancelComponentEditing() {
    resetComponentForm();
    resetComponentErrors();
    componentEditError.value = '';
    isComponentEditDialogOpen.value = false;
  }

  function setComponentEditDialogOpen(open: boolean) {
    if (open) {
      isComponentEditDialogOpen.value = true;
      return;
    }
    cancelComponentEditing();
  }

  async function saveComponentBasic() {
    const target = pendingComponent.value;
    if (!target || !pendingComponentVersionId.value) {
      return;
    }
    componentEditError.value = '';
    const result = componentBasicRequestFromForm(componentForm);
    if (!result.valid) {
      componentErrors.name =
        result.error === 'nameImage'
          ? componentForm.name.trim()
            ? ''
            : t('application.componentDetail.validation.componentNameRequired')
          : t('application.componentDetail.validation.componentName');
      componentErrors.image = componentForm.image.trim()
        ? ''
        : t('application.componentDetail.validation.imageRequired');
      return;
    }
    try {
      await executeOp(async () => {
        const updated = await applicationApi.updateVersionComponentBasic(
          pendingComponentVersionId.value,
          target.id,
          result.value
        );
        const components = versionComponents[pendingComponentVersionId.value];
        if (components) {
          const index = components.findIndex((item) => item.id === updated.id);
          if (index !== -1) {
            components.splice(index, 1, updated);
          }
        }
        pendingComponent.value = updated;
        cancelComponentEditing();
        toast.success(t('application.toast.updateSuccess'));
      });
    } catch (error) {
      componentEditError.value =
        error instanceof Error ? error.message : t('application.toast.updateFailed');
    }
  }

  function openComponentDeleteDialog(versionId: string, component: VersionComponentResp) {
    pendingComponentVersionId.value = versionId;
    pendingComponent.value = component;
    componentDeleteError.value = '';
    isComponentDeleteDialogOpen.value = true;
  }

  function cancelComponentDeletion() {
    isComponentDeleteDialogOpen.value = false;
    pendingComponent.value = undefined;
    pendingComponentVersionId.value = '';
    componentDeleteError.value = '';
  }

  async function deleteComponent() {
    const target = pendingComponent.value;
    if (!target || !pendingComponentVersionId.value) {
      return;
    }
    componentDeleteError.value = '';
    try {
      await executeOp(async () => {
        await applicationApi.deleteVersionComponent(pendingComponentVersionId.value, target.id);
        const components = versionComponents[pendingComponentVersionId.value];
        if (components) {
          const index = components.findIndex((item) => item.id === target.id);
          if (index !== -1) {
            components.splice(index, 1);
          }
        }
        cancelComponentDeletion();
        toast.success(t('application.toast.updateSuccess'));
      });
    } catch (error) {
      componentDeleteError.value =
        error instanceof Error ? error.message : t('application.toast.updateFailed');
    }
  }

  function openForkModal(version: VersionResp) {
    forkingVersionId.value = version.id;
    forkLabel.value = `${version.label}-copy`;
    forkLabelError.value = '';
    forkSubmitError.value = '';
    isForkDialogOpen.value = true;
  }

  async function handleForkOk() {
    forkSubmitError.value = '';
    forkLabelError.value = forkLabel.value.trim()
      ? ''
      : t('application.validation.versionLabelRequired');
    if (forkLabelError.value || !forkingVersionId.value) {
      return;
    }
    try {
      await executeOp(async () => {
        const created = await applicationApi.forkVersion(forkingVersionId.value, {
          label: forkLabel.value.trim(),
        });
        toast.success(t('application.toast.forkSuccess'));
        isForkDialogOpen.value = false;
        router.push(`/version/${created.id}`);
      });
    } catch (error) {
      forkSubmitError.value =
        error instanceof Error ? error.message : t('application.toast.forkFailed');
    }
  }

  onMounted(async () => {
    await fetchApplication();
    if (application.value) {
      await loadVersions();
    }
  });
</script>
