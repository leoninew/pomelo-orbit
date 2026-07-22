<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between gap-4">
      <div class="min-w-0">
        <button class="app-link mb-2 inline-flex items-center gap-1 text-sm" @click="router.push('/cd/gateways')">
          <ArrowLeft class="size-4" />
          {{ t('gateway.backToList') }}
        </button>
        <h1 class="truncate text-xl font-semibold text-foreground">
          {{ gateway?.name || t('gateway.detailTitle') }}
        </h1>
        <p v-if="gateway" class="mt-1 font-mono text-sm text-muted-foreground">
          {{ gateway.code }}
        </p>
      </div>
      <div v-if="gateway" class="flex shrink-0 items-center gap-3">
        <button class="app-button px-4" @click="goWorkload">
          {{ t('gateway.openWorkload') }}
        </button>
      </div>
    </div>

    <div class="app-surface">
      <AppSpinner v-if="status === 'loading'" class="py-16" />
      <div v-else-if="gateway" class="space-y-6 p-1">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div class="space-y-1.5">
            <label class="app-field-label block">{{ t('gateway.fields.name') }}</label>
            <input v-model="form.name" type="text" class="app-input" />
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">{{ t('gateway.fields.code') }}</label>
            <input :value="gateway.code" type="text" class="app-input" disabled />
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">{{ t('gateway.fields.restApiUrl') }}</label>
            <input v-model="form.rest_api_url" type="text" class="app-input" />
            <p class="app-field-hint">{{ t('gateway.hints.restApiUrl') }}</p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label block">{{ t('gateway.fields.baseDomain') }}</label>
            <input v-model="form.base_domain" type="text" class="app-input" />
            <p class="app-field-hint">{{ t('gateway.hints.baseDomain') }}</p>
          </div>
          <div class="space-y-1.5 md:col-span-2">
            <label class="app-field-label block">{{ t('gateway.fields.image') }}</label>
            <input
              v-model="form.image"
              type="text"
              class="app-input"
              :placeholder="t('gateway.placeholders.image')"
            />
            <p class="app-field-hint">{{ t('gateway.hints.image') }}</p>
          </div>
        </div>
        <p class="text-sm text-muted-foreground">{{ t('gateway.hints.compileOnSave') }}</p>
        <div class="flex justify-end gap-3">
          <button class="app-button px-4" :disabled="operating" @click="goWorkload">
            {{ t('gateway.openWorkload') }}
          </button>
          <button class="app-button-primary px-5" :disabled="operating" @click="handleSave">
            {{ t('common.save') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft } from 'lucide-vue-next';
  import { onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { gatewayApi } from '@/api/cd/gateway';
  import AppSpinner from '@/components/AppSpinner.vue';
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
  });

  const gatewayId = () => String(route.params.id || '');

  async function loadGateway() {
    const id = gatewayId();
    if (!id) {
      return;
    }
    try {
      await execute(async () => {
        const data = await gatewayApi.get(id);
        gateway.value = data;
        Object.assign(form, {
          name: data.name,
          rest_api_url: data.rest_api_url || '',
          base_domain: data.base_domain || '',
          image: data.image || '',
        });
      });
    } catch {
      toast.error(t('gateway.toast.loadDetailFailed'));
    }
  }

  function goWorkload() {
    if (!gateway.value) {
      return;
    }
    router.push(`/cd/applications/${gateway.value.id}`);
  }

  async function handleSave() {
    const id = gatewayId();
    if (!id) {
      return;
    }
    if (!form.name.trim() || !form.rest_api_url.trim() || !form.base_domain.trim()) {
      toast.error(t('gateway.validation.requiredFields'));
      return;
    }
    try {
      await executeOp(async () => {
        const data = await gatewayApi.update(id, {
          name: form.name.trim(),
          rest_api_url: form.rest_api_url.trim(),
          base_domain: form.base_domain.trim(),
          image: form.image.trim() || undefined,
        });
        gateway.value = data;
        toast.success(t('gateway.toast.saveCompiled'));
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
