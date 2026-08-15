<template>
  <div class="overflow-x-auto">
    <AppEmptyState v-if="declarations.length === 0" size="compact" />
    <table v-else class="app-data-table min-w-[720px]">
      <thead>
        <tr>
          <th>{{ t('variableDeclaration.name') }}</th>
          <th>{{ t('variableDeclaration.description') }}</th>
          <th>{{ t('variableDeclaration.value') }}</th>
          <th>{{ t('variableDeclaration.source') }}</th>
          <th v-if="!readonly" class="w-32">{{ t('common.operation') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="decl in declarations" :key="decl.name">
          <td>
            <span class="text-foreground">{{ decl.name }}</span>
          </td>
          <td class="max-w-md truncate" :title="decl.description">
            <span v-if="decl.description" class="text-muted-foreground">
              {{ decl.description }}
            </span>
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
              <button v-if="canEdit(decl)" class="app-link" @click="emit('edit', decl.name)">
                {{ t('common.edit') }}
              </button>
              <button
                v-if="canEdit(decl)"
                class="app-link-danger"
                @click="emit('delete', decl.name)"
              >
                {{ t('common.reset') }}
              </button>
              <button
                v-if="canOverride(decl)"
                class="app-link"
                @click="emit('override', decl.name)"
              >
                覆盖
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
  import { getSourceTone, isVariableEditable } from '@/utils/variableSource';

  const props = withDefaults(
    defineProps<{
      declarations: VariableDeclarationResp[];
      readonly?: boolean;
      allowOverride?: boolean;
    }>(),
    {
      readonly: false,
      allowOverride: false,
    }
  );

  const { t } = useI18n();

  const emit = defineEmits<{
    (e: 'edit', name: string): void;
    (e: 'delete', name: string): void;
    (e: 'override', name: string): void;
  }>();

  function hasDisplayValue(value: unknown) {
    if (value === null || value === undefined) {
      return false;
    }
    if (typeof value === 'string') {
      return value.trim().length > 0;
    }
    return true;
  }

  function effectiveValue(decl: VariableDeclarationResp) {
    return decl.value ?? decl.default;
  }

  function canEdit(decl: VariableDeclarationResp) {
    if (decl.editable !== undefined) {
      return decl.editable;
    }
    return isVariableEditable(requireSource(decl));
  }

  function canOverride(decl: VariableDeclarationResp) {
    return !props.readonly && props.allowOverride && requireSource(decl) === 'pipeline_stage';
  }

  function requireSource(decl: VariableDeclarationResp) {
    if (!decl.source) {
      throw new Error(`Variable declaration source is required: ${decl.name}`);
    }
    return decl.source;
  }
</script>
