<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">项目管理</h1>
			<button class="btn btn-sm btn-primary gap-1.5" @click="openCreateModal">
				<Plus class="size-4" />
				新建项目
			</button>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table min-h-48">
				<thead>
					<tr class="text-base-content/60">
						<th>项目名称</th>
						<th>仓库地址</th>
						<th>Git 凭据</th>
						<th>创建时间</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					<tr v-if="status === 'loading'">
						<td colspan="5" class="text-center py-8">
							<span class="loading loading-spinner loading-md text-primary" />
						</td>
					</tr>
					<tr v-else-if="status === 'error'">
						<td colspan="5" class="text-center py-8 text-error">{{ error }}</td>
					</tr>
					<tr v-else-if="projects.length === 0">
						<td colspan="5" class="text-center py-8 text-base-content/60">暂无项目</td>
					</tr>
					<tr v-for="p in projects" :key="p.id" class="hover">
						<td>
							<router-link :to="`/ci/projects/${p.id}`" class="link link-primary font-medium">
								{{ p.name }}
							</router-link>
						</td>
						<td class="cell-muted max-w-xs truncate">{{ p.repository_url }}</td>
						<td class="cell-muted">
							<router-link
								v-if="p.git_credential_id"
								:to="`/ci/credentials/${p.git_credential_id}`"
								class="link link-primary"
							>
								{{ p.git_credential_id }}
							</router-link>
							<span v-else class="text-base-content/40">-</span>
						</td>
						<td class="cell-muted">{{ formatTime(p.created_at) }}</td>
						<td>
							<router-link :to="`/ci/projects/${p.id}`" class="link link-primary">查看</router-link>
						</td>
					</tr>
				</tbody>
			</table>
			<div v-if="totalPages > 0" class="flex justify-end p-3 border-t border-base-200">
				<div class="join">
					<button
						v-for="p in totalPages"
						:key="p"
						class="join-item btn btn-sm"
						:class="p === pagination.current ? 'btn-primary' : 'btn-ghost'"
						@click="goPage(p)"
					>
						{{ p }}
					</button>
				</div>
			</div>
		</div>

		<!-- Create modal -->
		<dialog ref="createModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">新建项目</h3>
				<div v-if="modalStatus === 'loading'" class="flex justify-center py-8">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<div v-else class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">项目名称</legend>
						<input
							v-model="form.name"
							type="text"
							class="input w-full"
							:class="{ 'input-error': errors.name }"
							placeholder="例如: my-backend"
						/>
						<p v-if="errors.name" class="fieldset-label text-error">{{ errors.name }}</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">仓库地址</legend>
						<input
							v-model="form.repository_url"
							type="text"
							class="input w-full"
							:class="{ 'input-error': errors.repository_url }"
							placeholder="git@github.com:user/repo.git"
						/>
						<p v-if="errors.repository_url" class="fieldset-label text-error">
							{{ errors.repository_url }}
						</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">流水线快照</legend>
						<div class="flex gap-2">
							<select
								v-model="form.selectedTemplateId"
								class="select w-full"
								@change="onTemplateChange"
							>
								<option value="" disabled>选择模板</option>
								<option v-for="tpl in templates" :key="tpl.id" :value="tpl.id">
									{{ tpl.name }}
								</option>
							</select>
							<select
								v-model="form.pipeline_snapshot_id"
								class="select w-full"
								:class="{ 'select-error': errors.pipeline_snapshot_id }"
								:disabled="!form.selectedTemplateId"
							>
								<option value="" disabled>选择版本</option>
								<option v-for="snap in snapshots" :key="snap.id" :value="snap.id">
									v{{ snap.version }}
								</option>
							</select>
						</div>
						<p v-if="errors.pipeline_snapshot_id" class="fieldset-label text-error">
							{{ errors.pipeline_snapshot_id }}
						</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">Git 凭据（可选）</legend>
						<select v-model="form.git_credential_id" class="select w-full">
							<option value="">不使用凭据</option>
							<option v-for="cred in gitCredentials" :key="cred.id" :value="cred.id">
								{{ cred.name }}
							</option>
						</select>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">分支过滤（可选）</legend>
						<input
							v-model="form.branch_filter"
							type="text"
							class="input w-full"
							placeholder="main,develop（留空表示所有分支）"
						/>
					</fieldset>
				</div>
				<div class="modal-action">
					<button
						class="btn btn-primary"
						:disabled="operating || modalStatus === 'loading'"
						@click="handleCreateOk"
					>
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						创建
					</button>
					<button class="btn btn-ghost" @click="createModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Plus } from 'lucide-vue-next';
import { credentialApi, pipelineTemplateApi, projectApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { formatTime } from '@/utils/time';
import type { Credential, PipelineSnapshotListItem, PipelineTemplate, Project } from '@/types/api';

const toast = useToast();
const { status, error, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const { status: modalStatus, execute: executeModal } = useStatusAsync();

const projects = ref<Project[]>([]);
const templates = ref<PipelineTemplate[]>([]);
const credentials = ref<Credential[]>([]);
const snapshots = ref<PipelineSnapshotListItem[]>([]);
const gitCredentials = computed(() =>
	credentials.value.filter((c) => c.type === 'git_ssh' || c.type === 'git_token')
);

const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

const createModalRef = ref<HTMLDialogElement>();

const form = reactive({
	name: '',
	repository_url: '',
	pipeline_snapshot_id: '',
	selectedTemplateId: '',
	git_credential_id: '',
	branch_filter: '',
});
const errors = reactive({ name: '', repository_url: '', pipeline_snapshot_id: '' });

function validate() {
	errors.name = form.name.trim() ? '' : '请输入项目名称';
	errors.repository_url = form.repository_url.trim() ? '' : '请输入仓库地址';
	errors.pipeline_snapshot_id = form.pipeline_snapshot_id ? '' : '请选择流水线快照';
	return !errors.name && !errors.repository_url && !errors.pipeline_snapshot_id;
}

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
	} catch {
		toast.error('获取项目列表失败');
	}
}

function goPage(p: number) {
	pagination.current = p;
	fetchProjects();
}

async function openCreateModal() {
	Object.assign(form, {
		name: '',
		repository_url: '',
		pipeline_snapshot_id: '',
		selectedTemplateId: '',
		git_credential_id: '',
		branch_filter: '',
	});
	Object.assign(errors, { name: '', repository_url: '', pipeline_snapshot_id: '' });
	createModalRef.value?.showModal();
	try {
		await executeModal(async () => {
			const [tplRes, credRes] = await Promise.all([
				pipelineTemplateApi.list({ per_page: 100 }),
				credentialApi.list({ per_page: 100 }),
			]);
			templates.value = tplRes.items;
			credentials.value = credRes.items;
			snapshots.value = [];
		});
	} catch {
		toast.error('加载表单数据失败');
	}
}

async function onTemplateChange() {
	snapshots.value = [];
	form.pipeline_snapshot_id = '';
	if (form.selectedTemplateId) {
		snapshots.value = await pipelineTemplateApi.listSnapshots(form.selectedTemplateId);
	}
}

async function handleCreateOk() {
	if (!validate()) return;
	try {
		await executeOp(async () => {
			await projectApi.create({
				name: form.name,
				repository_url: form.repository_url,
				pipeline_snapshot_id: form.pipeline_snapshot_id,
				git_credential_id: form.git_credential_id || undefined,
			});
			toast.success('创建成功');
			createModalRef.value?.close();
			fetchProjects();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '创建失败');
	}
}

onMounted(fetchProjects);
</script>
