<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-xl font-semibold text-foreground">{{ user?.username ?? '用户详情' }}</h1>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="user && canWriteUsers"
          class="app-button-primary h-9 px-3"
          :disabled="operating"
          @click="openEditModal"
        >
          <Pencil class="size-4" />
          {{ t('common.edit') }}
        </button>
        <button
          v-if="user && canWriteUsers && user.status === 'enabled'"
          class="app-button-danger h-9 px-3"
          :disabled="operating"
          @click="openDisableModal"
        >
          <Ban class="size-4" />
          {{ t('userManagement.disable') }}
        </button>
        <button
          v-if="user && canWriteUsers && user.status === 'disabled'"
          class="app-button h-9 px-3"
          :disabled="operating"
          @click="handleEnable"
        >
          <CheckCircle class="size-4" />
          {{ t('userManagement.enable') }}
        </button>
        <button
          v-if="user && canWriteUsers"
          class="app-button-danger h-9 px-3"
          :disabled="operating"
          @click="openDeleteModal"
        >
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="$router.push('/users')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <div class="app-surface">
      <div class="app-section-header">
        <h2 class="font-semibold text-foreground">{{ t('userManagement.basicInfo') }}</h2>
      </div>

      <AppSpinner v-if="loading" class="px-5 py-10" />
      <dl
        v-else-if="user"
        class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2"
      >
        <div class="flex gap-2">
          <dt class="w-32 shrink-0 text-muted-foreground">{{ t('userManagement.username') }}</dt>
          <dd class="text-foreground">{{ user.username }}</dd>
        </div>
        <div class="flex gap-2">
          <dt class="w-32 shrink-0 text-muted-foreground">{{ t('userManagement.email') }}</dt>
          <dd class="text-foreground">{{ user.email || '-' }}</dd>
        </div>
        <div class="flex gap-2">
          <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.status') }}</dt>
          <dd>
            <AppBadge variant="status" :tone="user.status === 'enabled' ? 'success' : 'default'">
              {{ user.status }}
            </AppBadge>
          </dd>
        </div>
        <div class="flex gap-2">
          <dt class="w-32 shrink-0 text-muted-foreground">{{ t('userManagement.authSource') }}</dt>
          <dd><AppBadge variant="pill">{{ user.auth_source }}</AppBadge></dd>
        </div>
        <div class="flex gap-2">
          <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.createdAt') }}</dt>
          <dd class="text-muted-foreground">{{ formatTime(user.created_at) }}</dd>
        </div>
        <div class="flex gap-2">
          <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.updatedAt') }}</dt>
          <dd class="text-muted-foreground">{{ formatTime(user.updated_at) }}</dd>
        </div>
        <div class="flex gap-2">
          <dt class="w-32 shrink-0 text-muted-foreground">{{ t('userManagement.lastLoginAt') }}</dt>
          <dd class="text-muted-foreground">
            {{ user.last_login_at ? formatTime(user.last_login_at) : '-' }}
          </dd>
        </div>
      </dl>
    </div>

    <div class="app-surface">
      <div class="app-section-header flex flex-wrap items-center justify-between gap-3">
        <h2 class="font-semibold text-foreground">{{ t('userManagement.rolesAndPermissions') }}</h2>
        <button
          v-if="user && canAssignRoles"
          class="app-button-primary h-8 px-3"
          :disabled="operating"
          @click="openRoleModal"
        >
          <Pencil class="size-4" />
          {{ t('common.edit') }}
        </button>
      </div>

      <div v-if="user && user.role_items.length > 0" class="px-5 py-4">
        <TabsRoot :default-value="user.role_items[0]?.id">
          <TabsList class="flex gap-1 border-b border-border">
            <TabsTrigger
              v-for="role in user.role_items"
              :key="role.id"
              :value="role.id"
              class="px-4 py-2 text-sm font-medium text-muted-foreground hover:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary data-[state=active]:text-foreground"
            >
              {{ role.name }}
            </TabsTrigger>
          </TabsList>
          <TabsContent v-for="role in user.role_items" :key="role.id" :value="role.id" class="pt-4">
            <div v-if="role.permission_codes.length > 0" class="flex flex-wrap gap-2">
              <AppBadge v-for="permission in role.permission_codes" :key="permission">
                {{ permission }}
              </AppBadge>
            </div>
            <p v-else class="text-sm text-muted-foreground">{{ t('common.noData') }}</p>
          </TabsContent>
        </TabsRoot>
      </div>
      <div v-else class="px-5 py-4">
        <p class="text-sm text-muted-foreground">{{ t('common.noData') }}</p>
      </div>
    </div>

    <AppDialog
      v-model:open="isEditModalOpen"
      :title="t('userManagement.edit')"
      width-class="w-[min(600px,calc(100vw-32px))]"
    >
      <form id="user-edit-form" class="space-y-5" @submit.prevent="handleEditOk">
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
            :disabled="operating"
          />
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
            :disabled="operating"
          />
          <p class="app-field-hint">
            {{ t('common.emptyKeepUnchanged') }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('common.status') }}</label>
          <RawValueSelect v-model="form.status" :values="userStatusValues" :disabled="operating" />
        </div>
      </form>
      <template #footer>
        <button class="app-button" :disabled="operating" @click="isEditModalOpen = false">
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

    <AppDialog
      v-model:open="isRoleModalOpen"
      :title="t('userManagement.roles')"
      width-class="w-[min(600px,calc(100vw-32px))]"
    >
      <form id="user-role-form" class="space-y-4" @submit.prevent="handleRoleOk">
        <div class="grid gap-2 sm:grid-cols-2">
          <label
            v-for="role in roleOptions"
            :key="role.id"
            class="flex items-start gap-2 rounded-xl border border-border bg-background px-3 py-2.5 text-sm transition-colors hover:border-primary/40"
          >
            <input
              v-model="roleForm.roleIds"
              type="checkbox"
              :value="role.id"
              class="app-checkbox mt-0.5"
              :disabled="operating"
            />
            <span>
              <span class="block text-foreground">{{ role.name }}</span>
              <span class="block text-xs text-muted-foreground">{{ role.code }}</span>
            </span>
          </label>
        </div>
      </form>
      <template #footer>
        <button class="app-button" :disabled="operating" @click="isRoleModalOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button
          class="app-button-primary"
          type="submit"
          form="user-role-form"
          :disabled="operating"
        >
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDisableModalOpen"
      :title="t('userManagement.disable')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('userManagement.disableConfirm') }}
        <strong>{{ user?.username }}</strong>
      </p>
      <template #footer>
        <button class="app-button" :disabled="operating" @click="isDisableModalOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-danger" :disabled="operating" @click="handleDisable">
          {{ t('userManagement.disable') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteModalOpen"
      :title="t('common.delete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('userManagement.deleteConfirm') }}
        <strong>{{ user?.username }}</strong>
      </p>
      <template #footer>
        <button class="app-button" :disabled="operating" @click="isDeleteModalOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-danger" :disabled="operating" @click="handleDelete">
          {{ t('common.delete') }}
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Ban, CheckCircle, Pencil, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { TabsContent, TabsList, TabsRoot, TabsTrigger } from 'reka-ui';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { roleApi } from '@/api/role/role';
  import { userApi } from '@/api/user/user';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useAuthStore } from '@/stores/auth';
  import { PERMISSIONS } from '@/constants/permissions';
  import type { RoleResp } from '@/gen/proto/orbit/v1/role/role';
  import type { UserResp } from '@/gen/proto/orbit/v1/user/user';
  import { formatTime } from '@/utils/time';

  const props = defineProps<{ id: string }>();
  const { t } = useI18n();
  const $router = useRouter();
  const toast = useToast();
  const authStore = useAuthStore();
  const { loading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const user = ref<UserResp>();
  const roleOptions = ref<RoleResp[]>([]);
  const isEditModalOpen = ref(false);
  const isRoleModalOpen = ref(false);
  const isDisableModalOpen = ref(false);
  const isDeleteModalOpen = ref(false);
  const form = reactive({
    username: '',
    password: '',
    status: '',
  });
  const roleForm = reactive<{ roleIds: string[] }>({ roleIds: [] });

  const canWriteUsers = computed(() => authStore.hasPermission(PERMISSIONS.USER_WRITE));
  const canAssignRoles = computed(
    () =>
      authStore.hasPermission(PERMISSIONS.ROLE_READ) &&
      authStore.hasPermission(PERMISSIONS.ROLE_WRITE)
  );
  const userStatusValues = ['enabled', 'disabled'];

  async function fetchUser() {
    try {
      await execute(async () => {
        user.value = await userApi.get(props.id);
      });
    } catch {
      toast.error(t('userManagement.loadFailed'));
    }
  }

  async function fetchRoleOptions() {
    if (!canAssignRoles.value) {
      return;
    }
    try {
      const res = await roleApi.list({ page: 1, per_page: 100 });
      roleOptions.value = res.items;
    } catch {
      toast.error(t('userManagement.loadRolesFailed'));
    }
  }

  function openEditModal() {
    form.username = user.value?.username ?? '';
    form.password = '';
    if (!user.value) {
      throw new Error('User detail is not loaded');
    }
    form.status = user.value.status;
    isEditModalOpen.value = true;
  }

  function openRoleModal() {
    roleForm.roleIds = user.value?.role_items.map((role) => role.id) ?? [];
    isRoleModalOpen.value = true;
  }

  async function handleEditOk() {
    try {
      await executeOp(async () => {
        await userApi.update(props.id, {
          username: form.username.trim(),
          password: form.password.trim() || undefined,
          status: form.status,
        });
        if (props.id === authStore.user?.id) {
          await authStore.fetchUser();
        }
        toast.success(t('userManagement.updated'));
        isEditModalOpen.value = false;
        await fetchUser();
      });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('userManagement.saveFailed'));
    }
  }

  async function handleRoleOk() {
    try {
      await executeOp(async () => {
        await userApi.updateRoles(props.id, { role_ids: roleForm.roleIds });
        if (props.id === authStore.user?.id) {
          await authStore.fetchUser();
        }
        toast.success(t('userManagement.updated'));
        isRoleModalOpen.value = false;
        await fetchUser();
      });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('userManagement.saveFailed'));
    }
  }

  function openDisableModal() {
    isDisableModalOpen.value = true;
  }

  async function handleDisable() {
    try {
      await executeOp(async () => {
        await userApi.disable(props.id, {});
        toast.success(t('userManagement.disabledToast'));
        isDisableModalOpen.value = false;
        await fetchUser();
      });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('userManagement.disableFailed'));
    }
  }

  async function handleEnable() {
    try {
      await executeOp(async () => {
        await userApi.enable(props.id, {});
        toast.success(t('userManagement.enabledToast'));
        await fetchUser();
      });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('userManagement.enableFailed'));
    }
  }

  function openDeleteModal() {
    isDeleteModalOpen.value = true;
  }

  async function handleDelete() {
    try {
      await executeOp(async () => {
        await userApi.delete(props.id);
        toast.success(t('userManagement.deleted'));
        $router.push('/users');
      });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('userManagement.deleteFailed'));
    }
  }

  onMounted(() => {
    fetchUser();
    fetchRoleOptions();
  });
</script>
