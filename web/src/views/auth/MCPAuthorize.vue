<template>
  <div class="min-h-screen flex items-center justify-center bg-background p-4">
    <div class="app-surface w-full max-w-sm rounded-md p-6 text-center">
      <div v-if="error" class="app-field-error" role="alert">{{ error }}</div>
      <div v-else class="text-sm text-muted-foreground">正在完成 MCP 授权...</div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { onMounted, ref } from 'vue';
  import { useRoute } from 'vue-router';
  import { authApi } from '@/api/auth/auth';

  const route = useRoute();
  const error = ref('');

  onMounted(async () => {
    const callbackUrl = singleQueryValue(route.query.callback_url);
    const state = singleQueryValue(route.query.state);

    if (!isValidRequest(callbackUrl, state)) {
      error.value = 'MCP 授权请求无效。';
      return;
    }

    try {
      const grant = await authApi.createMcpGrant({ callback_url: callbackUrl, state });
      const callback = new URL(callbackUrl);
      callback.searchParams.set('code', grant.code);
      callback.searchParams.set('state', state);
      window.location.replace(callback.toString());
    } catch (reason: unknown) {
      error.value = reason instanceof Error ? reason.message : 'MCP 授权失败。';
    }
  });

  function singleQueryValue(value: unknown): string {
    return typeof value === 'string' ? value.trim() : '';
  }

  function isValidRequest(callbackUrl: string, state: string): boolean {
    if (state.length < 32) {
      return false;
    }
    try {
      const callback = new URL(callbackUrl);
      const hostname = callback.hostname.toLowerCase();
      return (
        callback.protocol === 'http:' &&
        callback.port !== '' &&
        callback.pathname === '/mcp/callback' &&
        callback.search === '' &&
        (hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '::1')
      );
    } catch {
      return false;
    }
  }
</script>
