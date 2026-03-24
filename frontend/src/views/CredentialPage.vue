<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>凭据管理</h2>
			<a-space>
				<a-input-search
					v-model:value="searchText"
					placeholder="搜索凭据名称"
					style="width: 200px"
					@search="handleSearch"
				/>
				<a-button type="primary" @click="showCreateModal">
					<template #icon><PlusOutlined /></template>
					新建凭据
				</a-button>
			</a-space>
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
					<a-tag :color="getTypeColor(record.type)">{{ record.type }}</a-tag>
				</template>
				<template v-else-if="column.key === 'application'">
					<a @click="$router.push(`/applications/${record.application_id}`)">查看应用</a>
				</template>
				<template v-else-if="column.key === 'created_at'">
					{{ formatTime(record.created_at) }}
				</template>
				<template v-else-if="column.key === 'actions'">
					<a-button-group>
						<a-button
							type="link"
							size="small"
							block
							@click="$router.push(`/credentials/${record.id}`)"
						>
							查看
						</a-button>
						<a-button type="link" size="small" block @click="showEditModal(record)">编辑</a-button>
						<a-popconfirm title="确定要删除此凭据吗？" @confirm="handleDelete(record.id)">
							<a-button type="link" size="small" danger block>删除</a-button>
						</a-popconfirm>
					</a-button-group>
				</template>
			</template>
		</a-table>

		<!-- 创建/编辑凭据弹窗 -->
		<a-modal v-model:open="showCredentialModal" :title="editingId ? '编辑凭据' : '新建凭据'">
			<a-form
				ref="formRef"
				:model="form"
				:rules="formRules"
				:label-col="{ span: 6 }"
				:wrapper-col="{ span: 16 }"
			>
				<a-form-item label="应用ID" name="application_id">
					<a-input v-model:value="form.application_id" placeholder="请输入应用ID" />
				</a-form-item>
				<a-form-item label="凭据名称" name="name">
					<a-input v-model:value="form.name" placeholder="例如：Github Token" />
				</a-form-item>
				<a-form-item v-if="!editingId" label="凭据类型" name="type">
					<a-select v-model:value="form.type">
						<a-select-option value="github_token">Github Token</a-select-option>
						<a-select-option value="docker_registry">Docker Registry</a-select-option>
						<a-select-option value="ssh_key">SSH Key</a-select-option>
					</a-select>
				</a-form-item>
				<a-form-item label="凭据值" :name="editingId ? undefined : 'value'">
					<a-textarea
						v-model:value="form.value"
						:rows="4"
						:placeholder="editingId ? '留空则不修改' : '请输入凭据值（将被加密存储）'"
					/>
				</a-form-item>
				<a-form-item label="附加信息" name="extra_data">
					<a-input v-model:value="form.extra_data" placeholder="JSON 格式（可选）" />
				</a-form-item>
			</a-form>
			<template #footer>
				<a-button type="primary" :loading="operating" @click="handleSubmit">保存</a-button>
				<a-button @click="showCredentialModal = false">取消</a-button>
			</template>
		</a-modal>
	</a-space>
</template>

<script setup lang="ts">
import type { FormInstance } from 'ant-design-vue';
import { message } from 'ant-design-vue';
import { onMounted, reactive, ref } from 'vue';
import { credentialApi } from '@/api/credential';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { formatTime } from '@/utils/time';
import type { Credential } from '@/types/api';

const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const credentials = ref<Credential[]>([]);
const searchText = ref('');
const pagination = reactive({
	current: 1,
	pageSize: 10,
	total: 0,
	showSizeChanger: true,
	showTotal: (total: number) => `共 ${total} 条`,
});
const showCredentialModal = ref(false);
const formRef = ref<FormInstance>();
const editingId = ref<string>();

const form = reactive({
	application_id: '',
	name: '',
	type: 'github_token',
	value: '',
	extra_data: '',
});

const formRules = {
	application_id: [{ required: true, message: '请选择所属应用' }],
	name: [{ required: true, message: '请输入凭据名称' }],
	type: [{ required: true, message: '请选择凭据类型' }],
	value: [{ required: true, message: '请输入凭据值' }],
};

const columns = [
	{ title: '名称', dataIndex: 'name' },
	{ title: '类型', key: 'type', width: 120 },
	{ title: '所属应用', key: 'application', width: 120 },
	{ title: '创建时间', key: 'created_at', width: 160 },
	{ title: '操作', key: 'actions', width: 180 },
];

function getTypeColor(type: string) {
	const colors: Record<string, string> = {
		github_token: 'blue',
		gitlab_token: 'orange',
		docker_registry: 'cyan',
		ssh_key: 'purple',
	};
	return colors[type] || 'default';
}

async function fetchCredentials() {
	try {
		await execute(async () => {
			const res = await credentialApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				search: searchText.value || undefined,
			});
			credentials.value = res.items;
			pagination.total = res.total;
		});
	} catch (error) {
		message.error('获取凭据列表失败\n' + error);
	}
}

function handleSearch() {
	pagination.current = 1;
	fetchCredentials();
}

function handleTableChange(pag: { current?: number; pageSize?: number }) {
	pagination.current = pag.current || 1;
	pagination.pageSize = pag.pageSize || 10;
	fetchCredentials();
}

function showCreateModal() {
	editingId.value = undefined;
	Object.assign(form, {
		application_id: '',
		name: '',
		type: 'github_token',
		value: '',
		extra_data: '',
	});
	showCredentialModal.value = true;
}

async function showEditModal(credential: Credential) {
	editingId.value = credential.id;

	try {
		await execute(async () => {
			// 获取凭据详情（包含值）
			const detail = await credentialApi.get(credential.id);
			form.name = detail.name;
			form.type = detail.type;
			form.value = detail.value;
			form.extra_data = detail.extra_data;
			showCredentialModal.value = true;
		});
	} catch (error) {
		message.error('获取凭据详情失败\n' + error);
	}
}

async function handleSubmit() {
	try {
		await formRef.value?.validate();
	} catch {
		return;
	}

	try {
		await executeOp(async () => {
			if (editingId.value) {
				await credentialApi.update(editingId.value, {
					name: form.name,
					value: form.value || undefined,
					extra_data: form.extra_data || undefined,
				});
				message.success('更新成功');
			} else {
				await credentialApi.create({
					application_id: form.application_id,
					name: form.name,
					type: form.type,
					value: form.value,
					extra_data: form.extra_data || undefined,
				});
				message.success('创建成功');
			}
			showCredentialModal.value = false;
			fetchCredentials();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '操作失败');
	}
}

async function handleDelete(id: string) {
	try {
		await execute(async () => {
			await credentialApi.delete(id);
			message.success('删除成功');
			fetchCredentials();
		});
	} catch (error) {
		message.error('删除失败\n' + error);
	}
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
