<script setup lang="ts">
import { ArrowRight, FolderGit2, LayoutGrid, Play, RefreshCw, Rocket } from 'lucide-vue-next'
import { computed, onMounted, reactive, ref } from 'vue'
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
import { ToolbarRoot } from 'reka-ui'

const router = useRouter()
const toast = useToast()
const { status, execute } = useStatusAsync()

const ciStats = reactive({ projectCount: 0, todayRuns: 0 })
const cdStats = reactive({ applicationCount: 0, todayDeploys: 0 })
const recentRuns = ref<PipelineRun[]>([])
const recentDeploys = ref<Deployment[]>([])

const overviewCards = computed(() => [
	{
		label: '代码仓库',
		value: ciStats.projectCount,
		description: '已接入 CI 项目',
		path: '/ci/repository',
		icon: FolderGit2
	},
	{
		label: '今日构建',
		value: ciStats.todayRuns,
		description: '今日触发流水线',
		path: '/ci/run',
		icon: Play
	},
	{
		label: '应用管理',
		value: cdStats.applicationCount,
		description: '已接入 CD 应用',
		path: '/cd/applications',
		icon: LayoutGrid
	},
	{
		label: '今日部署',
		value: cdStats.todayDeploys,
		description: '今日部署任务',
		path: '/cd/deployments',
		icon: Rocket
	}
])

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
		<ToolbarRoot class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between" aria-label="概览工具栏">
			<div class="space-y-1">
				<h1 class="text-xl font-semibold text-foreground">项目概览</h1>
				<p class="text-sm text-muted-foreground">CI/CD 今日运行与最近活动</p>
			</div>
			<button
				class="inline-flex h-10 items-center gap-2 rounded-md border border-input bg-background px-4 text-sm font-medium text-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
				:disabled="status === 'loading'"
				@click="refresh"
			>
				<RefreshCw class="size-4" :class="{ 'animate-spin': status === 'loading' }" />
				刷新
			</button>
		</ToolbarRoot>

		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
			<button
				v-for="card in overviewCards"
				:key="card.label"
				class="rounded-lg border border-border bg-card p-4 text-left shadow-sm transition-colors hover:bg-muted/30"
				@click="router.push(card.path)"
			>
				<div class="flex items-start justify-between gap-4">
					<div class="min-w-0">
						<p class="text-sm text-muted-foreground">{{ card.label }}</p>
						<p class="mt-3 text-2xl font-semibold text-foreground">{{ card.value }}</p>
						<p class="mt-1 text-xs text-muted-foreground">{{ card.description }}</p>
					</div>
					<span class="flex size-9 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
						<component :is="card.icon" class="size-4" />
					</span>
				</div>
			</button>
		</div>

		<div class="grid grid-cols-1 gap-6 xl:grid-cols-2">
			<section class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
				<div class="flex items-center justify-between border-b border-border bg-muted/30 px-5 py-4">
					<h2 class="text-sm font-semibold text-foreground">最近构建</h2>
					<button
						class="inline-flex items-center gap-1 text-sm text-primary hover:underline"
						@click="router.push('/ci/run')"
					>
						查看全部
						<ArrowRight class="size-4" />
					</button>
				</div>
				<div v-if="status === 'loading'" class="flex justify-center py-16">
					<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
				</div>
				<div v-else-if="recentRuns.length === 0" class="text-center py-16 text-muted-foreground">
					<p class="text-sm">暂无构建记录</p>
				</div>
				<div v-else>
					<div class="hidden grid-cols-[minmax(0,1fr)_140px_80px] border-b border-border px-5 py-3 text-xs text-muted-foreground sm:grid">
						<span>仓库</span>
						<span>创建时间</span>
						<span class="text-right">状态</span>
					</div>
					<div class="divide-y divide-border">
						<button
							v-for="run in recentRuns"
							:key="run.id"
							class="grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-4 px-5 py-4 text-left transition-colors hover:bg-muted/30 sm:grid-cols-[minmax(0,1fr)_140px_80px]"
							@click="router.push(`/ci/run/${run.id}`)"
						>
							<span class="min-w-0">
								<span class="block truncate text-sm font-medium text-foreground">
									{{ run.repository_name || run.repository_id }}
								</span>
								<span class="mt-1 block text-xs text-muted-foreground sm:hidden">
									{{ formatTime(run.created_at) }}
								</span>
							</span>
							<span class="hidden text-sm text-muted-foreground sm:block">{{ formatTime(run.created_at) }}</span>
							<span class="justify-self-end rounded px-2 py-1 text-xs" :class="statusBadgeClass(run.status)">
								{{ statusLabel(run.status) }}
							</span>
						</button>
					</div>
				</div>
			</section>

			<section class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
				<div class="flex items-center justify-between border-b border-border bg-muted/30 px-5 py-4">
					<h2 class="text-sm font-semibold text-foreground">最近部署</h2>
					<button
						class="inline-flex items-center gap-1 text-sm text-primary hover:underline"
						@click="router.push('/cd/deployments')"
					>
						查看全部
						<ArrowRight class="size-4" />
					</button>
				</div>
				<div v-if="status === 'loading'" class="flex justify-center py-16">
					<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
				</div>
				<div v-else-if="recentDeploys.length === 0" class="text-center py-16 text-muted-foreground">
					<p class="text-sm">暂无部署记录</p>
				</div>
				<div v-else>
					<div class="hidden grid-cols-[minmax(0,1fr)_140px_80px] border-b border-border px-5 py-3 text-xs text-muted-foreground sm:grid">
						<span>应用</span>
						<span>开始时间</span>
						<span class="text-right">状态</span>
					</div>
					<div class="divide-y divide-border">
						<button
							v-for="deployment in recentDeploys"
							:key="deployment.id"
							class="grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-4 px-5 py-4 text-left transition-colors hover:bg-muted/30 sm:grid-cols-[minmax(0,1fr)_140px_80px]"
							@click="router.push(`/cd/deployments/${deployment.id}`)"
						>
							<span class="min-w-0">
								<span class="block truncate text-sm font-medium text-foreground">
									{{ deployment.application_name || deployment.application_id }}
								</span>
								<span class="mt-1 block text-xs text-muted-foreground sm:hidden">
									{{ formatTime(deployment.started_at) }}
								</span>
							</span>
							<span class="hidden text-sm text-muted-foreground sm:block">{{ formatTime(deployment.started_at) }}</span>
							<span class="justify-self-end rounded px-2 py-1 text-xs" :class="statusBadgeClass(deployment.status)">
								{{ statusLabel(deployment.status) }}
							</span>
						</button>
					</div>
				</div>
			</section>
		</div>
	</div>
</template>
