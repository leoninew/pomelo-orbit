<template>
	<div class="space-y-6">
		<ToolbarRoot class="app-toolbar-scroll" aria-label="部署记录工具栏">
			<div class="app-toolbar-row">
				<SearchControl
					v-model="searchText"
					placeholder="搜索应用/环境"
					:loading="status === 'loading'"
					class="shrink-0"
					@search="handleSearch"
				/>
			</div>
		</ToolbarRoot>

		<!-- Table Card -->
		<div class="app-surface">
			<AppSpinner v-if="status === 'loading'" class="py-16" />
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error || '加载失败' }}</p>
			</div>
			<div v-else-if="deployments.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[1360px]">
					<thead>
						<tr>
							<th>应用</th>
							<th>操作类型</th>
							<th>触发方式</th>
							<th>环境</th>
							<th>环境文件</th>
							<th>状态</th>
							<th>错误信息</th>
							<th>开始时间</th>
							<th>耗时</th>
							<th>操作</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="deployment in deployments" :key="deployment.id">
							<td>
								<button
									class="app-link"
									@click="router.push(`/cd/applications/${deployment.application_id}`)"
								>
									{{ deployment.application_name || deployment.application_id }}
								</button>
							</td>
							<td class="text-foreground">{{ operationTypeLabel(deployment.operation_type) }}</td>
							<td class="text-foreground">{{ triggerTypeLabel(deployment.trigger_type) }}</td>
							<td class="text-foreground">{{ deployment.environment || '—' }}</td>
							<td
								class="max-w-48 truncate text-foreground"
								:title="deployment.env_file || undefined"
							>
								{{ deployment.env_file || '—' }}
							</td>
							<td>
								<span
									class="inline-flex rounded-md px-2 py-0.5 text-sm"
									:class="statusBadgeClass(deployment.status)"
								>
									{{ statusLabel(deployment.status) }}
								</span>
							</td>
							<td
								class="max-w-56 truncate text-destructive"
								:title="deployment.error_message || undefined"
							>
								{{ deployment.error_message || '—' }}
							</td>
							<td class="text-foreground">{{ formatTime(deployment.started_at) }}</td>
							<td class="text-foreground">{{ formatDuration(deployment.duration_ms) }}</td>
							<td>
								<div class="flex items-center gap-3">
									<button class="app-link" @click="router.push(`/cd/deployments/${deployment.id}`)">
										查看
									</button>
									<button
										v-if="isCancelable(deployment)"
										class="app-link-danger"
										:disabled="operating"
										@click="openCancelDialog(deployment)"
									>
										取消
									</button>
								</div>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>

		<ListPagination
			:current="pagination.current"
			:page-size="pagination.pageSize"
			:total="pagination.total"
			:total-pages="totalPages"
			@change-page="goPage"
			@change-page-size="handlePageSizeChange"
		/>

		<AppDialog
			v-model:open="isCancelDialogOpen"
			title="确认取消"
			:description="`确定要取消 ${deploymentToCancel?.application_name || '该应用'} 的部署吗？`"
			width-class="w-[min(420px,calc(100vw-32px))]"
			body-class="hidden"
		>
			<template #footer>
				<button class="app-button" @click="isCancelDialogOpen = false">取消</button>
				<button class="app-button-destructive" :disabled="operating" @click="handleCancelOk">
					确认取消
				</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { computed, onMounted, reactive, ref } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { deploymentApi } from '@/api/cd/deployments';
	import AppDialog from '@/components/AppDialog.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import ListPagination from '@/components/ListPagination.vue';
	import SearchControl from '@/components/SearchControl.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import type { Deployment } from '@/types/cd/deployment';
	import { formatDuration, statusBadgeClass, statusLabel } from '@/utils/status';
	import { formatTime } from '@/utils/time';
	import { ToolbarRoot } from 'reka-ui';

	const router = useRouter();
	const route = useRoute();
	const toast = useToast();
	const { status, error, execute } = useStatusAsync();
	const { loading: operating, execute: executeOp } = useStatusAsync();

	const deployments = ref<Deployment[]>([]);
	const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
	const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
	const searchText = ref('');
	const applicationId = ref<string | undefined>(route.query.application_id as string | undefined);
	const isCancelDialogOpen = ref(false);
	const deploymentToCancel = ref<Deployment | null>(null);

	function operationTypeLabel(type: string) {
		const map: Record<string, string> = {
			deploy: '部署',
			stop: '停止',
			restart: '重启',
		};
		return map[type] ?? type;
	}

	function triggerTypeLabel(type: string) {
		const map: Record<string, string> = {
			manual: '手动',
		};
		return map[type] ?? type;
	}

	async function fetchDeployments() {
		try {
			await execute(async () => {
				const res = await deploymentApi.list({
					page: pagination.current,
					per_page: pagination.pageSize,
					search: searchText.value || undefined,
					application_id: applicationId.value,
				});
				deployments.value = res.items;
				pagination.total = res.total;
			});
		} catch {
			toast.error('获取部署记录失败');
		}
	}

	function handleSearch() {
		pagination.current = 1;
		fetchDeployments();
	}

	function goPage(p: number) {
		pagination.current = p;
		fetchDeployments();
	}

	function handlePageSizeChange(pageSize: number) {
		pagination.pageSize = pageSize;
		pagination.current = 1;
		fetchDeployments();
	}

	function isCancelable(deployment: Deployment) {
		return ['running', 'waiting_to_run'].includes(deployment.status);
	}

	function openCancelDialog(deployment: Deployment) {
		deploymentToCancel.value = deployment;
		isCancelDialogOpen.value = true;
	}

	async function handleCancelOk() {
		if (!deploymentToCancel.value) {
			return;
		}
		const target = deploymentToCancel.value;
		try {
			await executeOp(async () => {
				await deploymentApi.cancel(target.id);
				toast.success('已取消部署');
				isCancelDialogOpen.value = false;
				deploymentToCancel.value = null;
				await fetchDeployments();
			});
		} catch {
			toast.error('取消失败');
		}
	}

	onMounted(fetchDeployments);
</script>
