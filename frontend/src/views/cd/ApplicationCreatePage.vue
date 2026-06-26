<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="space-y-1">
        <h1 class="text-xl font-semibold text-foreground">
          {{ t('application.createWizard.title') }}
        </h1>
        <p class="text-sm text-muted-foreground">
          {{ t('application.createWizard.description') }}
        </p>
      </div>
      <button class="app-button h-9 px-4" @click="router.push('/cd/applications')">
        <ArrowLeft class="size-4" />
        {{ t('common.back') }}
      </button>
    </div>

    <div class="app-surface p-5">
      <ApplicationCreateWizard :project-id="projectStore.activeProjectId" @created="handleCreated" />
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft } from 'lucide-vue-next';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import ApplicationCreateWizard from '@/components/ApplicationCreateWizard.vue';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';

  const { t } = useI18n();
  const router = useRouter();
  const toast = useToast();
  const projectStore = useProjectStore();

  function handleCreated() {
    toast.success(t('application.toast.createSuccess'));
  }
</script>
