<template>
  <div class="flex flex-col gap-4">
    <DetailInfoCard :title="t('route.sections.traefikRouters')" actions-class="flex-nowrap">
      <template #actions>
        <ToolbarRoot
          class="flex min-w-0 flex-1 items-center justify-end gap-3"
          :aria-label="t('traefikRoute.toolbar')"
        >
          <SearchControl
            v-model="traefikSearchText"
            class="min-w-0 flex-1"
            :placeholder="t('traefikRoute.searchPlaceholder')"
            :loading="traefikStatus === 'loading'"
            @search="handleTraefikSearch"
          />
          <div class="flex shrink-0 items-center gap-2">
            <button class="app-button-primary h-9 px-3" @click="openDashboard">
              <ExternalLink class="size-4" />
              {{ t('traefikRoute.openDashboard') }}
            </button>
          </div>
        </ToolbarRoot>
      </template>

      <AppLoadingState v-if="traefikStatus === 'loading'" />
      <div v-else-if="traefikStatus === 'error'" class="py-16 text-center">
        <p class="text-sm text-destructive">
          {{ traefikError || t('traefikRoute.toast.loadFailed') }}
        </p>
        <p class="mt-1 text-xs text-muted-foreground">{{ t('traefikRoute.serviceCheckHint') }}</p>
        <button class="app-link mx-auto mt-3 block text-sm" @click="fetchTraefikRoutes">
          {{ t('traefikRoute.retry') }}
        </button>
      </div>
      <AppEmptyState v-else-if="filteredTraefikRoutes.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[1080px]">
          <colgroup>
            <col class="w-[20%]" />
            <col class="w-[10%]" />
            <col class="w-[8%]" />
            <col class="w-[30%]" />
            <col class="w-[18%]" />
            <col class="w-[10%]" />
            <col class="w-[4%]" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ t('traefikRoute.fields.name') }}</th>
              <th>{{ t('traefikRoute.fields.provider') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('traefikRoute.fields.rule') }}</th>
              <th>{{ t('traefikRoute.fields.service') }}</th>
              <th>{{ t('traefikRoute.fields.entrypoints') }}</th>
              <th>{{ t('traefikRoute.fields.protocol') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="traefikRoute in filteredTraefikRoutes" :key="traefikRoute.name">
              <td class="max-w-0 truncate text-foreground" :title="traefikRoute.name">
                {{ traefikRoute.name }}
              </td>
              <td class="whitespace-nowrap text-foreground">{{ traefikRoute.provider }}</td>
              <td>
                <AppBadge
                  variant="status"
                  :tone="traefikRoute.status === 'enabled' ? 'success' : 'default'"
                >
                  {{ traefikRoute.status }}
                </AppBadge>
              </td>
              <td class="max-w-0" :title="traefikRoute.rule">
                <a
                  v-if="
                    !isTCPRouter(traefikRoute) && buildRouteUrl(traefikRoute.rule, traefikRoute.tls)
                  "
                  :href="buildRouteUrl(traefikRoute.rule, traefikRoute.tls)!"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="app-link flex items-center gap-1"
                >
                  <span class="truncate">{{ traefikRoute.rule }}</span>
                  <ExternalLink class="size-3 shrink-0" />
                </a>
                <span v-else class="block truncate text-foreground">{{ traefikRoute.rule }}</span>
              </td>
              <td class="max-w-0 truncate text-foreground" :title="traefikRoute.service">
                {{ traefikRoute.service }}
              </td>
              <td class="max-w-0" :title="traefikRoute.entrypoints.join(', ')">
                <div class="flex flex-nowrap gap-1 overflow-hidden">
                  <AppBadge
                    v-for="entrypoint in traefikRoute.entrypoints"
                    :key="entrypoint"
                    variant="pill"
                  >
                    {{ entrypoint }}
                  </AppBadge>
                </div>
              </td>
              <td class="whitespace-nowrap">
                <AppBadge
                  variant="status"
                  :tone="
                    isTCPRouter(traefikRoute) ? 'warning' : traefikRoute.tls ? 'info' : 'default'
                  "
                >
                  {{ isTCPRouter(traefikRoute) ? 'TCP' : traefikRoute.tls ? 'HTTPS' : 'HTTP' }}
                </AppBadge>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </DetailInfoCard>

    <DetailInfoCard :title="t('route.sections.customConfiguration')" actions-class="flex-nowrap">
      <template #actions>
        <ToolbarRoot
          class="flex min-w-0 flex-1 items-center justify-end gap-3"
          :aria-label="t('route.toolbar')"
        >
          <SearchControl
            v-model="routeSearchText"
            class="min-w-0 flex-1"
            :placeholder="t('route.searchPlaceholder')"
            :loading="routeStatus === 'loading'"
            @search="handleRouteSearch"
          />
          <div class="flex shrink-0 items-center gap-2">
            <button
              class="app-button-primary h-9 px-3"
              :disabled="routeOperating"
              @click="openCreateModal"
            >
              <Plus class="size-4" />
              {{ t('route.addRoute') }}
            </button>
            <button class="app-button h-9 px-3" :disabled="routeOperating" @click="handleSync">
              <RefreshCw class="size-4" :class="{ 'animate-spin': routeOperating }" />
              {{ t('route.syncAll') }}
            </button>
          </div>
        </ToolbarRoot>
      </template>

      <AppLoadingState v-if="routeStatus === 'loading'" />
      <div v-else-if="routeStatus === 'error'" class="py-16 text-center text-destructive">
        <p class="text-sm">{{ routeError || t('route.toast.loadFailed') }}</p>
      </div>
      <AppEmptyState v-else-if="routes.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[1200px]">
          <colgroup>
            <col class="w-[14%]" />
            <col class="w-[16%]" />
            <col class="w-[10%]" />
            <col class="w-[20%]" />
            <col class="w-[8%]" />
            <col class="w-[8%]" />
            <col class="w-[14%]" />
            <col class="w-[10%]" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ t('route.fields.name') }}</th>
              <th>{{ t('route.fields.domain') }}</th>
              <th>{{ t('route.fields.pathPrefix') }}</th>
              <th>{{ t('route.fields.targetUrl') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('route.fields.protocol') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="route in routes" :key="route.id">
              <td>
                <router-link :to="`/route/${route.id}`" class="app-link whitespace-nowrap">
                  {{ route.name }}
                </router-link>
              </td>
              <td>
                <a
                  v-if="route.protocol === 'http'"
                  :href="`${route.https_enabled ? 'https' : 'http'}://${route.domain}`"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="app-link inline-flex items-center gap-1 whitespace-nowrap"
                >
                  {{ route.domain }}
                  <ExternalLink class="size-3" />
                </a>
                <span v-else class="text-foreground">{{ route.domain }}</span>
              </td>
              <td class="whitespace-nowrap text-foreground">
                {{
                  route.protocol === 'tcp'
                    ? `${route.domain}:${route.listen_port}`
                    : route.path_prefix
                }}
              </td>
              <td class="max-w-0 truncate text-foreground" :title="routeTarget(route)">
                {{ routeTarget(route) }}
              </td>
              <td>
                <AppBadge variant="status" :tone="route.enabled ? 'success' : 'default'">
                  {{ route.enabled ? t('route.status.enabled') : t('route.status.disabled') }}
                </AppBadge>
              </td>
              <td>
                <AppBadge variant="status" :tone="route.https_enabled ? 'info' : 'default'">
                  {{ route.protocol === 'tcp' ? 'TCP' : route.https_enabled ? 'HTTPS' : 'HTTP' }}
                </AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ formatTime(route.created_at) }}</td>
              <td class="whitespace-nowrap">
                <div class="flex items-center gap-3">
                  <button class="app-link" :disabled="routeOperating" @click="openEditModal(route)">
                    {{ t('common.edit') }}
                  </button>
                  <button
                    v-if="!route.enabled"
                    class="app-link-success"
                    :disabled="routeOperating"
                    @click="handleEnable(route.id)"
                  >
                    {{ t('route.status.enabled') }}
                  </button>
                  <button
                    v-else
                    class="app-link-warning"
                    :disabled="routeOperating"
                    @click="handleDisable(route.id)"
                  >
                    {{ t('route.status.disabled') }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <ListPagination
        :current="pagination.current"
        :page-size="pagination.pageSize"
        :total="pagination.total"
        :total-pages="totalPages"
        @change-page="goPage"
        @change-page-size="handlePageSizeChange"
      />
    </DetailInfoCard>
  </div>

  <AppDialog v-model:open="isCreateDialogOpen" :title="t('route.addRoute')">
    <div class="space-y-4">
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('route.fields.protocol') }}</label>
          <SelectControl
            :model-value="form.protocol"
            :options="protocolOptions"
            @update:model-value="updateProtocol(form, $event)"
          />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('route.fields.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.name"
            type="text"
            class="app-input"
            :class="errors.name ? 'app-input-error' : ''"
            :placeholder="t('route.hints.name')"
            :aria-invalid="errors.name ? 'true' : undefined"
            @input="errors.name = ''"
          />
          <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
        </div>
      </div>
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
      <div v-if="form.protocol === 'http'" class="space-y-1.5">
        <label class="app-field-label block">{{ t('route.fields.pathPrefix') }}</label>
        <input
          v-model="form.path_prefix"
          type="text"
          class="app-input"
          :placeholder="t('route.hints.pathPrefix')"
        />
      </div>
      <template v-if="form.protocol === 'tcp'">
        <RouteManagedTargetSelect
          protocol="tcp"
          :services="targetServices"
          :service-id="form.service_id"
          :component-name="form.component_name"
          :endpoint-protocol="form.endpoint_protocol"
          :endpoint-container-port="form.endpoint_container_port"
          :listen-port="form.listen_port"
          :service-error="errors.service_id"
          :component-error="errors.component_name"
          :endpoint-error="errors.endpoint_protocol || errors.endpoint_container_port"
          :listen-port-error="errors.listen_port"
          @update:service-id="form.service_id = $event"
          @update:component-name="form.component_name = $event"
          @update:endpoint-protocol="
            form.endpoint_protocol = $event;
            errors.endpoint_protocol = '';
          "
          @update:endpoint-container-port="
            form.endpoint_container_port = $event;
            errors.endpoint_container_port = '';
          "
          @update:listen-port="
            form.listen_port = $event;
            errors.listen_port = '';
          "
        />
      </template>
      <template v-if="form.protocol === 'http'">
        <RouteManagedTargetSelect
          v-if="!form.custom_target"
          protocol="http"
          :services="targetServices"
          :service-id="form.service_id"
          :component-name="form.component_name"
          :endpoint-protocol="form.endpoint_protocol"
          :endpoint-container-port="form.endpoint_container_port"
          :service-error="errors.service_id"
          :component-error="errors.component_name"
          :endpoint-error="errors.endpoint_protocol || errors.endpoint_container_port"
          @update:service-id="form.service_id = $event"
          @update:component-name="form.component_name = $event"
          @update:endpoint-protocol="
            form.endpoint_protocol = $event;
            errors.endpoint_protocol = '';
          "
          @update:endpoint-container-port="
            form.endpoint_container_port = $event;
            errors.endpoint_container_port = '';
          "
        />
        <label class="flex cursor-pointer items-center gap-3">
          <SwitchRoot
            :model-value="form.custom_target"
            class="app-switch-root"
            @update:model-value="setCustomTarget(form, $event)"
          >
            <SwitchThumb class="app-switch-thumb" />
          </SwitchRoot>
          <span class="text-sm text-foreground">{{ t('route.advancedCustomTarget') }}</span>
        </label>
      </template>
      <div v-if="form.protocol === 'http' && form.custom_target" class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('route.fields.targetUrl') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.target_url"
          type="text"
          class="app-input"
          :class="errors.target_url ? 'app-input-error' : ''"
          :placeholder="t('route.hints.targetUrl')"
          :aria-invalid="errors.target_url ? 'true' : undefined"
          @input="errors.target_url = ''"
        />
        <p v-if="errors.target_url" class="app-field-error text-xs">
          {{ errors.target_url }}
        </p>
      </div>
      <label class="flex cursor-pointer items-center gap-3">
        <SwitchRoot v-model="form.enabled" class="app-switch-root">
          <SwitchThumb class="app-switch-thumb" />
        </SwitchRoot>
        <span class="text-sm text-foreground">{{ t('route.status.enabled') }}</span>
      </label>
    </div>
    <p v-if="createSubmitError" class="app-field-error mt-3" role="alert">
      {{ createSubmitError }}
    </p>

    <template #footer>
      <AppDialogActions
        :busy="routeOperating"
        @cancel="isCreateDialogOpen = false"
        @confirm="handleSave"
      />
    </template>
  </AppDialog>

  <AppDialog
    :open="isEditDialogOpen"
    :title="t('route.editRoute')"
    @update:open="handleEditDialogOpenChange"
  >
    <form class="space-y-4" novalidate @submit.prevent="handleEditSave">
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('route.fields.protocol') }}</label>
          <SelectControl
            :model-value="editForm.protocol"
            :options="protocolOptions"
            @update:model-value="updateProtocol(editForm, $event)"
          />
        </div>
        <div class="space-y-1.5">
          <label for="edit-route-name" class="app-field-label block">
            {{ t('route.fields.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="edit-route-name"
            v-model="editForm.name"
            type="text"
            class="app-input"
            :class="editErrors.name ? 'app-input-error' : ''"
            :placeholder="t('route.hints.name')"
            :aria-invalid="editErrors.name ? 'true' : undefined"
            :aria-describedby="editErrors.name ? 'edit-route-name-error' : undefined"
            @input="clearEditError('name')"
          />
          <p
            v-if="editErrors.name"
            id="edit-route-name-error"
            class="app-field-error text-xs"
            role="alert"
          >
            {{ editErrors.name }}
          </p>
        </div>
      </div>
      <div class="space-y-1.5">
        <label for="edit-route-domain" class="app-field-label block">
          {{ t('route.fields.domain') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          id="edit-route-domain"
          v-model="editForm.domain"
          type="text"
          class="app-input"
          :class="editErrors.domain ? 'app-input-error' : ''"
          placeholder="example.com"
          :aria-invalid="editErrors.domain ? 'true' : undefined"
          :aria-describedby="editErrors.domain ? 'edit-route-domain-error' : undefined"
          @input="clearEditError('domain')"
        />
        <p
          v-if="editErrors.domain"
          id="edit-route-domain-error"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ editErrors.domain }}
        </p>
      </div>
      <template v-if="editForm.protocol === 'tcp'">
        <RouteManagedTargetSelect
          protocol="tcp"
          :services="targetServices"
          :service-id="editForm.service_id"
          :component-name="editForm.component_name"
          :endpoint-protocol="editForm.endpoint_protocol"
          :endpoint-container-port="editForm.endpoint_container_port"
          :listen-port="editForm.listen_port"
          :service-error="editErrors.service_id"
          :component-error="editErrors.component_name"
          :endpoint-error="editErrors.endpoint_protocol || editErrors.endpoint_container_port"
          :listen-port-error="editErrors.listen_port"
          @update:service-id="editForm.service_id = $event"
          @update:component-name="editForm.component_name = $event"
          @update:endpoint-protocol="
            editForm.endpoint_protocol = $event;
            editErrors.endpoint_protocol = '';
          "
          @update:endpoint-container-port="
            editForm.endpoint_container_port = $event;
            editErrors.endpoint_container_port = '';
          "
          @update:listen-port="
            editForm.listen_port = $event;
            editErrors.listen_port = '';
          "
        />
      </template>
      <div v-if="editForm.protocol === 'http'" class="space-y-1.5">
        <label for="edit-route-path-prefix" class="app-field-label block">
          {{ t('route.fields.pathPrefix') }}
        </label>
        <input
          id="edit-route-path-prefix"
          v-model="editForm.path_prefix"
          type="text"
          class="app-input"
          :placeholder="t('route.hints.pathPrefix')"
        />
      </div>
      <template v-if="editForm.protocol === 'http'">
        <RouteManagedTargetSelect
          v-if="!editForm.custom_target"
          protocol="http"
          :services="targetServices"
          :service-id="editForm.service_id"
          :component-name="editForm.component_name"
          :endpoint-protocol="editForm.endpoint_protocol"
          :endpoint-container-port="editForm.endpoint_container_port"
          :service-error="editErrors.service_id"
          :component-error="editErrors.component_name"
          :endpoint-error="editErrors.endpoint_protocol || editErrors.endpoint_container_port"
          @update:service-id="editForm.service_id = $event"
          @update:component-name="editForm.component_name = $event"
          @update:endpoint-protocol="
            editForm.endpoint_protocol = $event;
            editErrors.endpoint_protocol = '';
          "
          @update:endpoint-container-port="
            editForm.endpoint_container_port = $event;
            editErrors.endpoint_container_port = '';
          "
        />
        <label class="flex cursor-pointer items-center gap-3">
          <SwitchRoot
            :model-value="editForm.custom_target"
            class="app-switch-root"
            @update:model-value="setCustomTarget(editForm, $event)"
          >
            <SwitchThumb class="app-switch-thumb" />
          </SwitchRoot>
          <span class="text-sm text-foreground">{{ t('route.advancedCustomTarget') }}</span>
        </label>
      </template>
      <div v-if="editForm.protocol === 'http' && editForm.custom_target" class="space-y-1.5">
        <label for="edit-route-target-url" class="app-field-label block">
          {{ t('route.fields.targetUrl') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          id="edit-route-target-url"
          v-model="editForm.target_url"
          type="text"
          class="app-input"
          :class="editErrors.target_url ? 'app-input-error' : ''"
          :placeholder="t('route.hints.targetUrl')"
          :aria-invalid="editErrors.target_url ? 'true' : undefined"
          :aria-describedby="editErrors.target_url ? 'edit-route-target-url-error' : undefined"
          @input="clearEditError('target_url')"
        />
        <p
          v-if="editErrors.target_url"
          id="edit-route-target-url-error"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ editErrors.target_url }}
        </p>
      </div>
      <label class="flex cursor-pointer items-center gap-3">
        <SwitchRoot v-model="editForm.enabled" class="app-switch-root">
          <SwitchThumb class="app-switch-thumb" />
        </SwitchRoot>
        <span class="text-sm text-foreground">{{ t('route.status.enabled') }}</span>
      </label>
    </form>
    <p v-if="editSubmitError" class="app-field-error mt-3" role="alert">
      {{ editSubmitError }}
    </p>

    <template #footer>
      <AppDialogActions :busy="routeOperating" @cancel="closeEditModal" @confirm="handleEditSave" />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { ExternalLink, Plus, RefreshCw } from '@lucide/vue';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { SwitchRoot, SwitchThumb, ToolbarRoot } from 'reka-ui';
  import { routeApi } from '@/api/route/route';
  import { traefikRouteApi } from '@/api/route/traefik';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import RouteManagedTargetSelect from '@/components/RouteManagedTargetSelect.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useRouteTargetServices } from '@/composables/useRouteTargetServices';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import type { RouteResp } from '@/gen/proto/orbit/v1/route/route';
  import type { TraefikRouterResp } from '@/gen/proto/orbit/v1/route/traefik';
  import { useProjectStore } from '@/stores/project';
  import { useToast } from '@/composables/useToast';
  import { formatTime } from '@/utils/time';

  const toast = useToast();
  const { t } = useI18n();
  const router = useRouter();
  const targetUrlPattern = /^https?:\/\/[a-zA-Z0-9.-]+(?::\d+)?$/;
  const projectStore = useProjectStore();
  const { status: routeStatus, error: routeError, execute: executeRoutes } = useStatusAsync();
  const { loading: routeOperating, execute: executeRouteOperation } = useStatusAsync();
  const { status: traefikStatus, error: traefikError, execute: executeTraefik } = useStatusAsync();
  const { services: targetServices, load: loadTargetServices } = useRouteTargetServices();
  const protocolOptions = [
    { value: 'http', label: 'HTTP' },
    { value: 'tcp', label: 'TCP' },
  ];

  const routes = ref<RouteResp[]>([]);
  const traefikRoutes = ref<TraefikRouterResp[]>([]);
  const routeSearchText = ref('');
  const traefikSearchText = ref('');
  const appliedTraefikSearch = ref('');
  const isCreateDialogOpen = ref(false);
  const isEditDialogOpen = ref(false);
  const editingRoute = ref<RouteResp>();
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

  const filteredTraefikRoutes = computed(() => {
    if (!appliedTraefikSearch.value.trim()) {
      return traefikRoutes.value;
    }
    const search = appliedTraefikSearch.value.toLowerCase();
    return traefikRoutes.value.filter(
      (router) =>
        router.name.toLowerCase().includes(search) ||
        router.rule.toLowerCase().includes(search) ||
        router.service.toLowerCase().includes(search) ||
        router.provider.toLowerCase().includes(search)
    );
  });

  const form = reactive({
    name: '',
    protocol: 'http',
    domain: '',
    path_prefix: '/',
    target_url: 'http://',
    custom_target: false,
    listen_port: undefined as number | undefined,
    service_id: '',
    component_name: '',
    endpoint_protocol: '',
    endpoint_container_port: undefined as number | undefined,
    enabled: false,
  });
  const errors = reactive({
    name: '',
    domain: '',
    target_url: '',
    listen_port: '',
    service_id: '',
    component_name: '',
    endpoint_protocol: '',
    endpoint_container_port: '',
  });
  const editForm = reactive({
    name: '',
    protocol: 'http',
    domain: '',
    path_prefix: '/',
    target_url: 'http://',
    custom_target: false,
    listen_port: undefined as number | undefined,
    service_id: '',
    component_name: '',
    endpoint_protocol: '',
    endpoint_container_port: undefined as number | undefined,
    enabled: false,
  });
  const editErrors = reactive({
    name: '',
    domain: '',
    target_url: '',
    listen_port: '',
    service_id: '',
    component_name: '',
    endpoint_protocol: '',
    endpoint_container_port: '',
  });
  const createSubmitError = ref('');
  const editSubmitError = ref('');

  type RouteForm = typeof form;
  type RouteErrors = typeof errors;

  function validateRouteForm(routeForm: RouteForm, routeErrors: RouteErrors) {
    routeErrors.name = /^[a-z][a-z0-9._-]*$/.test(routeForm.name)
      ? ''
      : t('route.validation.nameInvalid');
    routeErrors.domain = routeForm.domain.trim() ? '' : t('route.validation.domainRequired');
    routeErrors.target_url = '';
    routeErrors.listen_port = '';
    routeErrors.service_id = '';
    routeErrors.component_name = '';
    routeErrors.endpoint_protocol = '';
    routeErrors.endpoint_container_port = '';

    if (routeForm.protocol === 'http') {
      if (routeForm.custom_target) {
        routeErrors.target_url = targetUrlPattern.test(routeForm.target_url)
          ? ''
          : t('route.validation.targetUrlInvalid');
      } else {
        setManagedTargetErrors(routeForm, routeErrors);
      }
    } else {
      routeErrors.listen_port = isValidListenPort(routeForm.listen_port)
        ? ''
        : t('route.validation.listenPortInvalid');
      routeErrors.service_id = routeForm.service_id.trim()
        ? ''
        : t('route.validation.serviceIdRequired');
      routeErrors.component_name = routeForm.component_name.trim()
        ? ''
        : t('route.validation.componentNameRequired');
      routeErrors.endpoint_protocol = routeForm.endpoint_protocol.trim()
        ? ''
        : t('route.validation.endpointNameRequired');
      routeErrors.endpoint_container_port = isValidListenPort(routeForm.endpoint_container_port)
        ? ''
        : t('route.validation.endpointNameRequired');
    }
    return !Object.values(routeErrors).some(Boolean);
  }

  function isValidListenPort(port: number | undefined): boolean {
    return port !== undefined && Number.isInteger(port) && port >= 1 && port <= 65535;
  }

  function resetProtocolFields(routeForm: RouteForm) {
    routeForm.service_id = '';
    routeForm.component_name = '';
    routeForm.endpoint_protocol = '';
    routeForm.endpoint_container_port = undefined;
    if (routeForm.protocol === 'http') {
      routeForm.path_prefix ||= '/';
      routeForm.listen_port = undefined;
      return;
    }
    routeForm.path_prefix = '';
    routeForm.target_url = '';
    routeForm.custom_target = false;
  }

  function updateProtocol(routeForm: RouteForm, value: string | number) {
    if (value !== 'http' && value !== 'tcp') {
      return;
    }
    routeForm.protocol = value;
    resetProtocolFields(routeForm);
  }

  function setManagedTargetErrors(routeForm: RouteForm, routeErrors: RouteErrors) {
    const error = t('route.validation.managedTargetRequired');
    routeErrors.service_id = routeForm.service_id.trim() ? '' : error;
    routeErrors.component_name = routeForm.component_name.trim() ? '' : error;
    routeErrors.endpoint_protocol = routeForm.endpoint_protocol.trim() ? '' : error;
    routeErrors.endpoint_container_port = isValidListenPort(routeForm.endpoint_container_port)
      ? ''
      : error;
  }

  function setCustomTarget(routeForm: RouteForm, value: boolean) {
    routeForm.custom_target = value;
    if (value) {
      routeForm.service_id = '';
      routeForm.component_name = '';
      routeForm.endpoint_protocol = '';
      routeForm.endpoint_container_port = undefined;
      routeForm.target_url ||= 'http://';
    } else {
      routeForm.target_url = '';
    }
  }

  async function fetchRoutes() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('route.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeRoutes(async () => {
        const res = await routeApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: routeSearchText.value || undefined,
          project_id: projectId,
        });
        routes.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error(t('route.toast.loadFailed'));
    }
  }

  async function fetchTraefikRoutes() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('traefikRoute.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeTraefik(async () => {
        const data = await traefikRouteApi.list({ project_id: projectId });
        traefikRoutes.value = data.items;
      });
    } catch {
      // The card renders the error state from useStatusAsync.
    }
  }

  function handleRouteSearch() {
    pagination.current = 1;
    fetchRoutes();
  }

  function handleTraefikSearch() {
    appliedTraefikSearch.value = traefikSearchText.value;
    void fetchTraefikRoutes();
  }

  function goPage(page: number) {
    pagination.current = page;
    fetchRoutes();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchRoutes();
  }

  async function openCreateModal() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('route.toast.selectProjectRequired'));
      return;
    }
    let config;
    try {
      config = await traefikRouteApi.getConfig({ project_id: projectId });
      await loadTargetServices(projectId);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.loadFailed'));
      return;
    }
    Object.assign(form, {
      name: '',
      protocol: 'http',
      domain: config?.base_domain ?? '',
      path_prefix: '/',
      target_url: '',
      custom_target: false,
      listen_port: undefined,
      service_id: '',
      component_name: '',
      endpoint_protocol: '',
      endpoint_container_port: undefined,
      enabled: false,
    });
    Object.assign(errors, {
      name: '',
      domain: '',
      target_url: '',
      listen_port: '',
      service_id: '',
      component_name: '',
      endpoint_protocol: '',
      endpoint_container_port: '',
    });
    createSubmitError.value = '';
    isCreateDialogOpen.value = true;
  }

  async function resetEditForm() {
    if (!editingRoute.value) {
      return;
    }
    Object.assign(editForm, {
      name: editingRoute.value.name,
      protocol: editingRoute.value.protocol,
      domain: editingRoute.value.domain,
      path_prefix: editingRoute.value.path_prefix,
      target_url: editingRoute.value.target_url,
      custom_target: editingRoute.value.protocol === 'http' && !editingRoute.value.service_id,
      listen_port: editingRoute.value.listen_port,
      service_id: editingRoute.value.service_id ?? '',
      component_name: editingRoute.value.component_name ?? '',
      endpoint_protocol: editingRoute.value.endpoint_protocol ?? '',
      endpoint_container_port: editingRoute.value.endpoint_container_port,
      enabled: editingRoute.value.enabled,
    });
    Object.assign(editErrors, {
      name: '',
      domain: '',
      target_url: '',
      listen_port: '',
      service_id: '',
      component_name: '',
      endpoint_protocol: '',
      endpoint_container_port: '',
    });
    editSubmitError.value = '';
  }

  function validate() {
    return validateRouteForm(form, errors);
  }

  function validateEditForm() {
    return validateRouteForm(editForm, editErrors);
  }

  function clearEditError(field: keyof RouteErrors) {
    editErrors[field] = '';
  }

  async function openEditModal(route: RouteResp) {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('route.toast.selectProjectRequired'));
      return;
    }
    try {
      await loadTargetServices(projectId);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.loadFailed'));
      return;
    }
    editingRoute.value = route;
    await resetEditForm();
    isEditDialogOpen.value = true;
  }

  function handleEditDialogOpenChange(open: boolean) {
    isEditDialogOpen.value = open;
    if (!open) {
      Object.assign(editErrors, {
        name: '',
        domain: '',
        target_url: '',
        listen_port: '',
        service_id: '',
        component_name: '',
        endpoint_protocol: '',
        endpoint_container_port: '',
      });
      editSubmitError.value = '';
    }
  }

  function closeEditModal() {
    handleEditDialogOpenChange(false);
  }

  async function handleSave() {
    createSubmitError.value = '';
    if (!validate()) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      createSubmitError.value = t('route.toast.selectProjectRequired');
      return;
    }
    try {
      await executeRouteOperation(async () => {
        const created = await routeApi.create(
          {
            name: form.name,
            protocol: form.protocol,
            domain: form.domain,
            path_prefix: form.protocol === 'http' ? form.path_prefix : '',
            target_url: form.protocol === 'http' && form.custom_target ? form.target_url : '',
            listen_port: form.protocol === 'tcp' ? form.listen_port : undefined,
            service_id:
              form.protocol === 'tcp' || !form.custom_target ? form.service_id.trim() : '',
            component_name:
              form.protocol === 'tcp' || !form.custom_target ? form.component_name.trim() : '',
            endpoint_protocol:
              form.protocol === 'tcp' || !form.custom_target ? form.endpoint_protocol.trim() : '',
            endpoint_container_port:
              form.protocol === 'tcp' || !form.custom_target
                ? form.endpoint_container_port
                : undefined,
            enabled: form.enabled,
          },
          { project_id: projectId }
        );
        toast.success(t('route.toast.addSuccess'));
        isCreateDialogOpen.value = false;
        await router.push(`/route/${created.id}`);
      });
    } catch (error) {
      createSubmitError.value = error instanceof Error ? error.message : t('route.toast.addFailed');
    }
  }

  async function handleEditSave() {
    const route = editingRoute.value;
    editSubmitError.value = '';
    if (!route || !validateEditForm()) {
      return;
    }
    try {
      await executeRouteOperation(async () => {
        const updated = await routeApi.update(route.id, {
          name: editForm.name,
          protocol: editForm.protocol,
          domain: editForm.domain,
          path_prefix: editForm.protocol === 'http' ? editForm.path_prefix : '',
          target_url:
            editForm.protocol === 'http' && editForm.custom_target ? editForm.target_url : '',
          listen_port: editForm.protocol === 'tcp' ? editForm.listen_port : undefined,
          service_id:
            editForm.protocol === 'tcp' || !editForm.custom_target
              ? editForm.service_id.trim()
              : '',
          component_name:
            editForm.protocol === 'tcp' || !editForm.custom_target
              ? editForm.component_name.trim()
              : '',
          endpoint_protocol:
            editForm.protocol === 'tcp' || !editForm.custom_target
              ? editForm.endpoint_protocol.trim()
              : '',
          endpoint_container_port:
            editForm.protocol === 'tcp' || !editForm.custom_target
              ? editForm.endpoint_container_port
              : undefined,
          enabled: editForm.enabled,
        });
        routes.value = routes.value.map((item) => (item.id === updated.id ? updated : item));
        editingRoute.value = updated;
        toast.success(t('route.toast.updateSuccess'));
        closeEditModal();
        await fetchRoutes();
        await fetchTraefikRoutes();
      });
    } catch (error) {
      editSubmitError.value =
        error instanceof Error ? error.message : t('route.toast.updateFailed');
    }
  }

  async function handleEnable(id: string) {
    try {
      await executeRouteOperation(async () => {
        await routeApi.enable(id, {});
        toast.success(t('route.toast.enableSuccess'));
        fetchRoutes();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.enableFailed'));
    }
  }

  async function handleDisable(id: string) {
    try {
      await executeRouteOperation(async () => {
        await routeApi.disable(id, {});
        toast.success(t('route.toast.disableSuccess'));
        fetchRoutes();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.toast.disableFailed'));
    }
  }

  async function handleSync() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('route.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeRouteOperation(async () => {
        await routeApi.sync({}, { project_id: projectId });
        toast.success(t('route.syncSuccess'));
        fetchRoutes();
        fetchTraefikRoutes();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('route.syncFailed'));
    }
  }

  function buildRouteUrl(rule: string, tls: boolean): string | null {
    const match = rule.match(/Host\(`([^`]+)`\)/);
    if (!match) {
      return null;
    }
    return `${tls ? 'https' : 'http'}://${match[1]}`;
  }

  function isTCPRouter(router: TraefikRouterResp): boolean {
    return router.rule.startsWith('HostSNI(');
  }

  function routeTarget(route: RouteResp): string {
    return route.target_url;
  }

  async function openDashboard() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('traefikRoute.toast.selectProjectRequired'));
      return;
    }
    try {
      const config = await traefikRouteApi.getConfig({ project_id: projectId });
      window.open(
        `${config.https_enabled ? 'https' : 'http'}://${config.dashboard_domain}/dashboard/`,
        '_blank'
      );
    } catch {
      toast.error(t('traefikRoute.toast.openDashboardFailed'));
    }
  }

  onMounted(() => {
    fetchTraefikRoutes();
    fetchRoutes();
  });
</script>
