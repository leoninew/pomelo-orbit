<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">部署记录</h1>
			<label class="input input-sm input-bordered flex items-center gap-2">
				<Search class="size-3.5 text-base-content/40" />
				<input v-model="searchText" type="text" placeholder="搜索应用名称" class="w-36" @keyup.enter="handleSearch" />
			</label>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table table-sm">
				<thead>
					<tr class="text-base-content/60">
						<th>应用</th>
						<th>操作类型</th>
						<th>触发方式</th>
						<th>分支/Tag</th>
						<th>环境文件</th>
						<th>状态</th>
						<th>开始时间</th>
						<th>耗时</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					<tr v-if="loading">
						<td colspan="9" class="text-center py-8">
							<span class="loading loading-spinner loading-md text-primary" />
						</td>
					</tr>
					<tr v-else-if="deployments.length === 0">
						<td colspan="9" class="text-center py-8 text-base-content/40">暂无部署记录</td>
					</tr>
					<tr v-for="d in deployments" :key="d.id" class="hover">
						<td>
							<router-link :to="`/cd/applications/${d.application_id}`" class="link link-primary">
								{{ d.application_name || d.application_id }}
							</router-link>
						</td>
						<td class="cell-muted">{{ d.operation_type }}</td>
						<td class="cell-muted">{{ d.trigger_type }}</td>
						<td><code class="text-xs">{{ d.trigger_ref || '—' }}</code></td>
						<td class="cell-muted">{{ d.env_file || '—' }}</td>
						<td><span class="badge badge-sm" :class="deployBadgeClass(d.status)">{{ d.status }}</span></td>
						<td class="cell-muted">{{ formatTime(d.started_at) }}</td>
						<td class="cell-muted">{{ formatDuration(d.duration_ms) }}</td>
						<td>
							<div class="flex items-center gap-2">
								<router-link :to="`/cd/deployments/${d.id}`" class="link link-primary">查看</router-link>
								<button v-if="d.status === 'running' || d.status === 'queued'" class="link link-error" @click="handleCancel(d.id)">取消</button>
							</div>
						</td>
					</tr>
				</tbody>
			</table>
			<div v-if="pagination.total > pagination.pageSize" class="flex justify-end p-3 border-t border-base-200">
				<div class="join">
					<button v-for="p in totalPages" :key="p" class="join-item btn btn-sm" :class="p === pagination.current ? 'btn-primary' : 'btn-ghost'" @click="goPage(p)">{{ p }}</button>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';
import { Search } from 'lucide-vue-next';
import { deploymentApi } from '@/api/deployments';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { formatTime } from '@/utils/time';
import { formatDuration } from '@/utils/status';
import type { Deployment } from '@/types/api';

const route = useRoute();
const toast = useToast();
const { loading, execute } = useStatusAsync();

const deployments = ref<Deployment[]>([]);
const searchText = ref('');
const applicationId = ref<string | undefined>(route.query.application_id as string | undefined);
const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

const badgeMap: Record<string, string> = { ran_to_completion: 'badge-success', faulted: 'badge-error', running: 'badge-info', queued: 'badge-warning', canceled: 'badge-ghost' };
function deployBadgeClass(s: string) { return badgeMap[s] ?? 'badge-ghost'; }

async function fetchDeployments() {
	try {
		await execute(async () => {
			const res = await deploymentApi.list({ page: pagination.current, per_page: pagination.pageSize, search: searchText.value || undefined, application_id: applicationId.value });
			deployments.value = res.items;
			pagination.total = res.total;
		});
	} catch { toast.error('获取部署记录失败'); }
}

function handleSearch() { pagination.current = 1; fetchDeployments(); }
function goPage(p: number) { pagination.current = p; fetchDeployments(); }

async function handleCancel(id: string) {
	try {
		await execute(async () => { await deploymentApi.cancel(id); toast.success('已取消部署'); fetchDeployments(); });
	} catch { toast.error('取消失败'); }
}

onMounted(fetchDeployments);
</script>
