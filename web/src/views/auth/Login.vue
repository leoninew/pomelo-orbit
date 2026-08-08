<template>
  <div
    class="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary/10 via-background to-accent/10 p-4"
  >
    <div class="w-full max-w-md">
      <div class="app-surface rounded-lg p-8">
        <div class="text-center mb-8">
          <h1 class="text-base font-semibold text-foreground mb-2">{{ t('login.title') }}</h1>
          <p class="text-muted-foreground">{{ t('login.subtitle') }}</p>
        </div>

        <form class="space-y-4" @submit.prevent="handleLogin">
          <div class="space-y-1.5">
            <label for="username" class="app-field-label block">{{ t('login.username') }}</label>
            <input
              id="username"
              v-model="form.username"
              type="text"
              class="app-input"
              :class="errors.username ? 'app-input-error' : ''"
              :placeholder="t('login.usernamePlaceholder')"
              :disabled="sessionExpired"
              @input="errors.username = ''"
            />
            <p v-if="errors.username" class="app-field-error text-xs">{{ errors.username }}</p>
          </div>

          <div class="space-y-1.5">
            <label for="password" class="app-field-label block">{{ t('login.password') }}</label>
            <div class="relative">
              <input
                id="password"
                v-model="form.password"
                :type="showPassword ? 'text' : 'password'"
                class="app-input pr-10"
                :class="errors.password ? 'app-input-error' : ''"
                :placeholder="t('login.passwordPlaceholder')"
                :disabled="sessionExpired"
                @input="errors.password = ''"
                @keydown.enter="handleLogin"
              />
              <button
                type="button"
                class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground transition-colors hover:text-foreground"
                :disabled="sessionExpired"
                @click="showPassword = !showPassword"
              >
                <Eye v-if="!showPassword" class="size-4" />
                <EyeOff v-else class="size-4" />
              </button>
            </div>
            <p v-if="errors.password" class="app-field-error text-xs">{{ errors.password }}</p>
          </div>

          <div v-if="turnstileEnabled" class="space-y-1.5">
            <label class="app-field-label block">{{ t('login.verification') }}</label>
            <div ref="turnstileContainer" class="flex justify-center"></div>
            <p v-if="errors.verification" class="app-field-error text-xs">
              {{ errors.verification }}
            </p>
          </div>

          <div class="space-y-1.5">
            <button
              type="submit"
              class="app-button-primary flex w-full items-center justify-center gap-2"
              :disabled="!canSubmit || loading"
            >
              {{ loginButtonLabel }}
            </button>
            <p v-if="sessionExpired" class="app-field-error text-center text-xs">
              {{ t('login.sessionExpired') }}
            </p>
            <p v-else-if="initError" class="app-field-error text-center text-xs">
              {{ t('login.initFailed') }}
            </p>
          </div>

          <div class="relative my-6">
            <div class="absolute inset-0 flex items-center">
              <div class="w-full border-t border-border"></div>
            </div>
            <div class="relative flex justify-center text-xs uppercase">
              <span class="bg-background px-2 text-muted-foreground">{{ t('login.or') }}</span>
            </div>
          </div>

          <button
            type="button"
            class="app-button-outline flex w-full items-center justify-center gap-2"
            :disabled="sessionExpired"
            @click="handleGoogleLogin"
          >
            <svg class="size-5" viewBox="0 0 24 24" aria-hidden="true">
              <path
                fill="#4285F4"
                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
              />
              <path
                fill="#34A853"
                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
              />
              <path
                fill="#FBBC05"
                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z"
              />
              <path
                fill="#EA4335"
                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
              />
            </svg>
            {{ t('login.googleLogin') }}
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { Eye, EyeOff } from '@lucide/vue';
  import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import { authApi } from '@/api/auth/auth';
  import { buildApiUrl } from '@/config';
  import { useAuthStore } from '@/stores/auth';
  import { useToast } from '@/composables/useToast';
  import { ApiError } from '@/utils/request';
  import type { TurnstileConfigResp } from '@/gen/proto/orbit/v1/auth/auth';

  const TURNSTILE_SCRIPT_ID = 'cloudflare-turnstile-script';
  const TURNSTILE_SCRIPT_SRC =
    'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';
  const CSRF_SESSION_MS = 3 * 60 * 1000;

  const { t } = useI18n();
  const router = useRouter();
  const route = useRoute();
  const authStore = useAuthStore();
  const toast = useToast();

  const form = reactive({
    username: '',
    password: '',
  });

  const errors = reactive({
    username: '',
    password: '',
    verification: '',
  });

  const showPassword = ref(false);
  const loading = ref(false);
  const ready = ref(false);
  const initError = ref(false);
  const sessionExpired = ref(false);
  const csrfToken = ref('');
  const turnstileEnabled = ref(false);
  const turnstileSiteKey = ref('');
  const turnstileToken = ref('');
  const turnstileContainer = ref<HTMLElement>();
  const turnstileWidgetId = ref('');
  let sessionTimer: ReturnType<typeof setTimeout> | undefined;

  const canSubmit = computed(() => ready.value && !sessionExpired.value && !initError.value);
  const loginButtonLabel = computed(() => {
    // Only the pre-ready loading path uses "initializing"; expired sessions stay on "Login" but disabled.
    if (!ready.value && !initError.value && !sessionExpired.value) {
      return t('login.initializing');
    }
    return t('login.loginButton');
  });

  onMounted(async () => {
    try {
      const [csrfResponse, turnstileConfig] = await Promise.all([
        authApi.getCsrfToken(),
        authApi.getTurnstileConfig(),
      ]);
      if (!csrfResponse.token) {
        throw new Error('empty csrf token');
      }
      csrfToken.value = csrfResponse.token;
      applyTurnstileConfig(turnstileConfig);
      await renderTurnstile();
      ready.value = true;
      startSessionTimer();
    } catch (err) {
      console.error('Failed to initialize login:', err);
      initError.value = true;
      toast.error(t('login.initFailed'));
    }
  });

  onBeforeUnmount(() => {
    clearSessionTimer();
    if (turnstileWidgetId.value && window.turnstile) {
      window.turnstile.remove(turnstileWidgetId.value);
    }
  });

  function startSessionTimer() {
    clearSessionTimer();
    sessionTimer = setTimeout(() => {
      sessionExpired.value = true;
      csrfToken.value = '';
    }, CSRF_SESSION_MS);
  }

  function clearSessionTimer() {
    if (sessionTimer !== undefined) {
      clearTimeout(sessionTimer);
      sessionTimer = undefined;
    }
  }

  function applyTurnstileConfig(config: TurnstileConfigResp) {
    turnstileEnabled.value = config.enabled;
    turnstileSiteKey.value = config.site_key;
  }

  async function renderTurnstile() {
    if (!turnstileEnabled.value) {
      return;
    }
    await nextTick();
    await loadTurnstileScript();
    if (!window.turnstile || !turnstileContainer.value) {
      throw new Error('Turnstile failed to load');
    }
    turnstileWidgetId.value = window.turnstile.render(turnstileContainer.value, {
      sitekey: turnstileSiteKey.value,
      callback: (token: string) => {
        turnstileToken.value = token;
        errors.verification = '';
      },
      'expired-callback': () => {
        turnstileToken.value = '';
        errors.verification = t('login.verificationExpired');
      },
      'error-callback': () => {
        turnstileToken.value = '';
        errors.verification = t('login.verificationFailed');
      },
    });
  }

  function loadTurnstileScript() {
    if (window.turnstile) {
      return Promise.resolve();
    }
    const existingScript = document.getElementById(TURNSTILE_SCRIPT_ID) as HTMLScriptElement | null;
    if (existingScript) {
      return waitForTurnstile(existingScript);
    }
    const script = document.createElement('script');
    script.id = TURNSTILE_SCRIPT_ID;
    script.src = TURNSTILE_SCRIPT_SRC;
    script.async = true;
    script.defer = true;
    document.head.appendChild(script);
    return waitForTurnstile(script);
  }

  function waitForTurnstile(script: HTMLScriptElement) {
    return new Promise<void>((resolve, reject) => {
      if (window.turnstile) {
        resolve();
        return;
      }
      script.addEventListener('load', () => resolve(), { once: true });
      script.addEventListener('error', () => reject(new Error('Turnstile script failed to load')), {
        once: true,
      });
    });
  }

  function resetTurnstile() {
    turnstileToken.value = '';
    if (turnstileEnabled.value && window.turnstile && turnstileWidgetId.value) {
      window.turnstile.reset(turnstileWidgetId.value);
    }
  }

  function handleGoogleLogin() {
    if (sessionExpired.value) {
      return;
    }
    const destination = new URL(buildApiUrl('/api/auth/google'), window.location.origin);
    const redirect = redirectTarget();
    if (redirect !== '/') {
      destination.searchParams.set('redirect', redirect);
    }
    window.location.assign(destination.toString());
  }

  function validate() {
    errors.username = form.username.trim() ? '' : t('login.usernameRequired');
    errors.password = form.password.trim() ? '' : t('login.passwordRequired');
    errors.verification =
      turnstileEnabled.value && !turnstileToken.value ? t('login.verificationRequired') : '';
    return !errors.username && !errors.password && !errors.verification;
  }

  async function handleLogin() {
    if (sessionExpired.value) {
      toast.error(t('login.sessionExpired'));
      return;
    }
    if (!canSubmit.value || !csrfToken.value) {
      toast.error(t('login.refreshRequired'));
      return;
    }
    if (!validate()) {
      return;
    }

    loading.value = true;
    try {
      await authStore.login(form.username, form.password, csrfToken.value, turnstileToken.value);
      toast.success(t('login.loginSuccess'));
      await router.push(redirectTarget());
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('login.loginFailed');
      toast.error(message);
      resetTurnstile();

      const isRateLimited = err instanceof ApiError && err.status === 429;
      if (isRateLimited) {
        return;
      }

      // CSRF invalid/expired: do not silently re-issue; force page refresh.
      const isCsrfFailure =
        err instanceof ApiError &&
        err.status === 400 &&
        /token is invalid or expired|refresh the page|令牌|刷新/i.test(message);
      if (isCsrfFailure) {
        sessionExpired.value = true;
        csrfToken.value = '';
        clearSessionTimer();
      }
    } finally {
      loading.value = false;
    }
  }

  function redirectTarget() {
    const redirect = route.query.redirect;
    if (typeof redirect === 'string' && redirect.startsWith('/') && !redirect.startsWith('//')) {
      return redirect;
    }
    return '/';
  }
</script>
