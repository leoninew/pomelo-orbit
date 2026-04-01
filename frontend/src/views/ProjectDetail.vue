<template>
	<a-space direction="vertical" style="width: 100%">
		<div v-if="project" class="page-header">
			<h2>{{ project.name }}</h2>
			<a-space>
				<a-button @click="$router.push('/ci/projects')">返回</a-button>
				<a-button type="primary" @click="handleTrigger">手动触发</a-button>
				<a-button @click="showEditModal = true">编辑</a-button>
			</a-space>
		</div>

		<!-- 基本信息卡片 -->
		<a-card title="基本信息" :loading="loading">
			<a-descriptions v-if="project" :column="2" bordered size="small">
				<a-descriptions-item label="项目名称">
					{{ project.name }}
				</a-descriptions-item>
				<a-descriptions-item label="仓库地址">
					{{ project.repository_url }}
				</a-descriptions-item>
				<a-descriptions-item label="Pipeline 模板">
					<router-link :to="`/ci/templates/${project.pipeline_template_id}`">
						查看模板
					</router-link>
				</a-descriptions-item>
				<a-descriptions-item label="Git 凭据">
					{{ project.git_credential_id ? '已配置' : '未配置' }}
				</a-descriptions-item>
				<a-descriptions-item label="分支过滤">
					{{ project.branch_filter || '所有分支' }}
				</a-descriptions-item>
				<a-descriptions-item label="创建时间">
					{{ formatTime(project.created_at) }}
				</a-descriptions-item>
			</a-descriptions>
		</a-card>

		<!-- Webhook 配置卡片 -->
		<a-card title="Webhook 配置" :loading="loading">
			<a-descriptions v-if="project" :column="1" bordered size="small">
				<a-descriptions-item label="Webhook URL">
					<a-typography-text copyable>
						{{ webhookUrl }}
					</a-typography-text>
				</a-descriptions-item>
				<a-descriptions-item label="Webhook Secret">
					<a-space>
						<a-typography-text v-if="showSecret" copyable>
							{{ project.webhook_secret }}
						</a-typography-text>
						<a-typography-text v-else>
							••••••••••••••••
						</a-typography-text>
						<a-button size="small" @click="showSecret = !showSecret">
							{{ showSecret ? '隐藏' : '显示' }}
						</a-button>
					</a-space>
				</a-descriptions-item>
			</a-descriptions>
		</a-card>

		<!-- 变量配置卡片 -->
		<a-card title="变量配置" :loading="loading">
			<template #extra>
				<a-button size="small" @click="showVariableModal = true">编辑变量</a-button>
			</template>
			<a-table
				v-if="project && Object.keys(project.variable_overrides).length > 0"
				:columns="variableColumns"
				:data-source="variableList"
				:pagination="false"
				row-key="key"
			>
				<template #bodyCell="{ column, record }">
					<template v-if="column.key === 'value'">
						<code>{{ record.value }}</code>
					</template>
				</template>
			</a-table>
			<a-empty v-else description="未配置变量" />
		</a-card>

		<!-- Pipeline Runs 卡片 -->
		<a-card title="Pipeline Runs" :loading="runsLoading">
			<template #extra>
				<a @click="$router.push(`/ci/runs?project_id=${project?.id}`)">查看全部</a>
			</template>
			<a-table
				v-if="runs.length > 0"
				:columns="runColumns"
				:data-source="runs"
				:pagination="false"
				row-key="id"
			>
				<template #bodyCell="{ column, record }">
					<template v-if="column.key === 'id'">
						<router-link :to="`/ci/runs/${record.id}`">
							{{ record.id.substring(0, 8) }}
						</router-link>
					</template>
					<template v-else-if="column.key === 'trigger'">
						<a-tag>{{ record.trigger }}</a-tag>
					</template>
					<template v-else-if="column.key === 'status'">
						<a-tag :color="pipelineRunStatusColors[record.status]">
							{{ record.status }}
						</a-tag>
					</template>
					<template v-else-if="column.key === 'created_at'">
						{{ formatTime(record.created_at) }}
					</template>
				</template>
			</a-table>
			<a-empty v-else description="暂无运行记录" />
		</a-card>

		<!-- 编辑项目弹窗 -->
		<a-modal
			v-model:open="showEditModal"
			title="编辑项目"
			width="600px"
			@ok="handleEditOk"
		>
			<a-form
				ref="formRef"
				:model="form"
				:rules="formRules"
				:label-col="{ span: 6 }"
				:wrapper-col="{ span: 17 }"
			>
				<a-form-item label="项目名称" name="name">
					<a-input v-model:value="form.name" />
				</a-form-item>
				<a-form-item label="仓库地址" name="repository_url">
					<a-input v-model:value="form.repository_url" />
				</a-form-item>
				<a-form-item label="Pipeline 模板" name="pipeline_template_id">
					<a-select
						v-model:value="form.pipeline_template_id"
						:loading="templatesLoading"
					>
						<a-select-option v-for="tpl in templates" :key="tpl.id" :value="tpl.id">
							{{ tpl.name }}
						</a-select-option>
					</a-select>
				</a-form-item>
				<a-form-item label="Git 凭据">
					<a-select
						v-model:value="form.git_credential_id"
						:loading="credentialsLoading"
						allow-clear
					>
						<a-select-option v-for="cred in gitCredentials" :key="cred.id" :value="cred.id">
							{{ cred.name }}
						</a-select-option>
					</a-select>
				</a-form-item>
				<a-form-item label="分支过滤">
					<a-input v-model:value="form.branch_filter" />
				</a-form-item>
			</a-form>
			<template #footer>
				<a-button @click="showEditModal = false">取消</a-button>
				<a-button type="primary" :loading="operating" @click="handleEditOk">保存</a-button>
			</template>
		</a-modal>

		<!-- 编辑变量弹窗 -->
		<a-modal
			v-model:open="showVariableModal"
			title="编辑变量"
			width="600px"
			@ok="handleVariableOk"
		>
			<a-form layout="vertical">
				<a-form-item
					v-for="(value, key) in variableForm"
					:key="key"
					:label="key"
				>
					<a-input v-model:value="variableForm[key]" />
				</a-form-item>
				<a-form-item label="添加新变量">
					<a-space>
						<a-input
							v-model:value="newVariableKey"
							placeholder="变量名"
							style="width: 200px"
						/>
						<a-input
							v-model:value="newVariableValue"
							placeholder="变量值"
							style="width: 200px"
						/>
						<a-button @click="addVariable">添加</a-button>
					</a-space>
				</a-form-item>
			</a-form>
			<template #footer>
				<a-button @click="showVariableModal = false">取消</a-button>
				<a-button type="primary" :loading="operating" @click="handleVariableOk">保存</a-button>
			</template>
		</a-modal>
	</a-space>
</template>

<script setup lang="ts">
import type { FormInstance } from 'ant-design-vue';
import { message } from 'ant-design-vue';
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ciCredentialApi, pipelineRunApi, pipelineTemplateApi, projectApi } from '@/api/ci';
import { formatTime } from '@/utils/time';
import { useStatusAsync } from '@/composables/useStatusAsync';
import type { Credential, PipelineRun, PipelineTemplate, Project } from '@/types/api';
import { pipelineRunStatusColors } from '@/types/api';

const route = useRoute();
const router = useRouter();
const projectId = route.params.id as string;

const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const { loading: runsLoading, execute: executeRuns } = useStatusAsync();
const { loading: templatesLoading, execute: executeTemplates } = useStatusAsync();
const { loading: credentialsLoading, execute: executeCredentials } = useStatusAsync();

const project = ref<Project>();
const runs = ref<PipelineRun[]>([]);
const templates = ref<PipelineTemplate[]>([]);
const credentials = ref<Credential[]>([]);

const gitCredentials = computed(() =>
	credentials.value.filter((c) => c.type === 'git_ssh' || c.type === 'git_token')
);

const showSecret = ref(false);
const showEditModal = ref(false);
const showVariableModal = ref(false);
const formRef = ref<FormInstance>();

const webhookUrl = computed(() => {
	if (!project.value) return '';
	const baseUrl = window.location.origin;
	return `${baseUrl}/api/v1/ci/webhooks/git`;
});

const variableList = computed(() => {
	if (!project.value) return [];
	return Object.entries(project.value.variable_overrides).map(([key, value]) => ({
		key,
		value,
	}));
});

const variableColumns = [
	{ title: '变量名', key: 'key', dataIndex: 'key' },
	{ title: '变量值', key: 'value', dataIndex: 'value' },
];

const runColumns = [
	{ title: 'Run ID', key: 'id', width: 120 },
	{ title: '触发方式', key: 'trigger', width: 100 },
	{ title: 'Ref', key: 'trigger_ref', dataIndex: 'trigger_ref', width: 150 },
	{ title: '状态', key: 'status', width: 100 },
	{ title: '创建时间', key: 'created_at', width: 180 },
];

const form = reactive({
	name: '',
	repository_url: '',
	pipeline_template_id: '',
	git_credential_id: undefined as string | undefined,
	branch_filter: '',
});

const formRules = {
	name: [{ required: true, message: '请输入项目名称' }],
	repository_url: [{ required: true, message: '请输入仓库地址' }],
	pipeline_template_id: [{ required: true, message: '请选择 Pipeline 模板' }],
};

const variableForm = reactive<Record<string, string>>({});
const newVariableKey = ref('');
const newVariableValue = ref('');

async function fetchProject() {
	try {
		await execute(async () => {
			const data = await projectApi.get(projectId);
			project.value = data;
			form.name = data.name;
			form.repository_url = data.repository_url;
			form.pipeline_template_id = data.pipeline_template_id;
			form.git_credential_id = data.git_credential_id;
			form.branch_filter = data.branch_filter || '';
			Object.assign(variableForm, data.variable_overrides);
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取项目信息失败');
		router.push('/ci/projects');
	}
}

async function fetchRuns() {
	try {
		await executeRuns(async () => {
			const res = await pipelineRunApi.list({
				project_id: projectId,
				per_page: 10,
			});
			runs.value = res.items;
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取运行记录失败');
	}
}

async function fetchTemplates() {
	try {
		await executeTemplates(async () => {
			const res = await pipelineTemplateApi.list({ per_page: 100 });
			templates.value = res.items;
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取模板列表失败');
	}
}

async function fetchCredentials() {
	try {
		await executeCredentials(async () => {
			const res = await ciCredentialApi.list({ per_page: 100 });
			credentials.value = res.items;
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取凭据列表失败');
	}
}

async function handleTrigger() {
	try {
		await executeOp(async () => {
			const run = await projectApi.trigger(projectId);
			message.success(`触发成功，Run ID: ${run.id}`);
			router.push(`/ci/runs/${run.id}`);
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '触发失败');
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
			await projectApi.update(projectId, {
				name: form.name,
				repository_url: form.repository_url,
				pipeline_template_id: form.pipeline_template_id,
				git_credential_id: form.git_credential_id,
				branch_filter: form.branch_filter || undefined,
			});
			message.success('更新成功');
			showEditModal.value = false;
			fetchProject();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '更新失败');
	}
}

function addVariable() {
	if (!newVariableKey.value.trim()) {
		message.error('请输入变量名');
		return;
	}
	if (variableForm[newVariableKey.value]) {
		message.warning('变量名已存在，将覆盖原值');
	}
	variableForm[newVariableKey.value] = newVariableValue.value;
	newVariableKey.value = '';
	newVariableValue.value = '';
}

async function handleVariableOk() {
	try {
		await executeOp(async () => {
			await projectApi.update(projectId, {
				variable_overrides: variableForm,
			});
			message.success('更新成功');
			showVariableModal.value = false;
			fetchProject();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '更新失败');
	}
}

onMounted(() => {
	fetchProject();
	fetchRuns();
	fetchTemplates();
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
