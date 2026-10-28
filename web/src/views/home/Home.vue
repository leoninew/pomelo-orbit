<template>
  <div class="home-workspace space-y-8">
    <div class="home-overview">
      <button
        v-for="(card, index) in overviewCards"
        :key="card.label"
        :aria-label="`查看${card.label}详情，当前${card.value}个`"
        class="home-metric text-left"
        @click="router.push(card.path)"
      >
        <div class="home-metric-copy">
          <p class="text-sm text-muted-foreground">{{ card.label }}</p>
          <span class="home-metric-value text-base tabular-nums text-foreground">
            {{ card.value }}
          </span>
          <span class="text-xs text-muted-foreground">{{ card.description }}</span>
        </div>
        <img :src="cardImages[index]" :alt="card.label" class="home-metric-image" />
      </button>
    </div>

    <div class="home-records">
      <section class="min-w-0">
        <div class="home-record-header">
          <h2 class="text-base font-normal text-foreground">{{ t('home.recentBuilds') }}</h2>
          <button
            class="home-view-all app-link inline-flex items-center gap-1 text-xs"
            @click="router.push('/pipeline-run')"
          >
            {{ t('common.viewAll') }}
            <ArrowRight class="size-4" />
          </button>
        </div>
        <AppLoadingState v-if="status === 'loading'" />
        <div v-else-if="status === 'error'" class="py-16 text-center text-destructive">
          <p class="text-sm">{{ error || t('common.loadFailed') }}</p>
        </div>
        <AppEmptyState v-else-if="recentRuns.length === 0" />
        <div v-else class="overflow-x-auto">
          <table class="home-record-table app-data-table table-fixed">
            <colgroup>
              <col class="w-[46%]" />
              <col class="w-[22%]" />
              <col class="w-[32%]" />
            </colgroup>
            <thead>
              <tr>
                <th>{{ t('home.repository') }}</th>
                <th>{{ t('common.createdAt') }}</th>
                <th>{{ t('common.status') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="run in recentRuns" :key="run.id">
                <td class="overflow-hidden">
                  <div class="flex min-w-0 items-center gap-2">
                    <span class="home-entity-icon" aria-hidden="true">
                      <FolderGit2 class="size-4" />
                    </span>
                    <AppTruncatedText :text="run.repository_name" class="min-w-0" as-child>
                      <button
                        class="app-link home-entity-link"
                        @click="router.push(`/repository/${run.repository_id}`)"
                      >
                        {{ run.repository_name }}
                      </button>
                    </AppTruncatedText>
                  </div>
                </td>
                <td class="home-record-time">
                  <AppTableTime :time="run.created_at" />
                </td>
                <td>
                  <AppBadge variant="status" :tone="statusTone(run.status)">
                    {{ run.status }}
                  </AppBadge>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="min-w-0">
        <div class="home-record-header">
          <h2 class="text-base font-normal text-foreground">{{ t('home.recentDeploys') }}</h2>
          <button
            class="home-view-all app-link inline-flex items-center gap-1 text-xs"
            @click="router.push('/deployments')"
          >
            {{ t('common.viewAll') }}
            <ArrowRight class="size-4" />
          </button>
        </div>
        <AppLoadingState v-if="status === 'loading'" />
        <div v-else-if="status === 'error'" class="py-16 text-center text-destructive">
          <p class="text-sm">{{ error || t('common.loadFailed') }}</p>
        </div>
        <AppEmptyState v-else-if="recentDeploys.length === 0" />
        <div v-else class="overflow-x-auto">
          <table class="home-record-table app-data-table table-fixed">
            <colgroup>
              <col class="w-[46%]" />
              <col class="w-[22%]" />
              <col class="w-[32%]" />
            </colgroup>
            <thead>
              <tr>
                <th>{{ t('home.application') }}</th>
                <th>{{ t('home.startTime') }}</th>
                <th>{{ t('common.status') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="deployment in recentDeploys" :key="deployment.id">
                <td class="overflow-hidden">
                  <div class="flex min-w-0 items-center gap-2">
                    <span class="home-entity-icon" aria-hidden="true">
                      <Server class="size-4" />
                    </span>
                    <AppTruncatedText :text="deployment.application_name" class="min-w-0" as-child>
                      <button
                        class="app-link home-entity-link"
                        @click="router.push(`/application/${deployment.application_id}`)"
                      >
                        {{ deployment.application_name }}
                      </button>
                    </AppTruncatedText>
                  </div>
                </td>
                <td class="home-record-time">
                  <AppTableTime :time="deployment.started_at" />
                </td>
                <td>
                  <AppBadge variant="status" :tone="statusTone(deployment.status)">
                    {{ deployment.status }}
                  </AppBadge>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ArrowRight, FolderGit2, Server } from '@lucide/vue';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import { applicationApi } from '@/api/application/application';
  import { deploymentApi } from '@/api/deployment/deployment';
  import { pipelineRunApi } from '@/api/pipeline_run/pipeline_run';
  import { repositoryApi } from '@/api/repository/repository';
  import AppBadge from '@/components/ui/AppBadge.vue';
  import AppEmptyState from '@/components/ui/AppEmptyState.vue';
  import AppLoadingState from '@/components/ui/AppLoadingState.vue';
  import AppTableTime from '@/components/ui/AppTableTime.vue';
  import AppTruncatedText from '@/components/ui/AppTruncatedText.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useAuthStore } from '@/stores/auth';
  import { useProjectStore } from '@/stores/project';
  import type { DeploymentResp } from '@/gen/proto/orbit/v1/deployment/deployment';
  import type { PipelineRunResp } from '@/gen/proto/orbit/v1/pipeline_run/pipeline_run';
  import { statusTone } from '@/utils/status';
  import { getTodayStart } from '@/utils/time';
  import image1 from '@/assets/images/1.png';
  import image2 from '@/assets/images/2.png';
  import image3 from '@/assets/images/3.png';
  import image4 from '@/assets/images/4.png';

  const router = useRouter();
  const toast = useToast();
  const authStore = useAuthStore();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { t } = useI18n({ useScope: 'global' });

  const pipelineStats = reactive({ projectCount: 0, todayRuns: 0 });
  const deploymentStats = reactive({ applicationCount: 0, todayDeploys: 0 });
  const recentRuns = ref<PipelineRunResp[]>([]);
  const recentDeploys = ref<DeploymentResp[]>([]);

  const cardImages = [image1, image2, image3, image4];

  const overviewCards = computed(() => [
    {
      label: t('home.repositories'),
      value: pipelineStats.projectCount,
      description: t('home.repositoriesDesc'),
      path: '/repository',
    },
    {
      label: t('home.todayBuilds'),
      value: pipelineStats.todayRuns,
      description: t('home.todayBuildsDesc'),
      path: '/pipeline-run',
    },
    {
      label: t('home.applications'),
      value: deploymentStats.applicationCount,
      description: t('home.applicationsDesc'),
      path: '/applications',
    },
    {
      label: t('home.todayDeploys'),
      value: deploymentStats.todayDeploys,
      description: t('home.todayDeploysDesc'),
      path: '/deployments',
    },
  ]);

  function resetOverview() {
    pipelineStats.projectCount = 0;
    pipelineStats.todayRuns = 0;
    deploymentStats.applicationCount = 0;
    deploymentStats.todayDeploys = 0;
    recentRuns.value = [];
    recentDeploys.value = [];
  }

  async function refresh() {
    try {
      await execute(async () => {
        await projectStore.fetchProjects();

        const activeProjectId = projectStore.activeProjectId;
        if (!activeProjectId) {
          resetOverview();
          return;
        }

        const todayStart = getTodayStart();
        const todayEnd = todayStart.add(1, 'day');

        // 并行请求所有数据以提升性能
        const [
          repositoryRes,
          pipelineRunsRes,
          pipelineTodayRunsRes,
          applicationRes,
          deploymentTodayRes,
          deploymentRecentRes,
        ] = await Promise.all([
          repositoryApi.list(activeProjectId, { per_page: 1 }),
          pipelineRunApi.list(activeProjectId, { per_page: 5 }),
          pipelineRunApi.list(activeProjectId, {
            per_page: 1,
            date_from: todayStart.toISOString(),
            date_to: todayEnd.toISOString(),
          }),
          applicationApi.list(activeProjectId, { per_page: 1 }),
          deploymentApi.list(activeProjectId, {
            per_page: 1,
            date_from: todayStart.toISOString(),
            date_to: todayEnd.toISOString(),
          }),
          deploymentApi.list(activeProjectId, { per_page: 5 }),
        ]);

        pipelineStats.projectCount = repositoryRes.total;
        recentRuns.value = pipelineRunsRes.items;
        pipelineStats.todayRuns = pipelineTodayRunsRes.total;
        deploymentStats.applicationCount = applicationRes.total;
        deploymentStats.todayDeploys = deploymentTodayRes.total;
        recentDeploys.value = deploymentRecentRes.items;
      });
    } catch {
      toast.error('获取数据失败');
    }
  }

  onMounted(async () => {
    try {
      await authStore.fetchUser();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '获取用户信息失败，请重试');
    }
    refresh();
  });
</script>
