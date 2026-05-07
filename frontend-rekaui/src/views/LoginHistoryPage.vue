<template>
	<div class="space-y-6">
		<ToolbarRoot class="flex items-center" aria-label="登录历史工具栏">
			<SearchControl
				v-model="searchText"
				placeholder="搜索 IP 地址或用户代理"
				:loading="status === 'loading'"
				@search="handleSearch"
			/>
		</ToolbarRoot>

		<!-- Table Card -->
		<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div v-if="status === 'loading'" class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error?.message || '加载失败' }}</p>
			</div>
			<div v-else-if="history.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="w-full">
					<thead class="border-b border-border bg-muted/30">
						<tr>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">登录时间</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">IP 地址</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">用户代理</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">状态</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border">
						<tr v-for="record in history" :key="record.id" class="transition-colors hover:bg-muted/30">
							<td class="px-6 py-5 text-sm text-foreground">{{ formatTime(record.login_at) }}</td>
							<td class="px-6 py-5 text-sm text-foreground">{{ record.ip_address }}</td>
							<td class="px-6 py-5 text-sm text-foreground max-w-md truncate">{{ record.user_agent }}</td>
							<td class="px-6 py-5 text-sm">
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
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { authApi } from '@/api/auth';
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
