<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">
          {{ t('service.detail.title') }}
        </h1>
        <DetailHeaderMeta v-if="service">
          <AppBadge variant="status" :tone="appStatusTone(service.status)">
            {{ service.status }}
          </AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button v-if="service" class="app-button h-9 px-3" :disabled="operating" @click="preview">
          <FileCode2 class="size-4" />
          {{ t('application.detail.actions.preview') }}
        </button>
        <button
          v-if="service"
          class="app-button-primary h-9 px-3"
          :disabled="operating || service.status === 'deploying'"
          @click="deploy"
        >
          <Rocket class="size-4" />
          {{ t('service.actions.deploy') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/services')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppSpinner v-if="loading && !service" class="py-12" />
    <AppEmptyState v-else-if="!service" :message="t('service.detail.notFound')" />

    <template v-else>
      <ServiceBasicInfoCard :service="service" />
      <ServiceEnvironmentCard
        :rows="environmentRows"
        :saved-rows="savedEnvironmentRows"
        :disabled="operating"
        :validate-key="validateEnvironmentKey"
        @update:rows="environmentRows = $event"
        @save="persistEnvironment"
      />
      <ServiceComponentsCard :service="service" />
    </template>

    <AppDrawer
      :open="previewOpen"
      :title="t('application.detail.drawer.composePreview')"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="setPreviewOpen"
    >
      <div class="flex h-full flex-col gap-3 p-6">
        <div v-if="previewLoading" class="flex flex-1 items-center justify-center">
          <AppSpinner />
        </div>
        <div
          v-else-if="previewError"
          class="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
        >
          {{ previewError }}
        </div>
        <div v-else class="min-h-0 flex-1">
          <MonacoEditor
            :model-value="previewContent"
            language="yaml"
            height="100%"
            :readonly="true"
          />
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="setPreviewOpen(false)">
          {{ t('application.detail.actions.close') }}
        </button>
      </template>
    </AppDrawer>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, FileCode2, Rocket } from 'lucide-vue-next';
  import { onMounted, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { serviceApi } from '@/api/service/service';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import {
    cloneEnvironmentVariableRows,
    environmentVariableRowsFromEntries,
    type EnvironmentVariableEntry,
    type EnvironmentVariableListRow,
  } from '@/components/environmentVariableList';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
  import { appStatusTone } from '@/utils/status';
  import ServiceBasicInfoCard from './components/ServiceBasicInfoCard.vue';
  import ServiceComponentsCard from './components/ServiceComponentsCard.vue';
  import ServiceEnvironmentCard from './components/ServiceEnvironmentCard.vue';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const { loading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOperation } = useStatusAsync();
  const { loading: previewLoading, execute: executePreview } = useStatusAsync();
  const service = ref<ServiceResp>();
  const previewOpen = ref(false);
  const previewContent = ref('');
  const previewError = ref('');
  const serviceId = String(route.params.id || '');
  const environmentRows = ref<EnvironmentVariableListRow[]>([]);
  const savedEnvironmentRows = ref<EnvironmentVariableListRow[]>([]);
  const environmentKeyPattern = /^[A-Za-z_][A-Za-z0-9_]*$/;

  function setEnvironmentRows(value: ServiceResp) {
    const rows = environmentVariableRowsFromEntries(value.env, 'service-environment');
    environmentRows.value = rows;
    savedEnvironmentRows.value = cloneEnvironmentVariableRows(rows);
  }

  async function load() {
    try {
      await execute(async () => {
        service.value = await serviceApi.get(serviceId);
        setEnvironmentRows(service.value);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.loadDetailFailed'));
      await router.push('/services');
    }
  }

  function validateEnvironmentKey(key: string) {
    return environmentKeyPattern.test(key) ? undefined : t('environment.validation.invalidKey');
  }

  async function persistEnvironment(entries: EnvironmentVariableEntry[]) {
    try {
      await executeOperation(async () => {
        const updated = await serviceApi.updateEnv(serviceId, {
          env: entries,
        });
        service.value = updated;
        setEnvironmentRows(updated);
        toast.success(t('environment.saved'));
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('environment.saveFailed'));
    }
  }

  async function preview() {
    previewContent.value = '';
    previewError.value = '';
    previewOpen.value = true;
    try {
      await executePreview(async () => {
        const result = await serviceApi.preview(serviceId);
        previewContent.value = result.compose_yaml;
      });
    } catch (error) {
      previewError.value =
        error instanceof Error ? error.message : t('application.toast.loadPreviewFailed');
    }
  }

  function setPreviewOpen(open: boolean) {
    previewOpen.value = open;
    if (!open) {
      previewContent.value = '';
      previewError.value = '';
    }
  }

  async function deploy() {
    try {
      await executeOperation(async () => {
        const result = await serviceApi.deploy(serviceId, { force_recreate: false });
        for (const warning of result.warnings) toast.error(warning);
        toast.success(t('service.toast.deployQueued'));
        if (result.deployment_id) await router.push(`/deployment/${result.deployment_id}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.deployFailed'));
    }
  }

  onMounted(load);
</script>
