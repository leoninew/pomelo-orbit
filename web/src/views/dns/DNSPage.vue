<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-scroll" :aria-label="t('dns.toolbar')">
      <div class="app-toolbar-row">
        <ComboboxSelect
          :model-value="zoneId"
          :options="zones.map((zone) => ({ value: zone.id, label: zone.name }))"
          :placeholder="t('dns.selectZone')"
          :disabled="zonesLoading || saving || creating || deleting || zones.length === 0"
          width-class="app-toolbar-select"
          @update:model-value="selectZone(String($event))"
        />
        <SearchControl
          v-model="searchText"
          :placeholder="t('dns.searchPlaceholder')"
          :loading="recordsLoading"
          :disabled="!zoneId"
          class="shrink-0"
          @search="searchRecords"
        />
        <button
          type="button"
          class="app-button-primary ml-auto h-10 px-3"
          :disabled="!zoneId || zonesLoading || recordsLoading || saving || creating || deleting"
          @click="openCreator"
        >
          <Plus class="size-4" aria-hidden="true" />
          {{ t('dns.createTitle') }}
        </button>
      </div>
    </ToolbarRoot>
    <div class="app-list-surface">
      <AppEmptyState v-if="!projectStore.activeProjectId">
        {{ t('dns.selectProject') }}
      </AppEmptyState>
      <AppLoadingState v-else-if="zonesLoading || recordsLoading" />
      <div v-else-if="loadError" class="space-y-3 py-16 text-center">
        <p class="text-sm text-destructive" role="alert">{{ loadError }}</p>
        <router-link v-if="loadErrorCode === 'dns_not_configured'" to="/gateway" class="app-link">
          {{ t('dns.configureToken') }}
        </router-link>
        <button v-else type="button" class="app-link mx-auto block" @click="loadProject()">
          {{ t('dns.retry') }}
        </button>
      </div>
      <AppEmptyState v-else-if="zones.length === 0">{{ t('dns.noZones') }}</AppEmptyState>
      <AppEmptyState v-else-if="!zoneId">{{ t('dns.selectZone') }}</AppEmptyState>
      <template v-else>
        <AppEmptyState v-if="records.length === 0">
          {{ t(search ? 'dns.noSearchResults' : 'dns.noRecords') }}
        </AppEmptyState>
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[800px]">
            <thead>
              <tr>
                <th>{{ t('dns.name') }}</th>
                <th>{{ t('dns.type') }}</th>
                <th>{{ t('dns.address') }}</th>
                <th>{{ t('dns.ttl') }}</th>
                <th>{{ t('dns.proxy') }}</th>
                <th>{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="record in records" :key="record.id">
                <td class="max-w-80 text-foreground"><AppTruncatedText :text="record.name" /></td>
                <td><AppBadge variant="category">A</AppBadge></td>
                <td class="whitespace-nowrap text-foreground">{{ record.content }}</td>
                <td class="whitespace-nowrap">
                  {{
                    record.ttl === 1 ? t('dns.automatic') : t('dns.seconds', { value: record.ttl })
                  }}
                </td>
                <td>
                  <AppBadge variant="status" :tone="record.proxied ? 'warning' : 'default'">
                    {{ record.proxied ? t('dns.proxied') : t('dns.dnsOnly') }}
                  </AppBadge>
                </td>
                <td class="app-table-actions whitespace-nowrap">
                  <div class="flex items-center gap-3">
                    <button
                      type="button"
                      class="app-link whitespace-nowrap"
                      :aria-label="t('dns.editRecord', { name: record.name })"
                      @click="edit(record)"
                    >
                      {{ t('common.edit') }}
                    </button>
                    <button
                      type="button"
                      class="app-link-danger whitespace-nowrap"
                      :aria-label="t('dns.deleteRecord', { name: record.name })"
                      @click="openDelete(record)"
                    >
                      {{ t('common.delete') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <ListPagination
          :current="page"
          :page-size="perPage"
          :total="total"
          :total-pages="pages"
          @change-page="changePage"
          @change-page-size="changePageSize"
        />
      </template>
    </div>
  </div>
  <AppDialog :open="editing !== null" :title="t('dns.editTitle')" @update:open="closeEditor">
    <form class="space-y-4" novalidate @submit.prevent="save">
      <div class="space-y-1.5">
        <label for="dns-record-name" class="app-field-label block">{{ t('dns.name') }}</label>
        <AppInput
          id="dns-record-name"
          :model-value="editing?.name ?? ''"
          readonly
          :clearable="false"
        />
      </div>
      <div class="space-y-1.5">
        <label for="dns-record-content" class="app-field-label block">
          {{ t('dns.address') }}
          <span class="text-destructive">*</span>
        </label>
        <AppInput
          id="dns-record-content"
          v-model="form.content"
          :disabled="saving"
          :class="errors.content ? 'app-input-error' : ''"
          :aria-invalid="errors.content ? 'true' : undefined"
          :aria-describedby="errors.content ? 'dns-content-error' : undefined"
          placeholder="192.0.2.1"
          @input="delete errors.content"
        />
        <p v-if="errors.content" id="dns-content-error" class="app-field-error" role="alert">
          {{ t(`dns.${errors.content}`) }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label for="dns-record-ttl" class="app-field-label block">
          {{ t('dns.ttl') }}
          <span class="text-destructive">*</span>
        </label>
        <AppInput
          id="dns-record-ttl"
          v-model="form.ttl"
          inputmode="numeric"
          :disabled="saving || form.proxied"
          :class="errors.ttl ? 'app-input-error' : ''"
          :aria-invalid="errors.ttl ? 'true' : undefined"
          :aria-describedby="errors.ttl ? 'dns-ttl-error dns-ttl-hint' : 'dns-ttl-hint'"
          @input="delete errors.ttl"
        />
        <p id="dns-ttl-hint" class="app-field-hint">
          {{ t(form.proxied ? 'dns.proxiedTTLHint' : 'dns.ttlHint') }}
        </p>
        <p v-if="errors.ttl" id="dns-ttl-error" class="app-field-error" role="alert">
          {{ t(`dns.${errors.ttl}`) }}
        </p>
      </div>
      <label class="flex items-center gap-2 text-sm">
        <input
          v-model="form.proxied"
          type="checkbox"
          class="app-checkbox"
          :disabled="saving || !editing?.proxiable"
          @change="handleProxyChange"
        />
        {{ t('dns.enableProxy') }}
      </label>
      <p v-if="saveError" class="app-field-error" role="alert">{{ saveError }}</p>
    </form>
    <template #footer>
      <AppDialogActions :busy="saving" @cancel="closeEditor(false)" @confirm="save" />
    </template>
  </AppDialog>
  <AppDialog :open="creatorOpen" :title="t('dns.createTitle')" @update:open="closeCreator">
    <form class="space-y-4" novalidate @submit.prevent="createRecord">
      <div class="space-y-1.5">
        <label for="dns-create-name" class="app-field-label block">
          {{ t('dns.name') }}
          <span class="text-destructive">*</span>
        </label>
        <AppInput
          id="dns-create-name"
          v-model="createForm.name"
          :disabled="creating"
          :class="createErrors.name ? 'app-input-error' : ''"
          :aria-invalid="createErrors.name ? 'true' : undefined"
          :aria-describedby="createErrors.name ? 'dns-create-name-error' : undefined"
          :placeholder="t('dns.namePlaceholder')"
          @input="delete createErrors.name"
        />
        <p v-if="createErrors.name" id="dns-create-name-error" class="app-field-error" role="alert">
          {{ t('dns.invalidName') }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label for="dns-create-content" class="app-field-label block">
          {{ t('dns.address') }}
          <span class="text-destructive">*</span>
        </label>
        <AppInput
          id="dns-create-content"
          v-model="createForm.content"
          :disabled="creating"
          placeholder="192.0.2.1"
          :class="createErrors.content ? 'app-input-error' : ''"
          :aria-invalid="createErrors.content ? 'true' : undefined"
          :aria-describedby="createErrors.content ? 'dns-create-content-error' : undefined"
          @input="delete createErrors.content"
        />
        <p
          v-if="createErrors.content"
          id="dns-create-content-error"
          class="app-field-error"
          role="alert"
        >
          {{ t('dns.invalidIPv4') }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label for="dns-create-ttl" class="app-field-label block">
          {{ t('dns.ttl') }}
          <span class="text-destructive">*</span>
        </label>
        <AppInput
          id="dns-create-ttl"
          v-model="createForm.ttl"
          inputmode="numeric"
          :disabled="creating || createForm.proxied"
          :class="createErrors.ttl ? 'app-input-error' : ''"
          :aria-invalid="createErrors.ttl ? 'true' : undefined"
          :aria-describedby="
            createErrors.ttl ? 'dns-create-ttl-error dns-create-ttl-hint' : 'dns-create-ttl-hint'
          "
          @input="delete createErrors.ttl"
        />
        <p v-if="createErrors.ttl" id="dns-create-ttl-error" class="app-field-error" role="alert">
          {{ t('dns.invalidTTL') }}
        </p>
        <p id="dns-create-ttl-hint" class="app-field-hint">
          {{ t(createForm.proxied ? 'dns.proxiedTTLHint' : 'dns.ttlHint') }}
        </p>
      </div>
      <label class="flex items-center gap-2 text-sm">
        <input
          v-model="createForm.proxied"
          type="checkbox"
          class="app-checkbox"
          :disabled="creating"
          @change="handleCreateProxyChange"
        />
        {{ t('dns.enableProxy') }}
      </label>
      <p v-if="createError" class="app-field-error" role="alert">{{ createError }}</p>
    </form>
    <template #footer>
      <AppDialogActions :busy="creating" @cancel="closeCreator(false)" @confirm="createRecord" />
    </template>
  </AppDialog>
  <AppDialog
    :open="pendingDelete !== null"
    :title="t('dns.deleteTitle')"
    @update:open="closeDelete"
  >
    <p class="text-sm text-foreground">
      {{ t('dns.deleteConfirm', { name: pendingDelete?.name ?? '' }) }}
    </p>
    <p v-if="deleteError" class="app-field-error mt-3" role="alert">{{ deleteError }}</p>
    <template #footer>
      <AppDialogActions
        :busy="deleting"
        :confirm-label="t('common.delete')"
        variant="destructive"
        @cancel="closeDelete(false)"
        @confirm="deleteRecord"
      />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { Plus } from '@lucide/vue';
  import { ToolbarRoot } from 'reka-ui';
  import { onBeforeUnmount, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { dnsApi } from '@/api/dns/dns';
  import AppBadge from '@/components/ui/AppBadge.vue';
  import AppDialog from '@/components/ui/AppDialog.vue';
  import AppDialogActions from '@/components/ui/AppDialogActions.vue';
  import AppEmptyState from '@/components/ui/AppEmptyState.vue';
  import AppInput from '@/components/ui/AppInput.vue';
  import AppLoadingState from '@/components/ui/AppLoadingState.vue';
  import AppTruncatedText from '@/components/ui/AppTruncatedText.vue';
  import ListPagination from '@/components/ui/ListPagination.vue';
  import ComboboxSelect from '@/components/ui/ComboboxSelect.vue';
  import SearchControl from '@/components/ui/SearchControl.vue';
  import { useToast } from '@/composables/useToast';
  import type { RecordResp, ZoneResp } from '@/gen/proto/orbit/v1/dns/dns';
  import { useProjectStore } from '@/stores/project';
  import { ApiError } from '@/utils/request';
  import {
    validateDNSCreateForm,
    validateDNSRecordForm,
    type DNSRecordErrors,
    type DNSRecordForm,
  } from './dnsRecordForm';

  const { t } = useI18n();
  const projectStore = useProjectStore();
  const toast = useToast();
  const zones = ref<ZoneResp[]>([]);
  const records = ref<RecordResp[]>([]);
  const zoneId = ref('');
  const searchText = ref('');
  const search = ref('');
  const creatorOpen = ref(false);
  const creating = ref(false);
  const deleting = ref(false);
  const pendingDelete = ref<RecordResp | null>(null);
  const deleteError = ref('');
  const createError = ref('');
  const createForm = reactive<DNSRecordForm & { name: string }>({
    name: '',
    content: '',
    ttl: '1',
    proxied: false,
  });
  const createErrors = ref<ReturnType<typeof validateDNSCreateForm>>({});
  const page = ref(1);
  const perPage = ref(20);
  const total = ref(0);
  const pages = ref(0);
  const zonesLoading = ref(false);
  const recordsLoading = ref(false);
  const loadError = ref('');
  const loadErrorCode = ref('');
  const editing = ref<RecordResp | null>(null);
  const saving = ref(false);
  const saveError = ref('');
  const form = reactive<DNSRecordForm>({ content: '', ttl: '1', proxied: false });
  const errors = ref<DNSRecordErrors>({});
  let scopeVersion = 0;
  let recordVersion = 0;
  let saveVersion = 0;

  function errorMessage(error: unknown): string {
    if (error instanceof ApiError) {
      if (error.code === 'dns_not_configured') {
        return t('dns.tokenMissing');
      }
      if (error.code === 'dns_permission_denied') {
        return t('dns.permissionDenied');
      }
    }
    return error instanceof Error ? error.message : t('dns.loadFailed');
  }

  async function loadProject() {
    const projectId = projectStore.activeProjectId;
    const previousZone = zoneId.value;
    const version = ++scopeVersion;
    ++recordVersion;
    ++saveVersion;
    zones.value = [];
    records.value = [];
    zoneId.value = '';
    page.value = 1;
    total.value = 0;
    pages.value = 0;
    editing.value = null;
    creatorOpen.value = false;
    pendingDelete.value = null;
    deleting.value = false;
    creating.value = false;
    searchText.value = '';
    search.value = '';
    saving.value = false;
    recordsLoading.value = false;
    loadError.value = '';
    loadErrorCode.value = '';
    zonesLoading.value = !!projectId;
    if (!projectId) {
      return;
    }
    try {
      const result = await dnsApi.zones(projectId);
      if (scopeVersion !== version || projectStore.activeProjectId !== projectId) {
        return;
      }
      zones.value = result.items;
      zoneId.value = result.items.some((zone) => zone.id === previousZone)
        ? previousZone
        : (result.items[0]?.id ?? '');
      if (zoneId.value) {
        void loadRecords();
      }
    } catch (error) {
      if (scopeVersion !== version || projectStore.activeProjectId !== projectId) {
        return;
      }
      loadError.value = errorMessage(error);
      loadErrorCode.value = error instanceof ApiError ? (error.code ?? '') : '';
    } finally {
      if (scopeVersion === version) {
        zonesLoading.value = false;
      }
    }
  }

  async function loadRecords() {
    const projectId = projectStore.activeProjectId;
    const zone = zoneId.value;
    const version = ++recordVersion;
    records.value = [];
    loadError.value = '';
    loadErrorCode.value = '';
    total.value = 0;
    pages.value = 0;
    if (!projectId || !zone) {
      recordsLoading.value = false;
      return;
    }
    recordsLoading.value = true;
    try {
      const result = await dnsApi.records(projectId, zone, page.value, perPage.value, search.value);
      if (
        recordVersion !== version ||
        projectStore.activeProjectId !== projectId ||
        zoneId.value !== zone
      ) {
        return;
      }
      records.value = result.items;
      total.value = result.total;
      pages.value = result.pages;
    } catch (error) {
      if (
        recordVersion !== version ||
        projectStore.activeProjectId !== projectId ||
        zoneId.value !== zone
      ) {
        return;
      }
      loadError.value = errorMessage(error);
      loadErrorCode.value = error instanceof ApiError ? (error.code ?? '') : '';
    } finally {
      if (recordVersion === version) {
        recordsLoading.value = false;
      }
    }
  }

  function selectZone(value: string) {
    zoneId.value = value;
    page.value = 1;
    editing.value = null;
    creatorOpen.value = false;
    pendingDelete.value = null;
    searchText.value = '';
    search.value = '';
    void loadRecords();
  }

  function searchRecords() {
    search.value = searchText.value.trim();
    page.value = 1;
    void loadRecords();
  }

  function changePage(value: number) {
    page.value = value;
    void loadRecords();
  }

  function changePageSize(value: number) {
    perPage.value = value;
    page.value = 1;
    void loadRecords();
  }

  function edit(record: RecordResp) {
    editing.value = record;
    Object.assign(form, {
      content: record.content,
      ttl: String(record.ttl),
      proxied: record.proxied,
    });
    errors.value = {};
    saveError.value = '';
  }

  function closeEditor(open: boolean) {
    if (!open && !saving.value) {
      editing.value = null;
    }
  }

  function handleProxyChange() {
    if (form.proxied) {
      form.ttl = '1';
    }
    delete errors.value.ttl;
  }

  async function save() {
    const record = editing.value;
    const projectId = projectStore.activeProjectId;
    const zone = zoneId.value;
    if (saving.value || !record || !projectId || !zone) {
      return;
    }
    errors.value = validateDNSRecordForm(form);
    if (Object.keys(errors.value).length) {
      return;
    }
    const scope = scopeVersion;
    const version = ++saveVersion;
    saving.value = true;
    saveError.value = '';
    try {
      const updated = await dnsApi.update(projectId, zone, record.id, {
        content: form.content.trim(),
        ttl: form.proxied ? 1 : Number(form.ttl),
        proxied: form.proxied,
      });
      if (
        scopeVersion !== scope ||
        projectStore.activeProjectId !== projectId ||
        zoneId.value !== zone
      ) {
        return;
      }
      records.value = records.value.map((item) => (item.id === updated.id ? updated : item));
      editing.value = null;
      toast.success(t('dns.saved'));
    } catch (error) {
      if (
        scopeVersion === scope &&
        projectStore.activeProjectId === projectId &&
        zoneId.value === zone
      ) {
        saveError.value = `${errorMessage(error)} ${t('dns.verifyAfterFailure')}`;
      }
    } finally {
      if (saveVersion === version) {
        saving.value = false;
      }
    }
  }

  function openCreator() {
    Object.assign(createForm, { name: '', content: '', ttl: '1', proxied: false });
    createErrors.value = {};
    createError.value = '';
    creatorOpen.value = true;
  }

  function closeCreator(open: boolean) {
    if (!open && !creating.value) {
      creatorOpen.value = false;
    }
  }

  function handleCreateProxyChange() {
    if (createForm.proxied) {
      createForm.ttl = '1';
    }
    delete createErrors.value.ttl;
  }

  async function createRecord() {
    const projectId = projectStore.activeProjectId;
    const zone = zoneId.value;
    if (creating.value || !creatorOpen.value || !projectId || !zone) {
      return;
    }
    createErrors.value = validateDNSCreateForm(createForm);
    if (Object.keys(createErrors.value).length) {
      return;
    }
    const scope = scopeVersion;
    const version = ++saveVersion;
    creating.value = true;
    createError.value = '';
    try {
      await dnsApi.create(projectId, zone, {
        name: createForm.name.trim(),
        content: createForm.content.trim(),
        ttl: createForm.proxied ? 1 : Number(createForm.ttl),
        proxied: createForm.proxied,
      });
      if (
        scopeVersion !== scope ||
        projectStore.activeProjectId !== projectId ||
        zoneId.value !== zone
      ) {
        return;
      }
      creatorOpen.value = false;
      toast.success(t('dns.created'));
      page.value = 1;
      void loadRecords();
    } catch (error) {
      if (
        scopeVersion === scope &&
        projectStore.activeProjectId === projectId &&
        zoneId.value === zone
      ) {
        createError.value = `${errorMessage(error)} ${t('dns.verifyAfterFailure')}`;
      }
    } finally {
      if (saveVersion === version) {
        creating.value = false;
      }
    }
  }

  function openDelete(record: RecordResp) {
    pendingDelete.value = record;
    deleteError.value = '';
  }

  function closeDelete(open: boolean) {
    if (!open && !deleting.value) {
      pendingDelete.value = null;
    }
  }

  async function deleteRecord() {
    const record = pendingDelete.value;
    const projectId = projectStore.activeProjectId;
    const zone = zoneId.value;
    if (deleting.value || !record || !projectId || !zone) {
      return;
    }
    const scope = scopeVersion;
    const version = ++saveVersion;
    deleting.value = true;
    deleteError.value = '';
    try {
      await dnsApi.delete(projectId, zone, record.id);
      if (
        scopeVersion !== scope ||
        projectStore.activeProjectId !== projectId ||
        zoneId.value !== zone
      ) {
        return;
      }
      pendingDelete.value = null;
      toast.success(t('dns.deleted'));
      if (records.value.length === 1 && page.value > 1) {
        page.value--;
      }
      void loadRecords();
    } catch (error) {
      if (
        scopeVersion === scope &&
        projectStore.activeProjectId === projectId &&
        zoneId.value === zone
      ) {
        deleteError.value = `${errorMessage(error)} ${t('dns.verifyAfterFailure')}`;
      }
    } finally {
      if (saveVersion === version) {
        deleting.value = false;
      }
    }
  }

  onBeforeUnmount(() => {
    ++scopeVersion;
    ++recordVersion;
    ++saveVersion;
  });

  watch(
    () => projectStore.activeProjectId,
    () => {
      zoneId.value = '';
      void loadProject();
    },
    { immediate: true }
  );
</script>
