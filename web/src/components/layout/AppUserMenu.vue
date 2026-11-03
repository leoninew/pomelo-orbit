<template>
  <DropdownMenuRoot>
    <DropdownMenuTrigger
      class="app-user-menu-trigger flex h-10 min-w-0 cursor-pointer items-center gap-2 rounded-md px-2 text-sm text-foreground outline-none transition-colors hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring data-[state=open]:bg-accent"
      :class="collapsed ? 'w-10 justify-center' : 'flex-1'"
      :aria-label="t('app.userMenuAria')"
      :title="collapsed ? authStore.user?.username : undefined"
    >
      <CircleUserRound class="size-5 shrink-0 text-muted-foreground" />
      <span v-if="!collapsed" class="min-w-0 flex-1 truncate text-left">
        {{ authStore.user?.username }}
      </span>
      <ChevronUp v-if="!collapsed" class="size-4 shrink-0 text-muted-foreground" />
    </DropdownMenuTrigger>
    <DropdownMenuPortal>
      <DropdownMenuContent
        class="app-popover-content min-w-52 max-w-[calc(100vw-32px)] overflow-y-auto"
        side="top"
        align="start"
        :side-offset="8"
        :collision-padding="16"
      >
        <DropdownMenuItem
          class="app-option-item justify-start gap-2"
          @select="isAboutDialogOpen = true"
        >
          <CircleHelp class="size-4" />
          {{ t('app.about') }}
        </DropdownMenuItem>
        <DropdownMenuSeparator class="my-1 h-px bg-border" />
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

  <AppDialog v-model:open="isPasswordDialogOpen" :title="t('settings.passwordDialog.title')">
    <div class="space-y-4">
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('settings.passwordDialog.oldPassword') }}</label>
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
        <label class="app-field-label block">{{ t('settings.passwordDialog.newPassword') }}</label>
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
  import {
    BookOpenText,
    ChevronUp,
    CircleHelp,
    CircleUserRound,
    Code2,
    KeyRound,
    LogOut,
    Scale,
  } from '@lucide/vue';
  import {
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuPortal,
    DropdownMenuRoot,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
  } from 'reka-ui';
  import { reactive, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import AppDialog from '@/components/ui/AppDialog.vue';
  import AppDialogActions from '@/components/ui/AppDialogActions.vue';
  import AppInput from '@/components/ui/AppInput.vue';
  import { useAuthStore } from '@/stores/auth';
  import { useProjectStore } from '@/stores/project';
  import { useToast } from '@/composables/useToast';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import runtimeConfig from '@/config';

  defineProps<{ collapsed?: boolean }>();

  const router = useRouter();
  const authStore = useAuthStore();
  const projectStore = useProjectStore();
  const { t } = useI18n({ useScope: 'global' });
  const toast = useToast();
  const { loading: passwordLoading, execute: executeChangePassword } = useStatusAsync();
  const isPasswordDialogOpen = ref(false);
  const isAboutDialogOpen = ref(false);
  const passwordForm = reactive({ old_password: '', new_password: '', confirm_password: '' });
  const passwordErrors = reactive({ old_password: '', new_password: '', confirm_password: '' });
  const passwordSubmitError = ref('');

  function resetPasswordForm() {
    Object.assign(passwordForm, { old_password: '', new_password: '', confirm_password: '' });
    Object.assign(passwordErrors, { old_password: '', new_password: '', confirm_password: '' });
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
