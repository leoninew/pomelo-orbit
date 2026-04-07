<template>
	<div class="flex flex-col gap-4">
		<!-- Page header -->
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">{{ project?.name ?? '项目详情' }}</h1>
			<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/ci/projects')">
				<ArrowLeft class="size-4" />
				返回
			</button>
		</div>

		<!-- Basic info card -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<div class="flex items-center justify-between mb-4">
					<h2 class="font-semibold">基本信息</h2>
					<div v-if="project" class="flex items-center gap-2 flex-wrap">
						<button
							class="btn btn-sm btn-primary gap-1"
							:disabled="operating"
							@click="openTriggerModal"
						>
							<Play class="size-3.5" />
							触发
						</button>
						<button class="btn btn-sm btn-ghost" :disabled="operating" @click="openEditModal">
							编辑
						</button>
						<button
							class="btn btn-sm btn-error btn-ghost"
							:disabled="operating"
							@click="openDeleteModal"
						>
							删除
						</button>
					</div>
				</div>

				<div v-if="loading" class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3">
					<div v-for="i in 6" :key="i" class="flex gap-2">
						<div class="skeleton h-4 w-24 shrink-0"></div>
						<div class="skeleton h-4 w-32"></div>
					</div>
				</div>
				<dl v-else-if="project" class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">项目名称</dt>
						<dd>{{ project.name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">项目编码</dt>
						<dd>{{ project.code }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">仓库地址</dt>
						<dd class="text-xs truncate">{{ project.repository_url }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">默认分支</dt>
						<dd class="text-base-content/60">{{ project.default_branch }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">Git 凭据</dt>
						<dd class="text-base-content/60">
							{{ project.git_credential_id ? '已配置' : '未配置' }}
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">流水线记录</dt>
						<dd>
							<router-link
								:to="`/ci/runs?project_id=${project.id}`"
								class="link link-primary text-xs"
							>
								查看所有记录
							</router-link>
						</dd>
					</div>
				</dl>
			</div>
		</div>

		<!-- Variables card -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<div class="flex items-center justify-between mb-4">
					<h2 class="font-semibold">变量配置</h2>
					<button class="btn btn-sm btn-primary gap-1" @click="openAddVarModal">
						<Plus class="size-3.5" />
						添加变量
					</button>
				</div>
				<div v-if="variableList.length === 0" class="text-sm text-base-content/60 py-4 text-center">
					未配置变量
				</div>
				<table v-else class="table">
					<thead>
						<tr class="text-base-content/60">
							<th>变量名</th>
							<th>变量值</th>
							<th>操作</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="v in variableList" :key="v.key" class="hover">
							<td>{{ v.key }}</td>
							<td>
								<code class="text-xs">{{ v.value }}</code>
							</td>
							<td>
								<div class="flex items-center gap-3">
									<button class="link link-primary" @click="openEditVarModal(v.key, v.value)">
										编辑
									</button>
									<button class="link link-error" @click="deleteVariable(v.key)">删除</button>
								</div>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>

		<!-- Webhooks card -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<WebhookList
					v-if="!loading"
					:project-id="projectId"
					:webhooks="webhooks"
					:templates="templates"
					@refresh="fetchWebhooks"
				/>
			</div>
		</div>

		<!-- Trigger modal -->
		<TriggerModal
			ref="triggerModalRef"
			:project-id="projectId"
			:templates="templates"
			:default-branch="project?.default_branch"
			:project-variables="project?.variable_overrides"
			@trigger="handleTrigger"
		/>

		<!-- Edit modal -->
		<dialog ref="editModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">编辑项目</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">项目名称</legend>
						<input v-model="editForm.name" type="text" class="input w-full" />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">项目编码</legend>
						<input :value="project?.code" type="text" class="input w-full opacity-60" disabled />
						<p class="fieldset-label text-base-content/50">创建后不可修改</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">仓库地址</legend>
						<input v-model="editForm.repository_url" type="text" class="input w-full" />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">Git 凭据</legend>
						<select v-model="editForm.git_credential_id" class="select w-full">
							<option value="">不使用凭据</option>
							<option v-for="cred in gitCredentials" :key="cred.id" :value="cred.id">
								{{ cred.name }} ({{ credentialTypeLabels[cred.type] }})
							</option>
						</select>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">默认分支</legend>
						<input
							v-model="editForm.default_branch"
							type="text"
							class="input w-full"
							placeholder="master"
						/>
					</fieldset>
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
				<h3 class="font-bold text-lg">删除项目</h3>
				<p class="py-4 text-sm">
					确定要删除项目「
					<strong>{{ project?.name }}</strong>
					」吗？此操作不可撤销。
				</p>
				<label class="flex items-center gap-2 cursor-pointer mb-2">
					<input
						v-model="deleteWorkspace"
						type="checkbox"
						class="checkbox checkbox-sm checkbox-error"
					/>
					<span class="text-sm">同时删除工作目录（data/ci/{{ project?.code }}）</span>
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

		<!-- Add variable modal -->
		<dialog ref="addVarModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg mb-4">添加变量</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">变量名</legend>
						<input v-model="newVarKey" type="text" class="input w-full" placeholder="变量名" />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">变量值</legend>
						<input v-model="newVarValue" type="text" class="input w-full" placeholder="变量值" />
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleAddVarOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						保存
					</button>
					<button class="btn btn-ghost" @click="addVarModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Edit variable modal -->
		<dialog ref="editVarModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg mb-4">编辑变量</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">变量名</legend>
						<input :value="editingVarKey" type="text" class="input w-full opacity-60" disabled />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">变量值</legend>
						<input v-model="editingVarValue" type="text" class="input w-full" />
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleEditVarOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						保存
					</button>
					<button class="btn btn-ghost" @click="editVarModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { ArrowLeft, Play, Plus } from 'lucide-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { credentialApi, pipelineTemplateApi, projectApi, webhookApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import WebhookList from './components/WebhookList.vue';
import TriggerModal from './components/TriggerModal.vue';
import { credentialTypeLabels } from '@/types/api';
import type { Credential, PipelineTemplate, Project, ProjectWebhook } from '@/types/api';

const route = useRoute();
const router = useRouter();
const projectId = route.params.id as string;
const toast = useToast();

const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const project = ref<Project>();
const templates = ref<PipelineTemplate[]>([]);
const webhooks = ref<ProjectWebhook[]>([]);
const credentials = ref<Credential[]>([]);
const gitCredentials = computed(() =>
	credentials.value.filter((c) => c.type === 'git_ssh' || c.type === 'git_token')
);

const triggerModalRef = ref<InstanceType<typeof TriggerModal>>();
const editModalRef = ref<HTMLDialogElement>();
const deleteModalRef = ref<HTMLDialogElement>();
const addVarModalRef = ref<HTMLDialogElement>();
const editVarModalRef = ref<HTMLDialogElement>();

const deleteWorkspace = ref(false);
const editForm = reactive({
	name: '',
	repository_url: '',
	git_credential_id: '',
	default_branch: 'master',
});
const newVarKey = ref('');
const newVarValue = ref('');
const editingVarKey = ref('');
const editingVarValue = ref('');

const variableList = computed(() =>
	Object.entries(project.value?.variable_overrides ?? {}).map(([key, value]) => ({ key, value }))
);

async function fetchProject() {
	try {
		await execute(async () => {
			const data = await projectApi.get(projectId);
			project.value = data;
			Object.assign(editForm, {
				name: data.name,
				repository_url: data.repository_url,
				git_credential_id: data.git_credential_id ?? '',
				default_branch: data.default_branch ?? 'master',
			});
		});
	} catch {
		toast.error('获取项目信息失败');
		router.push('/ci/projects');
	}
}

async function fetchTemplates() {
	try {
		const res = await pipelineTemplateApi.list({ per_page: 100 });
		templates.value = res.items;
	} catch {
		toast.error('获取模板列表失败');
	}
}

async function fetchWebhooks() {
	try {
		webhooks.value = await webhookApi.list(projectId);
	} catch {
		toast.error('获取 Webhook 列表失败');
	}
}

async function fetchCredentials() {
	try {
		const res = await credentialApi.list({ per_page: 100 });
		credentials.value = res.items;
	} catch {
		toast.error('获取凭据列表失败');
	}
}

function openTriggerModal() {
	triggerModalRef.value?.open();
}

async function handleTrigger(data: {
	template_id: string
	trigger_ref: string
	variables: Record<string, string>
}) {
	try {
		await executeOp(async () => {
			const run = await projectApi.trigger(projectId, data);
			toast.success('触发成功');
			router.push(`/ci/runs/${run.id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '触发失败');
	}
}

async function openEditModal() {
	await fetchTemplates();
	await fetchCredentials();
	editModalRef.value?.showModal();
}

async function handleEditOk() {
	try {
		await executeOp(async () => {
			await projectApi.update(projectId, {
				name: editForm.name,
				repository_url: editForm.repository_url,
				git_credential_id: editForm.git_credential_id || undefined,
				default_branch: editForm.default_branch || 'master',
			});
			toast.success('更新成功');
			editModalRef.value?.close();
			await fetchProject();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '更新失败');
	}
}

function openDeleteModal() {
	deleteWorkspace.value = false;
	deleteModalRef.value?.showModal();
}

async function handleDeleteOk() {
	try {
		await executeOp(async () => {
			await projectApi.delete(projectId);
			toast.success('删除成功');
			router.push('/ci/projects');
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '删除失败');
	}
}

function openAddVarModal() {
	newVarKey.value = '';
	newVarValue.value = '';
	addVarModalRef.value?.showModal();
}

function openEditVarModal(key: string, value: string) {
	editingVarKey.value = key;
	editingVarValue.value = value;
	editVarModalRef.value?.showModal();
}

async function handleAddVarOk() {
	if (!newVarKey.value.trim()) {
		toast.error('请输入变量名');
		return;
	}
	try {
		await executeOp(async () => {
			await projectApi.update(projectId, {
				variable_overrides: {
					...project.value?.variable_overrides,
					[newVarKey.value]: newVarValue.value,
				},
			});
			toast.success('添加成功');
			addVarModalRef.value?.close();
			fetchProject();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '添加失败');
	}
}

async function handleEditVarOk() {
	try {
		await executeOp(async () => {
			await projectApi.update(projectId, {
				variable_overrides: {
					...project.value?.variable_overrides,
					[editingVarKey.value]: editingVarValue.value,
				},
			});
			toast.success('更新成功');
			editVarModalRef.value?.close();
			fetchProject();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '更新失败');
	}
}

async function deleteVariable(key: string) {
	try {
		await executeOp(async () => {
			const updated = { ...project.value?.variable_overrides };
			delete updated[key];
			await projectApi.update(projectId, { variable_overrides: updated });
			toast.success('删除成功');
			fetchProject();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '删除失败');
	}
}

onMounted(async () => {
	await Promise.all([fetchProject(), fetchTemplates(), fetchWebhooks(), fetchCredentials()]);
});
</script>
