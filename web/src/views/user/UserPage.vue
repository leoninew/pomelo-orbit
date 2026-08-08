<template>
  <div class="space-y-6">
    <div class="app-toolbar-simple" aria-label="用户工具栏">
      <SearchControl
        v-model="searchText"
        class="shrink-0"
        :placeholder="t('userManagement.searchPlaceholder')"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <div v-if="canWriteUsers" class="flex items-center gap-3">
        <button class="app-button-primary px-5" @click="openCreateDialog">
          <Plus class="size-4" />
          {{ t('userManagement.create') }}
        </button>
      </div>
    </div>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || t('userManagement.loadFailed') }}</p>
      </div>
      <AppEmptyState v-else-if="users.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[1080px]">
          <colgroup>
            <col class="w-[14%]" />
            <col class="w-[18%]" />
            <col class="w-[16%]" />
            <col class="w-[9%]" />
            <col class="w-[9%]" />
            <col class="w-[14%]" />
            <col class="w-[14%]" />
            <col class="w-[6%]" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ t('userManagement.username') }}</th>
              <th>{{ t('userManagement.email') }}</th>
              <th>{{ t('userManagement.roles') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('userManagement.authSource') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th>{{ t('userManagement.lastLoginAt') }}</th>
              <th v-if="canWriteUsers">{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in users" :key="user.id">
              <td class="max-w-0 truncate text-foreground" :title="user.username">
                <router-link :to="`/user/${user.id}`" class="app-link">
                  {{ user.username }}
                </router-link>
              </td>
              <td class="max-w-0 truncate text-foreground" :title="user.email || undefined">
                {{ user.email || '-' }}
              </td>
              <td class="max-w-0 truncate text-foreground" :title="formatRoleNames(user)">
                {{ formatRoleNames(user) || '-' }}
              </td>
              <td>
                <AppBadge
                  variant="status"
                  :tone="user.status === 'enabled' ? 'success' : 'default'"
                >
                  {{ user.status }}
                </AppBadge>
              </td>
              <td>
                <AppBadge variant="pill">{{ user.auth_source }}</AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(user.created_at) }}
              </td>
              <td class="whitespace-nowrap text-foreground">
                {{ user.last_login_at ? formatTime(user.last_login_at) : '-' }}
              </td>
              <td v-if="canWriteUsers" class="whitespace-nowrap">
                <div class="flex items-center gap-3">
                  <button class="app-link" @click="openEditDialog(user)">
                    {{ t('common.edit') }}
                  </button>
                  <button
                    v-if="user.status === 'enabled'"
                    class="app-link-danger"
                    @click="openConfirmDialog('disable', user)"
                  >
                    {{ t('userManagement.disable') }}
                  </button>
                  <button v-else class="app-link" @click="handleEnable(user)">
                    {{ t('userManagement.enable') }}
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

    <AppDialog v-model:open="isDialogOpen" :title="t('userManagement.create')">
      <form class="space-y-4" novalidate @submit.prevent="handleSave">
        <div class="space-y-1.5">
          <label class="app-field-label block" for="username">
            {{ t('userManagement.username') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="username"
            v-model="form.username"
            type="text"
            class="app-input"
            :class="formErrors.username ? 'app-input-error' : ''"
            maxlength="50"
            required
            :aria-invalid="formErrors.username ? 'true' : undefined"
            @input="formErrors.username = ''"
          />
          <p v-if="formErrors.username" class="app-field-error" role="alert">
            {{ formErrors.username }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="email">{{ t('userManagement.email') }}</label>
          <input
            id="email"
            v-model="form.email"
            type="email"
            class="app-input"
            :class="formErrors.email ? 'app-input-error' : ''"
            maxlength="255"
            :aria-invalid="formErrors.email ? 'true' : undefined"
            @input="formErrors.email = ''"
          />
          <p v-if="formErrors.email" class="app-field-error" role="alert">
            {{ formErrors.email }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="password">
            {{ t('userManagement.password') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="password"
            v-model="form.password"
            type="password"
            class="app-input"
            :class="formErrors.password ? 'app-input-error' : ''"
            minlength="6"
            maxlength="255"
            required
            :aria-invalid="formErrors.password ? 'true' : undefined"
            @input="formErrors.password = ''"
          />
          <p v-if="formErrors.password" class="app-field-error" role="alert">
            {{ formErrors.password }}
          </p>
        </div>
        <button type="submit" class="sr-only" tabindex="-1" aria-hidden="true"></button>
      </form>
      <p v-if="createSubmitError" class="app-field-error mt-3" role="alert">
        {{ createSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions :busy="operating" @cancel="closeCreateDialog" @confirm="handleSave" />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isEditDialogOpen"
      :title="t('userManagement.edit')"
      width-class="w-[min(600px,calc(100vw-32px))]"
    >
      <form class="space-y-5" novalidate @submit.prevent="handleEditSave">
        <div class="space-y-1.5">
          <label class="app-field-label block" for="edit-username">
            {{ t('userManagement.username') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="edit-username"
            v-model="editForm.username"
            type="text"
            class="app-input"
            :class="editFormErrors.username ? 'app-input-error' : ''"
            maxlength="50"
            required
            :aria-invalid="editFormErrors.username ? 'true' : undefined"
            @input="editFormErrors.username = ''"
          />
          <p v-if="editFormErrors.username" class="app-field-error" role="alert">
            {{ editFormErrors.username }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="edit-password">
            {{ t('userManagement.password') }}
          </label>
          <input
            id="edit-password"
            v-model="editForm.password"
            type="password"
            class="app-input"
            :class="editFormErrors.password ? 'app-input-error' : ''"
            minlength="6"
            maxlength="255"
            :aria-invalid="editFormErrors.password ? 'true' : undefined"
            @input="editFormErrors.password = ''"
          />
          <p v-if="editFormErrors.password" class="app-field-error" role="alert">
            {{ editFormErrors.password }}
          </p>
          <p class="app-field-hint">
            {{ t('common.emptyKeepUnchanged') }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('common.status') }}
            <span class="text-destructive">*</span>
          </label>
          <RawValueSelect
            v-model="editForm.status"
            :values="userStatusValues"
            :invalid="Boolean(editFormErrors.status)"
            @update:model-value="editFormErrors.status = ''"
          />
          <p v-if="editFormErrors.status" class="app-field-error" role="alert">
            {{ editFormErrors.status }}
          </p>
        </div>
        <button type="submit" class="sr-only" tabindex="-1" aria-hidden="true"></button>
      </form>
      <p v-if="editSubmitError" class="app-field-error mt-3" role="alert">
        {{ editSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions :busy="operating" @cancel="closeEditDialog" @confirm="handleEditSave" />
      </template>
    </AppDialog>

    <AppDialog v-model:open="confirmDialogOpen" :title="confirmTitle">
      <p class="text-sm text-foreground">
        {{ confirmMessage }}
        <span>{{ confirmAction?.user.username }}</span>
      </p>
      <p v-if="confirmSubmitError" class="app-field-error mt-3" role="alert">
        {{ confirmSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          variant="destructive"
          @cancel="closeConfirmDialog"
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
  import { userApi } from '@/api/user/user';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useAuthStore } from '@/stores/auth';
  import { PERMISSIONS } from '@/constants/permissions';
  import type { UserListResp } from '@/gen/proto/orbit/v1/user/user';
  import { formatTime } from '@/utils/time';

  const { t } = useI18n();
  const router = useRouter();
  const toast = useToast();
  const authStore = useAuthStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const users = ref<UserListResp[]>([]);
  const searchText = ref('');
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  type ConfirmAction = { type: 'disable'; user: UserListResp };

  const confirmAction = ref<ConfirmAction | null>(null);
  const isDialogOpen = ref(false);
  const isEditDialogOpen = ref(false);
  const editingUser = ref<UserListResp | null>(null);
  const form = reactive({ username: '', email: '', password: '' });
  const formErrors = reactive({ username: '', email: '', password: '' });
  const editForm = reactive({
    username: '',
    password: '',
    status: '',
  });
  const editFormErrors = reactive({ username: '', password: '', status: '' });
  const createSubmitError = ref('');
  const editSubmitError = ref('');
  const confirmSubmitError = ref('');
  const canWriteUsers = computed(() => authStore.hasPermission(PERMISSIONS.USER_WRITE));
  const userStatusValues = ['enabled', 'disabled'];
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
  const confirmTitle = computed(() => t('userManagement.disable'));
  const confirmMessage = computed(() => t('userManagement.disableConfirm'));

  function resetForm() {
    form.username = '';
    form.email = '';
    form.password = '';
    Object.assign(formErrors, { username: '', email: '', password: '' });
    createSubmitError.value = '';
  }

  function validateCreateForm() {
    formErrors.username = form.username.trim() ? '' : t('userManagement.usernameRequired');
    formErrors.email =
      !form.email.trim() || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email.trim())
        ? ''
        : t('userManagement.emailInvalid');
    formErrors.password =
      form.password.trim().length >= 6 ? '' : t('userManagement.passwordTooShort');
    return !formErrors.username && !formErrors.email && !formErrors.password;
  }

  function validateEditForm() {
    editFormErrors.username = editForm.username.trim() ? '' : t('userManagement.usernameRequired');
    editFormErrors.password =
      !editForm.password || editForm.password.trim().length >= 6
        ? ''
        : t('userManagement.passwordTooShort');
    editFormErrors.status = userStatusValues.includes(editForm.status)
      ? ''
      : t('userManagement.statusRequired');
    return !editFormErrors.username && !editFormErrors.password && !editFormErrors.status;
  }

  function formatRoleNames(user: UserListResp) {
    return user.role_items.map((role) => role.name).join(', ');
  }

  async function fetchUsers() {
    try {
      await execute(async () => {
        const res = await userApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
        });
        users.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error(t('userManagement.loadFailed'));
    }
  }

  function handleSearch() {
    pagination.current = 1;
    fetchUsers();
  }

  function goPage(page: number) {
    pagination.current = page;
    fetchUsers();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchUsers();
  }

  function openCreateDialog() {
    resetForm();
    isDialogOpen.value = true;
  }

  function closeCreateDialog() {
    createSubmitError.value = '';
    isDialogOpen.value = false;
  }

  async function openConfirmDialog(type: 'disable', user: UserListResp) {
    confirmAction.value = { type, user };
    confirmSubmitError.value = '';
    (document.activeElement as HTMLElement)?.blur();
    await nextTick();
  }

  function updateUserStatus(userId: string, status: string) {
    users.value = users.value.map((user) => (user.id === userId ? { ...user, status } : user));
  }

  async function handleSave() {
    createSubmitError.value = '';
    if (!validateCreateForm()) {
      return;
    }
    try {
      await executeOp(async () => {
        const user = await userApi.create({
          username: form.username.trim(),
          email: form.email.trim() || undefined,
          password: form.password.trim(),
        });
        toast.success(t('userManagement.created'));
        isDialogOpen.value = false;
        router.push({ name: 'UserDetail', params: { id: user.id } });
      });
    } catch (e: unknown) {
      createSubmitError.value = e instanceof Error ? e.message : t('userManagement.saveFailed');
    }
  }

  function openEditDialog(user: UserListResp) {
    editingUser.value = user;
    editForm.username = user.username;
    editForm.password = '';
    editForm.status = user.status;
    Object.assign(editFormErrors, { username: '', password: '', status: '' });
    editSubmitError.value = '';
    isEditDialogOpen.value = true;
  }

  function closeEditDialog() {
    editSubmitError.value = '';
    isEditDialogOpen.value = false;
  }

  async function handleEditSave() {
    const user = editingUser.value;
    if (!user) {
      return;
    }
    editSubmitError.value = '';
    if (!validateEditForm()) {
      return;
    }
    try {
      await executeOp(async () => {
        await userApi.update(user.id, {
          username: editForm.username.trim(),
          password: editForm.password.trim() || undefined,
          status: editForm.status,
        });
        updateUserStatus(user.id, editForm.status);
        if (user.id === authStore.user?.id) {
          await authStore.fetchUser();
        }
        toast.success(t('userManagement.updated'));
        isEditDialogOpen.value = false;
      });
    } catch (e: unknown) {
      editSubmitError.value = e instanceof Error ? e.message : t('userManagement.saveFailed');
    }
  }

  async function handleEnable(user: UserListResp) {
    try {
      await executeOp(async () => {
        await userApi.enable(user.id, {});
        updateUserStatus(user.id, 'enabled');
        toast.success(t('userManagement.enabledToast'));
      });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('userManagement.enableFailed'));
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
        await userApi.disable(action.user.id, {});
        updateUserStatus(action.user.id, 'disabled');
        toast.success(t('userManagement.disabledToast'));
        confirmAction.value = null;
      });
    } catch (e: unknown) {
      confirmSubmitError.value = e instanceof Error ? e.message : t('userManagement.disableFailed');
    }
  }

  function closeConfirmDialog() {
    confirmAction.value = null;
    confirmSubmitError.value = '';
  }

  onMounted(() => {
    fetchUsers();
  });
</script>
