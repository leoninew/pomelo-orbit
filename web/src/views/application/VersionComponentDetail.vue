<template>
  <div class="flex flex-col gap-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 items-center gap-2">
        <h1 class="min-w-0 break-words text-xl font-semibold text-foreground">
          {{ isNew ? t('application.componentDetail.newTitle') : form.name }}
        </h1>
        <AppBadge v-if="version" variant="status" :tone="versionStatusTone(version.status)">
          {{ version.status }}
        </AppBadge>
      </div>
      <div class="flex items-center gap-1">
        <button
          v-if="canEdit && !isNew && !editing"
          class="app-icon-button border border-input bg-background text-foreground"
          :aria-label="t('application.componentDetail.actions.edit')"
          :title="t('application.componentDetail.actions.edit')"
          @click="startEditing"
        >
          <Pencil class="size-4" />
        </button>
        <template v-if="canEdit && editing">
          <button
            class="app-icon-button border border-input bg-background text-foreground"
            :aria-label="t('application.componentDetail.actions.cancel')"
            :disabled="operating"
            :title="t('application.componentDetail.actions.cancel')"
            @click="cancelEditing"
          >
            <X class="size-4" />
          </button>
          <button
            class="app-icon-button bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground"
            :aria-label="t('application.componentDetail.actions.save')"
            :disabled="operating"
            :title="t('application.componentDetail.actions.save')"
            @click="save"
          >
            <Save class="size-4" />
          </button>
        </template>
        <button
          v-if="canEdit && !isNew"
          class="app-icon-button text-destructive hover:bg-destructive/10 hover:text-destructive"
          :aria-label="t('application.componentDetail.actions.delete')"
          :disabled="operating"
          :title="t('application.componentDetail.actions.delete')"
          @click="deleteDialogOpen = true"
        >
          <Trash2 class="size-4" />
        </button>
        <button
          class="app-icon-button border border-input bg-background text-foreground"
          :aria-label="t('common.back')"
          :title="t('common.back')"
          @click="goBack"
        >
          <ArrowLeft class="size-4" />
        </button>
      </div>
    </header>

    <AppSpinner v-if="loading" class="py-12" />

    <template v-else-if="version">
      <div v-if="isNew && !canEdit" class="app-surface p-5 text-sm text-muted-foreground">
        {{ t('application.componentDetail.empty') }}
      </div>

      <template v-else>
        <nav
          class="app-surface flex items-center gap-1 overflow-x-auto p-2"
          :aria-label="t('application.componentDetail.title')"
          role="tablist"
        >
          <button
            v-for="tab in tabs"
            :key="tab.id"
            class="shrink-0 rounded-lg px-3 py-2 text-sm font-medium transition-colors"
            :class="
              activeTab === tab.id
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground'
            "
            :aria-selected="activeTab === tab.id"
            role="tab"
            type="button"
            @click="activeTab = tab.id"
          >
            {{ tab.label }}
          </button>
        </nav>

        <section v-show="activeTab === 'overview'" class="app-surface" role="tabpanel">
          <div class="app-section-header">
            <h2 class="text-base font-semibold text-foreground">
              {{ t('application.componentDetail.sections.basic') }}
            </h2>
          </div>

          <div v-if="editing" class="grid grid-cols-1 gap-4 p-5 sm:grid-cols-2 sm:p-6">
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
          </div>

          <dl v-else class="grid grid-cols-1 gap-x-8 gap-y-5 p-5 text-sm sm:grid-cols-2 sm:p-6">
            <div class="min-w-0">
              <dt class="text-muted-foreground">{{ t('application.detail.fields.component') }}</dt>
              <dd class="mt-1 break-words font-medium text-foreground">{{ form.name }}</dd>
            </div>
            <div class="min-w-0">
              <dt class="text-muted-foreground">{{ t('application.detail.fields.image') }}</dt>
              <dd class="mt-1 break-all font-mono text-sm text-foreground">{{ form.image }}</dd>
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
              <dd class="mt-1 text-foreground">{{ form.restart_policy || t('common.notSet') }}</dd>
            </div>
          </dl>
        </section>

        <section v-show="activeTab === 'runtime'" class="app-surface" role="tabpanel">
          <div class="app-section-header">
            <h2 class="text-base font-semibold text-foreground">
              {{ t('application.componentDetail.tabs.runtime') }}
            </h2>
          </div>

          <div class="p-5 sm:p-6">
            <div class="flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold text-foreground">
                {{ t('application.componentDetail.sections.command') }}
              </h3>
              <button
                v-if="editing"
                class="app-icon-button"
                :aria-label="t('application.componentDetail.actions.addRow')"
                :title="t('application.componentDetail.actions.addRow')"
                @click="form.command.push({ value: '' })"
              >
                <Plus class="size-4" />
              </button>
            </div>

            <template v-if="editing">
              <div v-if="form.command.length === 0" class="app-tip mt-4">
                {{ t('application.componentDetail.empty') }}
              </div>
              <div v-else class="mt-4 space-y-2">
                <div
                  v-for="(row, index) in form.command"
                  :key="`command-${index}`"
                  class="flex items-center gap-2"
                >
                  <input v-model="row.value" class="app-input min-w-0 flex-1" type="text" />
                  <button
                    class="app-icon-button shrink-0"
                    :aria-label="t('common.delete')"
                    :title="t('common.delete')"
                    @click="form.command.splice(index, 1)"
                  >
                    <Trash2 class="size-4" />
                  </button>
                </div>
              </div>
            </template>
            <p v-else-if="form.command.length === 0" class="mt-3 text-sm text-muted-foreground">
              {{ t('common.notSet') }}
            </p>
            <code
              v-else
              class="mt-3 block break-all rounded-lg border border-border bg-muted/40 px-3 py-2 text-sm text-foreground"
            >
              {{ commandPreview }}
            </code>

            <div class="mt-6 flex items-center justify-between gap-3 border-t border-border pt-5">
              <h3 class="text-sm font-semibold text-foreground">
                {{ t('application.componentDetail.fields.args') }}
              </h3>
              <button
                v-if="editing"
                class="app-icon-button"
                :aria-label="t('application.componentDetail.actions.addRow')"
                :title="t('application.componentDetail.actions.addRow')"
                @click="form.args.push({ value: '' })"
              >
                <Plus class="size-4" />
              </button>
            </div>

            <template v-if="editing">
              <div v-if="form.args.length === 0" class="app-tip mt-4">
                {{ t('application.componentDetail.empty') }}
              </div>
              <div v-else class="mt-4 space-y-2">
                <div
                  v-for="(row, index) in form.args"
                  :key="`args-${index}`"
                  class="flex items-center gap-2"
                >
                  <input v-model="row.value" class="app-input min-w-0 flex-1" type="text" />
                  <button
                    class="app-icon-button shrink-0"
                    :aria-label="t('common.delete')"
                    :title="t('common.delete')"
                    @click="form.args.splice(index, 1)"
                  >
                    <Trash2 class="size-4" />
                  </button>
                </div>
              </div>
            </template>
            <p v-else-if="form.args.length === 0" class="mt-3 text-sm text-muted-foreground">
              {{ t('common.notSet') }}
            </p>
            <div v-else class="mt-3 flex flex-wrap gap-2">
              <code
                v-for="(row, index) in form.args"
                :key="`argument-${index}`"
                class="rounded-md bg-muted px-2 py-1 text-xs text-foreground"
              >
                {{ row.value }}
              </code>
            </div>
          </div>

          <div class="border-t border-border p-5 sm:p-6">
            <div class="flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold text-foreground">
                {{ t('application.componentDetail.sections.healthcheck') }}
              </h3>
              <button
                v-if="editing && !form.healthcheck_enabled"
                class="app-button h-8 px-2.5 text-xs"
                @click="form.healthcheck_enabled = true"
              >
                <Plus class="size-3.5" />
                {{ t('application.componentDetail.actions.enableHealthcheck') }}
              </button>
            </div>

            <p
              v-if="!form.healthcheck_enabled && !editing"
              class="mt-3 text-sm text-muted-foreground"
            >
              {{ t('common.notSet') }}
            </p>
            <template v-else-if="form.healthcheck_enabled">
              <template v-if="editing">
                <label class="mt-4 flex items-center gap-2 text-sm text-foreground">
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
                  <div class="flex items-center justify-between gap-3">
                    <h4 class="text-sm font-medium text-foreground">
                      {{ t('application.componentDetail.fields.test') }}
                    </h4>
                    <button
                      class="app-icon-button"
                      :aria-label="t('application.componentDetail.actions.addRow')"
                      :title="t('application.componentDetail.actions.addRow')"
                      @click="form.healthcheck_test.push({ value: '' })"
                    >
                      <Plus class="size-4" />
                    </button>
                  </div>
                  <div class="mt-3 space-y-2">
                    <div
                      v-for="(row, index) in form.healthcheck_test"
                      :key="`healthcheck-test-${index}`"
                      class="flex items-center gap-2"
                    >
                      <input v-model="row.value" class="app-input min-w-0 flex-1" type="text" />
                      <button
                        class="app-icon-button shrink-0"
                        :aria-label="t('common.delete')"
                        :title="t('common.delete')"
                        @click="form.healthcheck_test.splice(index, 1)"
                      >
                        <Trash2 class="size-4" />
                      </button>
                    </div>
                  </div>
                </div>
              </template>
              <dl
                v-else
                class="mt-4 grid grid-cols-1 gap-x-8 gap-y-4 text-sm sm:grid-cols-2 lg:grid-cols-3"
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
                  <dd v-if="form.healthcheck_test.length === 0" class="mt-1 text-foreground">
                    {{ t('common.notSet') }}
                  </dd>
                  <dd v-else class="mt-2 flex flex-wrap gap-2">
                    <code
                      v-for="(row, index) in form.healthcheck_test"
                      :key="`healthcheck-summary-${index}`"
                      class="break-all rounded-md bg-muted px-2 py-1 text-xs text-foreground"
                    >
                      {{ row.value }}
                    </code>
                  </dd>
                </div>
              </dl>
            </template>
          </div>

          <div class="border-t border-border p-5 sm:p-6">
            <h3 class="text-sm font-semibold text-foreground">
              {{ t('application.componentDetail.sections.resources') }}
            </h3>
            <div v-if="editing" class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label class="app-field-label mb-1 block">
                  {{ t('application.componentDetail.fields.limitCpus') }}
                </label>
                <input v-model="form.resources.limit_cpus" class="app-input" type="text" />
              </div>
              <div>
                <label class="app-field-label mb-1 block">
                  {{ t('application.componentDetail.fields.limitMemory') }}
                </label>
                <input v-model="form.resources.limit_memory" class="app-input" type="text" />
              </div>
              <div>
                <label class="app-field-label mb-1 block">
                  {{ t('application.componentDetail.fields.reservationCpus') }}
                </label>
                <input v-model="form.resources.reservation_cpus" class="app-input" type="text" />
              </div>
              <div>
                <label class="app-field-label mb-1 block">
                  {{ t('application.componentDetail.fields.reservationMemory') }}
                </label>
                <input v-model="form.resources.reservation_memory" class="app-input" type="text" />
              </div>
            </div>
            <dl v-else class="mt-4 grid grid-cols-2 gap-x-6 gap-y-4 text-sm sm:grid-cols-4">
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.limitCpus') }}
                </dt>
                <dd class="mt-1 text-foreground">
                  {{ form.resources.limit_cpus || t('common.notSet') }}
                </dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.limitMemory') }}
                </dt>
                <dd class="mt-1 text-foreground">
                  {{ form.resources.limit_memory || t('common.notSet') }}
                </dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.reservationCpus') }}
                </dt>
                <dd class="mt-1 text-foreground">
                  {{ form.resources.reservation_cpus || t('common.notSet') }}
                </dd>
              </div>
              <div>
                <dt class="text-muted-foreground">
                  {{ t('application.componentDetail.fields.reservationMemory') }}
                </dt>
                <dd class="mt-1 text-foreground">
                  {{ form.resources.reservation_memory || t('common.notSet') }}
                </dd>
              </div>
            </dl>
          </div>
        </section>

        <section v-show="activeTab === 'connectivity'" class="app-surface" role="tabpanel">
          <div class="app-section-header">
            <h2 class="text-base font-semibold text-foreground">
              {{ t('application.componentDetail.tabs.connectivity') }}
            </h2>
          </div>

          <div class="p-5 sm:p-6">
            <div class="flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold text-foreground">
                {{ t('application.componentDetail.sections.ports') }}
              </h3>
              <button
                v-if="editing"
                class="app-icon-button"
                :aria-label="t('application.componentDetail.actions.addRow')"
                :title="t('application.componentDetail.actions.addRow')"
                @click="form.ports.push({ host_port: '', container_port: '' })"
              >
                <Plus class="size-4" />
              </button>
            </div>
            <p
              v-if="form.ports.length === 0 && !editing"
              class="mt-3 text-sm text-muted-foreground"
            >
              {{ t('common.notSet') }}
            </p>
            <div v-if="editing" class="mt-4">
              <div
                class="hidden grid-cols-[1fr_1fr_auto] gap-2 px-3 text-xs font-medium text-muted-foreground sm:grid"
              >
                <span>{{ t('application.componentDetail.fields.hostPort') }}</span>
                <span>{{ t('application.componentDetail.fields.containerPort') }}</span>
                <span class="w-9"></span>
              </div>
              <div class="mt-2 space-y-2">
                <div
                  v-for="(row, index) in form.ports"
                  :key="`port-${index}`"
                  class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1fr_auto] sm:items-center"
                >
                  <input
                    v-model="row.host_port"
                    class="app-input"
                    :placeholder="t('application.componentDetail.fields.hostPort')"
                    inputmode="numeric"
                    type="text"
                  />
                  <input
                    v-model="row.container_port"
                    class="app-input"
                    :placeholder="t('application.componentDetail.fields.containerPort')"
                    inputmode="numeric"
                    type="text"
                  />
                  <button
                    class="app-icon-button justify-self-end"
                    :aria-label="t('common.delete')"
                    :title="t('common.delete')"
                    @click="form.ports.splice(index, 1)"
                  >
                    <Trash2 class="size-4" />
                  </button>
                </div>
              </div>
            </div>
            <div v-else class="mt-4 divide-y divide-border">
              <div
                v-for="(row, index) in form.ports"
                :key="`port-${index}`"
                class="grid grid-cols-1 gap-3 py-4 first:pt-0 sm:grid-cols-[1fr_1fr_auto] sm:items-end"
              >
                <div>
                  <p class="text-xs font-medium text-muted-foreground">
                    {{ t('application.componentDetail.fields.hostPort') }}
                  </p>
                  <p class="mt-1 text-sm text-foreground">{{ row.host_port }}</p>
                </div>
                <div>
                  <p class="text-xs font-medium text-muted-foreground">
                    {{ t('application.componentDetail.fields.containerPort') }}
                  </p>
                  <p class="mt-1 text-sm text-foreground">{{ row.container_port }}</p>
                </div>
              </div>
            </div>
          </div>

          <div class="border-t border-border p-5 sm:p-6">
            <div class="flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold text-foreground">
                {{ t('application.componentDetail.sections.env') }}
              </h3>
              <button
                v-if="editing"
                class="app-icon-button"
                :aria-label="t('application.componentDetail.actions.addRow')"
                :title="t('application.componentDetail.actions.addRow')"
                @click="form.env.push({ key: '', value: '' })"
              >
                <Plus class="size-4" />
              </button>
            </div>
            <p v-if="form.env.length === 0 && !editing" class="mt-3 text-sm text-muted-foreground">
              {{ t('common.notSet') }}
            </p>
            <div v-if="editing" class="mt-4">
              <div
                class="hidden grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)_auto] gap-2 px-3 text-xs font-medium text-muted-foreground sm:grid"
              >
                <span>{{ t('application.componentDetail.fields.key') }}</span>
                <span>{{ t('application.componentDetail.fields.value') }}</span>
                <span class="w-9"></span>
              </div>
              <div class="mt-2 space-y-2">
                <div
                  v-for="(row, index) in form.env"
                  :key="`env-${index}`"
                  class="grid grid-cols-1 gap-2 sm:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)_auto] sm:items-center"
                >
                  <input
                    v-model="row.key"
                    class="app-input"
                    :placeholder="t('application.componentDetail.fields.key')"
                    type="text"
                  />
                  <input
                    v-model="row.value"
                    class="app-input"
                    :placeholder="t('application.componentDetail.fields.value')"
                    type="text"
                  />
                  <button
                    class="app-icon-button justify-self-end"
                    :aria-label="t('common.delete')"
                    :title="t('common.delete')"
                    @click="form.env.splice(index, 1)"
                  >
                    <Trash2 class="size-4" />
                  </button>
                </div>
              </div>
            </div>
            <div v-else class="mt-4 divide-y divide-border">
              <div
                v-for="(row, index) in form.env"
                :key="`env-${index}`"
                class="grid grid-cols-1 gap-3 py-4 first:pt-0 sm:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)_auto] sm:items-end"
              >
                <div class="min-w-0">
                  <p class="text-xs font-medium text-muted-foreground">
                    {{ t('application.componentDetail.fields.key') }}
                  </p>
                  <p class="mt-1 break-all font-mono text-sm text-foreground">
                    {{ row.key }}
                  </p>
                </div>
                <div class="min-w-0">
                  <p class="text-xs font-medium text-muted-foreground">
                    {{ t('application.componentDetail.fields.value') }}
                  </p>
                  <p class="mt-1 break-all font-mono text-sm text-foreground">
                    {{ row.value }}
                  </p>
                </div>
              </div>
            </div>
          </div>

          <div class="border-t border-border p-5 sm:p-6">
            <div class="flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold text-foreground">
                {{ t('application.componentDetail.sections.mounts') }}
              </h3>
              <button
                v-if="editing"
                class="app-icon-button"
                :aria-label="t('application.componentDetail.actions.addRow')"
                :title="t('application.componentDetail.actions.addRow')"
                @click="addMount"
              >
                <Plus class="size-4" />
              </button>
            </div>
            <p
              v-if="form.mounts.length === 0 && !editing"
              class="mt-3 text-sm text-muted-foreground"
            >
              {{ t('common.notSet') }}
            </p>
            <div v-else class="mt-4 divide-y divide-border">
              <div
                v-for="(row, index) in form.mounts"
                :key="`mount-${index}`"
                class="py-4 first:pt-0"
              >
                <div v-if="editing">
                  <div
                    class="hidden grid-cols-[180px_minmax(0,1fr)_minmax(0,1fr)_auto_auto] gap-2 px-3 text-xs font-medium text-muted-foreground sm:grid"
                  >
                    <span>{{ t('application.componentDetail.fields.sourceType') }}</span>
                    <span>{{ t('application.componentDetail.fields.source') }}</span>
                    <span>{{ t('application.componentDetail.fields.target') }}</span>
                    <span class="w-20">{{ t('application.componentDetail.fields.readOnly') }}</span>
                    <span class="w-9"></span>
                  </div>
                  <div
                    class="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-[180px_minmax(0,1fr)_minmax(0,1fr)_auto_auto] sm:items-center"
                  >
                    <RawValueSelect
                      v-model="row.source_type"
                      :placeholder="t('application.detail.placeholders.mountSourceType')"
                      :values="mountSourceTypes"
                    />
                    <input
                      v-model="row.source"
                      class="app-input"
                      :placeholder="t('application.componentDetail.fields.source')"
                      type="text"
                    />
                    <input
                      v-model="row.target"
                      class="app-input"
                      :placeholder="t('application.componentDetail.fields.target')"
                      type="text"
                    />
                    <label class="flex h-10 items-center gap-2 px-3 text-sm text-foreground">
                      <input v-model="row.read_only" class="app-checkbox" type="checkbox" />
                      {{ t('application.componentDetail.fields.readOnly') }}
                    </label>
                    <button
                      class="app-icon-button justify-self-end"
                      :aria-label="t('common.delete')"
                      :title="t('common.delete')"
                      @click="form.mounts.splice(index, 1)"
                    >
                      <Trash2 class="size-4" />
                    </button>
                  </div>
                </div>
                <dl
                  v-else
                  class="grid grid-cols-1 gap-x-6 gap-y-3 text-sm sm:grid-cols-2 lg:grid-cols-4"
                >
                  <div>
                    <dt class="text-muted-foreground">
                      {{ t('application.componentDetail.fields.sourceType') }}
                    </dt>
                    <dd class="mt-1 text-foreground">{{ mountTypeLabel(row.source_type) }}</dd>
                  </div>
                  <div class="min-w-0">
                    <dt class="text-muted-foreground">
                      {{ t('application.componentDetail.fields.source') }}
                    </dt>
                    <dd class="mt-1 break-all font-mono text-foreground">{{ row.source }}</dd>
                  </div>
                  <div class="min-w-0">
                    <dt class="text-muted-foreground">
                      {{ t('application.componentDetail.fields.target') }}
                    </dt>
                    <dd class="mt-1 break-all font-mono text-foreground">{{ row.target }}</dd>
                  </div>
                  <div>
                    <dt class="text-muted-foreground">
                      {{ t('application.componentDetail.fields.readOnly') }}
                    </dt>
                    <dd class="mt-1 text-foreground">
                      {{ row.read_only ? t('common.yes') : t('common.no') }}
                    </dd>
                  </div>
                </dl>
                <div
                  v-if="
                    editing &&
                    (row.source_type === 'file' || row.content !== '' || row.content_mode !== '')
                  "
                  class="mt-3 grid grid-cols-1 gap-3 border-t border-border pt-3 sm:grid-cols-[1fr_180px]"
                >
                  <div>
                    <label class="app-field-label mb-1 block">
                      {{ t('application.componentDetail.fields.content') }}
                    </label>
                    <textarea v-model="row.content" class="app-textarea" />
                  </div>
                  <div>
                    <label class="app-field-label mb-1 block">
                      {{ t('application.componentDetail.fields.contentMode') }}
                    </label>
                    <RawValueSelect
                      v-model="row.content_mode"
                      :placeholder="t('application.detail.placeholders.mountContentMode')"
                      :values="contentModeValues"
                    />
                  </div>
                </div>
                <dl
                  v-else-if="!editing && (row.content !== '' || row.content_mode !== '')"
                  class="mt-3 grid grid-cols-1 gap-3 border-t border-border pt-3 text-sm sm:grid-cols-2"
                >
                  <div>
                    <dt class="text-muted-foreground">
                      {{ t('application.componentDetail.fields.contentMode') }}
                    </dt>
                    <dd class="mt-1 text-foreground">
                      {{ row.content_mode || t('common.notSet') }}
                    </dd>
                  </div>
                  <div class="min-w-0">
                    <dt class="text-muted-foreground">
                      {{ t('application.componentDetail.fields.content') }}
                    </dt>
                    <dd
                      class="mt-1 max-h-24 overflow-auto whitespace-pre-wrap break-words font-mono text-xs text-foreground"
                    >
                      {{ row.content }}
                    </dd>
                  </div>
                </dl>
              </div>
            </div>
          </div>

          <div class="border-t border-border p-5 sm:p-6">
            <h3 class="text-sm font-semibold text-foreground">
              {{ t('application.componentDetail.sections.network') }}
            </h3>
            <div class="mt-4 grid gap-6 lg:grid-cols-2">
              <div>
                <div class="flex items-center justify-between gap-3">
                  <h4 class="text-sm font-medium text-foreground">
                    {{ t('application.componentDetail.fields.network') }}
                  </h4>
                  <button
                    v-if="editing"
                    class="app-icon-button"
                    :aria-label="t('application.componentDetail.actions.addRow')"
                    :title="t('application.componentDetail.actions.addRow')"
                    @click="form.networks.push({ value: '' })"
                  >
                    <Plus class="size-4" />
                  </button>
                </div>
                <p
                  v-if="form.networks.length === 0 && !editing"
                  class="mt-3 text-sm text-muted-foreground"
                >
                  {{ t('common.notSet') }}
                </p>
                <div v-else class="mt-3 space-y-2">
                  <div
                    v-for="(row, index) in form.networks"
                    :key="`network-${index}`"
                    class="flex items-center gap-2"
                  >
                    <input
                      v-if="editing"
                      v-model="row.value"
                      class="app-input min-w-0 flex-1"
                      type="text"
                    />
                    <span v-else class="rounded-md bg-muted px-2 py-1 text-sm text-foreground">
                      {{ row.value }}
                    </span>
                    <button
                      v-if="editing"
                      class="app-icon-button shrink-0"
                      :aria-label="t('common.delete')"
                      :title="t('common.delete')"
                      @click="form.networks.splice(index, 1)"
                    >
                      <Trash2 class="size-4" />
                    </button>
                  </div>
                </div>
              </div>
              <div>
                <div class="flex items-center justify-between gap-3">
                  <h4 class="text-sm font-medium text-foreground">
                    {{ t('application.componentDetail.fields.dependency') }}
                  </h4>
                  <button
                    v-if="editing"
                    class="app-icon-button"
                    :aria-label="t('application.componentDetail.actions.addRow')"
                    :title="t('application.componentDetail.actions.addRow')"
                    @click="form.dependencies.push({ name: '', condition: '' })"
                  >
                    <Plus class="size-4" />
                  </button>
                </div>
                <p
                  v-if="form.dependencies.length === 0 && !editing"
                  class="mt-3 text-sm text-muted-foreground"
                >
                  {{ t('common.notSet') }}
                </p>
                <div v-if="editing" class="mt-3">
                  <div
                    class="hidden grid-cols-[1fr_1fr_auto] gap-2 px-3 text-xs font-medium text-muted-foreground sm:grid"
                  >
                    <span>{{ t('application.componentDetail.fields.dependency') }}</span>
                    <span>{{ t('application.componentDetail.fields.condition') }}</span>
                    <span class="w-9"></span>
                  </div>
                  <div class="mt-2 space-y-2">
                    <div
                      v-for="(row, index) in form.dependencies"
                      :key="`dependency-${index}`"
                      class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1fr_auto] sm:items-center"
                    >
                      <RawValueSelect
                        v-model="row.name"
                        :placeholder="t('application.componentDetail.fields.dependency')"
                        :values="dependencyNames"
                      />
                      <RawValueSelect
                        v-model="row.condition"
                        :placeholder="t('application.componentDetail.fields.condition')"
                        :values="dependencyConditions"
                      />
                      <button
                        class="app-icon-button justify-self-end"
                        :aria-label="t('common.delete')"
                        :title="t('common.delete')"
                        @click="form.dependencies.splice(index, 1)"
                      >
                        <Trash2 class="size-4" />
                      </button>
                    </div>
                  </div>
                </div>
                <div v-else class="mt-3 space-y-3">
                  <div
                    v-for="(row, index) in form.dependencies"
                    :key="`dependency-${index}`"
                    class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1fr_auto] sm:items-end"
                  >
                    <p class="text-sm text-foreground">{{ row.name }}</p>
                    <p class="text-sm text-muted-foreground">{{ row.condition }}</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section v-show="activeTab === 'advanced'" class="app-surface" role="tabpanel">
          <div class="app-section-header">
            <h2 class="text-base font-semibold text-foreground">
              {{ t('application.componentDetail.tabs.advanced') }}
            </h2>
          </div>

          <div class="p-5 sm:p-6">
            <div class="flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold text-foreground">
                {{ t('application.componentDetail.sections.tmpfs') }}
              </h3>
              <button
                v-if="editing"
                class="app-icon-button"
                :aria-label="t('application.componentDetail.actions.addRow')"
                :title="t('application.componentDetail.actions.addRow')"
                @click="form.tmpfs.push({ target: '', size_bytes: '', mode: '' })"
              >
                <Plus class="size-4" />
              </button>
            </div>
            <p
              v-if="form.tmpfs.length === 0 && !editing"
              class="mt-3 text-sm text-muted-foreground"
            >
              {{ t('common.notSet') }}
            </p>
            <div v-if="editing" class="mt-4">
              <div
                class="hidden grid-cols-[1fr_1fr_1fr_auto] gap-2 px-3 text-xs font-medium text-muted-foreground sm:grid"
              >
                <span>{{ t('application.componentDetail.fields.target') }}</span>
                <span>{{ t('application.componentDetail.fields.sizeBytes') }}</span>
                <span>{{ t('application.componentDetail.fields.mode') }}</span>
                <span class="w-9"></span>
              </div>
              <div class="mt-2 space-y-2">
                <div
                  v-for="(row, index) in form.tmpfs"
                  :key="`tmpfs-${index}`"
                  class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1fr_1fr_auto] sm:items-center"
                >
                  <input
                    v-model="row.target"
                    class="app-input"
                    :placeholder="t('application.componentDetail.fields.target')"
                    type="text"
                  />
                  <input
                    v-model="row.size_bytes"
                    class="app-input"
                    :placeholder="t('application.componentDetail.fields.sizeBytes')"
                    inputmode="numeric"
                    type="text"
                  />
                  <input
                    v-model="row.mode"
                    class="app-input"
                    :placeholder="t('application.componentDetail.fields.mode')"
                    type="text"
                  />
                  <button
                    class="app-icon-button justify-self-end"
                    :aria-label="t('common.delete')"
                    :title="t('common.delete')"
                    @click="form.tmpfs.splice(index, 1)"
                  >
                    <Trash2 class="size-4" />
                  </button>
                </div>
              </div>
            </div>
            <div v-else class="mt-4 divide-y divide-border">
              <div
                v-for="(row, index) in form.tmpfs"
                :key="`tmpfs-${index}`"
                class="grid grid-cols-1 gap-3 py-4 first:pt-0 sm:grid-cols-[1fr_1fr_1fr_auto] sm:items-end"
              >
                <div>
                  <p class="text-xs font-medium text-muted-foreground">
                    {{ t('application.componentDetail.fields.target') }}
                  </p>
                  <p class="mt-1 font-mono text-sm text-foreground">{{ row.target }}</p>
                </div>
                <div>
                  <p class="text-xs font-medium text-muted-foreground">
                    {{ t('application.componentDetail.fields.sizeBytes') }}
                  </p>
                  <p class="mt-1 text-sm text-foreground">{{ row.size_bytes }}</p>
                </div>
                <div>
                  <p class="text-xs font-medium text-muted-foreground">
                    {{ t('application.componentDetail.fields.mode') }}
                  </p>
                  <p class="mt-1 font-mono text-sm text-foreground">{{ row.mode }}</p>
                </div>
              </div>
            </div>
          </div>

          <div class="border-t border-border p-5 sm:p-6">
            <div class="flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold text-foreground">
                {{ t('application.componentDetail.sections.ulimits') }}
              </h3>
              <button
                v-if="editing"
                class="app-icon-button"
                :aria-label="t('application.componentDetail.actions.addRow')"
                :title="t('application.componentDetail.actions.addRow')"
                @click="form.ulimits.push({ name: '', soft: '', hard: '' })"
              >
                <Plus class="size-4" />
              </button>
            </div>
            <p
              v-if="form.ulimits.length === 0 && !editing"
              class="mt-3 text-sm text-muted-foreground"
            >
              {{ t('common.notSet') }}
            </p>
            <div v-if="editing" class="mt-4">
              <div
                class="hidden grid-cols-[1fr_1fr_1fr_auto] gap-2 px-3 text-xs font-medium text-muted-foreground sm:grid"
              >
                <span>{{ t('application.componentDetail.fields.name') }}</span>
                <span>{{ t('application.componentDetail.fields.soft') }}</span>
                <span>{{ t('application.componentDetail.fields.hard') }}</span>
                <span class="w-9"></span>
              </div>
              <div class="mt-2 space-y-2">
                <div
                  v-for="(row, index) in form.ulimits"
                  :key="`ulimit-${index}`"
                  class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1fr_1fr_auto] sm:items-center"
                >
                  <RawValueSelect
                    v-model="row.name"
                    :placeholder="t('application.componentDetail.fields.name')"
                    :values="ulimitNames"
                  />
                  <input
                    v-model="row.soft"
                    class="app-input"
                    :placeholder="t('application.componentDetail.fields.soft')"
                    inputmode="numeric"
                    type="text"
                  />
                  <input
                    v-model="row.hard"
                    class="app-input"
                    :placeholder="t('application.componentDetail.fields.hard')"
                    inputmode="numeric"
                    type="text"
                  />
                  <button
                    class="app-icon-button justify-self-end"
                    :aria-label="t('common.delete')"
                    :title="t('common.delete')"
                    @click="form.ulimits.splice(index, 1)"
                  >
                    <Trash2 class="size-4" />
                  </button>
                </div>
              </div>
            </div>
            <div v-else class="mt-4 divide-y divide-border">
              <div
                v-for="(row, index) in form.ulimits"
                :key="`ulimit-${index}`"
                class="grid grid-cols-1 gap-3 py-4 first:pt-0 sm:grid-cols-[1fr_1fr_1fr_auto] sm:items-end"
              >
                <div>
                  <p class="text-xs font-medium text-muted-foreground">
                    {{ t('application.componentDetail.fields.name') }}
                  </p>
                  <p class="mt-1 text-sm text-foreground">{{ row.name }}</p>
                </div>
                <div>
                  <p class="text-xs font-medium text-muted-foreground">
                    {{ t('application.componentDetail.fields.soft') }}
                  </p>
                  <p class="mt-1 text-sm text-foreground">{{ row.soft }}</p>
                </div>
                <div>
                  <p class="text-xs font-medium text-muted-foreground">
                    {{ t('application.componentDetail.fields.hard') }}
                  </p>
                  <p class="mt-1 text-sm text-foreground">{{ row.hard }}</p>
                </div>
              </div>
            </div>
          </div>
        </section>

        <p v-if="formError" class="app-field-error text-sm">{{ formError }}</p>
      </template>
    </template>

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
  import { ArrowLeft, Pencil, Plus, Save, Trash2, X } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { VersionComponentResp, VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import { versionStatusTone } from '@/utils/status';
  import {
    componentFormFromResponse,
    componentRequestFromForm,
    emptyComponentForm,
    type ComponentFormError,
  } from './componentForm';

  type ComponentTab = 'overview' | 'runtime' | 'connectivity' | 'advanced';

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
  const editing = ref(isNew);
  const activeTab = ref<ComponentTab>('overview');
  const deleteDialogOpen = ref(false);
  const formError = ref('');

  const canEdit = computed(() => version.value?.status === 'unpublished');
  const tabs = computed(() => [
    { id: 'overview' as const, label: t('application.componentDetail.tabs.overview') },
    { id: 'runtime' as const, label: t('application.componentDetail.tabs.runtime') },
    { id: 'connectivity' as const, label: t('application.componentDetail.tabs.connectivity') },
    { id: 'advanced' as const, label: t('application.componentDetail.tabs.advanced') },
  ]);
  const commandPreview = computed(() => form.command.map((item) => item.value).join(' '));
  const pullPolicyValues = ['always', 'missing', 'never'];
  const restartPolicyValues = ['no', 'unless-stopped'];
  const mountSourceTypes = ['directory', 'file', 'named_volume', 'special'];
  const contentModeValues = ['seed', 'sync'];
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

  function assignForm(source: ReturnType<typeof emptyComponentForm>) {
    Object.assign(form, source);
  }

  function mountTypeLabel(value: string): string {
    const labels: Record<string, string> = {
      directory: t('application.componentDetail.mountTypes.directory'),
      file: t('application.componentDetail.mountTypes.file'),
      named_volume: t('application.componentDetail.mountTypes.namedVolume'),
      special: t('application.componentDetail.mountTypes.special'),
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
          editing.value = loadedVersion.status === 'unpublished';
          return;
        }
        const loadedComponent = await applicationApi.getVersionComponent(versionId, componentId);
        component.value = loadedComponent;
        assignForm(componentFormFromResponse(loadedComponent));
        editing.value = false;
      });
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('application.toast.loadVersionsFailed')
      );
      router.replace(`/version/${versionId}`);
    }
  }

  function startEditing() {
    if (!component.value) {
      return;
    }
    assignForm(componentFormFromResponse(component.value));
    formError.value = '';
    editing.value = true;
  }

  function cancelEditing() {
    if (isNew) {
      goBack();
      return;
    }
    if (component.value) {
      assignForm(componentFormFromResponse(component.value));
    }
    formError.value = '';
    editing.value = false;
  }

  function addMount() {
    form.mounts.push({
      source_type: '',
      source: '',
      target: '',
      read_only: false,
      content: '',
      content_mode: '',
    });
  }

  function tabForError(error: ComponentFormError): ComponentTab {
    if (error === 'nameImage' || error === 'componentName') {
      return 'overview';
    }
    if (error === 'command' || error === 'args' || error === 'healthcheck') {
      return 'runtime';
    }
    if (
      error === 'ports' ||
      error === 'env' ||
      error === 'mounts' ||
      error === 'networks' ||
      error === 'dependencies'
    ) {
      return 'connectivity';
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
    const section = t(`application.componentDetail.sections.${error}`);
    return t('application.componentDetail.validation.rowIncomplete', { section });
  }

  async function save() {
    const result = componentRequestFromForm(form);
    if (!result.valid) {
      activeTab.value = tabForError(result.error);
      formError.value = messageFor(result.error);
      return;
    }
    formError.value = '';
    try {
      await executeOperation(async () => {
        if (isNew) {
          const created = await applicationApi.createVersionComponent(versionId, result.value);
          toast.success(t('application.toast.updateSuccess'));
          await router.replace(`/version/${versionId}/component/${created.id}`);
          return;
        }
        const updated = await applicationApi.updateVersionComponent(
          versionId,
          componentId,
          result.value
        );
        component.value = updated;
        assignForm(componentFormFromResponse(updated));
        editing.value = false;
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
