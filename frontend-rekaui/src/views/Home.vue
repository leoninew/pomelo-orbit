<script setup lang="ts">
import { RefreshCw } from 'lucide-vue-next'
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { applicationApi } from '@/api/cd/application'
import { deploymentApi } from '@/api/cd/deployments'
import { pipelineRunApi, repositoryApi } from '@/api/ci'
import { useStatusAsync } from '@/composables/useStatusAsync'
import { useToast } from '@/composables/useToast'
import type { Deployment } from '@/types/cd/deployment'
import type { PipelineRun } from '@/types/ci/run'
import { statusBadgeClass, statusLabel } from '@/utils/status'
import { formatTime, getTodayStart } from '@/utils/time'

const router = useRouter()
const toast = useToast()
const { status, execute } = useStatusAsync()

const ciStats = reactive({ projectCount: 0, todayRuns: 0 })
const cdStats = reactive({ applicationCount: 0, todayDeploys: 0 })
const recentRuns = ref<PipelineRun[]>([])
const recentDeploys = ref<Deployment[]>([])

async function refresh() {
	try {
		await execute(async () => {
			const todayStart = getTodayStart()
			const todayEnd = todayStart.add(1, 'day')

			const ciProjectsRes = await repositoryApi.list({ per_page: 1 })
			const ciRunsRes = await pipelineRunApi.list({ per_page: 5 })

			const cdAppsRes = await applicationApi.list({ per_page: 1 })
			const cdTodayRes = await deploymentApi.list({
				per_page: 100,
				date_from: todayStart.toISOString(),
				date_to: todayEnd.toISOString()
			})
			const cdRecentRes = await deploymentApi.list({ per_page: 5 })

			ciStats.projectCount = ciProjectsRes.total
			ciStats.todayRuns = ciRunsRes.items.filter((r) => {
				const createdAt = new Date(r.created_at)
				return createdAt >= todayStart.toDate() && createdAt < todayEnd.toDate()
			}).length
			recentRuns.value = ciRunsRes.items

			cdStats.applicationCount = cdAppsRes.total
			cdStats.todayDeploys = cdTodayRes.total
			recentDeploys.value = cdRecentRes.items
		})
	} catch {
		toast.error('获取数据失败')
	}
}

onMounted(refresh)
</script>

<template>
	<div class="space-y-6">
		<!-- Header -->
		<div class="flex items-center justify-between">
			<div>
				<h1 class="text-2xl font-semibold text-foreground">欢迎使用 Pomelo Orbit</h1>
				<p class="text-sm text-muted-foreground mt-1">持续集成与持续部署平台</p>
			</div>
			<button
				class="px-4 py-2 bg-background border border-input rounded-lg hover:bg-muted/50 flex items-center gap-2 disabled:opacity-50 transition-colors"
				:disabled="status === 'loading'"
				@click="refresh"
			>
				<RefreshCw class="w-4 h-4" :class="{ 'animate-spin': status === 'loading' }" />
				刷新
			</button>
		</div>

		<!-- Stats Cards -->
		<div class="grid grid-cols-4 gap-4">
			<button
				class="bg-card p-6 rounded-lg border border-border hover:shadow-md transition-shadow text-left"
				@click="router.push('/ci/repository')"
			>
				<p class="text-sm text-muted-foreground mb-2">CI 项目</p>
				<p class="text-3xl font-bold text-foreground">{{ ciStats.projectCount }}</p>
			</button>

			<button
				class="bg-card p-6 rounded-lg border border-border hover:shadow-md transition-shadow text-left"
				@click="router.push('/ci/run')"
			>
				<p class="text-sm text-muted-foreground mb-2">今日构建</p>
				<p class="text-3xl font-bold text-foreground">{{ ciStats.todayRuns }}</p>
			</button>

			<button
				class="bg-card p-6 rounded-lg border border-border hover:shadow-md transition-shadow text-left"
				@click="router.push('/cd/applications')"
			>
				<p class="text-sm text-muted-foreground mb-2">CD 应用</p>
				<p class="text-3xl font-bold text-foreground">{{ cdStats.applicationCount }}</p>
			</button>

			<button
				class="bg-card p-6 rounded-lg border border-border hover:shadow-md transition-shadow text-left"
				@click="router.push('/cd/deployments')"
			>
				<p class="text-sm text-muted-foreground mb-2">今日部署</p>
				<p class="text-3xl font-bold text-foreground">{{ cdStats.todayDeploys }}</p>
			</button>
		</div>

		<!-- Recent Activity -->
		<div class="grid grid-cols-2 gap-6">
			<!-- CI Builds -->
			<div class="bg-card rounded-lg border border-border">
				<div class="p-4 border-b border-border">
					<h2 class="font-semibold text-foreground">最近构建</h2>
				</div>
				<div class="p-4">
					<div v-if="status === 'loading'" class="flex justify-center py-12">
						<div class="w-8 h-8 border-4 border-primary/20 border-t-primary rounded-full animate-spin" />
					</div>
					<div v-else-if="recentRuns.length === 0" class="text-center py-12 text-muted-foreground">
						<p class="text-sm">暂无构建记录</p>
					</div>
					<div v-else class="space-y-3">
						<div
							v-for="run in recentRuns"
							:key="run.id"
							class="flex items-center justify-between py-2 cursor-pointer hover:bg-muted/50 px-2 rounded transition-colors"
							@click="router.push(`/ci/run/${run.id}`)"
						>
							<div class="flex-1 min-w-0">
								<p class="font-medium text-foreground truncate">
									{{ run.repository_name || run.repository_id }}
								</p>
								<p class="text-sm text-muted-foreground">{{ formatTime(run.created_at) }}</p>
							</div>
							<span
								class="px-2 py-1 text-xs rounded"
								:class="statusBadgeClass(run.status)"
							>
								{{ statusLabel(run.status) }}
							</span>
						</div>
					</div>
				</div>
			</div>

			<!-- CD Deployments -->
			<div class="bg-card rounded-lg border border-border">
				<div class="p-4 border-b border-border">
					<h2 class="font-semibold text-foreground">最近部署</h2>
				</div>
				<div class="p-4">
					<div v-if="status === 'loading'" class="flex justify-center py-12">
						<div class="w-8 h-8 border-4 border-primary/20 border-t-primary rounded-full animate-spin" />
					</div>
					<div v-else-if="recentDeploys.length === 0" class="text-center py-12 text-muted-foreground">
						<p class="text-sm">暂无部署记录</p>
					</div>
					<div v-else class="space-y-3">
						<div
							v-for="deployment in recentDeploys"
							:key="deployment.id"
							class="flex items-center justify-between py-2 cursor-pointer hover:bg-muted/50 px-2 rounded transition-colors"
							@click="router.push(`/cd/deployments/${deployment.id}`)"
						>
							<div class="flex-1 min-w-0">
								<p class="font-medium text-foreground truncate">
									{{ deployment.application_name || deployment.application_id }}
								</p>
								<p class="text-sm text-muted-foreground">{{ formatTime(deployment.started_at) }}</p>
							</div>
							<span
								class="px-2 py-1 text-xs rounded"
								:class="statusBadgeClass(deployment.status)"
							>
								{{ statusLabel(deployment.status) }}
							</span>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>
