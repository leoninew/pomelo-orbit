<template>
  <div class="flex min-h-[calc(100vh-160px)] items-center justify-center px-4 py-12">
    <section class="app-surface max-w-lg px-8 py-10 text-center">
      <img src="/403.png" alt="" class="mx-auto h-72 w-72 object-contain" />
      <h1 class="mt-4 text-2xl font-semibold text-foreground">{{ t('forbidden.title') }}</h1>
      <p class="mt-3 text-sm leading-6 text-muted-foreground">
        {{ t('forbidden.description') }}
      </p>
      <div class="mt-6 flex justify-center gap-3">
        <button class="app-button-primary px-5" @click="router.push('/')">
          {{ t('forbidden.backHome') }}
        </button>
        <button v-if="from" class="app-button px-5" @click="router.back()">
          {{ t('common.back') }}
        </button>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';

  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const from = computed(() => (typeof route.query.from === 'string' ? route.query.from : ''));
</script>
