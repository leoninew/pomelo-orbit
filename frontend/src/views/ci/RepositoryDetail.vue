<template>
	<div class="flex flex-col gap-4">
		<!-- Page header -->
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">{{ repository?.name ?? '仓库详情' }}</h1>
			<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/ci/repository')">
				<ArrowLeft class="size-4" />
				返回
			</button>
		</div>

		<!-- Basic info card -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<div class="flex items-center justify-between mb-4">
					<h2 class="font-semibold">基本信息</h2>
					<div v-if="repository" class="flex items-center gap-2 flex-wrap">
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
				<dl v-else-if="repository" class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">名称</dt>
						<dd>{{ repository.name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">编码</dt>
						<dd>{{ repository.code }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">地址</dt>
						<dd class="text-xs truncate">{{ repository.repository_url }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">默认分支</dt>
						<dd class="text-base-content/60">{{ repository.default_branch }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">Git 凭据</dt>
						<dd>
							<router-link
								v-if="repository.git_credential_id"
								:to="`/ci/credential/${repository.git_credential_id}`"
								class="link link-primary text-sm"
							>
								{{ repository.git_credential_name }}
							</router-link>
							<span v-else class="text-base-content/60">未配置</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">流水线记录</dt>
						<dd>
							<router-link
								:to="`/ci/run?repository_id=${repository.id}`"
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
					<button class="btn btn-sm btn-primary gap-1.5" @click="openAddVarModal">
						<Plus class="size-4" />
						添加自定义变量
					</button>
				</div>

				<VariableDeclarationsTable
					:declarations="allVariables"
					:readonly="false"
					@edit="openEditVarModal"
					@delete="deleteVariable"
				/>
			</div>
		</div>

		<!-- Webhooks card -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<WebhookList
					v-if="!loading"
					:repository-id="repositoryId"
					:webhooks="webhooks"
					:templates="templates"
					@refresh="fetchWebhooks"
				/>
			</div>
		</div>

		<!-- Trigger modal -->
		<TriggerModal
			ref="triggerModalRef"
			:repository-id="repositoryId"
			:templates="templates"
			:default-branch="repository?.default_branch"
			:project-variables="repositoryCustomVariables"
			:repository="repository"
			@trigger="handleTrigger"
		/>

		<!-- Edit modal -->
		<dialog ref="editModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">编辑项目</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">名称</legend>
						<input v-model="editForm.name" type="text" class="input w-full" />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">编码</legend>
						<input :value="repository?.code" type="text" class="input w-full opacity-60" disabled />
						<p class="fieldset-label text-base-content/50">创建后不可修改</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">地址</legend>
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
				<h3 class="font-bold text-lg">删除仓库</h3>
				<p class="py-4 text-sm">
					确定要删除仓库「
					<strong>{{ repository?.name }}</strong>
					」吗？此操作不可撤销。
				</p>
				<label class="flex items-center gap-2 cursor-pointer mb-2">
					<input
						v-model="deleteWorkspace"
						type="checkbox"
						class="checkbox checkbox-sm checkbox-error"
					/>
					<span class="text-sm">同时删除工作目录（data/ci/{{ repository?.code }}）</span>
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
					<fieldset class="fieldset">
						<legend class="fieldset-legend">说明（可选）</legend>
						<input
							v-model="newVarDescription"
							type="text"
							class="input w-full"
							placeholder="变量说明"
						/>
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
					<fieldset class="fieldset">
						<legend class="fieldset-legend">说明（可选）</legend>
						<input
							v-model="editingVarDescription"
							type="text"
							class="input w-full"
							placeholder="变量说明"
						/>
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
import { credentialApi, pipelineTemplateApi, repositoryApi, webhookApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { Credential, PipelineTemplate, Repository, RepositoryWebhook } from '@/types/api';
import { credentialTypeLabels } from '@/types/api';
import TriggerModal from './components/TriggerModal.vue';
import WebhookList from './components/WebhookList.vue';
import VariableDeclarationsTable from './components/VariableDeclarationsTable.vue';

const route = useRoute();
const router = useRouter();
const repositoryId = route.params.id as string;
const toast = useToast();

const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const repository = ref<Repository>();
const templates = ref<PipelineTemplate[]>([]);
const webhooks = ref<RepositoryWebhook[]>([]);
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
const newVarDescription = ref('');
const editingVarKey = ref('');
const editingVarValue = ref('');
const editingVarDescription = ref('');

const repositoryCustomVariables = computed(
	() =>
		repository.value?.variable_declarations.filter((v) => v.source === 'repository_custom') || []
);

// 合并所有变量到一个列表
const allVariables = computed(() => {
	return repository.value?.variable_declarations || [];
});

// 使用统一的工具函数获取来源标签和样式

async function fetchProject() {
	try {
		await execute(async () => {
			const data = await repositoryApi.get(repositoryId);
			repository.value = data;
			Object.assign(editForm, {
				name: data.name,
				repository_url: data.repository_url,
				git_credential_id: data.git_credential_id ?? '',
				default_branch: data.default_branch ?? 'master',
			});
		});
	} catch {
		toast.error('获取代码仓库信息失败');
		router.push('/ci/repository');
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
		webhooks.value = await webhookApi.list(repositoryId);
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

async function openTriggerModal() {
	await fetchTemplates();
	triggerModalRef.value?.open();
}

async function handleTrigger(data: {
	template_id: string
	trigger_ref: string
	variables: Record<string, string>
}) {
	try {
		await executeOp(async () => {
			const run = await repositoryApi.trigger(repositoryId, data);
			toast.success('触发成功');
			router.push(`/ci/run/${run.id}`);
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
			await repositoryApi.update(repositoryId, {
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
			await repositoryApi.delete(repositoryId);
			toast.success('删除成功');
			router.push('/ci/repository');
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '删除失败');
	}
}

function openAddVarModal() {
	newVarKey.value = '';
	newVarValue.value = '';
	newVarDescription.value = '';
	addVarModalRef.value?.showModal();
}

function openEditVarModal(name: string) {
	const variable = allVariables.value.find((v) => v.name === name);
	if (!variable) {
		return;
	}
	editingVarKey.value = variable.name;
	editingVarValue.value = variable.value || '';
	editingVarDescription.value = variable.description || '';
	editVarModalRef.value?.showModal();
}

async function handleAddVarOk() {
	if (!newVarKey.value.trim()) {
		toast.error('请输入变量名');
		return;
	}
	try {
		await executeOp(async () => {
			const data = await repositoryApi.update(repositoryId, {
				variable_overrides: [
					...repositoryCustomVariables.value,
					{
						name: newVarKey.value,
						value: newVarValue.value,
						description: newVarDescription.value,
					},
				],
			});
			const repositoryCustom = data.variable_declarations.filter(
				(v) => v.source === 'repository_custom'
			);
			const added = repositoryCustom.some((v) => v.name === newVarKey.value);
			if (!added) {
				toast.error('内置变量不能在项目级配置');
				return;
			}
			repository.value = data;
			toast.success('添加成功');
			addVarModalRef.value?.close();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '添加失败');
	}
}

async function handleEditVarOk() {
	try {
		await executeOp(async () => {
			const data = await repositoryApi.update(repositoryId, {
				variable_overrides: repositoryCustomVariables.value.map((v) =>
					v.name === editingVarKey.value
						? {
								name: v.name,
								value: editingVarValue.value,
								description: editingVarDescription.value,
							}
						: v
				),
			});
			const repositoryCustom = data.variable_declarations.filter(
				(v) => v.source === 'repository_custom'
			);
			const updated = repositoryCustom.some((v) => v.name === editingVarKey.value);
			if (!updated) {
				toast.error('内置变量不能在项目级配置');
				return;
			}
			repository.value = data;
			toast.success('更新成功');
			editVarModalRef.value?.close();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '更新失败');
	}
}

async function deleteVariable(key: string) {
	try {
		await executeOp(async () => {
			const data = await repositoryApi.update(repositoryId, {
				variable_overrides: repositoryCustomVariables.value.filter((v) => v.name !== key),
			});
			repository.value = data;
			toast.success('删除成功');
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '删除失败');
	}
}

onMounted(async () => {
	await fetchProject();
	await fetchWebhooks();
});
</script>
