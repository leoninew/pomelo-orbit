<template>
  <div class="flex flex-col gap-4 text-sm">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <DetailPageHeader
        :items="breadcrumbs"
        :title="isNew ? t('application.componentDetail.newTitle') : form.name"
      />
      <div class="flex flex-wrap items-center gap-2">
        <template v-if="canEdit && isNew">
          <button class="app-button h-9 px-3" :disabled="operating" @click="goBack">
            <X class="size-4" />
            {{ t('application.componentDetail.actions.cancel') }}
          </button>
          <button class="app-button-primary h-9 px-3" :disabled="operating" @click="save()">
            <Save class="size-4" />
            {{ t('application.componentDetail.actions.save') }}
          </button>
        </template>
        <button
          v-if="canEdit && !isNew"
          class="app-button-danger h-9 px-3"
          :disabled="operating"
          @click="deleteDialogOpen = true"
        >
          <Trash2 class="size-4" />
          {{ t('application.componentDetail.actions.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="goBack">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </header>

    <AppLoadingState v-if="loading" size="section" />

    <template v-else-if="version">
      <AppEmptyState v-if="isNew && !canEdit" :message="t('application.componentDetail.empty')" />

      <TabsRoot v-else v-model="activeTab" class="flex min-w-0 flex-col gap-4">
        <div class="overflow-x-auto border-b border-border">
          <TabsList
            :aria-label="t('application.componentDetail.title')"
            class="flex min-w-max gap-1"
          >
            <TabsTrigger
              v-for="tab in tabs"
              :key="tab.id"
              :value="tab.id"
              class="border-b-2 border-transparent px-3 py-2.5 text-sm text-muted-foreground hover:text-foreground data-[state=active]:border-primary data-[state=active]:text-foreground"
            >
              {{ tab.label }}
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="runtime" class="flex flex-col gap-4 outline-none">
          <DetailInfoCard
            :title="t('application.componentDetail.sections.basic')"
            :editable="Boolean(canEdit && !isNew)"
            :disabled="operating"
            @edit="startBasicEditing"
          >
            <div v-if="isNew" class="grid grid-cols-1 gap-4 p-5 sm:grid-cols-2 sm:p-6">
              <div>
                <label class="app-field-label mb-1.5 block">
                  {{ t('application.detail.fields.component') }}
                  <span class="text-destructive">*</span>
                </label>
                <input
                  v-model="form.name"
                  class="app-input"
                  :class="basicErrors.name ? 'app-input-error' : ''"
                  type="text"
                  :aria-invalid="basicErrors.name ? 'true' : undefined"
                  @input="basicErrors.name = ''"
                />
                <p v-if="basicErrors.name" class="app-field-error" role="alert">
                  {{ basicErrors.name }}
                </p>
              </div>
              <div class="sm:col-span-2">
                <label class="app-field-label mb-1.5 block">
                  {{ t('application.detail.fields.image') }}
                  <span class="text-destructive">*</span>
                </label>
                <input
                  v-model="form.image"
                  class="app-input"
                  :class="basicErrors.image ? 'app-input-error' : ''"
                  type="text"
                  :aria-invalid="basicErrors.image ? 'true' : undefined"
                  @input="basicErrors.image = ''"
                />
                <p v-if="basicErrors.image" class="app-field-error" role="alert">
                  {{ basicErrors.image }}
                </p>
              </div>
              <div>
                <label class="app-field-label mb-1.5 block">
                  {{ t('application.componentDetail.fields.pullPolicy') }}
                  <span class="text-destructive">*</span>
                </label>
                <RawValueSelect v-model="form.pull_policy" :values="pullPolicyValues" />
                <p v-if="basicErrors.pullPolicy" class="app-field-error" role="alert">
                  {{ basicErrors.pullPolicy }}
                </p>
              </div>
              <div>
                <label class="app-field-label mb-1.5 block">
                  {{ t('application.componentDetail.fields.restartPolicy') }}
                </label>
                <RawValueSelect
                  v-model="form.restart_policy"
                  :placeholder="t('common.notSet')"
                  :values="restartPolicyValues"
                />
              </div>
              <div class="sm:col-span-2">
                <label class="app-field-label mb-1.5 block">
                  {{ t('application.componentDetail.fields.containerEntrypoint') }}
                </label>
                <textarea v-model="form.entrypoint" class="app-textarea" rows="3" />
              </div>
              <div class="sm:col-span-2">
                <label class="app-field-label mb-1.5 block">
                  {{ t('application.componentDetail.fields.command') }}
                </label>
                <textarea v-model="form.command" class="app-textarea" rows="3" />
              </div>
            </div>

            <dl
              v-else
              class="app-detail-fields grid grid-cols-1 gap-x-8 gap-y-5 p-5 sm:grid-cols-2 sm:p-6"
            >
              <div class="min-w-0">
                <dt class="text-muted-foreground">
                  {{ t('application.detail.fields.component') }}
                </dt>
                <dd class="mt-1 break-words text-foreground">{{ form.name }}</dd>
              </div>
              <div class="min-w-0 sm:col-span-2">
                <dt class="text-muted-foreground">{{ t('application.detail.fields.image') }}</dt>
                <dd class="mt-1 break-all text-foreground">{{ form.image }}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.pullPolicy') }}
                </dt>
                <dd class="mt-1 text-foreground">{{ form.pull_policy }}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.restartPolicy') }}
                </dt>
                <dd class="mt-1 text-foreground">
                  {{ form.restart_policy || t('common.notSet') }}
                </dd>
              </div>
              <div class="min-w-0 sm:col-span-2">
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.containerEntrypoint') }}
                </dt>
                <dd v-if="!form.entrypoint" class="mt-1 text-foreground">
                  {{ t('common.notSet') }}
                </dd>
                <dd v-else class="mt-1 whitespace-pre-wrap break-words text-foreground">
                  {{ form.entrypoint }}
                </dd>
              </div>
              <div class="min-w-0 sm:col-span-2">
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.command') }}
                </dt>
                <dd v-if="!form.command" class="mt-1 text-foreground">
                  {{ t('common.notSet') }}
                </dd>
                <dd v-else class="mt-1 whitespace-pre-wrap break-words text-foreground">
                  {{ form.command }}
                </dd>
              </div>
            </dl>
          </DetailInfoCard>
          <DetailInfoCard
            v-if="!isNew"
            :title="t('application.componentDetail.sections.healthcheck')"
            :editable="canEdit"
            :disabled="operating"
            @edit="startHealthcheckEditing"
          >
            <p v-if="!form.healthcheck_enabled" class="p-5 text-sm text-muted-foreground sm:p-6">
              {{ t('common.notSet') }}
            </p>
            <dl
              v-else
              class="app-detail-fields grid grid-cols-1 gap-x-8 gap-y-4 p-5 sm:grid-cols-2 lg:grid-cols-3 sm:p-6"
            >
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.disabled') }}
                </dt>
                <dd class="mt-1 text-foreground">
                  {{ form.healthcheck_disabled ? t('common.yes') : t('common.no') }}
                </dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.testMode') }}
                </dt>
                <dd class="mt-1 text-foreground">
                  {{ form.healthcheck_test_mode || t('common.notSet') }}
                </dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.interval') }}
                </dt>
                <dd class="mt-1 text-foreground">
                  {{ form.healthcheck_interval || t('common.notSet') }}
                </dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.timeout') }}
                </dt>
                <dd class="mt-1 text-foreground">
                  {{ form.healthcheck_timeout || t('common.notSet') }}
                </dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.retries') }}
                </dt>
                <dd class="mt-1 text-foreground">
                  {{ form.healthcheck_retries || t('common.notSet') }}
                </dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.startPeriod') }}
                </dt>
                <dd class="mt-1 text-foreground">
                  {{ form.healthcheck_start_period || t('common.notSet') }}
                </dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.startInterval') }}
                </dt>
                <dd class="mt-1 text-foreground">
                  {{ form.healthcheck_start_interval || t('common.notSet') }}
                </dd>
              </div>
              <div class="sm:col-span-2 lg:col-span-3">
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.test') }}
                </dt>
                <dd v-if="!form.healthcheck_test" class="mt-1 text-foreground">
                  {{ t('common.notSet') }}
                </dd>
                <dd v-else class="mt-1 break-all text-foreground">
                  {{ form.healthcheck_test }}
                </dd>
              </div>
            </dl>
          </DetailInfoCard>

          <DetailInfoCard v-if="!isNew" :title="t('application.componentDetail.fields.dependency')">
            <template #actions>
              <button
                v-if="canEdit"
                class="app-button-primary h-9 px-3"
                :disabled="operating"
                @click="openRecordDialog('dependencies')"
              >
                <Plus class="size-4" />
                {{ t('common.add') }}
              </button>
            </template>
            <AppEmptyState v-if="form.dependencies.length === 0" size="compact" />
            <div v-else class="overflow-x-auto">
              <table class="app-data-table min-w-[560px]">
                <thead>
                  <tr>
                    <th>{{ t('application.componentDetail.fields.dependency') }}</th>
                    <th>{{ t('application.componentDetail.fields.condition') }}</th>
                    <th class="w-32">{{ t('common.operation') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, index) in form.dependencies" :key="`dependency-${index}`">
                    <td class="min-w-56 text-foreground">{{ row.name }}</td>
                    <td class="min-w-56 text-foreground">{{ row.condition }}</td>
                    <td class="w-32">
                      <div v-if="canEdit" class="flex items-center gap-3">
                        <button
                          class="app-link"
                          :disabled="operating"
                          @click="openRecordDialog('dependencies', index)"
                        >
                          {{ t('common.edit') }}
                        </button>
                        <button
                          class="app-link-danger"
                          :disabled="operating"
                          @click="deleteRecord('dependencies', index)"
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
        </TabsContent>

        <TabsContent value="connectivity" class="flex flex-col gap-4 outline-none">
          <DetailInfoCard :title="t('application.componentDetail.sections.ports')">
            <template #actions>
              <button
                v-if="canEdit"
                class="app-button-primary h-9 px-3"
                :disabled="operating"
                @click="openRecordDialog('ports')"
              >
                <Plus class="size-4" />
                {{ t('common.add') }}
              </button>
            </template>

            <AppEmptyState v-if="form.ports.length === 0" size="compact" />
            <div v-else class="overflow-x-auto">
              <table class="app-data-table min-w-[760px]">
                <thead>
                  <tr>
                    <th>{{ t('application.componentDetail.fields.protocol') }}</th>
                    <th>{{ t('application.componentDetail.fields.endpointMode') }}</th>
                    <th>{{ t('application.componentDetail.fields.hostPort') }}</th>
                    <th>{{ t('application.componentDetail.fields.containerPort') }}</th>
                    <th class="w-32">{{ t('common.operation') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, index) in form.ports" :key="`port-${index}`">
                    <td class="text-foreground">{{ row.protocol || 'tcp' }}</td>
                    <td class="text-foreground">{{ row.mode || 'host' }}</td>
                    <td class="text-foreground">{{ row.host_port || '-' }}</td>
                    <td class="text-foreground">{{ row.container_port }}</td>
                    <td class="w-32">
                      <div v-if="canEdit" class="flex items-center gap-3">
                        <button
                          class="app-link"
                          :disabled="operating"
                          @click="openRecordDialog('ports', index)"
                        >
                          {{ t('common.edit') }}
                        </button>
                        <button
                          class="app-link-danger"
                          :disabled="operating"
                          @click="deleteRecord('ports', index)"
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

          <EnvironmentVariableListEditor
            :rows="environmentRows"
            :saved-rows="savedEnvironmentRows"
            :title="t('environment.title')"
            :disabled="operating"
            :editable="canEdit"
            @update:rows="updateEnvironmentRows"
            @save="persistEnvironment"
          />
        </TabsContent>

        <TabsContent v-if="!isNew" value="mounts" class="flex flex-col gap-4 outline-none">
          <DetailInfoCard :title="t('application.componentDetail.sections.mounts')">
            <template #actions>
              <button
                v-if="canEdit"
                class="app-button-primary h-9 px-3"
                :disabled="operating"
                @click="openMountDialog()"
              >
                <Plus class="size-4" />
                {{ t('common.add') }}
              </button>
            </template>
            <AppEmptyState v-if="form.mounts.length === 0" size="compact" />
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
                  <tr v-for="(row, index) in form.mounts" :key="`mount-${index}`">
                    <td class="min-w-28 text-foreground">
                      {{ mountTypeLabel(row.source_type) }}
                    </td>
                    <td class="min-w-56 break-all text-foreground">{{ row.source }}</td>
                    <td class="min-w-56 break-all text-foreground">{{ row.target }}</td>
                    <td class="min-w-20 text-foreground">
                      {{ row.read_only ? t('common.yes') : t('common.no') }}
                    </td>
                    <td class="w-36">
                      <div class="flex items-center gap-3">
                        <button
                          v-if="canEdit && row.source_type === 'controlled_file'"
                          class="app-link"
                          :disabled="operating"
                          @click="openMountContentDrawer(index)"
                        >
                          {{ t('application.componentDetail.fields.content') }}
                        </button>
                        <button
                          v-if="canEdit"
                          class="app-link"
                          :disabled="operating"
                          @click="openMountDialog(index)"
                        >
                          {{ t('common.edit') }}
                        </button>
                        <button
                          v-if="canEdit"
                          class="app-link-danger"
                          :disabled="operating"
                          @click="deleteMount(index)"
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

          <DetailInfoCard :title="t('application.componentDetail.sections.tmpfs')">
            <template #actions>
              <button
                v-if="canEdit"
                class="app-button-primary h-9 px-3"
                :disabled="operating"
                @click="openRecordDialog('tmpfs')"
              >
                <Plus class="size-4" />
                {{ t('common.add') }}
              </button>
            </template>
            <AppEmptyState v-if="form.tmpfs.length === 0" size="compact" />
            <div v-else class="overflow-x-auto">
              <table class="app-data-table min-w-[720px]">
                <thead>
                  <tr>
                    <th>{{ t('application.componentDetail.fields.target') }}</th>
                    <th>{{ t('application.componentDetail.fields.sizeBytes') }}</th>
                    <th>{{ t('application.componentDetail.fields.mode') }}</th>
                    <th class="w-32">{{ t('common.operation') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, index) in form.tmpfs" :key="`tmpfs-${index}`">
                    <td class="min-w-60 break-all text-foreground">{{ row.target }}</td>
                    <td class="min-w-52 text-foreground">{{ row.size_bytes }}</td>
                    <td class="min-w-40 text-foreground">{{ row.mode }}</td>
                    <td class="w-32">
                      <div v-if="canEdit" class="flex items-center gap-3">
                        <button
                          class="app-link"
                          :disabled="operating"
                          @click="openRecordDialog('tmpfs', index)"
                        >
                          {{ t('common.edit') }}
                        </button>
                        <button
                          class="app-link-danger"
                          :disabled="operating"
                          @click="deleteRecord('tmpfs', index)"
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
        </TabsContent>

        <TabsContent v-if="!isNew" value="advanced" class="flex flex-col gap-4 outline-none">
          <DetailInfoCard
            :title="t('application.componentDetail.sections.resources')"
            :editable="canEdit"
            :disabled="operating"
            @edit="openResourcesDialog"
          >
            <dl class="app-detail-info-grid">
              <div>
                <dt>{{ t('application.componentDetail.fields.limitCpus') }}</dt>
                <dd class="text-foreground">
                  {{ form.resources.limit_cpus || t('common.notSet') }}
                </dd>
              </div>
              <div>
                <dt>{{ t('application.componentDetail.fields.limitMemory') }}</dt>
                <dd class="text-foreground">
                  {{ form.resources.limit_memory || t('common.notSet') }}
                </dd>
              </div>
              <div>
                <dt>{{ t('application.componentDetail.fields.reservationCpus') }}</dt>
                <dd class="text-foreground">
                  {{ form.resources.reservation_cpus || t('common.notSet') }}
                </dd>
              </div>
              <div>
                <dt>{{ t('application.componentDetail.fields.reservationMemory') }}</dt>
                <dd class="text-foreground">
                  {{ form.resources.reservation_memory || t('common.notSet') }}
                </dd>
              </div>
            </dl>
          </DetailInfoCard>

          <DetailInfoCard :title="t('application.componentDetail.sections.devices')">
            <template #actions>
              <button
                v-if="canEdit"
                class="app-button-primary h-9 px-3"
                :disabled="operating"
                @click="openRecordDialog('devices')"
              >
                <Plus class="size-4" />
                {{ t('common.add') }}
              </button>
            </template>
            <AppEmptyState v-if="form.devices.length === 0" size="compact" />
            <div v-else class="overflow-x-auto">
              <table class="app-data-table min-w-[720px]">
                <thead>
                  <tr>
                    <th>{{ t('application.componentDetail.fields.driver') }}</th>
                    <th>{{ t('application.componentDetail.fields.count') }}</th>
                    <th>{{ t('application.componentDetail.fields.capabilities') }}</th>
                    <th class="w-32">{{ t('common.operation') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, index) in form.devices" :key="`device-${index}`">
                    <td class="min-w-52 text-foreground">{{ row.driver }}</td>
                    <td class="min-w-40 text-foreground">{{ row.count }}</td>
                    <td class="min-w-60 break-all text-foreground">
                      {{ row.capabilities.join(', ') }}
                    </td>
                    <td class="w-32">
                      <div v-if="canEdit" class="flex items-center gap-3">
                        <button
                          class="app-link"
                          :disabled="operating"
                          @click="openRecordDialog('devices', index)"
                        >
                          {{ t('common.edit') }}
                        </button>
                        <button
                          class="app-link-danger"
                          :disabled="operating"
                          @click="deleteRecord('devices', index)"
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

          <DetailInfoCard :title="t('application.componentDetail.sections.ulimits')">
            <template #actions>
              <button
                v-if="canEdit"
                class="app-button-primary h-9 px-3"
                :disabled="operating"
                @click="openRecordDialog('ulimits')"
              >
                <Plus class="size-4" />
                {{ t('common.add') }}
              </button>
            </template>
            <AppEmptyState v-if="form.ulimits.length === 0" size="compact" />
            <div v-else class="overflow-x-auto">
              <table class="app-data-table min-w-[720px]">
                <thead>
                  <tr>
                    <th>{{ t('application.componentDetail.fields.name') }}</th>
                    <th>{{ t('application.componentDetail.fields.soft') }}</th>
                    <th>{{ t('application.componentDetail.fields.hard') }}</th>
                    <th class="w-32">{{ t('common.operation') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, index) in form.ulimits" :key="`ulimit-${index}`">
                    <td class="min-w-52 text-foreground">{{ row.name }}</td>
                    <td class="min-w-52 text-foreground">{{ row.soft }}</td>
                    <td class="min-w-52 text-foreground">{{ row.hard }}</td>
                    <td class="w-32">
                      <div v-if="canEdit" class="flex items-center gap-3">
                        <button
                          class="app-link"
                          :disabled="operating"
                          @click="openRecordDialog('ulimits', index)"
                        >
                          {{ t('common.edit') }}
                        </button>
                        <button
                          class="app-link-danger"
                          :disabled="operating"
                          @click="deleteRecord('ulimits', index)"
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
        </TabsContent>

        <p v-if="formError" class="app-field-error text-sm">{{ formError }}</p>
      </TabsRoot>
    </template>

    <AppDialog
      :open="basicDialogOpen"
      :title="t('application.componentDetail.sections.basic')"
      width-class="w-[min(640px,calc(100vw-32px))]"
      @update:open="setBasicDialogOpen"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.component') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.name"
            class="app-input"
            :class="basicErrors.name ? 'app-input-error' : ''"
            type="text"
            :aria-invalid="basicErrors.name ? 'true' : undefined"
            @input="basicErrors.name = ''"
          />
          <p v-if="basicErrors.name" class="app-field-error" role="alert">
            {{ basicErrors.name }}
          </p>
        </div>
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.image') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.image"
            class="app-input"
            :class="basicErrors.image ? 'app-input-error' : ''"
            type="text"
            :aria-invalid="basicErrors.image ? 'true' : undefined"
            @input="basicErrors.image = ''"
          />
          <p v-if="basicErrors.image" class="app-field-error" role="alert">
            {{ basicErrors.image }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.pullPolicy') }}
            <span class="text-destructive">*</span>
          </label>
          <RawValueSelect v-model="form.pull_policy" :values="pullPolicyValues" />
          <p v-if="basicErrors.pullPolicy" class="app-field-error" role="alert">
            {{ basicErrors.pullPolicy }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.restartPolicy') }}
          </label>
          <RawValueSelect
            v-model="form.restart_policy"
            :placeholder="t('common.notSet')"
            :values="restartPolicyValues"
          />
        </div>
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.containerEntrypoint') }}
          </label>
          <textarea v-model="form.entrypoint" class="app-textarea" rows="3" />
        </div>
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.command') }}
          </label>
          <textarea v-model="form.command" class="app-textarea" rows="3" />
        </div>
      </div>
      <p v-if="formError" class="app-field-error">{{ formError }}</p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="cancelBasicEditing"
          @confirm="save('basic')"
        />
      </template>
    </AppDialog>

    <AppDialog
      :open="healthcheckDialogOpen"
      :title="t('application.componentDetail.sections.healthcheck')"
      width-class="w-[min(760px,calc(100vw-32px))]"
      @update:open="setHealthcheckDialogOpen"
    >
      <label class="flex items-center gap-2 text-sm text-foreground">
        <input
          v-model="form.healthcheck_disabled"
          class="app-checkbox"
          type="checkbox"
          @change="resetHealthcheckErrors"
        />
        {{ t('application.componentDetail.fields.disabled') }}
      </label>
      <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
        <div>
          <label class="app-field-label mb-1 block">
            {{ t('application.componentDetail.fields.testMode') }}
            <span v-if="!form.healthcheck_disabled" class="text-destructive">*</span>
          </label>
          <RawValueSelect
            v-model="form.healthcheck_test_mode"
            :disabled="form.healthcheck_disabled"
            :invalid="Boolean(healthcheckErrors.test_mode)"
            :placeholder="t('application.componentDetail.fields.testMode')"
            :values="healthcheckTestModes"
            @update:model-value="healthcheckErrors.test_mode = ''"
          />
          <p v-if="healthcheckErrors.test_mode" class="app-field-error" role="alert">
            {{ healthcheckErrors.test_mode }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1 block">
            {{ t('application.componentDetail.fields.interval') }}
          </label>
          <input
            v-model="form.healthcheck_interval"
            class="app-input"
            :disabled="form.healthcheck_disabled"
            type="text"
          />
        </div>
        <div>
          <label class="app-field-label mb-1 block">
            {{ t('application.componentDetail.fields.timeout') }}
          </label>
          <input
            v-model="form.healthcheck_timeout"
            class="app-input"
            :disabled="form.healthcheck_disabled"
            type="text"
          />
        </div>
        <div>
          <label class="app-field-label mb-1 block">
            {{ t('application.componentDetail.fields.retries') }}
          </label>
          <input
            v-model="form.healthcheck_retries"
            class="app-input"
            :disabled="form.healthcheck_disabled"
            inputmode="numeric"
            type="text"
            :class="healthcheckErrors.retries ? 'app-input-error' : ''"
            :aria-invalid="healthcheckErrors.retries ? 'true' : undefined"
            @input="healthcheckErrors.retries = ''"
          />
          <p v-if="healthcheckErrors.retries" class="app-field-error" role="alert">
            {{ healthcheckErrors.retries }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1 block">
            {{ t('application.componentDetail.fields.startPeriod') }}
          </label>
          <input
            v-model="form.healthcheck_start_period"
            class="app-input"
            :disabled="form.healthcheck_disabled"
            type="text"
          />
        </div>
        <div>
          <label class="app-field-label mb-1 block">
            {{ t('application.componentDetail.fields.startInterval') }}
          </label>
          <input
            v-model="form.healthcheck_start_interval"
            class="app-input"
            :disabled="form.healthcheck_disabled"
            type="text"
          />
        </div>
      </div>
      <div v-if="!form.healthcheck_disabled" class="mt-5 border-t border-border pt-5">
        <label class="app-field-label mb-1 block">
          {{ t('application.componentDetail.fields.test') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.healthcheck_test"
          class="app-input"
          :class="healthcheckErrors.test ? 'app-input-error' : ''"
          type="text"
          :aria-invalid="healthcheckErrors.test ? 'true' : undefined"
          @input="healthcheckErrors.test = ''"
        />
        <p v-if="healthcheckErrors.test" class="app-field-error" role="alert">
          {{ healthcheckErrors.test }}
        </p>
      </div>
      <p v-if="formError" class="app-field-error mt-4">{{ formError }}</p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="cancelHealthcheckEditing"
          @confirm="saveHealthcheck"
        />
      </template>
    </AppDialog>

    <AppDialog
      :open="resourcesDialogOpen"
      :title="t('application.componentDetail.sections.resources')"
      width-class="w-[min(640px,calc(100vw-32px))]"
      @update:open="setResourcesDialogOpen"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.limitCpus') }}
          </label>
          <input v-model="resourcesForm.limit_cpus" class="app-input" type="text" />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.limitMemory') }}
          </label>
          <input v-model="resourcesForm.limit_memory" class="app-input" type="text" />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.reservationCpus') }}
          </label>
          <input v-model="resourcesForm.reservation_cpus" class="app-input" type="text" />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.reservationMemory') }}
          </label>
          <input v-model="resourcesForm.reservation_memory" class="app-input" type="text" />
        </div>
      </div>
      <p v-if="resourcesFormError" class="app-field-error mt-4">
        {{ resourcesFormError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="closeResourcesDialog"
          @confirm="saveResources"
        />
      </template>
    </AppDialog>

    <AppDialog
      :open="recordDialogGroup !== null"
      :title="recordDialogTitle"
      width-class="w-[min(640px,calc(100vw-32px))]"
      body-class="space-y-4 px-6 py-4 text-sm"
      @update:open="setRecordDialogOpen"
    >
      <template v-if="recordDialogGroup === 'ports'">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.protocol') }}
              <span class="text-destructive">*</span>
            </label>
            <RawValueSelect
              v-model="portForm.protocol"
              :invalid="Boolean(recordErrors.protocol)"
              :values="endpointProtocolValues"
              @update:model-value="normalizePortMode"
            />
            <p v-if="recordErrors.protocol" class="app-field-error" role="alert">
              {{ recordErrors.protocol }}
            </p>
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.endpointMode') }}
              <span class="text-destructive">*</span>
            </label>
            <RawValueSelect
              v-model="portForm.mode"
              :invalid="Boolean(recordErrors.endpoint_mode)"
              :values="endpointModeValues(portForm.protocol)"
            />
            <p v-if="recordErrors.endpoint_mode" class="app-field-error" role="alert">
              {{ recordErrors.endpoint_mode }}
            </p>
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.hostPort') }}
              <span v-if="portRequiresListenPort" class="text-destructive">*</span>
            </label>
            <input
              v-model="portForm.host_port"
              class="app-input"
              :class="recordErrors.host_port ? 'app-input-error' : ''"
              :placeholder="t('application.componentDetail.fields.hostPort')"
              inputmode="numeric"
              type="text"
              :aria-invalid="recordErrors.host_port ? 'true' : undefined"
              @input="recordErrors.host_port = ''"
            />
            <p v-if="recordErrors.host_port" class="app-field-error" role="alert">
              {{ recordErrors.host_port }}
            </p>
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.containerPort') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="portForm.container_port"
              class="app-input"
              :class="recordErrors.container_port ? 'app-input-error' : ''"
              :placeholder="t('application.componentDetail.fields.containerPort')"
              inputmode="numeric"
              type="text"
              :aria-invalid="recordErrors.container_port ? 'true' : undefined"
              @input="recordErrors.container_port = ''"
            />
            <p v-if="recordErrors.container_port" class="app-field-error" role="alert">
              {{ recordErrors.container_port }}
            </p>
          </div>
        </div>
        <div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.bindAddress') }}
            </label>
            <input v-model="portForm.bind_address" class="app-input" type="text" />
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.entrypoint') }}
            </label>
            <input v-model="portForm.entrypoint" class="app-input" type="text" />
          </div>
          <div class="sm:col-span-2">
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.pathPrefix') }}
            </label>
            <input v-model="portForm.path_prefix" class="app-input" type="text" />
          </div>
        </div>
      </template>

      <template v-else-if="recordDialogGroup === 'dependencies'">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.dependency') }}
              <span class="text-destructive">*</span>
            </label>
            <RawValueSelect
              v-model="dependencyForm.name"
              :invalid="Boolean(recordErrors.name)"
              :placeholder="t('application.componentDetail.fields.dependency')"
              :values="dependencyNames"
              @update:model-value="recordErrors.name = ''"
            />
            <p v-if="recordErrors.name" class="app-field-error" role="alert">
              {{ recordErrors.name }}
            </p>
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.condition') }}
              <span class="text-destructive">*</span>
            </label>
            <RawValueSelect
              v-model="dependencyForm.condition"
              :invalid="Boolean(recordErrors.condition)"
              :placeholder="t('application.componentDetail.fields.condition')"
              :values="dependencyConditions"
              @update:model-value="recordErrors.condition = ''"
            />
            <p v-if="recordErrors.condition" class="app-field-error" role="alert">
              {{ recordErrors.condition }}
            </p>
          </div>
        </div>
      </template>

      <template v-else-if="recordDialogGroup === 'tmpfs'">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.target') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="tmpfsForm.target"
              class="app-input"
              :class="recordErrors.target ? 'app-input-error' : ''"
              type="text"
              :aria-invalid="recordErrors.target ? 'true' : undefined"
              @input="recordErrors.target = ''"
            />
            <p v-if="recordErrors.target" class="app-field-error" role="alert">
              {{ recordErrors.target }}
            </p>
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.sizeBytes') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="tmpfsForm.size_bytes"
              class="app-input"
              :class="recordErrors.size_bytes ? 'app-input-error' : ''"
              inputmode="numeric"
              type="text"
              :aria-invalid="recordErrors.size_bytes ? 'true' : undefined"
              @input="recordErrors.size_bytes = ''"
            />
            <p v-if="recordErrors.size_bytes" class="app-field-error" role="alert">
              {{ recordErrors.size_bytes }}
            </p>
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.mode') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="tmpfsForm.mode"
              class="app-input"
              :class="recordErrors.mode ? 'app-input-error' : ''"
              type="text"
              :aria-invalid="recordErrors.mode ? 'true' : undefined"
              @input="recordErrors.mode = ''"
            />
            <p v-if="recordErrors.mode" class="app-field-error" role="alert">
              {{ recordErrors.mode }}
            </p>
          </div>
        </div>
      </template>

      <template v-else-if="recordDialogGroup === 'ulimits'">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.name') }}
              <span class="text-destructive">*</span>
            </label>
            <RawValueSelect
              v-model="ulimitForm.name"
              :invalid="Boolean(recordErrors.name)"
              :placeholder="t('application.componentDetail.fields.name')"
              :values="ulimitNames"
              @update:model-value="recordErrors.name = ''"
            />
            <p v-if="recordErrors.name" class="app-field-error" role="alert">
              {{ recordErrors.name }}
            </p>
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.soft') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="ulimitForm.soft"
              class="app-input"
              :class="recordErrors.soft ? 'app-input-error' : ''"
              inputmode="numeric"
              type="text"
              :aria-invalid="recordErrors.soft ? 'true' : undefined"
              @input="recordErrors.soft = ''"
            />
            <p v-if="recordErrors.soft" class="app-field-error" role="alert">
              {{ recordErrors.soft }}
            </p>
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.hard') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="ulimitForm.hard"
              class="app-input"
              :class="recordErrors.hard ? 'app-input-error' : ''"
              inputmode="numeric"
              type="text"
              :aria-invalid="recordErrors.hard ? 'true' : undefined"
              @input="recordErrors.hard = ''"
            />
            <p v-if="recordErrors.hard" class="app-field-error" role="alert">
              {{ recordErrors.hard }}
            </p>
          </div>
        </div>
      </template>

      <template v-else-if="recordDialogGroup === 'devices'">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.driver') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="deviceForm.driver"
              class="app-input"
              :class="recordErrors.driver ? 'app-input-error' : ''"
              type="text"
              :aria-invalid="recordErrors.driver ? 'true' : undefined"
              @input="recordErrors.driver = ''"
            />
            <p v-if="recordErrors.driver" class="app-field-error" role="alert">
              {{ recordErrors.driver }}
            </p>
          </div>
          <div>
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.count') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="deviceForm.count"
              class="app-input"
              :class="recordErrors.count ? 'app-input-error' : ''"
              type="text"
              :aria-invalid="recordErrors.count ? 'true' : undefined"
              @input="recordErrors.count = ''"
            />
            <p v-if="recordErrors.count" class="app-field-error" role="alert">
              {{ recordErrors.count }}
            </p>
          </div>
          <div class="sm:col-span-2">
            <label class="app-field-label mb-1.5 block">
              {{ t('application.componentDetail.fields.capabilities') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="deviceForm.capabilities"
              class="app-input"
              :class="recordErrors.capabilities ? 'app-input-error' : ''"
              type="text"
              :aria-invalid="recordErrors.capabilities ? 'true' : undefined"
              @input="recordErrors.capabilities = ''"
            />
            <p v-if="recordErrors.capabilities" class="app-field-error" role="alert">
              {{ recordErrors.capabilities }}
            </p>
          </div>
        </div>
      </template>

      <p v-if="recordDialogError" class="app-field-error">{{ recordDialogError }}</p>

      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="closeRecordDialog"
          @confirm="saveRecord"
        />
      </template>
    </AppDialog>

    <AppDialog
      :open="mountDialogOpen"
      :title="editingMountIndex === null ? t('common.add') : t('common.edit')"
      width-class="w-[min(760px,calc(100vw-32px))]"
      body-class="space-y-4 px-6 py-4 text-sm"
      @update:open="setMountDialogOpen"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div class="sm:col-span-2">
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.sourceType') }}
            <span class="text-destructive">*</span>
          </label>
          <RawValueSelect
            :model-value="mountForm.source_type"
            :invalid="Boolean(mountErrors.source_type)"
            :placeholder="t('application.detail.placeholders.mountSourceType')"
            :values="mountSourceTypes"
            @update:model-value="updateMountFormSourceType"
          />
          <p v-if="mountErrors.source_type" class="app-field-error" role="alert">
            {{ mountErrors.source_type }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.source') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="mountForm.source"
            class="app-input"
            :class="mountErrors.source ? 'app-input-error' : ''"
            type="text"
            :aria-invalid="mountErrors.source ? 'true' : undefined"
            @input="mountErrors.source = ''"
          />
          <p v-if="mountErrors.source" class="app-field-error" role="alert">
            {{ mountErrors.source }}
          </p>
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.target') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="mountForm.target"
            class="app-input"
            :class="mountErrors.target ? 'app-input-error' : ''"
            type="text"
            :aria-invalid="mountErrors.target ? 'true' : undefined"
            @input="mountErrors.target = ''"
          />
          <p v-if="mountErrors.target" class="app-field-error" role="alert">
            {{ mountErrors.target }}
          </p>
        </div>
        <label class="flex items-center gap-2 self-end pb-2">
          <input v-model="mountForm.read_only" class="app-checkbox" type="checkbox" />
          <span class="text-sm text-foreground">
            {{ t('application.componentDetail.fields.readOnly') }}
          </span>
        </label>
      </div>
      <p v-if="mountDialogError" class="app-field-error" role="alert">{{ mountDialogError }}</p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="closeMountDialog"
          @confirm="saveMount"
        />
      </template>
    </AppDialog>

    <AppDrawer
      :open="mountContentDrawerOpen"
      :title="t('application.componentDetail.fields.content')"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="setMountContentDrawerOpen"
    >
      <div class="flex h-full min-h-0 flex-col gap-4 p-6 text-sm">
        <div class="shrink-0 space-y-3">
          <div class="space-y-1.5">
            <label class="app-field-label block">
              {{ t('application.componentDetail.fields.fileMode') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="mountContentForm.mode"
              class="app-input"
              :class="mountContentErrors.mode ? 'app-input-error' : ''"
              inputmode="numeric"
              placeholder="0600"
              :aria-invalid="mountContentErrors.mode ? 'true' : undefined"
              @input="mountContentErrors.mode = ''"
            />
            <p v-if="mountContentErrors.mode" class="app-field-error" role="alert">
              {{ mountContentErrors.mode }}
            </p>
          </div>
          <label class="flex items-center gap-2">
            <input
              v-model="mountContentForm.ignore_if_exists"
              class="app-checkbox"
              type="checkbox"
            />
            <span class="text-sm text-foreground">
              {{ t('application.componentDetail.fields.ignoreIfExists') }}
            </span>
          </label>
        </div>
        <label class="app-field-label shrink-0">
          {{ t('application.componentDetail.fields.content') }}
        </label>
        <div class="min-h-0 flex-1">
          <MonacoEditor v-model="mountContentForm.content" language="plaintext" height="100%" />
        </div>
        <p v-if="mountContentError" class="app-field-error shrink-0" role="alert">
          {{ mountContentError }}
        </p>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="closeMountContentDrawer"
          @confirm="saveMountContent"
        />
      </template>
    </AppDrawer>

    <AppDialog
      v-model:open="deleteDialogOpen"
      :title="t('application.componentDetail.actions.delete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{ t('application.componentDetail.deleteDescription', { name: form.name }) }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="deleteDialogOpen = false"
          @confirm="remove"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Plus, Save, Trash2, X } from '@lucide/vue';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { TabsContent, TabsList, TabsRoot, TabsTrigger } from 'reka-ui';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import DetailPageHeader from '@/components/DetailPageHeader.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import EnvironmentVariableListEditor from '@/components/EnvironmentVariableListEditor.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import type {
    VersionComponentAdvancedUpdateReq,
    VersionComponentResp,
    VersionResp,
  } from '@/gen/proto/orbit/v1/application/version';
  import {
    componentBasicRequestFromForm,
    componentCreateRequestFromForm,
    componentDependenciesRequestFromForm,
    componentDevicesRequestFromForm,
    componentFormFromResponse,
    componentMountsRequestFromForm,
    componentEndpointsRequestFromForm,
    componentResourcesRequestFromForm,
    componentRuntimeRequestFromForm,
    componentTmpfsRequestFromForm,
    componentUlimitsRequestFromForm,
    emptyComponentForm,
    type ComponentForm,
    type ComponentFormError,
    type DeviceRow,
    type MountRow,
    type PortRow,
    type TmpfsRow,
    type UlimitRow,
  } from './componentForm';
  import {
    cloneEnvironmentVariableRows,
    environmentVariableRowsFromEntries,
    type EnvironmentVariableEntry,
    type EnvironmentVariableListRow,
  } from '@/components/environmentVariableList';

  type ComponentTab = 'runtime' | 'connectivity' | 'mounts' | 'advanced';
  type ComponentSaveGroup = 'basic' | 'runtime';
  type ConnectivityGroup = 'ports' | 'dependencies';
  type RecordGroup = ConnectivityGroup | 'tmpfs' | 'ulimits' | 'devices';
  type PersistOutcome = 'saved' | 'invalid' | 'failed';
  type MountPersistOutcome = { status: PersistOutcome; error?: string };

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const versionId = route.params.versionId as string;
  const componentId = route.params.componentId as string;
  const isNew = componentId === 'new';

  const { loading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOperation } = useStatusAsync();
  const version = ref<VersionResp>();
  const application = ref<ApplicationResp>();
  const breadcrumbs = computed(() => {
    const currentVersion = version.value;
    const currentApplication = application.value;
    if (!currentVersion || !currentApplication) {
      return [];
    }
    return [
      {
        label: currentApplication.name,
        to: `/application/${currentApplication.id}`,
      },
      {
        label: currentVersion.label,
        to: `/version/${currentVersion.id}`,
      },
    ];
  });
  const component = ref<VersionComponentResp>();
  const form = reactive(emptyComponentForm());
  const environmentRows = ref<EnvironmentVariableListRow[]>([]);
  const savedEnvironmentRows = ref<EnvironmentVariableListRow[]>([]);
  const activeTab = ref<ComponentTab>('runtime');
  const basicDialogOpen = ref(false);
  const basicErrors = reactive({ name: '', image: '', pullPolicy: '' });
  const healthcheckDialogOpen = ref(false);
  const healthcheckErrors = reactive({ test_mode: '', test: '', retries: '' });
  const resourcesDialogOpen = ref(false);
  const recordDialogGroup = ref<RecordGroup | null>(null);
  const editingRecordIndex = ref<number | null>(null);
  const recordDialogError = ref('');
  const recordErrors = reactive({
    host_port: '',
    container_port: '',
    protocol: '',
    endpoint_mode: '',
    name: '',
    condition: '',
    target: '',
    size_bytes: '',
    mode: '',
    soft: '',
    hard: '',
    driver: '',
    count: '',
    capabilities: '',
  });
  const portForm = reactive<PortRow>({
    protocol: 'http',
    host_port: '',
    container_port: '',
    mode: 'host',
    bind_address: '',
    entrypoint: '',
    path_prefix: '',
  });
  const dependencyForm = reactive({ name: '', condition: '' });
  const tmpfsForm = reactive<TmpfsRow>({ target: '', size_bytes: '', mode: '' });
  const ulimitForm = reactive<UlimitRow>({ name: '', soft: '', hard: '' });
  const deviceForm = reactive({ driver: 'nvidia', count: 'all', capabilities: 'gpu' });
  const mountDialogOpen = ref(false);
  const editingMountIndex = ref<number | null>(null);
  const mountForm = reactive<MountRow>(emptyMount());
  const mountDialogError = ref('');
  const mountErrors = reactive({ source_type: '', source: '', target: '' });
  const mountContentDrawerOpen = ref(false);
  const contentMountIndex = ref<number | null>(null);
  const mountContentForm = reactive({
    content: '',
    mode: '0644',
    ignore_if_exists: false,
  });
  const mountContentError = ref('');
  const mountContentErrors = reactive({ mode: '' });
  const deleteDialogOpen = ref(false);
  const formError = ref('');
  const resourcesFormError = ref('');
  const resourcesForm = reactive({ ...emptyComponentForm().resources });

  const canEdit = computed(() => version.value?.status === 'unpublished');
  const tabs = computed(() => {
    const runtime = {
      id: 'runtime' as const,
      label: t('application.componentDetail.tabs.runtime'),
    };
    if (isNew) {
      return [
        runtime,
        { id: 'connectivity' as const, label: t('application.componentDetail.tabs.connectivity') },
      ];
    }
    return [
      runtime,
      { id: 'connectivity' as const, label: t('application.componentDetail.tabs.connectivity') },
      { id: 'mounts' as const, label: t('application.componentDetail.tabs.mounts') },
      { id: 'advanced' as const, label: t('application.componentDetail.tabs.advanced') },
    ];
  });
  const pullPolicyValues = ['always', 'missing', 'never'];
  const restartPolicyValues = ['no', 'unless-stopped'];
  const mountSourceTypes = ['directory', 'file', 'named_volume', 'controlled_file'];
  const endpointProtocolValues = ['http', 'tcp'];
  function endpointModeValues(protocol?: string) {
    return protocol === 'http'
      ? ['internal', 'local', 'host', 'gateway']
      : ['internal', 'local', 'host'];
  }
  const portRequiresListenPort = computed(() =>
    ['local', 'host'].includes(portForm.mode || 'host')
  );
  const dependencyConditions = [
    'service_started',
    'service_healthy',
    'service_completed_successfully',
  ];
  const healthcheckTestModes = ['CMD', 'CMD-SHELL'];
  const ulimitNames = ['memlock', 'nofile'];
  const dependencyNames = computed(() =>
    (version.value?.components ?? [])
      .filter((item) => item.id !== componentId)
      .map((item) => item.name)
  );
  const recordDialogTitle = computed(() =>
    editingRecordIndex.value === null ? t('common.add') : t('common.edit')
  );

  function assignForm(source: ReturnType<typeof emptyComponentForm>) {
    Object.assign(form, source);
    setEnvironmentRows(source.env);
  }

  function setEnvironmentRows(entries: EnvironmentVariableEntry[]) {
    const rows = environmentVariableRowsFromEntries(entries, 'version-component-environment');
    environmentRows.value = rows;
    savedEnvironmentRows.value = cloneEnvironmentVariableRows(rows);
  }

  function updateEnvironmentRows(rows: EnvironmentVariableListRow[]) {
    environmentRows.value = rows;
    form.env = rows.map(({ key, value }) => ({ key, value }));
  }

  function resetBasicErrors() {
    Object.assign(basicErrors, { name: '', image: '', pullPolicy: '' });
  }

  function validateBasicForm() {
    const name = form.name.trim();
    basicErrors.name = !name
      ? t('application.componentDetail.validation.componentNameRequired')
      : /^[a-z][a-z0-9-]*$/.test(name)
        ? ''
        : t('application.componentDetail.validation.componentName');
    basicErrors.image = form.image.trim()
      ? ''
      : t('application.componentDetail.validation.imageRequired');
    basicErrors.pullPolicy = ['always', 'missing', 'never'].includes(form.pull_policy)
      ? ''
      : t('application.componentDetail.validation.pullPolicy');
    return !basicErrors.name && !basicErrors.image && !basicErrors.pullPolicy;
  }

  function resetHealthcheckErrors() {
    Object.assign(healthcheckErrors, { test_mode: '', test: '', retries: '' });
  }

  function validateHealthcheckForm() {
    resetHealthcheckErrors();
    if (form.healthcheck_disabled) {
      return true;
    }
    const invalidMessage = t('application.componentDetail.validation.healthcheckTest');
    healthcheckErrors.test_mode = healthcheckTestModes.includes(form.healthcheck_test_mode)
      ? ''
      : invalidMessage;
    healthcheckErrors.test = form.healthcheck_test.trim() ? '' : invalidMessage;
    healthcheckErrors.retries =
      form.healthcheck_retries === '' || /^\d+$/.test(form.healthcheck_retries)
        ? ''
        : invalidMessage;
    return !healthcheckErrors.test_mode && !healthcheckErrors.test && !healthcheckErrors.retries;
  }

  function resetRecordErrors() {
    Object.assign(recordErrors, {
      host_port: '',
      container_port: '',
      protocol: '',
      endpoint_mode: '',
      key: '',
      name: '',
      condition: '',
      target: '',
      size_bytes: '',
      mode: '',
      soft: '',
      hard: '',
      driver: '',
      count: '',
      capabilities: '',
    });
  }

  function normalizePortMode() {
    if (!endpointModeValues(portForm.protocol).includes(portForm.mode || '')) {
      portForm.mode = 'internal';
    }
    recordErrors.protocol = '';
    recordErrors.endpoint_mode = '';
  }

  function validateRecordForm(group: RecordGroup) {
    resetRecordErrors();
    const invalidPort = t('application.componentDetail.validation.invalidPort');
    const invalidTmpfs = t('application.componentDetail.validation.invalidTmpfs');
    const invalidUlimit = t('application.componentDetail.validation.invalidUlimit');
    const isPort = (value: string) => /^\d+$/.test(value) && Number(value) <= 65535;
    if (group === 'ports') {
      const protocol = portForm.protocol || 'tcp';
      const mode = portForm.mode || 'host';
      const needsListenPort = mode === 'local' || mode === 'host';
      recordErrors.protocol =
        endpointProtocolValues.includes(protocol) && !(mode === 'gateway' && protocol !== 'http')
          ? ''
          : t('application.componentDetail.validation.invalidEndpoint');
      recordErrors.endpoint_mode = endpointModeValues(protocol).includes(mode)
        ? ''
        : t('application.componentDetail.validation.invalidEndpoint');
      recordErrors.host_port =
        (!needsListenPort && portForm.host_port === '') ||
        (isPort(portForm.host_port) && Number(portForm.host_port) > 0)
          ? ''
          : invalidPort;
      recordErrors.container_port =
        isPort(portForm.container_port) && Number(portForm.container_port) > 0 ? '' : invalidPort;
    } else if (group === 'dependencies') {
      const message = t('application.componentDetail.validation.rowIncomplete', {
        section: t('application.componentDetail.fields.dependency'),
      });
      recordErrors.name = dependencyForm.name ? '' : message;
      recordErrors.condition = dependencyForm.condition ? '' : message;
    } else if (group === 'tmpfs') {
      recordErrors.target = tmpfsForm.target.trim() ? '' : invalidTmpfs;
      recordErrors.size_bytes =
        /^\d+$/.test(tmpfsForm.size_bytes) &&
        Number(tmpfsForm.size_bytes) >= 1048576 &&
        Number(tmpfsForm.size_bytes) <= 8589934592
          ? ''
          : invalidTmpfs;
      recordErrors.mode = /^[0-7]{3,4}$/.test(tmpfsForm.mode) ? '' : invalidTmpfs;
    } else if (group === 'ulimits') {
      recordErrors.name = ulimitForm.name ? '' : invalidUlimit;
      recordErrors.soft = /^-?\d+$/.test(ulimitForm.soft) ? '' : invalidUlimit;
      recordErrors.hard = /^-?\d+$/.test(ulimitForm.hard) ? '' : invalidUlimit;
    } else {
      const invalidDevice = t('application.componentDetail.validation.invalidDevice');
      const capabilities = deviceForm.capabilities.split(',').map((item) => item.trim());
      recordErrors.driver =
        deviceForm.driver !== '' && !/\s/.test(deviceForm.driver) ? '' : invalidDevice;
      recordErrors.count =
        deviceForm.count === 'all' || /^[1-9]\d*$/.test(deviceForm.count) ? '' : invalidDevice;
      recordErrors.capabilities =
        capabilities.length > 0 &&
        capabilities.every((item) => item !== '' && !/\s/.test(item)) &&
        new Set(capabilities).size === capabilities.length &&
        (deviceForm.driver !== 'nvidia' || capabilities.includes('gpu'))
          ? ''
          : invalidDevice;
    }
    return !Object.values(recordErrors).some(Boolean);
  }

  function resetMountErrors() {
    Object.assign(mountErrors, { source_type: '', source: '', target: '' });
  }

  function validateMountForm() {
    const message = messageFor('mounts');
    mountErrors.source_type = mountSourceTypes.includes(mountForm.source_type) ? '' : message;
    mountErrors.source = mountForm.source.trim() ? '' : message;
    mountErrors.target = mountForm.target.trim() ? '' : message;
    return !mountErrors.source_type && !mountErrors.source && !mountErrors.target;
  }

  function validateMountContentForm() {
    mountContentErrors.mode = /^0[0-7]{3}$/.test(mountContentForm.mode) ? '' : messageFor('mounts');
    return !mountContentErrors.mode;
  }

  function cloneComponentForm(source: ComponentForm): ComponentForm {
    return {
      ...source,
      env: source.env.map((row) => ({ ...row })),
      ports: source.ports.map((row) => ({ ...row })),
      mounts: source.mounts.map((row) => ({ ...row })),
      dependencies: source.dependencies.map((row) => ({ ...row })),
      resources: { ...source.resources },
      tmpfs: source.tmpfs.map((row) => ({ ...row })),
      ulimits: source.ulimits.map((row) => ({ ...row })),
      devices: source.devices.map((row) => ({ ...row, capabilities: [...row.capabilities] })),
    };
  }

  function emptyMount(): MountRow {
    return {
      source_type: '',
      source: '',
      target: '',
      read_only: false,
      source_is_host_path: false,
      content: '',
      mode: '',
      ignore_if_exists: false,
    };
  }

  function mountTypeLabel(value: string): string {
    const labels: Record<string, string> = {
      directory: t('application.componentDetail.mountTypes.directory'),
      file: t('application.componentDetail.mountTypes.file'),
      named_volume: t('application.componentDetail.mountTypes.namedVolume'),
      controlled_file: t('application.componentDetail.mountTypes.controlledFile'),
    };
    return (labels[value] ?? value) || t('common.notSet');
  }

  async function fetchData() {
    try {
      await execute(async () => {
        const loadedVersion = await applicationApi.getVersion(versionId);
        version.value = loadedVersion;
        application.value = await applicationApi.get(loadedVersion.application_id);
        if (isNew) {
          assignForm(emptyComponentForm());
          return;
        }
        const loadedComponent = await applicationApi.getVersionComponent(versionId, componentId);
        component.value = loadedComponent;
        assignForm(componentFormFromResponse(loadedComponent));
      });
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('application.toast.loadVersionsFailed')
      );
      router.replace(`/version/${versionId}`);
    }
  }

  function startBasicEditing() {
    if (!component.value || healthcheckDialogOpen.value) {
      return;
    }
    assignForm(componentFormFromResponse(component.value));
    formError.value = '';
    resetBasicErrors();
    basicDialogOpen.value = true;
  }

  function cancelBasicEditing() {
    if (component.value) {
      assignForm(componentFormFromResponse(component.value));
    }
    formError.value = '';
    resetBasicErrors();
    basicDialogOpen.value = false;
  }

  function setBasicDialogOpen(open: boolean) {
    if (open) {
      basicDialogOpen.value = true;
      return;
    }
    cancelBasicEditing();
  }

  function startHealthcheckEditing() {
    if (basicDialogOpen.value) {
      return;
    }
    if (component.value) {
      assignForm(componentFormFromResponse(component.value));
    }
    form.healthcheck_enabled = true;
    formError.value = '';
    resetHealthcheckErrors();
    healthcheckDialogOpen.value = true;
  }

  function cancelHealthcheckEditing() {
    if (component.value) {
      assignForm(componentFormFromResponse(component.value));
    }
    formError.value = '';
    resetHealthcheckErrors();
    healthcheckDialogOpen.value = false;
  }

  function setHealthcheckDialogOpen(open: boolean) {
    if (open) {
      healthcheckDialogOpen.value = true;
      return;
    }
    cancelHealthcheckEditing();
  }

  function saveHealthcheck() {
    if (!validateHealthcheckForm()) {
      return;
    }
    if (isNew) {
      healthcheckDialogOpen.value = false;
      return;
    }
    void save('runtime');
  }

  function openResourcesDialog() {
    Object.assign(resourcesForm, form.resources);
    resourcesFormError.value = '';
    resourcesDialogOpen.value = true;
  }

  function closeResourcesDialog() {
    resourcesFormError.value = '';
    resourcesDialogOpen.value = false;
  }

  function setResourcesDialogOpen(open: boolean) {
    if (!open) {
      closeResourcesDialog();
    }
  }

  function currentAdvancedPayload(): VersionComponentAdvancedUpdateReq | undefined {
    if (!component.value) {
      return undefined;
    }
    return {
      resources: component.value.resources,
      tmpfs: component.value.tmpfs.map((row) => ({ ...row })),
      ulimits: component.value.ulimits.map((row) => ({ ...row })),
    };
  }

  async function persistAdvanced(
    payload: VersionComponentAdvancedUpdateReq,
    onSaved: () => void
  ): Promise<boolean> {
    try {
      await executeOperation(async () => {
        const updated = await applicationApi.updateVersionComponentAdvanced(
          versionId,
          componentId,
          payload
        );
        formError.value = '';
        component.value = updated;
        assignForm(componentFormFromResponse(updated));
        onSaved();
        toast.success(t('application.toast.updateSuccess'));
      });
      return true;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
      return false;
    }
  }

  async function saveResources() {
    const resources = componentResourcesRequestFromForm(resourcesForm);
    if (isNew) {
      form.resources = { ...resourcesForm };
      closeResourcesDialog();
      return;
    }
    const payload = currentAdvancedPayload();
    if (!payload) {
      return;
    }
    await persistAdvanced({ ...payload, resources }, closeResourcesDialog);
  }

  function nextRecordRows<T>(rows: T[], record: T): T[] {
    const nextRows = rows.map((row) => ({ ...row }));
    if (editingRecordIndex.value === null) {
      nextRows.push(record);
      return nextRows;
    }
    if (!nextRows[editingRecordIndex.value]) {
      return rows;
    }
    nextRows[editingRecordIndex.value] = record;
    return nextRows;
  }

  async function persistPorts(nextPorts: ComponentForm['ports']): Promise<PersistOutcome> {
    const draft = cloneComponentForm(form);
    draft.ports = nextPorts.map((row) => ({ ...row }));
    const result = componentEndpointsRequestFromForm(draft);
    if (!result.valid) {
      return 'invalid';
    }
    if (isNew) {
      form.ports = draft.ports;
      return 'saved';
    }
    try {
      await executeOperation(async () => {
        const updated = await applicationApi.updateVersionComponentEndpoints(
          versionId,
          componentId,
          result.value
        );
        component.value = updated;
        assignForm(componentFormFromResponse(updated));
        toast.success(t('application.toast.updateSuccess'));
      });
      return 'saved';
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
      return 'failed';
    }
  }

  async function persistEnvironment(entries: EnvironmentVariableEntry[]) {
    if (isNew) {
      updateEnvironmentRows(
        environmentVariableRowsFromEntries(entries, 'version-component-environment')
      );
      savedEnvironmentRows.value = cloneEnvironmentVariableRows(environmentRows.value);
      return;
    }
    try {
      await executeOperation(async () => {
        const updated = await applicationApi.updateVersionComponentEnv(versionId, componentId, {
          env: entries,
        });
        component.value = updated;
        assignForm(componentFormFromResponse(updated));
        toast.success(t('application.toast.updateSuccess'));
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  async function persistDependencies(
    nextDependencies: ComponentForm['dependencies']
  ): Promise<PersistOutcome> {
    const draft = cloneComponentForm(form);
    draft.dependencies = nextDependencies.map((row) => ({ ...row }));
    const result = componentDependenciesRequestFromForm(draft);
    if (!result.valid) {
      return 'invalid';
    }
    if (isNew) {
      form.dependencies = draft.dependencies;
      return 'saved';
    }
    try {
      await executeOperation(async () => {
        const updated = await applicationApi.updateVersionComponentDependencies(
          versionId,
          componentId,
          result.value
        );
        component.value = updated;
        assignForm(componentFormFromResponse(updated));
        toast.success(t('application.toast.updateSuccess'));
      });
      return 'saved';
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
      return 'failed';
    }
  }

  async function persistTmpfs(nextTmpfs: TmpfsRow[]): Promise<PersistOutcome> {
    const result = componentTmpfsRequestFromForm(nextTmpfs);
    if (!result.valid) {
      return 'invalid';
    }
    if (isNew) {
      form.tmpfs = nextTmpfs.map((row) => ({ ...row }));
      return 'saved';
    }
    const payload = currentAdvancedPayload();
    if (!payload) {
      return 'failed';
    }
    return (await persistAdvanced({ ...payload, tmpfs: result.value }, () => undefined))
      ? 'saved'
      : 'failed';
  }

  async function persistUlimits(nextUlimits: UlimitRow[]): Promise<PersistOutcome> {
    const result = componentUlimitsRequestFromForm(nextUlimits);
    if (!result.valid) {
      return 'invalid';
    }
    if (isNew) {
      form.ulimits = nextUlimits.map((row) => ({ ...row }));
      return 'saved';
    }
    const payload = currentAdvancedPayload();
    if (!payload) {
      return 'failed';
    }
    return (await persistAdvanced({ ...payload, ulimits: result.value }, () => undefined))
      ? 'saved'
      : 'failed';
  }

  async function persistDevices(nextDevices: DeviceRow[]): Promise<PersistOutcome> {
    const result = componentDevicesRequestFromForm(nextDevices);
    if (!result.valid) {
      return 'invalid';
    }
    if (isNew) {
      form.devices = nextDevices.map((row) => ({ ...row, capabilities: [...row.capabilities] }));
      return 'saved';
    }
    try {
      await executeOperation(async () => {
        const updated = await applicationApi.updateVersionComponentDevices(
          versionId,
          componentId,
          result.value
        );
        component.value = updated;
        assignForm(componentFormFromResponse(updated));
        toast.success(t('application.toast.updateSuccess'));
      });
      return 'saved';
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
      return 'failed';
    }
  }

  function openRecordDialog(group: RecordGroup, index?: number) {
    editingRecordIndex.value = index ?? null;
    recordDialogError.value = '';
    resetRecordErrors();
    if (group === 'ports') {
      Object.assign(
        portForm,
        index === undefined
          ? {
              protocol: 'http',
              host_port: '',
              container_port: '',
              mode: 'host',
              bind_address: '',
              entrypoint: '',
              path_prefix: '',
            }
          : form.ports[index]
      );
    } else if (group === 'dependencies') {
      Object.assign(
        dependencyForm,
        index === undefined ? { name: '', condition: '' } : form.dependencies[index]
      );
    } else if (group === 'tmpfs') {
      Object.assign(
        tmpfsForm,
        index === undefined ? { target: '', size_bytes: '', mode: '' } : form.tmpfs[index]
      );
    } else if (group === 'ulimits') {
      Object.assign(
        ulimitForm,
        index === undefined ? { name: '', soft: '', hard: '' } : form.ulimits[index]
      );
    } else {
      const device = index === undefined ? undefined : form.devices[index];
      Object.assign(
        deviceForm,
        device === undefined
          ? { driver: 'nvidia', count: 'all', capabilities: 'gpu' }
          : {
              driver: device.driver,
              count: device.count,
              capabilities: device.capabilities.join(', '),
            }
      );
    }
    recordDialogGroup.value = group;
  }

  function closeRecordDialog() {
    recordDialogGroup.value = null;
    editingRecordIndex.value = null;
    recordDialogError.value = '';
    resetRecordErrors();
  }

  function setRecordDialogOpen(open: boolean) {
    if (!open) {
      closeRecordDialog();
    }
  }

  async function saveRecord() {
    const group = recordDialogGroup.value;
    if (!group) {
      return;
    }
    if (!validateRecordForm(group)) {
      return;
    }
    const outcome =
      group === 'ports'
        ? await persistPorts(nextRecordRows(form.ports, { ...portForm }))
        : group === 'dependencies'
          ? await persistDependencies(nextRecordRows(form.dependencies, { ...dependencyForm }))
          : group === 'tmpfs'
            ? await persistTmpfs(nextRecordRows(form.tmpfs, { ...tmpfsForm }))
            : group === 'ulimits'
              ? await persistUlimits(nextRecordRows(form.ulimits, { ...ulimitForm }))
              : await persistDevices(
                  nextRecordRows(form.devices, {
                    driver: deviceForm.driver,
                    count: deviceForm.count,
                    capabilities: deviceForm.capabilities.split(',').map((item) => item.trim()),
                  })
                );
    if (outcome === 'invalid') {
      recordDialogError.value = messageFor(group);
      return;
    }
    if (outcome === 'saved') {
      closeRecordDialog();
    }
  }

  async function deleteRecord(group: RecordGroup, index: number) {
    const outcome =
      group === 'ports'
        ? await persistPorts(form.ports.filter((_, rowIndex) => rowIndex !== index))
        : group === 'dependencies'
          ? await persistDependencies(form.dependencies.filter((_, rowIndex) => rowIndex !== index))
          : group === 'tmpfs'
            ? await persistTmpfs(form.tmpfs.filter((_, rowIndex) => rowIndex !== index))
            : group === 'ulimits'
              ? await persistUlimits(form.ulimits.filter((_, rowIndex) => rowIndex !== index))
              : await persistDevices(form.devices.filter((_, rowIndex) => rowIndex !== index));
    if (outcome === 'invalid') {
      toast.error(messageFor(group));
    }
  }

  function openMountDialog(index?: number) {
    if (!isNew && !component.value) {
      return;
    }
    const mount = index === undefined ? emptyMount() : form.mounts[index];
    if (!mount) {
      return;
    }
    Object.assign(mountForm, { ...mount });
    editingMountIndex.value = index ?? null;
    mountDialogError.value = '';
    resetMountErrors();
    mountDialogOpen.value = true;
  }

  function closeMountDialog() {
    mountDialogOpen.value = false;
    editingMountIndex.value = null;
    mountDialogError.value = '';
    resetMountErrors();
  }

  function setMountDialogOpen(open: boolean) {
    if (open) {
      mountDialogOpen.value = true;
      return;
    }
    closeMountDialog();
  }

  function updateMountFormSourceType(value: string | number) {
    mountForm.source_type = String(value);
    mountErrors.source_type = '';
    if (mountForm.source_type !== 'directory' && mountForm.source_type !== 'file') {
      mountForm.source_is_host_path = false;
    }
    mountForm.content = '';
    mountForm.mode = mountForm.source_type === 'controlled_file' ? '0644' : '';
    mountForm.ignore_if_exists = false;
  }

  function openMountContentDrawer(index: number) {
    const mount = form.mounts[index];
    if (!mount || mount.source_type !== 'controlled_file') {
      return;
    }
    contentMountIndex.value = index;
    mountContentForm.content = mount.content;
    mountContentForm.mode = mount.mode;
    mountContentForm.ignore_if_exists = mount.ignore_if_exists;
    mountContentError.value = '';
    mountContentErrors.mode = '';
    mountContentDrawerOpen.value = true;
  }

  function closeMountContentDrawer() {
    mountContentDrawerOpen.value = false;
    contentMountIndex.value = null;
    mountContentError.value = '';
    mountContentErrors.mode = '';
  }

  function setMountContentDrawerOpen(open: boolean) {
    if (open) {
      mountContentDrawerOpen.value = true;
      return;
    }
    closeMountContentDrawer();
  }

  async function persistMounts(nextMounts: MountRow[]): Promise<MountPersistOutcome> {
    const draft = cloneComponentForm(form);
    draft.mounts = nextMounts.map((row) => ({ ...row }));
    const result = componentMountsRequestFromForm(draft);
    if (!result.valid) {
      return { status: 'invalid' };
    }
    if (isNew) {
      form.mounts = draft.mounts;
      return { status: 'saved' };
    }
    try {
      await executeOperation(async () => {
        const updated = await applicationApi.updateVersionComponentMounts(
          versionId,
          componentId,
          result.value
        );
        formError.value = '';
        component.value = updated;
        assignForm(componentFormFromResponse(updated));
        toast.success(t('application.toast.updateSuccess'));
      });
      return { status: 'saved' };
    } catch (error) {
      return {
        status: 'failed',
        error: error instanceof Error ? error.message : t('application.toast.updateFailed'),
      };
    }
  }

  async function saveMount() {
    if (!validateMountForm()) {
      return;
    }
    const nextMounts = form.mounts.map((row) => ({ ...row }));
    const mount = { ...mountForm };
    if (editingMountIndex.value === null) {
      nextMounts.push(mount);
    } else if (nextMounts[editingMountIndex.value]) {
      nextMounts[editingMountIndex.value] = mount;
    } else {
      return;
    }
    const outcome = await persistMounts(nextMounts);
    if (outcome.status === 'invalid') {
      mountDialogError.value = messageFor('mounts');
      return;
    }
    if (outcome.status === 'failed') {
      mountDialogError.value = outcome.error ?? t('application.toast.updateFailed');
      return;
    }
    if (outcome.status === 'saved') {
      closeMountDialog();
    }
  }

  async function saveMountContent() {
    if (contentMountIndex.value === null) {
      return;
    }
    const nextMounts = form.mounts.map((row) => ({ ...row }));
    const mount = nextMounts[contentMountIndex.value];
    if (!mount || mount.source_type !== 'controlled_file') {
      return;
    }
    if (!validateMountContentForm()) {
      return;
    }
    mount.content = mountContentForm.content;
    mount.mode = mountContentForm.mode;
    mount.ignore_if_exists = mountContentForm.ignore_if_exists;
    const outcome = await persistMounts(nextMounts);
    if (outcome.status === 'invalid') {
      mountContentError.value = messageFor('mounts');
      return;
    }
    if (outcome.status === 'failed') {
      mountContentError.value = outcome.error ?? t('application.toast.updateFailed');
      return;
    }
    if (outcome.status === 'saved') {
      closeMountContentDrawer();
    }
  }

  async function deleteMount(index: number) {
    const outcome = await persistMounts(form.mounts.filter((_, rowIndex) => rowIndex !== index));
    if (outcome.status === 'invalid') {
      toast.error(messageFor('mounts'));
    } else if (outcome.status === 'failed') {
      toast.error(outcome.error ?? t('application.toast.updateFailed'));
    }
  }

  function tabForError(error: ComponentFormError): ComponentTab {
    if (error === 'nameImage' || error === 'componentName' || error === 'pullPolicy') {
      return 'runtime';
    }
    if (error === 'command' || error === 'healthcheck') {
      return 'runtime';
    }
    if (error === 'dependencies') {
      return 'runtime';
    }
    if (error === 'ports' || error === 'env') {
      return 'connectivity';
    }
    if (error === 'mounts') {
      return 'mounts';
    }
    return 'advanced';
  }

  function messageFor(error: ComponentFormError): string {
    if (error === 'nameImage') {
      return t('application.componentDetail.validation.nameImage');
    }
    if (error === 'componentName') {
      return t('application.componentDetail.validation.componentName');
    }
    if (error === 'pullPolicy') {
      return t('application.componentDetail.validation.pullPolicy');
    }
    if (error === 'ports') {
      return t('application.componentDetail.validation.invalidPort');
    }
    if (error === 'tmpfs') {
      return t('application.componentDetail.validation.invalidTmpfs');
    }
    if (error === 'ulimits') {
      return t('application.componentDetail.validation.invalidUlimit');
    }
    if (error === 'healthcheck') {
      return t('application.componentDetail.validation.healthcheckTest');
    }
    if (error === 'dependencies') {
      return t('application.componentDetail.validation.rowIncomplete', {
        section: t('application.componentDetail.fields.dependency'),
      });
    }
    const section = t(`application.componentDetail.sections.${error}`);
    return t('application.componentDetail.validation.rowIncomplete', { section });
  }

  function showValidationError(error: ComponentFormError) {
    activeTab.value = tabForError(error);
    formError.value = messageFor(error);
  }

  async function save(group?: ComponentSaveGroup) {
    if (isNew) {
      if (!validateBasicForm()) {
        return;
      }
      const result = componentCreateRequestFromForm(form);
      if (!result.valid) {
        showValidationError(result.error);
        return;
      }
      formError.value = '';
      try {
        await executeOperation(async () => {
          const created = await applicationApi.createVersionComponent(versionId, result.value);
          toast.success(t('application.toast.updateSuccess'));
          await router.replace(`/version/${versionId}/component/${created.id}`);
        });
      } catch (error) {
        toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
      }
      return;
    }
    if (!group) {
      return;
    }

    try {
      await executeOperation(async () => {
        let updated: VersionComponentResp;
        if (group === 'basic') {
          if (!validateBasicForm()) {
            return;
          }
          const result = componentBasicRequestFromForm(form);
          if (!result.valid) {
            showValidationError(result.error);
            return;
          }
          updated = await applicationApi.updateVersionComponentBasic(
            versionId,
            componentId,
            result.value
          );
        } else {
          if (!validateHealthcheckForm()) {
            return;
          }
          const result = componentRuntimeRequestFromForm(form);
          if (!result.valid) {
            showValidationError(result.error);
            return;
          }
          updated = await applicationApi.updateVersionComponentRuntime(
            versionId,
            componentId,
            result.value
          );
        }
        formError.value = '';
        component.value = updated;
        assignForm(componentFormFromResponse(updated));
        basicDialogOpen.value = false;
        healthcheckDialogOpen.value = false;
        toast.success(t('application.toast.updateSuccess'));
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  async function remove() {
    if (isNew) {
      return;
    }
    try {
      await executeOperation(async () => {
        await applicationApi.deleteVersionComponent(versionId, componentId);
        toast.success(t('application.toast.updateSuccess'));
        await router.replace(`/version/${versionId}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  function goBack() {
    router.push(`/version/${versionId}`);
  }

  onMounted(fetchData);
</script>
