<template>
  <div class="space-y-6">
    <div class="app-toolbar-simple" aria-label="角色工具栏">
      <SearchControl
        v-model="searchText"
        class="shrink-0"
        :placeholder="t('roleManagement.searchPlaceholder')"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <div v-if="canWriteRoles" class="flex items-center gap-3">
        <button class="app-button-primary px-5" @click="openCreateDialog">
          <Plus class="size-4" />
          {{ t('roleManagement.create') }}
        </button>
      </div>
    </div>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || t('roleManagement.loadFailed') }}</p>
      </div>
      <AppEmptyState v-else-if="roles.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[760px]">
          <colgroup>
            <col class="w-[20%]" />
            <col class="w-[20%]" />
            <col class="w-[32%]" />
            <col class="w-[16%]" />
            <col class="w-[12%]" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ t('roleManagement.code') }}</th>
              <th>{{ t('common.name') }}</th>
              <th>{{ t('common.description') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th v-if="canWriteRoles">{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="role in roles" :key="role.id">
              <td class="max-w-0 truncate text-foreground" :title="role.code">
                <router-link :to="`/role/${role.id}`" class="app-link">
                  {{ role.code }}
                </router-link>
              </td>
              <td class="max-w-0 truncate" :title="role.name">
                {{ role.name }}
              </td>
              <td class="max-w-0 truncate text-foreground" :title="role.description || undefined">
                {{ role.description || '-' }}
              </td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(role.created_at) }}
              </td>
              <td v-if="canWriteRoles" class="whitespace-nowrap">
                <div class="flex items-center gap-3">
                  <button class="app-link" @click="openEditDialog(role)">
                    {{ t('common.edit') }}
                  </button>
                  <button class="app-link-danger" @click="openConfirmDialog(role)">
                    {{ t('common.delete') }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <ListPagination
        :current="pagination.current"
        :page-size="pagination.pageSize"
        :total="pagination.total"
        :total-pages="totalPages"
        @change-page="goPage"
        @change-page-size="handlePageSizeChange"
      />
    </div>

    <AppDialog
      v-model:open="isDialogOpen"
      :title="editingRole ? t('roleManagement.edit') : t('roleManagement.create')"
    >
      <form class="space-y-4" novalidate @submit.prevent="handleSave">
        <div class="space-y-1.5">
          <label class="app-field-label block" for="role-code">
            {{ t('roleManagement.code') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="role-code"
            v-model="form.code"
            type="text"
            class="app-input"
            :class="formErrors.code ? 'app-input-error' : ''"
            maxlength="50"
            pattern="[A-Za-z0-9_-]+"
            required
            :aria-invalid="formErrors.code ? 'true' : undefined"
            @input="formErrors.code = ''"
          />
          <p v-if="formErrors.code" class="app-field-error" role="alert">
            {{ formErrors.code }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="role-name">
            {{ t('common.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="role-name"
            v-model="form.name"
            type="text"
            class="app-input"
            :class="formErrors.name ? 'app-input-error' : ''"
            maxlength="100"
            required
            :aria-invalid="formErrors.name ? 'true' : undefined"
            @input="formErrors.name = ''"
          />
          <p v-if="formErrors.name" class="app-field-error" role="alert">
            {{ formErrors.name }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="role-description">
            {{ t('common.description') }}
          </label>
          <textarea
            id="role-description"
            v-model="form.description"
            class="app-input min-h-24"
            maxlength="500"
          />
        </div>
        <button type="submit" class="sr-only" tabindex="-1" aria-hidden="true"></button>
      </form>
      <p v-if="submitError" class="app-field-error mt-3" role="alert">
        {{ submitError }}
      </p>
      <template #footer>
        <AppDialogActions :busy="operating" @cancel="isDialogOpen = false" @confirm="handleSave" />
      </template>
    </AppDialog>

    <AppDialog v-model:open="confirmDialogOpen" :title="t('roleManagement.delete')">
      <p class="text-sm text-foreground">
        {{ t('roleManagement.deleteConfirm') }}
        <span>{{ confirmAction?.role.name }}</span>
      </p>
      <p v-if="confirmSubmitError" class="app-field-error mt-3" role="alert">
        {{ confirmSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          variant="destructive"
          @cancel="confirmAction = null"
          @confirm="handleConfirm"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { Plus } from '@lucide/vue';
  import { computed, nextTick, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { roleApi } from '@/api/role/role';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useAuthStore } from '@/stores/auth';
  import { PERMISSIONS } from '@/constants/permissions';
  import type { RoleResp } from '@/gen/proto/orbit/v1/role/role';
  import { formatTime } from '@/utils/time';

  const { t } = useI18n();
  const router = useRouter();
  const toast = useToast();
  const authStore = useAuthStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const roles = ref<RoleResp[]>([]);
  const searchText = ref('');
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  type ConfirmAction = { role: RoleResp };

  const editingRole = ref<RoleResp | null>(null);
  const confirmAction = ref<ConfirmAction | null>(null);
  const isDialogOpen = ref(false);
  const form = reactive({ code: '', name: '', description: '' });
  const formErrors = reactive({ code: '', name: '' });
  const submitError = ref('');
  const confirmSubmitError = ref('');
  const canWriteRoles = computed(() => authStore.hasPermission(PERMISSIONS.ROLE_WRITE));
  const totalPages = computed(() => Math.max(1, Math.ceil(pagination.total / pagination.pageSize)));
  const confirmDialogOpen = computed({
    get: () => confirmAction.value !== null,
    set: (open) => {
      if (!open) {
        confirmAction.value = null;
        confirmSubmitError.value = '';
      }
    },
  });

  function resetForm(role?: RoleResp) {
    form.code = role?.code ?? '';
    form.name = role?.name ?? '';
    form.description = role?.description ?? '';
    Object.assign(formErrors, { code: '', name: '' });
    submitError.value = '';
  }

  function validateForm() {
    const code = form.code.trim();
    formErrors.code = !code
      ? t('roleManagement.codeRequired')
      : /^[A-Za-z0-9_-]+$/.test(code)
        ? ''
        : t('roleManagement.codeInvalid');
    formErrors.name = form.name.trim() ? '' : t('roleManagement.nameRequired');
    return !formErrors.code && !formErrors.name;
  }

  async function fetchRoles() {
    try {
      await execute(async () => {
        const res = await roleApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
        });
        roles.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error(t('roleManagement.loadFailed'));
    }
  }

  function handleSearch() {
    pagination.current = 1;
    fetchRoles();
  }

  function goPage(page: number) {
    pagination.current = page;
    fetchRoles();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchRoles();
  }

  function openCreateDialog() {
    editingRole.value = null;
    resetForm();
    isDialogOpen.value = true;
  }

  function openEditDialog(role: RoleResp) {
    editingRole.value = role;
    resetForm(role);
    isDialogOpen.value = true;
  }

  async function openConfirmDialog(role: RoleResp) {
    confirmAction.value = { role };
    confirmSubmitError.value = '';
    (document.activeElement as HTMLElement)?.blur();
    await nextTick();
  }

  async function handleSave() {
    submitError.value = '';
    if (!validateForm()) {
      return;
    }
    try {
      await executeOp(async () => {
        const payload = {
          code: form.code.trim(),
          name: form.name.trim(),
          description: form.description.trim() || undefined,
          permission_codes: editingRole.value?.permission_codes ?? [],
        };
        if (editingRole.value) {
          await roleApi.update(editingRole.value.id, payload);
          await authStore.fetchUser();
          toast.success(t('roleManagement.updated'));
          isDialogOpen.value = false;
          await fetchRoles();
        } else {
          const role = await roleApi.create(payload);
          toast.success(t('roleManagement.created'));
          isDialogOpen.value = false;
          router.push({ name: 'RoleDetail', params: { id: role.id } });
        }
      });
    } catch (e: unknown) {
      submitError.value = e instanceof Error ? e.message : t('roleManagement.saveFailed');
    }
  }

  async function handleConfirm() {
    const action = confirmAction.value;
    if (!action) {
      return;
    }
    confirmSubmitError.value = '';
    try {
      await executeOp(async () => {
        await roleApi.delete(action.role.id);
        await authStore.fetchUser();
        toast.success(t('roleManagement.deleted'));
        confirmAction.value = null;
        await fetchRoles();
      });
    } catch (e: unknown) {
      confirmSubmitError.value = e instanceof Error ? e.message : t('roleManagement.deleteFailed');
    }
  }

  onMounted(() => {
    fetchRoles();
  });
</script>
