<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>回调事件</h2>
			<a-space>
				<a-input-search
					v-model:value="searchText"
					placeholder="搜索仓库或发送者"
					style="width: 200px"
					@search="handleSearch"
				/>
				<a-select
					v-model:value="filters.source"
					style="width: 120px"
					placeholder="来源"
					allow-clear
					@change="fetchEvents"
				>
					<a-select-option value="github">Github</a-select-option>
					<a-select-option value="gitlab">GitLab</a-select-option>
				</a-select>
				<a-select
					v-model:value="filters.status"
					style="width: 120px"
					placeholder="状态"
					allow-clear
					@change="fetchEvents"
				>
					<a-select-option value="received">已接收</a-select-option>
					<a-select-option value="matched">已匹配</a-select-option>
					<a-select-option value="ignored">已忽略</a-select-option>
					<a-select-option value="error">错误</a-select-option>
				</a-select>
			</a-space>
		</div>

		<a-table
			:columns="columns"
			:data-source="events"
			:loading="loading"
			:pagination="pagination"
			row-key="id"
			@change="handleTableChange"
		>
			<template #bodyCell="{ column, record }">
				<template v-if="column.key === 'source'">
					<a-tag :color="record.source === 'github' ? 'blue' : 'orange'">
						{{ record.source }}
					</a-tag>
				</template>
				<template v-else-if="column.key === 'status'">
					<a-tag :color="getStatusColor(record.status)">{{ record.status }}</a-tag>
				</template>
				<template v-else-if="column.key === 'signature_valid'">
					<a-tag v-if="record.signature_valid === null" color="default">未验证</a-tag>
					<a-tag v-else-if="record.signature_valid" color="success">有效</a-tag>
					<a-tag v-else color="error">无效</a-tag>
				</template>
				<template v-else-if="column.key === 'received_at'">
					{{ formatTime(record.received_at) }}
				</template>
				<template v-else-if="column.key === 'actions'">
					<a-button type="link" size="small" block @click="$router.push(`/events/${record.id}`)">
						查看
					</a-button>
				</template>
			</template>
		</a-table>
	</a-space>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue';
import { formatTime } from '@/utils/time';
import { onMounted, reactive, ref } from 'vue';
import { eventApi } from '@/api/event';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { eventStatusColors, type WebhookEvent } from '@/types/api';

const { loading, execute } = useStatusAsync();
const events = ref<WebhookEvent[]>([]);
const searchText = ref('');

const filters = reactive({
	source: undefined as string | undefined,
	status: undefined as string | undefined,
});

const pagination = reactive({
	current: 1,
	pageSize: 10,
	total: 0,
	showSizeChanger: true,
	showTotal: (total: number) => `共 ${total} 条`,
});

const columns = [
	{ title: '来源', key: 'source', width: 90 },
	{ title: '事件类型', dataIndex: 'event_type', width: 120 },
	{ title: '仓库', dataIndex: 'repository_name', width: 250, ellipsis: true },
	{ title: '分支', dataIndex: 'branch', width: 120 },
	{ title: '签名', key: 'signature_valid', width: 90 },
	{ title: '状态', key: 'status', width: 90 },
	{ title: '接收时间', key: 'received_at', width: 180 },
	{ title: '操作', key: 'actions', width: 80 },
];

function getStatusColor(status: string) {
	return eventStatusColors[status] || 'default';
}

async function fetchEvents() {
	try {
		await execute(async () => {
			const res = await eventApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				search: searchText.value || undefined,
				source: filters.source,
				status: filters.status,
			});
			events.value = res.items;
			pagination.total = res.total;
		});
	} catch (error) {
		message.error('获取事件列表失败\n' + error);
	}
}

function handleSearch() {
	pagination.current = 1;
	fetchEvents();
}

function handleTableChange(pag: { current?: number; pageSize?: number }) {
	pagination.current = pag.current || 1;
	pagination.pageSize = pag.pageSize || 10;
	fetchEvents();
}

onMounted(() => {
	fetchEvents();
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
