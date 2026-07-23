<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="min-w-0">
        <h1 class="truncate text-xl font-semibold text-foreground">
          {{ gateway?.name || t('gateway.detailTitle') }}
        </h1>
        <p v-if="gateway" class="mt-1 text-sm text-muted-foreground">
          {{ gateway.code }}
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="gateway"
          class="app-button-primary h-9 px-3"
          @click="router.push(`/cd/gateway/${gateway.id}/edit`)"
        >
          <Pencil class="size-4" />
          {{ t('common.edit') }}
        </button>
        <button v-if="gateway" class="app-button h-9 px-3" @click="goWorkload">
          <Layers class="size-4" />
          {{ t('gateway.openWorkload') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/cd/gateways')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppSpinner v-if="status === 'loading'" class="py-12" />

    <template v-else-if="gateway">
      <div class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">{{ t('gateway.sections.config') }}</h2>
        </div>
        <dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('gateway.fields.name') }}</dt>
            <dd class="text-foreground">{{ gateway.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('gateway.fields.code') }}</dt>
            <dd class="text-foreground">{{ gateway.code }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">
              {{ t('gateway.fields.restApiUrl') }}
            </dt>
            <dd class="min-w-0 break-all text-foreground">{{ gateway.rest_api_url }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">
              {{ t('gateway.fields.baseDomain') }}
            </dt>
            <dd class="text-foreground">{{ gateway.base_domain }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">
              {{ t('gateway.fields.defaultEntrypoint') }}
            </dt>
            <dd class="text-foreground">{{ gateway.default_entrypoint || '—' }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('gateway.fields.tlsMode') }}</dt>
            <dd class="text-foreground">{{ gateway.tls_mode || 'none' }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">
              {{ t('gateway.fields.imagePullPolicy') }}
            </dt>
            <dd class="text-foreground">{{ imagePullPolicyLabel(gateway.image_pull_policy) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('gateway.fields.image') }}</dt>
            <dd class="min-w-0 break-all text-foreground">{{ gateway.image || '—' }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.createdAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(gateway.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.updatedAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(gateway.updated_at) }}</dd>
          </div>
        </dl>
      </div>

      <div class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">{{ t('gateway.exposures.title') }}</h2>
        </div>
        <div class="px-5 py-4">
          <p class="mb-4 text-sm text-muted-foreground">{{ t('gateway.exposures.hint') }}</p>
          <div v-if="!(gateway.exposures || []).length" class="text-sm text-muted-foreground">
            {{ t('gateway.exposures.empty') }}
          </div>
          <div v-else class="overflow-x-auto">
            <table class="app-table-list min-w-[720px]">
              <thead>
                <tr>
                  <th>{{ t('gateway.exposures.app') }}</th>
                  <th>{{ t('gateway.exposures.component') }}</th>
                  <th>{{ t('gateway.exposures.protocol') }}</th>
                  <th>{{ t('gateway.exposures.access') }}</th>
                  <th>{{ t('gateway.exposures.listen') }}</th>
                  <th>{{ t('gateway.exposures.internalDns') }}</th>
                  <th>{{ t('gateway.exposures.clientHint') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, idx) in gateway.exposures" :key="idx">
                  <td class="text-foreground">{{ row.application_code }}</td>
                  <td class="text-foreground">{{ row.component_name }}</td>
                  <td class="text-foreground">{{ row.protocol }}</td>
                  <td class="text-foreground">{{ row.access }}</td>
                  <td class="text-foreground">{{ row.listen_port }} → {{ row.container_port }}</td>
                  <td class="text-foreground">{{ row.internal_dns }}</td>
                  <td class="text-foreground">{{ row.client_hint }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Layers, Pencil } from 'lucide-vue-next';
  import { onMounted, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { gatewayApi } from '@/api/cd/gateway';
  import AppSpinner from '@/components/AppSpinner.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { GatewayResp } from '@/gen/proto/orbit/v1/gateway';
  import { formatTime } from '@/utils/time';

  const toast = useToast();
  const { t, te } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const { status, execute } = useStatusAsync();

  const gateway = ref<GatewayResp | null>(null);

  const gatewayId = () => String(route.params.id || '');

  function imagePullPolicyLabel(policy: string) {
    const key = `application.imagePullPolicyLabels.${policy}`;
    return te(key) ? t(key) : policy || '—';
  }

  async function loadGateway() {
    const id = gatewayId();
    if (!id) {
      return;
    }
    try {
      await execute(async () => {
        gateway.value = await gatewayApi.get(id);
      });
    } catch {
      toast.error(t('gateway.toast.loadDetailFailed'));
    }
  }

  function goWorkload() {
    if (!gateway.value) {
      return;
    }
    router.push(`/cd/application/${gateway.value.id}`);
  }

  watch(
    () => route.params.id,
    () => {
      loadGateway();
    }
  );

  onMounted(loadGateway);
</script>
