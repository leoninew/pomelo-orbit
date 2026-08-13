<template>
  <div class="space-y-1.5">
    <label class="app-field-label block">
      {{ t('route.fields.service') }}
      <span class="text-destructive">*</span>
    </label>
    <ComboboxSelect
      :model-value="serviceId"
      :options="serviceOptions"
      :placeholder="t('route.placeholders.selectService')"
      :disabled="disabled"
      :invalid="Boolean(serviceError)"
      @update:model-value="handleServiceChange"
    />
    <p v-if="serviceError" class="app-field-error text-xs">{{ serviceError }}</p>
  </div>
  <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
    <div class="space-y-1.5">
      <label class="app-field-label block">
        {{ t('route.fields.component') }}
        <span class="text-destructive">*</span>
      </label>
      <ComboboxSelect
        :model-value="componentName"
        :options="componentOptions"
        :placeholder="t('route.placeholders.selectComponent')"
        :disabled="disabled || !serviceId"
        :invalid="Boolean(componentError)"
        @update:model-value="handleComponentChange"
      />
      <p v-if="componentError" class="app-field-error text-xs">{{ componentError }}</p>
    </div>
    <div class="space-y-1.5">
      <label class="app-field-label block">
        {{ t('route.fields.endpoint') }}
        <span class="text-destructive">*</span>
      </label>
      <ComboboxSelect
        :model-value="endpointIdentity"
        :options="endpointOptions"
        :placeholder="t('route.placeholders.selectEndpoint')"
        :disabled="disabled || !componentName"
        :invalid="Boolean(endpointError)"
        description-inline
        @update:model-value="handleEndpointChange"
      />
      <p v-if="endpointError" class="app-field-error text-xs">{{ endpointError }}</p>
    </div>
  </div>
  <div v-if="protocol === 'tcp'" class="space-y-1.5">
    <label class="app-field-label block">
      {{ t('route.fields.listenPort') }}
      <span class="text-destructive">*</span>
    </label>
    <input
      :value="listenPort"
      type="number"
      min="1"
      max="65535"
      class="app-input"
      :class="listenPortError ? 'app-input-error' : ''"
      :disabled="disabled || endpointContainerPort === undefined"
      @input="handleListenPortInput"
    />
    <p v-if="listenPortError" class="app-field-error text-xs">{{ listenPortError }}</p>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import ComboboxSelect, { type ComboboxOptionValue } from '@/components/ComboboxSelect.vue';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';

  const props = withDefaults(
    defineProps<{
      protocol: 'http' | 'tcp';
      services: ServiceResp[];
      serviceId: string;
      componentName: string;
      endpointProtocol: string;
      endpointContainerPort?: number;
      listenPort?: number;
      serviceError?: string;
      componentError?: string;
      endpointError?: string;
      listenPortError?: string;
      disabled?: boolean;
    }>(),
    {
      serviceError: '',
      componentError: '',
      endpointError: '',
      listenPortError: '',
      disabled: false,
    }
  );

  const emit = defineEmits<{
    'update:service-id': [value: string];
    'update:component-name': [value: string];
    'update:endpoint-protocol': [value: string];
    'update:endpoint-container-port': [value: number | undefined];
    'update:listen-port': [value: number | undefined];
  }>();

  const { t } = useI18n();
  const selectedService = computed(() =>
    props.services.find((service) => service.id === props.serviceId)
  );
  const endpointIdentity = computed(() =>
    props.endpointProtocol && props.endpointContainerPort !== undefined
      ? `${props.endpointProtocol}:${props.endpointContainerPort}`
      : ''
  );
  const serviceOptions = computed(() =>
    props.services
      .filter(hasSelectableEndpoint)
      .map((service) => ({
        value: service.id,
        label: service.application_name,
      }))
  );
  const componentOptions = computed(() =>
    (selectedService.value?.components ?? [])
      .filter((component) => component.effective_endpoints.some(isSelectableEndpoint))
      .map((component) => ({
        value: component.component_name,
        label: component.component_name,
      }))
  );
  const endpointOptions = computed(() => {
    const component = selectedService.value?.components.find(
      (item) => item.component_name === props.componentName
    );
    return (component?.effective_endpoints ?? [])
      .filter(isSelectableEndpoint)
      .map((endpoint) => ({
        value: `${endpoint.protocol}:${endpoint.container_port}`,
        label: `${endpoint.protocol}${endpoint.container_port}`,
      }));
  });

  function handleServiceChange(value: ComboboxOptionValue) {
    emit('update:service-id', String(value || ''));
    emit('update:component-name', '');
    emit('update:endpoint-protocol', '');
    emit('update:endpoint-container-port', undefined);
    emit('update:listen-port', undefined);
  }

  function handleComponentChange(value: ComboboxOptionValue) {
    emit('update:component-name', String(value || ''));
    emit('update:endpoint-protocol', '');
    emit('update:endpoint-container-port', undefined);
    emit('update:listen-port', undefined);
  }

  function handleEndpointChange(value: ComboboxOptionValue) {
    const identity = String(value || '');
    const component = selectedService.value?.components.find(
      (item) => item.component_name === props.componentName
    );
    const endpoint = component?.effective_endpoints.find(
      (item) => `${item.protocol}:${item.container_port}` === identity
    );
    emit('update:endpoint-protocol', endpoint?.protocol ?? '');
    emit('update:endpoint-container-port', endpoint?.container_port);
    emit('update:listen-port', endpoint?.container_port);
  }

  function handleListenPortInput(event: Event) {
    const input = event.target as HTMLInputElement;
    emit('update:listen-port', Number.isNaN(input.valueAsNumber) ? undefined : input.valueAsNumber);
  }

  function hasSelectableEndpoint(service: ServiceResp) {
    return service.components.some((component) => component.effective_endpoints.some(isSelectableEndpoint));
  }

  function isSelectableEndpoint(endpoint: { protocol: string; mode: string }) {
    return (
      endpoint.protocol === props.protocol &&
      (props.protocol !== 'tcp' || endpoint.mode === 'internal')
    );
  }
</script>
