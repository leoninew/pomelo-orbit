<template>
  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <label class="app-field-label block" :for="id('target-type')">
        {{ t('project.environment.targetType') }}
        <span class="text-destructive">*</span>
      </label>
      <SelectControl
        :id="id('target-type')"
        :model-value="form.targetType"
        :options="targetTypeOptions"
        :disabled="disabled"
        @update:model-value="updateTargetType"
      />
    </div>
    <div v-if="showState" class="space-y-1.5">
      <label class="app-field-label block" :for="id('state')">
        {{ t('project.environment.state') }}
        <span class="text-destructive">*</span>
      </label>
      <SelectControl
        :id="id('state')"
        :model-value="form.state"
        :options="stateOptions"
        :disabled="disabled"
        @update:model-value="form.state = String($event)"
      />
    </div>
    <template v-if="form.targetType === 'local'">
      <div class="space-y-1.5 sm:col-span-2">
        <label class="app-field-label block" :for="id('workspace-root')">
          {{ t('project.environment.workspaceRoot') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          :id="id('workspace-root')"
          v-model="form.workspaceRoot"
          class="app-input"
          :class="errors.workspaceRoot ? 'app-input-error' : ''"
          :placeholder="workspacePlaceholder"
          :disabled="disabled"
          :aria-invalid="errors.workspaceRoot ? 'true' : undefined"
          :aria-describedby="errors.workspaceRoot ? id('workspace-root-error') : undefined"
          @input="errors.workspaceRoot = ''"
        />
        <p
          v-if="errors.workspaceRoot"
          :id="id('workspace-root-error')"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ errors.workspaceRoot }}
        </p>
      </div>
    </template>
    <template v-else>
      <div class="space-y-1.5">
        <label class="app-field-label block" :for="id('platform')">
          {{ t('project.environment.platform') }}
          <span class="text-destructive">*</span>
        </label>
        <SelectControl
          :id="id('platform')"
          :model-value="form.platform"
          :options="platformOptions"
          :disabled="disabled"
          @update:model-value="updatePlatform"
        />
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block" :for="id('host')">
          {{ t('project.environment.host') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          :id="id('host')"
          v-model="form.host"
          class="app-input"
          :class="errors.host ? 'app-input-error' : ''"
          :placeholder="hostPlaceholder"
          :disabled="disabled"
          :aria-invalid="errors.host ? 'true' : undefined"
          :aria-describedby="errors.host ? id('host-error') : undefined"
          @input="errors.host = ''"
        />
        <p v-if="errors.host" :id="id('host-error')" class="app-field-error text-xs" role="alert">
          {{ errors.host }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block" :for="id('port')">
          {{ t('project.environment.port') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          :id="id('port')"
          v-model.number="form.port"
          type="number"
          min="1"
          max="65535"
          class="app-input"
          :class="errors.port ? 'app-input-error' : ''"
          :disabled="disabled"
          :aria-invalid="errors.port ? 'true' : undefined"
          :aria-describedby="errors.port ? id('port-error') : undefined"
          @input="errors.port = ''"
        />
        <p v-if="errors.port" :id="id('port-error')" class="app-field-error text-xs" role="alert">
          {{ errors.port }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block" :for="id('username')">
          {{ t('project.environment.username') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          :id="id('username')"
          v-model="form.username"
          class="app-input"
          :class="errors.username ? 'app-input-error' : ''"
          :placeholder="usernamePlaceholder"
          :disabled="disabled"
          :aria-invalid="errors.username ? 'true' : undefined"
          :aria-describedby="errors.username ? id('username-error') : undefined"
          @input="errors.username = ''"
        />
        <p
          v-if="errors.username"
          :id="id('username-error')"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ errors.username }}
        </p>
      </div>
      <div class="space-y-1.5 sm:col-span-2">
        <label class="app-field-label block" :for="id('workspace-root')">
          {{ t('project.environment.workspaceRoot') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          :id="id('workspace-root')"
          v-model="form.workspaceRoot"
          class="app-input"
          :class="errors.workspaceRoot ? 'app-input-error' : ''"
          :placeholder="workspacePlaceholder"
          :disabled="disabled"
          :aria-invalid="errors.workspaceRoot ? 'true' : undefined"
          :aria-describedby="errors.workspaceRoot ? id('workspace-root-error') : undefined"
          @input="errors.workspaceRoot = ''"
        />
        <p
          v-if="errors.workspaceRoot"
          :id="id('workspace-root-error')"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ errors.workspaceRoot }}
        </p>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import SelectControl from '@/components/SelectControl.vue';
  import type { EnvironmentLocalTargetResp } from '@/gen/proto/orbit/v1/environment/environment';
  import {
    applyEnvironmentPlatform,
    applyEnvironmentTargetType,
    emptyEnvironmentFormErrors,
    environmentWorkspacePlaceholder,
    type EnvironmentForm,
    type EnvironmentFormErrors,
    type EnvironmentPlatform,
    type EnvironmentTargetType,
  } from '@/views/environment/environmentForm';

  const form = defineModel<EnvironmentForm>({ required: true });
  const errors = defineModel<EnvironmentFormErrors>('errors', { required: true });

  const props = withDefaults(
    defineProps<{
      disabled?: boolean;
      showState?: boolean;
      localWorkspaceRoot?: string;
      localDisplay?: EnvironmentLocalTargetResp;
      idPrefix?: string;
    }>(),
    {
      disabled: false,
      showState: false,
      localWorkspaceRoot: '',
      idPrefix: 'environment',
    }
  );

  const { t } = useI18n();
  const targetTypeOptions = computed(() => [
    { value: 'local', label: t('project.environment.targetTypes.local') },
    { value: 'ssh', label: t('project.environment.targetTypes.ssh') },
  ]);
  const stateOptions = computed(() => [
    { value: 'active', label: t('project.environment.states.active') },
    { value: 'disabled', label: t('project.environment.states.disabled') },
  ]);
  const platformOptions = computed(() => [
    { value: 'linux', label: t('project.environment.platforms.linux') },
    { value: 'windows', label: t('project.environment.platforms.windows') },
  ]);
  const workspacePlaceholder = computed(() =>
    environmentWorkspacePlaceholder(form.value, props.localWorkspaceRoot)
  );
  const hostPlaceholder = computed(() => t('project.environment.hostPlaceholder'));
  const usernamePlaceholder = computed(() => t('project.environment.usernamePlaceholder'));

  function id(suffix: string) {
    return `${props.idPrefix}-${suffix}`;
  }

  function clearErrors() {
    Object.assign(errors.value, emptyEnvironmentFormErrors());
  }

  function updateTargetType(value: string | number) {
    Object.assign(
      form.value,
      applyEnvironmentTargetType(
        form.value,
        String(value) as EnvironmentTargetType,
        props.localDisplay,
        props.localWorkspaceRoot
      )
    );
    clearErrors();
  }

  function updatePlatform(value: string | number) {
    const platform = String(value) as EnvironmentPlatform;
    if (platform === form.value.platform) return;
    Object.assign(form.value, applyEnvironmentPlatform(form.value, platform, props.localDisplay));
    errors.value.workspaceRoot = '';
    errors.value.host = '';
    errors.value.username = '';
  }
</script>
