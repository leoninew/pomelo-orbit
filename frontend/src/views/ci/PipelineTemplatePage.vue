<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">流水线模板</h1>
			<button class="btn btn-sm btn-primary gap-1.5" @click="openCreateModal">
				<Plus class="size-4" />
				新建模板
			</button>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table min-h-48">
				<thead>
					<tr class="text-base-content/60">
						<th>模板名称</th>
						<th>类型</th>
						<th>描述</th>
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
					<tr v-else-if="templates.length === 0">
						<td colspan="5" class="text-center py-8 text-base-content/60">暂无模板</td>
					</tr>
					<tr v-for="t in templates" :key="t.id" class="hover">
						<td>
							<router-link :to="`/ci/templates/${t.id}`" class="link link-primary font-medium">
								{{ t.name }}
							</router-link>
						</td>
						<td>
							<span class="badge badge-sm badge-ghost">自定义</span>
						</td>
						<td class="cell-muted max-w-xs truncate">{{ t.description || '—' }}</td>
						<td class="cell-muted">{{ formatTime(t.created_at) }}</td>
						<td>
							<div class="flex items-center gap-2">
								<router-link :to="`/ci/templates/${t.id}`" class="link link-primary">
									查看
								</router-link>
								<button class="link link-error" @click="confirmDelete(t.id)">删除</button>
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

		<!-- Create modal -->
		<dialog ref="createModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">新建模板</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">模板名称</legend>
						<input
							v-model="form.name"
							type="text"
							class="input w-full"
							:class="{ 'input-error': errors.name }"
							placeholder="例如: Python FastAPI 构建"
						/>
						<p v-if="errors.name" class="fieldset-label text-error">{{ errors.name }}</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">描述（可选）</legend>
						<textarea v-model="form.description" class="textarea w-full" rows="2" />
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleCreateOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						创建
					</button>
					<button class="btn btn-ghost" @click="createModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Delete confirm modal -->
		<dialog ref="deleteModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">删除模板</h3>
				<p class="py-4 text-sm">确定删除此模板？</p>
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
import { useRouter } from 'vue-router';
import { pipelineTemplateApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { PipelineTemplate } from '@/types/api';
import { formatTime } from '@/utils/time';

const router = useRouter();
const toast = useToast();
const { status, error, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const templates = ref<PipelineTemplate[]>([]);
const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

const createModalRef = ref<HTMLDialogElement>();
const deleteModalRef = ref<HTMLDialogElement>();
const pendingDeleteId = ref('');

const form = reactive({ name: '', description: '' });
const errors = reactive({ name: '' });

async function fetchTemplates() {
	try {
		await execute(async () => {
			const res = await pipelineTemplateApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
			});
			templates.value = res.items;
			pagination.total = res.total;
		});
	} catch {
		toast.error('获取模板列表失败');
	}
}

function goPage(p: number) {
	pagination.current = p;
	fetchTemplates();
}

function openCreateModal() {
	Object.assign(form, { name: '', description: '' });
	Object.assign(errors, { name: '' });
	createModalRef.value?.showModal();
}

async function handleCreateOk() {
	errors.name = form.name.trim() ? '' : '请输入模板名称';
	if (errors.name) return;
	try {
		await executeOp(async () => {
			const tpl = await pipelineTemplateApi.create({
				name: form.name,
				description: form.description || undefined,
				stages: [],
				variable_declarations: [],
			});
			toast.success('创建成功');
			createModalRef.value?.close();
			router.push(`/ci/templates/${tpl.id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '创建失败');
	}
}

function confirmDelete(id: string) {
	pendingDeleteId.value = id;
	deleteModalRef.value?.showModal();
}

async function handleDelete() {
	try {
		await executeOp(async () => {
			await pipelineTemplateApi.delete(pendingDeleteId.value);
			toast.success('删除成功');
			deleteModalRef.value?.close();
			fetchTemplates();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '删除失败');
	}
}

onMounted(fetchTemplates);
</script>
