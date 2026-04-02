<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between">
			<h1 class="text-xl font-semibold">Pipeline Runs</h1>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table">
				<thead>
					<tr class="text-base-content/60">
						<th>Run ID</th><th>Project</th><th>触发方式</th><th>Ref</th><th>状态</th><th>重试自</th><th>创建时间</th><th>操作</th>
					</tr>
				</thead>
				<tbody>
					<tr v-if="loading"><td colspan="8" class="text-center py-8"><span class="loading loading-spinner loading-md text-primary" /></td></tr>
					<tr v-else-if="runs.length === 0"><td colspan="8" class="text-center py-8 text-base-content/40">暂无记录</td></tr>
					<tr v-for="r in runs" :key="r.id" class="hover">
						<td><router-link :to="`/ci/runs/${r.id}`" class="link link-primary cell-mono">{{ r.id.substring(0, 12) }}</router-link></td>
						<td><router-link :to="`/ci/projects/${r.project_id}`" class="link link-primary cell-mono">{{ r.project_id.substring(0, 8) }}</router-link></td>
						<td><span class="badge badge-xs badge-ghost">{{ r.trigger }}</span></td>
						<td class="cell-muted">{{ r.trigger_ref }}</td>
						<td><span class="badge badge-sm" :class="runBadgeClass(r.status)">{{ r.status }}</span></td>
						<td>
							<router-link v-if="r.retry_of" :to="`/ci/runs/${r.retry_of}`" class="link link-primary cell-mono">{{ r.retry_of.substring(0, 8) }}</router-link>
							<span v-else class="text-base-content/40">—</span>
						</td>
						<td class="cell-muted">{{ formatTime(r.created_at) }}</td>
						<td>
							<div class="flex items-center gap-2">
								<router-link :to="`/ci/runs/${r.id}`" class="link link-primary">查看</router-link>
								<button v-if="r.status === 'failed' || r.status === 'success'" class="link link-info" @click="handleRetry(r.id)">重试</button>
								<button v-if="r.status === 'waiting' || r.status === 'running'" class="link link-error" @click="confirmCancel(r.id)">取消</button>
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

		<dialog ref="cancelModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">取消 Run</h3>
				<p class="py-4">确定取消此 Run？</p>
				<div class="modal-action">
					<button class="btn btn-error" @click="handleCancel">确定</button>
					<button class="btn btn-ghost" @click="cancelModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { pipelineRunApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { formatTime } from '@/utils/time';
import type { PipelineRun } from '@/types/api';

const route = useRoute();
const router = useRouter();
const toast = useToast();
const { loading, execute } = useStatusAsync();

const runs = ref<PipelineRun[]>([]);
const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
const cancelModalRef = ref<HTMLDialogElement>();
const pendingCancelId = ref('');

const badgeMap: Record<string, string> = { success: 'badge-success', failed: 'badge-error', running: 'badge-info', waiting: 'badge-warning', canceled: 'badge-ghost' };
function runBadgeClass(s: string) { return badgeMap[s] ?? 'badge-ghost'; }

async function fetchRuns() {
	try {
		await execute(async () => {
			const projectId = route.query.project_id as string | undefined;
			const res = await pipelineRunApi.list({ page: pagination.current, per_page: pagination.pageSize, project_id: projectId });
			runs.value = res.items; pagination.total = res.total;
		});
	} catch { toast.error('获取运行记录失败'); }
}

function goPage(p: number) { pagination.current = p; fetchRuns(); }

async function handleRetry(runId: string) {
	try {
		const newRun = await pipelineRunApi.retry(runId);
		toast.success('重试成功');
		router.push(`/ci/runs/${newRun.id}`);
	} catch (error) { toast.error(error instanceof Error ? error.message : '重试失败'); }
}

function confirmCancel(runId: string) { pendingCancelId.value = runId; cancelModalRef.value?.showModal(); }

async function handleCancel() {
	try {
		await pipelineRunApi.cancel(pendingCancelId.value);
		toast.success('已取消'); cancelModalRef.value?.close(); fetchRuns();
	} catch (error) { toast.error(error instanceof Error ? error.message : '取消失败'); }
}

onMounted(fetchRuns);
</script>
