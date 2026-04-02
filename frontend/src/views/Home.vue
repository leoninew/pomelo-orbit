<template>
	<div class="flex flex-col gap-6">
		<!-- Page header -->
		<div class="flex items-center justify-between">
			<h1 class="text-xl font-semibold">仪表盘</h1>
			<button class="btn btn-sm btn-ghost gap-1.5" :disabled="loading" @click="refresh">
				<RefreshCw class="size-4" :class="{ 'animate-spin': loading }" />
				刷新
			</button>
		</div>

		<!-- Stats -->
		<div class="stats stats-horizontal shadow bg-base-100 w-full">
			<div class="stat">
				<div class="stat-figure text-primary">
					<LayoutGrid class="size-8" />
				</div>
				<div class="stat-title">应用总数</div>
				<div class="stat-value text-primary">{{ loading ? '—' : stats.projectCount }}</div>
			</div>
			<div class="stat">
				<div class="stat-figure text-secondary">
					<Rocket class="size-8" />
				</div>
				<div class="stat-title">今日部署</div>
				<div class="stat-value text-secondary">{{ loading ? '—' : stats.todayDeploys }}</div>
			</div>
			<div class="stat">
				<div class="stat-figure text-accent">
					<Activity class="size-8" />
				</div>
				<div class="stat-title">运行中</div>
				<div class="stat-value text-accent">{{ loading ? '—' : stats.runningDeploys }}</div>
			</div>
		</div>

		<!-- Recent deployments -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-0">
				<div class="flex items-center justify-between px-5 pt-4 pb-3 border-b border-base-200">
					<h2 class="font-semibold">最近部署</h2>
				</div>
				<div v-if="loading" class="flex justify-center py-12">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<div v-else-if="recentDeploys.length === 0" class="flex flex-col items-center gap-2 py-12 text-base-content/40">
					<Inbox class="size-10" />
					<span class="text-sm">暂无部署记录</span>
				</div>
				<div v-else class="overflow-x-auto">
					<table class="table">
						<thead>
							<tr class="text-base-content/60">
								<th>状态</th>
								<th>应用</th>
								<th>触发方式</th>
								<th>时间</th>
								<th></th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="d in recentDeploys" :key="d.id" class="hover">
								<td>
									<span class="badge badge-sm" :class="deploymentBadgeClass(d.status)">
										{{ d.status }}
									</span>
								</td>
								<td class="font-medium">{{ d.application_name || d.application_id }}</td>
								<td class="cell-muted">{{ d.trigger_type }}</td>
								<td class="cell-muted">{{ formatTime(d.started_at) }}</td>
								<td>
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
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { RefreshCw, LayoutGrid, Rocket, Activity, Inbox } from 'lucide-vue-next';
import { applicationApi } from '@/api/application';
import { deploymentApi } from '@/api/deployments';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { formatTime, getTodayStart } from '@/utils/time';
import type { Deployment } from '@/types/api';

const toast = useToast();
const { loading, execute } = useStatusAsync();

const stats = reactive({ projectCount: 0, todayDeploys: 0, runningDeploys: 0 });
const recentDeploys = ref<Deployment[]>([]);

const statusBadgeMap: Record<string, string> = {
	ran_to_completion: 'badge-outline badge-success',
	success: 'badge-outline badge-success',
	faulted: 'badge-outline badge-error',
	failed: 'badge-outline badge-error',
	running: 'badge-outline badge-info',
	queued: 'badge-outline badge-warning',
	canceled: 'badge-ghost',
};

function deploymentBadgeClass(status: string) {
	return statusBadgeMap[status] ?? 'badge-ghost';
}

async function refresh() {
	try {
		await execute(async () => {
			const todayStart = getTodayStart();
			const todayEnd = todayStart.add(1, 'day');

			const [appsRes, todayRes, recentRes] = await Promise.all([
				applicationApi.list({ per_page: 1 }),
				deploymentApi.list({ per_page: 100, date_from: todayStart.toISOString(), date_to: todayEnd.toISOString() }),
				deploymentApi.list({ per_page: 5 }),
			]);

			stats.projectCount = appsRes.total;
			stats.todayDeploys = todayRes.total;
			stats.runningDeploys = todayRes.items.filter((d) => d.status === 'running' || d.status === 'queued').length;
			recentDeploys.value = recentRes.items;
		});
	} catch {
		toast.error('获取数据失败');
	}
}

onMounted(refresh);
</script>
