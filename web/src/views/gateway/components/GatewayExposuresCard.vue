<template>
  <DetailInfoCard :title="t('gateway.exposures.title')" :loading="loading">
    <template #actions>
      <div class="flex flex-wrap items-center gap-2">
        <div v-if="gatewayOptions.length > 1">
          <label class="sr-only" for="gateway-exposures-gateway">
            {{ t('gateway.exposures.selectGateway') }}
          </label>
          <SelectControl
            id="gateway-exposures-gateway"
            :model-value="selectedGatewayId"
            :options="gatewayOptions"
            :disabled="loading"
            width-class="w-48"
            @update:model-value="emit('update:selected-gateway-id', String($event))"
          />
        </div>
        <SearchControl
          v-model="searchText"
          :placeholder="t('gateway.exposures.searchPlaceholder')"
          :loading="loading"
          :show-button="false"
          class="shrink-0"
        />
      </div>
    </template>
    <div class="px-5 py-4">
      <AppEmptyState v-if="filteredExposures.length === 0" size="compact" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[720px]">
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
            <tr v-for="(row, index) in filteredExposures" :key="exposureKey(row, index)">
              <td class="text-foreground">{{ row.application_code }}</td>
              <td class="text-foreground">{{ row.component_name }}</td>
              <td>
                <AppBadge variant="pill">{{ row.protocol }}</AppBadge>
              </td>
              <td>
                <AppBadge variant="pill">{{ row.access }}</AppBadge>
              </td>
              <td class="text-foreground">{{ row.listen_port }}:{{ row.container_port }}</td>
              <td class="text-foreground">{{ row.internal_dns }}</td>
              <td class="text-foreground">
                <a
                  v-if="isHTTPAddress(row.client_hint)"
                  :href="row.client_hint"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="app-link inline-flex items-center gap-1"
                >
                  {{ row.client_hint }}
                  <ExternalLink class="size-3.5 shrink-0" />
                </a>
                <template v-else>{{ row.client_hint }}</template>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </DetailInfoCard>
</template>

<script setup lang="ts">
  import { ExternalLink } from '@lucide/vue';
  import { computed, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import SelectControl, { type SelectOption } from '@/components/SelectControl.vue';
  import type { GatewayExposureItem, GatewayResp } from '@/gen/proto/orbit/v1/gateway/gateway';

  const props = withDefaults(
    defineProps<{
      gateway: GatewayResp;
      gatewayOptions?: SelectOption[];
      selectedGatewayId?: string;
      loading?: boolean;
    }>(),
    {
      gatewayOptions: () => [],
      selectedGatewayId: '',
      loading: false,
    }
  );
  const emit = defineEmits<{
    'update:selected-gateway-id': [id: string];
  }>();
  const { t } = useI18n();
  const searchText = ref('');

  const filteredExposures = computed(() => {
    const keyword = searchText.value.trim().toLowerCase();
    if (!keyword) {
      return props.gateway.exposures;
    }
    return props.gateway.exposures.filter(
      (row) =>
        row.application_code.toLowerCase().includes(keyword) ||
        row.component_name.toLowerCase().includes(keyword) ||
        row.protocol.toLowerCase().includes(keyword) ||
        row.access.toLowerCase().includes(keyword) ||
        String(row.listen_port).includes(keyword) ||
        row.internal_dns.toLowerCase().includes(keyword) ||
        row.client_hint.toLowerCase().includes(keyword)
    );
  });

  function exposureKey(row: GatewayExposureItem, index: number) {
    return `${row.application_id}:${row.component_name}:${row.protocol}:${row.container_port}:${index}`;
  }

  function isHTTPAddress(value: string) {
    return /^https?:\/\//i.test(value);
  }
</script>
