<template>
  <div class="grid gap-4 sm:grid-cols-2">
    <div class="space-y-1.5 sm:col-span-2">
      <label class="app-field-label block" :for="id('bootstrap-username')">
        {{ t('project.environment.bootstrapUsername') }}
        <span class="text-destructive">*</span>
      </label>
      <input
        :id="id('bootstrap-username')"
        v-model="form.username"
        class="app-input"
        :class="errors.username ? 'app-input-error' : ''"
        :disabled="disabled"
        :aria-invalid="errors.username ? 'true' : undefined"
        :aria-describedby="errors.username ? id('bootstrap-username-error') : undefined"
        @input="errors.username = ''"
      />
      <p
        v-if="errors.username"
        :id="id('bootstrap-username-error')"
        class="app-field-error text-xs"
        role="alert"
      >
        {{ errors.username }}
      </p>
    </div>
    <div class="space-y-1.5 sm:col-span-2">
      <label class="app-field-label block">{{ t('project.environment.authentication') }}</label>
      <div class="flex h-10 overflow-hidden rounded-md border border-border bg-background">
        <button
          type="button"
          class="flex-1 px-3 text-sm text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
          :class="form.authType === 'password' ? 'bg-primary/10 text-primary' : ''"
          :disabled="disabled"
          :aria-pressed="form.authType === 'password'"
          @click="selectAuth('password')"
        >
          {{ t('project.environment.authenticationModes.password') }}
        </button>
        <button
          type="button"
          class="flex-1 border-l border-border px-3 text-sm text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
          :class="form.authType === 'privateKey' ? 'bg-primary/10 text-primary' : ''"
          :disabled="disabled"
          :aria-pressed="form.authType === 'privateKey'"
          @click="selectAuth('privateKey')"
        >
          {{ t('project.environment.authenticationModes.privateKey') }}
        </button>
      </div>
    </div>
    <template v-if="form.authType === 'password'">
      <div class="space-y-1.5 sm:col-span-2">
        <label class="app-field-label block" :for="id('bootstrap-password')">
          {{ t('project.environment.bootstrapPassword') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          :id="id('bootstrap-password')"
          v-model="form.password"
          type="password"
          autocomplete="current-password"
          class="app-input"
          :class="errors.credential ? 'app-input-error' : ''"
          :disabled="disabled"
          :aria-invalid="errors.credential ? 'true' : undefined"
          :aria-describedby="errors.credential ? id('bootstrap-credential-error') : undefined"
          @input="errors.credential = ''"
        />
        <p
          v-if="errors.credential"
          :id="id('bootstrap-credential-error')"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ errors.credential }}
        </p>
      </div>
    </template>
    <template v-else>
      <div class="space-y-1.5 sm:col-span-2">
        <label class="app-field-label block" :for="id('bootstrap-private-key')">
          {{ t('project.environment.bootstrapPrivateKey') }}
          <span class="text-destructive">*</span>
        </label>
        <textarea
          :id="id('bootstrap-private-key')"
          v-model="form.privateKey"
          rows="8"
          class="app-textarea w-full font-mono text-xs"
          :class="errors.credential ? 'app-input-error' : ''"
          :disabled="disabled"
          :aria-invalid="errors.credential ? 'true' : undefined"
          :aria-describedby="errors.credential ? id('bootstrap-credential-error') : undefined"
          @input="errors.credential = ''"
        />
        <p
          v-if="errors.credential"
          :id="id('bootstrap-credential-error')"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ errors.credential }}
        </p>
      </div>
      <div class="space-y-1.5 sm:col-span-2">
        <label class="app-field-label block" :for="id('bootstrap-passphrase')">
          {{ t('project.environment.bootstrapPrivateKeyPassphrase') }}
        </label>
        <input
          :id="id('bootstrap-passphrase')"
          v-model="form.privateKeyPassphrase"
          type="password"
          autocomplete="off"
          class="app-input"
          :disabled="disabled"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n';
  import type {
    EnvironmentBootstrapAuthType,
    EnvironmentBootstrapForm,
    EnvironmentBootstrapFormErrors,
  } from '@/views/environment/environmentBootstrapForm';

  const form = defineModel<EnvironmentBootstrapForm>({ required: true });
  const errors = defineModel<EnvironmentBootstrapFormErrors>('errors', { required: true });

  const props = withDefaults(
    defineProps<{
      disabled?: boolean;
      idPrefix?: string;
    }>(),
    {
      disabled: false,
      idPrefix: 'environment',
    }
  );

  const { t } = useI18n();

  function id(suffix: string) {
    return `${props.idPrefix}-${suffix}`;
  }

  function selectAuth(authType: EnvironmentBootstrapAuthType) {
    if (form.value.authType === authType) return;
    form.value.authType = authType;
    form.value.password = '';
    form.value.privateKey = '';
    form.value.privateKeyPassphrase = '';
    errors.value.credential = '';
  }
</script>
