<template>
  <div class="flex flex-col gap-4 text-sm">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">
          {{ isNew ? t('application.componentDetail.newTitle') : form.name }}
        </h1>
        <AppBadge v-if="version" variant="status" :tone="versionStatusTone(version.status)">
          {{ version.status }}
        </AppBadge>
      </div>
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

    <AppSpinner v-if="loading" class="py-12" />

    <template v-else-if="version">
      <div
        v-if="isNew && !canEdit"
        class="app-surface p-5 text-sm text-muted-foreground app-detail-card"
      >
        {{ t('application.componentDetail.empty') }}
      </div>

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
          <section class="app-surface app-detail-card">
            <div class="app-section-header app-detail-section-header">
              <h2 class="app-detail-section-title">
                {{ t('application.componentDetail.sections.basic') }}
              </h2>
              <div v-if="canEdit && !isNew" class="flex items-center gap-2">
                <button class="app-button h-9 px-3" @click="startBasicEditing">
                  <Pencil class="size-4" />
                  {{ t('application.componentDetail.actions.edit') }}
                </button>
              </div>
            </div>

            <div v-if="isNew" class="grid grid-cols-1 gap-4 p-5 sm:grid-cols-2 sm:p-6">
              <div>
                <label class="app-field-label mb-1.5 block">
                  {{ t('application.detail.fields.component') }}
                  <span class="text-destructive">*</span>
                </label>
                <input v-model="form.name" class="app-input" type="text" />
              </div>
              <div>
                <label class="app-field-label mb-1.5 block">
                  {{ t('application.detail.fields.image') }}
                  <span class="text-destructive">*</span>
                </label>
                <input v-model="form.image" class="app-input" type="text" />
              </div>
              <div>
                <label class="app-field-label mb-1.5 block">
                  {{ t('application.componentDetail.fields.pullPolicy') }}
                </label>
                <RawValueSelect
                  v-model="form.pull_policy"
                  :placeholder="t('common.notSet')"
                  :values="pullPolicyValues"
                />
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
              <div class="min-w-0">
                <dt class="text-muted-foreground">{{ t('application.detail.fields.image') }}</dt>
                <dd class="mt-1 break-all text-foreground">{{ form.image }}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.pullPolicy') }}
                </dt>
                <dd class="mt-1 text-foreground">{{ form.pull_policy || t('common.notSet') }}</dd>
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
          </section>
          <section v-if="!isNew" class="app-surface app-detail-card">
            <div class="app-section-header app-detail-section-header">
              <h2 class="app-detail-section-title">
                {{ t('application.componentDetail.sections.healthcheck') }}
              </h2>
              <div v-if="canEdit" class="flex items-center gap-2">
                <button class="app-button h-9 px-3" @click="startHealthcheckEditing">
                  <Pencil class="size-4" />
                  {{ t('application.componentDetail.actions.edit') }}
                </button>
              </div>
            </div>

            <div class="app-detail-card-body">
              <p v-if="!form.healthcheck_enabled" class="text-sm text-muted-foreground">
                {{ t('common.notSet') }}
              </p>
              <dl
                v-else
                class="app-detail-fields grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3"
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
            </div>
          </section>

          <section v-if="!isNew" class="app-surface app-detail-card">
            <div class="app-section-header app-detail-section-header">
              <h2 class="app-detail-section-title">
                {{ t('application.componentDetail.fields.dependency') }}
              </h2>
              <div v-if="canEdit" class="flex items-center gap-2">
                <button class="app-button h-9 px-3" @click="openConnectivityDialog('dependencies')">
                  <Pencil class="size-4" />
                  {{ t('application.componentDetail.actions.edit') }}
                </button>
              </div>
            </div>
            <AppEmptyState v-if="form.dependencies.length === 0" size="compact" />
            <div v-else class="overflow-x-auto">
              <table class="app-data-table min-w-[560px]">
                <thead>
                  <tr>
                    <th>{{ t('application.componentDetail.fields.dependency') }}</th>
                    <th>{{ t('application.componentDetail.fields.condition') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, index) in form.dependencies" :key="`dependency-${index}`">
                    <td class="min-w-56 text-foreground">{{ row.name }}</td>
                    <td class="min-w-56 text-foreground">{{ row.condition }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>
        </TabsContent>

        <TabsContent v-if="!isNew" value="connectivity" class="flex flex-col gap-4 outline-none">
          <section class="app-surface app-detail-card">
            <div class="app-section-header app-detail-section-header">
              <h2 class="app-detail-section-title">
                {{ t('application.componentDetail.sections.ports') }}
              </h2>
              <div v-if="canEdit" class="flex items-center gap-2">
                <button class="app-button h-9 px-3" @click="openConnectivityDialog('ports')">
                  <Pencil class="size-4" />
                  {{ t('application.componentDetail.actions.edit') }}
                </button>
              </div>
            </div>

            <AppEmptyState v-if="form.ports.length === 0" size="compact" />
            <div v-else class="overflow-x-auto">
              <table class="app-data-table min-w-[560px]">
                <thead>
                  <tr>
                    <th>{{ t('application.componentDetail.fields.hostPort') }}</th>
                    <th>{{ t('application.componentDetail.fields.containerPort') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, index) in form.ports" :key="`port-${index}`">
                    <td class="min-w-56 text-foreground">{{ row.host_port }}</td>
                    <td class="min-w-56 text-foreground">{{ row.container_port }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <section class="app-surface app-detail-card">
            <div class="app-section-header app-detail-section-header">
              <h2 class="app-detail-section-title">
                {{ t('application.componentDetail.sections.env') }}
              </h2>
              <div v-if="canEdit" class="flex items-center gap-2">
                <button class="app-button h-9 px-3" @click="openConnectivityDialog('env')">
                  <Pencil class="size-4" />
                  {{ t('application.componentDetail.actions.edit') }}
                </button>
              </div>
            </div>
            <AppEmptyState v-if="form.env.length === 0" size="compact" />
            <div v-else class="overflow-x-auto">
              <table class="app-data-table min-w-[640px]">
                <thead>
                  <tr>
                    <th>{{ t('application.componentDetail.fields.key') }}</th>
                    <th>{{ t('application.componentDetail.fields.value') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, index) in form.env" :key="`env-${index}`">
                    <td class="min-w-56 break-all text-foreground">{{ row.key }}</td>
                    <td class="min-w-80 break-all text-foreground">{{ row.value }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>
        </TabsContent>

        <TabsContent v-if="!isNew" value="mounts" class="flex flex-col gap-4 outline-none">
          <section class="app-surface app-detail-card">
            <div class="app-section-header app-detail-section-header">
              <h2 class="app-detail-section-title">
                {{ t('application.componentDetail.sections.mounts') }}
              </h2>
              <div v-if="canEdit" class="flex items-center gap-2">
                <button
                  class="app-button-primary h-9 px-3"
                  :disabled="operating"
                  @click="openMountDialog()"
                >
                  <Plus class="size-4" />
                  {{ t('common.add') }}
                </button>
              </div>
            </div>
            <AppEmptyState v-if="form.mounts.length === 0" size="compact" />
            <div v-else class="overflow-x-auto">
              <table class="app-data-table min-w-[760px]">
                <thead>
                  <tr>
                    <th>{{ t('application.componentDetail.fields.sourceType') }}</th>
                    <th>{{ t('application.componentDetail.fields.source') }}</th>
                    <th>{{ t('application.componentDetail.fields.target') }}</th>
                    <th>{{ t('application.componentDetail.fields.readOnly') }}</th>
                    <th class="w-36">{{ t('common.operation') }}</th>
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
                      <div class="flex items-center gap-1">
                        <button
                          v-if="canEdit && row.source_type === 'controlled_file'"
                          class="app-icon-button"
                          :aria-label="t('application.componentDetail.fields.content')"
                          :disabled="operating"
                          :title="t('application.componentDetail.fields.content')"
                          @click="openMountContentDrawer(index)"
                        >
                          <FileText class="size-4" />
                        </button>
                        <button
                          v-if="canEdit"
                          class="app-icon-button"
                          :aria-label="t('common.edit')"
                          :disabled="operating"
                          :title="t('common.edit')"
                          @click="openMountDialog(index)"
                        >
                          <Pencil class="size-4" />
                        </button>
                        <button
                          v-if="canEdit"
                          class="app-icon-button"
                          :aria-label="t('common.delete')"
                          :disabled="operating"
                          :title="t('common.delete')"
                          @click="deleteMount(index)"
                        >
                          <Trash2 class="size-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>
        </TabsContent>

        <TabsContent v-if="!isNew" value="advanced" class="flex flex-col gap-4 outline-none">
          <section class="app-surface app-detail-card">
            <div class="app-section-header app-detail-section-header">
              <h2 class="app-detail-section-title">
                {{ t('application.componentDetail.sections.resources') }}
              </h2>
              <div v-if="canEdit" class="flex items-center gap-2">
                <button class="app-button h-9 px-3" @click="openResourcesDialog">
                  <Pencil class="size-4" />
                  {{ t('application.componentDetail.actions.edit') }}
                </button>
              </div>
            </div>
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
          </section>

          <section class="app-surface app-detail-card">
            <div class="app-section-header app-detail-section-header">
              <h2 class="app-detail-section-title">
                {{ t('application.componentDetail.sections.tmpfs') }}
              </h2>
              <button v-if="canEdit" class="app-button h-9 px-3" @click="openTmpfsDialog">
                <Pencil class="size-4" />
                {{ t('application.componentDetail.actions.edit') }}
              </button>
            </div>
            <AppEmptyState v-if="form.tmpfs.length === 0" size="compact" />
            <div v-else class="overflow-x-auto">
              <table class="app-data-table min-w-[720px]">
                <thead>
                  <tr>
                    <th>{{ t('application.componentDetail.fields.target') }}</th>
                    <th>{{ t('application.componentDetail.fields.sizeBytes') }}</th>
                    <th>{{ t('application.componentDetail.fields.mode') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, index) in form.tmpfs" :key="`tmpfs-${index}`">
                    <td class="min-w-60 break-all text-foreground">{{ row.target }}</td>
                    <td class="min-w-52 text-foreground">{{ row.size_bytes }}</td>
                    <td class="min-w-40 text-foreground">{{ row.mode }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <section class="app-surface app-detail-card">
            <div class="app-section-header app-detail-section-header">
              <h2 class="app-detail-section-title">
                {{ t('application.componentDetail.sections.ulimits') }}
              </h2>
              <button v-if="canEdit" class="app-button h-9 px-3" @click="openUlimitsDialog">
                <Pencil class="size-4" />
                {{ t('application.componentDetail.actions.edit') }}
              </button>
            </div>
            <AppEmptyState v-if="form.ulimits.length === 0" size="compact" />
            <div v-else class="overflow-x-auto">
              <table class="app-data-table min-w-[720px]">
                <thead>
                  <tr>
                    <th>{{ t('application.componentDetail.fields.name') }}</th>
                    <th>{{ t('application.componentDetail.fields.soft') }}</th>
                    <th>{{ t('application.componentDetail.fields.hard') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, index) in form.ulimits" :key="`ulimit-${index}`">
                    <td class="min-w-52 text-foreground">{{ row.name }}</td>
                    <td class="min-w-52 text-foreground">{{ row.soft }}</td>
                    <td class="min-w-52 text-foreground">{{ row.hard }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>
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
          <input v-model="form.name" class="app-input" type="text" />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.detail.fields.image') }}
            <span class="text-destructive">*</span>
          </label>
          <input v-model="form.image" class="app-input" type="text" />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.pullPolicy') }}
          </label>
          <RawValueSelect
            v-model="form.pull_policy"
            :placeholder="t('common.notSet')"
            :values="pullPolicyValues"
          />
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
            {{ t('application.componentDetail.fields.command') }}
          </label>
          <textarea v-model="form.command" class="app-textarea" rows="3" />
        </div>
      </div>
      <p v-if="formError" class="app-field-error">{{ formError }}</p>
      <template #footer>
        <button class="app-button" :disabled="operating" @click="cancelBasicEditing">
          <X class="size-4" />
          {{ t('application.componentDetail.actions.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="save('basic')">
          <Save class="size-4" />
          {{ t('application.componentDetail.actions.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      :open="healthcheckDialogOpen"
      :title="t('application.componentDetail.sections.healthcheck')"
      width-class="w-[min(760px,calc(100vw-32px))]"
      @update:open="setHealthcheckDialogOpen"
    >
      <label class="flex items-center gap-2 text-sm text-foreground">
        <input v-model="form.healthcheck_disabled" class="app-checkbox" type="checkbox" />
        {{ t('application.componentDetail.fields.disabled') }}
      </label>
      <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
        <div>
          <label class="app-field-label mb-1 block">
            {{ t('application.componentDetail.fields.testMode') }}
          </label>
          <RawValueSelect
            v-model="form.healthcheck_test_mode"
            :disabled="form.healthcheck_disabled"
            :placeholder="t('application.componentDetail.fields.testMode')"
            :values="healthcheckTestModes"
          />
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
          />
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
        </label>
        <input v-model="form.healthcheck_test" class="app-input" type="text" />
      </div>
      <p v-if="formError" class="app-field-error mt-4">{{ formError }}</p>
      <template #footer>
        <button class="app-button" :disabled="operating" @click="cancelHealthcheckEditing">
          <X class="size-4" />
          {{ t('application.componentDetail.actions.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="saveHealthcheck">
          <Save class="size-4" />
          {{ t('application.componentDetail.actions.save') }}
        </button>
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
        <button class="app-button" :disabled="operating" @click="closeResourcesDialog">
          <X class="size-4" />
          {{ t('application.componentDetail.actions.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="saveResources">
          <Save class="size-4" />
          {{ t('application.componentDetail.actions.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      :open="tmpfsDialogOpen"
      :title="t('application.componentDetail.sections.tmpfs')"
      width-class="w-[min(840px,calc(100vw-32px))]"
      body-class="max-h-[calc(100vh-160px)] space-y-4 overflow-y-auto px-6 py-4 text-sm"
      @update:open="setTmpfsDialogOpen"
    >
      <div class="overflow-x-auto">
        <table class="app-data-table min-w-[720px]">
          <thead>
            <tr>
              <th>{{ t('application.componentDetail.fields.target') }}</th>
              <th>{{ t('application.componentDetail.fields.sizeBytes') }}</th>
              <th>{{ t('application.componentDetail.fields.mode') }}</th>
              <th class="w-14">
                <button
                  class="app-icon-button"
                  :aria-label="t('application.componentDetail.actions.addRow')"
                  :title="t('application.componentDetail.actions.addRow')"
                  @click="tmpfsForm.push({ target: '', size_bytes: '', mode: '' })"
                >
                  <Plus class="size-4" />
                </button>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in tmpfsForm" :key="`tmpfs-${index}`">
              <td class="min-w-60">
                <input
                  v-model="row.target"
                  class="app-input"
                  :placeholder="t('application.componentDetail.fields.target')"
                  type="text"
                />
              </td>
              <td class="min-w-52">
                <input
                  v-model="row.size_bytes"
                  class="app-input"
                  :placeholder="t('application.componentDetail.fields.sizeBytes')"
                  inputmode="numeric"
                  type="text"
                />
              </td>
              <td class="min-w-40">
                <input
                  v-model="row.mode"
                  class="app-input"
                  :placeholder="t('application.componentDetail.fields.mode')"
                  type="text"
                />
              </td>
              <td class="w-14">
                <button
                  class="app-icon-button"
                  :aria-label="t('common.delete')"
                  :title="t('common.delete')"
                  @click="tmpfsForm.splice(index, 1)"
                >
                  <Trash2 class="size-4" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-if="tmpfsFormError" class="app-field-error">{{ tmpfsFormError }}</p>
      <template #footer>
        <button class="app-button" :disabled="operating" @click="closeTmpfsDialog">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="saveTmpfs">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      :open="ulimitsDialogOpen"
      :title="t('application.componentDetail.sections.ulimits')"
      width-class="w-[min(840px,calc(100vw-32px))]"
      body-class="max-h-[calc(100vh-160px)] space-y-4 overflow-y-auto px-6 py-4 text-sm"
      @update:open="setUlimitsDialogOpen"
    >
      <div class="overflow-x-auto">
        <table class="app-data-table min-w-[720px]">
          <thead>
            <tr>
              <th>{{ t('application.componentDetail.fields.name') }}</th>
              <th>{{ t('application.componentDetail.fields.soft') }}</th>
              <th>{{ t('application.componentDetail.fields.hard') }}</th>
              <th class="w-14">
                <button
                  class="app-icon-button"
                  :aria-label="t('application.componentDetail.actions.addRow')"
                  :title="t('application.componentDetail.actions.addRow')"
                  @click="ulimitsForm.push({ name: '', soft: '', hard: '' })"
                >
                  <Plus class="size-4" />
                </button>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in ulimitsForm" :key="`ulimit-${index}`">
              <td class="min-w-52">
                <RawValueSelect
                  v-model="row.name"
                  :placeholder="t('application.componentDetail.fields.name')"
                  :values="ulimitNames"
                />
              </td>
              <td class="min-w-52">
                <input
                  v-model="row.soft"
                  class="app-input"
                  :placeholder="t('application.componentDetail.fields.soft')"
                  inputmode="numeric"
                  type="text"
                />
              </td>
              <td class="min-w-52">
                <input
                  v-model="row.hard"
                  class="app-input"
                  :placeholder="t('application.componentDetail.fields.hard')"
                  inputmode="numeric"
                  type="text"
                />
              </td>
              <td class="w-14">
                <button
                  class="app-icon-button"
                  :aria-label="t('common.delete')"
                  :title="t('common.delete')"
                  @click="ulimitsForm.splice(index, 1)"
                >
                  <Trash2 class="size-4" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-if="ulimitsFormError" class="app-field-error">{{ ulimitsFormError }}</p>
      <template #footer>
        <button class="app-button" :disabled="operating" @click="closeUlimitsDialog">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="saveUlimits">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      :open="connectivityDialogGroup !== null"
      :title="connectivityDialogTitle"
      :width-class="connectivityDialogWidthClass"
      body-class="max-h-[calc(100vh-160px)] space-y-4 overflow-y-auto px-6 py-4 text-sm"
      @update:open="setConnectivityDialogOpen"
    >
      <template v-if="connectivityDialogGroup === 'ports'">
        <div class="overflow-x-auto">
          <table class="app-data-table min-w-[600px]">
            <thead>
              <tr>
                <th>{{ t('application.componentDetail.fields.hostPort') }}</th>
                <th>{{ t('application.componentDetail.fields.containerPort') }}</th>
                <th class="w-14">
                  <button
                    class="app-icon-button"
                    :aria-label="t('application.componentDetail.actions.addRow')"
                    :title="t('application.componentDetail.actions.addRow')"
                    @click="connectivityForm.ports.push({ host_port: '', container_port: '' })"
                  >
                    <Plus class="size-4" />
                  </button>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, index) in connectivityForm.ports" :key="`port-${index}`">
                <td class="min-w-64">
                  <input
                    v-model="row.host_port"
                    class="app-input"
                    :placeholder="t('application.componentDetail.fields.hostPort')"
                    inputmode="numeric"
                    type="text"
                  />
                </td>
                <td class="min-w-64">
                  <input
                    v-model="row.container_port"
                    class="app-input"
                    :placeholder="t('application.componentDetail.fields.containerPort')"
                    inputmode="numeric"
                    type="text"
                  />
                </td>
                <td class="w-14">
                  <button
                    class="app-icon-button"
                    :aria-label="t('common.delete')"
                    :title="t('common.delete')"
                    @click="connectivityForm.ports.splice(index, 1)"
                  >
                    <Trash2 class="size-4" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>

      <template v-else-if="connectivityDialogGroup === 'env'">
        <div class="overflow-x-auto">
          <table class="app-data-table min-w-[720px]">
            <thead>
              <tr>
                <th>{{ t('application.componentDetail.fields.key') }}</th>
                <th>{{ t('application.componentDetail.fields.value') }}</th>
                <th class="w-14">
                  <button
                    class="app-icon-button"
                    :aria-label="t('application.componentDetail.actions.addRow')"
                    :title="t('application.componentDetail.actions.addRow')"
                    @click="connectivityForm.env.push({ key: '', value: '' })"
                  >
                    <Plus class="size-4" />
                  </button>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, index) in connectivityForm.env" :key="`env-${index}`">
                <td class="min-w-64">
                  <input
                    v-model="row.key"
                    class="app-input"
                    :placeholder="t('application.componentDetail.fields.key')"
                    type="text"
                  />
                </td>
                <td class="min-w-96">
                  <input
                    v-model="row.value"
                    class="app-input"
                    :placeholder="t('application.componentDetail.fields.value')"
                    type="text"
                  />
                </td>
                <td class="w-14">
                  <button
                    class="app-icon-button"
                    :aria-label="t('common.delete')"
                    :title="t('common.delete')"
                    @click="connectivityForm.env.splice(index, 1)"
                  >
                    <Trash2 class="size-4" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>

      <template v-else-if="connectivityDialogGroup === 'dependencies'">
        <div class="overflow-x-auto">
          <table class="app-data-table min-w-[600px]">
            <thead>
              <tr>
                <th>{{ t('application.componentDetail.fields.dependency') }}</th>
                <th>{{ t('application.componentDetail.fields.condition') }}</th>
                <th class="w-14">
                  <button
                    class="app-icon-button"
                    :aria-label="t('application.componentDetail.actions.addRow')"
                    :title="t('application.componentDetail.actions.addRow')"
                    @click="connectivityForm.dependencies.push({ name: '', condition: '' })"
                  >
                    <Plus class="size-4" />
                  </button>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(row, index) in connectivityForm.dependencies"
                :key="`dependency-${index}`"
              >
                <td class="min-w-64">
                  <RawValueSelect
                    v-model="row.name"
                    :placeholder="t('application.componentDetail.fields.dependency')"
                    :values="dependencyNames"
                  />
                </td>
                <td class="min-w-64">
                  <RawValueSelect
                    v-model="row.condition"
                    :placeholder="t('application.componentDetail.fields.condition')"
                    :values="dependencyConditions"
                  />
                </td>
                <td class="w-14">
                  <button
                    class="app-icon-button"
                    :aria-label="t('common.delete')"
                    :title="t('common.delete')"
                    @click="connectivityForm.dependencies.splice(index, 1)"
                  >
                    <Trash2 class="size-4" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>

      <p v-if="connectivityFormError" class="app-field-error">{{ connectivityFormError }}</p>

      <template #footer>
        <button class="app-button" :disabled="operating" @click="cancelConnectivityEditing">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="saveConnectivity">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      :open="mountDialogOpen"
      :title="editingMountIndex === null ? t('common.add') : t('common.edit')"
      body-class="space-y-4 px-6 py-4 text-sm"
      @update:open="setMountDialogOpen"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.sourceType') }}
          </label>
          <RawValueSelect
            :model-value="mountForm.source_type"
            :placeholder="t('application.detail.placeholders.mountSourceType')"
            :values="mountSourceTypes"
            @update:model-value="updateMountFormSourceType"
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.source') }}
          </label>
          <input v-model="mountForm.source" class="app-input" type="text" />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('application.componentDetail.fields.target') }}
          </label>
          <input v-model="mountForm.target" class="app-input" type="text" />
        </div>
        <label class="flex items-center gap-2 self-end pb-2">
          <input v-model="mountForm.read_only" class="app-checkbox" type="checkbox" />
          <span class="text-sm text-foreground">
            {{ t('application.componentDetail.fields.readOnly') }}
          </span>
        </label>
        <label
          v-if="mountForm.source_type === 'directory' || mountForm.source_type === 'file'"
          class="flex items-center gap-2 self-end pb-2"
        >
          <input v-model="mountForm.source_is_host_path" class="app-checkbox" type="checkbox" />
          <span class="text-sm text-foreground">
            {{ t('application.componentDetail.fields.sourceIsHostPath') }}
          </span>
        </label>
      </div>
      <p v-if="mountDialogError" class="app-field-error">{{ mountDialogError }}</p>
      <template #footer>
        <button class="app-button" :disabled="operating" @click="closeMountDialog">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="saveMount">
          {{ t('common.save') }}
        </button>
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
          <label class="app-field-label">
            {{ t('application.componentDetail.fields.fileMode') }}
            <input
              v-model="mountContentForm.mode"
              class="app-input mt-1"
              inputmode="numeric"
              placeholder="0600"
            />
          </label>
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
        <p v-if="mountContentError" class="app-field-error shrink-0">
          {{ mountContentError }}
        </p>
      </div>
      <template #footer>
        <button class="app-button" :disabled="operating" @click="closeMountContentDrawer">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="saveMountContent">
          {{ t('common.save') }}
        </button>
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
        <button class="app-button" @click="deleteDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-destructive" :disabled="operating" @click="remove">
          {{ t('common.delete') }}
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, FileText, Pencil, Plus, Save, Trash2, X } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { TabsContent, TabsList, TabsRoot, TabsTrigger } from 'reka-ui';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type {
    VersionComponentAdvancedUpdateReq,
    VersionComponentResp,
    VersionResp,
  } from '@/gen/proto/orbit/v1/application/version';
  import { versionStatusTone } from '@/utils/status';
  import {
    componentBasicRequestFromForm,
    componentCreateRequestFromForm,
    componentDependenciesRequestFromForm,
    componentEnvRequestFromForm,
    componentFormFromResponse,
    componentMountsRequestFromForm,
    componentPortsRequestFromForm,
    componentResourcesRequestFromForm,
    componentRuntimeRequestFromForm,
    componentTmpfsRequestFromForm,
    componentUlimitsRequestFromForm,
    emptyComponentForm,
    type ComponentForm,
    type ComponentFormError,
    type MountRow,
    type TmpfsRow,
    type UlimitRow,
  } from './componentForm';

  type ComponentTab = 'runtime' | 'connectivity' | 'mounts' | 'advanced';
  type ComponentSaveGroup = 'basic' | 'runtime';
  type ConnectivityGroup = 'ports' | 'env' | 'dependencies';

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
  const component = ref<VersionComponentResp>();
  const form = reactive(emptyComponentForm());
  const connectivityForm = reactive(emptyComponentForm());
  const activeTab = ref<ComponentTab>('runtime');
  const basicDialogOpen = ref(false);
  const healthcheckDialogOpen = ref(false);
  const resourcesDialogOpen = ref(false);
  const tmpfsDialogOpen = ref(false);
  const ulimitsDialogOpen = ref(false);
  const connectivityDialogGroup = ref<ConnectivityGroup | null>(null);
  const mountDialogOpen = ref(false);
  const editingMountIndex = ref<number | null>(null);
  const mountForm = reactive<MountRow>(emptyMount());
  const mountDialogError = ref('');
  const mountContentDrawerOpen = ref(false);
  const contentMountIndex = ref<number | null>(null);
  const mountContentForm = reactive({
    content: '',
    mode: '0644',
    ignore_if_exists: false,
  });
  const mountContentError = ref('');
  const deleteDialogOpen = ref(false);
  const formError = ref('');
  const connectivityFormError = ref('');
  const resourcesFormError = ref('');
  const tmpfsFormError = ref('');
  const ulimitsFormError = ref('');
  const resourcesForm = reactive({ ...emptyComponentForm().resources });
  const tmpfsForm = ref<TmpfsRow[]>([]);
  const ulimitsForm = ref<UlimitRow[]>([]);

  const canEdit = computed(() => version.value?.status === 'unpublished');
  const tabs = computed(() => {
    const runtime = {
      id: 'runtime' as const,
      label: t('application.componentDetail.tabs.runtime'),
    };
    if (isNew) {
      return [runtime];
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
  const connectivityDialogTitle = computed(() => {
    const group = connectivityDialogGroup.value;
    if (group === 'dependencies') {
      return t('application.componentDetail.fields.dependency');
    }
    return group ? t(`application.componentDetail.sections.${group}`) : '';
  });
  const connectivityDialogWidthClass = computed(() =>
    connectivityDialogGroup.value === 'env'
      ? 'w-[min(840px,calc(100vw-32px))]'
      : 'w-[min(760px,calc(100vw-32px))]'
  );

  function assignForm(source: ReturnType<typeof emptyComponentForm>) {
    Object.assign(form, source);
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
    };
  }

  function assignConnectivityForm(source: ComponentForm) {
    Object.assign(connectivityForm, cloneComponentForm(source));
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

  function openConnectivityDialog(group: ConnectivityGroup) {
    if (isNew) {
      assignConnectivityForm(form);
    } else {
      const currentComponent = component.value;
      if (!currentComponent) {
        return;
      }
      assignConnectivityForm(componentFormFromResponse(currentComponent));
    }
    connectivityFormError.value = '';
    connectivityDialogGroup.value = group;
  }

  function cancelConnectivityEditing() {
    connectivityFormError.value = '';
    connectivityDialogGroup.value = null;
  }

  function setConnectivityDialogOpen(open: boolean) {
    if (!open) {
      cancelConnectivityEditing();
    }
  }

  function startBasicEditing() {
    if (!component.value || healthcheckDialogOpen.value) {
      return;
    }
    assignForm(componentFormFromResponse(component.value));
    formError.value = '';
    basicDialogOpen.value = true;
  }

  function cancelBasicEditing() {
    if (component.value) {
      assignForm(componentFormFromResponse(component.value));
    }
    formError.value = '';
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
    healthcheckDialogOpen.value = true;
  }

  function cancelHealthcheckEditing() {
    if (component.value) {
      assignForm(componentFormFromResponse(component.value));
    }
    formError.value = '';
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

  function openTmpfsDialog() {
    tmpfsForm.value = form.tmpfs.map((row) => ({ ...row }));
    tmpfsFormError.value = '';
    tmpfsDialogOpen.value = true;
  }

  function closeTmpfsDialog() {
    tmpfsForm.value = [];
    tmpfsFormError.value = '';
    tmpfsDialogOpen.value = false;
  }

  function setTmpfsDialogOpen(open: boolean) {
    if (!open) {
      closeTmpfsDialog();
    }
  }

  function openUlimitsDialog() {
    ulimitsForm.value = form.ulimits.map((row) => ({ ...row }));
    ulimitsFormError.value = '';
    ulimitsDialogOpen.value = true;
  }

  function closeUlimitsDialog() {
    ulimitsForm.value = [];
    ulimitsFormError.value = '';
    ulimitsDialogOpen.value = false;
  }

  function setUlimitsDialogOpen(open: boolean) {
    if (!open) {
      closeUlimitsDialog();
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

  async function persistAdvanced(payload: VersionComponentAdvancedUpdateReq, onSaved: () => void) {
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
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
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

  async function saveTmpfs() {
    const result = componentTmpfsRequestFromForm(tmpfsForm.value);
    if (!result.valid) {
      tmpfsFormError.value = messageFor(result.error);
      return;
    }
    if (isNew) {
      form.tmpfs = tmpfsForm.value.map((row) => ({ ...row }));
      closeTmpfsDialog();
      return;
    }
    const payload = currentAdvancedPayload();
    if (!payload) {
      return;
    }
    await persistAdvanced({ ...payload, tmpfs: result.value }, closeTmpfsDialog);
  }

  async function saveUlimits() {
    const result = componentUlimitsRequestFromForm(ulimitsForm.value);
    if (!result.valid) {
      ulimitsFormError.value = messageFor(result.error);
      return;
    }
    if (isNew) {
      form.ulimits = ulimitsForm.value.map((row) => ({ ...row }));
      closeUlimitsDialog();
      return;
    }
    const payload = currentAdvancedPayload();
    if (!payload) {
      return;
    }
    await persistAdvanced({ ...payload, ulimits: result.value }, closeUlimitsDialog);
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
    mountDialogOpen.value = true;
  }

  function closeMountDialog() {
    mountDialogOpen.value = false;
    editingMountIndex.value = null;
    mountDialogError.value = '';
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
    mountContentDrawerOpen.value = true;
  }

  function closeMountContentDrawer() {
    mountContentDrawerOpen.value = false;
    contentMountIndex.value = null;
    mountContentError.value = '';
  }

  function setMountContentDrawerOpen(open: boolean) {
    if (open) {
      mountContentDrawerOpen.value = true;
      return;
    }
    closeMountContentDrawer();
  }

  async function persistMounts(nextMounts: MountRow[]): Promise<'saved' | 'invalid' | 'failed'> {
    const draft = cloneComponentForm(form);
    draft.mounts = nextMounts.map((row) => ({ ...row }));
    const result = componentMountsRequestFromForm(draft);
    if (!result.valid) {
      return 'invalid';
    }
    if (isNew) {
      form.mounts = draft.mounts;
      return 'saved';
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
      return 'saved';
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
      return 'failed';
    }
  }

  async function saveMount() {
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
    if (outcome === 'invalid') {
      mountDialogError.value = messageFor('mounts');
      return;
    }
    if (outcome === 'saved') {
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
    mount.content = mountContentForm.content;
    mount.mode = mountContentForm.mode;
    mount.ignore_if_exists = mountContentForm.ignore_if_exists;
    const outcome = await persistMounts(nextMounts);
    if (outcome === 'invalid') {
      mountContentError.value = messageFor('mounts');
      return;
    }
    if (outcome === 'saved') {
      closeMountContentDrawer();
    }
  }

  async function deleteMount(index: number) {
    const outcome = await persistMounts(form.mounts.filter((_, rowIndex) => rowIndex !== index));
    if (outcome === 'invalid') {
      toast.error(messageFor('mounts'));
    }
  }

  function tabForError(error: ComponentFormError): ComponentTab {
    if (error === 'nameImage' || error === 'componentName') {
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

  function applyConnectivityDraft(group: ConnectivityGroup) {
    if (group === 'ports') {
      form.ports = connectivityForm.ports.map((row) => ({ ...row }));
      return;
    }
    if (group === 'env') {
      form.env = connectivityForm.env.map((row) => ({ ...row }));
      return;
    }
    form.dependencies = connectivityForm.dependencies.map((row) => ({ ...row }));
  }

  function showConnectivityValidationError(error: ComponentFormError) {
    connectivityFormError.value = messageFor(error);
  }

  async function persistConnectivity(
    group: ConnectivityGroup,
    update: () => Promise<VersionComponentResp>
  ) {
    if (isNew) {
      applyConnectivityDraft(group);
      cancelConnectivityEditing();
      return;
    }
    try {
      await executeOperation(async () => {
        const updated = await update();
        formError.value = '';
        component.value = updated;
        assignForm(componentFormFromResponse(updated));
        cancelConnectivityEditing();
        toast.success(t('application.toast.updateSuccess'));
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.updateFailed'));
    }
  }

  async function saveConnectivity() {
    const group = connectivityDialogGroup.value;
    if (!group) {
      return;
    }
    if (group === 'ports') {
      const result = componentPortsRequestFromForm(connectivityForm);
      if (!result.valid) {
        showConnectivityValidationError(result.error);
        return;
      }
      await persistConnectivity(group, () =>
        applicationApi.updateVersionComponentPorts(versionId, componentId, result.value)
      );
      return;
    }
    if (group === 'env') {
      const result = componentEnvRequestFromForm(connectivityForm);
      if (!result.valid) {
        showConnectivityValidationError(result.error);
        return;
      }
      await persistConnectivity(group, () =>
        applicationApi.updateVersionComponentEnv(versionId, componentId, result.value)
      );
      return;
    }
    const result = componentDependenciesRequestFromForm(connectivityForm);
    if (!result.valid) {
      showConnectivityValidationError(result.error);
      return;
    }
    await persistConnectivity(group, () =>
      applicationApi.updateVersionComponentDependencies(versionId, componentId, result.value)
    );
  }

  function showValidationError(error: ComponentFormError) {
    activeTab.value = tabForError(error);
    formError.value = messageFor(error);
  }

  async function save(group?: ComponentSaveGroup) {
    if (isNew) {
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
