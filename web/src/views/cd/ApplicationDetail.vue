<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="flex items-center gap-2 text-xl font-semibold text-foreground">
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
              :disabled="operating || !deployVersionId"
              class="app-button-primary h-9 rounded-r-none px-3"
              @click="handleDeploy()"
            >
              <Play class="size-4" />
              {{ t('application.deploy') }}
            </button>
            <DropdownMenuTrigger as-child>
              <button
                type="button"
                :disabled="operating || !deployVersionId"
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
          :disabled="operating || !canStop"
          class="app-button h-9 px-3"
          @click="handleStop"
        >
          <Square class="size-4" />
          {{ t('application.stop') }}
        </button>
        <button
          v-if="application"
          :disabled="operating || !canRestart"
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
          :disabled="operating || !canDelete"
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

    <AppSpinner v-if="basicInfoLoading" class="py-12" />

    <div v-else-if="application" class="flex flex-col gap-4">
      <!-- 基本信息 -->
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
              {{ t('application.kind') }}
            </dt>
            <dd class="text-foreground">
              {{ t('application.kindLabels.' + (application.kind || 'standard')) }}
            </dd>
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
              {{ t('application.serviceCount') }}
            </dt>
            <dd class="text-foreground">{{ application.service_count ?? 0 }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 whitespace-nowrap text-muted-foreground">
              {{ t('application.detail.fields.currentVersion') }}
            </dt>
            <dd class="text-foreground">
              {{ currentVersionLabel }}
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

      <!-- 版本 -->
      <div class="app-surface">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="font-semibold text-foreground">
            {{ t('application.detail.sections.versions') }}
          </h2>
          <button class="app-button-primary h-8 px-3" @click="openCreateVersionModal">
            <Plus class="size-4" />
            {{ t('application.detail.actions.createVersion') }}
          </button>
        </div>
        <div class="overflow-x-auto">
          <table class="app-table-detail min-w-[960px]">
            <thead>
              <tr>
                <th class="w-10" />
                <th>{{ t('application.detail.fields.versionLabel') }}</th>
                <th>{{ t('common.status') }}</th>
                <th>{{ t('application.detail.fields.components') }}</th>
                <th>{{ t('application.detail.fields.note') }}</th>
                <th>{{ t('common.createdAt') }}</th>
                <th>{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="versionListLoading">
                <td colspan="7" class="text-center text-muted-foreground">
                  <AppSpinner />
                </td>
              </tr>
              <tr v-else-if="versions.length === 0">
                <td colspan="7" class="text-center text-muted-foreground">
                  {{ t('application.detail.empty.versions') }}
                </td>
              </tr>
              <tr v-for="version in versions" :key="version.id">
                <td>
                  <input
                    type="radio"
                    class="size-4 accent-primary"
                    name="selected-version"
                    :checked="selectedVersionId === version.id"
                    :aria-label="version.label"
                    @change="selectedVersionId = version.id"
                  />
                </td>
                <td class="text-foreground">
                  <span class="font-medium">{{ version.label }}</span>
                  <span
                    v-if="application.version_id === version.id"
                    class="ml-2 text-xs text-muted-foreground"
                  >
                    {{ t('application.detail.fields.boundVersion') }}
                  </span>
                </td>
                <td>
                  <AppBadge variant="pill" :tone="versionStatusTone(version.status)">
                    {{ t('status.' + version.status) }}
                  </AppBadge>
                </td>
                <td
                  class="max-w-xs truncate text-muted-foreground"
                  :title="componentSummary(version)"
                >
                  {{ componentSummary(version) }}
                </td>
                <td class="max-w-xs truncate text-muted-foreground" :title="version.note || ''">
                  {{ version.note || '—' }}
                </td>
                <td class="text-muted-foreground">{{ formatTime(version.created_at) }}</td>
                <td>
                  <div class="flex flex-wrap items-center gap-3">
                    <button class="app-link" @click="openPreview(version.id)">
                      {{ t('application.detail.actions.preview') }}
                    </button>
                    <button
                      class="app-link"
                      :disabled="operating"
                      @click="handleDeployVersion(version.id)"
                    >
                      {{ t('application.deploy') }}
                    </button>
                    <button
                      v-if="version.status === 'unpublished'"
                      class="app-link"
                      @click="openEditVersionModal(version)"
                    >
                      {{ t('common.edit') }}
                    </button>
                    <button
                      v-if="version.status === 'unpublished'"
                      class="app-link"
                      :disabled="operating"
                      @click="handlePublish(version.id)"
                    >
                      {{ t('application.detail.actions.publish') }}
                    </button>
                    <button class="app-link" @click="openForkModal(version)">
                      {{ t('application.detail.actions.fork') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Services -->
      <div class="app-surface">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="font-semibold text-foreground">
            {{ t('application.detail.sections.services') }}
          </h2>
        </div>
        <div class="overflow-x-auto">
          <table class="app-table-detail min-w-[760px]">
            <thead>
              <tr>
                <th>{{ t('application.detail.fields.environment') }}</th>
                <th>{{ t('application.detail.fields.instanceKey') }}</th>
                <th>{{ t('common.status') }}</th>
                <th>{{ t('application.detail.fields.versionLabel') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="serviceListLoading">
                <td colspan="4" class="text-center text-muted-foreground">
                  <AppSpinner />
                </td>
              </tr>
              <tr v-else-if="services.length === 0">
                <td colspan="4" class="text-center text-muted-foreground">
                  {{ t('application.detail.empty.services') }}
                </td>
              </tr>
              <tr v-for="svc in services" :key="svc.id">
                <td class="text-foreground">{{ environmentLabel(svc.environment_id) }}</td>
                <td class="text-muted-foreground">{{ svc.instance_key }}</td>
                <td>
                  <AppBadge
                    variant="pill"
                    :tone="appStatusTone(normalizeServiceStatus(svc.status))"
                  >
                    {{ t('status.' + normalizeServiceStatus(svc.status)) }}
                  </AppBadge>
                </td>
                <td class="text-muted-foreground">{{ versionLabelById(svc.version_id) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- 编辑应用 -->
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
      <template #footer>
        <button class="app-button" @click="isEditDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-primary" @click="handleEditOk">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <!-- 删除应用 -->
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

    <!-- 创建 / 编辑版本 -->
    <AppDialog
      v-model:open="isVersionDialogOpen"
      :title="
        editingVersionId
          ? t('application.detail.dialog.editVersion')
          : t('application.detail.dialog.createVersion')
      "
      width-class="w-[min(720px,calc(100vw-32px))]"
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
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="app-field-label">{{ t('application.detail.fields.envVars') }}</label>
            <button
              type="button"
              class="app-link text-sm"
              @click="versionForm.env.push({ key: '', value: '' })"
            >
              {{ t('application.detail.actions.addEnv') }}
            </button>
          </div>
          <div
            v-for="(envRow, envIndex) in versionForm.env"
            :key="'venv-' + envIndex"
            class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1.2fr_auto]"
          >
            <input
              v-model="envRow.key"
              type="text"
              class="app-input font-mono text-xs"
              placeholder="KEY"
            />
            <input
              v-model="envRow.value"
              type="text"
              class="app-input font-mono text-xs"
              :placeholder="t('application.detail.placeholders.envValue')"
            />
            <button
              type="button"
              class="app-link-danger"
              @click="versionForm.env.splice(envIndex, 1)"
            >
              {{ t('common.delete') }}
            </button>
          </div>
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="app-field-label">
              {{ t('application.detail.fields.components') }}
              <span class="text-destructive">*</span>
            </label>
            <button type="button" class="app-link text-sm" @click="addComponentRow">
              {{ t('application.detail.actions.addComponent') }}
            </button>
          </div>
          <p v-if="versionFormErrors.components" class="app-field-error text-xs">
            {{ versionFormErrors.components }}
          </p>
          <div
            v-for="(row, index) in versionForm.components"
            :key="index"
            class="space-y-3 rounded-md border border-border p-3"
          >
            <div class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1.4fr_auto]">
              <input
                v-model="row.name"
                type="text"
                class="app-input"
                :placeholder="t('application.detail.placeholders.componentName')"
              />
              <input
                v-model="row.image"
                type="text"
                class="app-input"
                :placeholder="t('application.detail.placeholders.componentImage')"
              />
              <button
                type="button"
                class="app-link-danger justify-self-start sm:justify-self-end"
                :disabled="versionForm.components.length <= 1"
                @click="removeComponentRow(index)"
              >
                {{ t('common.delete') }}
              </button>
            </div>
            <div class="space-y-2 border-t border-border pt-2">
              <div class="flex items-center justify-between">
                <span class="text-xs text-muted-foreground">
                  {{ t('application.detail.fields.ports') }}
                </span>
                <button
                  type="button"
                  class="app-link text-xs"
                  @click="row.ports.push({ host_port: 80, container_port: 80 })"
                >
                  {{ t('application.detail.actions.addPort') }}
                </button>
              </div>
              <div
                v-for="(portRow, portIndex) in row.ports"
                :key="'port-' + index + '-' + portIndex"
                class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1fr_auto]"
              >
                <input
                  v-model.number="portRow.host_port"
                  type="number"
                  min="1"
                  max="65535"
                  class="app-input font-mono text-xs"
                  :placeholder="t('application.detail.placeholders.hostPort')"
                />
                <input
                  v-model.number="portRow.container_port"
                  type="number"
                  min="1"
                  max="65535"
                  class="app-input font-mono text-xs"
                  :placeholder="t('application.detail.placeholders.containerPort')"
                />
                <button
                  type="button"
                  class="app-link-danger"
                  @click="row.ports.splice(portIndex, 1)"
                >
                  {{ t('common.delete') }}
                </button>
              </div>
            </div>
            <div class="space-y-2 border-t border-border pt-2">
              <div class="flex items-center justify-between">
                <span class="text-xs text-muted-foreground">
                  {{ t('application.detail.fields.componentEnv') }}
                </span>
                <button
                  type="button"
                  class="app-link text-xs"
                  @click="row.env.push({ key: '', value: '' })"
                >
                  {{ t('application.detail.actions.addEnv') }}
                </button>
              </div>
              <div
                v-for="(envRow, envIndex) in row.env"
                :key="'cenv-' + index + '-' + envIndex"
                class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1.2fr_auto]"
              >
                <input
                  v-model="envRow.key"
                  type="text"
                  class="app-input font-mono text-xs"
                  placeholder="KEY"
                />
                <input
                  v-model="envRow.value"
                  type="text"
                  class="app-input font-mono text-xs"
                  :placeholder="t('application.detail.placeholders.envValue')"
                />
                <button type="button" class="app-link-danger" @click="row.env.splice(envIndex, 1)">
                  {{ t('common.delete') }}
                </button>
              </div>
            </div>
            <div class="space-y-2 border-t border-border pt-2">
              <div class="flex items-center justify-between">
                <span class="text-xs text-muted-foreground">
                  {{ t('application.detail.fields.mounts') }}
                </span>
                <button
                  type="button"
                  class="app-link text-xs"
                  @click="
                    row.mounts.push({
                      source_type: 'logical',
                      source: '',
                      target: '',
                      read_only: false,
                      content: '',
                      content_mode: 'seed',
                    })
                  "
                >
                  {{ t('application.detail.actions.addMount') }}
                </button>
              </div>
              <div
                v-for="(mountRow, mountIndex) in row.mounts"
                :key="'mnt-' + index + '-' + mountIndex"
                class="space-y-2 rounded border border-dashed border-border p-2"
              >
                <div class="grid grid-cols-1 gap-2 sm:grid-cols-[0.9fr_1fr_1fr_auto_auto]">
                  <SelectControl
                    v-model="mountRow.source_type"
                    :options="mountSourceTypeOptions"
                    :placeholder="t('application.detail.placeholders.mountSourceType')"
                  />
                  <input
                    v-model="mountRow.source"
                    type="text"
                    class="app-input font-mono text-xs"
                    :placeholder="t('application.detail.placeholders.mountSource')"
                  />
                  <input
                    v-model="mountRow.target"
                    type="text"
                    class="app-input font-mono text-xs"
                    :placeholder="t('application.detail.placeholders.mountTarget')"
                  />
                  <label class="flex items-center gap-1 text-xs text-muted-foreground">
                    <input v-model="mountRow.read_only" type="checkbox" class="app-checkbox" />
                    ro
                  </label>
                  <button
                    type="button"
                    class="app-link-danger"
                    @click="row.mounts.splice(mountIndex, 1)"
                  >
                    {{ t('common.delete') }}
                  </button>
                </div>
                <div v-if="isFileMountRow(mountRow)" class="space-y-2 border-t border-border pt-2">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="text-xs text-muted-foreground">
                      {{ t('application.detail.fields.mountContent') }}
                    </span>
                    <SelectControl
                      v-model="mountRow.content_mode"
                      :options="mountContentModeOptions"
                      :placeholder="t('application.detail.placeholders.mountContentMode')"
                      class="w-36"
                    />
                  </div>
                  <textarea
                    v-model="mountRow.content"
                    rows="5"
                    class="app-input min-h-[6rem] w-full font-mono text-xs"
                    :placeholder="t('application.detail.placeholders.mountContent')"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="app-field-label">{{ t('application.detail.fields.exposes') }}</label>
            <button
              type="button"
              class="app-link text-sm"
              :disabled="versionComponentNameOptions.length === 0"
              @click="addExposeRow"
            >
              {{ t('application.detail.actions.addExpose') }}
            </button>
          </div>
          <p v-if="versionFormErrors.exposes" class="app-field-error text-xs">
            {{ versionFormErrors.exposes }}
          </p>
          <div
            v-for="(row, index) in versionForm.exposes"
            :key="'expose-' + index"
            class="grid grid-cols-1 gap-2 rounded-md border border-border p-3 sm:grid-cols-[1fr_0.8fr_0.7fr_auto]"
          >
            <SelectControl
              v-model="row.component_name"
              :options="exposeComponentOptions(row.component_name)"
              :placeholder="t('application.detail.placeholders.exposeComponent')"
              :disabled="versionComponentNameOptions.length === 0"
            />
            <SelectControl
              v-model="row.protocol"
              :options="exposeProtocolOptions"
              :placeholder="t('application.detail.placeholders.exposeProtocol')"
            />
            <input
              v-model.number="row.container_port"
              type="number"
              min="1"
              max="65535"
              class="app-input"
              :placeholder="t('application.detail.placeholders.port')"
            />
            <button
              type="button"
              class="app-link-danger justify-self-start sm:justify-self-end"
              @click="removeExposeRow(index)"
            >
              {{ t('common.delete') }}
            </button>
          </div>
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isVersionDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-primary" @click="handleVersionSave">
          {{ t('common.save') }}
        </button>
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

    <!-- Deploy -->
    <AppDialog
      v-model:open="isDeployDialogOpen"
      :title="t('application.detail.dialog.deploy')"
      width-class="w-[min(480px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.environment') }}
            <span class="text-destructive">*</span>
          </label>
          <SelectControl
            v-model="deployForm.environment_id"
            :options="environmentOptions"
            :placeholder="t('application.detail.placeholders.environment')"
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.instanceKey') }}
          </label>
          <input v-model="deployForm.instance_key" type="text" class="app-input" />
        </div>
        <label class="flex items-center gap-2">
          <input v-model="deployForm.force_recreate" type="checkbox" class="app-checkbox" />
          <span class="text-sm text-foreground">
            {{ t('application.detail.fields.forceRecreate') }}
          </span>
        </label>
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="app-field-label">
              {{ t('application.detail.fields.runtimeConfig') }}
            </label>
            <button
              type="button"
              class="app-link text-sm"
              @click="deployForm.runtime_config.push({ key: '', value: '' })"
            >
              {{ t('application.detail.actions.addRuntimeConfig') }}
            </button>
          </div>
          <div
            v-for="(row, index) in deployForm.runtime_config"
            :key="'rt-' + index"
            class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1.2fr_auto]"
          >
            <input
              v-model="row.key"
              type="text"
              class="app-input font-mono text-xs"
              placeholder="NAME"
            />
            <input
              v-model="row.value"
              type="text"
              class="app-input font-mono text-xs"
              placeholder="value"
            />
            <button
              type="button"
              class="app-link-danger"
              @click="deployForm.runtime_config.splice(index, 1)"
            >
              {{ t('common.delete') }}
            </button>
          </div>
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isDeployDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-primary" @click="confirmDeploy">
          {{ t('application.deploy') }}
        </button>
      </template>
    </AppDialog>

    <!-- Stop / Restart target -->
    <AppDialog
      v-model:open="isServiceTargetDialogOpen"
      :title="
        serviceTargetAction === 'restart'
          ? t('application.detail.dialog.restartTarget')
          : t('application.detail.dialog.stopTarget')
      "
      width-class="w-[min(480px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <p class="text-sm text-muted-foreground">
          {{ t('application.detail.hints.selectServiceTarget') }}
        </p>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.serviceInstance') }}
            <span class="text-destructive">*</span>
          </label>
          <SelectControl
            v-model="serviceTargetForm.service_id"
            :options="serviceTargetOptions"
            :placeholder="t('application.detail.placeholders.serviceInstance')"
          />
        </div>
        <label v-if="serviceTargetAction === 'stop'" class="flex items-center gap-2">
          <input v-model="serviceTargetForm.remove_volumes" type="checkbox" class="app-checkbox" />
          <span class="text-sm text-foreground">
            {{ t('application.detail.fields.removeVolumes') }}
          </span>
        </label>
      </div>
      <template #footer>
        <button class="app-button" @click="isServiceTargetDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button
          :disabled="operating"
          class="app-button-primary"
          @click="() => confirmServiceTarget()"
        >
          {{
            serviceTargetAction === 'restart'
              ? t('application.detail.actions.restart')
              : t('application.stop')
          }}
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
  import {
    ArrowLeft,
    ChevronDown,
    Download,
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
  import { environmentApi } from '@/api/cd/environment';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application';
  import type { EnvironmentResp } from '@/gen/proto/orbit/v1/environment';
  import type {
    ServiceResp,
    VersionComponentReq,
    VersionExposeReq,
    VersionResp,
  } from '@/gen/proto/orbit/v1/version';
  import { appStatusTone, normalizeServiceStatus, versionStatusTone } from '@/utils/status';
  import { delayAsync, formatTime } from '@/utils/time';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const applicationId = route.params.id as string;
  const toast = useToast();

  const { loading: basicInfoLoading, execute: executeBasicInfo } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { loading: versionListLoading, execute: executeVersionList } = useStatusAsync();
  const { loading: serviceListLoading, execute: executeServiceList } = useStatusAsync();
  const { loading: composePreviewLoading, execute: executeComposePreview } = useStatusAsync();

  const application = ref<ApplicationResp>();
  const versions = ref<VersionResp[]>([]);
  const services = ref<ServiceResp[]>([]);
  const environments = ref<EnvironmentResp[]>([]);
  const selectedVersionId = ref('');
  const pendingDeployVersionId = ref('');

  const isEditDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isVersionDialogOpen = ref(false);
  const isForkDialogOpen = ref(false);
  const isDeployDialogOpen = ref(false);
  const isServiceTargetDialogOpen = ref(false);
  const serviceTargetAction = ref<'stop' | 'restart'>('stop');
  const composePreviewDrawerOpen = ref(false);
  const editingVersionId = ref('');
  const forkingVersionId = ref('');
  const forkLabel = ref('');
  const forkLabelError = ref('');
  const deleteDir = ref(false);
  const composePreviewYaml = ref('');
  const composePreviewError = ref('');

  const editForm = reactive({
    name: '',
    code: '',
    image_pull_policy: 'missing',
  });
  const editErrors = reactive({ name: '' });
  const imagePullPolicyOptions = computed(() => [
    { value: 'missing', label: t('application.imagePullPolicyOptions.missing') },
    { value: 'always', label: t('application.imagePullPolicyOptions.always') },
    { value: 'never', label: t('application.imagePullPolicyOptions.never') },
  ]);

  type EnvFormRow = { key: string; value: string };
  type PortFormRow = { host_port: number; container_port: number };
  type MountFormRow = {
    source_type: string;
    source: string;
    target: string;
    read_only: boolean;
    content: string;
    content_mode: string;
  };
  const fileMountSuffixes = [
    '.json',
    '.yml',
    '.yaml',
    '.toml',
    '.pem',
    '.key',
    '.crt',
    '.conf',
    '.cfg',
    '.txt',
    '.env',
  ];
  type ComponentFormRow = {
    name: string;
    image: string;
    ports: PortFormRow[];
    env: EnvFormRow[];
    mounts: MountFormRow[];
  };
  type ExposeFormRow = { component_name: string; protocol: string; container_port: number };
  const versionForm = reactive({
    label: '',
    note: '',
    env: [] as EnvFormRow[],
    components: [emptyComponentRow()] as ComponentFormRow[],
    exposes: [] as ExposeFormRow[],
  });
  const versionFormErrors = reactive({ label: '', components: '', exposes: '' });
  const exposeProtocolOptions = [
    { value: 'http', label: 'http' },
    { value: 'tcp', label: 'tcp' },
  ];
  const mountSourceTypeOptions = [
    { value: 'logical', label: 'logical' },
    { value: 'volume', label: 'volume' },
    { value: 'special', label: 'special' },
  ];
  const mountContentModeOptions = [
    { value: 'seed', label: t('application.detail.contentMode.seed') },
    { value: 'sync', label: t('application.detail.contentMode.sync') },
  ];
  const versionComponentNameOptions = computed(() => {
    const names = versionForm.components.map((row) => row.name.trim()).filter(Boolean);
    return [...new Set(names)].map((name) => ({ value: name, label: name }));
  });

  const deployForm = reactive({
    environment_id: '',
    instance_key: 'default',
    force_recreate: false,
    runtime_config: [] as EnvFormRow[],
  });
  const serviceTargetForm = reactive({
    service_id: '',
    remove_volumes: false,
  });

  const environmentOptions = computed(() =>
    environments.value.map((item) => ({
      value: item.id,
      label: `${item.name} (${item.code})`,
    }))
  );

  const actionableServices = computed(() =>
    services.value.filter((svc) => {
      const status = normalizeServiceStatus(svc.status);
      return status === 'running' || status === 'faulted';
    })
  );

  const serviceTargetOptions = computed(() =>
    actionableServices.value.map((svc) => ({
      value: svc.id,
      label: `${environmentLabel(svc.environment_id)} / ${svc.instance_key} (${t(
        'status.' + normalizeServiceStatus(svc.status)
      )})`,
    }))
  );

  const serviceStatus = computed(() => normalizeServiceStatus(application.value?.service_status));
  const statusTone = computed(() => appStatusTone(serviceStatus.value));
  const statusText = computed(() => t('status.' + serviceStatus.value));
  const canStop = computed(() =>
    actionableServices.value.some((svc) => normalizeServiceStatus(svc.status) === 'running')
  );
  const canRestart = computed(() => actionableServices.value.length > 0);
  const canDelete = computed(
    () =>
      services.value.length === 0 ||
      services.value.every((svc) => {
        const status = normalizeServiceStatus(svc.status);
        return status === 'undeployed' || status === 'stopped' || status === 'faulted';
      })
  );
  const deployVersionId = computed(
    () => selectedVersionId.value || application.value?.version_id || ''
  );
  const currentVersionLabel = computed(() => {
    const boundId = application.value?.version_id;
    if (!boundId) {
      return '—';
    }
    const found = versions.value.find((item) => item.id === boundId);
    return found?.label || boundId;
  });

  function componentSummary(version: VersionResp) {
    if (!version.components?.length) {
      return '—';
    }
    return version.components.map((c) => `${c.name}:${c.image}`).join(', ');
  }

  function emptyComponentRow(): ComponentFormRow {
    return { name: '', image: '', ports: [], env: [], mounts: [] };
  }

  function parsePortsJson(raw?: string): PortFormRow[] {
    if (!raw?.trim()) {
      return [];
    }
    try {
      const parsed = JSON.parse(raw) as unknown;
      if (!Array.isArray(parsed)) {
        return [];
      }
      const rows: PortFormRow[] = [];
      for (const item of parsed) {
        if (typeof item === 'number') {
          const port = item;
          if (port >= 1 && port <= 65535) {
            rows.push({ host_port: port, container_port: port });
          }
          continue;
        }
        if (typeof item === 'string') {
          const text = item.trim();
          if (!text) {
            continue;
          }
          const parts = text.split(':');
          if (parts.length === 1) {
            const port = Number(parts[0]);
            if (port >= 1 && port <= 65535) {
              rows.push({ host_port: port, container_port: port });
            }
            continue;
          }
          // host:container or ip:host:container — take last two numeric segments
          const hostPort = Number(parts[parts.length - 2]);
          const containerPort = Number(parts[parts.length - 1]);
          if (hostPort >= 1 && hostPort <= 65535 && containerPort >= 1 && containerPort <= 65535) {
            rows.push({ host_port: hostPort, container_port: containerPort });
          }
          continue;
        }
        if (item && typeof item === 'object') {
          const row = item as {
            host_port?: number;
            published?: number;
            target?: number;
            container_port?: number;
          };
          const hostPort = Number(row.host_port ?? row.published ?? 0);
          const containerPort = Number(row.container_port ?? row.target ?? 0);
          if (hostPort >= 1 && hostPort <= 65535 && containerPort >= 1 && containerPort <= 65535) {
            rows.push({ host_port: hostPort, container_port: containerPort });
          }
        }
      }
      return rows;
    } catch {
      return [];
    }
  }

  function serializePortsRows(rows: PortFormRow[]): string | undefined {
    const items: string[] = [];
    for (const row of rows) {
      const hostPort = Number(row.host_port);
      const containerPort = Number(row.container_port);
      if (
        !Number.isInteger(hostPort) ||
        !Number.isInteger(containerPort) ||
        hostPort < 1 ||
        hostPort > 65535 ||
        containerPort < 1 ||
        containerPort > 65535
      ) {
        continue;
      }
      items.push(`${hostPort}:${containerPort}`);
    }
    if (items.length === 0) {
      return undefined;
    }
    return JSON.stringify(items);
  }

  function parseEnvJson(raw?: string): EnvFormRow[] {
    if (!raw?.trim()) {
      return [];
    }
    try {
      const parsed = JSON.parse(raw) as unknown;
      if (!Array.isArray(parsed)) {
        return [];
      }
      return parsed
        .map((item) => {
          const row = item as { key?: string; value?: string };
          return { key: String(row.key || ''), value: String(row.value ?? '') };
        })
        .filter((row) => row.key || row.value);
    } catch {
      return [];
    }
  }

  function isFileMountPath(source: string, target: string): boolean {
    for (const path of [source, target]) {
      const lower = path.toLowerCase();
      if (fileMountSuffixes.some((suffix) => lower.endsWith(suffix))) {
        return true;
      }
    }
    return false;
  }

  function isFileMountRow(row: MountFormRow): boolean {
    return row.source_type === 'logical' && isFileMountPath(row.source, row.target);
  }

  function parseMountsJson(raw?: string): MountFormRow[] {
    if (!raw?.trim()) {
      return [];
    }
    try {
      const parsed = JSON.parse(raw) as unknown;
      if (!Array.isArray(parsed)) {
        return [];
      }
      return parsed.map((item) => {
        const row = item as {
          source_type?: string;
          source?: string;
          target?: string;
          read_only?: boolean;
          content?: string;
          content_mode?: string;
        };
        return {
          source_type: String(row.source_type || 'logical'),
          source: String(row.source || ''),
          target: String(row.target || ''),
          read_only: Boolean(row.read_only),
          content: String(row.content ?? ''),
          content_mode: String(row.content_mode || 'seed') === 'sync' ? 'sync' : 'seed',
        };
      });
    } catch {
      return [];
    }
  }

  function serializeEnvRows(rows: EnvFormRow[]): string | undefined {
    const items = rows
      .map((row) => ({ key: row.key.trim(), value: row.value }))
      .filter((row) => row.key);
    if (items.length === 0) {
      return undefined;
    }
    return JSON.stringify(items);
  }

  function serializeMountRows(rows: MountFormRow[]): string | undefined {
    const items = rows
      .map((row) => {
        const item: {
          source_type: string;
          source: string;
          target: string;
          read_only: boolean;
          content?: string;
          content_mode?: string;
        } = {
          source_type: row.source_type.trim(),
          source: row.source.trim(),
          target: row.target.trim(),
          read_only: row.read_only,
        };
        if (
          item.source_type === 'logical' &&
          isFileMountPath(item.source, item.target) &&
          (row.content.trim() || row.content_mode === 'sync')
        ) {
          if (row.content) {
            item.content = row.content;
          }
          item.content_mode = row.content_mode === 'sync' ? 'sync' : 'seed';
        }
        return item;
      })
      .filter((row) => row.source_type && row.source && row.target);
    if (items.length === 0) {
      return undefined;
    }
    return JSON.stringify(items);
  }

  function runtimeConfigMap(): Record<string, string> | undefined {
    const out: Record<string, string> = {};
    for (const row of deployForm.runtime_config) {
      const key = row.key.trim();
      if (!key) {
        continue;
      }
      out[key] = row.value;
    }
    return Object.keys(out).length > 0 ? out : undefined;
  }

  function defaultExposeComponentName() {
    return versionComponentNameOptions.value[0]?.value || '';
  }

  function emptyExposeRow(): ExposeFormRow {
    return {
      component_name: defaultExposeComponentName(),
      protocol: 'http',
      container_port: 80,
    };
  }

  function exposeComponentOptions(currentName: string) {
    const options = [...versionComponentNameOptions.value];
    const current = currentName.trim();
    if (current && !options.some((item) => item.value === current)) {
      options.unshift({ value: current, label: current });
    }
    return options;
  }

  function componentToReq(row: ComponentFormRow): VersionComponentReq {
    return {
      name: row.name.trim(),
      image: row.image.trim(),
      ports_json: serializePortsRows(row.ports),
      env_json: serializeEnvRows(row.env),
      mounts_json: serializeMountRows(row.mounts),
    };
  }

  function exposeToReq(row: ExposeFormRow): VersionExposeReq {
    return {
      component_name: String(row.component_name || '').trim(),
      protocol: String(row.protocol || 'http').trim() || 'http',
      container_port: Number(row.container_port) || 0,
    };
  }

  function defaultEnvironmentId() {
    const local = environments.value.find((item) => item.code === 'local');
    return local?.id || environments.value[0]?.id || '';
  }

  function environmentLabel(environmentId: string) {
    const found = environments.value.find((item) => item.id === environmentId);
    if (!found) {
      return environmentId || '—';
    }
    return `${found.name} (${found.code})`;
  }

  function versionLabelById(versionId: string) {
    if (!versionId) {
      return '—';
    }
    const found = versions.value.find((item) => item.id === versionId);
    return found?.label || versionId;
  }

  async function fetchApplication() {
    try {
      await executeBasicInfo(async () => {
        const data = await applicationApi.get(applicationId);
        application.value = data;
        Object.assign(editForm, {
          name: data.name,
          code: data.code,
          image_pull_policy: data.image_pull_policy,
        });
        if (data.version_id && !selectedVersionId.value) {
          selectedVersionId.value = data.version_id;
        }
      });
      if (serviceStatus.value === 'deploying') {
        pollActiveDeployment();
      }
    } catch {
      toast.error(t('application.toast.loadDetailFailed'));
      router.push('/cd/applications');
    }
  }

  async function loadVersions() {
    try {
      await executeVersionList(async () => {
        const resp = await applicationApi.listVersions(applicationId);
        versions.value = resp.items ?? [];
        if (!selectedVersionId.value && versions.value.length > 0) {
          const bound = application.value?.version_id;
          selectedVersionId.value =
            (bound && versions.value.find((v) => v.id === bound)?.id) || versions.value[0].id;
        }
      });
    } catch {
      toast.error(t('application.toast.loadVersionsFailed'));
    }
  }

  async function loadServices() {
    try {
      await executeServiceList(async () => {
        const resp = await applicationApi.listServices(applicationId);
        services.value = resp.items ?? [];
      });
    } catch {
      toast.error(t('application.toast.loadServicesFailed'));
    }
  }

  async function loadEnvironments() {
    const projectId = application.value?.project_id;
    if (!projectId) {
      environments.value = [];
      return;
    }
    try {
      const resp = await environmentApi.list({ project_id: projectId, per_page: 100 });
      environments.value = resp.items ?? [];
      if (!deployForm.environment_id) {
        deployForm.environment_id = defaultEnvironmentId();
      }
    } catch {
      toast.error(t('application.toast.loadEnvironmentsFailed'));
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
            await fetchApplication();
            await loadServices();
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

  function openDeployDialog(versionId: string, forceRecreate = false) {
    if (!versionId) {
      toast.error(t('application.toast.deployVersionRequired'));
      return;
    }
    if (!environments.value.length) {
      toast.error(t('application.toast.environmentRequired'));
      return;
    }
    pendingDeployVersionId.value = versionId;
    deployForm.force_recreate = forceRecreate;
    deployForm.instance_key = deployForm.instance_key.trim() || 'default';
    deployForm.environment_id = deployForm.environment_id || defaultEnvironmentId();
    if (deployForm.runtime_config.length === 0) {
      deployForm.runtime_config = [];
    }
    isDeployDialogOpen.value = true;
  }

  function handleDeploy(forceRecreate = false) {
    openDeployDialog(deployVersionId.value, forceRecreate);
  }

  function handleDeployVersion(versionId: string, forceRecreate = false) {
    openDeployDialog(versionId, forceRecreate);
  }

  async function confirmDeploy() {
    const versionId = pendingDeployVersionId.value || deployVersionId.value;
    if (!versionId) {
      toast.error(t('application.toast.deployVersionRequired'));
      return;
    }
    if (!deployForm.environment_id) {
      toast.error(t('application.toast.environmentRequired'));
      return;
    }
    const instanceKey = deployForm.instance_key.trim() || 'default';
    try {
      await executeOp(async () => {
        const res = await applicationApi.deploy(applicationId, {
          version_id: versionId,
          environment_id: deployForm.environment_id,
          instance_key: instanceKey,
          force_recreate: deployForm.force_recreate,
          runtime_config: runtimeConfigMap() ?? {},
        });
        toast.success(t('application.toast.deployTriggeredDetail'));
        isDeployDialogOpen.value = false;
        router.push({
          path: `/cd/deployments/${res.deployment_id}`,
          query: { from: 'application' },
        });
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.deployFailed'));
    }
  }

  function openServiceTargetDialog(action: 'stop' | 'restart') {
    if (actionableServices.value.length === 0) {
      toast.error(t('application.toast.serviceTargetRequired'));
      return;
    }
    serviceTargetAction.value = action;
    const preferred = actionableServices.value[0];
    serviceTargetForm.service_id = preferred.id;
    serviceTargetForm.remove_volumes = false;
    if (actionableServices.value.length === 1) {
      void confirmServiceTarget(action, preferred.id, false);
      return;
    }
    isServiceTargetDialogOpen.value = true;
  }

  function handleStop() {
    openServiceTargetDialog('stop');
  }

  function handleRestart() {
    openServiceTargetDialog('restart');
  }

  async function confirmServiceTarget(
    action = serviceTargetAction.value,
    serviceId = serviceTargetForm.service_id,
    removeVolumes = serviceTargetForm.remove_volumes
  ) {
    if (!serviceId) {
      toast.error(t('application.toast.serviceTargetRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        if (action === 'stop') {
          const res = await applicationApi.stop(applicationId, {
            remove_volumes: removeVolumes,
            service_id: serviceId,
          });
          toast.success(t('application.toast.stopSubmitted'));
          isServiceTargetDialogOpen.value = false;
          const maxAttempts = 20;
          let attempts = 0;
          while (attempts < maxAttempts) {
            await delayAsync(3000);
            attempts++;
            try {
              const detail = await deploymentApi.get(res.deployment_id);
              if (['ran_to_completion', 'faulted', 'canceled'].includes(detail.status)) {
                await fetchApplication();
                await loadServices();
                break;
              }
            } catch {
              break;
            }
          }
          return;
        }
        const res = await applicationApi.restart(applicationId, { service_id: serviceId });
        toast.success(t('application.toast.restartSubmitted'));
        isServiceTargetDialogOpen.value = false;
        router.push({
          path: `/cd/deployments/${res.deployment_id}`,
          query: { from: 'application' },
        });
      });
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : action === 'stop'
            ? t('application.toast.stopFailed')
            : t('application.toast.restartFailed')
      );
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
        });
        toast.success(t('application.toast.updateSuccess'));
        isEditDialogOpen.value = false;
        await fetchApplication();
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

  function resetVersionForm() {
    versionForm.label = '';
    versionForm.note = '';
    versionForm.env = [];
    versionForm.components = [emptyComponentRow()];
    versionForm.exposes = [];
    Object.assign(versionFormErrors, { label: '', components: '', exposes: '' });
  }

  function openCreateVersionModal() {
    editingVersionId.value = '';
    resetVersionForm();
    versionForm.label = suggestNextLabel();
    isVersionDialogOpen.value = true;
  }

  function openEditVersionModal(version: VersionResp) {
    editingVersionId.value = version.id;
    versionForm.label = version.label;
    versionForm.note = version.note || '';
    versionForm.env = parseEnvJson(version.env_json);
    versionForm.components =
      version.components?.length > 0
        ? version.components.map((c) => ({
            name: c.name,
            image: c.image,
            ports: parsePortsJson(c.ports_json),
            env: parseEnvJson(c.env_json),
            mounts: parseMountsJson(c.mounts_json),
          }))
        : [emptyComponentRow()];
    versionForm.exposes =
      version.exposes?.map((item) => ({
        component_name: item.component_name,
        protocol: item.protocol || 'http',
        container_port: item.container_port,
      })) ?? [];
    Object.assign(versionFormErrors, { label: '', components: '', exposes: '' });
    isVersionDialogOpen.value = true;
  }

  function suggestNextLabel() {
    const labels = versions.value.map((v) => v.label);
    let n = versions.value.length + 1;
    while (labels.includes(`v${n}`)) {
      n += 1;
    }
    return `v${n}`;
  }

  function addComponentRow() {
    versionForm.components.push(emptyComponentRow());
  }

  function removeComponentRow(index: number) {
    if (versionForm.components.length <= 1) {
      return;
    }
    versionForm.components.splice(index, 1);
    syncExposeComponentNames();
  }

  function addExposeRow() {
    if (versionComponentNameOptions.value.length === 0) {
      versionFormErrors.exposes = t('application.validation.exposeNeedsComponent');
      return;
    }
    versionForm.exposes.push(emptyExposeRow());
    versionFormErrors.exposes = '';
  }

  function removeExposeRow(index: number) {
    versionForm.exposes.splice(index, 1);
    if (versionForm.exposes.length === 0) {
      versionFormErrors.exposes = '';
    }
  }

  function syncExposeComponentNames() {
    const names = new Set(versionComponentNameOptions.value.map((item) => item.value));
    const fallback = defaultExposeComponentName();
    for (const row of versionForm.exposes) {
      if (!names.has(String(row.component_name || '').trim())) {
        row.component_name = fallback;
      }
    }
  }

  function validateVersionForm() {
    versionFormErrors.label = versionForm.label.trim()
      ? ''
      : t('application.validation.versionLabelRequired');
    const rows = versionForm.components.map(componentToReq).filter((c) => c.name || c.image);
    if (rows.length === 0) {
      versionFormErrors.components = t('application.validation.componentRequired');
    } else if (rows.some((c) => !c.name || !c.image)) {
      versionFormErrors.components = t('application.validation.componentNameImageRequired');
    } else {
      versionFormErrors.components = '';
    }

    const componentNames = new Set(rows.filter((c) => c.name && c.image).map((c) => c.name));
    const exposeRows = versionForm.exposes.map(exposeToReq);
    if (exposeRows.some((item) => !item.component_name)) {
      versionFormErrors.exposes = t('application.validation.exposeComponentRequired');
    } else if (exposeRows.some((item) => !componentNames.has(item.component_name))) {
      versionFormErrors.exposes = t('application.validation.exposeComponentNotFound');
    } else if (exposeRows.some((item) => item.container_port < 1 || item.container_port > 65535)) {
      versionFormErrors.exposes = t('application.validation.portRange');
    } else {
      versionFormErrors.exposes = '';
    }

    if (!versionFormErrors.components) {
      for (const row of versionForm.components) {
        for (const port of row.ports) {
          const hostPort = Number(port.host_port);
          const containerPort = Number(port.container_port);
          if (
            !Number.isInteger(hostPort) ||
            !Number.isInteger(containerPort) ||
            hostPort < 1 ||
            hostPort > 65535 ||
            containerPort < 1 ||
            containerPort > 65535
          ) {
            versionFormErrors.components = t('application.validation.portRange');
            break;
          }
        }
        if (versionFormErrors.components) {
          break;
        }
      }
    }

    return !versionFormErrors.label && !versionFormErrors.components && !versionFormErrors.exposes;
  }

  async function handleVersionSave() {
    if (!validateVersionForm()) {
      return;
    }
    const components = versionForm.components.map(componentToReq).filter((c) => c.name && c.image);
    const exposes = versionForm.exposes
      .map(exposeToReq)
      .filter((item) => item.component_name && item.container_port > 0);
    const envJson = serializeEnvRows(versionForm.env);
    const note = versionForm.note.trim() || undefined;
    try {
      await executeOp(async () => {
        if (editingVersionId.value) {
          await applicationApi.updateVersion(editingVersionId.value, {
            label: versionForm.label.trim(),
            env_json: envJson,
            note,
            components,
            exposes,
          });
          toast.success(t('application.toast.updateSuccess'));
        } else {
          const created = await applicationApi.createVersion(applicationId, {
            application_id: applicationId,
            label: versionForm.label.trim(),
            env_json: envJson,
            note,
            components,
            exposes,
          });
          selectedVersionId.value = created.id;
          toast.success(t('application.toast.createVersionSuccess'));
        }
        isVersionDialogOpen.value = false;
        await loadVersions();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.saveFailed'));
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

  function openForkModal(version: VersionResp) {
    forkingVersionId.value = version.id;
    forkLabel.value = `${version.label}-fork`;
    forkLabelError.value = '';
    isForkDialogOpen.value = true;
  }

  async function handleForkOk() {
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
        selectedVersionId.value = created.id;
        toast.success(t('application.toast.forkSuccess'));
        isForkDialogOpen.value = false;
        await loadVersions();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.forkFailed'));
    }
  }

  async function openPreview(versionId: string) {
    const environmentId = deployForm.environment_id || defaultEnvironmentId();
    if (!environmentId) {
      toast.error(t('application.toast.environmentRequired'));
      return;
    }
    composePreviewYaml.value = '';
    composePreviewError.value = '';
    composePreviewDrawerOpen.value = true;
    try {
      await executeComposePreview(async () => {
        const { compose_yaml } = await applicationApi.previewVersion(versionId, {
          environment_id: environmentId,
          instance_key: 'default',
        });
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

  onMounted(async () => {
    await fetchApplication();
    await Promise.all([loadVersions(), loadServices(), loadEnvironments()]);
  });
</script>
