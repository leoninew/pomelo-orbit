<template>
	<div class="space-y-6">
		<ToolbarRoot class="flex items-center" aria-label="登录历史工具栏">
			<SearchControl
				v-model="searchText"
				placeholder="搜索用户名、IP 地址或用户代理"
				:loading="status === 'loading'"
				@search="handleSearch"
			/>
		</ToolbarRoot>

		<!-- Table Card -->
		<div class="app-surface">
			<AppSpinner v-if="status === 'loading'" class="py-16" />
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error || '加载失败' }}</p>
			</div>
			<div v-else-if="history.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[920px]">
					<thead>
						<tr>
							<th>登录时间</th>
							<th>用户名</th>
							<th>IP 地址</th>
							<th>用户代理</th>
							<th>状态</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="record in history" :key="record.id">
							<td class="text-foreground">{{ formatTime(record.login_at) }}</td>
							<td class="text-foreground">{{ record.username }}</td>
							<td class="text-foreground">{{ record.ip_address }}</td>
							<td class="max-w-md truncate text-foreground" :title="record.user_agent || undefined">
								{{ record.user_agent || '-' }}
							</td>
							<td>
								<span
									:class="[
										'inline-flex rounded-md border px-2 py-0.5 text-sm',
										record.success
											? 'border-green-200 bg-green-50 text-green-700'
											: 'border-red-200 bg-red-50 text-red-700',
									]"
								>
									{{ record.success ? '成功' : '失败' }}
								</span>
							</td>
						</tr>
					</tbody>
				</table>
			</div>

			<ListPagination
				:current="pagination.current"
				:page-size="pagination.pageSize"
				:total="pagination.total"
				:total-pages="totalPages"
				@change-page="goPage"
				@change-page-size="handlePageSizeChange"
			/>
		</div>
	</div>
</template>

<script setup lang="ts">
	import { computed, onMounted, reactive, ref } from 'vue';
	import { authApi } from '@/api/auth';
	import AppSpinner from '@/components/AppSpinner.vue';
	import ListPagination from '@/components/ListPagination.vue';
	import SearchControl from '@/components/SearchControl.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import type { LoginHistory } from '@/types/auth';
	import { formatTime } from '@/utils/time';
	import { ToolbarRoot } from 'reka-ui';

	const toast = useToast();
	const { status, error, execute } = useStatusAsync();
	const history = ref<LoginHistory[]>([]);
	const searchText = ref('');
	const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
	const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

	async function fetchHistory() {
		try {
			await execute(async () => {
				const res = await authApi.listLoginHistory({
					page: pagination.current,
					per_page: pagination.pageSize,
					search: searchText.value || undefined,
				});
				history.value = res.items;
				pagination.total = res.total;
			});
		} catch {
			toast.error('获取登录历史失败');
		}
	}

	function handleSearch() {
		pagination.current = 1;
		fetchHistory();
	}
	function goPage(p: number) {
		pagination.current = p;
		fetchHistory();
	}

	function handlePageSizeChange(pageSize: number) {
		pagination.pageSize = pageSize;
		pagination.current = 1;
		fetchHistory();
	}

	onMounted(fetchHistory);
</script>
