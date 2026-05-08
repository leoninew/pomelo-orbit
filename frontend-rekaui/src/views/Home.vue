<template>
	<div class="space-y-6">
		<ToolbarRoot
			class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
			aria-label="概览工具栏"
		>
			<div class="space-y-1">
				<h1 class="text-xl font-semibold text-foreground">项目概览</h1>
				<p class="text-sm text-muted-foreground">CI/CD 今日运行与最近活动</p>
			</div>
			<button
				class="app-button inline-flex h-10 items-center gap-2 px-4"
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
				class="app-surface rounded-lg p-4 text-left transition-colors hover:bg-muted/30"
				@click="router.push(card.path)"
			>
				<div class="flex items-start justify-between gap-4">
					<div class="min-w-0">
						<p class="text-sm text-muted-foreground">{{ card.label }}</p>
						<p class="mt-3 text-2xl font-semibold text-foreground">{{ card.value }}</p>
						<p class="mt-1 text-xs text-muted-foreground">{{ card.description }}</p>
					</div>
					<span
						class="flex size-9 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary"
					>
						<component :is="card.icon" class="size-4" />
					</span>
				</div>
			</button>
		</div>

		<div class="grid grid-cols-1 gap-6 xl:grid-cols-2">
			<section class="app-surface overflow-hidden rounded-lg">
				<div class="app-section-header flex items-center justify-between">
					<h2 class="text-sm font-semibold text-foreground">最近构建</h2>
					<button
						class="app-link inline-flex items-center gap-1 text-sm"
						@click="router.push('/ci/run')"
					>
						查看全部
						<ArrowRight class="size-4" />
					</button>
				</div>
				<div v-if="status === 'loading'" class="flex justify-center py-16">
					<div
						class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary"
					/>
				</div>
				<div v-else-if="recentRuns.length === 0" class="text-center py-16 text-muted-foreground">
					<p class="text-sm">暂无构建记录</p>
				</div>
				<div v-else class="overflow-x-auto">
					<table class="app-table-list min-w-[520px]">
						<thead>
							<tr>
								<th>仓库</th>
								<th>创建时间</th>
								<th class="text-right">状态</th>
							</tr>
						</thead>
						<tbody>
							<tr
								v-for="run in recentRuns"
								:key="run.id"
								class="cursor-pointer"
								@click="router.push(`/ci/run/${run.id}`)"
							>
								<td
									class="max-w-56 truncate text-foreground"
									:title="run.repository_name || run.repository_id"
								>
									{{ run.repository_name || run.repository_id }}
								</td>
								<td class="text-muted-foreground">{{ formatTime(run.created_at) }}</td>
								<td class="text-right">
									<span
										class="inline-flex rounded px-2 py-1 text-xs"
										:class="statusBadgeClass(run.status)"
									>
										{{ statusLabel(run.status) }}
									</span>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</section>

			<section class="app-surface overflow-hidden rounded-lg">
				<div class="app-section-header flex items-center justify-between">
					<h2 class="text-sm font-semibold text-foreground">最近部署</h2>
					<button
						class="app-link inline-flex items-center gap-1 text-sm"
						@click="router.push('/cd/deployments')"
					>
						查看全部
						<ArrowRight class="size-4" />
					</button>
				</div>
				<div v-if="status === 'loading'" class="flex justify-center py-16">
					<div
						class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary"
					/>
				</div>
				<div v-else-if="recentDeploys.length === 0" class="text-center py-16 text-muted-foreground">
					<p class="text-sm">暂无部署记录</p>
				</div>
				<div v-else class="overflow-x-auto">
					<table class="app-table-list min-w-[520px]">
						<thead>
							<tr>
								<th>应用</th>
								<th>开始时间</th>
								<th class="text-right">状态</th>
							</tr>
						</thead>
						<tbody>
							<tr
								v-for="deployment in recentDeploys"
								:key="deployment.id"
								class="cursor-pointer"
								@click="router.push(`/cd/deployments/${deployment.id}`)"
							>
								<td
									class="max-w-56 truncate text-foreground"
									:title="deployment.application_name || deployment.application_id"
								>
									{{ deployment.application_name || deployment.application_id }}
								</td>
								<td class="text-muted-foreground">{{ formatTime(deployment.started_at) }}</td>
								<td class="text-right">
									<span
										class="inline-flex rounded px-2 py-1 text-xs"
										:class="statusBadgeClass(deployment.status)"
									>
										{{ statusLabel(deployment.status) }}
									</span>
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
	import { ArrowRight, FolderGit2, LayoutGrid, Play, RefreshCw, Rocket } from 'lucide-vue-next';
	import { computed, onMounted, reactive, ref } from 'vue';
	import { useRouter } from 'vue-router';
	import { applicationApi } from '@/api/cd/application';
	import { deploymentApi } from '@/api/cd/deployments';
	import { pipelineRunApi, repositoryApi } from '@/api/ci';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import type { Deployment } from '@/types/cd/deployment';
	import type { PipelineRun } from '@/types/ci/run';
	import { statusBadgeClass, statusLabel } from '@/utils/status';
	import { formatTime, getTodayStart } from '@/utils/time';
	import { ToolbarRoot } from 'reka-ui';

	const router = useRouter();
	const toast = useToast();
	const { status, execute } = useStatusAsync();

	const ciStats = reactive({ projectCount: 0, todayRuns: 0 });
	const cdStats = reactive({ applicationCount: 0, todayDeploys: 0 });
	const recentRuns = ref<PipelineRun[]>([]);
	const recentDeploys = ref<Deployment[]>([]);

	const overviewCards = computed(() => [
		{
			label: '代码仓库',
			value: ciStats.projectCount,
			description: '已接入 CI 项目',
			path: '/ci/repository',
			icon: FolderGit2,
		},
		{
			label: '今日构建',
			value: ciStats.todayRuns,
			description: '今日触发流水线',
			path: '/ci/run',
			icon: Play,
		},
		{
			label: '应用管理',
			value: cdStats.applicationCount,
			description: '已接入 CD 应用',
			path: '/cd/applications',
			icon: LayoutGrid,
		},
		{
			label: '今日部署',
			value: cdStats.todayDeploys,
			description: '今日部署任务',
			path: '/cd/deployments',
			icon: Rocket,
		},
	]);

	async function refresh() {
		try {
			await execute(async () => {
				const todayStart = getTodayStart();
				const todayEnd = todayStart.add(1, 'day');

				const ciProjectsRes = await repositoryApi.list({ per_page: 1 });
				const ciRunsRes = await pipelineRunApi.list({ per_page: 5 });

				const cdAppsRes = await applicationApi.list({ per_page: 1 });
				const cdTodayRes = await deploymentApi.list({
					per_page: 100,
					date_from: todayStart.toISOString(),
					date_to: todayEnd.toISOString(),
				});
				const cdRecentRes = await deploymentApi.list({ per_page: 5 });

				ciStats.projectCount = ciProjectsRes.total;
				ciStats.todayRuns = ciRunsRes.items.filter((r) => {
					const createdAt = new Date(r.created_at);
					return createdAt >= todayStart.toDate() && createdAt < todayEnd.toDate();
				}).length;
				recentRuns.value = ciRunsRes.items;

				cdStats.applicationCount = cdAppsRes.total;
				cdStats.todayDeploys = cdTodayRes.total;
				recentDeploys.value = cdRecentRes.items;
			});
		} catch {
			toast.error('获取数据失败');
		}
	}

	onMounted(refresh);
</script>
