<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">凭据管理</h1>
			<button class="btn btn-sm btn-primary gap-1.5" @click="openCreateModal">
				<Plus class="size-4" />
				新建凭据
			</button>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table min-h-48">
				<thead>
					<tr class="text-base-content/60">
						<th>凭据名称</th>
						<th>类型</th>
						<th>创建时间</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					<tr v-if="status === 'loading'">
						<td colspan="4" class="text-center py-8">
							<span class="loading loading-spinner loading-md text-primary" />
						</td>
					</tr>
					<tr v-else-if="status === 'error'">
						<td colspan="4" class="text-center py-8 text-error">{{ error }}</td>
					</tr>
					<tr v-else-if="credentials.length === 0">
						<td colspan="4" class="text-center py-8 text-base-content/60">暂无凭据</td>
					</tr>
					<tr v-for="c in credentials" :key="c.id" class="hover">
						<td class="font-medium">{{ c.name }}</td>
						<td>
							<span class="badge badge-sm badge-ghost">
								{{ credentialTypeLabels[c.type] ?? c.type }}
							</span>
						</td>
						<td class="cell-muted">{{ formatTime(c.created_at) }}</td>
						<td>
							<div class="flex items-center gap-2">
								<button class="link link-primary" @click="openEditModal(c)">编辑</button>
								<button class="link link-error" @click="confirmDelete(c.id)">删除</button>
							</div>
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

		<!-- Create/Edit modal -->
		<dialog ref="modalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">{{ isEditing ? '编辑凭据' : '新建凭据' }}</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">凭据名称</legend>
						<input
							v-model="form.name"
							type="text"
							class="input w-full"
							:class="{ 'input-error': errors.name }"
							placeholder="例如: GitHub SSH Key"
						/>
						<p v-if="errors.name" class="fieldset-label text-error">{{ errors.name }}</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">凭据类型</legend>
						<select v-model="form.type" class="select w-full" :disabled="isEditing">
							<option value="git_ssh">Git SSH</option>
							<option value="git_token">Git Token</option>
							<option value="registry_token">Registry Token</option>
						</select>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">
							凭据内容
							<span v-if="isEditing" class="text-base-content/60 font-normal">
								（留空则不修改）
							</span>
						</legend>
						<textarea
							v-model="form.data"
							class="textarea w-full font-mono text-xs"
							rows="8"
							:class="{ 'textarea-error': errors.data }"
							:placeholder="getDataPlaceholder(form.type)"
						/>
						<p v-if="errors.data" class="fieldset-label text-error">{{ errors.data }}</p>
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleModalOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						保存
					</button>
					<button class="btn btn-ghost" @click="modalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Delete confirm modal -->
		<dialog ref="deleteModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">删除凭据</h3>
				<p class="py-4 text-sm">确定删除此凭据？</p>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="operating" @click="handleDelete">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						删除
					</button>
					<button class="btn btn-ghost" @click="deleteModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { Plus } from 'lucide-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';
import { credentialApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { Credential } from '@/types/api';
import { credentialTypeLabels } from '@/types/api';
import { formatTime } from '@/utils/time';

const toast = useToast();
const { status, error, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const credentials = ref<Credential[]>([]);
const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

const modalRef = ref<HTMLDialogElement>();
const deleteModalRef = ref<HTMLDialogElement>();
const isEditing = ref(false);
const currentId = ref('');
const pendingDeleteId = ref('');

const form = reactive({ name: '', type: 'git_ssh' as string, data: '' });
const errors = reactive({ name: '', data: '' });

function validate() {
	errors.name = form.name.trim() ? '' : '请输入凭据名称';
	errors.data = !isEditing.value && !form.data.trim() ? '请输入凭据内容' : '';
	return !errors.name && !errors.data;
}

async function fetchCredentials() {
	try {
		await execute(async () => {
			const res = await credentialApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
			});
			credentials.value = res.items;
			pagination.total = res.total;
		});
	} catch {
		toast.error('获取凭据列表失败');
	}
}

function goPage(p: number) {
	pagination.current = p;
	fetchCredentials();
}

function openCreateModal() {
	isEditing.value = false;
	currentId.value = '';
	Object.assign(form, { name: '', type: 'git_ssh', data: '' });
	Object.assign(errors, { name: '', data: '' });
	modalRef.value?.showModal();
}

function openEditModal(record: Credential) {
	isEditing.value = true;
	currentId.value = record.id;
	Object.assign(form, { name: record.name, type: record.type, data: '' });
	Object.assign(errors, { name: '', data: '' });
	modalRef.value?.showModal();
}

async function handleModalOk() {
	if (!validate()) return;
	try {
		await executeOp(async () => {
			if (isEditing.value) {
				await credentialApi.update(currentId.value, {
					name: form.name,
					...(form.data ? { data: form.data } : {}),
				});
				toast.success('更新成功');
			} else {
				await credentialApi.create({
					name: form.name,
					type: form.type,
					data: form.data,
				});
				toast.success('创建成功');
			}
			modalRef.value?.close();
			fetchCredentials();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '操作失败');
	}
}

function confirmDelete(id: string) {
	pendingDeleteId.value = id;
	deleteModalRef.value?.showModal();
}

async function handleDelete() {
	try {
		await executeOp(async () => {
			await credentialApi.delete(pendingDeleteId.value);
			toast.success('删除成功');
			deleteModalRef.value?.close();
			fetchCredentials();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '删除失败');
	}
}

function getDataPlaceholder(type: string) {
	if (type === 'git_ssh') return '-----BEGIN OPENSSH PRIVATE KEY-----\n...';
	if (type === 'git_token') return 'ghp_xxxxxxxxxxxxxxxxxxxx';
	return 'registry_token_here';
}

onMounted(fetchCredentials);
</script>
