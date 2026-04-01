<template>
	<a-space direction="vertical" style="width: 100%">
		<div v-if="template" class="page-header">
			<h2>
				{{ template.name }}
				<a-tag v-if="template.is_builtin" color="blue" style="margin-left: 8px">内置</a-tag>
			</h2>
			<a-space>
				<a-button @click="$router.push('/ci/templates')">返回</a-button>
				<a-button v-if="!template.is_builtin" type="primary" @click="showEditModal = true">
					编辑
				</a-button>
			</a-space>
		</div>

		<a-card title="基本信息" :loading="loading">
			<a-descriptions v-if="template" :column="1" bordered size="small">
				<a-descriptions-item label="模板名称">
					{{ template.name }}
				</a-descriptions-item>
				<a-descriptions-item label="描述">
					{{ template.description || '-' }}
				</a-descriptions-item>
				<a-descriptions-item label="类型">
					<a-tag :color="template.is_builtin ? 'blue' : 'default'">
						{{ template.is_builtin ? '内置模板' : '自定义模板' }}
					</a-tag>
				</a-descriptions-item>
				<a-descriptions-item label="创建时间">
					{{ formatTime(template.created_at) }}
				</a-descriptions-item>
				<a-descriptions-item label="更新时间">
					{{ formatTime(template.updated_at) }}
				</a-descriptions-item>
			</a-descriptions>
		</a-card>

		<a-card title="变量声明" :loading="loading">
			<a-table
				v-if="template && template.variable_declarations.length > 0"
				:columns="variableColumns"
				:data-source="template.variable_declarations"
				:pagination="false"
				row-key="name"
			>
				<template #bodyCell="{ column, record }">
					<template v-if="column.key === 'required'">
						<a-tag :color="record.required ? 'red' : 'default'">
							{{ record.required ? '必填' : '可选' }}
						</a-tag>
					</template>
					<template v-else-if="column.key === 'default'">
						<code v-if="record.default">{{ record.default }}</code>
						<span v-else>-</span>
					</template>
				</template>
			</a-table>
			<a-empty v-else description="无变量声明" />
		</a-card>

		<a-card title="Pipeline YAML" :loading="loading">
			<CodeEditor
				v-if="template"
				v-model:value="template.content"
				:style="{ height: '500px', border: '1px solid #d9d9d9', borderRadius: '4px' }"
				:theme="'vs-dark'"
				:language="'yaml'"
				:options="{
					readOnly: true,
					minimap: { enabled: false },
					fontSize: 14,
					automaticLayout: true,
				}"
			/>
		</a-card>

		<!-- 编辑模板弹窗 -->
		<a-modal
			v-model:open="showEditModal"
			title="编辑模板"
			width="800px"
			@ok="handleEditOk"
		>
			<a-form
				ref="formRef"
				:model="form"
				:rules="formRules"
				:label-col="{ span: 4 }"
				:wrapper-col="{ span: 19 }"
			>
				<a-form-item label="模板名称" name="name">
					<a-input v-model:value="form.name" />
				</a-form-item>
				<a-form-item label="描述">
					<a-textarea v-model:value="form.description" :rows="2" />
				</a-form-item>
				<a-form-item label="Pipeline YAML" name="content">
					<a-textarea
						v-model:value="form.content"
						:rows="12"
						style="font-family: 'Consolas', 'Monaco', monospace"
					/>
				</a-form-item>
			</a-form>
			<template #footer>
				<a-button @click="showEditModal = false">取消</a-button>
				<a-button type="primary" :loading="operating" @click="handleEditOk">保存</a-button>
			</template>
		</a-modal>
	</a-space>
</template>

<script setup lang="ts">
import type { FormInstance } from 'ant-design-vue';
import { message } from 'ant-design-vue';
import { CodeEditor } from 'monaco-editor-vue3';
import { onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { pipelineTemplateApi } from '@/api/ci';
import { formatTime } from '@/utils/time';
import { useStatusAsync } from '@/composables/useStatusAsync';
import type { PipelineTemplate } from '@/types/api';

const route = useRoute();
const router = useRouter();
const templateId = route.params.id as string;

const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const template = ref<PipelineTemplate>();
const showEditModal = ref(false);
const formRef = ref<FormInstance>();

const variableColumns = [
	{ title: '变量名', key: 'name', dataIndex: 'name', width: 200 },
	{ title: '描述', key: 'description', dataIndex: 'description', ellipsis: true },
	{ title: '必填', key: 'required', width: 100 },
	{ title: '默认值', key: 'default', width: 200 },
];

const form = reactive({
	name: '',
	description: '',
	content: '',
});

const formRules = {
	name: [{ required: true, message: '请输入模板名称' }],
	content: [{ required: true, message: '请输入 Pipeline YAML' }],
};

async function fetchTemplate() {
	try {
		await execute(async () => {
			const data = await pipelineTemplateApi.get(templateId);
			template.value = data;
			form.name = data.name;
			form.description = data.description || '';
			form.content = data.content;
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取模板信息失败');
		router.push('/ci/templates');
	}
}

async function handleEditOk() {
	try {
		await formRef.value?.validate();
	} catch {
		return;
	}

	try {
		await executeOp(async () => {
			await pipelineTemplateApi.update(templateId, {
				name: form.name,
				description: form.description || undefined,
				content: form.content,
			});
			message.success('更新成功');
			showEditModal.value = false;
			fetchTemplate();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '更新失败');
	}
}

onMounted(() => {
	fetchTemplate();
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
