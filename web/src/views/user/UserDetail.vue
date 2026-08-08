<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">
          {{ user?.username ?? '用户详情' }}
        </h1>
        <DetailHeaderMeta v-if="user">
          <AppBadge variant="status" :tone="user.status === 'enabled' ? 'success' : 'default'">
            {{ user.status }}
          </AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
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

    <DetailInfoCard
      :title="t('userManagement.basicInfo')"
      :loading="loading"
      :editable="Boolean(canWriteUsers && user)"
      :disabled="operating"
      @edit="openEditModal"
    >
      <dl v-if="user" class="app-detail-info-grid">
        <div class="flex gap-2">
          <dt>{{ t('userManagement.username') }}</dt>
          <dd class="text-foreground">{{ user.username }}</dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('userManagement.email') }}</dt>
          <dd class="text-foreground">{{ user.email || '-' }}</dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('common.status') }}</dt>
          <dd>
            <AppBadge variant="status" :tone="user.status === 'enabled' ? 'success' : 'default'">
              {{ user.status }}
            </AppBadge>
          </dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('userManagement.authSource') }}</dt>
          <dd>
            <AppBadge variant="pill">{{ user.auth_source }}</AppBadge>
          </dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('common.createdAt') }}</dt>
          <dd class="text-muted-foreground">{{ formatTime(user.created_at) }}</dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('common.updatedAt') }}</dt>
          <dd class="text-muted-foreground">{{ formatTime(user.updated_at) }}</dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('userManagement.lastLoginAt') }}</dt>
          <dd class="text-muted-foreground">
            {{ user.last_login_at ? formatTime(user.last_login_at) : '-' }}
          </dd>
        </div>
      </dl>
    </DetailInfoCard>
    <UserRolesCard
      :user="user"
      :editable="canAssignRoles"
      :disabled="operating"
      @edit="openRoleModal"
    />

    <AppDialog
      v-model:open="isEditModalOpen"
      :title="t('userManagement.edit')"
      width-class="w-[min(600px,calc(100vw-32px))]"
    >
      <form class="space-y-5" novalidate @submit.prevent="handleEditOk">
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
            :disabled="operating"
            :aria-invalid="formErrors.username ? 'true' : undefined"
            @input="formErrors.username = ''"
          />
          <p v-if="formErrors.username" class="app-field-error" role="alert">
            {{ formErrors.username }}
          </p>
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
            :class="formErrors.password ? 'app-input-error' : ''"
            minlength="6"
            maxlength="255"
            :disabled="operating"
            :aria-invalid="formErrors.password ? 'true' : undefined"
            @input="formErrors.password = ''"
          />
          <p v-if="formErrors.password" class="app-field-error" role="alert">
            {{ formErrors.password }}
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
            v-model="form.status"
            :values="userStatusValues"
            :disabled="operating"
            :invalid="Boolean(formErrors.status)"
            @update:model-value="formErrors.status = ''"
          />
          <p v-if="formErrors.status" class="app-field-error" role="alert">
            {{ formErrors.status }}
          </p>
        </div>
        <button type="submit" class="sr-only" tabindex="-1" aria-hidden="true"></button>
      </form>
      <p v-if="editSubmitError" class="app-field-error mt-3" role="alert">
        {{ editSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          @cancel="isEditModalOpen = false"
          @confirm="handleEditOk"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isRoleModalOpen"
      :title="t('userManagement.roles')"
      width-class="w-[min(600px,calc(100vw-32px))]"
    >
      <form class="space-y-4" novalidate @submit.prevent="handleRoleOk">
        <div class="grid gap-2 sm:grid-cols-2">
          <label v-for="role in roleOptions" :key="role.id" class="app-detail-list-item">
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
        <button type="submit" class="sr-only" tabindex="-1" aria-hidden="true"></button>
      </form>
      <p v-if="roleSubmitError" class="app-field-error mt-3" role="alert">
        {{ roleSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          @cancel="isRoleModalOpen = false"
          @confirm="handleRoleOk"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDisableModalOpen"
      :title="t('userManagement.disable')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('userManagement.disableConfirm') }}
        <span>{{ user?.username }}</span>
      </p>
      <p v-if="disableSubmitError" class="app-field-error mt-3" role="alert">
        {{ disableSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          variant="destructive"
          @cancel="isDisableModalOpen = false"
          @confirm="handleDisable"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteModalOpen"
      :title="t('common.delete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('userManagement.deleteConfirm') }}
        <span>{{ user?.username }}</span>
      </p>
      <p v-if="deleteSubmitError" class="app-field-error mt-3" role="alert">
        {{ deleteSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          variant="destructive"
          @cancel="isDeleteModalOpen = false"
          @confirm="handleDelete"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Ban, CheckCircle, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { roleApi } from '@/api/role/role';
  import { userApi } from '@/api/user/user';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { formatTime } from '@/utils/time';
  import { useAuthStore } from '@/stores/auth';
  import { PERMISSIONS } from '@/constants/permissions';
  import type { RoleResp } from '@/gen/proto/orbit/v1/role/role';
  import type { UserResp } from '@/gen/proto/orbit/v1/user/user';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import UserRolesCard from './components/UserRolesCard.vue';

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
  const formErrors = reactive({ username: '', password: '', status: '' });
  const roleForm = reactive<{ roleIds: string[] }>({ roleIds: [] });
  const editSubmitError = ref('');
  const roleSubmitError = ref('');
  const disableSubmitError = ref('');
  const deleteSubmitError = ref('');

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
    Object.assign(formErrors, { username: '', password: '', status: '' });
    editSubmitError.value = '';
    isEditModalOpen.value = true;
  }

  function validateEditForm() {
    formErrors.username = form.username.trim() ? '' : t('userManagement.usernameRequired');
    formErrors.password =
      !form.password || form.password.trim().length >= 6
        ? ''
        : t('userManagement.passwordTooShort');
    formErrors.status = userStatusValues.includes(form.status)
      ? ''
      : t('userManagement.statusRequired');
    return !formErrors.username && !formErrors.password && !formErrors.status;
  }

  function openRoleModal() {
    roleForm.roleIds = user.value?.role_items.map((role) => role.id) ?? [];
    roleSubmitError.value = '';
    isRoleModalOpen.value = true;
  }

  async function handleEditOk() {
    editSubmitError.value = '';
    if (!validateEditForm()) {
      return;
    }
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
      editSubmitError.value = e instanceof Error ? e.message : t('userManagement.saveFailed');
    }
  }

  async function handleRoleOk() {
    roleSubmitError.value = '';
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
      roleSubmitError.value = e instanceof Error ? e.message : t('userManagement.saveFailed');
    }
  }

  function openDisableModal() {
    disableSubmitError.value = '';
    isDisableModalOpen.value = true;
  }

  async function handleDisable() {
    disableSubmitError.value = '';
    try {
      await executeOp(async () => {
        await userApi.disable(props.id, {});
        toast.success(t('userManagement.disabledToast'));
        isDisableModalOpen.value = false;
        await fetchUser();
      });
    } catch (e: unknown) {
      disableSubmitError.value = e instanceof Error ? e.message : t('userManagement.disableFailed');
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
    deleteSubmitError.value = '';
    isDeleteModalOpen.value = true;
  }

  async function handleDelete() {
    deleteSubmitError.value = '';
    try {
      await executeOp(async () => {
        await userApi.delete(props.id);
        toast.success(t('userManagement.deleted'));
        $router.push('/users');
      });
    } catch (e: unknown) {
      deleteSubmitError.value = e instanceof Error ? e.message : t('userManagement.deleteFailed');
    }
  }

  onMounted(() => {
    fetchUser();
    fetchRoleOptions();
  });
</script>
