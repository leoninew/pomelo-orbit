<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('project.toolbar')">
      <SearchControl
        v-model="searchText"
        class="shrink-0"
        :placeholder="t('project.searchPlaceholder')"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <div class="flex items-center gap-3">
        <button class="app-button-primary px-5" @click="openCreateDialog">
          <Plus class="size-4" />
          {{ t('project.createProject') }}
        </button>
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
        <p class="text-sm">{{ error || t('project.loadFailed') }}</p>
      </div>
      <AppEmptyState v-else-if="filteredProjects.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[900px]">
          <colgroup>
            <col class="w-[20%]" />
            <col class="w-[15%]" />
            <col class="w-[10%]" />
            <col class="w-[20%]" />
            <col class="w-[20%]" />
            <col class="w-[15%]" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ t('project.name') }}</th>
              <th>{{ t('project.code') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th>{{ t('common.updatedAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="project in pagedProjects" :key="project.id">
              <td class="max-w-0 truncate text-foreground" :title="project.name">
                <router-link :to="`/project/${project.id}`" class="app-link">
                  {{ project.name }}
                </router-link>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ project.code }}</td>
              <td>
                <AppBadge v-if="project.is_active" variant="status" tone="success">
                  {{ t('project.active') }}
                </AppBadge>
                <AppBadge v-else variant="status" tone="default">
                  {{ t('project.deprecated') }}
                </AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(project.created_at) }}
              </td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(project.updated_at) }}
              </td>
              <td class="whitespace-nowrap">
                <div class="flex items-center gap-3">
                  <button class="app-link" @click="openEditDialog(project)">
                    {{ t('common.edit') }}
                  </button>
                  <button
                    v-if="project.is_active"
                    class="app-link-danger"
                    @click="openDeprecateDialog(project)"
                  >
                    {{ t('project.deprecate') }}
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
        :total="filteredProjects.length"
        :total-pages="totalPages"
        @change-page="goPage"
        @change-page-size="handlePageSizeChange"
      />
    </div>

    <AppDialog
      v-model:open="isDialogOpen"
      :title="editingProject ? t('project.editProject') : t('project.createProject')"
      width-class="w-[min(760px,calc(100vw-32px))]"
      body-class="min-h-0 flex-1 space-y-4 overflow-y-auto px-6 py-4"
      content-class="max-h-[calc(100vh-32px)] flex flex-col"
    >
      <form class="space-y-4" novalidate @submit.prevent="handleSave">
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
            :aria-invalid="errors.name ? 'true' : undefined"
            @input="errors.name = ''"
          />
          <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
        </div>
        <template v-if="!editingProject">
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
              :aria-invalid="errors.code ? 'true' : undefined"
              @input="errors.code = ''"
            />
            <p v-if="errors.code" class="app-field-error text-xs">{{ errors.code }}</p>
            <p v-else class="app-field-hint">{{ t('project.codeHint') }}</p>
          </div>
          <div class="border-t border-border pt-4">
            <h3 class="text-sm font-semibold text-foreground">
              {{ t('project.environment.createTitle') }}
            </h3>
            <div class="mt-4 grid gap-4 sm:grid-cols-2">
              <div class="space-y-1.5">
                <label class="app-field-label block">{{ t('project.environment.state') }}</label>
                <SelectControl
                  v-model="form.environment.state"
                  :options="environmentStateOptions"
                />
              </div>
              <div class="space-y-1.5">
                <label class="app-field-label block">{{ t('project.environment.platform') }}</label>
                <SelectControl
                  v-model="form.environment.platform"
                  :options="environmentPlatformOptions"
                />
              </div>
              <div class="space-y-1.5">
                <label class="app-field-label block">
                  {{ t('project.environment.host') }}
                  <span class="text-destructive">*</span>
                </label>
                <input
                  v-model="form.environment.host"
                  type="text"
                  class="app-input"
                  :class="errors.host ? 'app-input-error' : ''"
                  @input="errors.host = ''"
                />
                <p v-if="errors.host" class="app-field-error text-xs">{{ errors.host }}</p>
              </div>
              <div class="space-y-1.5">
                <label class="app-field-label block">
                  {{ t('project.environment.port') }}
                  <span class="text-destructive">*</span>
                </label>
                <input
                  v-model.number="form.environment.port"
                  type="number"
                  min="1"
                  max="65535"
                  class="app-input"
                  :class="errors.port ? 'app-input-error' : ''"
                  @input="errors.port = ''"
                />
                <p v-if="errors.port" class="app-field-error text-xs">{{ errors.port }}</p>
              </div>
              <div class="space-y-1.5">
                <label class="app-field-label block">
                  {{ t('project.environment.username') }}
                  <span class="text-destructive">*</span>
                </label>
                <input
                  v-model="form.environment.username"
                  type="text"
                  class="app-input"
                  :class="errors.username ? 'app-input-error' : ''"
                  @input="errors.username = ''"
                />
                <p v-if="errors.username" class="app-field-error text-xs">{{ errors.username }}</p>
              </div>
              <div class="space-y-1.5">
                <label class="app-field-label block">
                  {{ t('project.environment.workspaceRoot') }}
                  <span class="text-destructive">*</span>
                </label>
                <input
                  v-model="form.environment.workspaceRoot"
                  type="text"
                  class="app-input"
                  :class="errors.workspaceRoot ? 'app-input-error' : ''"
                  :placeholder="workspaceRootPlaceholder"
                  @input="errors.workspaceRoot = ''"
                />
                <p v-if="errors.workspaceRoot" class="app-field-error text-xs">
                  {{ errors.workspaceRoot }}
                </p>
              </div>
              <div class="space-y-1.5 sm:col-span-2">
                <label class="app-field-label block">
                  {{ t('project.environment.hostKeyFingerprint') }}
                  <span class="text-destructive">*</span>
                </label>
                <input
                  v-model="form.environment.hostKeyFingerprint"
                  type="text"
                  class="app-input"
                  :class="errors.hostKeyFingerprint ? 'app-input-error' : ''"
                  placeholder="SHA256:..."
                  @input="errors.hostKeyFingerprint = ''"
                />
                <p v-if="errors.hostKeyFingerprint" class="app-field-error text-xs">
                  {{ errors.hostKeyFingerprint }}
                </p>
              </div>
              <div class="space-y-1.5">
                <label class="app-field-label block">
                  {{ t('project.environment.deploymentSSHKeyName') }}
                  <span class="text-destructive">*</span>
                </label>
                <input
                  v-model="form.environment.deploymentSSHKeyName"
                  type="text"
                  class="app-input"
                  :class="errors.deploymentSSHKeyName ? 'app-input-error' : ''"
                  @input="errors.deploymentSSHKeyName = ''"
                />
                <p v-if="errors.deploymentSSHKeyName" class="app-field-error text-xs">
                  {{ errors.deploymentSSHKeyName }}
                </p>
              </div>
              <div class="space-y-1.5">
                <label class="app-field-label block">
                  {{ t('project.environment.deploymentSSHKeyPassphrase') }}
                </label>
                <input
                  v-model="form.environment.deploymentSSHKeyPassphrase"
                  type="password"
                  autocomplete="new-password"
                  class="app-input"
                />
              </div>
              <div class="space-y-1.5 sm:col-span-2">
                <label class="app-field-label block">
                  {{ t('project.environment.deploymentSSHPrivateKey') }}
                  <span class="text-destructive">*</span>
                </label>
                <textarea
                  v-model="form.environment.deploymentSSHPrivateKey"
                  class="app-textarea min-h-40 font-mono text-xs"
                  :class="errors.deploymentSSHPrivateKey ? 'app-input-error' : ''"
                  @input="errors.deploymentSSHPrivateKey = ''"
                />
                <p v-if="errors.deploymentSSHPrivateKey" class="app-field-error text-xs">
                  {{ errors.deploymentSSHPrivateKey }}
                </p>
              </div>
            </div>
          </div>
        </template>
        <button type="submit" class="sr-only" tabindex="-1" aria-hidden="true"></button>
      </form>
      <p v-if="submitError" class="app-field-error mt-3" role="alert">{{ submitError }}</p>
      <template #footer>
        <AppDialogActions :busy="operating" @cancel="isDialogOpen = false" @confirm="handleSave" />
      </template>
    </AppDialog>

    <AppDialog v-model:open="isDeprecateDialogOpen" :title="t('project.deprecateProject')">
      <p class="text-sm text-foreground">
        {{ t('project.deprecateConfirmPrefix') }}
        <span>{{ deprecatingProject?.name }}</span>
        {{ t('project.deprecateConfirmSuffix') }}
      </p>
      <p v-if="deprecateSubmitError" class="app-field-error mt-3" role="alert">
        {{ deprecateSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          variant="destructive"
          @cancel="isDeprecateDialogOpen = false"
          @confirm="handleDeprecate"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { Plus } from '@lucide/vue';
  import { computed, nextTick, onMounted, reactive, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import { ToolbarRoot } from 'reka-ui';
  import type { ProjectResp } from '@/gen/proto/orbit/v1/project/project';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import { formatTime } from '@/utils/time';

  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const searchText = ref('');
  const appliedSearch = ref('');
  const isDialogOpen = ref(false);
  const isDeprecateDialogOpen = ref(false);
  const editingProject = ref<ProjectResp | null>(null);
  const deprecatingProject = ref<ProjectResp | null>(null);
  const pagination = reactive({ current: 1, pageSize: 10 });
  const form = reactive({
    name: '',
    code: '',
    environment: {
      state: 'active',
      platform: 'linux',
      host: '',
      port: 22,
      username: '',
      workspaceRoot: '/srv/pomelo-orbit',
      deploymentSSHKeyName: 'project-deploy-key',
      deploymentSSHPrivateKey: '',
      deploymentSSHKeyPassphrase: '',
      hostKeyFingerprint: '',
    },
  });
  const errors = reactive({
    name: '',
    code: '',
    host: '',
    port: '',
    username: '',
    workspaceRoot: '',
    deploymentSSHKeyName: '',
    deploymentSSHPrivateKey: '',
    hostKeyFingerprint: '',
  });
  const submitError = ref('');
  const deprecateSubmitError = ref('');

  const environmentStateOptions = computed(() => [
    { value: 'active', label: t('project.environment.states.active') },
    { value: 'disabled', label: t('project.environment.states.disabled') },
  ]);
  const environmentPlatformOptions = computed(() => [
    { value: 'linux', label: t('project.environment.platforms.linux') },
    { value: 'windows', label: t('project.environment.platforms.windows') },
  ]);
  const workspaceRootPlaceholder = computed(() =>
    form.environment.platform === 'windows'
      ? t('project.environment.windowsWorkspacePlaceholder')
      : t('project.environment.linuxWorkspacePlaceholder')
  );

  const filteredProjects = computed(() => {
    const keyword = appliedSearch.value.trim().toLowerCase();
    if (!keyword) {
      return projectStore.projects;
    }
    return projectStore.projects.filter(
      (project) =>
        project.name.toLowerCase().includes(keyword) || project.code.toLowerCase().includes(keyword)
    );
  });
  const totalPages = computed(() =>
    Math.max(1, Math.ceil(filteredProjects.value.length / pagination.pageSize))
  );
  const pagedProjects = computed(() => {
    const start = (pagination.current - 1) * pagination.pageSize;
    return filteredProjects.value.slice(start, start + pagination.pageSize);
  });

  function resetForm(project?: ProjectResp) {
    form.name = project?.name ?? '';
    form.code = project?.code ?? '';
    form.environment = {
      state: 'active',
      platform: 'linux',
      host: '',
      port: 22,
      username: '',
      workspaceRoot: '/srv/pomelo-orbit',
      deploymentSSHKeyName: 'project-deploy-key',
      deploymentSSHPrivateKey: '',
      deploymentSSHKeyPassphrase: '',
      hostKeyFingerprint: '',
    };
    Object.keys(errors).forEach((key) => {
      errors[key as keyof typeof errors] = '';
    });
    submitError.value = '';
  }

  function validate() {
    errors.name = form.name.trim() ? '' : t('project.nameRequired');
    if (editingProject.value) return !errors.name;
    const environment = form.environment;
    errors.code = /^[a-z0-9_-]+$/.test(form.code) ? '' : t('project.codeInvalid');
    errors.host =
      environment.host.trim() && !/\s/.test(environment.host)
        ? ''
        : t('project.environment.validation.host');
    errors.port =
      Number.isInteger(environment.port) && environment.port >= 1 && environment.port <= 65535
        ? ''
        : t('project.environment.validation.port');
    errors.username =
      environment.username.trim() && !/[\r\n]/.test(environment.username)
        ? ''
        : t('project.environment.validation.username');
    const workspaceRootValid =
      environment.platform === 'linux'
        ? environment.workspaceRoot.trim().startsWith('/')
        : /^[A-Za-z]:\\/.test(environment.workspaceRoot.trim());
    errors.workspaceRoot = workspaceRootValid
      ? ''
      : t('project.environment.validation.workspaceRoot');
    errors.deploymentSSHKeyName = environment.deploymentSSHKeyName.trim()
      ? ''
      : t('project.environment.validation.deploymentSSHKeyName');
    errors.deploymentSSHPrivateKey = environment.deploymentSSHPrivateKey.trim()
      ? ''
      : t('project.environment.validation.deploymentSSHPrivateKey');
    errors.hostKeyFingerprint = /^SHA256:[A-Za-z0-9+/]+={0,2}$/.test(
      environment.hostKeyFingerprint.trim()
    )
      ? ''
      : t('project.environment.validation.hostKeyFingerprint');
    return !Object.values(errors).some(Boolean);
  }

  function openCreateDialog() {
    editingProject.value = null;
    resetForm();
    isDialogOpen.value = true;
  }

  function openEditDialog(project: ProjectResp) {
    editingProject.value = project;
    resetForm(project);
    isDialogOpen.value = true;
  }

  async function openDeprecateDialog(project: ProjectResp) {
    deprecatingProject.value = project;
    deprecateSubmitError.value = '';
    (document.activeElement as HTMLElement)?.blur();
    await nextTick();
    isDeprecateDialogOpen.value = true;
  }

  function handleSearch() {
    appliedSearch.value = searchText.value;
    pagination.current = 1;
    void fetchProjects();
  }

  function goPage(page: number) {
    pagination.current = page;
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
  }

  async function fetchProjects() {
    await execute(async () => {
      await projectStore.fetchProjects();
    });
  }

  async function handleSave() {
    submitError.value = '';
    if (!validate()) {
      return;
    }
    try {
      await executeOp(async () => {
        if (editingProject.value) {
          await projectStore.updateProject(editingProject.value.id, { name: form.name.trim() });
          toast.success(t('project.updated'));
          isDialogOpen.value = false;
        } else {
          const environment = form.environment;
          const project = await projectStore.createProject({
            name: form.name.trim(),
            code: form.code.trim(),
            environment: {
              state: environment.state,
              platform: environment.platform,
              host: environment.host.trim(),
              port: environment.port,
              username: environment.username.trim(),
              workspace_root: environment.workspaceRoot.trim(),
              deployment_ssh_key_name: environment.deploymentSSHKeyName.trim(),
              deployment_ssh_private_key: environment.deploymentSSHPrivateKey.trim(),
              deployment_ssh_key_passphrase: environment.deploymentSSHKeyPassphrase || undefined,
              host_key_fingerprint: environment.hostKeyFingerprint.trim(),
            },
          });
          toast.success(t('project.created'));
          isDialogOpen.value = false;
          router.push({ name: 'ProjectDetail', params: { id: project.id } });
        }
      });
    } catch (error: unknown) {
      submitError.value = error instanceof Error ? error.message : t('project.saveFailed');
    }
  }

  async function handleDeprecate() {
    if (!deprecatingProject.value) {
      return;
    }
    deprecateSubmitError.value = '';
    const projectId = deprecatingProject.value.id;
    try {
      await executeOp(async () => {
        await projectStore.deprecateProject(projectId);
        toast.success(t('project.deprecatedToast'));
        isDeprecateDialogOpen.value = false;
      });
    } catch (error: unknown) {
      deprecateSubmitError.value = error instanceof Error ? error.message : t('project.saveFailed');
    }
  }

  onMounted(fetchProjects);
</script>
