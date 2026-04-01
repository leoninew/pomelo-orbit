<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>Pipeline 模板</h2>
			<a-button type="primary" @click="showCreateModal">
				<template #icon><PlusOutlined /></template>
				新建模板
			</a-button>
		</div>

		<a-table
			:columns="columns"
			:data-source="templates"
			:loading="loading"
			:pagination="pagination"
			row-key="id"
			@change="handleTableChange"
		>
			<template #bodyCell="{ column, record }">
				<template v-if="column.key === 'name'">
					<router-link :to="`/ci/templates/${record.id}`">
						{{ record.name }}
					</router-link>
					<a-tag v-if="record.is_builtin" color="blue" style="margin-left: 8px">内置</a-tag>
				</template>
				<template v-else-if="column.key === 'variables'">
					{{ (record.variable_declarations ?? []).length }} 个变量
				</template>
				<template v-else-if="column.key === 'created_at'">
					{{ formatTime(record.created_at) }}
				</template>
				<template v-else-if="column.key === 'actions'">
					<a-space>
						<a @click="$router.push(`/ci/templates/${record.id}`)">查看</a>
						<a-popconfirm
							v-if="!record.is_builtin"
							title="确定删除此模板？"
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

		<!-- 创建模板弹窗 -->
		<a-modal
			v-model:open="showModal"
			title="新建模板"
			width="800px"
			@ok="handleModalOk"
		>
			<a-form
				ref="formRef"
				:model="form"
				:rules="formRules"
				:label-col="{ span: 4 }"
				:wrapper-col="{ span: 19 }"
			>
				<a-form-item label="模板名称" name="name">
					<a-input v-model:value="form.name" placeholder="例如: Python FastAPI 构建" />
				</a-form-item>
				<a-form-item label="描述">
					<a-textarea v-model:value="form.description" :rows="2" />
				</a-form-item>
				<a-form-item label="Pipeline YAML" name="content">
					<a-textarea
						v-model:value="form.content"
						:rows="12"
						placeholder="version: v1&#10;steps:&#10;  - name: checkout&#10;    uses: checkout"
						style="font-family: 'Consolas', 'Monaco', monospace"
					/>
				</a-form-item>
			</a-form>
			<template #footer>
				<a-button @click="showModal = false">取消</a-button>
				<a-button type="primary" :loading="operating" @click="handleModalOk">创建</a-button>
			</template>
		</a-modal>
	</a-space>
</template>

<script setup lang="ts">
import type { FormInstance } from 'ant-design-vue';
import { message } from 'ant-design-vue';
import { onMounted, reactive, ref } from 'vue';
import { pipelineTemplateApi } from '@/api/ci';
import { formatTime } from '@/utils/time';
import { useStatusAsync } from '@/composables/useStatusAsync';
import type { PipelineTemplate } from '@/types/api';

const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const templates = ref<PipelineTemplate[]>([]);
const pagination = reactive({
	current: 1,
	pageSize: 10,
	total: 0,
	showSizeChanger: true,
	showTotal: (total: number) => `共 ${total} 条`,
});

const columns = [
	{ title: '模板名称', key: 'name', dataIndex: 'name' },
	{ title: '描述', key: 'description', dataIndex: 'description', ellipsis: true },
	{ title: '变量', key: 'variables', width: 120 },
	{ title: '创建时间', key: 'created_at', width: 180 },
	{ title: '操作', key: 'actions', width: 150 },
];

const showModal = ref(false);
const formRef = ref<FormInstance>();

const form = reactive({
	name: '',
	description: '',
	content: '',
});

const formRules = {
	name: [{ required: true, message: '请输入模板名称' }],
	content: [{ required: true, message: '请输入 Pipeline YAML' }],
};

async function fetchTemplates() {
	try {
		await execute(async () => {
			const res = await pipelineTemplateApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
			});
			templates.value = res.items;
			pagination.total = res.total;
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取模板列表失败');
	}
}

function handleTableChange(pag: { current?: number; pageSize?: number }) {
	pagination.current = pag.current || 1;
	pagination.pageSize = pag.pageSize || 10;
	fetchTemplates();
}

function showCreateModal() {
	Object.assign(form, {
		name: '',
		description: '',
		content: '',
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
			await pipelineTemplateApi.create({
				name: form.name,
				description: form.description || undefined,
				content: form.content,
			});
			message.success('创建成功');
			showModal.value = false;
			fetchTemplates();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '创建失败');
	}
}

async function handleDelete(id: string) {
	try {
		await executeOp(async () => {
			await pipelineTemplateApi.delete(id);
			message.success('删除成功');
			fetchTemplates();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '删除失败');
	}
}

onMounted(() => {
	fetchTemplates();
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
