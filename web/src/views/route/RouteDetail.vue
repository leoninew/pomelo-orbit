<template>
  <div class="flex flex-col gap-4">
    <!-- Header -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">
          {{ routeData?.name ?? t('route.detailTitle') }}
        </h1>
        <DetailHeaderMeta v-if="routeData">
          <AppBadge variant="status" :tone="routeData.enabled ? 'success' : 'default'">
            {{ routeData.enabled ? t('route.status.enabled') : t('route.status.disabled') }}
          </AppBadge>
          <AppBadge variant="status" :tone="routeData.https_enabled ? 'info' : 'default'">
            {{ routeData.https_enabled ? 'HTTPS' : 'HTTP' }}
          </AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
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
          @click="openDeleteModal"
        >
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/routes')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <!-- Loading State -->
    <AppLoadingState v-if="basicInfoLoading" size="section" />

    <!-- Content -->
    <template v-else-if="routeData">
      <!-- Basic Info Card -->
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">{{ t('route.basicInfo') }}</h2>
          <button
            v-if="routeData"
            class="app-button-primary h-9 px-3"
            :disabled="operating"
            @click="openEditModal"
          >
            <Pencil class="size-4" />
            {{ t('common.edit') }}
          </button>
        </div>
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>{{ t('route.fields.name') }}</dt>
            <dd class="text-foreground">{{ routeData.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('route.fields.domain') }}</dt>
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
            <dt>{{ t('route.fields.pathPrefix') }}</dt>
            <dd class="text-foreground">{{ routeData.path_prefix }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('route.fields.targetUrl') }}</dt>
            <dd class="text-foreground">{{ routeData.target_url }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.status') }}</dt>
            <dd>
              <AppBadge variant="status" :tone="routeData.enabled ? 'success' : 'default'">
                {{ routeData.enabled ? t('route.status.enabled') : t('route.status.disabled') }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.createdAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(routeData.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.updatedAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(routeData.updated_at) }}</dd>
          </div>
        </dl>
      </div>

      <!-- HTTPS Config Card -->
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">{{ t('route.httpsConfig') }}</h2>
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
      <p v-if="editSubmitError" class="app-field-error mt-3" role="alert">
        {{ editSubmitError }}
      </p>
    </template>

    <AppDialog v-model:open="isEditDialogOpen" :title="t('route.editRoute')">
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('route.fields.domain') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.domain"
            type="text"
            class="app-input"
            :class="errors.domain ? 'app-input-error' : ''"
            placeholder="example.com"
            :aria-invalid="errors.domain ? 'true' : undefined"
            @input="errors.domain = ''"
          />
          <p v-if="errors.domain" class="app-field-error text-xs">{{ errors.domain }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('route.fields.pathPrefix') }}</label>
          <input v-model="form.path_prefix" type="text" class="app-input" placeholder="/" />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('route.fields.targetUrl') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.target_url"
            type="text"
            class="app-input"
            :class="errors.target_url ? 'app-input-error' : ''"
            placeholder="http://host:port"
            :aria-invalid="errors.target_url ? 'true' : undefined"
            @input="errors.target_url = ''"
          />
          <p v-if="errors.target_url" class="app-field-error text-xs">{{ errors.target_url }}</p>
        </div>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          @cancel="isEditDialogOpen = false"
          @confirm="handleSave"
        />
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
      <p v-if="deleteSubmitError" class="app-field-error mt-3" role="alert">
        {{ deleteSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          variant="destructive"
          @cancel="isDeleteDialogOpen = false"
          @confirm="handleDelete"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, ExternalLink, Pencil, Power, PowerOff, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import type { RouteResp } from '@/gen/proto/orbit/v1/route/route';
  import { routeApi } from '@/api/route/route';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
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
  const editSubmitError = ref('');
  const deleteSubmitError = ref('');

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
      router.push('/routes');
    }
  }

  function openEditModal() {
    if (!routeData.value) {
      return;
    }
    Object.assign(form, {
      name: routeData.value.name,
      domain: routeData.value.domain,
      path_prefix: routeData.value.path_prefix,
      target_url: routeData.value.target_url,
      enabled: routeData.value.enabled,
    });
    Object.assign(errors, { domain: '', target_url: '' });
    editSubmitError.value = '';
    isEditDialogOpen.value = true;
  }

  function openDeleteModal() {
    deleteSubmitError.value = '';
    isDeleteDialogOpen.value = true;
  }

  async function handleSave() {
    editSubmitError.value = '';
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
      editSubmitError.value =
        error instanceof Error ? error.message : t('route.toast.updateFailed');
    }
  }

  async function handleEnable() {
    try {
      await executeOp(async () => {
        await routeApi.enable(routeId, {});
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
        await routeApi.disable(routeId, {});
        toast.success(t('route.toast.disableSuccess'));
        fetchRoute();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.disableFailed'));
    }
  }

  async function handleDelete() {
    deleteSubmitError.value = '';
    try {
      await executeOp(async () => {
        await routeApi.delete(routeId);
        toast.success(t('route.toast.deleteSuccess'));
        router.push('/routes');
      });
    } catch (error) {
      deleteSubmitError.value =
        error instanceof Error ? error.message : t('route.toast.deleteFailed');
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
        await routeApi.enableLetsencrypt(routeId, {});
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
        await routeApi.enableMkcert(routeId, {});
        toast.success(t('route.toast.mkcertEnabled'));
        fetchRoute();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.operationFailed'));
    }
  }

  onMounted(fetchRoute);
</script>
