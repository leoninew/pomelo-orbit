<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>路由配置</h2>
			<a-space>
				<a-button type="primary" @click="showCreateModal = true">添加路由</a-button>
				<a-button @click="handleSync">同步全部</a-button>
			</a-space>
		</div>

		<a-table
			:columns="columns"
			:data-source="routes"
			:loading="loading"
			:pagination="pagination"
			row-key="id"
			@change="handleTableChange"
		>
			<template #bodyCell="{ column, record }">
				<template v-if="column.key === 'name'">
					<router-link :to="`/cd/routes/${record.id}`">
						{{ record.name }}
					</router-link>
				</template>
				<template v-if="column.key === 'domain'">
					<a
						:href="`${record.https_enabled ? 'https' : 'http'}://${record.domain}`"
						target="_blank"
					>
						{{ record.domain }}
						<LinkOutlined />
					</a>
				</template>
				<template v-if="column.key === 'enabled'">
					<a-tag :color="record.enabled ? 'success' : 'default'">
						{{ record.enabled ? '启用' : '停用' }}
					</a-tag>
				</template>
				<template v-if="column.key === 'https'">
					<a-tag :color="record.https_enabled ? 'green' : 'default'">
						{{ record.https_enabled ? 'HTTPS' : 'HTTP' }}
					</a-tag>
				</template>
				<template v-if="column.key === 'action'">
					<a-space>
						<a @click="$router.push(`/cd/routes/${record.id}`)">查看</a>
						<a v-if="!record.enabled" @click="handleEnable(record.id)">启用</a>
						<a v-else @click="handleDisable(record.id)">停用</a>
					</a-space>
				</template>
			</template>
		</a-table>

		<!-- 添加路由弹窗 -->
		<a-modal v-model:open="showCreateModal" title="添加路由">
			<a-form
				ref="formRef"
				:model="form"
				:rules="formRules"
				:label-col="{ span: 6 }"
				:wrapper-col="{ span: 16 }"
			>
				<a-form-item label="路由名称" name="name">
					<a-input v-model:value="form.name" placeholder="my-route" />
				</a-form-item>
				<a-form-item label="域名" name="domain">
					<a-input v-model:value="form.domain" placeholder="example.com" />
				</a-form-item>
				<a-form-item label="路径前缀" name="path_prefix">
					<a-input v-model:value="form.path_prefix" placeholder="/" />
				</a-form-item>
				<a-form-item label="目标地址" name="target_url">
					<a-input v-model:value="form.target_url" placeholder="host.docker.internal:端口" />
				</a-form-item>
				<a-form-item label="启用" name="enabled">
					<a-switch v-model:checked="form.enabled" />
				</a-form-item>
			</a-form>
			<template #footer>
				<a-button type="primary" :loading="operating" @click="handleSave">保存</a-button>
				<a-button @click="showCreateModal = false">取消</a-button>
			</template>
		</a-modal>
	</a-space>
</template>

<script setup lang="ts">
import type { FormInstance } from 'ant-design-vue';
import { message } from 'ant-design-vue';
import { onMounted, reactive, ref } from 'vue';
import { routeApi, type Route } from '@/api/route';
import { useStatusAsync } from '@/composables/useStatusAsync';

const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const routes = ref<Route[]>([]);
const showCreateModal = ref(false);
const formRef = ref<FormInstance>();

const columns = [
	{ title: '路由名称', key: 'name', dataIndex: 'name', width: 150 },
	{ title: '域名', key: 'domain', dataIndex: 'domain', width: 180 },
	{ title: '路径前缀', key: 'path_prefix', dataIndex: 'path_prefix', width: 100 },
	{ title: '目标地址', key: 'target_url', dataIndex: 'target_url', width: 220 },
	{ title: '状态', key: 'enabled', width: 80 },
	{ title: '协议', key: 'https', width: 80 },
	{ title: '操作', key: 'action', width: 180 },
];

const pagination = reactive({
	current: 1,
	pageSize: 10,
	total: 0,
	showSizeChanger: true,
	showTotal: (total: number) => `共 ${total} 条`,
});

const form = reactive({
	name: '',
	domain: '',
	path_prefix: '/',
	target_url: 'http://',
	enabled: false,
});

const formRules = {
	name: [
		{ required: true, message: '请输入路由名称' },
		{
			pattern: /^[a-z][a-z0-9._-]*$/,
			message: '必须以小写字母开头，只能包含小写字母、数字、点号、下划线和连字符',
		},
	],
	domain: [{ required: true, message: '请输入域名' }],
	path_prefix: [{ required: true, message: '请输入路径前缀' }],
	target_url: [
		{ required: true, message: '请输入目标地址' },
		{
			pattern: /^https?:\/\/[a-zA-Z0-9.-]+:\d+$/,
			message: '格式应为 http://host:port',
		},
	],
};

async function fetchData() {
	try {
		await execute(async () => {
			const res = await routeApi.list(pagination.current, pagination.pageSize);
			routes.value = res.items;
			pagination.total = res.total;
		});
	} catch (error) {
		message.error('获取数据失败: ' + error);
	}
}

function handleTableChange(pag: { current?: number; pageSize?: number }) {
	pagination.current = pag.current || 1;
	pagination.pageSize = pag.pageSize || 10;
	fetchData();
}

async function handleSave() {
	try {
		await formRef.value?.validate();
	} catch {
		return;
	}

	try {
		await executeOp(async () => {
			await routeApi.create(form);
			message.success('添加成功');
			showCreateModal.value = false;
			Object.assign(form, {
				name: '',
				domain: '',
				path_prefix: '/',
				target_url: '',
				enabled: false,
			});
			fetchData();
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '保存失败');
	}
}

async function handleEnable(id: string) {
	try {
		await executeOp(async () => {
			await routeApi.enable(id);
			message.success('启用成功');
			fetchData();
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '启用失败');
	}
}

async function handleDisable(id: string) {
	try {
		await executeOp(async () => {
			await routeApi.disable(id);
			message.success('停用成功');
			fetchData();
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '停用失败');
	}
}

async function handleSync() {
	try {
		await executeOp(async () => {
			await routeApi.sync();
			message.success('同步成功');
			fetchData();
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '同步失败');
	}
}

onMounted(() => {
	fetchData();
});
</script>

<style scoped>
.page-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	min-height: 32px;
}

.page-header h2 {
	margin: 0;
}
</style>
