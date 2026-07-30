<template>
  <header
    class="sticky top-0 z-40 flex min-h-14 shrink-0 items-center overflow-hidden border-b border-border bg-card/95 px-3 shadow-sm backdrop-blur md:h-16 md:px-4"
  >
    <RouterLink
      to="/"
      class="flex h-14 w-auto shrink-0 items-center gap-3 rounded-md px-2 text-foreground outline-none transition-colors hover:text-primary focus-visible:ring-2 focus-visible:ring-ring/20 sm:px-3 md:h-full md:w-60"
      :aria-label="t('app.homeAria')"
    >
      <img src="/logo-128.png" alt="Pomelo Orbit Logo" class="size-9" width="36" height="36" />
      <div class="hidden flex-col sm:flex">
        <span class="text-base font-semibold leading-tight tracking-normal">Pomelo Orbit</span>
        <span
          v-if="runtimeConfig.envLabel"
          class="text-xs leading-tight text-amber-600 dark:text-amber-400"
        >
          {{ runtimeConfig.envLabel }}
        </span>
      </div>
    </RouterLink>

    <NavigationMenuRoot
      :model-value="currentModule ?? undefined"
      class="flex h-14 min-w-0 flex-1 overflow-x-auto md:h-full md:flex-none"
      :aria-label="t('app.primaryNavAria')"
      :delay-duration="100"
      :skip-delay-duration="200"
    >
      <NavigationMenuList class="flex h-full min-w-max items-stretch gap-0">
        <NavigationMenuItem
          v-for="item in localizedPrimaryNavigation"
          :key="item.key"
          :value="item.key"
        >
          <NavigationMenuLink as-child :active="isActive(item.key)">
            <RouterLink
              :to="item.path"
              class="flex h-full min-w-24 items-center justify-center border-b-2 px-3 text-sm outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring/20 md:min-w-28 md:px-5"
              :class="
                isActive(item.key)
                  ? 'border-primary text-primary'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              "
            >
              {{ item.label }}
            </RouterLink>
          </NavigationMenuLink>
        </NavigationMenuItem>
      </NavigationMenuList>
    </NavigationMenuRoot>

    <div class="hidden flex-1 md:block" />

    <ToolbarRoot class="hidden items-center gap-3 md:flex" :aria-label="t('app.globalToolbarAria')">
      <ToolbarButton
        class="inline-flex size-9 items-center justify-center rounded-md text-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/20"
        :aria-label="t('theme.' + theme)"
        @click="cycleTheme"
      >
        <Monitor v-if="theme === 'system'" class="size-5" />
        <Sun v-else-if="theme === 'light'" class="size-5" />
        <Moon v-else class="size-5" />
      </ToolbarButton>
      <ToolbarButton
        class="inline-flex h-9 min-w-12 items-center justify-center gap-1.5 rounded-md px-2 text-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/20"
        :aria-label="switchLocaleLabel"
        :title="switchLocaleLabel"
        @click="toggleLocale"
      >
        <Languages class="size-5" />
        <span class="text-xs leading-none">{{ nextLocaleShortName }}</span>
      </ToolbarButton>
    </ToolbarRoot>

    <DropdownMenuRoot>
      <DropdownMenuTrigger
        class="ml-1 flex h-10 cursor-pointer items-center gap-2 rounded-md px-2 text-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/20 data-[state=open]:bg-accent data-[state=open]:text-accent-foreground md:ml-0 md:h-11 md:gap-3 md:px-3"
        :aria-label="t('app.userMenuAria')"
        @click="handleUserMenuOpen"
      >
        <span
          class="flex size-8 items-center justify-center rounded-full bg-primary text-sm text-primary-foreground"
        >
          {{ (authStore.user?.username || 'admin').slice(0, 1).toUpperCase() }}
        </span>
        <span class="hidden text-sm lg:inline">{{ authStore.user?.username || 'admin' }}</span>
        <ChevronDown class="hidden size-4 text-muted-foreground sm:block" />
      </DropdownMenuTrigger>
      <DropdownMenuPortal>
        <DropdownMenuContent
          class="z-50 min-w-64 rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-lg outline-none data-[state=open]:animate-slideDownAndFade"
          align="end"
          :side-offset="8"
        >
          <DropdownMenuSub>
            <DropdownMenuSubTrigger
              class="flex cursor-pointer items-center gap-2 rounded px-3 py-2 text-sm outline-none transition-colors data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground data-[state=open]:bg-accent"
            >
              <FolderKanban class="size-4" />
              <span class="min-w-0 flex-1 truncate">{{ activeProjectLabel }}</span>
              <ChevronRight class="size-4 text-muted-foreground" />
            </DropdownMenuSubTrigger>
            <DropdownMenuPortal>
              <DropdownMenuSubContent
                class="z-50 min-w-48 rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-lg outline-none data-[state=open]:animate-slideDownAndFade"
                :side-offset="8"
              >
                <div v-if="projectStore.loading" class="px-3 py-2 text-sm text-muted-foreground">
                  {{ t('common.loading') }}
                </div>
                <template v-else>
                  <DropdownMenuItem
                    v-for="project in activeProjects"
                    :key="project.id"
                    class="flex cursor-pointer items-center gap-2 rounded px-3 py-2 text-sm outline-none transition-colors data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground"
                    :class="{ 'bg-accent': project.id === projectStore.activeProjectId }"
                    @select="handleSwitchProject(project.id)"
                  >
                    <span class="min-w-0 flex-1 truncate">{{ project.name }}</span>
                    <span class="text-xs text-muted-foreground">{{ project.code }}</span>
                  </DropdownMenuItem>
                  <div
                    v-if="activeProjects.length === 0"
                    class="px-3 py-2 text-sm text-muted-foreground"
                  >
                    {{ t('project.noProjects') }}
                  </div>
                </template>
              </DropdownMenuSubContent>
            </DropdownMenuPortal>
          </DropdownMenuSub>
          <div class="my-1 h-px bg-border" />
          <DropdownMenuItem
            class="flex cursor-pointer items-center gap-2 rounded px-3 py-2 text-sm outline-none transition-colors data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground"
            @select="isPasswordDialogOpen = true"
          >
            <KeyRound class="size-4" />
            {{ t('user.changePassword') }}
          </DropdownMenuItem>
          <DropdownMenuItem
            class="flex cursor-pointer items-center gap-2 rounded px-3 py-2 text-sm outline-none transition-colors data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground"
            @select="handleLogout"
          >
            <LogOut class="size-4" />
            {{ t('user.logout') }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenuPortal>
    </DropdownMenuRoot>
  </header>

  <AppDialog v-model:open="isPasswordDialogOpen" :title="t('settings.passwordDialog.title')">
    <div class="space-y-4">
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('settings.passwordDialog.oldPassword') }}
        </label>
        <input
          v-model="passwordForm.old_password"
          type="password"
          :placeholder="t('settings.passwordDialog.oldPasswordPlaceholder')"
          class="app-input"
          :class="passwordErrors.old_password ? 'app-input-error' : ''"
        />
        <p v-if="passwordErrors.old_password" class="app-field-error text-xs">
          {{ passwordErrors.old_password }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('settings.passwordDialog.newPassword') }}
        </label>
        <input
          v-model="passwordForm.new_password"
          type="password"
          :placeholder="t('settings.passwordDialog.newPasswordPlaceholder')"
          class="app-input"
          :class="passwordErrors.new_password ? 'app-input-error' : ''"
        />
        <p v-if="passwordErrors.new_password" class="app-field-error text-xs">
          {{ passwordErrors.new_password }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('settings.passwordDialog.confirmPassword') }}
        </label>
        <input
          v-model="passwordForm.confirm_password"
          type="password"
          :placeholder="t('settings.passwordDialog.confirmPasswordPlaceholder')"
          class="app-input"
          :class="passwordErrors.confirm_password ? 'app-input-error' : ''"
        />
        <p v-if="passwordErrors.confirm_password" class="app-field-error text-xs">
          {{ passwordErrors.confirm_password }}
        </p>
      </div>
    </div>
    <template #footer>
      <AppDialogActions
        :busy="passwordLoading"
        @cancel="closePasswordDialog"
        @confirm="handleChangePassword"
      />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import {
    ChevronDown,
    ChevronRight,
    FolderKanban,
    KeyRound,
    Languages,
    LogOut,
    Monitor,
    Moon,
    Sun,
  } from 'lucide-vue-next';
  import { computed, reactive, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import { primaryNavigation, type PrimaryNavigationKey } from '@/navigation';
  import { useAuthStore } from '@/stores/auth';
  import { useProjectStore } from '@/stores/project';
  import { useTheme } from '@/composables/useTheme';
  import { setLocale, type Locale } from '@/i18n';
  import { useToast } from '@/composables/useToast';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import runtimeConfig from '@/config';
  import {
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuPortal,
    DropdownMenuRoot,
    DropdownMenuSub,
    DropdownMenuSubContent,
    DropdownMenuSubTrigger,
    DropdownMenuTrigger,
    NavigationMenuItem,
    NavigationMenuLink,
    NavigationMenuList,
    NavigationMenuRoot,
    ToolbarButton,
    ToolbarRoot,
  } from 'reka-ui';

  const props = defineProps<{
    currentModule: PrimaryNavigationKey | null;
  }>();

  const router = useRouter();
  const authStore = useAuthStore();
  const projectStore = useProjectStore();
  const { theme, cycleTheme } = useTheme();
  const { t, locale } = useI18n({ useScope: 'global' });
  const toast = useToast();
  const { loading: passwordLoading, execute: executeChangePassword } = useStatusAsync();

  const isPasswordDialogOpen = ref(false);
  const passwordForm = reactive({
    old_password: '',
    new_password: '',
    confirm_password: '',
  });
  const passwordErrors = reactive({
    old_password: '',
    new_password: '',
    confirm_password: '',
  });

  const activeProjectLabel = computed(() => {
    const project = projectStore.activeProject;
    return project ? `${project.name} / ${project.code}` : t('project.noProjects');
  });

  const activeProjects = computed(() =>
    projectStore.projects.filter((project) => project.is_active)
  );

  const localizedPrimaryNavigation = computed(() =>
    primaryNavigation.map((item) => ({
      ...item,
      label: t(item.labelKey),
    }))
  );

  const nextLocale = computed<Locale>(() => (locale.value === 'zh-CN' ? 'en-US' : 'zh-CN'));
  const nextLocaleShortName = computed(() => (nextLocale.value === 'zh-CN' ? '中' : 'EN'));
  const switchLocaleLabel = computed(() => {
    const localeName = t(`language.${nextLocale.value === 'zh-CN' ? 'zhCN' : 'enUS'}`);
    return t('language.switchTo', { language: localeName });
  });

  function toggleLocale() {
    setLocale(nextLocale.value);
  }

  function isActive(moduleKey: PrimaryNavigationKey) {
    return props.currentModule === moduleKey;
  }

  async function handleUserMenuOpen() {
    if (projectStore.projects.length === 0 && !projectStore.loading) {
      try {
        await projectStore.fetchProjects();
      } catch (error: unknown) {
        toast.error(error instanceof Error ? error.message : '加载项目列表失败');
      }
    }
  }

  function handleSwitchProject(projectId: string) {
    if (projectId !== projectStore.activeProjectId) {
      projectStore.setActiveProject(projectId);
      router.go(0);
    }
  }

  function resetPasswordForm() {
    Object.assign(passwordForm, {
      old_password: '',
      new_password: '',
      confirm_password: '',
    });
    Object.assign(passwordErrors, {
      old_password: '',
      new_password: '',
      confirm_password: '',
    });
  }

  function closePasswordDialog() {
    isPasswordDialogOpen.value = false;
    resetPasswordForm();
  }

  function validatePassword() {
    passwordErrors.old_password = passwordForm.old_password
      ? ''
      : t('settings.passwordDialog.oldPasswordRequired');
    passwordErrors.new_password =
      passwordForm.new_password.length >= 6 ? '' : t('settings.passwordDialog.newPasswordTooShort');
    passwordErrors.confirm_password =
      passwordForm.confirm_password === passwordForm.new_password
        ? ''
        : t('settings.passwordDialog.passwordMismatch');
    return (
      !passwordErrors.old_password &&
      !passwordErrors.new_password &&
      !passwordErrors.confirm_password
    );
  }

  async function handleChangePassword() {
    if (!validatePassword()) {
      return;
    }
    try {
      await executeChangePassword(async () => {
        await authStore.changePassword(passwordForm.old_password, passwordForm.new_password);
        toast.success(t('settings.passwordDialog.changeSuccess'));
        closePasswordDialog();
      });
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('settings.passwordDialog.changeFailed')
      );
    }
  }

  async function handleLogout() {
    projectStore.clearProjects();
    await authStore.logout();
    router.push('/login');
  }
</script>
