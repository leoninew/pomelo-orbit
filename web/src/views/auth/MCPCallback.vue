<template>
  <div class="flex min-h-screen items-center justify-center bg-background p-4">
    <section
      class="app-surface w-full max-w-sm p-6 text-center"
      :role="isRejected ? 'alert' : 'status'"
      aria-live="polite"
    >
      <div
        class="mx-auto flex size-10 items-center justify-center rounded-full"
        :class="isRejected ? 'bg-destructive/10 text-destructive' : 'bg-primary/10 text-primary'"
      >
        <CircleX v-if="isRejected" class="size-5" aria-hidden="true" />
        <CircleCheckBig v-else class="size-5" aria-hidden="true" />
      </div>
      <p class="mt-4 text-xs font-medium text-muted-foreground">{{ status }}</p>
      <h1 class="mt-2 text-base font-semibold text-foreground">{{ title }}</h1>
      <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ description }}</p>
      <button class="app-button mt-6 h-9 px-3" type="button" @click="closeWindow">
        <X class="size-4" />
        {{ t('common.close') }}
      </button>
    </section>
  </div>
</template>

<script setup lang="ts">
  import { CircleCheckBig, CircleX, X } from '@lucide/vue';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute } from 'vue-router';

  const route = useRoute();
  const { t } = useI18n();
  const isRejected = computed(() => route.query.status === 'error');
  const status = computed(() =>
    t(isRejected.value ? 'mcpCallback.rejected' : 'mcpCallback.authorized')
  );
  const title = computed(() =>
    t(isRejected.value ? 'mcpCallback.rejectedTitle' : 'mcpCallback.authorizedTitle')
  );
  const description = computed(() =>
    t(isRejected.value ? 'mcpCallback.rejectedDescription' : 'mcpCallback.authorizedDescription')
  );

  function closeWindow() {
    window.close();
  }
</script>
