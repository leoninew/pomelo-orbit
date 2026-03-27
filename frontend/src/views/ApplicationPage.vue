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
				<a-button @click="triggerImport">
					<template #icon><UploadOutlined /></template>
					导入
				</a-button>
				<input
					ref="fileInput"
					type="file"
					accept=".json"
					style="display: none"
					@change="handleFileImport"
				/>
			</a-space>
		</div>
		<!-- 卡片视图 -->
		<a-spin v-if="viewMode === 'card'" :spinning="loading">
			<a-row :gutter="[16, 16]">
				<a-col v-for="app in applications" :key="app.id" :xs="24" :sm="12" :lg="8" :xl="6">
					<a-card hoverable :class="['app-card', `app-card--${app.status}`]">
						<template #title>
							<router-link :to="`/applications/${app.id}`" style="display: block">
								{{ app.name }}
							</router-link>
						</template>
						<template #extra>
							<a-tooltip :title="cardActionTooltip(app.status)">
								<a-button
									type="text"
									size="small"
									:disabled="app.status === 'deploying'"
									@click.stop="handleCardAction(app)"
								>
									<template #icon>
										<LoadingOutlined
											v-if="app.status === 'deploying'"
											class="spin"
										/>
										<CloseCircleOutlined
											v-else-if="app.status === 'deployed'"
										/>
										<ReloadOutlined
											v-else-if="app.status === 'deploy_failed'"
											style="color: #fa8c16"
										/>
										<PlayCircleOutlined v-else />
									</template>
								</a-button>
							</a-tooltip>
						</template>
						<a-descriptions :column="1" size="small">
							<a-descriptions-item label="编码">{{ app.code }}</a-descriptions-item>
							<a-descriptions-item label="仓库">
								{{ app.git_source?.repository_url || '-' }}
							</a-descriptions-item>
							<a-descriptions-item label="状态">
								<a-tag :color="appStatusColor(app.status)">
									{{ appStatusLabel(app.status) }}
								</a-tag>
							</a-descriptions-item>
							<a-descriptions-item label="镜像拉取策略">
								{{ app.image_pull_policy }}
							</a-descriptions-item>
							<a-descriptions-item label="部署记录">
								<a @click.stop="viewLastDeployment(app.id)">查看部署记录</a>
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
					<a-tag :color="appStatusColor(record.status)">
						{{ appStatusLabel(record.status) }}
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
							:class="{
								disabled:
									record.status === 'deployed' || record.status === 'deploying' || operating,
							}"
							@click="handleDeploy(record)"
						>
							部署
						</a>
						<a
							:class="{ disabled: record.status !== 'deployed' || operating }"
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

		<!-- 导入应用弹窗 -->
		<a-modal v-model:open="showImportPreview" title="导入应用" width="600px">
			<a-form
				ref="importFormRef"
				:model="importForm"
				:rules="importFormRules"
				:label-col="{ span: 6 }"
				:wrapper-col="{ span: 16 }"
			>
				<a-form-item label="应用名称" name="name">
					<a-input v-model:value="importForm.name" />
				</a-form-item>
				<a-form-item label="应用编码" name="code">
					<a-input v-model:value="importForm.code" placeholder="小写字母、数字和连字符" />
				</a-form-item>
				<a-form-item label="仓库地址" name="repository_url">
					<a-input v-model:value="importForm.repository_url" />
				</a-form-item>
				<a-form-item label="部署分支" name="deploy_branches">
					<a-input v-model:value="importForm.deploy_branches" placeholder="master,develop" />
				</a-form-item>
				<a-form-item label="自动部署" name="auto_deploy">
					<a-switch v-model:checked="importForm.auto_deploy" />
				</a-form-item>
				<a-form-item label="镜像拉取策略" name="image_pull_policy">
					<a-select v-model:value="importForm.image_pull_policy">
						<a-select-option value="always">always</a-select-option>
						<a-select-option value="missing">missing</a-select-option>
						<a-select-option value="never">never</a-select-option>
					</a-select>
				</a-form-item>
				<a-form-item label="启用" name="enabled">
					<a-switch v-model:checked="importForm.enabled" />
				</a-form-item>
				<a-form-item label="配置文件">
					<div v-if="importForm.config_files.length > 0" class="config-files-list">
						<div
							v-for="(file, index) in importForm.config_files"
							:key="index"
							class="config-file-item"
						>
							<span>{{ file.path }}</span>
						</div>
					</div>
					<div v-else class="config-files-empty">无配置文件</div>
				</a-form-item>
			</a-form>
			<template #footer>
				<a-button type="primary" :loading="operating" @click="handleImportOk">导入</a-button>
				<a-button @click="showImportPreview = false">取消</a-button>
			</template>
		</a-modal>
	</a-space>
</template>

<script setup lang="ts">
import type { FormInstance } from 'ant-design-vue';
import { message } from 'ant-design-vue';
import { onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { applicationApi } from '@/api/application';
import { deploymentApi } from '@/api/deployments';
import { delayAsync } from '@/utils/time';
import { useStatusAsync } from '@/composables/useStatusAsync';
import type { Application, ApplicationImportReq } from '@/types/api';

const $router = useRouter();

const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const applications = ref<Application[]>([]);
const searchText = ref('');
const viewMode = ref<'card' | 'table'>('card');
const fileInput = ref<HTMLInputElement>();
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
const showImportPreview = ref(false);
const formRef = ref<FormInstance>();
const importFormRef = ref<FormInstance>();

const form = reactive({
	name: '',
	code: '',
	repository_url: '',
	deploy_branches: 'master',
	auto_deploy: false,
	image_pull_policy: 'missing',
	enabled: true,
});

const importForm = reactive({
	name: '',
	code: '',
	repository_url: '',
	deploy_branches: '',
	auto_deploy: false,
	image_pull_policy: 'missing',
	enabled: true,
	config_files: [] as { path: string; content: string }[],
});

const formRules = {
	name: [{ required: true, message: '请输入应用名称' }],
	code: [
		{ required: true, message: '请输入应用编码' },
		{ pattern: /^[a-z][a-z0-9-]*$/, message: '必须以小写字母开头，只能包含小写字母、数字和连字符' },
	],
};

const importFormRules = {
	name: [{ required: true, message: '请输入应用名称' }],
	code: [
		{ required: true, message: '请输入应用编码' },
		{ pattern: /^[a-z][a-z0-9-]*$/, message: '必须以小写字母开头，只能包含小写字母、数字和连字符' },
	],
};

async function pollDeployment(deploymentId: string, app: Application) {
	while (true) {
		await delayAsync(3000);
		try {
			const data = await deploymentApi.get(deploymentId);
			if (['ran_to_completion', 'faulted', 'canceled'].includes(data.status)) {
				const target = applications.value.find((a) => a.id === app.id);
				if (data.status === 'ran_to_completion') {
					if (target) target.status = 'deployed';
					message.success(`${app.name} 部署成功`);
				} else {
					if (target) target.status = 'deploy_failed';
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
			// 立即反映部署中状态
			const target = applications.value.find((a) => a.id === app.id);
			if (target) target.status = 'deploying';
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
			app.status = 'undeployed';
			message.success(`${app.name} 已停止`);
		});
	} catch (error) {
		app.status = 'deploy_failed';
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

async function viewLastDeployment(appId: string) {
	const resp = await deploymentApi.list({ application_id: appId, per_page: 1 });
	if (resp.items.length > 0) {
		$router.push(`/deployments/${resp.items[0].id}`);
	} else {
		$router.push(`/deployments?application_id=${appId}`);
	}
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

function triggerImport() {
	fileInput.value?.click();
}

async function handleFileImport(event: Event) {
	const target = event.target as HTMLInputElement;
	const file = target.files?.[0];
	if (!file) return;

	try {
		const text = await file.text();
		const data = JSON.parse(text) as ApplicationImportReq;

		// 填充导入表单
		Object.assign(importForm, {
			name: data.name || '',
			code: data.code || '',
			repository_url: data.git_source?.repository_url || '',
			deploy_branches: data.git_source?.deploy_branches || 'master',
			auto_deploy: data.git_source?.auto_deploy ?? false,
			image_pull_policy: data.image_pull_policy || 'missing',
			enabled: data.enabled ?? true,
			config_files: data.config_files || [],
		});

		showImportPreview.value = true;
	} catch (error) {
		message.error('解析文件失败：' + (error instanceof Error ? error.message : '未知错误'));
	} finally {
		target.value = '';
	}
}

async function handleImportOk() {
	try {
		await importFormRef.value?.validate();
	} catch {
		return;
	}

	try {
		await executeOp(async () => {
			const payload = {
				name: importForm.name,
				code: importForm.code,
				image_pull_policy: importForm.image_pull_policy,
				enabled: importForm.enabled,
				git_source: importForm.repository_url
					? {
							repository_url: importForm.repository_url,
							deploy_branches: importForm.deploy_branches,
							auto_deploy: importForm.auto_deploy,
						}
					: null,
				config_files: importForm.config_files,
			};

			await applicationApi.importApplication(payload);
			message.success('导入成功');
			showImportPreview.value = false;
			fetchApplications();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '导入失败');
	}
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

function cardActionTooltip(status: string) {
	if (status === 'deploying') return '部署中...';
	if (status === 'deployed') return '停止';
	if (status === 'deploy_failed') return '重新部署';
	return '部署';
}

function handleCardAction(app: Application) {
	if (app.status === 'deployed') handleStop(app);
	else handleDeploy(app);
}

function appStatusColor(status: string) {
	if (status === 'deployed') return 'success';
	if (status === 'deploy_failed') return 'error';
	if (status === 'deploying') return 'processing';
	return 'default'; // undeployed
}

function appStatusLabel(status: string) {
	if (status === 'deployed') return '运行中';
	if (status === 'deploy_failed') return '部署失败';
	if (status === 'deploying') return '部署中';
	return '未部署'; // undeployed
}
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

.app-card--deployed {
	border-color: #52c41a !important;
}

.app-card--deploy_failed {
	border-color: #ff4d4f !important;
}

.config-files-list {
	max-height: 150px;
	overflow-y: auto;
}

.config-file-item {
	padding: 4px 0;
	border-bottom: 1px solid #f0f0f0;
}

.config-file-item:last-child {
	border-bottom: none;
}

.config-files-empty {
	color: #999;
	font-style: italic;
}

.spin {
	animation: spin 1s linear infinite;
}

@keyframes spin {
	from {
		transform: rotate(0deg);
	}
	to {
		transform: rotate(360deg);
	}
}
</style>
