<template>
	<div class="flex flex-col gap-6">
		<!-- Page header -->
		<div class="flex items-center justify-between">
			<div>
				<h1 class="text-2xl font-semibold">欢迎使用 Pomelo Orbit</h1>
				<p class="text-sm text-base-content/60 mt-1">持续集成与持续部署平台</p>
			</div>
			<button
				class="btn btn-sm btn-ghost gap-1.5"
				:disabled="status === 'loading'"
				@click="refresh"
			>
				<RefreshCw class="size-4" :class="{ 'animate-spin': status === 'loading' }" />
				刷新
			</button>
		</div>

		<!-- Quick actions -->
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
			<router-link
				to="/ci/repository"
				class="card bg-gradient-to-br from-blue-500/10 to-blue-600/5 hover:shadow-md transition-shadow"
			>
				<div class="card-body p-4">
					<div class="flex items-center gap-3">
						<div class="p-2 bg-blue-500/20 rounded-lg">
							<FolderGit2 class="size-5 text-blue-600" />
						</div>
						<div>
							<div class="text-sm text-base-content/60">CI 项目</div>
							<div class="text-2xl font-semibold">{{ ciStats.projectCount }}</div>
						</div>
					</div>
				</div>
			</router-link>

			<router-link
				to="/ci/run"
				class="card bg-gradient-to-br from-green-500/10 to-green-600/5 hover:shadow-md transition-shadow"
			>
				<div class="card-body p-4">
					<div class="flex items-center gap-3">
						<div class="p-2 bg-green-500/20 rounded-lg">
							<Play class="size-5 text-green-600" />
						</div>
						<div>
							<div class="text-sm text-base-content/60">今日构建</div>
							<div class="text-2xl font-semibold">{{ ciStats.todayRuns }}</div>
						</div>
					</div>
				</div>
			</router-link>

			<router-link
				to="/cd/applications"
				class="card bg-gradient-to-br from-purple-500/10 to-purple-600/5 hover:shadow-md transition-shadow"
			>
				<div class="card-body p-4">
					<div class="flex items-center gap-3">
						<div class="p-2 bg-purple-500/20 rounded-lg">
							<LayoutGrid class="size-5 text-purple-600" />
						</div>
						<div>
							<div class="text-sm text-base-content/60">CD 应用</div>
							<div class="text-2xl font-semibold">{{ cdStats.applicationCount }}</div>
						</div>
					</div>
				</div>
			</router-link>

			<router-link
				to="/cd/deployments"
				class="card bg-gradient-to-br from-orange-500/10 to-orange-600/5 hover:shadow-md transition-shadow"
			>
				<div class="card-body p-4">
					<div class="flex items-center gap-3">
						<div class="p-2 bg-orange-500/20 rounded-lg">
							<Rocket class="size-5 text-orange-600" />
						</div>
						<div>
							<div class="text-sm text-base-content/60">今日部署</div>
							<div class="text-2xl font-semibold">{{ cdStats.todayDeploys }}</div>
						</div>
					</div>
				</div>
			</router-link>
		</div>

		<!-- Two columns layout -->
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
			<!-- CI Recent runs -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-0">
					<div class="flex items-center justify-between px-5 pt-4 pb-3 border-b border-base-200">
						<h2 class="font-semibold flex items-center gap-2">
							<GitBranch class="size-4" />
							最近构建
						</h2>
						<router-link to="/ci/run" class="link link-primary text-sm">查看全部</router-link>
					</div>
					<div v-if="status === 'loading'" class="flex justify-center py-12">
						<span class="loading loading-spinner loading-md text-primary" />
					</div>
					<div
						v-else-if="recentRuns.length === 0"
						class="flex flex-col items-center gap-2 py-12 text-base-content/60"
					>
						<Inbox class="size-10" />
						<span class="text-sm">暂无构建记录</span>
					</div>
					<div v-else class="overflow-x-auto">
						<table class="table table-sm">
							<tbody>
								<tr v-for="run in recentRuns" :key="run.id" class="hover">
									<td class="font-medium">{{ run.repository_id }}</td>
									<td class="w-24">
										<span class="badge badge-sm" :class="statusBadgeClass(run.status)">
											{{ statusLabel(run.status) }}
										</span>
									</td>
									<td class="text-base-content/60">{{ formatTime(run.created_at) }}</td>
									<td class="w-16 text-right">
										<router-link :to="`/ci/run/${run.id}`" class="link link-primary">
											详情
										</router-link>
									</td>
								</tr>
							</tbody>
						</table>
					</div>
				</div>
			</div>

			<!-- CD Recent deployments -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-0">
					<div class="flex items-center justify-between px-5 pt-4 pb-3 border-b border-base-200">
						<h2 class="font-semibold flex items-center gap-2">
							<Rocket class="size-4" />
							最近部署
						</h2>
						<router-link to="/cd/deployments" class="link link-primary text-sm">
							查看全部
						</router-link>
					</div>
					<div v-if="status === 'loading'" class="flex justify-center py-12">
						<span class="loading loading-spinner loading-md text-primary" />
					</div>
					<div
						v-else-if="recentDeploys.length === 0"
						class="flex flex-col items-center gap-2 py-12 text-base-content/60"
					>
						<Inbox class="size-10" />
						<span class="text-sm">暂无部署记录</span>
					</div>
					<div v-else class="overflow-x-auto">
						<table class="table table-sm">
							<tbody>
								<tr v-for="d in recentDeploys" :key="d.id" class="hover">
									<td class="font-medium">
										{{ d.application_name || d.application_id }}
									</td>
									<td class="w-24">
										<span class="badge badge-sm" :class="statusBadgeClass(d.status)">
											{{ statusLabel(d.status) }}
										</span>
									</td>
									<td class="text-base-content/60">{{ formatTime(d.started_at) }}</td>
									<td class="w-16 text-right">
										<router-link :to="`/cd/deployments/${d.id}`" class="link link-primary">
											详情
										</router-link>
									</td>
								</tr>
							</tbody>
						</table>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import { FolderGit2, GitBranch, Inbox, LayoutGrid, Play, RefreshCw, Rocket } from 'lucide-vue-next';
import { onMounted, reactive, ref } from 'vue';
import { applicationApi } from '@/api/cd/application';
import { deploymentApi } from '@/api/cd/deployments';
import { pipelineRunApi, repositoryApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { Deployment, PipelineRun } from '@/types/api';
import { statusBadgeClass, statusLabel } from '@/utils/status';
import { formatTime, getTodayStart } from '@/utils/time';

const toast = useToast();
const { status, execute } = useStatusAsync();

const ciStats = reactive({ projectCount: 0, todayRuns: 0 });
const cdStats = reactive({ applicationCount: 0, todayDeploys: 0 });
const recentRuns = ref<PipelineRun[]>([]);
const recentDeploys = ref<Deployment[]>([]);

async function refresh() {
	try {
		await execute(async () => {
			const todayStart = getTodayStart();
			const todayEnd = todayStart.add(1, 'day');

			// Fetch CI data
			const ciProjectsRes = await repositoryApi.list({ per_page: 1 });
			const ciRunsRes = await pipelineRunApi.list({ per_page: 5 });

			// Fetch CD data
			const cdAppsRes = await applicationApi.list({ per_page: 1 });
			const cdTodayRes = await deploymentApi.list({
				per_page: 100,
				date_from: todayStart.toISOString(),
				date_to: todayEnd.toISOString(),
			});
			const cdRecentRes = await deploymentApi.list({ per_page: 5 });

			// Update CI stats
			ciStats.projectCount = ciProjectsRes.total;
			ciStats.todayRuns = ciRunsRes.items.filter((r) => {
				const createdAt = new Date(r.created_at);
				return createdAt >= todayStart.toDate() && createdAt < todayEnd.toDate();
			}).length;
			recentRuns.value = ciRunsRes.items;

			// Update CD stats
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
