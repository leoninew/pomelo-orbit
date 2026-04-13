<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">流水线记录</h1>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table min-h-48">
				<thead>
					<tr class="text-base-content/60">
						<th>项目</th>
						<th>模板</th>
						<th>触发方式</th>
						<th>Ref</th>
						<th>状态</th>
						<th>异常信息</th>
						<th>开始时间</th>
						<th>结束时间</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					<tr v-if="status === 'loading'">
						<td colspan="9" class="text-center py-8">
							<span class="loading loading-spinner loading-md text-primary" />
						</td>
					</tr>
					<tr v-else-if="status === 'error'">
						<td colspan="9" class="text-center py-8 text-error">{{ error }}</td>
					</tr>
					<tr v-else-if="runs.length === 0">
						<td colspan="9" class="text-center py-8 text-base-content/60">暂无数据</td>
					</tr>
					<tr v-for="r in runs" :key="r.id" class="hover">
						<td>
							<router-link :to="`/ci/repository/${r.repository_id}`" class="link link-primary">
								{{ r.repository_name }}
							</router-link>
						</td>
						<td>
							<router-link :to="`/ci/template/${r.template_id}`" class="link link-primary text-xs">
								{{ r.template_name }}
							</router-link>
						</td>
						<td>
							<span class="badge badge-sm badge-ghost">{{ r.trigger }}</span>
						</td>
						<td class="cell-muted">{{ r.trigger_ref }}</td>
						<td>
							<span class="badge badge-sm" :class="statusBadgeClass(r.status)">
								{{ statusLabel(r.status) }}
							</span>
						</td>
						<td class="max-w-xs">
							<span
								v-if="r.error_message"
								class="tooltip tooltip-top cursor-help"
								:data-tip="r.error_message"
							>
								<span class="text-xs truncate block max-w-xs">{{ r.error_message }}</span>
							</span>
							<span v-else class="text-base-content/40 text-xs">—</span>
						</td>
						<td class="cell-muted">{{ r.started_at ? formatTime(r.started_at) : '—' }}</td>
						<td class="cell-muted">{{ r.finished_at ? formatTime(r.finished_at) : '—' }}</td>
						<td>
							<div class="flex items-center gap-2">
								<router-link :to="`/ci/run/${r.id}`" class="link link-primary">查看</router-link>
								<button
									v-if="r.status === 'faulted'"
									class="link link-info"
									:disabled="operating"
									@click="handleRetry(r.id)"
								>
									重试
								</button>
								<button
									v-if="r.status === 'waiting_to_run' || r.status === 'running'"
									class="link link-error"
									:disabled="operating"
									@click="confirmCancel(r.id)"
								>
									取消
								</button>
							</div>
						</td>
					</tr>
				</tbody>
			</table>
			<div v-if="totalPages > 0" class="flex justify-end p-3 border-t border-base-200">
				<div class="join">
					<button
						v-for="p in totalPages"
						:key="p"
						class="join-item btn btn-sm"
						:class="p === pagination.current ? 'btn-primary' : 'btn-ghost'"
						@click="goPage(p)"
					>
						{{ p }}
					</button>
				</div>
			</div>
		</div>

		<dialog ref="cancelModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">取消 Run</h3>
				<p class="py-4 text-sm">确定取消此 Run？</p>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="operating" @click="handleCancel">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						确定
					</button>
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
import type { PipelineRun } from '@/types/ci/run';
import { statusBadgeClass, statusLabel } from '@/utils/status';
import { formatTime } from '@/utils/time';

const route = useRoute();
const router = useRouter();
const toast = useToast();
const { status, error, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const runs = ref<PipelineRun[]>([]);
const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
const cancelModalRef = ref<HTMLDialogElement>();
const pendingCancelId = ref('');

async function fetchRuns() {
	try {
		await execute(async () => {
			const projectId = route.query.repository_id as string | undefined;
			const res = await pipelineRunApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				repository_id: projectId,
			});
			runs.value = res.items;
			pagination.total = res.total;
		});
	} catch {
		toast.error('获取运行记录失败');
	}
}

function goPage(p: number) {
	pagination.current = p;
	fetchRuns();
}

async function handleRetry(runId: string) {
	try {
		await executeOp(async () => {
			const newRun = await pipelineRunApi.retry(runId);
			toast.success('重试成功');
			router.push(`/ci/run/${newRun.id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '重试失败');
	}
}

function confirmCancel(runId: string) {
	pendingCancelId.value = runId;
	cancelModalRef.value?.showModal();
}

async function handleCancel() {
	try {
		await executeOp(async () => {
			await pipelineRunApi.cancel(pendingCancelId.value);
			toast.success('已取消');
			cancelModalRef.value?.close();
			fetchRuns();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '取消失败');
	}
}

onMounted(fetchRuns);
</script>
