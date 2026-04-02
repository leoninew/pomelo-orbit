<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">{{ project?.name ?? '项目详情' }}</h1>
			<div class="flex items-center gap-2">
				<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/ci/projects')">
					<ArrowLeft class="size-4" />
					返回
				</button>
				<button class="btn btn-sm btn-primary gap-1" @click="openTriggerModal">
					<Play class="size-4" />
					手动触发
				</button>
				<button class="btn btn-sm btn-ghost" @click="openEditModal">编辑</button>
			</div>
		</div>

		<!-- Basic info -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<h2 class="font-semibold mb-3">基本信息</h2>
				<div v-if="loading" class="flex justify-center py-6">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<dl v-else-if="project" class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">项目名称</dt>
						<dd>{{ project.name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">仓库地址</dt>
						<dd class="text-xs truncate">{{ project.repository_url }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">流水线模板</dt>
						<dd>
							<router-link
								:to="`/ci/templates/${project.pipeline_template_id}`"
								class="link link-primary text-xs"
							>
								查看模板
							</router-link>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">Git 凭据</dt>
						<dd class="text-base-content/60">
							{{ project.git_credential_id ? '已配置' : '未配置' }}
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">分支过滤</dt>
						<dd class="text-base-content/60">{{ project.branch_filter || '所有分支' }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">默认分支</dt>
						<dd class="text-base-content/60">{{ project.default_branch }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">创建时间</dt>
						<dd>{{ formatTime(project.created_at) }}</dd>
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

		<!-- Webhook -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<h2 class="font-semibold mb-3">Webhook 配置</h2>
				<dl v-if="project" class="flex flex-col gap-3 text-sm">
					<div class="flex gap-2 items-start">
						<dt class="text-base-content/70 w-32 shrink-0">Webhook URL</dt>
						<dd class="flex items-center gap-2">
							<code class="text-xs bg-base-200 px-2 py-1 rounded break-all">{{ webhookUrl }}</code>
							<button class="btn btn-xs btn-ghost" @click="copyText(webhookUrl)">
								<Copy class="size-3" />
							</button>
						</dd>
					</div>
					<div class="flex gap-2 items-center">
						<dt class="text-base-content/70 w-32 shrink-0">Webhook Secret</dt>
						<dd class="flex items-center gap-2">
							<code class="text-xs bg-base-200 px-2 py-1 rounded">
								{{ showSecret ? project.webhook_secret : '••••••••••••••••' }}
							</code>
							<button class="btn btn-xs btn-ghost" @click="showSecret = !showSecret">
								{{ showSecret ? '隐藏' : '显示' }}
							</button>
							<button
								v-if="showSecret"
								class="btn btn-xs btn-ghost"
								@click="copyText(project.webhook_secret ?? '')"
							>
								<Copy class="size-3" />
							</button>
						</dd>
					</div>
				</dl>
			</div>
		</div>

		<!-- Variables -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<div class="flex items-center justify-between mb-3">
					<h2 class="font-semibold">变量配置</h2>
					<button class="btn btn-xs btn-primary gap-1" @click="openAddVarModal">
						<Plus class="size-3" />
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
							<td>
								<code class="text-xs">{{ v.key }}</code>
							</td>
							<td>
								<code class="text-xs">{{ v.value }}</code>
							</td>
							<td>
								<div class="flex items-center gap-2">
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

		<!-- Trigger modal -->
		<dialog ref="triggerModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg mb-4">手动触发</h3>
				<fieldset class="fieldset">
					<legend class="fieldset-legend">分支</legend>
					<input v-model="triggerRef" type="text" class="input w-full" placeholder="输入分支名" />
				</fieldset>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleTriggerOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						触发
					</button>
					<button class="btn btn-ghost" @click="triggerModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

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
						<legend class="fieldset-legend">仓库地址</legend>
						<input v-model="editForm.repository_url" type="text" class="input w-full" />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">流水线模板</legend>
						<select v-model="editForm.pipeline_template_id" class="select w-full">
							<option v-for="tpl in templates" :key="tpl.id" :value="tpl.id">{{ tpl.name }}</option>
						</select>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">Git 凭据</legend>
						<select v-model="editForm.git_credential_id" class="select w-full">
							<option value="">不使用凭据</option>
							<option v-for="cred in gitCredentials" :key="cred.id" :value="cred.id">
								{{ cred.name }}
							</option>
						</select>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">分支过滤</legend>
						<input v-model="editForm.branch_filter" type="text" class="input w-full" />
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
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ArrowLeft, Play, Plus, Copy } from 'lucide-vue-next';
import { credentialApi, pipelineTemplateApi, projectApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { formatTime } from '@/utils/time';
import type { Credential, PipelineTemplate, Project } from '@/types/api';

const route = useRoute();
const router = useRouter();
const projectId = route.params.id as string;
const toast = useToast();

const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const project = ref<Project>();
const templates = ref<PipelineTemplate[]>([]);
const credentials = ref<Credential[]>([]);
const gitCredentials = computed(() =>
	credentials.value.filter((c) => c.type === 'git_ssh' || c.type === 'git_token')
);
const showSecret = ref(false);

const triggerModalRef = ref<HTMLDialogElement>();
const editModalRef = ref<HTMLDialogElement>();
const addVarModalRef = ref<HTMLDialogElement>();
const editVarModalRef = ref<HTMLDialogElement>();

const triggerRef = ref('');
const editForm = reactive({
	name: '',
	repository_url: '',
	pipeline_template_id: '',
	git_credential_id: '',
	branch_filter: '',
	default_branch: 'master',
});
const newVarKey = ref('');
const newVarValue = ref('');
const editingVarKey = ref('');
const editingVarValue = ref('');

const webhookUrl = computed(() => `${window.location.origin}/api/v1/ci/webhooks/git`);
const variableList = computed(() =>
	Object.entries(project.value?.variable_overrides ?? {}).map(([key, value]) => ({ key, value }))
);

async function copyText(text: string) {
	await navigator.clipboard.writeText(text);
	toast.success('已复制');
}

async function fetchProject() {
	try {
		await execute(async () => {
			const data = await projectApi.get(projectId);
			project.value = data;
			Object.assign(editForm, {
				name: data.name,
				repository_url: data.repository_url,
				pipeline_template_id: data.pipeline_template_id,
				git_credential_id: data.git_credential_id ?? '',
				branch_filter: data.branch_filter ?? '',
				default_branch: data.default_branch ?? 'master',
			});
		});
	} catch {
		toast.error('获取项目信息失败');
		router.push('/ci/projects');
	}
}

function openTriggerModal() {
	triggerRef.value = project.value?.default_branch || 'master';
	triggerModalRef.value?.showModal();
}

async function handleTriggerOk() {
	try {
		await executeOp(async () => {
			const run = await projectApi.trigger(projectId, { trigger_ref: triggerRef.value });
			toast.success('触发成功');
			triggerModalRef.value?.close();
			router.push(`/ci/runs/${run.id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '触发失败');
	}
}

async function openEditModal() {
	const [tplRes, credRes] = await Promise.all([
		pipelineTemplateApi.list({ per_page: 100 }),
		credentialApi.list({ per_page: 100 }),
	]);
	templates.value = tplRes.items;
	credentials.value = credRes.items;
	editModalRef.value?.showModal();
}

async function handleEditOk() {
	try {
		await executeOp(async () => {
			await projectApi.update(projectId, {
				name: editForm.name,
				repository_url: editForm.repository_url,
				pipeline_template_id: editForm.pipeline_template_id,
				git_credential_id: editForm.git_credential_id || undefined,
				branch_filter: editForm.branch_filter || undefined,
				default_branch: editForm.default_branch || 'master',
			});
			toast.success('更新成功');
			editModalRef.value?.close();
			fetchProject();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '更新失败');
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

onMounted(fetchProject);
</script>
