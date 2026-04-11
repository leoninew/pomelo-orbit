<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">{{ credential?.name ?? '凭据详情' }}</h1>
			<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/ci/credential')">
				<ArrowLeft class="size-4" />
				返回
			</button>
		</div>

		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<div class="flex items-center justify-between mb-4">
					<h2 class="font-semibold">基本信息</h2>
					<div v-if="credential" class="flex items-center gap-2">
						<button class="btn btn-sm btn-ghost" @click="openEditModal">编辑</button>
						<button class="btn btn-sm btn-ghost gap-1" @click="handleExport">
							<Download class="size-3.5" />
							导出
						</button>
						<button class="btn btn-sm btn-error btn-ghost" @click="openDeleteModal">删除</button>
					</div>
				</div>

				<div v-if="loading" class="flex justify-center py-8">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<dl v-else-if="credential" class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">凭据名称</dt>
						<dd>{{ credential.name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">类型</dt>
						<dd>
							<span class="badge badge-sm badge-ghost">
								{{ credentialTypeLabels[credential.type] ?? credential.type }}
							</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">创建时间</dt>
						<dd class="text-base-content/70">{{ formatTime(credential.created_at) }}</dd>
					</div>
				</dl>
			</div>
		</div>

		<!-- Edit modal -->
		<dialog ref="editModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">编辑凭据</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">凭据名称</legend>
						<input
							v-model="form.name"
							type="text"
							class="input w-full"
							:class="{ 'input-error': errors.name }"
						/>
						<p v-if="errors.name" class="fieldset-label text-error">{{ errors.name }}</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">
							凭据内容
							<span class="text-base-content/60 font-normal">（留空则不修改）</span>
						</legend>
						<textarea
							v-model="form.data"
							class="textarea w-full font-mono text-xs"
							rows="8"
							:placeholder="credential ? getDataPlaceholder(credential.type) : ''"
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
import { ArrowLeft, Download } from 'lucide-vue-next';
import { onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { credentialApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { Credential } from '@/types/ci/credential';
import { credentialTypeLabels } from '@/types/ci/credential';
import { formatTime } from '@/utils/time';

const props = defineProps<{ id: string }>();
const $router = useRouter();
const toast = useToast();
const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const credential = ref<Credential>();
const editModalRef = ref<HTMLDialogElement>();
const deleteModalRef = ref<HTMLDialogElement>();
const form = reactive({ name: '', data: '' });
const errors = reactive({ name: '' });

async function fetchCredential() {
	try {
		await execute(async () => {
			credential.value = await credentialApi.get(props.id);
		});
	} catch {
		toast.error('获取凭据详情失败');
	}
}

function openEditModal() {
	Object.assign(form, { name: credential.value?.name ?? '', data: '' });
	Object.assign(errors, { name: '' });
	editModalRef.value?.showModal();
}

async function handleEditOk() {
	errors.name = form.name.trim() ? '' : '请输入凭据名称';
	if (errors.name) {
		return;
	}
	try {
		await executeOp(async () => {
			await credentialApi.update(props.id, {
				name: form.name,
				...(form.data ? { data: form.data } : {}),
			});
			toast.success('更新成功');
			editModalRef.value?.close();
			fetchCredential();
		});
	} catch (err) {
		toast.error(err instanceof Error ? err.message : '操作失败');
	}
}

function openDeleteModal() {
	deleteModalRef.value?.showModal();
}

async function handleDelete() {
	try {
		await executeOp(async () => {
			await credentialApi.delete(props.id);
			toast.success('删除成功');
			$router.push('/ci/credential');
		});
	} catch (err) {
		toast.error(err instanceof Error ? err.message : '删除失败');
	}
}

async function handleExport() {
	try {
		const data = await credentialApi.exportCredential(props.id);
		const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `${credential.value?.name ?? 'credential'}.json`;
		a.click();
		URL.revokeObjectURL(url);
	} catch (err) {
		toast.error(err instanceof Error ? err.message : '导出失败');
	}
}

function getDataPlaceholder(type: string) {
	if (type === 'git_ssh') {
		return '-----BEGIN OPENSSH PRIVATE KEY-----\n...';
	}
	if (type === 'git_token') {
		return 'ghp_xxxxxxxxxxxxxxxxxxxx';
	}
	return 'registry_token_here';
}

onMounted(fetchCredential);
</script>
