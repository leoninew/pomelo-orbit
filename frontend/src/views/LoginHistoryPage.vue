<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">登录历史</h1>
			<label class="input input-sm input-bordered flex items-center gap-2">
				<Search class="size-3.5 text-base-content/60" />
				<input
					v-model="searchText"
					type="text"
					placeholder="搜索用户名"
					class="w-36"
					@keyup.enter="handleSearch"
				/>
			</label>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table">
				<thead>
					<tr class="text-base-content/60">
						<th>用户名</th>
						<th>IP 地址</th>
						<th>用户代理</th>
						<th>状态</th>
						<th>登录时间</th>
					</tr>
				</thead>
				<tbody>
					<tr v-if="loading">
						<td colspan="5" class="text-center py-8">
							<span class="loading loading-spinner loading-md text-primary" />
						</td>
					</tr>
					<tr v-else-if="history.length === 0">
						<td colspan="5" class="text-center py-8 text-base-content/60">暂无记录</td>
					</tr>
					<tr v-for="h in history" :key="h.id" class="hover">
						<td class="font-medium">{{ h.username }}</td>
						<td>{{ h.ip_address }}</td>
						<td class="cell-muted max-w-xs truncate">{{ h.user_agent }}</td>
						<td>
							<span
								class="badge badge-sm"
								:class="h.success ? 'badge-outline badge-success' : 'badge-outline badge-error'"
							>
								{{ h.success ? '成功' : '失败' }}
							</span>
						</td>
						<td class="cell-muted">{{ formatTime(h.login_at) }}</td>
					</tr>
				</tbody>
			</table>
			<div
				v-if="pagination.total > pagination.pageSize"
				class="flex justify-end p-3 border-t border-base-200"
			>
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
	</div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Search } from 'lucide-vue-next';
import { authApi } from '@/api/auth';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { formatTime } from '@/utils/time';
import type { LoginHistory } from '@/types/api';

const toast = useToast();
const { loading, execute } = useStatusAsync();
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

onMounted(fetchHistory);
</script>
