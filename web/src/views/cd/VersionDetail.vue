<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="flex flex-wrap items-center gap-2 text-xl font-semibold text-foreground">
        {{ version?.label || t('application.versionDetail.title') }}
        <AppBadge v-if="version" variant="pill" :tone="versionStatusTone(version.status)">
          {{ t('status.' + version.status) }}
        </AppBadge>
      </h1>
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
                {{ t('status.' + version.status) }}
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
              <div v-else class="space-y-1 font-mono text-xs text-foreground">
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
          <table class="app-table-detail min-w-[720px]">
            <thead>
              <tr>
                <th>{{ t('application.detail.fields.component') }}</th>
                <th>{{ t('application.detail.placeholders.componentImage') }}</th>
                <th>{{ t('application.detail.fields.ports') }}</th>
                <th v-if="isEditable">{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="(version.components ?? []).length === 0">
                <td :colspan="isEditable ? 4 : 3" class="text-center text-muted-foreground">
                  {{ t('application.versionDetail.empty.components') }}
                </td>
              </tr>
              <tr v-for="(comp, index) in version.components" :key="comp.id || index">
                <td class="text-foreground">{{ comp.name }}</td>
                <td
                  class="max-w-xs truncate font-mono text-xs text-muted-foreground"
                  :title="comp.image"
                >
                  {{ comp.image }}
                </td>
                <td class="text-muted-foreground">{{ portsSummary(comp.ports_json) }}</td>
                <td v-if="isEditable">
                  <div class="flex items-center gap-3">
                    <button class="app-link" @click="openComponentModal(index)">
                      {{ t('common.edit') }}
                    </button>
                    <button
                      class="app-link-danger"
                      :disabled="operating"
                      @click="removeComponent(index)"
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
                <td class="text-muted-foreground">{{ row.protocol }}</td>
                <td class="text-muted-foreground">{{ row.access || 'public' }}</td>
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

    <!-- 组件表单 -->
    <AppDialog
      v-model:open="isComponentDialogOpen"
      :title="
        editingComponentIndex === null
          ? t('application.versionDetail.dialog.addComponent')
          : t('application.versionDetail.dialog.editComponent')
      "
      width-class="w-[min(720px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.detail.fields.component') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="componentForm.name"
              type="text"
              class="app-input"
              :placeholder="t('application.detail.placeholders.componentName')"
            />
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.detail.placeholders.componentImage') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="componentForm.image"
              type="text"
              class="app-input"
              :placeholder="t('application.detail.placeholders.componentImage')"
            />
          </div>
        </div>
        <p v-if="componentFormError" class="app-field-error text-xs">{{ componentFormError }}</p>

        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="app-field-label">{{ t('application.detail.fields.ports') }}</label>
            <button
              type="button"
              class="app-link text-sm"
              @click="componentForm.ports.push({ host_port: 80, container_port: 80 })"
            >
              {{ t('application.detail.actions.addPort') }}
            </button>
          </div>
          <div
            v-for="(portRow, portIndex) in componentForm.ports"
            :key="'port-' + portIndex"
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
              @click="componentForm.ports.splice(portIndex, 1)"
            >
              {{ t('common.delete') }}
            </button>
          </div>
        </div>

        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="app-field-label">{{ t('application.detail.fields.componentEnv') }}</label>
            <button
              type="button"
              class="app-link text-sm"
              @click="componentForm.env.push({ key: '', value: '' })"
            >
              {{ t('application.detail.actions.addEnv') }}
            </button>
          </div>
          <div
            v-for="(envRow, envIndex) in componentForm.env"
            :key="'cenv-' + envIndex"
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
              @click="componentForm.env.splice(envIndex, 1)"
            >
              {{ t('common.delete') }}
            </button>
          </div>
        </div>

        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="app-field-label">{{ t('application.detail.fields.mounts') }}</label>
            <button type="button" class="app-link text-sm" @click="addMountRow">
              {{ t('application.detail.actions.addMount') }}
            </button>
          </div>
          <div
            v-for="(mountRow, mountIndex) in componentForm.mounts"
            :key="'mnt-' + mountIndex"
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
                @click="componentForm.mounts.splice(mountIndex, 1)"
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
      <template #footer>
        <button class="app-button" @click="isComponentDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button :disabled="operating" class="app-button-primary" @click="saveComponent">
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
          <SelectControl
            v-model="exposeForm.component_name"
            :options="componentNameOptions"
            :placeholder="t('application.detail.placeholders.exposeComponent')"
          />
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.detail.placeholders.exposeProtocol') }}
            </label>
            <SelectControl
              v-model="exposeForm.protocol"
              :options="exposeProtocolOptions"
              :placeholder="t('application.detail.placeholders.exposeProtocol')"
            />
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.detail.placeholders.exposeAccess') }}
            </label>
            <SelectControl
              v-model="exposeForm.access"
              :options="exposeAccessOptions"
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
            {{ t('application.detail.fields.environment') }}
            <span class="text-destructive">*</span>
          </label>
          <SelectControl
            v-model="deployForm.environment_id"
            :options="environmentSelectOptions"
            :placeholder="t('application.detail.fields.environment')"
          />
          <p v-if="deployError" class="app-field-error mt-1 text-xs">{{ deployError }}</p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.instanceKey') }}
          </label>
          <input
            v-model="deployForm.instance_key"
            type="text"
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
  import { ArrowLeft, Pencil, Plus, Rocket, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/cd/application';
  import { environmentApi } from '@/api/cd/environment';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { EnvironmentResp } from '@/gen/proto/orbit/v1/environment';
  import type {
    VersionComponentReq,
    VersionExposeReq,
    VersionResp,
  } from '@/gen/proto/orbit/v1/version';
  import { versionStatusTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';

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

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const versionId = route.params.id as string;

  const { loading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { loading: composePreviewLoading, execute: executeComposePreview } = useStatusAsync();

  const version = ref<VersionResp>();
  const environments = ref<EnvironmentResp[]>([]);

  const isBasicDialogOpen = ref(false);
  const isComponentDialogOpen = ref(false);
  const isExposeDialogOpen = ref(false);
  const isForkDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isDeployDialogOpen = ref(false);
  const composePreviewDrawerOpen = ref(false);
  const composePreviewYaml = ref('');
  const composePreviewError = ref('');
  const editingComponentIndex = ref<number | null>(null);
  const editingExposeIndex = ref<number | null>(null);
  const forkLabel = ref('');
  const forkLabelError = ref('');
  const basicFormError = ref('');
  const componentFormError = ref('');
  const exposeFormError = ref('');
  const deployError = ref('');
  const deployForm = reactive({
    environment_id: '',
    instance_key: 'default',
    force_recreate: false,
  });

  const basicForm = reactive({
    label: '',
    note: '',
    env: [] as EnvFormRow[],
  });
  const componentForm = reactive({
    name: '',
    image: '',
    ports: [] as PortFormRow[],
    env: [] as EnvFormRow[],
    mounts: [] as MountFormRow[],
  });
  const exposeForm = reactive({
    component_name: '',
    protocol: 'http',
    container_port: 80,
    access: 'public',
    listen_port: 0,
  });

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
  const exposeProtocolOptions = [
    { value: 'http', label: 'http' },
    { value: 'tcp', label: 'tcp' },
  ];
  const exposeAccessOptions = [
    { value: 'public', label: 'public' },
    { value: 'local', label: 'local' },
  ];
  const mountSourceTypeOptions = [
    { value: 'logical', label: 'logical' },
    { value: 'volume', label: 'volume' },
    { value: 'special', label: 'special' },
  ];
  const mountContentModeOptions = computed(() => [
    { value: 'seed', label: t('application.detail.contentMode.seed') },
    { value: 'sync', label: t('application.detail.contentMode.sync') },
  ]);

  const isEditable = computed(() => version.value?.status === 'unpublished');
  const isDeployable = computed(() => Boolean(version.value));
  const environmentSelectOptions = computed(() =>
    environments.value.map((env) => ({
      value: env.id,
      label: env.name ? `${env.name} (${env.code})` : env.code,
    }))
  );
  const envRows = computed(() => parseEnvJson(version.value?.env_json));
  const componentNameOptions = computed(() =>
    (version.value?.components ?? []).map((c) => ({ value: c.name, label: c.name }))
  );

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

  function serializeEnvRows(rows: EnvFormRow[]): string | undefined {
    const items = rows
      .map((row) => ({ key: row.key.trim(), value: row.value }))
      .filter((row) => row.key);
    if (items.length === 0) {
      return undefined;
    }
    return JSON.stringify(items);
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
          if (item >= 1 && item <= 65535) {
            rows.push({ host_port: item, container_port: item });
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

  function portsSummary(raw?: string) {
    const rows = parsePortsJson(raw);
    if (rows.length === 0) {
      return '—';
    }
    return rows.map((r) => `${r.host_port}:${r.container_port}`).join(', ');
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

  function componentToReq(row: {
    name: string;
    image: string;
    ports: PortFormRow[];
    env: EnvFormRow[];
    mounts: MountFormRow[];
  }): VersionComponentReq {
    return {
      name: row.name.trim(),
      image: row.image.trim(),
      ports_json: serializePortsRows(row.ports),
      env_json: serializeEnvRows(row.env),
      mounts_json: serializeMountRows(row.mounts),
    };
  }

  function componentsPayload(): VersionComponentReq[] {
    return (version.value?.components ?? []).map((c) => ({
      name: c.name,
      image: c.image,
      command_json: c.command_json,
      args_json: c.args_json,
      env_json: c.env_json,
      ports_json: c.ports_json,
      mounts_json: c.mounts_json,
      networks_json: c.networks_json,
      depends_on_json: c.depends_on_json,
      healthcheck_json: c.healthcheck_json,
      resources_json: c.resources_json,
      pull_policy: c.pull_policy,
    }));
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
      router.push('/cd/versions');
    }
  }

  async function loadEnvironments() {
    const appId = version.value?.application_id;
    if (!appId) {
      environments.value = [];
      return;
    }
    try {
      const app = await applicationApi.get(appId);
      if (!app.project_id) {
        environments.value = [];
        return;
      }
      const resp = await environmentApi.list({ project_id: app.project_id, per_page: 100 });
      environments.value = resp.items ?? [];
    } catch {
      toast.error(t('application.toast.loadEnvironmentsFailed'));
    }
  }

  function defaultEnvironmentId() {
    const local = environments.value.find((item) => item.code === 'local');
    return local?.id || environments.value[0]?.id || '';
  }

  function goBack() {
    const appId = version.value?.application_id;
    if (appId) {
      router.push({ path: '/cd/versions', query: { application_id: appId } });
      return;
    }
    router.push('/cd/versions');
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

  function addMountRow() {
    componentForm.mounts.push({
      source_type: 'logical',
      source: '',
      target: '',
      read_only: false,
      content: '',
      content_mode: 'seed',
    });
  }

  function openComponentModal(index?: number) {
    editingComponentIndex.value = index ?? null;
    componentFormError.value = '';
    if (index === undefined || !version.value?.components?.[index]) {
      componentForm.name = '';
      componentForm.image = '';
      componentForm.ports = [];
      componentForm.env = [];
      componentForm.mounts = [];
    } else {
      const c = version.value.components[index];
      componentForm.name = c.name;
      componentForm.image = c.image;
      componentForm.ports = parsePortsJson(c.ports_json);
      componentForm.env = parseEnvJson(c.env_json);
      componentForm.mounts = parseMountsJson(c.mounts_json);
    }
    isComponentDialogOpen.value = true;
  }

  async function saveComponent() {
    const name = componentForm.name.trim();
    const image = componentForm.image.trim();
    if (!name || !image) {
      componentFormError.value = t('application.validation.componentNameImageRequired');
      return;
    }
    for (const port of componentForm.ports) {
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
        componentFormError.value = t('application.validation.portRange');
        return;
      }
    }
    const next = componentsPayload();
    const req = componentToReq(componentForm);
    if (editingComponentIndex.value === null) {
      if (next.some((c) => c.name === req.name)) {
        componentFormError.value = t('application.versionDetail.validation.duplicateComponent');
        return;
      }
      next.push(req);
    } else {
      const oldName = version.value?.components?.[editingComponentIndex.value]?.name;
      if (next.some((c, i) => i !== editingComponentIndex.value && c.name === req.name)) {
        componentFormError.value = t('application.versionDetail.validation.duplicateComponent');
        return;
      }
      next[editingComponentIndex.value] = req;
      // Keep exposes in sync when renaming a component.
      let exposes = exposesPayload();
      if (oldName && oldName !== req.name) {
        exposes = exposes.map((item) =>
          item.component_name === oldName ? { ...item, component_name: req.name } : item
        );
      }
      try {
        await executeOp(async () => {
          version.value = await applicationApi.updateVersion(versionId, {
            components: next,
            exposes,
          });
          toast.success(t('application.toast.updateSuccess'));
          isComponentDialogOpen.value = false;
        });
      } catch (error) {
        toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
      }
      return;
    }
    try {
      await executeOp(async () => {
        version.value = await applicationApi.updateVersion(versionId, {
          components: next,
          exposes: exposesPayload(),
        });
        toast.success(t('application.toast.updateSuccess'));
        isComponentDialogOpen.value = false;
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  async function removeComponent(index: number) {
    const name = version.value?.components?.[index]?.name;
    if (!name) {
      return;
    }
    if ((version.value?.exposes ?? []).some((item) => item.component_name === name)) {
      toast.error(t('application.versionDetail.validation.componentHasExposes'));
      return;
    }
    const next = componentsPayload().filter((_, i) => i !== index);
    try {
      await executeOp(async () => {
        version.value = await applicationApi.updateVersion(versionId, {
          components: next,
          exposes: exposesPayload(),
        });
        toast.success(t('application.toast.updateSuccess'));
      });
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
      exposeForm.access = 'public';
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

  function openForkModal() {
    forkLabel.value = `${version.value?.label || 'v'}-fork`;
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
        router.push(`/cd/version/${created.id}`);
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
    if (environments.value.length === 0) {
      await loadEnvironments();
    }
    const environmentId = defaultEnvironmentId();
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

  async function openDeployModal() {
    if (!version.value || !isDeployable.value) {
      return;
    }
    deployError.value = '';
    deployForm.force_recreate = false;
    deployForm.instance_key = 'default';
    if (environments.value.length === 0) {
      await loadEnvironments();
    }
    deployForm.environment_id = defaultEnvironmentId();
    if (!deployForm.environment_id) {
      toast.error(t('application.toast.environmentRequired'));
      return;
    }
    isDeployDialogOpen.value = true;
  }

  async function handleDeployOk() {
    const current = version.value;
    if (!current) {
      return;
    }
    if (!deployForm.environment_id) {
      deployError.value = t('application.toast.environmentRequired');
      return;
    }
    deployError.value = '';
    try {
      await executeOp(async () => {
        const result = await applicationApi.deploy(current.application_id, {
          version_id: current.id,
          environment_id: deployForm.environment_id,
          instance_key: deployForm.instance_key.trim() || 'default',
          force_recreate: deployForm.force_recreate,
          runtime_config: {},
        });
        toast.success(t('application.toast.deployTriggeredDetail'));
        isDeployDialogOpen.value = false;
        if (result.deployment_id) {
          router.push(`/cd/deployment/${result.deployment_id}`);
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
