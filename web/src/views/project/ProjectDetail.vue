<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">
          {{ project?.name ?? t('project.detailTitle') }}
        </h1>
        <DetailHeaderMeta v-if="project">
          <AppBadge v-if="project.is_active" variant="status" tone="success">
            {{ t('project.active') }}
          </AppBadge>
          <AppBadge v-else variant="status" tone="default">
            {{ t('project.deprecated') }}
          </AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button class="app-button h-9 px-4" @click="router.push('/projects')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <DetailInfoCard
      :title="t('userManagement.basicInfo')"
      :loading="loading"
      :editable="Boolean(project)"
      :disabled="operating"
      @edit="openEditModal"
    >
      <dl v-if="project" class="app-detail-info-grid">
        <div class="flex gap-2">
          <dt>{{ t('project.name') }}</dt>
          <dd class="text-foreground">{{ project.name }}</dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('project.code') }}</dt>
          <dd class="text-foreground">{{ project.code }}</dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('common.status') }}</dt>
          <dd>
            <AppBadge v-if="project.is_active" variant="status" tone="success">
              {{ t('project.active') }}
            </AppBadge>
            <AppBadge v-else variant="status" tone="default">
              {{ t('project.deprecated') }}
            </AppBadge>
          </dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('common.createdAt') }}</dt>
          <dd class="text-muted-foreground">{{ formatTime(project.created_at) }}</dd>
        </div>
        <div class="flex gap-2">
          <dt>{{ t('common.updatedAt') }}</dt>
          <dd class="text-muted-foreground">{{ formatTime(project.updated_at) }}</dd>
        </div>
      </dl>
    </DetailInfoCard>

    <DetailInfoCard :title="t('project.members')">
      <template #actions>
        <button class="app-button-primary h-9 px-3" :disabled="operating" @click="openMemberModal">
          <UserPlus class="size-4" />
          {{ t('common.add') }}
        </button>
      </template>

      <AppLoadingState v-if="loadingMembers" size="section" />
      <AppEmptyState v-else-if="members.length === 0" size="compact" />
      <div v-else class="px-5 py-4">
        <table class="app-data-table">
          <thead>
            <tr>
              <th>{{ t('userManagement.username') }}</th>
              <th>{{ t('userManagement.email') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('userManagement.authSource') }}</th>
              <th>{{ t('userManagement.lastLoginAt') }}</th>
              <th class="w-20">{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="member in members" :key="member.id">
              <td>{{ member.username }}</td>
              <td>{{ member.email }}</td>
              <td>
                <AppBadge
                  variant="status"
                  :tone="member.status === 'enabled' ? 'success' : 'default'"
                >
                  {{ member.status }}
                </AppBadge>
              </td>
              <td>
                <AppBadge variant="pill">{{ member.auth_source }}</AppBadge>
              </td>
              <td>{{ member.last_login_at ? formatTime(member.last_login_at) : '' }}</td>
              <td class="w-20">
                <button
                  class="app-link-danger"
                  :disabled="operating"
                  @click="handleRemoveMember(member.id)"
                >
                  {{ t('project.removeMember') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </DetailInfoCard>

    <AppDialog
      v-model:open="isEditModalOpen"
      :title="t('project.editProject')"
      width-class="w-[min(600px,calc(100vw-32px))]"
    >
      <form class="space-y-4" novalidate @submit.prevent="handleEditOk">
        <div class="space-y-1.5">
          <label class="app-field-label block" for="project-name">
            {{ t('project.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="project-name"
            v-model="form.name"
            type="text"
            class="app-input"
            :class="errors.name ? 'app-input-error' : ''"
            maxlength="100"
            required
            :disabled="operating"
            :aria-invalid="errors.name ? 'true' : undefined"
            @input="errors.name = ''"
          />
          <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="project-code">
            {{ t('project.code') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="project-code"
            v-model="form.code"
            type="text"
            class="app-input"
            :class="errors.code ? 'app-input-error' : ''"
            maxlength="50"
            pattern="[a-z0-9_-]+"
            required
            :disabled="operating"
            :aria-invalid="errors.code ? 'true' : undefined"
            @input="errors.code = ''"
          />
          <p v-if="errors.code" class="app-field-error text-xs">{{ errors.code }}</p>
          <p v-else class="app-field-hint">{{ t('project.codeHint') }}</p>
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

    <AppDialog v-model:open="isMemberModalOpen" :title="t('project.addMember')">
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('project.selectUser') }}
            <span class="text-destructive">*</span>
          </label>
          <ComboboxSelect
            v-model="selectedUserId"
            :options="userOptions"
            :placeholder="t('project.searchUser')"
            :empty-text="t('project.noAvailableUsers')"
            :disabled="operating"
            :invalid="Boolean(memberErrors.userId)"
            @update:model-value="memberErrors.userId = ''"
          />
          <p v-if="memberErrors.userId" class="app-field-error" role="alert">
            {{ memberErrors.userId }}
          </p>
        </div>
      </div>
      <p v-if="memberSubmitError" class="app-field-error mt-3" role="alert">
        {{ memberSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions :busy="operating" @cancel="closeMemberModal" @confirm="handleAddMember" />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, UserPlus } from '@lucide/vue';
  import { onMounted, reactive, ref, computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { projectApi } from '@/api/project/project';
  import { userApi } from '@/api/user/user';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { ProjectMemberResp, ProjectResp } from '@/gen/proto/orbit/v1/project/project';
  import type { UserListResp } from '@/gen/proto/orbit/v1/user/user';
  import { formatTime } from '@/utils/time';

  const props = defineProps<{ id: string }>();
  const { t } = useI18n();
  const router = useRouter();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { loading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const project = ref<ProjectResp>();
  const isEditModalOpen = ref(false);
  const isMemberModalOpen = ref(false);
  const members = ref<ProjectMemberResp[]>([]);
  const users = ref<UserListResp[]>([]);
  const selectedUserId = ref('');
  const form = reactive({ name: '', code: '' });
  const errors = reactive({ name: '', code: '' });
  const memberErrors = reactive({ userId: '' });
  const editSubmitError = ref('');
  const memberSubmitError = ref('');
  const { loading: loadingMembers, execute: executeMembers } = useStatusAsync();

  const availableUsers = computed(() => {
    const memberIds = new Set(members.value.map((m) => m.id));
    return users.value.filter((u) => u.status === 'enabled' && !memberIds.has(u.id));
  });

  const userOptions = computed(() =>
    availableUsers.value.map((u) => ({
      value: u.id,
      label: u.username,
      description: u.email || undefined,
    }))
  );

  function resetForm() {
    form.name = project.value?.name ?? '';
    form.code = project.value?.code ?? '';
    errors.name = '';
    errors.code = '';
    editSubmitError.value = '';
  }

  function validate() {
    errors.name = form.name.trim() ? '' : t('project.nameRequired');
    errors.code = /^[a-z0-9_-]+$/.test(form.code) ? '' : t('project.codeInvalid');
    return !errors.name && !errors.code;
  }

  async function fetchProject() {
    try {
      await execute(async () => {
        project.value = await projectApi.get(props.id);
      });
    } catch {
      toast.error(t('project.loadProjectFailed'));
    }
  }

  async function fetchMembers() {
    try {
      await executeMembers(async () => {
        const [memberList, userPage] = await Promise.all([
          projectApi.listMembers(props.id),
          userApi.list({ page: 1, per_page: 100 }),
        ]);
        members.value = memberList.items;
        users.value = userPage.items;
      });
    } catch {
      toast.error(t('project.loadMembersFailed'));
    }
  }

  function openEditModal() {
    resetForm();
    isEditModalOpen.value = true;
  }

  async function handleEditOk() {
    editSubmitError.value = '';
    if (!validate()) {
      return;
    }
    try {
      await executeOp(async () => {
        project.value = await projectStore.updateProject(props.id, {
          name: form.name.trim(),
          code: form.code.trim(),
        });
        toast.success(t('project.updated'));
        isEditModalOpen.value = false;
      });
    } catch (e: unknown) {
      editSubmitError.value = e instanceof Error ? e.message : t('project.saveFailed');
    }
  }

  function openMemberModal() {
    selectedUserId.value = '';
    memberErrors.userId = '';
    memberSubmitError.value = '';
    isMemberModalOpen.value = true;
  }

  function closeMemberModal() {
    selectedUserId.value = '';
    memberErrors.userId = '';
    memberSubmitError.value = '';
    isMemberModalOpen.value = false;
  }

  async function handleAddMember() {
    memberSubmitError.value = '';
    if (!selectedUserId.value) {
      memberErrors.userId = t('project.selectUserRequired');
      return;
    }
    try {
      await executeOp(async () => {
        const resp = await projectApi.addMember(props.id, { user_id: selectedUserId.value });
        members.value = resp.items;
        selectedUserId.value = '';
        memberErrors.userId = '';
        toast.success(t('project.memberAdded'));
        isMemberModalOpen.value = false;
      });
    } catch (e: unknown) {
      memberSubmitError.value = e instanceof Error ? e.message : t('project.addMemberFailed');
    }
  }

  async function handleRemoveMember(userId: string) {
    try {
      await executeOp(async () => {
        const resp = await projectApi.removeMember(props.id, userId);
        members.value = resp.items;
        toast.success(t('project.memberRemoved'));
      });
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('project.removeMemberFailed'));
    }
  }

  onMounted(() => {
    fetchProject();
    fetchMembers();
  });
</script>
