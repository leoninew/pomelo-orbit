<template>
  <AppDialog
    v-model:open="isOpen"
    :title="t('application.versionDetail.dialog.componentRuntime')"
    width-class="w-[min(860px,calc(100vw-32px))]"
  >
    <div class="space-y-6">
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('application.runtime.command') }}</label>
          <textarea
            v-model="commandLines"
            rows="4"
            class="app-textarea font-mono text-xs"
            :disabled="readonly"
            :placeholder="t('application.runtime.oneArgumentPerLine')"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('application.runtime.args') }}</label>
          <textarea
            v-model="argsLines"
            rows="4"
            class="app-textarea font-mono text-xs"
            :disabled="readonly"
            :placeholder="t('application.runtime.oneArgumentPerLine')"
          />
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('application.runtime.healthcheck') }}</label>
          <textarea
            v-model="healthcheckJson"
            rows="5"
            class="app-textarea font-mono text-xs"
            :disabled="readonly"
            :placeholder="t('application.runtime.jsonObject')"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('application.runtime.resources') }}</label>
          <textarea
            v-model="resourcesJson"
            rows="5"
            class="app-textarea font-mono text-xs"
            :disabled="readonly"
            :placeholder="t('application.runtime.jsonObject')"
          />
        </div>
      </div>

      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('application.runtime.restartPolicy') }}</label>
        <ComboboxSelect
          v-model="restartPolicy"
          :options="restartPolicyOptions"
          :disabled="readonly"
          :placeholder="t('application.runtime.restartPolicyPlaceholder')"
          width-class="w-full sm:w-64"
        />
      </div>

      <div class="space-y-2">
        <div class="flex items-center justify-between gap-3">
          <label class="app-field-label">{{ t('application.runtime.tmpfs') }}</label>
          <button
            v-if="!readonly"
            type="button"
            class="app-link inline-flex items-center gap-1 text-sm"
            @click="tmpfsRows.push({ target: '', size_bytes: 1048576, mode: '1777' })"
          >
            <Plus class="size-3.5" />
            {{ t('application.runtime.addTmpfs') }}
          </button>
        </div>
        <div
          v-for="(row, index) in tmpfsRows"
          :key="`tmpfs-${index}`"
          class="grid grid-cols-1 gap-2 sm:grid-cols-[1.2fr_1fr_0.7fr_auto]"
        >
          <input
            v-model="row.target"
            type="text"
            class="app-input text-xs"
            :disabled="readonly"
            placeholder="/tmp"
          />
          <input
            v-model.number="row.size_bytes"
            type="number"
            min="1048576"
            max="8589934592"
            class="app-input text-xs"
            :disabled="readonly"
            :placeholder="t('application.runtime.sizeBytes')"
          />
          <input
            v-model="row.mode"
            type="text"
            class="app-input font-mono text-xs"
            :disabled="readonly"
            placeholder="1777"
          />
          <button
            v-if="!readonly"
            type="button"
            class="app-link-danger inline-flex items-center justify-center"
            :aria-label="t('common.delete')"
            @click="tmpfsRows.splice(index, 1)"
          >
            <Trash2 class="size-4" />
          </button>
        </div>
      </div>

      <div class="space-y-2">
        <div class="flex items-center justify-between gap-3">
          <label class="app-field-label">{{ t('application.runtime.ulimits') }}</label>
          <button
            v-if="!readonly"
            type="button"
            class="app-link inline-flex items-center gap-1 text-sm"
            @click="ulimitRows.push({ name: 'nofile', soft: 1024, hard: 4096 })"
          >
            <Plus class="size-3.5" />
            {{ t('application.runtime.addUlimit') }}
          </button>
        </div>
        <div
          v-for="(row, index) in ulimitRows"
          :key="`ulimit-${index}`"
          class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1fr_1fr_auto]"
        >
          <RawValueSelect
            v-model="row.name"
            :values="ulimitNameValues"
            :disabled="readonly"
            :placeholder="t('application.runtime.ulimitName')"
          />
          <input
            v-model.number="row.soft"
            type="number"
            min="-1"
            class="app-input text-xs"
            :disabled="readonly"
            :placeholder="t('application.runtime.softLimit')"
          />
          <input
            v-model.number="row.hard"
            type="number"
            min="-1"
            class="app-input text-xs"
            :disabled="readonly"
            :placeholder="t('application.runtime.hardLimit')"
          />
          <button
            v-if="!readonly"
            type="button"
            class="app-link-danger inline-flex items-center justify-center"
            :aria-label="t('common.delete')"
            @click="ulimitRows.splice(index, 1)"
          >
            <Trash2 class="size-4" />
          </button>
        </div>
      </div>

      <p v-if="formError" class="app-field-error text-xs">{{ formError }}</p>
    </div>
    <template #footer>
      <button class="app-button" @click="isOpen = false">{{ t('common.cancel') }}</button>
      <button v-if="!readonly" :disabled="saving" class="app-button-primary" @click="save">
        {{ t('common.save') }}
      </button>
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { Plus, Trash2 } from 'lucide-vue-next';
  import AppDialog from '@/components/AppDialog.vue';
  import ComboboxSelect, { type ComboboxOption } from '@/components/ComboboxSelect.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import type { VersionComponentResp } from '@/gen/proto/orbit/v1/application/version';

  interface RuntimeConfig {
    command_json?: string;
    args_json?: string;
    healthcheck_json?: string;
    resources_json?: string;
    restart_policy?: string;
    tmpfs_json?: string;
    ulimits_json?: string;
  }

  interface TmpfsRow {
    target: string;
    size_bytes: number;
    mode: string;
  }

  interface UlimitRow {
    name: string;
    soft: number;
    hard: number;
  }

  const props = withDefaults(
    defineProps<{
      open: boolean;
      component?: VersionComponentResp;
      readonly?: boolean;
      saving?: boolean;
    }>(),
    { readonly: false, saving: false }
  );

  const emit = defineEmits<{
    'update:open': [open: boolean];
    save: [config: RuntimeConfig];
  }>();

  const { t } = useI18n();
  const commandLines = ref('');
  const argsLines = ref('');
  const healthcheckJson = ref('');
  const resourcesJson = ref('');
  const restartPolicy = ref('');
  const tmpfsRows = ref<TmpfsRow[]>([]);
  const ulimitRows = ref<UlimitRow[]>([]);
  const formError = ref('');
  const restartPolicyOptions: ComboboxOption[] = [
    { value: 'no', label: 'no' },
    { value: 'unless-stopped', label: 'unless-stopped' },
  ];
  const ulimitNameValues = ['memlock', 'nofile'];
  const isOpen = computed({
    get: () => props.open,
    set: (open) => emit('update:open', open),
  });

  watch(
    () => props.open,
    (open) => {
      if (!open) {
        return;
      }
      commandLines.value = parseStringArray(props.component?.command_json).join('\n');
      argsLines.value = parseStringArray(props.component?.args_json).join('\n');
      healthcheckJson.value = props.component?.healthcheck_json || '';
      resourcesJson.value = props.component?.resources_json || '';
      restartPolicy.value = props.component?.restart_policy || '';
      tmpfsRows.value = parseTmpfsRows(props.component?.tmpfs_json);
      ulimitRows.value = parseUlimitRows(props.component?.ulimits_json);
      formError.value = '';
    }
  );

  function parseStringArray(raw?: string): string[] {
    if (!raw?.trim()) {
      return [];
    }
    try {
      const parsed = JSON.parse(raw) as unknown;
      return Array.isArray(parsed) && parsed.every((value) => typeof value === 'string')
        ? parsed
        : [];
    } catch {
      return [];
    }
  }

  function parseTmpfsRows(raw?: string): TmpfsRow[] {
    return parseArray(raw, (item) => {
      const value = item as Partial<TmpfsRow>;
      return {
        target: String(value.target || ''),
        size_bytes: Number(value.size_bytes || 0),
        mode: String(value.mode || ''),
      };
    });
  }

  function parseUlimitRows(raw?: string): UlimitRow[] {
    return parseArray(raw, (item) => {
      const value = item as Partial<UlimitRow>;
      return {
        name: String(value.name || ''),
        soft: Number(value.soft ?? 0),
        hard: Number(value.hard ?? 0),
      };
    });
  }

  function parseArray<T>(raw: string | undefined, mapper: (item: unknown) => T): T[] {
    if (!raw?.trim()) {
      return [];
    }
    try {
      const parsed = JSON.parse(raw) as unknown;
      return Array.isArray(parsed) ? parsed.map(mapper) : [];
    } catch {
      return [];
    }
  }

  function serializeStringArray(lines: string): string | undefined {
    const values = lines.split('\n').filter((line) => line.length > 0);
    return values.length > 0 ? JSON.stringify(values) : undefined;
  }

  function validateJsonObject(label: string, raw: string): string | undefined {
    if (!raw.trim()) {
      return undefined;
    }
    try {
      const parsed = JSON.parse(raw) as unknown;
      if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
        return t('application.runtime.jsonObjectRequired', { label });
      }
    } catch {
      return t('application.runtime.jsonInvalid', { label });
    }
    return undefined;
  }

  function save() {
    formError.value =
      validateJsonObject(t('application.runtime.healthcheck'), healthcheckJson.value) ||
      validateJsonObject(t('application.runtime.resources'), resourcesJson.value) ||
      validateTmpfs() ||
      validateUlimits() ||
      '';
    if (formError.value) {
      return;
    }
    emit('save', {
      command_json: serializeStringArray(commandLines.value),
      args_json: serializeStringArray(argsLines.value),
      healthcheck_json: healthcheckJson.value.trim() || undefined,
      resources_json: resourcesJson.value.trim() || undefined,
      restart_policy: restartPolicy.value || undefined,
      tmpfs_json: tmpfsRows.value.length > 0 ? JSON.stringify(tmpfsRows.value) : undefined,
      ulimits_json: ulimitRows.value.length > 0 ? JSON.stringify(ulimitRows.value) : undefined,
    });
  }

  function validateTmpfs(): string | undefined {
    const targetSet = new Set<string>();
    for (const row of tmpfsRows.value) {
      if (
        !row.target.startsWith('/') ||
        row.target === '/' ||
        row.target.split('/').includes('..') ||
        /^\/(proc|sys|dev)(\/|$)/.test(row.target) ||
        !Number.isInteger(row.size_bytes) ||
        row.size_bytes < 1048576 ||
        row.size_bytes > 8589934592 ||
        !/^[0-7]{3,4}$/.test(row.mode) ||
        targetSet.has(row.target)
      ) {
        return t('application.runtime.tmpfsInvalid');
      }
      targetSet.add(row.target);
    }
    return undefined;
  }

  function validateUlimits(): string | undefined {
    const names = new Set<string>();
    for (const row of ulimitRows.value) {
      const unlimitedMemlock = row.name === 'memlock' && row.soft === -1 && row.hard === -1;
      if (
        !ulimitNameValues.includes(row.name) ||
        names.has(row.name) ||
        (!unlimitedMemlock &&
          (!Number.isInteger(row.soft) ||
            !Number.isInteger(row.hard) ||
            row.soft < 0 ||
            row.hard < 0 ||
            row.soft > row.hard))
      ) {
        return t('application.runtime.ulimitInvalid');
      }
      names.add(row.name);
    }
    return undefined;
  }

</script>
