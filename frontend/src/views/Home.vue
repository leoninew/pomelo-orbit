<template>
  <div class="space-y-6">
    <div class="grid grid-cols-1 gap-6 sm:grid-cols-2 xl:grid-cols-4">
      <button
        v-for="(card, index) in overviewCards"
        :key="card.label"
        :aria-label="`查看${card.label}详情，当前${card.value}个`"
        class="group app-surface relative flex min-h-36 items-center gap-4 overflow-hidden px-4 py-4 text-left shadow-sm transition-all duration-300 hover:scale-[1.02] hover:border-primary/30 hover:shadow-lg"
        @click="router.push(card.path)"
      >
        <!-- 图标 -->
        <div class="flex shrink-0 items-center">
          <div
            class="flex size-11 shrink-0 items-center justify-center rounded-xl transition-transform duration-300 group-hover:scale-110"
            :class="card.iconBgClass"
          >
            <component :is="card.icon" class="size-5" :class="card.iconColorClass" />
          </div>
        </div>

        <!-- 指标文字 -->
        <div class="flex min-w-0 flex-1 flex-col justify-center gap-2">
          <p class="text-base font-medium text-foreground">{{ card.label }}</p>
          <span class="text-4xl font-bold leading-none tracking-tight text-foreground">
            {{ card.value }}
          </span>
          <span class="text-sm text-muted-foreground">{{ card.description }}</span>
        </div>

        <!-- 插图 -->
        <div
          class="flex shrink-0 items-center opacity-90 transition-opacity duration-300 group-hover:opacity-100"
        >
          <img
            :src="cardImages[index]"
            :alt="card.label"
            class="h-24 w-24 rounded-xl bg-white/80 object-contain p-1 ring-1 ring-black/5 transition-colors dark:bg-white/90 dark:ring-white/10 sm:h-28 sm:w-28 xl:h-24 xl:w-24 2xl:h-28 2xl:w-28"
          />
        </div>

        <!-- Hover 光晕效果 -->
        <div
          class="pointer-events-none absolute inset-0 rounded-[20px] opacity-0 transition-opacity duration-300 group-hover:opacity-100"
          :style="{ background: getCardGlow(card.iconColorClass) }"
        ></div>
      </button>
    </div>

    <div class="grid grid-cols-1 gap-6 xl:grid-cols-2">
      <section class="app-surface overflow-hidden">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="text-sm font-semibold text-foreground">{{ t('home.recentBuilds') }}</h2>
          <button
            class="app-link inline-flex items-center gap-1 text-sm"
            @click="router.push('/ci/run')"
          >
            {{ t('common.viewAll') }}
            <ArrowRight class="size-4" />
          </button>
        </div>
        <AppSpinner v-if="status === 'loading'" class="py-16" />
        <AppEmptyState v-else-if="recentRuns.length === 0" />
        <div v-else class="overflow-x-auto">
          <table class="app-table-list table-fixed min-w-[560px]">
            <colgroup>
              <col class="w-[42%]" />
              <col class="w-[34%]" />
              <col class="w-[24%]" />
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
                  <button
                    class="app-link block truncate"
                    :title="run.repository_name || run.repository_id"
                    @click="router.push(`/ci/repository/${run.repository_id}`)"
                  >
                    {{ run.repository_name || run.repository_id }}
                  </button>
                </td>
                <td
                  class="overflow-hidden truncate text-foreground"
                  :title="formatTime(run.created_at)"
                >
                  {{ formatTime(run.created_at) }}
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

      <section class="app-surface overflow-hidden">
        <div class="app-section-header flex items-center justify-between">
          <h2 class="text-sm font-semibold text-foreground">{{ t('home.recentDeploys') }}</h2>
          <button
            class="app-link inline-flex items-center gap-1 text-sm"
            @click="router.push('/cd/deployments')"
          >
            {{ t('common.viewAll') }}
            <ArrowRight class="size-4" />
          </button>
        </div>
        <AppSpinner v-if="status === 'loading'" class="py-16" />
        <AppEmptyState v-else-if="recentDeploys.length === 0" />
        <div v-else class="overflow-x-auto">
          <table class="app-table-list table-fixed min-w-[560px]">
            <colgroup>
              <col class="w-[42%]" />
              <col class="w-[34%]" />
              <col class="w-[24%]" />
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
                  <button
                    class="app-link block truncate"
                    :title="deployment.application_name || deployment.application_id"
                    @click="router.push(`/cd/applications/${deployment.application_id}`)"
                  >
                    {{ deployment.application_name || deployment.application_id }}
                  </button>
                </td>
                <td
                  class="overflow-hidden truncate text-foreground"
                  :title="formatTime(deployment.started_at)"
                >
                  {{ formatTime(deployment.started_at) }}
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
  import { ArrowRight, FolderGit2, LayoutGrid, Play, Rocket } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import { applicationApi } from '@/api/cd/application';
  import { deploymentApi } from '@/api/cd/deployments';
  import { pipelineRunApi, repositoryApi } from '@/api/ci';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useAuthStore } from '@/stores/auth';
  import { useProjectStore } from '@/stores/project';
  import type { Deployment } from '@/types/cd/deployment';
  import type { PipelineRun } from '@/types/ci/run';
  import { statusTone } from '@/utils/status';
  import { formatTime, getTodayStart } from '@/utils/time';

  // 导入卡片图片
  import image1 from '@/assets/images/1.png';
  import image2 from '@/assets/images/2.png';
  import image3 from '@/assets/images/3.png';
  import image4 from '@/assets/images/4.png';

  const router = useRouter();
  const toast = useToast();
  const authStore = useAuthStore();
  const projectStore = useProjectStore();
  const { status, execute } = useStatusAsync();
  const { t } = useI18n({ useScope: 'global' });

  const ciStats = reactive({ projectCount: 0, todayRuns: 0 });
  const cdStats = reactive({ applicationCount: 0, todayDeploys: 0 });
  const recentRuns = ref<PipelineRun[]>([]);
  const recentDeploys = ref<Deployment[]>([]);

  const cardImages = [image1, image2, image3, image4];

  function getCardGlow(iconColorClass: string): string {
    const colorMap: Record<string, string> = {
      blue: 'rgba(59, 130, 246, 0.05)',
      purple: 'rgba(168, 85, 247, 0.05)',
      indigo: 'rgba(99, 102, 241, 0.05)',
    };
    const color = Object.keys(colorMap).find((key) => iconColorClass.includes(key));
    const glowColor = color ? colorMap[color] : 'rgba(249, 115, 22, 0.05)';
    return `radial-gradient(circle at 50% 50%, ${glowColor} 0%, transparent 70%)`;
  }

  const overviewCards = computed(() => [
    {
      label: t('home.repositories'),
      value: ciStats.projectCount,
      description: t('home.repositoriesDesc'),
      path: '/ci/repository',
      icon: FolderGit2,
      iconBgClass: 'bg-blue-500/10 dark:bg-blue-400/10',
      iconColorClass: 'text-blue-600 dark:text-blue-400',
    },
    {
      label: t('home.todayBuilds'),
      value: ciStats.todayRuns,
      description: t('home.todayBuildsDesc'),
      path: '/ci/run',
      icon: Play,
      iconBgClass: 'bg-purple-500/10 dark:bg-purple-400/10',
      iconColorClass: 'text-purple-600 dark:text-purple-400',
    },
    {
      label: t('home.applications'),
      value: cdStats.applicationCount,
      description: t('home.applicationsDesc'),
      path: '/cd/applications',
      icon: LayoutGrid,
      iconBgClass: 'bg-indigo-500/10 dark:bg-indigo-400/10',
      iconColorClass: 'text-indigo-600 dark:text-indigo-400',
    },
    {
      label: t('home.todayDeploys'),
      value: cdStats.todayDeploys,
      description: t('home.todayDeploysDesc'),
      path: '/cd/deployments',
      icon: Rocket,
      iconBgClass: 'bg-orange-500/10 dark:bg-orange-400/10',
      iconColorClass: 'text-orange-600 dark:text-orange-400',
    },
  ]);

  function resetOverview() {
    ciStats.projectCount = 0;
    ciStats.todayRuns = 0;
    cdStats.applicationCount = 0;
    cdStats.todayDeploys = 0;
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
        const [ciProjectsRes, ciRunsRes, ciTodayRunsRes, cdAppsRes, cdTodayRes, cdRecentRes] =
          await Promise.all([
            repositoryApi.list({ per_page: 1, project_id: activeProjectId }),
            pipelineRunApi.list({ per_page: 5, project_id: activeProjectId }),
            pipelineRunApi.list({
              per_page: 1,
              date_from: todayStart.toISOString(),
              date_to: todayEnd.toISOString(),
              project_id: activeProjectId,
            }),
            applicationApi.list({ per_page: 1, project_id: activeProjectId }),
            deploymentApi.list({
              per_page: 1,
              date_from: todayStart.toISOString(),
              date_to: todayEnd.toISOString(),
              project_id: activeProjectId,
            }),
            deploymentApi.list({ per_page: 5, project_id: activeProjectId }),
          ]);

        ciStats.projectCount = ciProjectsRes.total;
        recentRuns.value = ciRunsRes.items;
        ciStats.todayRuns = ciTodayRunsRes.total;
        cdStats.applicationCount = cdAppsRes.total;
        cdStats.todayDeploys = cdTodayRes.total;
        recentDeploys.value = cdRecentRes.items;
      });
    } catch {
      toast.error('获取数据失败');
    }
  }

  onMounted(async () => {
    await authStore.fetchUser();
    refresh();
  });
</script>
