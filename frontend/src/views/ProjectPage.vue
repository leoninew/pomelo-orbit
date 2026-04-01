<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>CI 项目</h2>
			<a-button type="primary" @click="showCreateModal">
				<template #icon><PlusOutlined /></template>
				新建项目
			</a-button>
		</div>

		<a-table
			:columns="columns"
			:data-source="projects"
			:loading="loading"
			:pagination="pagination"
			row-key="id"
			@change="handleTableChange"
		>
			<template #bodyCell="{ column, record }">
				<template v-if="column.key === 'name'">
					<router-link :to="`/ci/projects/${record.id}`">
						{{ record.name }}
					</router-link>
				</template>
				<template v-else-if="column.key === 'repository_url'">
					<a-typography-text :ellipsis="{ tooltip: record.repository_url }" style="max-width: 300px">
						{{ record.repository_url }}
					</a-typography-text>
				</template>
				<template v-else-if="column.key === 'created_at'">
					{{ formatTime(record.created_at) }}
				</template>
				<template v-else-if="column.key === 'actions'">
					<a-space>
						<a @click="$router.push(`/ci/projects/${record.id}`)">查看</a>
						<a @click="handleTrigger(record.id)">触发</a>
						<a-popconfirm
							title="确定删除此项目？"
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

		<!-- 创建项目弹窗 -->
		<a-modal
			v-model:open="showModal"
			title="新建项目"
			width="600px"
			@ok="handleModalOk"
		>
			<a-form
				ref="formRef"
				:model="form"
				:rules="formRules"
				:label-col="{ span: 6 }"
				:wrapper-col="{ span: 17 }"
			>
				<a-form-item label="项目名称" name="name">
					<a-input v-model:value="form.name" placeholder="例如: my-backend" />
				</a-form-item>
				<a-form-item label="仓库地址" name="repository_url">
					<a-input
						v-model:value="form.repository_url"
						placeholder="git@github.com:user/repo.git"
					/>
				</a-form-item>
				<a-form-item label="Pipeline 模板" name="pipeline_template_id">
					<a-select
						v-model:value="form.pipeline_template_id"
						placeholder="选择模板"
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
						placeholder="选择凭据（可选）"
						:loading="credentialsLoading"
						allow-clear
					>
						<a-select-option v-for="cred in gitCredentials" :key="cred.id" :value="cred.id">
							{{ cred.name }}
						</a-select-option>
					</a-select>
				</a-form-item>
				<a-form-item label="分支过滤">
					<a-input
						v-model:value="form.branch_filter"
						placeholder="例如: main,develop (留空表示所有分支)"
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
import { computed, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { ciCredentialApi, pipelineTemplateApi, projectApi } from '@/api/ci';
import { formatTime } from '@/utils/time';
import { useStatusAsync } from '@/composables/useStatusAsync';
import type { Credential, PipelineTemplate, Project } from '@/types/api';

const router = useRouter();

const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const { loading: templatesLoading, execute: executeTemplates } = useStatusAsync();
const { loading: credentialsLoading, execute: executeCredentials } = useStatusAsync();

const projects = ref<Project[]>([]);
const templates = ref<PipelineTemplate[]>([]);
const credentials = ref<Credential[]>([]);

const gitCredentials = computed(() =>
	credentials.value.filter((c) => c.type === 'git_ssh' || c.type === 'git_token')
);

const pagination = reactive({
	current: 1,
	pageSize: 10,
	total: 0,
	showSizeChanger: true,
	showTotal: (total: number) => `共 ${total} 条`,
});

const columns = [
	{ title: '项目名称', key: 'name', dataIndex: 'name', width: 200 },
	{ title: '仓库地址', key: 'repository_url' },
	{ title: '创建时间', key: 'created_at', width: 180 },
	{ title: '操作', key: 'actions', width: 180 },
];

const showModal = ref(false);
const formRef = ref<FormInstance>();

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

async function fetchProjects() {
	try {
		await execute(async () => {
			const res = await projectApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
			});
			projects.value = res.items;
			pagination.total = res.total;
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取项目列表失败');
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

function handleTableChange(pag: { current?: number; pageSize?: number }) {
	pagination.current = pag.current || 1;
	pagination.pageSize = pag.pageSize || 10;
	fetchProjects();
}

function showCreateModal() {
	Object.assign(form, {
		name: '',
		repository_url: '',
		pipeline_template_id: '',
		git_credential_id: undefined,
		branch_filter: '',
	});
	showModal.value = true;
	fetchTemplates();
	fetchCredentials();
}

async function handleModalOk() {
	try {
		await formRef.value?.validate();
	} catch {
		return;
	}

	try {
		await executeOp(async () => {
			await projectApi.create({
				name: form.name,
				repository_url: form.repository_url,
				pipeline_template_id: form.pipeline_template_id,
				git_credential_id: form.git_credential_id,
				branch_filter: form.branch_filter || undefined,
			});
			message.success('创建成功');
			showModal.value = false;
			fetchProjects();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '创建失败');
	}
}

async function handleTrigger(id: string) {
	try {
		await executeOp(async () => {
			const run = await projectApi.trigger(id);
			message.success(`触发成功，Run ID: ${run.id}`);
			router.push(`/ci/runs/${run.id}`);
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '触发失败');
	}
}

async function handleDelete(id: string) {
	try {
		await executeOp(async () => {
			await projectApi.delete(id);
			message.success('删除成功');
			fetchProjects();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '删除失败');
	}
}

onMounted(() => {
	fetchProjects();
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
