<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="min-w-0">
        <h1 class="truncate text-base font-semibold text-foreground">
          {{ t('gateway.dialog.edit') }}
        </h1>
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

    <AppLoadingState v-if="status === 'loading'" size="section" />

    <div v-else-if="gateway" class="app-surface">
      <div class="app-section-header">
        <h2 class="text-base font-semibold text-foreground">{{ t('gateway.sections.config') }}</h2>
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
              :aria-invalid="errors.name ? 'true' : undefined"
              @input="errors.name = ''"
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
              :aria-invalid="errors.rest_api_url ? 'true' : undefined"
              @input="errors.rest_api_url = ''"
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
              :aria-invalid="errors.base_domain ? 'true' : undefined"
              @input="errors.base_domain = ''"
            />
            <p v-if="errors.base_domain" class="app-field-error">{{ errors.base_domain }}</p>
            <p v-else class="app-field-hint">{{ t('gateway.hints.baseDomain') }}</p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">{{ t('gateway.fields.defaultEntrypoint') }}</label>
            <RawValueSelect
              v-model="form.default_entrypoint"
              :values="entrypointValues"
              :placeholder="t('gateway.placeholders.defaultEntrypoint')"
            />
            <p class="app-field-hint">{{ t('gateway.hints.defaultEntrypoint') }}</p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">{{ t('gateway.fields.tlsMode') }}</label>
            <RawValueSelect
              v-model="form.tls_mode"
              :values="tlsModeValues"
              :placeholder="t('gateway.placeholders.tlsMode')"
            />
          </div>
        </div>
        <p class="text-sm text-muted-foreground">{{ t('gateway.hints.compileOnSave') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Save } from 'lucide-vue-next';
  import { onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { gatewayApi } from '@/api/gateway/gateway';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';

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
    default_entrypoint: 'web',
    tls_mode: 'none',
  });
  const errors = reactive({
    name: '',
    rest_api_url: '',
    base_domain: '',
  });

  const entrypointValues = ['web', 'websecure'];
  const tlsModeValues = ['none', 'letsencrypt', 'tls'];

  const gatewayId = () => String(route.params.id || '');

  function fillForm(data: GatewayResp) {
    Object.assign(form, {
      name: data.name,
      rest_api_url: data.rest_api_url || '',
      base_domain: data.base_domain || '',
      default_entrypoint: data.default_entrypoint,
      tls_mode: data.tls_mode,
    });
    errors.name = '';
    errors.rest_api_url = '';
    errors.base_domain = '';
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
    return !errors.name && !errors.rest_api_url && !errors.base_domain;
  }

  function goBack() {
    const id = gatewayId();
    if (id) {
      router.push(`/gateway/${id}`);
      return;
    }
    router.push('/gateways');
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
          default_entrypoint: form.default_entrypoint,
          tls_mode: form.tls_mode,
        });
        gateway.value = data;
        fillForm(data);
        toast.success(t('gateway.toast.saveCompiled'));
        await router.push(`/gateway/${id}`);
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
