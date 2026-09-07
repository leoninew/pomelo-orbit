<template>
  <div class="overflow-x-auto">
    <AppEmptyState v-if="declarations.length === 0" size="compact" />
    <table v-else class="app-data-table min-w-[980px]">
      <thead>
        <tr>
          <th>{{ t('variableDeclaration.name') }}</th>
          <th>{{ t('variableDeclaration.stage') }}</th>
          <th>{{ t('variableDeclaration.description') }}</th>
          <th>{{ t('variableDeclaration.stageDefaults') }}</th>
          <th>{{ t('variableDeclaration.value') }}</th>
          <th>{{ t('variableDeclaration.source') }}</th>
          <th v-if="!readonly" class="w-32">{{ t('common.operation') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="decl in declarations" :key="variableKey(decl)">
          <td>
            <span class="text-foreground">{{ decl.name }}</span>
          </td>
          <td class="max-w-40 truncate" :title="stageLabel(decl)">
            <span v-if="decl.stage_id" class="text-foreground">{{ stageLabel(decl) }}</span>
            <span v-else class="text-muted-foreground">{{ t('variableDeclaration.global') }}</span>
          </td>
          <td class="max-w-md truncate" :title="decl.description">
            <span v-if="decl.description" class="text-muted-foreground">
              {{ decl.description }}
            </span>
          </td>
          <td class="max-w-sm">
            <div v-if="decl.stage_defaults.length > 0" class="flex flex-col gap-1">
              <span
                v-for="(stageDefault, index) in decl.stage_defaults"
                :key="stageDefault.stage_id + '-' + index"
                class="min-w-0 truncate text-foreground"
                :title="displayValue(stageDefault.default)"
              >
                {{ displayValue(stageDefault.default) }}
              </span>
            </div>
          </td>
          <td class="max-w-sm truncate" :title="String(effectiveValue(decl) ?? '')">
            <span v-if="hasDisplayValue(effectiveValue(decl))" class="text-foreground">
              {{ effectiveValue(decl) }}
            </span>
          </td>
          <td>
            <AppBadge variant="status" :tone="getSourceTone(requireSource(decl))">
              {{ requireSource(decl) }}
            </AppBadge>
          </td>
          <td v-if="!readonly" class="w-32">
            <div class="flex items-center gap-3">
              <button v-if="canEdit(decl)" class="app-link" @click="emit('edit', decl)">
                {{ t('common.edit') }}
              </button>
              <button v-if="canEdit(decl)" class="app-link-danger" @click="emit('delete', decl)">
                {{ t('common.reset') }}
              </button>
              <button v-if="canOverride(decl)" class="app-link" @click="emit('override', decl)">
                {{ t('common.edit') }}
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import type { VariableDeclarationResp } from '@/gen/proto/orbit/v1/common/common';
  import { effectiveVariableValue } from '@/utils/variableDeclaration';
  import { getSourceTone, isVariableEditable } from '@/utils/variableSource';

  const props = withDefaults(
    defineProps<{
      declarations: VariableDeclarationResp[];
      readonly?: boolean;
      allowOverride?: boolean;
    }>(),
    { readonly: false, allowOverride: false }
  );

  const { t } = useI18n();

  const emit = defineEmits<{
    (e: 'edit', declaration: VariableDeclarationResp): void;
    (e: 'delete', declaration: VariableDeclarationResp): void;
    (e: 'override', declaration: VariableDeclarationResp): void;
  }>();

  function variableKey(decl: VariableDeclarationResp) {
    return decl.name + ':' + (decl.stage_id || 'global') + ':' + decl.source;
  }

  function stageLabel(decl: VariableDeclarationResp) {
    return decl.stage_name || decl.stage_id;
  }

  function hasDisplayValue(value: unknown) {
    if (value === null || value === undefined) return false;
    return typeof value !== 'string' || value.trim().length > 0;
  }

  function effectiveValue(decl: VariableDeclarationResp) {
    return effectiveVariableValue(decl);
  }

  function displayValue(value: unknown) {
    return value == null ? '' : String(value);
  }

  function canEdit(decl: VariableDeclarationResp) {
    return decl.editable ?? isVariableEditable(requireSource(decl));
  }

  function canOverride(decl: VariableDeclarationResp) {
    return !props.readonly && props.allowOverride && requireSource(decl) === 'pipeline_stage';
  }

  function requireSource(decl: VariableDeclarationResp) {
    if (!decl.source) throw new Error('Variable declaration source is required: ' + decl.name);
    return decl.source;
  }
</script>
