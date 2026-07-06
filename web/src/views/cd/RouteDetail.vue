<template>
  <div class="flex flex-col gap-4">
    <!-- Header -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <h1 class="text-xl font-semibold text-foreground">
          {{ routeData?.name ?? t('route.detailTitle') }}
        </h1>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button v-if="routeData" class="app-button-primary h-9 px-3" @click="openEditModal">
          <Pencil class="size-4" />
          {{ t('common.edit') }}
        </button>
        <button
          v-if="routeData && !routeData.enabled"
          class="app-button h-9 px-3"
          :disabled="operating"
          @click="handleEnable"
        >
          <Power class="size-4" />
          {{ t('route.status.enabled') }}
        </button>
        <button
          v-else-if="routeData"
          class="app-button h-9 px-3"
          :disabled="operating"
          @click="handleDisable"
        >
          <PowerOff class="size-4" />
          {{ t('route.status.disabled') }}
        </button>
        <button
          v-if="routeData"
          class="app-button-danger h-9 px-3"
          :disabled="operating"
          @click="isDeleteDialogOpen = true"
        >
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/cd/routes')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <!-- Loading State -->
    <AppSpinner v-if="basicInfoLoading" class="py-12" />

    <!-- Content -->
    <template v-else-if="routeData">
      <!-- Basic Info Card -->
      <div class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">{{ t('route.basicInfo') }}</h2>
        </div>
        <dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('route.fields.name') }}</dt>
            <dd class="text-foreground">{{ routeData.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('route.fields.domain') }}</dt>
            <dd>
              <a
                :href="`${routeData.https_enabled ? 'https' : 'http'}://${routeData.domain}`"
                target="_blank"
                class="app-link inline-flex items-center gap-1"
              >
                {{ routeData.domain }}
                <ExternalLink class="size-3" />
              </a>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('route.fields.pathPrefix') }}</dt>
            <dd class="text-foreground">{{ routeData.path_prefix }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('route.fields.targetUrl') }}</dt>
            <dd class="text-foreground">{{ routeData.target_url }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.status') }}</dt>
            <dd>
              <AppBadge variant="status" :tone="routeData.enabled ? 'success' : 'default'">
                {{ routeData.enabled ? t('route.status.enabled') : t('route.status.disabled') }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.createdAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(routeData.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.updatedAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(routeData.updated_at) }}</dd>
          </div>
        </dl>
      </div>

      <!-- HTTPS Config Card -->
      <div class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">{{ t('route.httpsConfig') }}</h2>
        </div>
        <div class="space-y-4 px-5 py-4">
          <div class="flex items-center justify-between rounded-md bg-muted/30 p-3">
            <div>
              <p class="text-sm text-foreground">{{ t('route.currentStatus') }}</p>
              <p class="text-sm text-muted-foreground">
                {{ routeData.https_enabled ? t('route.httpsEnabled') : t('route.httpsDisabled') }}
              </p>
            </div>
            <AppBadge variant="status" :tone="routeData.https_enabled ? 'info' : 'default'">
              {{ routeData.https_enabled ? 'HTTPS' : 'HTTP' }}
            </AppBadge>
          </div>

          <div v-if="!routeData.https_enabled" class="space-y-2">
            <button
              v-if="canUseLetsencrypt"
              class="app-action-item"
              :disabled="operating"
              @click="handleEnableLetsencrypt"
            >
              <p class="text-sm text-foreground">{{ t('route.enableLetsencrypt') }}</p>
              <p class="text-sm text-muted-foreground">{{ t('route.letsencryptHint') }}</p>
            </button>
            <button class="app-action-item" :disabled="operating" @click="handleEnableMkcert">
              <p class="text-sm text-foreground">{{ t('route.enableMkcert') }}</p>
              <p class="text-sm text-muted-foreground">{{ t('route.mkcertHint') }}</p>
            </button>
            <label class="app-action-item cursor-pointer">
              <p class="text-sm text-foreground">{{ t('route.uploadCustomCert') }}</p>
              <p class="text-sm text-muted-foreground">{{ t('route.uploadCustomCertHint') }}</p>
              <input type="file" accept=".pem" class="hidden" @change="handleCertUpload" />
            </label>
          </div>

          <div v-else class="flex justify-end">
            <button class="app-button-danger" :disabled="operating" @click="handleDisableHttps">
              {{ t('route.disableHttps') }}
            </button>
          </div>
        </div>
      </div>
    </template>

    <AppDialog v-model:open="isEditDialogOpen" :title="t('route.editRoute')">
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('route.fields.domain') }}</label>
          <input
            v-model="form.domain"
            type="text"
            class="app-input"
            :class="errors.domain ? 'app-input-error' : ''"
            placeholder="example.com"
          />
          <p v-if="errors.domain" class="app-field-error text-xs">{{ errors.domain }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('route.fields.pathPrefix') }}</label>
          <input v-model="form.path_prefix" type="text" class="app-input" placeholder="/" />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('route.fields.targetUrl') }}</label>
          <input
            v-model="form.target_url"
            type="text"
            class="app-input"
            :class="errors.target_url ? 'app-input-error' : ''"
            placeholder="http://host:port"
          />
          <p v-if="errors.target_url" class="app-field-error text-xs">{{ errors.target_url }}</p>
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isEditDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="handleSave">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('route.deleteRoute')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('route.deleteConfirm', { domain: routeData?.domain }) }}
      </p>
      <template #footer>
        <button class="app-button" @click="isDeleteDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-destructive" :disabled="operating" @click="handleDelete">
          {{ t('common.delete') }}
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, ExternalLink, Pencil, Power, PowerOff, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import type { RouteResp } from '@/gen/proto/orbit/api/v1/route';
  import { routeApi } from '@/api/cd/route';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { formatTime } from '@/utils/time';

  const currentRoute = useRoute();
  const router = useRouter();
  const routeId = currentRoute.params.id as string;
  const toast = useToast();
  const { t } = useI18n();

  const { loading: basicInfoLoading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const routeData = ref<RouteResp>();
  const isEditDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);

  const form = reactive({
    name: '',
    domain: '',
    path_prefix: '/',
    target_url: '',
    enabled: false,
  });
  const errors = reactive({ domain: '', target_url: '' });

  const canUseLetsencrypt = computed(() => {
    if (!routeData.value) {
      return false;
    }
    const d = routeData.value.domain;
    return (
      d !== 'localhost' &&
      !d.endsWith('.localhost') &&
      !d.endsWith('.lvh.me') &&
      !/^\d+\.\d+\.\d+\.\d+$/.test(d)
    );
  });

  async function fetchRoute() {
    try {
      await execute(async () => {
        const data = await routeApi.get(routeId);
        routeData.value = data;
        Object.assign(form, {
          name: data.name,
          domain: data.domain,
          path_prefix: data.path_prefix,
          target_url: data.target_url,
          enabled: data.enabled,
        });
      });
    } catch {
      toast.error(t('route.toast.loadDetailFailed'));
      router.push('/cd/routes');
    }
  }

  function openEditModal() {
    Object.assign(errors, { domain: '', target_url: '' });
    isEditDialogOpen.value = true;
  }

  async function handleSave() {
    errors.domain = form.domain.trim() ? '' : t('route.validation.domainRequired');
    errors.target_url = /^https?:\/\/[a-zA-Z0-9.-]+:\d+$/.test(form.target_url)
      ? ''
      : t('route.validation.targetUrlInvalid');
    if (errors.domain || errors.target_url) {
      return;
    }
    try {
      await executeOp(async () => {
        const updateData = {
          domain: form.domain,
          path_prefix: form.path_prefix,
          target_url: form.target_url,
          enabled: form.enabled,
        };
        await routeApi.update(routeId, updateData);
        toast.success(t('route.toast.updateSuccess'));
        isEditDialogOpen.value = false;
        fetchRoute();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.updateFailed'));
    }
  }

  async function handleEnable() {
    try {
      await executeOp(async () => {
        await routeApi.enable(routeId);
        toast.success(t('route.toast.enableSuccess'));
        fetchRoute();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.enableFailed'));
    }
  }

  async function handleDisable() {
    try {
      await executeOp(async () => {
        await routeApi.disable(routeId);
        toast.success(t('route.toast.disableSuccess'));
        fetchRoute();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.disableFailed'));
    }
  }

  async function handleDelete() {
    try {
      await executeOp(async () => {
        await routeApi.delete(routeId);
        toast.success(t('route.toast.deleteSuccess'));
        router.push('/cd/routes');
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.deleteFailed'));
    }
  }

  async function handleCertUpload(event: Event) {
    const file = (event.target as HTMLInputElement).files?.[0];
    if (!file) {
      return;
    }
    try {
      await executeOp(async () => {
        await routeApi.uploadCert(routeId, file);
        toast.success(t('route.toast.certUploadSuccess'));
        fetchRoute();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.certUploadFailed'));
    }
  }

  async function handleDisableHttps() {
    try {
      await executeOp(async () => {
        await routeApi.disableHttps(routeId);
        toast.success(t('route.toast.httpsDisabled'));
        fetchRoute();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.operationFailed'));
    }
  }

  async function handleEnableLetsencrypt() {
    try {
      await executeOp(async () => {
        await routeApi.enableLetsencrypt(routeId);
        toast.success(t('route.toast.letsencryptEnabled'));
        fetchRoute();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.operationFailed'));
    }
  }

  async function handleEnableMkcert() {
    try {
      await executeOp(async () => {
        await routeApi.enableMkcert(routeId);
        toast.success(t('route.toast.mkcertEnabled'));
        fetchRoute();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.operationFailed'));
    }
  }

  onMounted(fetchRoute);
</script>
