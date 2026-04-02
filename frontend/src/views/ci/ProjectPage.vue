<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">项目管理</h1>
			<button class="btn btn-sm btn-primary gap-1.5" @click="openCreateModal">
				<Plus class="size-4" />新建项目
			</button>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table table-sm">
				<thead>
					<tr class="text-base-content/60">
						<th>项目名称</th>
						<th>仓库地址</th>
						<th>创建时间</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					<tr v-if="loading"><td colspan="4" class="text-center py-8"><span class="loading loading-spinner loading-md text-primary" /></td></tr>
					<tr v-else-if="projects.length === 0"><td colspan="4" class="text-center py-8 text-base-content/40">暂无项目</td></tr>
					<tr v-for="p in projects" :key="p.id" class="hover">
						<td><router-link :to="`/ci/projects/${p.id}`" class="link link-primary font-medium">{{ p.name }}</router-link></td>
						<td class="cell-muted max-w-xs truncate">{{ p.repository_url }}</td>
						<td class="cell-muted">{{ formatTime(p.created_at) }}</td>
						<td>
							<div class="flex items-center gap-2">
								<router-link :to="`/ci/projects/${p.id}`" class="link link-primary">查看</router-link>
								<button class="link link-info" @click="handleTrigger(p.id)">触发</button>
								<button class="link link-error" @click="confirmDelete(p.id)">删除</button>
							</div>
						</td>
					</tr>
				</tbody>
			</table>
			<div v-if="pagination.total > pagination.pageSize" class="flex justify-end p-3 border-t border-base-200">
				<div class="join">
					<button v-for="p in totalPages" :key="p" class="join-item btn btn-sm" :class="p === pagination.current ? 'btn-primary' : 'btn-ghost'" @click="goPage(p)">{{ p }}</button>
				</div>
			</div>
		</div>

		<!-- Create modal -->
		<dialog ref="createModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">新建项目</h3>
				<div class="flex flex-col gap-3">
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">项目名称</span></div>
						<input v-model="form.name" type="text" class="input input-bordered input-sm" :class="{ 'input-error': errors.name }" placeholder="例如: my-backend" />
						<div v-if="errors.name" class="label pt-1"><span class="label-text-alt text-error">{{ errors.name }}</span></div>
					</label>
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">仓库地址</span></div>
						<input v-model="form.repository_url" type="text" class="input input-bordered input-sm" :class="{ 'input-error': errors.repository_url }" placeholder="git@github.com:user/repo.git" />
						<div v-if="errors.repository_url" class="label pt-1"><span class="label-text-alt text-error">{{ errors.repository_url }}</span></div>
					</label>
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">流水线模板</span></div>
						<select v-model="form.pipeline_template_id" class="select select-bordered select-sm" :class="{ 'select-error': errors.pipeline_template_id }">
							<option value="" disabled>选择模板</option>
							<option v-for="tpl in templates" :key="tpl.id" :value="tpl.id">{{ tpl.name }}</option>
						</select>
						<div v-if="errors.pipeline_template_id" class="label pt-1"><span class="label-text-alt text-error">{{ errors.pipeline_template_id }}</span></div>
					</label>
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">Git 凭据（可选）</span></div>
						<select v-model="form.git_credential_id" class="select select-bordered select-sm">
							<option value="">不使用凭据</option>
							<option v-for="cred in gitCredentials" :key="cred.id" :value="cred.id">{{ cred.name }}</option>
						</select>
					</label>
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">分支过滤（可选）</span></div>
						<input v-model="form.branch_filter" type="text" class="input input-bordered input-sm" placeholder="main,develop（留空表示所有分支）" />
					</label>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleCreateOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />创建
					</button>
					<button class="btn btn-ghost" @click="createModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Delete confirm modal -->
		<dialog ref="deleteModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">删除项目</h3>
				<p class="py-4">确定删除此项目？此操作不可撤销。</p>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="operating" @click="handleDelete">
						<span v-if="operating" class="loading loading-spinner loading-xs" />删除
					</button>
					<button class="btn btn-ghost" @click="deleteModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Plus } from 'lucide-vue-next';
import { ciCredentialApi, pipelineTemplateApi, projectApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { formatTime } from '@/utils/time';
import type { Credential, PipelineTemplate, Project } from '@/types/api';

const router = useRouter();
const toast = useToast();
const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const projects = ref<Project[]>([]);
const templates = ref<PipelineTemplate[]>([]);
const credentials = ref<Credential[]>([]);
const gitCredentials = computed(() => credentials.value.filter((c) => c.type === 'git_ssh' || c.type === 'git_token'));

const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

const createModalRef = ref<HTMLDialogElement>();
const deleteModalRef = ref<HTMLDialogElement>();
const pendingDeleteId = ref('');

const form = reactive({ name: '', repository_url: '', pipeline_template_id: '', git_credential_id: '', branch_filter: '' });
const errors = reactive({ name: '', repository_url: '', pipeline_template_id: '' });

function validate() {
	errors.name = form.name.trim() ? '' : '请输入项目名称';
	errors.repository_url = form.repository_url.trim() ? '' : '请输入仓库地址';
	errors.pipeline_template_id = form.pipeline_template_id ? '' : '请选择流水线模板';
	return !errors.name && !errors.repository_url && !errors.pipeline_template_id;
}

async function fetchProjects() {
	try {
		await execute(async () => {
			const res = await projectApi.list({ page: pagination.current, per_page: pagination.pageSize });
			projects.value = res.items; pagination.total = res.total;
		});
	} catch { toast.error('获取项目列表失败'); }
}

function goPage(p: number) { pagination.current = p; fetchProjects(); }

async function openCreateModal() {
	Object.assign(form, { name: '', repository_url: '', pipeline_template_id: '', git_credential_id: '', branch_filter: '' });
	Object.assign(errors, { name: '', repository_url: '', pipeline_template_id: '' });
	// Load templates and credentials
	const [tplRes, credRes] = await Promise.all([
		pipelineTemplateApi.list({ per_page: 100 }),
		ciCredentialApi.list({ per_page: 100 }),
	]);
	templates.value = tplRes.items;
	credentials.value = credRes.items;
	createModalRef.value?.showModal();
}

async function handleCreateOk() {
	if (!validate()) return;
	try {
		await executeOp(async () => {
			await projectApi.create({ name: form.name, repository_url: form.repository_url, pipeline_template_id: form.pipeline_template_id, git_credential_id: form.git_credential_id || undefined, branch_filter: form.branch_filter || undefined });
			toast.success('创建成功'); createModalRef.value?.close(); fetchProjects();
		});
	} catch (error) { toast.error(error instanceof Error ? error.message : '创建失败'); }
}

async function handleTrigger(id: string) {
	try {
		await executeOp(async () => {
			const run = await projectApi.trigger(id);
			toast.success(`触发成功`);
			router.push(`/ci/runs/${run.id}`);
		});
	} catch (error) { toast.error(error instanceof Error ? error.message : '触发失败'); }
}

function confirmDelete(id: string) { pendingDeleteId.value = id; deleteModalRef.value?.showModal(); }

async function handleDelete() {
	try {
		await executeOp(async () => {
			await projectApi.delete(pendingDeleteId.value);
			toast.success('删除成功'); deleteModalRef.value?.close(); fetchProjects();
		});
	} catch (error) { toast.error(error instanceof Error ? error.message : '删除失败'); }
}

onMounted(fetchProjects);
</script>
