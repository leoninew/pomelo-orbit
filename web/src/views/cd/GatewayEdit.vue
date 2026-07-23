<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="min-w-0">
        <h1 class="truncate text-xl font-semibold text-foreground">
          {{ t('gateway.dialog.edit') }}
        </h1>
        <p v-if="gateway" class="mt-1 text-sm text-muted-foreground">
          {{ gateway.code }}
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button class="app-button-primary h-9 px-3" :disabled="operating" @click="handleSave">
          <Save class="size-4" />
          {{ t('common.save') }}
        </button>
        <button class="app-button h-9 px-4" @click="goBack">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppSpinner v-if="status === 'loading'" class="py-12" />

    <div v-else-if="gateway" class="app-surface">
      <div class="app-section-header">
        <h2 class="font-semibold text-foreground">{{ t('gateway.sections.config') }}</h2>
      </div>
      <div class="space-y-4 px-5 py-4">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div class="space-y-1.5">
            <label class="app-field-label block">
              {{ t('gateway.fields.name') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="form.name"
              type="text"
              class="app-input"
              :class="errors.name ? 'app-input-error' : ''"
              :placeholder="t('gateway.placeholders.name')"
            />
            <p v-if="errors.name" class="app-field-error">{{ errors.name }}</p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">{{ t('gateway.fields.code') }}</label>
            <input :value="gateway.code" type="text" class="app-input" disabled />
            <p class="app-field-hint">{{ t('gateway.hints.code') }}</p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">
              {{ t('gateway.fields.restApiUrl') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="form.rest_api_url"
              type="text"
              class="app-input"
              :class="errors.rest_api_url ? 'app-input-error' : ''"
              :placeholder="t('gateway.placeholders.restApiUrl')"
            />
            <p v-if="errors.rest_api_url" class="app-field-error">{{ errors.rest_api_url }}</p>
            <p v-else class="app-field-hint">{{ t('gateway.hints.restApiUrl') }}</p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">
              {{ t('gateway.fields.baseDomain') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="form.base_domain"
              type="text"
              class="app-input"
              :class="errors.base_domain ? 'app-input-error' : ''"
              :placeholder="t('gateway.placeholders.baseDomain')"
            />
            <p v-if="errors.base_domain" class="app-field-error">{{ errors.base_domain }}</p>
            <p v-else class="app-field-hint">{{ t('gateway.hints.baseDomain') }}</p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">{{ t('gateway.fields.defaultEntrypoint') }}</label>
            <SelectControl
              v-model="form.default_entrypoint"
              :options="entrypointOptions"
              :placeholder="t('gateway.placeholders.defaultEntrypoint')"
            />
            <p class="app-field-hint">{{ t('gateway.hints.defaultEntrypoint') }}</p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">{{ t('gateway.fields.tlsMode') }}</label>
            <SelectControl
              v-model="form.tls_mode"
              :options="tlsModeOptions"
              :placeholder="t('gateway.placeholders.tlsMode')"
            />
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">{{ t('gateway.fields.imagePullPolicy') }}</label>
            <SelectControl
              v-model="form.image_pull_policy"
              :options="imagePullPolicyOptions"
              :placeholder="t('application.imagePullPolicyPlaceholder')"
            />
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">
              {{ t('gateway.fields.image') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              v-model="form.image"
              type="text"
              class="app-input"
              :class="errors.image ? 'app-input-error' : ''"
              :placeholder="t('gateway.placeholders.image')"
            />
            <p v-if="errors.image" class="app-field-error">{{ errors.image }}</p>
            <p v-else class="app-field-hint">{{ t('gateway.hints.image') }}</p>
          </div>
        </div>
        <p class="text-sm text-muted-foreground">{{ t('gateway.hints.compileOnSave') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Save } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { gatewayApi } from '@/api/cd/gateway';
  import AppSpinner from '@/components/AppSpinner.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway';

  const toast = useToast();
  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const { status, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const gateway = ref<GatewayResp | null>(null);
  const form = reactive({
    name: '',
    rest_api_url: '',
    base_domain: '',
    image: '',
    image_pull_policy: 'missing',
    default_entrypoint: 'web',
    tls_mode: 'none',
  });
  const errors = reactive({
    name: '',
    rest_api_url: '',
    base_domain: '',
    image: '',
  });

  const imagePullPolicyOptions = computed(() => [
    { value: 'missing', label: t('application.imagePullPolicyOptions.missing') },
    { value: 'always', label: t('application.imagePullPolicyOptions.always') },
    { value: 'never', label: t('application.imagePullPolicyOptions.never') },
  ]);
  const entrypointOptions = computed(() => [
    { value: 'web', label: 'web' },
    { value: 'websecure', label: 'websecure' },
  ]);
  const tlsModeOptions = computed(() => [
    { value: 'none', label: 'none' },
    { value: 'letsencrypt', label: 'letsencrypt' },
    { value: 'tls', label: 'tls' },
  ]);

  const gatewayId = () => String(route.params.id || '');

  function fillForm(data: GatewayResp) {
    Object.assign(form, {
      name: data.name,
      rest_api_url: data.rest_api_url || '',
      base_domain: data.base_domain || '',
      image: data.image || '',
      image_pull_policy: data.image_pull_policy || 'missing',
      default_entrypoint: data.default_entrypoint || 'web',
      tls_mode: data.tls_mode || 'none',
    });
    errors.name = '';
    errors.rest_api_url = '';
    errors.base_domain = '';
    errors.image = '';
  }

  async function loadGateway() {
    const id = gatewayId();
    if (!id) {
      return;
    }
    try {
      await execute(async () => {
        const data = await gatewayApi.get(id);
        gateway.value = data;
        fillForm(data);
      });
    } catch {
      toast.error(t('gateway.toast.loadDetailFailed'));
    }
  }

  function validateForm() {
    errors.name = form.name.trim() ? '' : t('gateway.validation.nameRequired');
    errors.rest_api_url = form.rest_api_url.trim()
      ? ''
      : t('gateway.validation.restApiUrlRequired');
    errors.base_domain = form.base_domain.trim() ? '' : t('gateway.validation.baseDomainRequired');
    errors.image = form.image.trim() ? '' : t('gateway.validation.imageRequired');
    return !errors.name && !errors.rest_api_url && !errors.base_domain && !errors.image;
  }

  function goBack() {
    const id = gatewayId();
    if (id) {
      router.push(`/cd/gateway/${id}`);
      return;
    }
    router.push('/cd/gateways');
  }

  async function handleSave() {
    const id = gatewayId();
    if (!id || !validateForm()) {
      return;
    }
    try {
      await executeOp(async () => {
        const data = await gatewayApi.update(id, {
          name: form.name.trim(),
          rest_api_url: form.rest_api_url.trim(),
          base_domain: form.base_domain.trim(),
          image: form.image.trim(),
          image_pull_policy: form.image_pull_policy,
          default_entrypoint: form.default_entrypoint,
          tls_mode: form.tls_mode,
        });
        gateway.value = data;
        fillForm(data);
        toast.success(t('gateway.toast.saveCompiled'));
        await router.push(`/cd/gateway/${id}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('gateway.toast.saveFailed'));
    }
  }

  watch(
    () => route.params.id,
    () => {
      loadGateway();
    }
  );

  onMounted(loadGateway);
</script>
