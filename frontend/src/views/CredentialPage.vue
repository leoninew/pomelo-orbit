<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>凭据管理</h2>
			<a-button type="primary" @click="showCreateModal">
				<template #icon><PlusOutlined /></template>
				新建凭据
			</a-button>
		</div>

		<a-table
			:columns="columns"
			:data-source="credentials"
			:loading="loading"
			:pagination="pagination"
			row-key="id"
			@change="handleTableChange"
		>
			<template #bodyCell="{ column, record }">
				<template v-if="column.key === 'type'">
					<a-tag>{{ credentialTypeLabels[record.type] }}</a-tag>
				</template>
				<template v-else-if="column.key === 'created_at'">
					{{ formatTime(record.created_at) }}
				</template>
				<template v-else-if="column.key === 'actions'">
					<a-space>
						<a @click="showUpdateModal(record)">编辑</a>
						<a-popconfirm
							title="确定删除此凭据？"
							ok-text="删除"
							cancel-text="取消"
							@confirm="handleDelete(record.id)"
						>
							<a style="color: #ff4d4f">删除</a>
						</a-popconfirm>
					</a-space>
				</template>
			</template>
		</a-table>

		<!-- 创建/编辑凭据弹窗 -->
		<a-modal
			v-model:open="showModal"
			:title="isEditing ? '编辑凭据' : '新建凭据'"
			@ok="handleModalOk"
		>
			<a-form
				ref="formRef"
				:model="form"
				:rules="formRules"
				:label-col="{ span: 6 }"
				:wrapper-col="{ span: 16 }"
			>
				<a-form-item label="凭据名称" name="name">
					<a-input v-model:value="form.name" placeholder="例如: GitHub SSH Key" />
				</a-form-item>
				<a-form-item label="凭据类型" name="type">
					<a-select v-model:value="form.type" :disabled="isEditing">
						<a-select-option value="git_ssh">Git SSH</a-select-option>
						<a-select-option value="git_token">Git Token</a-select-option>
						<a-select-option value="registry_token">Registry Token</a-select-option>
					</a-select>
				</a-form-item>
				<a-form-item label="凭据内容" name="data">
					<a-textarea
						v-model:value="form.data"
						:rows="8"
						:placeholder="getDataPlaceholder(form.type)"
					/>
				</a-form-item>
			</a-form>
			<template #footer>
				<a-button @click="showModal = false">取消</a-button>
				<a-button type="primary" :loading="operating" @click="handleModalOk">保存</a-button>
			</template>
		</a-modal>
	</a-space>
</template>

<script setup lang="ts">
import type { FormInstance } from 'ant-design-vue';
import { message } from 'ant-design-vue';
import { onMounted, reactive, ref } from 'vue';
import { ciCredentialApi } from '@/api/ci';
import { formatTime } from '@/utils/time';
import { useStatusAsync } from '@/composables/useStatusAsync';
import type { Credential } from '@/types/api';
import { credentialTypeLabels } from '@/types/api';

const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const credentials = ref<Credential[]>([]);
const pagination = reactive({
	current: 1,
	pageSize: 10,
	total: 0,
	showSizeChanger: true,
	showTotal: (total: number) => `共 ${total} 条`,
});

const columns = [
	{ title: '凭据名称', key: 'name', dataIndex: 'name' },
	{ title: '类型', key: 'type', width: 150 },
	{ title: '创建时间', key: 'created_at', width: 180 },
	{ title: '操作', key: 'actions', width: 150 },
];

const showModal = ref(false);
const isEditing = ref(false);
const currentId = ref('');
const formRef = ref<FormInstance>();

const form = reactive({
	name: '',
	type: 'git_ssh' as 'git_ssh' | 'git_token' | 'registry_token',
	data: '',
});

const formRules = {
	name: [{ required: true, message: '请输入凭据名称' }],
	type: [{ required: true, message: '请选择凭据类型' }],
	data: [{ required: true, message: '请输入凭据内容' }],
};

async function fetchCredentials() {
	try {
		await execute(async () => {
			const res = await ciCredentialApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
			});
			credentials.value = res.items;
			pagination.total = res.total;
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取凭据列表失败');
	}
}

function handleTableChange(pag: { current?: number; pageSize?: number }) {
	pagination.current = pag.current || 1;
	pagination.pageSize = pag.pageSize || 10;
	fetchCredentials();
}

function showCreateModal() {
	isEditing.value = false;
	currentId.value = '';
	Object.assign(form, {
		name: '',
		type: 'git_ssh',
		data: '',
	});
	showModal.value = true;
}

function showUpdateModal(record: Credential) {
	isEditing.value = true;
	currentId.value = record.id;
	Object.assign(form, {
		name: record.name,
		type: record.type,
		data: '',
	});
	showModal.value = true;
}

async function handleModalOk() {
	try {
		await formRef.value?.validate();
	} catch {
		return;
	}

	try {
		await executeOp(async () => {
			if (isEditing.value) {
				const payload: { name?: string; data?: string } = { name: form.name };
				if (form.data) {
					payload.data = form.data;
				}
				await ciCredentialApi.update(currentId.value, payload);
				message.success('更新成功');
			} else {
				await ciCredentialApi.create({
					name: form.name,
					type: form.type,
					data: form.data,
				});
				message.success('创建成功');
			}
			showModal.value = false;
			fetchCredentials();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '操作失败');
	}
}

async function handleDelete(id: string) {
	try {
		await executeOp(async () => {
			await ciCredentialApi.delete(id);
			message.success('删除成功');
			fetchCredentials();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '删除失败');
	}
}

function getDataPlaceholder(type: string): string {
	if (type === 'git_ssh') {
		return '-----BEGIN OPENSSH PRIVATE KEY-----\n...';
	}
	if (type === 'git_token') {
		return 'ghp_xxxxxxxxxxxxxxxxxxxx';
	}
	return 'registry_token_here';
}

onMounted(() => {
	fetchCredentials();
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
