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
      <AppSpinner v-if="status === 'loading'" class="py-16" />
      <AppEmptyState v-else-if="users.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-table-list min-w-[1080px]">
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
                <router-link :to="`/users/${user.id}`" class="app-link">
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
                <AppBadge v-if="user.status === 'enabled'" variant="status" tone="success">
                  {{ t('userManagement.enabled') }}
                </AppBadge>
                <AppBadge v-else variant="status" tone="default">
                  {{ t('userManagement.disabled') }}
                </AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatAuthSource(user.auth_source) }}
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
      <form id="user-form" class="space-y-4" @submit.prevent="handleSave">
        <div class="space-y-1.5">
          <label class="app-field-label block" for="username">
            {{ t('userManagement.username') }}
          </label>
          <input
            id="username"
            v-model="form.username"
            type="text"
            class="app-input"
            maxlength="50"
            required
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="email">{{ t('userManagement.email') }}</label>
          <input id="email" v-model="form.email" type="email" class="app-input" maxlength="255" />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="password">
            {{ t('userManagement.password') }}
          </label>
          <input
            id="password"
            v-model="form.password"
            type="password"
            class="app-input"
            minlength="6"
            maxlength="255"
            required
          />
        </div>
      </form>
      <template #footer>
        <button class="app-button" @click="isDialogOpen = false">{{ t('common.cancel') }}</button>
        <button class="app-button-primary" type="submit" form="user-form" :disabled="operating">
          {{ t('userManagement.create') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isEditDialogOpen"
      :title="t('userManagement.edit')"
      width-class="w-[min(600px,calc(100vw-32px))]"
    >
      <form id="user-edit-form" class="space-y-5" @submit.prevent="handleEditSave">
        <div class="space-y-1.5">
          <label class="app-field-label block" for="edit-username">
            {{ t('userManagement.username') }}
          </label>
          <input
            id="edit-username"
            v-model="editForm.username"
            type="text"
            class="app-input"
            maxlength="50"
            required
          />
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
            minlength="6"
            maxlength="255"
          />
          <p class="app-field-hint">
            {{ t('common.emptyKeepUnchanged') }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('common.status') }}</label>
          <SelectControl v-model="editForm.status" :options="userStatusOptions" />
        </div>
      </form>
      <template #footer>
        <button class="app-button" @click="isEditDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button
          class="app-button-primary"
          type="submit"
          form="user-edit-form"
          :disabled="operating"
        >
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog v-model:open="confirmDialogOpen" :title="confirmTitle">
      <p class="text-sm text-foreground">
        {{ confirmMessage }}
        <strong>{{ confirmAction?.user.username }}</strong>
      </p>
      <template #footer>
        <button class="app-button" @click="confirmAction = null">{{ t('common.cancel') }}</button>
        <button class="app-button-danger" :disabled="operating" @click="handleConfirm">
          {{ confirmButtonText }}
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { Plus } from 'lucide-vue-next';
  import { computed, nextTick, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { userApi } from '@/api/user';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useAuthStore } from '@/stores/auth';
  import { PERMISSIONS } from '@/constants/permissions';
  import type { AuthSource } from '@/types/auth';
  import type { UserListResp, UserStatus } from '@/types/user';
  import { formatTime } from '@/utils/time';

  const { t } = useI18n();
  const router = useRouter();
  const toast = useToast();
  const authStore = useAuthStore();
  const { status, execute } = useStatusAsync();
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
  const editForm = reactive<{ username: string; password: string; status: UserStatus }>({
    username: '',
    password: '',
    status: 'enabled',
  });
  const canWriteUsers = computed(() => authStore.hasPermission(PERMISSIONS.USER_WRITE));
  const userStatusOptions = computed(() => [
    { value: 'enabled', label: t('userManagement.enabled') },
    { value: 'disabled', label: t('userManagement.disabled') },
  ]);
  const totalPages = computed(() => Math.max(1, Math.ceil(pagination.total / pagination.pageSize)));
  const confirmDialogOpen = computed({
    get: () => confirmAction.value !== null,
    set: (open) => {
      if (!open) {
        confirmAction.value = null;
      }
    },
  });
  const confirmTitle = computed(() => t('userManagement.disable'));
  const confirmMessage = computed(() => t('userManagement.disableConfirm'));
  const confirmButtonText = computed(() => t('userManagement.disable'));

  function resetForm() {
    form.username = '';
    form.email = '';
    form.password = '';
  }

  function formatAuthSource(authSource: AuthSource) {
    switch (authSource) {
      case 'oauth':
        return t('userManagement.authSourceOAuth');
      case 'password':
        return t('userManagement.authSourcePassword');
      default:
        throw new Error(`Unsupported auth source: ${authSource}`);
    }
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

  async function openConfirmDialog(type: ConfirmAction['type'], user: UserListResp) {
    confirmAction.value = { type, user };
    (document.activeElement as HTMLElement)?.blur();
    await nextTick();
  }

  function updateUserStatus(userId: string, status: UserStatus) {
    users.value = users.value.map((user) => (user.id === userId ? { ...user, status } : user));
  }

  async function handleSave() {
    try {
      await executeOp(async () => {
        const user = await userApi.create({
          username: form.username.trim(),
          email: form.email.trim() || null,
          password: form.password.trim(),
        });
        toast.success(t('userManagement.created'));
        isDialogOpen.value = false;
        router.push({ name: 'UserDetail', params: { id: user.id } });
      });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('userManagement.saveFailed'));
    }
  }

  function openEditDialog(user: UserListResp) {
    editingUser.value = user;
    editForm.username = user.username;
    editForm.password = '';
    editForm.status = user.status;
    isEditDialogOpen.value = true;
  }

  async function handleEditSave() {
    const user = editingUser.value;
    if (!user) {
      return;
    }
    try {
      await executeOp(async () => {
        await userApi.update(user.id, {
          username: editForm.username.trim(),
          password: editForm.password.trim() || null,
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
      toast.error(e instanceof Error ? e.message : t('userManagement.saveFailed'));
    }
  }

  async function handleEnable(user: UserListResp) {
    try {
      await executeOp(async () => {
        await userApi.enable(user.id);
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
    try {
      await executeOp(async () => {
        await userApi.disable(action.user.id);
        updateUserStatus(action.user.id, 'disabled');
        toast.success(t('userManagement.disabledToast'));
        confirmAction.value = null;
      });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('userManagement.disableFailed'));
    }
  }

  onMounted(() => {
    fetchUsers();
  });
</script>
