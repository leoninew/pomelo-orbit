<template>
	<a-space direction="vertical" style="width: 100%">
		<div v-if="application" class="page-header">
			<h2>{{ application.name }}</h2>
			<a-button @click="() => $router.push('/applications')">返回</a-button>
		</div>

		<!-- 基本信息卡片 -->
		<a-card title="基本信息" :loading="basicInfoLoading">
			<template v-if="application" #extra>
				<a-space>
					<a-dropdown v-if="envs.length > 1">
						<template #overlay>
							<a-menu @click="handleDeployWithEnv">
								<a-menu-item v-for="env in envs.filter((e) => e !== '.env')" :key="env">
									{{ env }}
								</a-menu-item>
							</a-menu>
						</template>
						<a-button type="primary" @click="handleDeploy">
							部署
							<DownOutlined />
						</a-button>
					</a-dropdown>
					<a-button v-else type="primary" @click="handleDeploy">部署</a-button>
					<a-button :loading="operating" @click="handleStop">停止</a-button>
					<a-button :loading="operating" @click="handleRestart">重启</a-button>
					<a-button :disabled="operating" @click="showBasicInfoModal = true">编辑</a-button>
					<a-button
						danger
						:disabled="
							operating || application.status === 'deployed' || application.status === 'deploying'
						"
						@click="openDeleteModal"
					>
						删除
					</a-button>
					<a-button @click="handleExport">导出</a-button>
				</a-space>
			</template>
			<a-descriptions v-if="application" :column="2" bordered size="small">
				<a-descriptions-item label="应用编码">
					{{ application.code }}
				</a-descriptions-item>
				<a-descriptions-item label="仓库地址">
					<span v-if="application.git_source">{{ application.git_source.repository_url }}</span>
					<span v-else>-</span>
				</a-descriptions-item>
				<a-descriptions-item label="部署分支">
					<span v-if="application.git_source">{{ application.git_source.deploy_branches }}</span>
					<span v-else>-</span>
				</a-descriptions-item>
				<a-descriptions-item label="状态">
					<a-tag :color="appStatusColor(application.status)">
						{{ appStatusLabel(application.status) }}
					</a-tag>
				</a-descriptions-item>
				<a-descriptions-item label="自动部署">
					<span v-if="application.git_source">
						<a-tag :color="application.git_source.auto_deploy ? 'blue' : 'default'">
							{{ application.git_source.auto_deploy ? '是' : '否' }}
						</a-tag>
					</span>
					<span v-else>-</span>
				</a-descriptions-item>
				<a-descriptions-item label="镜像拉取策略">
					{{ application.image_pull_policy }}
				</a-descriptions-item>
				<a-descriptions-item label="创建时间">
					{{ formatTime(application.created_at) }}
				</a-descriptions-item>
				<a-descriptions-item label="部署记录">
					<a @click="$router.push(`/deployments?application_id=${application.id}`)">查看部署记录</a>
				</a-descriptions-item>
			</a-descriptions>
		</a-card>

		<!-- 配置文件卡片 -->
		<a-card title="配置文件" :loading="fileListLoading">
			<template #extra>
				<a-button type="primary" @click="openAddFileDrawer">添加文件</a-button>
			</template>
			<a-table
				v-if="files.length > 0"
				:columns="fileColumns"
				:data-source="files"
				:pagination="false"
				row-key="id"
			>
				<template #bodyCell="{ column, record }">
					<template v-if="column.key === 'path'">
						{{ record.path }}
					</template>
					<template v-if="column.key === 'created_at'">
						{{ formatTime(record.created_at) }}
					</template>
					<template v-if="column.key === 'action'">
						<a-space>
							<a @click="openFileDrawer(record.id)">查看</a>
							<a @click="openFileDrawer(record.id, true)">编辑</a>
							<a-popconfirm title="确定删除此文件？" @confirm="deleteFile(record.id)">
								<a style="color: #ff4d4f">删除</a>
							</a-popconfirm>
						</a-space>
					</template>
				</template>
			</a-table>
			<a-empty v-else description="暂无配置文件" />
		</a-card>

		<!-- 文件查看/编辑/新建抽屉 -->
		<a-drawer
			v-model:open="fileDrawerVisible"
			:title="currentFileId ? '文件: ' + (currentFilePath || '未命名') : '新建文件'"
			:width="720"
			:loading="fileContentLoading"
			:closable="false"
			@close="handleDrawerClose"
		>
			<div v-if="!isEditingInDrawer && currentFileId">
				<CodeEditor
					v-if="!fileContentLoading"
					v-model:value="currentFileContent"
					:style="{ height: 'calc(100vh - 200px)' }"
					:theme="'vs-dark'"
					:language="currentFileLanguage"
					:options="{ readOnly: true }"
				/>
				<a-empty v-else description="加载中..." />
			</div>
			<div v-if="isEditingInDrawer || !currentFileId">
				<a-form layout="vertical">
					<a-form-item label="文件路径">
						<a-input v-model:value="currentFilePath" placeholder="例如: nginx.conf" />
					</a-form-item>
					<a-form-item v-if="!fileContentLoading" label="文件内容">
						<CodeEditor
							v-model:value="currentFileContent"
							:style="{ height: '450px', border: '1px solid #d9d9d9', borderRadius: '4px' }"
							:theme="'vs-dark'"
							:language="currentFileLanguage"
							:options="{
								minimap: { enabled: false },
								fontSize: 14,
								automaticLayout: true,
							}"
						/>
					</a-form-item>
				</a-form>
			</div>
			<template #footer>
				<div style="display: flex; justify-content: flex-end">
					<a-space>
						<a-button
							v-if="!isEditingInDrawer && currentFileId"
							type="primary"
							@click="isEditingInDrawer = true"
						>
							编辑
						</a-button>
						<a-button v-if="!isEditingInDrawer && currentFileId" @click="handleDrawerClose">
							关闭
						</a-button>
						<a-button
							v-if="isEditingInDrawer || !currentFileId"
							type="primary"
							:loading="fileContentLoading"
							@click="saveCurrentFile"
						>
							保存
						</a-button>
						<a-button v-if="isEditingInDrawer || !currentFileId" @click="handleDrawerClose">
							取消
						</a-button>
					</a-space>
				</div>
			</template>
		</a-drawer>

		<!-- 基本信息编辑弹窗 -->
		<a-modal v-model:open="showBasicInfoModal" title="编辑基本信息">
			<a-form
				ref="basicInfoFormRef"
				:model="basicInfoForm"
				:rules="basicInfoFormRules"
				:label-col="{ span: 6 }"
				:wrapper-col="{ span: 16 }"
			>
				<a-form-item label="应用名称" name="name">
					<a-input v-model:value="basicInfoForm.name" />
				</a-form-item>
				<a-form-item label="应用编码">
					<a-input :value="basicInfoForm.code" disabled />
				</a-form-item>
				<a-form-item label="仓库地址">
					<a-input v-model:value="basicInfoForm.repository_url" />
				</a-form-item>
				<a-form-item label="部署分支">
					<a-input v-model:value="basicInfoForm.deploy_branches" />
				</a-form-item>
				<a-form-item label="自动部署">
					<a-switch v-model:checked="basicInfoForm.auto_deploy" />
				</a-form-item>
				<a-form-item label="镜像拉取策略" name="image_pull_policy">
					<a-select v-model:value="basicInfoForm.image_pull_policy">
						<a-select-option value="always">always</a-select-option>
						<a-select-option value="missing">missing</a-select-option>
						<a-select-option value="never">never</a-select-option>
					</a-select>
				</a-form-item>
				<a-form-item label="启用">
					<a-switch v-model:checked="basicInfoForm.enabled" />
				</a-form-item>
			</a-form>
			<template #footer>
				<a-button type="primary" :loading="operating" @click="handleBasicInfoOk">保存</a-button>
				<a-button @click="showBasicInfoModal = false">取消</a-button>
			</template>
		</a-modal>

		<!-- 删除应用弹窗 -->
		<a-modal v-model:open="showDeleteModal" title="删除应用">
			<a-space direction="vertical" style="width: 100%">
				<p>
					确定要删除应用「
					<strong>{{ application?.name }}</strong>
					」吗？
				</p>
				<a-checkbox v-model:checked="deleteDir">
					同时删除应用工作目录（data/apps/{{ application?.code }}）
				</a-checkbox>
			</a-space>
			<template #footer>
				<a-button type="primary" danger :loading="operating" @click="handleDeleteOk">删除</a-button>
				<a-button @click="showDeleteModal = false">取消</a-button>
			</template>
		</a-modal>
	</a-space>
</template>

<script setup lang="ts">
import type { FormInstance } from 'ant-design-vue';
import { message } from 'ant-design-vue';
import { formatTime, delayAsync } from '@/utils/time';
import { CodeEditor } from 'monaco-editor-vue3';
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { applicationApi } from '@/api/application';
import { deploymentApi } from '@/api/deployments';
import { useStatusAsync } from '@/composables/useStatusAsync';
import type { Application, ConfigFile } from '@/types/api';

const route = useRoute();
const router = useRouter();
const applicationId = route.params.id as string;

const { loading: basicInfoLoading, execute: executeBasicInfo } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const { loading: fileListLoading, execute: executeFileList } = useStatusAsync();
const { loading: fileContentLoading, execute: executeFileContent } = useStatusAsync();

const application = ref<Application>();
const envs = computed(() => {
	const envFiles = files.value.filter((f) => f.path.match(/^\.env(\..+)?$/));
	return envFiles.map((f) => f.path);
});

const files = ref<ConfigFile[]>([]);

const fileColumns = [
	{ title: '文件路径', key: 'path', dataIndex: 'path', width: 400 },
	{ title: '创建时间', key: 'created_at', width: 180 },
	{ title: '操作', key: 'action', width: 150 },
];

const fileDrawerVisible = ref(false);
const currentFileId = ref('');
const currentFilePath = ref('');
const currentFileContent = ref('');
const isEditingInDrawer = ref(false);

const currentFileLanguage = computed(() => {
	const path = currentFilePath.value.toLowerCase();
	if (path.endsWith('.sh') || path.endsWith('.bash')) return 'shell';
	if (path.startsWith('.env') || path.endsWith('.ini') || path.endsWith('.properties'))
		return 'ini';
	return 'yaml';
});

const showBasicInfoModal = ref(false);
const showDeleteModal = ref(false);
const deleteDir = ref(false);

const basicInfoFormRef = ref<FormInstance>();
const basicInfoForm = reactive({
	name: '',
	code: '',
	repository_url: '',
	deploy_branches: '',
	auto_deploy: false,
	image_pull_policy: 'missing',
	enabled: true,
});

const basicInfoFormRules = {
	name: [{ required: true, message: '请输入应用名称' }],
	image_pull_policy: [{ required: true, message: '请选择镜像拉取策略' }],
};

async function fetchApplication() {
	try {
		await executeBasicInfo(async () => {
			const data = await applicationApi.get(applicationId);
			application.value = data;
			basicInfoForm.name = data.name;
			basicInfoForm.code = data.code;
			basicInfoForm.image_pull_policy = data.image_pull_policy;
			basicInfoForm.enabled = data.enabled;
			if (data.git_source) {
				basicInfoForm.repository_url = data.git_source.repository_url;
				basicInfoForm.deploy_branches = data.git_source.deploy_branches;
				basicInfoForm.auto_deploy = data.git_source.auto_deploy;
			}
		});
		// 如果应用正在部署中，找到对应部署记录并轮询直到结束
		if (application.value?.status === 'deploying') {
			pollActiveDeployment();
		}
	} catch (error) {
		message.error(error instanceof Error ? error.message : '获取应用信息失败');
		router.push('/applications');
	}
}

async function pollActiveDeployment() {
	try {
		const resp = await deploymentApi.list({ application_id: applicationId, per_page: 1 });
		const latest = resp.items[0];
		if (!latest) return;
		while (true) {
			await delayAsync(3000);
			try {
				const detail = await deploymentApi.get(latest.id);
				if (['ran_to_completion', 'faulted', 'canceled'].includes(detail.status)) {
					if (application.value) {
						application.value.status =
							detail.status === 'ran_to_completion' ? 'deployed' : 'deploy_failed';
					}
					break;
				}
			} catch {
				break;
			}
		}
	} catch {
		// 查不到部署记录时静默退出
	}
}

async function handleDeploy() {
	try {
		await executeOp(async () => {
			const defaultEnv = envs.value.includes('.env') ? '.env' : undefined;
			const res = await applicationApi.deploy(applicationId, undefined, defaultEnv);
			message.success(`部署已触发，ID: ${res.deployment_id}`);
			router.push(`/deployments/${res.deployment_id}`);
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '触发部署失败');
	}
}

async function handleDeployWithEnv({ key }: { key: string }) {
	try {
		await executeOp(async () => {
			const res = await applicationApi.deploy(applicationId, undefined, key);
			message.success(`部署已触发，ID: ${res.deployment_id}`);
			router.push(`/deployments/${res.deployment_id}`);
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '触发部署失败');
	}
}

async function handleStop() {
	try {
		await executeOp(async () => {
			const res = await applicationApi.stop(applicationId);
			message.success('停止操作已提交');
			while (true) {
				await delayAsync(3000);
				try {
					const detail = await deploymentApi.get(res.deployment_id);
					if (['ran_to_completion', 'faulted', 'canceled'].includes(detail.status)) {
						if (detail.status === 'ran_to_completion') {
							if (application.value) application.value.status = 'undeployed';
						} else {
							if (application.value) application.value.status = 'deploy_failed';
							router.push(`/deployments/${res.deployment_id}`);
						}
						break;
					}
				} catch {
					break;
				}
			}
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '停止失败');
	}
}

async function handleRestart() {
	try {
		await executeOp(async () => {
			const res = await applicationApi.restart(applicationId);
			message.success('重启操作已提交');
			router.push(`/deployments/${res.deployment_id}`);
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '重启失败');
	}
}
function openDeleteModal() {
	deleteDir.value = false;
	showDeleteModal.value = true;
}

async function handleExport() {
	try {
		const data = await applicationApi.exportApplication(applicationId);
		const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `${data.code || 'application'}.json`;
		a.click();
		URL.revokeObjectURL(url);
		message.success('导出成功');
	} catch (error) {
		message.error(error instanceof Error ? error.message : '导出失败');
	}
}

async function handleDeleteOk() {
	try {
		await executeOp(async () => {
			await applicationApi.delete(applicationId, deleteDir.value);
			message.success('删除成功');
			router.push('/applications');
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '删除失败');
	}
}

async function handleBasicInfoOk() {
	try {
		await basicInfoFormRef.value?.validate();
	} catch {
		return;
	}

	try {
		await executeOp(async () => {
			const payload = {
				name: basicInfoForm.name,
				image_pull_policy: basicInfoForm.image_pull_policy,
				enabled: basicInfoForm.enabled,
				git_source: basicInfoForm.repository_url
					? {
							repository_url: basicInfoForm.repository_url,
							deploy_branches: basicInfoForm.deploy_branches,
							auto_deploy: basicInfoForm.auto_deploy,
						}
					: null,
			};
			await applicationApi.update(applicationId, payload);
			message.success('更新成功');
			showBasicInfoModal.value = false;
			fetchApplication();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '更新失败');
	}
}

async function loadFiles() {
	try {
		await executeFileList(async () => {
			const fileList = await applicationApi.listFiles(applicationId);
			files.value = fileList;
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '加载配置文件失败');
	}
}

async function openFileDrawer(fileId: string, isEdit = false) {
	currentFileId.value = fileId;
	isEditingInDrawer.value = isEdit;
	fileDrawerVisible.value = true;
	currentFileContent.value = '';
	currentFilePath.value = '';

	try {
		await executeFileContent(async () => {
			const result = await applicationApi.readFile(applicationId, fileId);
			currentFileContent.value = result.content ?? '';
			currentFilePath.value = result.path || '';
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '加载文件内容失败');
	}
}

function handleDrawerClose() {
	fileDrawerVisible.value = false;
	isEditingInDrawer.value = false;
}

async function saveCurrentFile() {
	if (!currentFilePath.value.trim()) {
		message.error('请输入文件路径');
		return;
	}

	try {
		await executeFileContent(async () => {
			if (currentFileId.value) {
				// 编辑模式
				const updatedFile = await applicationApi.writeFile(
					applicationId,
					currentFileId.value,
					currentFilePath.value,
					currentFileContent.value
				);
				// 更新文件列表中对应记录
				const index = files.value.findIndex((f) => f.id === currentFileId.value);
				if (index >= 0) {
					files.value[index] = updatedFile;
				}
				message.success('保存成功');
			} else {
				// 新建模式
				await applicationApi.createFile(
					applicationId,
					currentFilePath.value,
					currentFileContent.value
				);
				message.success('添加成功');
			}
			fileDrawerVisible.value = false;
			isEditingInDrawer.value = false;
			loadFiles();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '保存失败');
	}
}

async function deleteFile(fileId: string) {
	try {
		await executeFileList(async () => {
			await applicationApi.deleteFile(applicationId, fileId);
			message.success('删除成功');
			loadFiles();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '删除失败');
	}
}

function openAddFileDrawer() {
	currentFileId.value = '';
	currentFilePath.value = '';
	currentFileContent.value = '';
	isEditingInDrawer.value = true;
	fileDrawerVisible.value = true;
}

onMounted(() => {
	fetchApplication();
	loadFiles();
});

function appStatusColor(status: string) {
	if (status === 'deployed') return 'success';
	if (status === 'deploy_failed') return 'error';
	if (status === 'deploying') return 'processing';
	return 'default';
}

function appStatusLabel(status: string) {
	if (status === 'deployed') return '运行中';
	if (status === 'deploy_failed') return '部署失败';
	if (status === 'deploying') return '部署中';
	return '未部署';
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
</style>
