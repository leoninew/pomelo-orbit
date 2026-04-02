<template>
	<div class="flex flex-col gap-4">
		<!-- Page header -->
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">{{ application?.name ?? '应用详情' }}</h1>
			<button class="btn btn-sm btn-ghost gap-1.5" @click="$router.push('/cd/applications')">
				<ArrowLeft class="size-4" />
				返回
			</button>
		</div>

		<!-- Basic info card -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<div class="flex items-center justify-between mb-4">
					<h2 class="font-semibold">基本信息</h2>
					<div v-if="application" class="flex items-center gap-2 flex-wrap">
						<!-- Deploy with optional env dropdown -->
						<div v-if="envs.length > 1" class="dropdown dropdown-end">
							<button tabindex="0" class="btn btn-sm btn-primary gap-1" :disabled="operating">
								<Rocket class="size-3.5" />
								部署
								<ChevronDown class="size-3" />
							</button>
							<ul
								tabindex="0"
								class="dropdown-content menu bg-base-100 rounded-box shadow-lg border border-base-200 w-40 mt-1 p-1 z-50"
							>
								<li v-for="env in envs.filter((e) => e !== '.env')" :key="env">
									<button @click="handleDeployWithEnv(env)">{{ env }}</button>
								</li>
							</ul>
						</div>
						<button
							v-else
							class="btn btn-sm btn-primary gap-1"
							:disabled="operating"
							@click="handleDeploy"
						>
							<Rocket class="size-3.5" />
							部署
						</button>
						<button class="btn btn-sm btn-ghost" :disabled="operating" @click="handleStop">
							停止
						</button>
						<button class="btn btn-sm btn-ghost" :disabled="operating" @click="handleRestart">
							重启
						</button>
						<button class="btn btn-sm btn-ghost" :disabled="operating" @click="openEditModal">
							编辑
						</button>
						<button class="btn btn-sm btn-ghost" @click="handleExport">导出</button>
						<button
							class="btn btn-sm btn-error btn-ghost"
							:disabled="
								operating || application.status === 'deployed' || application.status === 'deploying'
							"
							@click="openDeleteModal"
						>
							删除
						</button>
					</div>
				</div>

				<div v-if="basicInfoLoading" class="flex justify-center py-8">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<dl v-else-if="application" class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">应用编码</dt>
						<dd>
							<code class="text-xs bg-base-200 px-1.5 py-0.5 rounded">{{ application.code }}</code>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">状态</dt>
						<dd>
							<span class="badge badge-sm" :class="appBadgeClass(application.status)">
								{{ appStatusLabel(application.status) }}
							</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">仓库地址</dt>
						<dd class="truncate">{{ application.git_source?.repository_url || '—' }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">部署分支</dt>
						<dd>{{ application.git_source?.deploy_branches || '—' }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">自动部署</dt>
						<dd>
							<span
								class="badge badge-sm"
								:class="
									application.git_source?.auto_deploy ? 'badge-outline badge-info' : 'badge-ghost'
								"
							>
								{{ application.git_source?.auto_deploy ? '是' : '否' }}
							</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">拉取策略</dt>
						<dd>{{ application.image_pull_policy }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">创建时间</dt>
						<dd class="text-base-content/60">{{ formatTime(application.created_at) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">部署记录</dt>
						<dd>
							<router-link
								:to="`/cd/deployments?application_id=${application.id}`"
								class="link link-primary text-xs"
							>
								查看所有部署
							</router-link>
						</dd>
					</div>
				</dl>
			</div>
		</div>

		<!-- Config files card -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<div class="flex items-center justify-between mb-4">
					<h2 class="font-semibold">配置文件</h2>
					<button class="btn btn-sm btn-primary gap-1" @click="openAddFileDrawer">
						<Plus class="size-3.5" />
						添加文件
					</button>
				</div>
				<div v-if="fileListLoading" class="flex justify-center py-8">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<div
					v-else-if="files.length === 0"
					class="flex flex-col items-center gap-2 py-8 text-base-content/60"
				>
					<FileX class="size-10" />
					<span class="text-sm">暂无配置文件</span>
				</div>
				<div v-else class="overflow-x-auto">
					<table class="table">
						<thead>
							<tr class="text-base-content/60">
								<th>文件路径</th>
								<th>创建时间</th>
								<th>操作</th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="file in files" :key="file.id" class="hover">
								<td>
									<code class="text-xs">{{ file.path }}</code>
								</td>
								<td class="cell-muted">{{ formatTime(file.created_at) }}</td>
								<td>
									<div class="flex items-center gap-3">
										<button class="link link-primary" @click="openFileDrawer(file.id)">查看</button>
										<button class="link link-primary" @click="openFileDrawer(file.id, true)">
											编辑
										</button>
										<button class="link link-error" @click="confirmDeleteFile(file.id)">
											删除
										</button>
									</div>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</div>

		<!-- File panel -->
		<Teleport to="body">
			<Transition
				enter-active-class="transition-transform duration-300 ease-out"
				enter-from-class="translate-x-full"
				enter-to-class="translate-x-0"
				leave-active-class="transition-transform duration-300 ease-in"
				leave-from-class="translate-x-0"
				leave-to-class="translate-x-full"
			>
				<div
					v-if="fileDrawerVisible"
					class="fixed inset-y-0 right-0 z-50 w-[720px] max-w-full bg-base-100 shadow-2xl flex flex-col border-l border-base-200"
				>
					<div class="flex items-center justify-between px-5 py-4 border-b border-base-200">
						<h3 class="font-semibold">
							{{
								currentFileId
									? (isEditingInDrawer ? '编辑文件' : '查看文件') + ': ' + currentFilePath
									: '新建文件'
							}}
						</h3>
						<button class="btn btn-sm btn-ghost btn-circle" @click="handleDrawerClose">
							<X class="size-4" />
						</button>
					</div>
					<div class="flex-1 overflow-auto p-5 flex flex-col gap-4">
						<div v-if="isEditingInDrawer || !currentFileId">
							<fieldset class="fieldset">
								<legend class="fieldset-legend">文件路径</legend>
								<input
									v-model="currentFilePath"
									type="text"
									class="input w-full"
									placeholder="例如: nginx.conf"
								/>
							</fieldset>
						</div>
						<div style="height: calc(100vh - 180px)">
							<CodeEditor
								v-if="!fileContentLoading"
								v-model:value="currentFileContent"
								:style="{ height: '100%' }"
								theme="vs"
								:language="currentFileLanguage"
								:options="{
									readOnly: !isEditingInDrawer && !!currentFileId,
									minimap: { enabled: false },
									fontSize: 14,
									automaticLayout: true,
								}"
							/>
							<div v-else class="flex justify-center items-center h-full">
								<span class="loading loading-spinner loading-md text-primary" />
							</div>
						</div>
					</div>
					<div class="flex items-center justify-end gap-2 px-5 py-4 border-t border-base-200">
						<template v-if="!isEditingInDrawer && currentFileId">
							<button class="btn btn-sm btn-primary" @click="isEditingInDrawer = true">编辑</button>
							<button class="btn btn-sm btn-ghost" @click="handleDrawerClose">关闭</button>
						</template>
						<template v-else>
							<button
								class="btn btn-sm btn-primary"
								:disabled="fileContentLoading"
								@click="saveCurrentFile"
							>
								<span v-if="fileContentLoading" class="loading loading-spinner loading-xs" />
								保存
							</button>
							<button class="btn btn-sm btn-ghost" @click="handleDrawerClose">取消</button>
						</template>
					</div>
				</div>
			</Transition>
			<!-- Overlay -->
			<Transition
				enter-active-class="transition-opacity duration-300"
				enter-from-class="opacity-0"
				enter-to-class="opacity-100"
				leave-active-class="transition-opacity duration-300"
				leave-from-class="opacity-100"
				leave-to-class="opacity-0"
			>
				<div
					v-if="fileDrawerVisible"
					class="fixed inset-0 z-40 bg-black/30"
					@click="handleDrawerClose"
				/>
			</Transition>
		</Teleport>

		<!-- Edit basic info modal -->
		<dialog ref="editModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">编辑基本信息</h3>
				<div class="flex flex-col gap-4">
					<div class="form-control w-full">
						<label class="label">
							<span class="label-text">应用名称</span>
						</label>
						<input
							v-model="editForm.name"
							type="text"
							class="input input-bordered w-full"
							:class="{ 'input-error': editErrors.name }"
						/>
						<label v-if="editErrors.name" class="label">
							<span class="label-text-alt text-error">{{ editErrors.name }}</span>
						</label>
					</div>
					<div class="form-control w-full">
						<label class="label">
							<span class="label-text">应用编码</span>
						</label>
						<input
							:value="editForm.code"
							type="text"
							class="input input-bordered w-full opacity-60"
							disabled
						/>
					</div>
					<div class="form-control w-full">
						<label class="label">
							<span class="label-text">仓库地址</span>
						</label>
						<input
							v-model="editForm.repository_url"
							type="text"
							class="input input-bordered w-full"
						/>
					</div>
					<div class="form-control w-full">
						<label class="label">
							<span class="label-text">部署分支</span>
						</label>
						<input
							v-model="editForm.deploy_branches"
							type="text"
							class="input input-bordered w-full"
						/>
					</div>
					<div class="form-control w-full">
						<label class="label">
							<span class="label-text">镜像拉取策略</span>
						</label>
						<select v-model="editForm.image_pull_policy" class="select select-bordered w-full">
							<option value="always">always</option>
							<option value="missing">missing</option>
							<option value="never">never</option>
						</select>
					</div>
					<div class="form-control w-full">
						<label class="label cursor-pointer justify-start gap-3">
							<input v-model="editForm.auto_deploy" type="checkbox" class="toggle toggle-primary" />
							<span class="label-text">自动部署</span>
						</label>
					</div>
					<div class="form-control w-full">
						<label class="label cursor-pointer justify-start gap-3">
							<input v-model="editForm.enabled" type="checkbox" class="toggle toggle-primary" />
							<span class="label-text">启用</span>
						</label>
					</div>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleEditOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						保存
					</button>
					<button class="btn btn-ghost" @click="editModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Delete modal -->
		<dialog ref="deleteModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">删除应用</h3>
				<p class="py-4 text-sm">
					确定要删除应用「
					<strong>{{ application?.name }}</strong>
					」吗？此操作不可撤销。
				</p>
				<label class="flex items-center gap-2 cursor-pointer mb-2">
					<input v-model="deleteDir" type="checkbox" class="checkbox checkbox-sm checkbox-error" />
					<span class="text-sm">同时删除应用工作目录（data/apps/{{ application?.code }}）</span>
				</label>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="operating" @click="handleDeleteOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						删除
					</button>
					<button class="btn btn-ghost" @click="deleteModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Delete file confirm modal -->
		<dialog ref="deleteFileModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">删除文件</h3>
				<p class="py-4 text-sm">确定删除此配置文件？</p>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="fileListLoading" @click="executeDeleteFile">
						<span v-if="fileListLoading" class="loading loading-spinner loading-xs" />
						删除
					</button>
					<button class="btn btn-ghost" @click="deleteFileModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ArrowLeft, Rocket, ChevronDown, Plus, FileX, X } from 'lucide-vue-next';
import { CodeEditor } from 'monaco-editor-vue3';
import { applicationApi } from '@/api/application';
import { deploymentApi } from '@/api/deployments';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { appStatusLabel } from '@/utils/status';
import { formatTime, delayAsync } from '@/utils/time';
import type { Application, ConfigFile } from '@/types/api';

const route = useRoute();
const router = useRouter();
const applicationId = route.params.id as string;
const toast = useToast();

const { loading: basicInfoLoading, execute: executeBasicInfo } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const { loading: fileListLoading, execute: executeFileList } = useStatusAsync();
const { loading: fileContentLoading, execute: executeFileContent } = useStatusAsync();

const application = ref<Application>();
const files = ref<ConfigFile[]>([]);

const editModalRef = ref<HTMLDialogElement>();
const deleteModalRef = ref<HTMLDialogElement>();
const deleteFileModalRef = ref<HTMLDialogElement>();
const pendingDeleteFileId = ref('');
const deleteDir = ref(false);

const fileDrawerVisible = ref(false);
const currentFileId = ref('');
const currentFilePath = ref('');
const currentFileContent = ref('');
const isEditingInDrawer = ref(false);

const editForm = reactive({
	name: '',
	code: '',
	repository_url: '',
	deploy_branches: '',
	auto_deploy: false,
	image_pull_policy: 'missing',
	enabled: true,
});
const editErrors = reactive({ name: '' });

const envs = computed(() =>
	files.value.filter((f) => f.path.match(/^\.env(\..+)?$/)).map((f) => f.path)
);

const badgeMap: Record<string, string> = {
	deployed: 'badge-outline badge-success',
	deploy_failed: 'badge-outline badge-error',
	deploying: 'badge-outline badge-info',
	undeployed: 'badge-ghost',
};
function appBadgeClass(s: string) {
	return badgeMap[s] ?? 'badge-ghost';
}

const currentFileLanguage = computed(() => {
	const p = currentFilePath.value.toLowerCase();
	if (p.endsWith('.sh') || p.endsWith('.bash')) return 'shell';
	if (p.startsWith('.env') || p.endsWith('.ini') || p.endsWith('.properties')) return 'ini';
	return 'yaml';
});

async function fetchApplication() {
	try {
		await executeBasicInfo(async () => {
			const data = await applicationApi.get(applicationId);
			application.value = data;
			Object.assign(editForm, {
				name: data.name,
				code: data.code,
				image_pull_policy: data.image_pull_policy,
				enabled: data.enabled,
				repository_url: data.git_source?.repository_url ?? '',
				deploy_branches: data.git_source?.deploy_branches ?? '',
				auto_deploy: data.git_source?.auto_deploy ?? false,
			});
		});
		if (application.value?.status === 'deploying') pollActiveDeployment();
	} catch {
		toast.error('获取应用信息失败');
		router.push('/cd/applications');
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
					if (application.value)
						application.value.status =
							detail.status === 'ran_to_completion' ? 'deployed' : 'deploy_failed';
					break;
				}
			} catch {
				break;
			}
		}
	} catch {
		/* silent */
	}
}

async function handleDeploy() {
	try {
		await executeOp(async () => {
			const defaultEnv = envs.value.includes('.env') ? '.env' : undefined;
			const res = await applicationApi.deploy(applicationId, undefined, defaultEnv);
			toast.success(`部署已触发`);
			router.push(`/cd/deployments/${res.deployment_id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '触发部署失败');
	}
}

async function handleDeployWithEnv(env: string) {
	try {
		await executeOp(async () => {
			const res = await applicationApi.deploy(applicationId, undefined, env);
			toast.success(`部署已触发`);
			router.push(`/cd/deployments/${res.deployment_id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '触发部署失败');
	}
}

async function handleStop() {
	try {
		await executeOp(async () => {
			const res = await applicationApi.stop(applicationId);
			toast.success('停止操作已提交');
			while (true) {
				await delayAsync(3000);
				try {
					const detail = await deploymentApi.get(res.deployment_id);
					if (['ran_to_completion', 'faulted', 'canceled'].includes(detail.status)) {
						if (application.value)
							application.value.status =
								detail.status === 'ran_to_completion' ? 'undeployed' : 'deploy_failed';
						break;
					}
				} catch {
					break;
				}
			}
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '停止失败');
	}
}

async function handleRestart() {
	try {
		await executeOp(async () => {
			const res = await applicationApi.restart(applicationId);
			toast.success('重启操作已提交');
			router.push(`/cd/deployments/${res.deployment_id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '重启失败');
	}
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
		toast.success('导出成功');
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '导出失败');
	}
}

function openEditModal() {
	editErrors.name = '';
	editModalRef.value?.showModal();
}

async function handleEditOk() {
	editErrors.name = editForm.name.trim() ? '' : '请输入应用名称';
	if (editErrors.name) return;
	try {
		await executeOp(async () => {
			await applicationApi.update(applicationId, {
				name: editForm.name,
				image_pull_policy: editForm.image_pull_policy,
				enabled: editForm.enabled,
				git_source: editForm.repository_url
					? {
							repository_url: editForm.repository_url,
							deploy_branches: editForm.deploy_branches,
							auto_deploy: editForm.auto_deploy,
						}
					: null,
			});
			toast.success('更新成功');
			editModalRef.value?.close();
			fetchApplication();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '更新失败');
	}
}

function openDeleteModal() {
	deleteDir.value = false;
	deleteModalRef.value?.showModal();
}

async function handleDeleteOk() {
	try {
		await executeOp(async () => {
			await applicationApi.delete(applicationId, deleteDir.value);
			toast.success('删除成功');
			router.push('/cd/applications');
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '删除失败');
	}
}

async function loadFiles() {
	try {
		await executeFileList(async () => {
			files.value = await applicationApi.listFiles(applicationId);
		});
	} catch {
		toast.error('加载配置文件失败');
	}
}

async function openFileDrawer(fileId: string, isEdit = false) {
	currentFileId.value = fileId;
	isEditingInDrawer.value = isEdit;
	currentFileContent.value = '';
	currentFilePath.value = '';
	fileDrawerVisible.value = true;
	try {
		await executeFileContent(async () => {
			const result = await applicationApi.readFile(applicationId, fileId);
			currentFileContent.value = result.content ?? '';
			currentFilePath.value = result.path || '';
		});
	} catch {
		toast.error('加载文件内容失败');
	}
}

function openAddFileDrawer() {
	currentFileId.value = '';
	currentFilePath.value = '';
	currentFileContent.value = '';
	isEditingInDrawer.value = true;
	fileDrawerVisible.value = true;
}

function handleDrawerClose() {
	fileDrawerVisible.value = false;
	isEditingInDrawer.value = false;
}

async function saveCurrentFile() {
	if (!currentFilePath.value.trim()) {
		toast.error('请输入文件路径');
		return;
	}
	const lowerPath = currentFilePath.value.toLowerCase();
	const content =
		lowerPath.endsWith('.sh') || lowerPath.endsWith('.bash')
			? currentFileContent.value.replace(/\r\n/g, '\n')
			: currentFileContent.value;
	try {
		await executeFileContent(async () => {
			if (currentFileId.value) {
				const updated = await applicationApi.writeFile(
					applicationId,
					currentFileId.value,
					currentFilePath.value,
					content
				);
				const idx = files.value.findIndex((f) => f.id === currentFileId.value);
				if (idx >= 0) files.value[idx] = updated;
				toast.success('保存成功');
			} else {
				await applicationApi.createFile(applicationId, currentFilePath.value, content);
				toast.success('添加成功');
			}
			fileDrawerVisible.value = false;
			isEditingInDrawer.value = false;
			loadFiles();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '保存失败');
	}
}

function confirmDeleteFile(fileId: string) {
	pendingDeleteFileId.value = fileId;
	deleteFileModalRef.value?.showModal();
}

async function executeDeleteFile() {
	try {
		await executeFileList(async () => {
			await applicationApi.deleteFile(applicationId, pendingDeleteFileId.value);
			toast.success('删除成功');
			deleteFileModalRef.value?.close();
			loadFiles();
		});
	} catch {
		toast.error('删除失败');
	}
}

onMounted(() => {
	fetchApplication();
	loadFiles();
});
</script>
