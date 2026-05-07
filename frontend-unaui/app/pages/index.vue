<script setup lang="ts">
import type { PipelineRun } from '~/types/ci/run'
import type { Deployment } from '~/types/cd/deployment'
import { formatTime, getTodayStart, getTodayEnd } from '~/utils/time'
import { statusLabel, statusBadgeVariant } from '~/utils/status'
import {
  useRepositoryApi,
  usePipelineRunApi,
  useApplicationApi,
  useDeploymentApi
} from '~/composables/api'

definePageMeta({
  title: '首页'
})

const repositoryApi = useRepositoryApi()
const pipelineRunApi = usePipelineRunApi()
const applicationApi = useApplicationApi()
const deploymentApi = useDeploymentApi()

const loading = ref(false)
const ciStats = ref({ projectCount: 0, todayRuns: 0 })
const cdStats = ref({ applicationCount: 0, todayDeploys: 0 })
const recentRuns = ref<PipelineRun[]>([])
const recentDeploys = ref<Deployment[]>([])

async function refresh() {
  loading.value = true
  try {
    const todayStart = getTodayStart()
    const todayEnd = getTodayEnd()

    // Fetch CI data
    const [ciProjectsRes, ciRunsRes] = await Promise.all([
      repositoryApi.list({ per_page: 1 }),
      pipelineRunApi.list({ per_page: 5 })
    ])

    // Fetch CD data
    const [cdAppsRes, cdTodayRes, cdRecentRes] = await Promise.all([
      applicationApi.list({ per_page: 1 }),
      deploymentApi.list({
        per_page: 100,
        date_from: todayStart.toISOString(),
        date_to: todayEnd.toISOString()
      }),
      deploymentApi.list({ per_page: 5 })
    ])

    // Update CI stats
    ciStats.value.projectCount = ciProjectsRes.total
    ciStats.value.todayRuns = ciRunsRes.items.filter((r: PipelineRun) => {
      const createdAt = new Date(r.created_at)
      return createdAt >= todayStart && createdAt <= todayEnd
    }).length
    recentRuns.value = ciRunsRes.items

    // Update CD stats
    cdStats.value.applicationCount = cdAppsRes.total
    cdStats.value.todayDeploys = cdTodayRes.total
    recentDeploys.value = cdRecentRes.items
  } catch (error) {
    console.error('获取数据失败:', error)
  } finally {
    loading.value = false
  }
}

// 初始加载
onMounted(() => {
  refresh()
})
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900">欢迎使用 Pomelo Orbit</h1>
        <p class="text-sm text-gray-600 mt-1">持续集成与持续部署平台</p>
      </div>
      <NButton
        label="刷新"
        btn="solid-gray"
        :loading="loading"
        leading="i-lucide-refresh-cw"
        @click="refresh"
      />
    </div>

    <!-- Stats Cards -->
    <div class="grid grid-cols-4 gap-4">
      <NuxtLink to="/ci/repository">
        <NCard class="hover:shadow-md transition-shadow cursor-pointer">
          <div class="space-y-3">
            <p class="text-sm text-gray-600">CI 项目</p>
            <p class="text-3xl font-bold text-gray-900">
              {{ ciStats.projectCount }}
            </p>
          </div>
        </NCard>
      </NuxtLink>

      <NuxtLink to="/ci/builds">
        <NCard class="hover:shadow-md transition-shadow cursor-pointer">
          <div class="space-y-3">
            <p class="text-sm text-gray-600">今日构建</p>
            <p class="text-3xl font-bold text-gray-900">
              {{ ciStats.todayRuns }}
            </p>
          </div>
        </NCard>
      </NuxtLink>

      <NuxtLink to="/cd/apps">
        <NCard class="hover:shadow-md transition-shadow cursor-pointer">
          <div class="space-y-3">
            <p class="text-sm text-gray-600">CD 应用</p>
            <p class="text-3xl font-bold text-gray-900">
              {{ cdStats.applicationCount }}
            </p>
          </div>
        </NCard>
      </NuxtLink>

      <NuxtLink to="/cd/deployments">
        <NCard class="hover:shadow-md transition-shadow cursor-pointer">
          <div class="space-y-3">
            <p class="text-sm text-gray-600">今日部署</p>
            <p class="text-3xl font-bold text-gray-900">
              {{ cdStats.todayDeploys }}
            </p>
          </div>
        </NCard>
      </NuxtLink>
    </div>

    <!-- Recent Activity -->
    <div class="grid grid-cols-2 gap-6">
      <!-- CI Builds -->
      <NCard title="最近构建">
        <div v-if="loading" class="flex justify-center py-12">
          <div class="i-lucide-loader-2 w-8 h-8 animate-spin text-primary" />
        </div>
        <div
          v-else-if="recentRuns.length === 0"
          class="flex flex-col items-center gap-2 py-12 text-gray-600"
        >
          <div class="i-lucide-inbox w-10 h-10" />
          <span class="text-sm">暂无构建记录</span>
        </div>
        <div v-else class="space-y-3">
          <div
            v-for="run in recentRuns"
            :key="run.id"
            class="flex items-center justify-between py-2"
          >
            <div class="space-y-1 flex-1">
              <NuxtLink
                :to="`/ci/repository/${run.repository_id}`"
                class="font-medium text-gray-900 hover:text-primary"
              >
                {{ run.repository_name || run.repository_id }}
              </NuxtLink>
              <p class="text-sm text-gray-600">
                {{ formatTime(run.created_at) }}
              </p>
            </div>
            <NBadge :label="statusLabel(run.status)" :badge="statusBadgeVariant(run.status)" />
          </div>
        </div>
      </NCard>

      <!-- CD Deployments -->
      <NCard title="最近部署">
        <div v-if="loading" class="flex justify-center py-12">
          <div class="i-lucide-loader-2 w-8 h-8 animate-spin text-primary" />
        </div>
        <div
          v-else-if="recentDeploys.length === 0"
          class="flex flex-col items-center gap-2 py-12 text-gray-600"
        >
          <div class="i-lucide-inbox w-10 h-10" />
          <span class="text-sm">暂无部署记录</span>
        </div>
        <div v-else class="space-y-3">
          <div
            v-for="deployment in recentDeploys"
            :key="deployment.id"
            class="flex items-center justify-between py-2"
          >
            <div class="space-y-1 flex-1">
              <p class="font-medium text-gray-900">
                {{ deployment.application_name || deployment.application_id }}
              </p>
              <p class="text-sm text-gray-600">
                {{ formatTime(deployment.started_at) }}
              </p>
            </div>
            <NBadge
              :label="statusLabel(deployment.status)"
              :badge="statusBadgeVariant(deployment.status)"
            />
          </div>
        </div>
      </NCard>
    </div>
  </div>
</template>
