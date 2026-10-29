<template>
  <header
    class="app-topbar sticky top-0 z-40 flex min-h-14 shrink-0 items-center overflow-hidden border-b border-border bg-background px-3 md:h-16 md:px-4"
  >
    <RouterLink
      to="/"
      class="app-brand-link flex h-14 w-auto shrink-0 items-center gap-3 rounded-md px-2 text-foreground outline-none transition-colors hover:text-primary focus-visible:bg-accent focus-visible:text-primary sm:px-3 md:h-full md:w-60"
      :aria-label="t('app.homeAria')"
    >
      <img
        src="/logo-128.png"
        alt="Pomelo Orbit Logo"
        class="app-brand-image size-9"
        width="36"
        height="36"
      />
      <div class="hidden flex-col sm:flex">
        <span class="app-brand-name text-base font-medium leading-tight tracking-normal">
          Pomelo Orbit
        </span>
        <span
          v-if="runtimeConfig.envLabel"
          class="text-sm leading-tight text-amber-600 dark:text-amber-400"
        >
          {{ runtimeConfig.envLabel }}
        </span>
      </div>
    </RouterLink>

    <NavigationMenuRoot
      :model-value="currentModule ?? undefined"
      class="app-primary-nav flex h-14 min-w-0 flex-1 overflow-x-auto md:h-full md:flex-none"
      :aria-label="t('app.primaryNavAria')"
      :delay-duration="100"
      :skip-delay-duration="200"
    >
      <NavigationMenuList class="flex h-full min-w-max items-center gap-1">
        <NavigationMenuItem
          v-for="item in localizedPrimaryNavigation"
          :key="item.key"
          :value="item.key"
          class="app-module-item"
        >
          <NavigationMenuLink as-child :active="isActive(item.key)">
            <RouterLink
              :to="item.path"
              class="app-module-link flex h-9 min-w-24 items-center justify-center border-b px-3 text-sm outline-none transition-colors focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring md:min-w-28 md:px-5"
              :data-active="isActive(item.key) ? '' : undefined"
              :class="
                isActive(item.key)
                  ? 'border-primary text-foreground'
                  : 'border-transparent text-muted-foreground hover:border-border hover:text-foreground'
              "
            >
              {{ item.label }}
            </RouterLink>
          </NavigationMenuLink>
        </NavigationMenuItem>
      </NavigationMenuList>
    </NavigationMenuRoot>

    <div class="hidden flex-1 md:block" />

    <ToolbarRoot
      class="app-global-tools hidden items-center gap-3 md:flex"
      :aria-label="t('app.globalToolbarAria')"
    >
      <ToolbarButton
        class="inline-flex size-9 items-center justify-center rounded-md text-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground"
        :aria-label="t('theme.' + theme)"
        @click="cycleTheme"
      >
        <Monitor v-if="theme === 'system'" class="size-5" />
        <Sun v-else-if="theme === 'light'" class="size-5" />
        <Moon v-else class="size-5" />
      </ToolbarButton>
      <ToolbarButton
        class="inline-flex h-9 min-w-12 items-center justify-center gap-1.5 rounded-md px-2 text-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground"
        :aria-label="switchLocaleLabel"
        :title="switchLocaleLabel"
        @click="toggleLocale"
      >
        <Languages class="size-5" />
        <span>{{ nextLocaleShortName }}</span>
      </ToolbarButton>
    </ToolbarRoot>

    <DropdownMenuRoot>
      <DropdownMenuTrigger
        class="app-project-switcher ml-1 flex h-10 max-w-40 cursor-pointer items-center gap-2 rounded-md px-2 text-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground data-[state=open]:bg-accent data-[state=open]:text-accent-foreground md:ml-0 md:h-11 md:max-w-56"
        :aria-label="t('app.projectMenuAria', { project: activeProjectLabel })"
        @click="handleProjectMenuOpen"
      >
        <Cloud class="size-4 shrink-0" />
        <AppTruncatedText :text="activeProjectLabel" class="text-sm" as-child>
          <span>{{ activeProjectLabel }}</span>
        </AppTruncatedText>
        <ChevronDown class="size-4 shrink-0 text-muted-foreground" />
      </DropdownMenuTrigger>
      <DropdownMenuPortal>
        <DropdownMenuContent
          class="app-popover-content min-w-56 max-w-[calc(100vw-1rem)] overflow-y-auto data-[state=open]:animate-slideDownAndFade motion-reduce:animate-none"
          align="end"
          :side-offset="8"
        >
          <div v-if="projectStore.loading" class="px-3 py-2 text-sm text-muted-foreground">
            {{ t('common.loading') }}
          </div>
          <template v-else>
            <DropdownMenuItem
              v-for="project in projectStore.activeProjects"
              :key="project.id"
              class="app-option-item justify-start gap-2"
              :class="{ 'bg-accent': project.id === projectStore.activeProjectId }"
              @select="handleSwitchProject(project.id)"
            >
              <AppTruncatedText :text="project.name" class="flex-1" as-child>
                <span>{{ project.name }}</span>
              </AppTruncatedText>
              <span class="shrink-0 text-sm text-muted-foreground">{{ project.code }}</span>
              <Check
                v-if="project.id === projectStore.activeProjectId"
                class="size-4 shrink-0 text-primary"
              />
            </DropdownMenuItem>
            <div
              v-if="projectStore.activeProjects.length === 0"
              class="px-3 py-2 text-sm text-muted-foreground"
            >
              {{ t('project.noProjects') }}
            </div>
          </template>
        </DropdownMenuContent>
      </DropdownMenuPortal>
    </DropdownMenuRoot>
    <DropdownMenuRoot>
      <DropdownMenuTrigger
        class="app-user-menu-trigger ml-1 flex h-10 cursor-pointer items-center gap-2 rounded-md px-2 text-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground data-[state=open]:bg-accent data-[state=open]:text-accent-foreground md:ml-0 md:h-11 md:gap-3 md:px-3"
        :aria-label="t('app.userMenuAria')"
      >
        <span class="max-w-16 truncate text-sm sm:max-w-32">
          {{ authStore.user?.username || 'admin' }}
        </span>
        <ChevronDown class="size-4 shrink-0 text-muted-foreground" />
      </DropdownMenuTrigger>
      <DropdownMenuPortal>
        <DropdownMenuContent
          class="app-popover-content min-w-64 data-[state=open]:animate-slideDownAndFade motion-reduce:animate-none"
          align="end"
          :side-offset="8"
        >
          <DropdownMenuItem
            class="app-option-item justify-start gap-2"
            @select="isAboutDialogOpen = true"
          >
            <CircleHelp class="size-4" />
            {{ t('app.about') }}
          </DropdownMenuItem>
          <div class="my-1 h-px bg-border" />
          <DropdownMenuItem
            class="app-option-item justify-start gap-2"
            @select="isPasswordDialogOpen = true"
          >
            <KeyRound class="size-4" />
            {{ t('user.changePassword') }}
          </DropdownMenuItem>
          <DropdownMenuItem class="app-option-item justify-start gap-2" @select="handleLogout">
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
        <AppInput
          v-model="passwordForm.old_password"
          type="password"
          :placeholder="t('settings.passwordDialog.oldPasswordPlaceholder')"
          :class="passwordErrors.old_password ? 'app-input-error' : ''"
        />
        <p v-if="passwordErrors.old_password" class="app-field-error">
          {{ passwordErrors.old_password }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('settings.passwordDialog.newPassword') }}
        </label>
        <AppInput
          v-model="passwordForm.new_password"
          type="password"
          :placeholder="t('settings.passwordDialog.newPasswordPlaceholder')"
          :class="passwordErrors.new_password ? 'app-input-error' : ''"
        />
        <p v-if="passwordErrors.new_password" class="app-field-error">
          {{ passwordErrors.new_password }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('settings.passwordDialog.confirmPassword') }}
        </label>
        <AppInput
          v-model="passwordForm.confirm_password"
          type="password"
          :placeholder="t('settings.passwordDialog.confirmPasswordPlaceholder')"
          :class="passwordErrors.confirm_password ? 'app-input-error' : ''"
        />
        <p v-if="passwordErrors.confirm_password" class="app-field-error">
          {{ passwordErrors.confirm_password }}
        </p>
      </div>
    </div>
    <p v-if="passwordSubmitError" class="app-field-error mt-3" role="alert">
      {{ passwordSubmitError }}
    </p>
    <template #footer>
      <AppDialogActions
        :busy="passwordLoading"
        @cancel="closePasswordDialog"
        @confirm="handleChangePassword"
      />
    </template>
  </AppDialog>

  <AppDialog
    v-model:open="isAboutDialogOpen"
    :title="t('app.about')"
    width-class="w-[min(400px,calc(100vw-32px))]"
  >
    <div class="flex items-center justify-center gap-4">
      <img src="/logo-128.png" alt="Pomelo Orbit Logo" class="size-14" width="56" height="56" />
      <div>
        <p class="text-base font-normal text-foreground">Pomelo Orbit</p>
        <p class="mt-1 font-mono text-sm text-muted-foreground">
          {{ t('app.versionLabel', { version: runtimeConfig.appVersion }) }}
        </p>
      </div>
    </div>

    <div
      class="flex flex-wrap items-center justify-center gap-x-4 gap-y-2 border-t border-border pt-4 text-sm"
    >
      <a
        class="app-link inline-flex items-center gap-1.5"
        href="https://github.com/leoninew/pomelo-orbit"
        target="_blank"
        rel="noreferrer"
      >
        <Code2 class="size-4" aria-hidden="true" />
        {{ t('app.repository') }}
      </a>
      <a
        class="app-link inline-flex items-center gap-1.5"
        href="https://github.com/leoninew/pomelo-orbit/tree/develop/docs"
        target="_blank"
        rel="noreferrer"
      >
        <BookOpenText class="size-4" aria-hidden="true" />
        {{ t('app.documentation') }}
      </a>
      <a
        class="app-link inline-flex items-center gap-1.5"
        href="https://github.com/leoninew/pomelo-orbit/blob/develop/LICENSE"
        target="_blank"
        rel="noreferrer"
      >
        <Scale class="size-4" aria-hidden="true" />
        MIT License
      </a>
    </div>
  </AppDialog>
</template>

<script setup lang="ts">
  import AppInput from '@/components/ui/AppInput.vue';
  import {
    BookOpenText,
    ChevronDown,
    Check,
    CircleHelp,
    Code2,
    Cloud,
    KeyRound,
    Languages,
    LogOut,
    Monitor,
    Moon,
    Scale,
    Sun,
  } from '@lucide/vue';
  import { computed, reactive, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import AppDialog from '@/components/ui/AppDialog.vue';
  import AppDialogActions from '@/components/ui/AppDialogActions.vue';
  import AppTruncatedText from '@/components/ui/AppTruncatedText.vue';
  import {
    getProjectSwitchListPath,
    primaryNavigation,
    type PrimaryNavigationKey,
  } from '@/navigation';
  import { useAuthStore } from '@/stores/auth';
  import { useProjectStore } from '@/stores/project';
  import { isProjectInitializationPath } from '@/router/projectReadiness';
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
  const isAboutDialogOpen = ref(false);
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
  const passwordSubmitError = ref('');

  const activeProjectLabel = computed(
    () => projectStore.activeProjectName ?? t('project.noProjects')
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

  async function handleProjectMenuOpen() {
    if (projectStore.projects.length === 0 && !projectStore.loading) {
      try {
        await projectStore.fetchProjects();
      } catch (error: unknown) {
        toast.error(error instanceof Error ? error.message : '加载项目列表失败');
      }
    }
  }

  async function handleSwitchProject(projectId: string) {
    if (projectId === projectStore.activeProjectId) {
      return;
    }
    const current = router.currentRoute.value;
    try {
      if (isProjectInitializationPath(current.path)) {
        projectStore.setActiveProject(projectId);
        await router.push({ name: 'ProjectInitialization', params: { id: projectId } });
        return;
      }
      const listPath = getProjectSwitchListPath(current);
      if (listPath) {
        // Leave the old detail before its project watchers can request the new project.
        await router.replace(listPath);
      }
      projectStore.setActiveProject(projectId);
      router.go(0);
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('project.loadFailed'));
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
    passwordSubmitError.value = '';
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
    passwordSubmitError.value = '';
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
      passwordSubmitError.value =
        error instanceof Error ? error.message : t('settings.passwordDialog.changeFailed');
    }
  }

  async function handleLogout() {
    projectStore.clearProjects();
    await authStore.logout();
    router.push('/login');
  }
</script>
