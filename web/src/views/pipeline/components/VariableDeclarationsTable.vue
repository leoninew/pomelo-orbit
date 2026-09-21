<template>
  <div>
    <div role="tablist" class="mb-4 flex border-b border-border">
      <button
        v-for="tab in tabs"
        :key="tab.value"
        role="tab"
        type="button"
        :aria-selected="activeScope === tab.value"
        class="border-b-2 px-3 py-2 text-sm"
        :class="
          activeScope === tab.value
            ? 'border-primary text-foreground'
            : 'border-transparent text-muted-foreground hover:text-foreground'
        "
        @click="activeScope = tab.value"
      >
        {{ tab.label }} ({{ tab.count }})
      </button>
    </div>
    <div role="tabpanel" class="overflow-x-auto">
      <AppEmptyState v-if="visibleVariables.length === 0" size="compact" />
      <table v-else class="app-data-table min-w-[1020px]">
        <thead>
          <tr>
            <th>{{ t('variableDeclaration.name') }}</th>
            <th>类型</th>
            <th>{{ t('variableDeclaration.stage') }}</th>
            <th>{{ t('variableDeclaration.description') }}</th>
            <th>{{ t('variableDeclaration.value') }}</th>
            <th v-if="!readonly" class="w-32">{{ t('common.operation') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="variable in visibleVariables" :key="variableKey(variable)">
            <td>
              <span class="text-foreground">{{ variable.name }}</span>
            </td>
            <td>
              <AppBadge variant="status">{{ variable.kind }}</AppBadge>
            </td>
            <td class="max-w-40 truncate" :title="stageLabel(variable)">
              <span v-if="variable.stage_binding" class="text-foreground">
                {{ stageLabel(variable) }}
              </span>
              <span v-else class="text-muted-foreground">
                {{ t('variableDeclaration.global') }}
              </span>
            </td>
            <td class="max-w-md truncate" :title="configuration(variable)?.description">
              <span v-if="configuration(variable)?.description" class="text-muted-foreground">
                {{ configuration(variable)?.description }}
              </span>
            </td>
            <td class="max-w-sm truncate" :title="displayValue(variable)">
              <span v-if="hasDisplayValue(configurationValue(variable))" class="text-foreground">
                {{ displayValue(variable) }}
              </span>
            </td>
            <td v-if="!readonly" class="w-32">
              <div class="flex items-center gap-3">
                <button v-if="canEdit(variable)" class="app-link" @click="emit('edit', variable)">
                  {{ t('common.edit') }}
                </button>
                <button
                  v-if="canEdit(variable)"
                  class="app-link-danger"
                  @click="emit('delete', variable)"
                >
                  {{ t('common.reset') }}
                </button>
                <button
                  v-if="canOverride(variable)"
                  class="app-link"
                  @click="emit('override', variable)"
                >
                  {{ t('common.edit') }}
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n';
  import { computed, ref } from 'vue';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import type { VariableConfigurationResp, VariableResp } from '@/gen/proto/orbit/v1/common/common';
  import { effectiveVariableValue } from '@/utils/variableDeclaration';

  const props = withDefaults(
    defineProps<{
      variables: VariableResp[];
      readonly?: boolean;
      allowOverride?: boolean;
    }>(),
    { readonly: false, allowOverride: false }
  );

  const { t } = useI18n();

  const emit = defineEmits<{
    (e: 'edit', variable: VariableResp): void;
    (e: 'delete', variable: VariableResp): void;
    (e: 'override', variable: VariableResp): void;
  }>();

  type VariableScope = 'global' | 'stage';

  const activeScope = ref<VariableScope>('global');
  const globalVariables = computed(() =>
    props.variables.filter((variable) => variable.scope === 'global')
  );
  const stageVariables = computed(() =>
    props.variables.filter((variable) => variable.scope === 'stage')
  );
  const visibleVariables = computed(() =>
    activeScope.value === 'global' ? globalVariables.value : stageVariables.value
  );
  const tabs = computed(() => [
    { value: 'global' as const, label: '全局变量', count: globalVariables.value.length },
    { value: 'stage' as const, label: '阶段变量', count: stageVariables.value.length },
  ]);

  function variableKey(variable: VariableResp) {
    return (
      variable.name + ':' + (variable.stage_binding?.stage_id || 'global') + ':' + variable.kind
    );
  }

  function stageLabel(variable: VariableResp) {
    return variable.stage_binding?.stage_name || variable.stage_binding?.stage_id || '';
  }

  function hasDisplayValue(value: unknown) {
    if (value === null || value === undefined) {
      return false;
    }
    return typeof value !== 'string' || value.trim().length > 0;
  }

  function configuration(variable: VariableResp): VariableConfigurationResp | undefined {
    return variable.configuration ?? variable.stage_override ?? variable.global_configuration;
  }

  function configurationValue(variable: VariableResp) {
    return effectiveVariableValue(configuration(variable));
  }

  function displayValue(variable: VariableResp) {
    const value = configurationValue(variable);
    if (configuration(variable)?.secret && hasDisplayValue(value)) {
      return '已设置';
    }
    return value == null ? '' : String(value);
  }

  function canEdit(variable: VariableResp) {
    return (
      !props.readonly &&
      variable.editable &&
      Boolean(variable.configuration?.editable || variable.stage_override?.editable)
    );
  }

  function canOverride(variable: VariableResp) {
    return (
      !props.readonly &&
      props.allowOverride &&
      variable.editable &&
      variable.scope === 'stage' &&
      !variable.configuration?.editable &&
      !variable.stage_override?.editable
    );
  }
</script>
