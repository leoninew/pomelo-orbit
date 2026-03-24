<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>登录历史</h2>
			<a-input-search
				v-model:value="searchText"
				placeholder="搜索用户名"
				style="width: 200px"
				@search="handleSearch"
			/>
		</div>

		<a-table
			:columns="columns"
			:data-source="history"
			:loading="loading"
			:pagination="pagination"
			row-key="id"
			@change="handleTableChange"
		>
			<template #bodyCell="{ column, record }">
				<template v-if="column.key === 'success'">
					<a-tag :color="record.success ? 'success' : 'error'">
						{{ record.success ? '成功' : '失败' }}
					</a-tag>
				</template>
				<template v-else-if="column.key === 'login_at'">
					{{ formatTime(record.login_at) }}
				</template>
			</template>
		</a-table>
	</a-space>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue';
import { formatTime } from '@/utils/time';
import { onMounted, reactive, ref } from 'vue';
import { authApi } from '@/api/auth';
import { useStatusAsync } from '@/composables/useStatusAsync';
import type { LoginHistory } from '@/types/api';

const { loading, execute } = useStatusAsync();
const history = ref<LoginHistory[]>([]);
const searchText = ref('');
const pagination = reactive({
	current: 1,
	pageSize: 10,
	total: 0,
	showSizeChanger: true,
	showTotal: (total: number) => `共 ${total} 条`,
});

const columns = [
	{ title: '用户名', dataIndex: 'username', width: 120 },
	{ title: 'IP地址', dataIndex: 'ip_address', width: 140 },
	{ title: '用户代理', dataIndex: 'user_agent', width: 400, ellipsis: true },
	{ title: '状态', key: 'success', width: 80 },
	{ title: '登录时间', key: 'login_at', width: 180 },
];

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
	} catch (error) {
		message.error('获取登录历史失败\n' + error);
	}
}

function handleSearch() {
	pagination.current = 1;
	fetchHistory();
}

function handleTableChange(pag: { current?: number; pageSize?: number }) {
	pagination.current = pag.current || 1;
	pagination.pageSize = pag.pageSize || 10;
	fetchHistory();
}

onMounted(() => {
	fetchHistory();
});
</script>

<style scoped>
.page-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.page-header h2 {
	margin: 0;
}
</style>
