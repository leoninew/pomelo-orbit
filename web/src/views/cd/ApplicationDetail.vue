<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-xl font-semibold text-foreground flex items-center gap-2">
        {{ application?.name || t('application.detail.title') }}
        <AppBadge v-if="application" variant="pill" :tone="statusTone">
          {{ statusText }}
        </AppBadge>
      </h1>
      <div class="flex flex-wrap items-center gap-2">
        <button v-if="application" class="app-button-primary h-9 px-3" @click="openEditModal">
          <Pencil class="size-4" />
          {{ t('common.edit') }}
        </button>
        <DropdownMenuRoot v-if="application">
          <div class="inline-flex h-9 overflow-hidden rounded-md">
            <button
              :disabled="operating"
              class="app-button-primary h-9 rounded-r-none px-3"
              @click="handleDeploy()"
            >
              <Play class="size-4" />
              {{ t('application.deploy') }}
            </button>
            <DropdownMenuTrigger as-child>
              <button
                type="button"
                :disabled="operating"
                class="app-button-primary h-9 rounded-l-none border-l border-primary-foreground/20 px-2"
                :aria-label="t('application.detail.actions.deployOptions')"
              >
                <ChevronDown class="size-4" />
              </button>
            </DropdownMenuTrigger>
          </div>
          <DropdownMenuPortal>
            <DropdownMenuContent
              class="z-50 min-w-48 rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-lg outline-none data-[state=open]:animate-slideDownAndFade"
              align="end"
              :side-offset="8"
            >
              <DropdownMenuItem
                class="flex cursor-pointer items-center gap-2 rounded px-3 py-2 text-sm outline-none transition-colors data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground"
                @select="handleDeploy(true)"
              >
                <Play class="size-4" />
                {{ t('application.detail.actions.deployForceRecreate') }}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenuPortal>
        </DropdownMenuRoot>
        <button
          v-if="application"
          :disabled="operating"
          class="app-button h-9 px-3"
          @click="handleStop"
        >
          <Square class="size-4" />
          {{ t('application.stop') }}
        </button>
        <button
          v-if="application"
          :disabled="operating"
          class="app-button h-9 px-3"
          @click="handleRestart"
        >
          <RotateCcw class="size-4" />
          {{ t('application.detail.actions.restart') }}
        </button>
        <button v-if="application" class="app-button h-9 px-3" @click="handleExport">
          <Download class="size-4" />
          {{ t('application.detail.actions.export') }}
        </button>
        <button
          v-if="application"
          :disabled="
            operating || application.status === 'deployed' || application.status === 'deploying'
          "
          class="app-button-danger h-9 px-3"
          @click="openDeleteModal"
        >
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/cd/applications')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <!-- 加载状态 -->
    <AppSpinner v-if="basicInfoLoading" class="py-12" />

    <!-- 内容 -->
    <div v-else-if="application" class="flex flex-col gap-4">
      <!-- 基本信息卡片 -->
      <div class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">
            {{ t('application.detail.sections.basicInfo') }}
          </h2>
        </div>
        <dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.name') }}
            </dt>
            <dd class="text-foreground">{{ application.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.code') }}
            </dt>
            <dd class="text-foreground">{{ application.code }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('common.status') }}
            </dt>
            <dd>
              <AppBadge variant="pill" :tone="statusTone">
                {{ statusText }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.imagePullPolicy') }}
            </dt>
            <dd class="text-foreground">
              {{ t('application.imagePullPolicyLabels.' + application.image_pull_policy) }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.routeManaged') }}
            </dt>
            <dd class="text-foreground">
              {{
                application.route_managed
                  ? t('application.routeManagedLabels.enabled')
                  : t('application.routeManagedLabels.disabled')
              }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.detail.fields.deployments') }}
            </dt>
            <dd>
              <router-link
                :to="`/cd/deployments?application_id=${application.id}`"
                class="text-primary hover:underline"
              >
                {{ t('application.detail.actions.viewAllDeployments') }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('common.createdAt') }}
            </dt>
            <dd class="text-muted-foreground">{{ formatTime(application.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('common.updatedAt') }}
            </dt>
            <dd class="text-muted-foreground">{{ formatTime(application.updated_at) }}</dd>
          </div>
        </dl>
      </div>

      <!-- 配置文件 -->
      <div class="app-surface">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="font-semibold text-foreground">
            {{ t('application.detail.sections.configFiles') }}
          </h2>
          <div class="flex items-center gap-2">
            <button class="app-button-primary h-8 px-3" @click="openAddFileDrawer">
              <Plus class="size-4" />
              {{ t('application.detail.actions.addFile') }}
            </button>
            <button class="app-button h-8 px-3" @click="openComposePreview">
              <Eye class="size-4" />
              {{ t('application.detail.actions.preview') }}
            </button>
          </div>
        </div>
        <div class="overflow-x-auto">
          <table class="app-table-detail min-w-[760px]">
            <thead>
              <tr>
                <th>{{ t('application.detail.fields.filePath') }}</th>
                <th>{{ t('common.createdAt') }}</th>
                <th>{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="fileListLoading">
                <td colspan="3" class="text-center text-muted-foreground">
                  <AppSpinner />
                </td>
              </tr>
              <tr v-else-if="files.length === 0">
                <td colspan="3" class="text-center text-muted-foreground">
                  {{ t('application.detail.empty.configFiles') }}
                </td>
              </tr>
              <tr v-for="file in files" :key="file.id">
                <td class="max-w-md truncate text-foreground" :title="file.path">
                  {{ file.path }}
                </td>
                <td class="text-muted-foreground">{{ formatTime(file.created_at) }}</td>
                <td>
                  <div class="flex items-center gap-3">
                    <button class="app-link" @click="openFileDrawer(file.id, false)">
                      {{ t('application.view') }}
                    </button>
                    <button class="app-link" @click="openFileDrawer(file.id, true)">
                      {{ t('common.edit') }}
                    </button>
                    <button class="app-link-danger" @click="confirmDeleteFile(file.id)">
                      {{ t('common.delete') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 镜像配置 -->
      <div class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">
            {{ t('application.detail.sections.imageConfig') }}
          </h2>
        </div>
        <div class="overflow-x-auto">
          <table class="app-table-detail min-w-[960px]">
            <thead>
              <tr>
                <th>{{ t('application.detail.fields.service') }}</th>
                <th>{{ t('application.detail.fields.defaultDomain') }}</th>
                <th>{{ t('application.detail.fields.defaultPort') }}</th>
                <th>{{ t('application.detail.fields.baseImage') }}</th>
                <th>{{ t('application.detail.fields.currentImage') }}</th>
                <th>{{ t('application.detail.fields.overridden') }}</th>
                <th>{{ t('common.updatedAt') }}</th>
                <th>{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="serviceConfigListLoading">
                <td colspan="8" class="text-center text-muted-foreground">
                  <AppSpinner />
                </td>
              </tr>
              <tr v-else-if="serviceConfigError">
                <td colspan="8" class="text-center text-destructive">
                  {{ serviceConfigError }}
                </td>
              </tr>
              <tr v-else-if="serviceConfigs.length === 0">
                <td colspan="8" class="text-center text-muted-foreground">
                  {{ t('application.detail.empty.imageConfig') }}
                </td>
              </tr>
              <tr v-for="config in serviceConfigs" :key="config.service_name">
                <td class="text-foreground">{{ config.service_name }}</td>
                <td class="text-muted-foreground">{{ config.default_domain }}</td>
                <td class="text-muted-foreground">{{ config.default_port }}</td>
                <td
                  class="max-w-xs truncate text-muted-foreground"
                  :title="config.base_image ?? ''"
                >
                  {{ config.base_image || '—' }}
                </td>
                <td
                  class="max-w-xs truncate text-foreground"
                  :title="getServiceDisplayImage(config)"
                >
                  {{ getServiceDisplayImage(config) || '—' }}
                </td>
                <td>
                  <AppBadge v-if="isServiceOverridden(config)" variant="pill" tone="primary">
                    {{ t('application.detail.fields.overridden') }}
                  </AppBadge>
                  <span v-else class="text-muted-foreground">—</span>
                </td>
                <td class="text-muted-foreground">
                  {{ config.updated_at ? formatTime(config.updated_at) : '—' }}
                </td>
                <td>
                  <div class="flex items-center gap-3">
                    <button class="app-link" @click="openEditServiceConfigModal(config)">
                      {{ t('common.edit') }}
                    </button>
                    <button
                      v-if="canResetServiceConfig(config)"
                      class="app-link-danger"
                      @click="confirmResetServiceConfig(config)"
                    >
                      {{ t('common.reset') }}
                    </button>
                    <span v-else class="text-muted-foreground">—</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 路由配置 -->
      <div class="app-surface">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="font-semibold text-foreground">
            {{ t('application.detail.sections.routeConfig') }}
          </h2>
          <button
            class="app-button-primary h-8 px-3"
            :disabled="!application.route_managed"
            :title="
              !application.route_managed
                ? t('application.detail.hints.routeDisabledTitle')
                : undefined
            "
            @click="openAddRouteModal"
          >
            <Plus class="size-4" />
            {{ t('application.detail.actions.addRoute') }}
          </button>
        </div>
        <div v-if="!application.route_managed" class="px-5 pt-4">
          <div class="app-tip">{{ t('application.detail.hints.routeDisabledTip') }}</div>
        </div>
        <div class="overflow-x-auto">
          <table class="app-table-detail min-w-[760px]">
            <thead>
              <tr>
                <th>{{ t('application.detail.fields.domain') }}</th>
                <th>{{ t('application.detail.fields.service') }}</th>
                <th>{{ t('application.detail.fields.port') }}</th>
                <th>{{ t('common.createdAt') }}</th>
                <th>{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="routeListLoading">
                <td colspan="5" class="text-center text-muted-foreground">
                  <AppSpinner />
                </td>
              </tr>
              <tr v-else-if="appRoutes.length === 0">
                <td colspan="5" class="text-center text-muted-foreground">
                  {{ t('application.detail.empty.routeConfig') }}
                </td>
              </tr>
              <tr v-for="r in appRoutes" :key="r.id">
                <td class="text-foreground">{{ r.domain }}</td>
                <td class="text-muted-foreground">{{ r.service_name }}</td>
                <td class="text-muted-foreground">{{ r.port }}</td>
                <td class="text-muted-foreground">{{ formatTime(r.created_at) }}</td>
                <td>
                  <div class="flex items-center gap-3">
                    <button
                      class="app-link"
                      :disabled="!application.route_managed"
                      @click="openEditRouteModal(r)"
                    >
                      {{ t('common.edit') }}
                    </button>
                    <button class="app-link-danger" @click="confirmDeleteRoute(r.id)">
                      {{ t('common.delete') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <AppDrawer
      :open="fileDrawerVisible"
      :title="fileDrawerTitle"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="handleFileDrawerOpenChange"
    >
      <div class="flex h-full flex-col gap-4 p-6">
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('application.detail.fields.filePath') }}</label>
          <input
            v-model="currentFilePath"
            type="text"
            :disabled="!isEditingInDrawer"
            :placeholder="t('application.detail.placeholders.filePath')"
            class="app-input"
          />
        </div>
        <div class="min-h-0 flex-1">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.fileContent') }}
          </label>
          <div v-if="fileContentLoading" class="flex h-full items-center justify-center">
            <AppSpinner />
          </div>
          <MonacoEditor
            v-else
            v-model="currentFileContent"
            language="yaml"
            height="100%"
            :readonly="!isEditingInDrawer"
          />
        </div>
      </div>
      <template #footer>
        <template v-if="!isEditingInDrawer && currentFileId">
          <button class="app-button" @click="handleDrawerClose">
            {{ t('application.detail.actions.close') }}
          </button>
          <button
            class="app-button-primary"
            :disabled="fileContentLoading"
            @click="isEditingInDrawer = true"
          >
            {{ t('common.edit') }}
          </button>
        </template>
        <template v-else>
          <button class="app-button" @click="handleDrawerClose">{{ t('common.cancel') }}</button>
          <button
            :disabled="fileContentLoading"
            class="app-button-primary"
            @click="saveCurrentFile"
          >
            {{ t('common.save') }}
          </button>
        </template>
      </template>
    </AppDrawer>

    <AppDialog
      v-model:open="isEditDialogOpen"
      :title="t('application.detail.dialog.editApplication')"
    >
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('application.name') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="editForm.name"
          type="text"
          class="app-input"
          :class="editErrors.name ? 'app-input-error' : ''"
        />
        <p v-if="editErrors.name" class="app-field-error mt-1 text-xs">{{ editErrors.name }}</p>
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">{{ t('application.code') }}</label>
        <input v-model="editForm.code" type="text" disabled class="app-input" />
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">{{ t('application.imagePullPolicy') }}</label>
        <SelectControl v-model="editForm.image_pull_policy" :options="imagePullPolicyOptions" />
      </div>
      <div class="space-y-1.5">
        <label class="flex items-center gap-2">
          <input v-model="editForm.route_managed" type="checkbox" class="app-checkbox" />
          <span class="text-sm font-medium text-foreground">
            {{ t('application.enableRouteManaged') }}
          </span>
        </label>
        <p class="app-field-hint ml-6">{{ t('application.detail.hints.editRouteManaged') }}</p>
      </div>
      <template #footer>
        <button class="app-button" @click="isEditDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-primary" @click="handleEditOk">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
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
      <template #footer>
        <button class="app-button" @click="isDeleteDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-destructive" @click="handleDeleteOk">
          {{ t('common.delete') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteFileDialogOpen"
      :title="t('application.detail.dialog.confirmDelete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{ t('application.detail.dialog.deleteConfigFileRespConfirm') }}
      </p>
      <template #footer>
        <button class="app-button" @click="isDeleteFileDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button
          :disabled="fileListLoading"
          class="app-button-destructive"
          @click="executeDeleteFile"
        >
          {{ t('common.delete') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isServiceConfigDialogOpen"
      :title="t('application.detail.dialog.editServiceImage')"
    >
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('application.detail.fields.service') }}
        </label>
        <input :value="selectedServiceName" type="text" disabled class="app-input" />
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('application.detail.fields.image') }}
        </label>
        <input
          v-model="serviceConfigForm.image"
          type="text"
          class="app-input"
          :placeholder="t('application.detail.placeholders.image')"
        />
        <p class="app-field-hint mt-1.5">{{ t('application.detail.hints.serviceImageEmpty') }}</p>
      </div>
      <template #footer>
        <button class="app-button" @click="isServiceConfigDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button
          :disabled="!serviceConfigDirty || serviceConfigSaving"
          class="app-button-primary"
          @click="saveServiceConfig"
        >
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteServiceConfigDialogOpen"
      :title="t('application.detail.dialog.confirmReset')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{
          t('application.detail.dialog.resetServiceImageConfirm', {
            name: pendingDeleteServiceName || '-',
          })
        }}
      </p>
      <template #footer>
        <button class="app-button" @click="cancelResetServiceConfig">
          {{ t('common.cancel') }}
        </button>
        <button
          :disabled="serviceConfigSaving"
          class="app-button-destructive"
          @click="executeResetServiceConfig"
        >
          {{ t('common.reset') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isRouteDialogOpen"
      :title="
        editingRouteId
          ? t('application.detail.dialog.editRoute')
          : t('application.detail.dialog.addRoute')
      "
    >
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('application.detail.fields.service') }}
          <span class="text-destructive">*</span>
        </label>
        <ComboboxSelect
          :model-value="routeForm.service_name"
          :options="composeServiceOptions"
          :invalid="Boolean(routeFormErrors.service_name)"
          :portal="false"
          :placeholder="t('application.detail.placeholders.serviceSearch')"
          :empty-text="t('application.detail.placeholders.noService')"
          @update:model-value="handleRouteServiceChange"
        />
        <p v-if="routeFormErrors.service_name" class="app-field-error mt-1 text-xs">
          {{ routeFormErrors.service_name }}
        </p>
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('application.detail.fields.domain') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="routeForm.domain"
          type="text"
          placeholder="example.com"
          class="app-input"
          :class="routeFormErrors.domain ? 'app-input-error' : ''"
        />
        <p v-if="routeFormErrors.domain" class="app-field-error mt-1 text-xs">
          {{ routeFormErrors.domain }}
        </p>
      </div>
      <div>
        <label class="app-field-label mb-1.5 block">
          {{ t('application.detail.fields.containerPort') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model.number="routeForm.port"
          type="number"
          min="1"
          max="65535"
          :placeholder="t('application.detail.placeholders.port')"
          class="app-input"
          :class="routeFormErrors.port ? 'app-input-error' : ''"
        />
        <p v-if="routeFormErrors.port" class="app-field-error mt-1 text-xs">
          {{ routeFormErrors.port }}
        </p>
      </div>
      <template #footer>
        <button class="app-button" @click="isRouteDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="routeLoading" class="app-button-primary" @click="handleRouteOk">
          {{ editingRouteId ? t('common.save') : t('common.add') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteRouteDialogOpen"
      :title="t('application.detail.dialog.confirmDelete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{ t('application.detail.dialog.deleteRouteConfirm') }}
      </p>
      <template #footer>
        <button class="app-button" @click="isDeleteRouteDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="routeLoading" class="app-button-destructive" @click="executeDeleteRoute">
          {{ t('common.delete') }}
        </button>
      </template>
    </AppDialog>

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
  import {
    ArrowLeft,
    ChevronDown,
    Download,
    Eye,
    Pencil,
    Play,
    Plus,
    RotateCcw,
    Square,
    Trash2,
  } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import {
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuPortal,
    DropdownMenuRoot,
    DropdownMenuTrigger,
  } from 'reka-ui';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/cd/application';
  import { deploymentApi } from '@/api/cd/deployments';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import ComboboxSelect, { type ComboboxOptionValue } from '@/components/ComboboxSelect.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ApplicationResp } from '@/gen/orbit/api/v1/application';
  import type { ApplicationRouteResp } from '@/gen/orbit/api/v1/application_route';
  import type { ConfigFileResp } from '@/gen/orbit/api/v1/config_file';
  import type {
    ApplicationServiceConfigResp,
    ComposeServiceResp,
  } from '@/gen/orbit/api/v1/service_config';
  import { appStatusTone } from '@/utils/status';
  import { delayAsync, formatTime } from '@/utils/time';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const applicationId = route.params.id as string;
  const toast = useToast();

  const { loading: basicInfoLoading, execute: executeBasicInfo } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { loading: fileListLoading, execute: executeFileList } = useStatusAsync();
  const { loading: fileContentLoading, execute: executeFileContent } = useStatusAsync();
  const { loading: routeListLoading, execute: executeRouteList } = useStatusAsync();
  const { loading: routeLoading, execute: executeRoute } = useStatusAsync();
  const { loading: serviceConfigListLoading, execute: executeServiceConfigList } = useStatusAsync();
  const { loading: serviceConfigSaving, execute: executeServiceConfigSave } = useStatusAsync();
  const { loading: composePreviewLoading, execute: executeComposePreview } = useStatusAsync();

  const application = ref<ApplicationResp>();
  const files = ref<ConfigFileResp[]>([]);
  const appRoutes = ref<ApplicationRouteResp[]>([]);
  const composeServices = ref<ComposeServiceResp[]>([]);
  const serviceConfigs = ref<ApplicationServiceConfigResp[]>([]);
  const selectedServiceName = ref('');
  const serviceConfigError = ref('');

  const isEditDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isDeleteFileDialogOpen = ref(false);
  const isServiceConfigDialogOpen = ref(false);
  const isDeleteServiceConfigDialogOpen = ref(false);
  const isRouteDialogOpen = ref(false);
  const isDeleteRouteDialogOpen = ref(false);
  const composePreviewDrawerOpen = ref(false);
  const pendingDeleteFileId = ref('');
  const pendingDeleteServiceName = ref('');
  const pendingDeleteRouteId = ref('');
  const editingRouteId = ref('');
  const deleteDir = ref(false);
  const composePreviewYaml = ref('');
  const composePreviewError = ref('');

  const routeForm = reactive({ service_name: '', domain: '', port: 80 });
  const routeFormErrors = reactive({ service_name: '', domain: '', port: '' });
  const routeDomainPattern =
    /^(localhost|([a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)$/i;
  const serviceConfigForm = reactive({ image: '' });

  const fileDrawerVisible = ref(false);
  const currentFileId = ref('');
  const currentFilePath = ref('');
  const currentFileContent = ref('');
  const isEditingInDrawer = ref(false);

  const editForm = reactive({
    name: '',
    code: '',
    image_pull_policy: 'missing',
    route_managed: false,
  });
  const editErrors = reactive({ name: '' });
  const imagePullPolicyOptions = computed(() => [
    { value: 'missing', label: t('application.imagePullPolicyOptions.missing') },
    { value: 'always', label: t('application.imagePullPolicyOptions.always') },
    { value: 'never', label: t('application.imagePullPolicyOptions.never') },
  ]);

  const activeServiceConfig = computed(() =>
    serviceConfigs.value.find((item) => item.service_name === selectedServiceName.value)
  );
  const currentServiceImage = computed(() =>
    activeServiceConfig.value ? getServiceDisplayImage(activeServiceConfig.value) : ''
  );

  const composeServiceOptions = computed(() =>
    composeServices.value.map((service) => ({
      value: service.service_name,
      label: service.service_name,
      description: `${service.default_domain}:${service.default_port}`,
    }))
  );
  const fileDrawerTitle = computed(() => {
    if (!currentFileId.value) {
      return t('application.detail.drawer.addFile');
    }
    const action = isEditingInDrawer.value
      ? t('application.detail.drawer.editFile')
      : t('application.detail.drawer.viewFile');
    return currentFilePath.value ? `${action}: ${currentFilePath.value}` : action;
  });
  const serviceConfigDirty = computed(
    () => serviceConfigForm.image.trim() !== currentServiceImage.value.trim()
  );

  const statusTone = computed(() =>
    application.value ? appStatusTone(application.value.status) : 'default'
  );
  const statusText = computed(() =>
    application.value ? t('status.' + application.value.status) : ''
  );

  async function fetchApplication() {
    try {
      await executeBasicInfo(async () => {
        const data = await applicationApi.get(applicationId);
        application.value = data;
        Object.assign(editForm, {
          name: data.name,
          code: data.code,
          image_pull_policy: data.image_pull_policy,
          route_managed: data.route_managed,
        });
      });
      if (application.value?.status === 'deploying') {
        pollActiveDeployment();
      }
    } catch {
      toast.error(t('application.toast.loadDetailFailed'));
      router.push('/cd/applications');
    }
  }

  async function pollActiveDeployment() {
    try {
      const resp = await deploymentApi.list({
        application_id: applicationId,
        per_page: 1,
      });
      const latest = resp.items[0];
      if (!latest) {
        return;
      }
      while (true) {
        await delayAsync(3000);
        try {
          const detail = await deploymentApi.get(latest.id);
          if (['ran_to_completion', 'faulted', 'canceled'].includes(detail.status)) {
            if (application.value) {
              application.value.status =
                detail.status === 'ran_to_completion' ? 'deployed' : 'deploy_failed';
            }
            break;
          }
        } catch {
          break;
        }
      }
    } catch {
      /* silent */
    }
  }

  async function handleDeploy(forceRecreate = false) {
    try {
      await executeOp(async () => {
        const res = await applicationApi.deploy(applicationId, { force_recreate: forceRecreate });
        toast.success(t('application.toast.deployTriggeredDetail'));
        router.push({
          path: `/cd/deployments/${res.deployment_id}`,
          query: { from: 'application' },
        });
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.deployFailed'));
    }
  }

  async function handleStop() {
    try {
      await executeOp(async () => {
        const res = await applicationApi.stop(applicationId, { remove_volumes: false });
        toast.success(t('application.toast.stopSubmitted'));
        const maxAttempts = 20; // 最多轮询 20 次（60 秒）
        let attempts = 0;
        while (attempts < maxAttempts) {
          await delayAsync(3000);
          attempts++;
          try {
            const detail = await deploymentApi.get(res.deployment_id);
            if (['ran_to_completion', 'faulted', 'canceled'].includes(detail.status)) {
              await fetchApplication(); // 重新加载应用状态
              break;
            }
          } catch {
            break;
          }
        }
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.stopFailed'));
    }
  }

  async function handleRestart() {
    try {
      await executeOp(async () => {
        const res = await applicationApi.restart(applicationId, {});
        toast.success(t('application.toast.restartSubmitted'));
        router.push({
          path: `/cd/deployments/${res.deployment_id}`,
          query: { from: 'application' },
        });
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.restartFailed'));
    }
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

  function openEditModal() {
    editErrors.name = '';
    if (application.value) {
      Object.assign(editForm, {
        name: application.value.name,
        code: application.value.code,
        image_pull_policy: application.value.image_pull_policy,
        route_managed: application.value.route_managed,
      });
    }
    isEditDialogOpen.value = true;
  }

  async function handleEditOk() {
    editErrors.name = editForm.name.trim() ? '' : t('application.validation.nameRequired');
    if (editErrors.name) {
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.update(applicationId, {
          name: editForm.name,
          image_pull_policy: editForm.image_pull_policy,
          route_managed: editForm.route_managed,
        });
        toast.success(t('application.toast.updateSuccess'));
        isEditDialogOpen.value = false;
        await fetchApplication();
        if (application.value?.route_managed) {
          await loadRoutes();
        } else {
          appRoutes.value = [];
        }
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  function openDeleteModal() {
    deleteDir.value = false;
    isDeleteDialogOpen.value = true;
  }

  async function handleDeleteOk() {
    try {
      await executeOp(async () => {
        await applicationApi.delete(applicationId, deleteDir.value);
        toast.success(t('application.toast.deleteSuccess'));
        router.push('/cd/applications');
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.deleteFailed'));
    }
  }

  async function loadFiles() {
    try {
      await executeFileList(async () => {
        const resp = await applicationApi.listFiles(applicationId);
        files.value = resp.items;
      });
    } catch {
      toast.error(t('application.toast.loadConfigFileRespsFailed'));
    }
  }

  async function loadServiceConfigs() {
    serviceConfigError.value = '';
    try {
      await executeServiceConfigList(async () => {
        const resp = await applicationApi.listServiceConfigs(applicationId);
        serviceConfigs.value = resp.items;
      });
    } catch (error) {
      serviceConfigs.value = [];
      serviceConfigError.value =
        error instanceof Error ? error.message : t('application.toast.loadServiceConfigFailed');
    }
  }

  function getServiceDisplayImage(serviceConfig: ApplicationServiceConfigResp) {
    return serviceConfig.image?.trim() || serviceConfig.base_image || '';
  }

  function isServiceOverridden(serviceConfig: ApplicationServiceConfigResp) {
    const overrideImage = serviceConfig.image?.trim() || '';
    const baseImage = serviceConfig.base_image?.trim() || '';
    return Boolean(overrideImage) && overrideImage !== baseImage;
  }

  function canResetServiceConfig(serviceConfig: ApplicationServiceConfigResp) {
    return Boolean(serviceConfig.image?.trim());
  }

  function openEditServiceConfigModal(serviceConfig: ApplicationServiceConfigResp) {
    selectedServiceName.value = serviceConfig.service_name;
    serviceConfigForm.image = getServiceDisplayImage(serviceConfig);
    isServiceConfigDialogOpen.value = true;
  }

  function confirmResetServiceConfig(serviceConfig: ApplicationServiceConfigResp) {
    if (!canResetServiceConfig(serviceConfig)) {
      return;
    }
    pendingDeleteServiceName.value = serviceConfig.service_name;
    isDeleteServiceConfigDialogOpen.value = true;
  }

  function cancelResetServiceConfig() {
    pendingDeleteServiceName.value = '';
    isDeleteServiceConfigDialogOpen.value = false;
  }

  async function executeResetServiceConfig() {
    if (!pendingDeleteServiceName.value) {
      return;
    }
    try {
      await executeServiceConfigSave(async () => {
        const saved = await applicationApi.updateServiceConfig(
          applicationId,
          pendingDeleteServiceName.value,
          {}
        );
        const idx = serviceConfigs.value.findIndex(
          (item) => item.service_name === saved.service_name
        );
        if (idx >= 0) {
          serviceConfigs.value[idx] = saved;
        } else {
          serviceConfigs.value.push(saved);
        }
        if (selectedServiceName.value === saved.service_name) {
          serviceConfigForm.image = getServiceDisplayImage(saved);
          isServiceConfigDialogOpen.value = false;
        }
        isDeleteServiceConfigDialogOpen.value = false;
        pendingDeleteServiceName.value = '';
        toast.success(t('application.toast.resetServiceImageSuccess'));
      });
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('application.toast.resetServiceImageFailed')
      );
    }
  }

  async function saveServiceConfig() {
    const active = activeServiceConfig.value;
    if (!active || !serviceConfigDirty.value) {
      return;
    }
    try {
      await executeServiceConfigSave(async () => {
        const saved = await applicationApi.updateServiceConfig(
          applicationId,
          active.service_name,
          { image: serviceConfigForm.image }
        );
        const idx = serviceConfigs.value.findIndex(
          (item) => item.service_name === saved.service_name
        );
        if (idx >= 0) {
          serviceConfigs.value[idx] = saved;
        } else {
          serviceConfigs.value.push(saved);
        }
        selectedServiceName.value = saved.service_name;
        serviceConfigForm.image = getServiceDisplayImage(saved);
        isServiceConfigDialogOpen.value = false;
        toast.success(t('application.toast.saveServiceImageSuccess'));
      });
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('application.toast.saveServiceImageFailed')
      );
    }
  }

  async function openFileDrawer(fileId: string, isEdit = false) {
    currentFileId.value = fileId;
    isEditingInDrawer.value = isEdit;
    currentFileContent.value = '';
    currentFilePath.value = '';
    fileDrawerVisible.value = true;
    try {
      await executeFileContent(async () => {
        const result = await applicationApi.readFile(applicationId, fileId);
        currentFileContent.value = result.content ?? '';
        currentFilePath.value = result.path || '';
      });
    } catch {
      toast.error(t('application.toast.loadFileContentFailed'));
    }
  }

  function openAddFileDrawer() {
    currentFileId.value = '';
    currentFilePath.value = '';
    currentFileContent.value = '';
    isEditingInDrawer.value = true;
    fileDrawerVisible.value = true;
  }

  function handleDrawerClose() {
    fileDrawerVisible.value = false;
    isEditingInDrawer.value = false;
  }

  function handleFileDrawerOpenChange(open: boolean) {
    fileDrawerVisible.value = open;
    if (!open) {
      isEditingInDrawer.value = false;
    }
  }

  async function saveCurrentFile() {
    if (!currentFilePath.value.trim()) {
      toast.error(t('application.validation.filePathRequired'));
      return;
    }
    const lowerPath = currentFilePath.value.toLowerCase();
    const content =
      lowerPath.endsWith('.sh') || lowerPath.endsWith('.bash')
        ? currentFileContent.value.replace(/\r\n/g, '\n')
        : currentFileContent.value;
    try {
      await executeFileContent(async () => {
        if (currentFileId.value) {
          const updated = await applicationApi.writeFile(applicationId, currentFileId.value, {
            path: currentFilePath.value,
            content,
          });
          const idx = files.value.findIndex((f) => f.id === currentFileId.value);
          if (idx >= 0) {
            files.value[idx] = updated;
          }
          toast.success(t('application.toast.saveSuccess'));
        } else {
          await applicationApi.createFile(applicationId, { path: currentFilePath.value, content });
          toast.success(t('application.toast.addSuccess'));
        }
        fileDrawerVisible.value = false;
        isEditingInDrawer.value = false;
        await loadFiles();
        await loadServiceConfigs();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.saveFailed'));
    }
  }

  function confirmDeleteFile(fileId: string) {
    pendingDeleteFileId.value = fileId;
    isDeleteFileDialogOpen.value = true;
  }

  async function executeDeleteFile() {
    try {
      await executeFileList(async () => {
        await applicationApi.deleteFile(applicationId, pendingDeleteFileId.value);
        toast.success(t('application.toast.deleteSuccess'));
        isDeleteFileDialogOpen.value = false;
        await loadFiles();
        await loadServiceConfigs();
      });
    } catch {
      toast.error(t('application.toast.deleteFailed'));
    }
  }

  async function openComposePreview() {
    composePreviewYaml.value = '';
    composePreviewError.value = '';
    composePreviewDrawerOpen.value = true;
    try {
      await executeComposePreview(async () => {
        const { compose_yaml } = await applicationApi.previewCompose(applicationId, {});
        composePreviewYaml.value = compose_yaml;
      });
    } catch (error) {
      composePreviewError.value =
        error instanceof Error ? error.message : t('application.toast.loadPreviewFailed');
    }
  }

  function handleComposePreviewDrawerOpenChange(open: boolean) {
    composePreviewDrawerOpen.value = open;
    if (!open) {
      composePreviewYaml.value = '';
      composePreviewError.value = '';
    }
  }

  // ── Route management ──

  async function loadRoutes() {
    try {
      await executeRouteList(async () => {
        const resp = await applicationApi.listRoutes(applicationId);
        appRoutes.value = resp.items;
      });
    } catch {
      toast.error(t('application.toast.loadRouteConfigFailed'));
    }
  }

  async function loadComposeServices() {
    composeServices.value = [];
    try {
      const resp = await applicationApi.listComposeServices(applicationId);
      composeServices.value = resp.items;
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('application.toast.parseComposeFailed')
      );
    }
  }

  function handleRouteServiceChange(value: ComboboxOptionValue) {
    const serviceName = String(value || '');
    const service = composeServices.value.find((item) => item.service_name === serviceName);
    if (service) {
      routeForm.service_name = service.service_name;
      routeForm.domain = service.default_domain;
      routeForm.port = service.default_port;
      routeFormErrors.service_name = '';
    } else {
      routeForm.service_name = '';
    }
  }

  async function openAddRouteModal() {
    if (!application.value?.route_managed) {
      return;
    }
    editingRouteId.value = '';
    Object.assign(routeForm, { service_name: '', domain: '', port: 80 });
    Object.assign(routeFormErrors, { service_name: '', domain: '', port: '' });
    await loadComposeServices();
    if (composeServices.value.length === 0) {
      return;
    }
    isRouteDialogOpen.value = true;
  }

  async function openEditRouteModal(r: ApplicationRouteResp) {
    if (!application.value?.route_managed) {
      return;
    }
    editingRouteId.value = r.id;
    Object.assign(routeForm, { service_name: r.service_name, domain: r.domain, port: r.port });
    Object.assign(routeFormErrors, { service_name: '', domain: '', port: '' });
    await loadComposeServices();
    if (composeServices.value.length === 0) {
      return;
    }
    isRouteDialogOpen.value = true;
  }

  function validateRouteForm() {
    const domain = routeForm.domain.trim();
    routeFormErrors.service_name = routeForm.service_name
      ? ''
      : t('application.validation.serviceRequired');
    routeFormErrors.domain = isValidRouteDomain(domain)
      ? ''
      : t('application.validation.domainInvalid');
    routeFormErrors.port =
      routeForm.port >= 1 && routeForm.port <= 65535 ? '' : t('application.validation.portRange');
    return !routeFormErrors.service_name && !routeFormErrors.domain && !routeFormErrors.port;
  }

  function isValidRouteDomain(domain: string) {
    return (
      domain.length > 0 &&
      domain.length <= 253 &&
      !/[\s`]/.test(domain) &&
      routeDomainPattern.test(domain)
    );
  }

  async function handleRouteOk() {
    if (!validateRouteForm()) {
      return;
    }
    try {
      await executeRoute(async () => {
        const data = {
          service_name: routeForm.service_name,
          domain: routeForm.domain.trim().toLowerCase(),
          port: routeForm.port,
        };
        if (editingRouteId.value) {
          await applicationApi.updateRoute(applicationId, editingRouteId.value, data);
        } else {
          await applicationApi.createRoute(applicationId, data);
        }
        toast.success(t('application.toast.saveSuccess'));
        isRouteDialogOpen.value = false;
        await loadRoutes();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.saveFailed'));
    }
  }

  function confirmDeleteRoute(routeId: string) {
    pendingDeleteRouteId.value = routeId;
    isDeleteRouteDialogOpen.value = true;
  }

  async function executeDeleteRoute() {
    try {
      await executeRoute(async () => {
        await applicationApi.deleteRoute(applicationId, pendingDeleteRouteId.value);
        toast.success(t('application.toast.deleteSuccess'));
        isDeleteRouteDialogOpen.value = false;
        await loadRoutes();
      });
    } catch {
      toast.error(t('application.toast.deleteFailed'));
    }
  }

  onMounted(async () => {
    await fetchApplication();
    await loadFiles();
    await loadServiceConfigs();
    if (application.value?.route_managed) {
      await loadRoutes();
    }
  });
</script>
