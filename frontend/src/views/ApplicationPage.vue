<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>应用管理</h2>
			<a-space>
				<a-input-search
					v-model:value="searchText"
					placeholder="搜索应用名称"
					style="width: 200px"
					@search="handleSearch"
				/>
				<a-button-group>
					<a-button :type="viewMode === 'card' ? 'primary' : 'default'" @click="viewMode = 'card'">
						<template #icon><AppstoreOutlined /></template>
					</a-button>
					<a-button
						:type="viewMode === 'table' ? 'primary' : 'default'"
						@click="viewMode = 'table'"
					>
						<template #icon><UnorderedListOutlined /></template>
					</a-button>
				</a-button-group>
				<a-button type="primary" @click="showApplicationCreateModal">
					<template #icon><PlusOutlined /></template>
					新建应用
				</a-button>
			</a-space>
		</div>
		<!-- 卡片视图 -->
		<a-spin v-if="viewMode === 'card'" :spinning="loading">
			<a-row :gutter="[16, 16]">
				<a-col v-for="app in applications" :key="app.id" :xs="24" :sm="12" :lg="8" :xl="6">
					<a-card hoverable class="app-card">
						<template #title>
							<router-link :to="`/applications/${app.id}`" style="display: block">
								{{ app.name }}
							</router-link>
						</template>
						<template #extra>
							<a-space class="card-actions">
								<a-button
									size="small"
									type="primary"
									:disabled="app.status === 'started'"
									:loading="operating"
									@click.stop="handleDeploy(app)"
								>
									部署
								</a-button>
								<a-button
									size="small"
									:disabled="app.status !== 'started'"
									:loading="operating"
									@click.stop="handleStop(app)"
								>
									停止
								</a-button>
							</a-space>
						</template>
						<a-descriptions :column="1" size="small">
							<a-descriptions-item label="编码">{{ app.code }}</a-descriptions-item>
							<a-descriptions-item label="仓库">
								{{ app.git_source?.repository_url || '-' }}
							</a-descriptions-item>
							<a-descriptions-item label="状态">
								<a-tag :color="app.status === 'started' ? 'success' : 'default'">
									{{ app.status === 'started' ? '已启动' : '已停止' }}
								</a-tag>
							</a-descriptions-item>
							<a-descriptions-item label="镜像拉取策略">
								{{ app.image_pull_policy }}
							</a-descriptions-item>
							<a-descriptions-item label="部署记录">
								<a @click.stop="$router.push(`/deployments?application_id=${app.id}`)">
									查看部署记录
								</a>
							</a-descriptions-item>
						</a-descriptions>
					</a-card>
				</a-col>
			</a-row>
			<a-pagination
				v-model:current="pagination.current"
				v-model:page-size="pagination.pageSize"
				:total="pagination.total"
				:show-size-changer="true"
				:show-total="pagination.showTotal"
				style="margin-top: 16px; text-align: right"
				@change="fetchApplications"
			/>
		</a-spin>

		<!-- 表格视图 -->
		<a-table
			v-else
			:columns="columns"
			:data-source="applications"
			:loading="loading"
			:pagination="pagination"
			row-key="id"
			@change="handleTableChange"
		>
			<template #bodyCell="{ column, record }">
				<template v-if="column.key === 'name'">
					<router-link :to="`/applications/${record.id}`">
						{{ record.name }}
					</router-link>
				</template>
				<template v-else-if="column.key === 'repository'">
					{{ record.git_source?.repository_url || '-' }}
				</template>
				<template v-else-if="column.key === 'deploy_branches'">
					{{ record.git_source?.deploy_branches || '-' }}
				</template>
				<template v-else-if="column.key === 'status'">
					<a-tag :color="record.status === 'started' ? 'success' : 'default'">
						{{ record.status === 'started' ? '已启动' : '已停止' }}
					</a-tag>
				</template>
				<template v-else-if="column.key === 'auto_deploy'">
					<a-tag :color="record.git_source?.auto_deploy ? 'blue' : 'default'">
						{{ record.git_source?.auto_deploy ? '自动' : '手动' }}
					</a-tag>
				</template>
				<template v-else-if="column.key === 'actions'">
					<a-space>
						<a @click="$router.push(`/applications/${record.id}`)">查看</a>
						<a
							:class="{ disabled: record.status === 'started' || operating }"
							@click="handleDeploy(record)"
						>
							部署
						</a>
						<a
							:class="{ disabled: record.status !== 'started' || operating }"
							@click="handleStop(record)"
						>
							停止
						</a>
					</a-space>
				</template>
			</template>
		</a-table>

		<!-- 创建应用弹窗 -->
		<a-modal v-model:open="showApplicationCreate" title="新建应用">
			<a-form
				ref="formRef"
				:model="form"
				:rules="formRules"
				:label-col="{ span: 6 }"
				:wrapper-col="{ span: 16 }"
			>
				<a-form-item label="应用名称" name="name">
					<a-input v-model:value="form.name" />
				</a-form-item>
				<a-form-item label="应用编码" name="code">
					<a-input v-model:value="form.code" placeholder="小写字母、数字和连字符" />
				</a-form-item>
				<a-form-item label="仓库地址" name="repository_url">
					<a-input v-model:value="form.repository_url" />
				</a-form-item>
				<a-form-item label="部署分支" name="deploy_branches">
					<a-input v-model:value="form.deploy_branches" placeholder="master,develop" />
				</a-form-item>
				<a-form-item label="自动部署" name="auto_deploy">
					<a-switch v-model:checked="form.auto_deploy" />
				</a-form-item>
				<a-form-item label="镜像拉取策略" name="image_pull_policy">
					<a-select v-model:value="form.image_pull_policy">
						<a-select-option value="always">always</a-select-option>
						<a-select-option value="missing">missing</a-select-option>
						<a-select-option value="never">never</a-select-option>
					</a-select>
				</a-form-item>
				<a-form-item label="启用" name="enabled">
					<a-switch v-model:checked="form.enabled" />
				</a-form-item>
			</a-form>
			<template #footer>
				<a-button type="primary" :loading="operating" @click="handleModalOk">保存</a-button>
				<a-button @click="showApplicationCreate = false">取消</a-button>
			</template>
		</a-modal>
	</a-space>
</template>

<script setup lang="ts">
import { AppstoreOutlined, PlusOutlined, UnorderedListOutlined } from '@ant-design/icons-vue';
import type { FormInstance } from 'ant-design-vue';
import { message } from 'ant-design-vue';
import { onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { applicationApi } from '@/api/application';
import { deploymentApi } from '@/api/deployments';
import { useStatusAsync } from '@/composables/useStatusAsync';
import type { Application } from '@/types/api';

const $router = useRouter();

const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const applications = ref<Application[]>([]);
const searchText = ref('');
const viewMode = ref<'card' | 'table'>('card');
const pagination = reactive({
	current: 1,
	pageSize: 10,
	total: 0,
	showSizeChanger: true,
	showTotal: (total: number) => `共 ${total} 条`,
});

const columns = [
	{ title: '应用名称', key: 'name', dataIndex: 'name' },
	{ title: '应用编码', key: 'code', dataIndex: 'code' },
	{ title: '仓库', key: 'repository' },
	{ title: '部署分支', key: 'deploy_branches' },
	{ title: '状态', key: 'status' },
	{ title: '自动部署', key: 'auto_deploy' },
	{ title: '操作', key: 'actions', width: 150 },
];

// 弹窗相关
const showApplicationCreate = ref(false);
const formRef = ref<FormInstance>();

const form = reactive({
	name: '',
	code: '',
	repository_url: '',
	deploy_branches: 'master',
	auto_deploy: false,
	image_pull_policy: 'missing',
	enabled: true,
});

const formRules = {
	name: [{ required: true, message: '请输入应用名称' }],
	code: [
		{ required: true, message: '请输入应用编码' },
		{ pattern: /^[a-z][a-z0-9-]*$/, message: '必须以小写字母开头，只能包含小写字母、数字和连字符' },
	],
};

async function pollDeployment(deploymentId: string, app: Application) {
	while (true) {
		await new Promise((resolve) => setTimeout(resolve, 3000));
		try {
			const data = await deploymentApi.get(deploymentId);
			if (['success', 'failed'].includes(data.status)) {
				const target = applications.value.find((a) => a.id === app.id);
				if (data.status === 'success') {
					if (target) target.status = 'started';
					message.success(`${app.name} 部署成功`);
				} else {
					message.error(`${app.name} 部署失败`);
				}
				break;
			}
		} catch {
			break;
		}
	}
}

async function handleDeploy(app: Application) {
	try {
		await executeOp(async () => {
			const { deployment_id } = await applicationApi.deploy(app.id);
			message.success(`${app.name} 部署已触发`);
			pollDeployment(deployment_id, app);
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '部署失败');
	}
}

async function handleStop(app: Application) {
	try {
		await executeOp(async () => {
			await applicationApi.stop(app.id);
			app.status = 'stopped';
			message.success(`${app.name} 已停止`);
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '停止失败');
	}
}

async function fetchApplications() {
	try {
		await execute(async () => {
			const res = await applicationApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				search: searchText.value || undefined,
			});
			applications.value = res.items;
			pagination.total = res.total;
		});
	} catch (error) {
		message.error('获取应用列表失败\n' + error);
	}
}

function handleSearch() {
	pagination.current = 1;
	fetchApplications();
}

function handleTableChange(pag: { current?: number; pageSize?: number }) {
	pagination.current = pag.current || 1;
	pagination.pageSize = pag.pageSize || 10;
	fetchApplications();
}

function showApplicationCreateModal() {
	Object.assign(form, {
		name: '',
		code: '',
		repository_url: '',
		deploy_branches: 'master',
		auto_deploy: false,
		image_pull_policy: 'missing',
		enabled: true,
	});
	showApplicationCreate.value = true;
}

async function handleModalOk() {
	try {
		await formRef.value?.validate();
	} catch {
		return;
	}

	try {
		await executeOp(async () => {
			const payload = {
				name: form.name,
				code: form.code,
				image_pull_policy: form.image_pull_policy,
				enabled: form.enabled,
				git_source: form.repository_url
					? {
							repository_url: form.repository_url,
							deploy_branches: form.deploy_branches,
							auto_deploy: form.auto_deploy,
						}
					: null,
			};

			await applicationApi.create(payload);
			message.success('创建成功');
			showApplicationCreate.value = false;
			fetchApplications();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '操作失败');
	}
}

onMounted(() => {
	fetchApplications();
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

a.disabled {
	color: rgba(0, 0, 0, 0.25);
	cursor: not-allowed;
}

.app-card .card-actions {
	opacity: 0;
	transition: opacity 0.2s;
}

.app-card:hover .card-actions {
	opacity: 1;
}
</style>
