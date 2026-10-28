<template>
  <div class="min-h-screen flex items-center justify-center bg-muted">
    <div class="text-center">
      <div
        class="inline-block animate-spin rounded-full h-12 w-12 border-4 border-primary border-t-transparent"
      ></div>
      <p class="mt-4 text-muted-foreground">{{ message }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { onMounted, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import { authApi } from '@/api/auth/auth';
  import { useAuthStore } from '@/stores/auth';
  import { useToast } from '@/composables/useToast';
  import { resolveLoginRedirect } from '@/utils/login-redirect';

  const { t } = useI18n();
  const router = useRouter();
  const authStore = useAuthStore();
  const toast = useToast();
  const message = ref(t('login.googleProcessing'));

  onMounted(async () => {
    const params = new URLSearchParams(window.location.search);
    const code = params.get('code');
    const state = params.get('state');
    const error = params.get('error');
    await router.replace('/google/callback');

    if (error || !code || !state) {
      message.value = t('login.googleCancelled');
      setTimeout(() => router.push('/login'), 1500);
      return;
    }

    try {
      const response = await authApi.googleCallback({ code, state });
      authStore.setToken(response.access_token);
      if (!(await authStore.fetchUser())) {
        throw new Error('登录状态已变更，请重试');
      }
      toast.success(t('login.loginSuccess'));
      await router.push(resolveLoginRedirect(router, response.redirect));
    } catch (err: unknown) {
      message.value = err instanceof Error ? err.message : t('login.googleFailed');
      setTimeout(() => router.push('/login'), 2000);
    }
  });
</script>
