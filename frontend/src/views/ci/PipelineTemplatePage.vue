<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">流水线模板</h1>
			<button class="btn btn-sm btn-primary gap-1.5" @click="openCreateModal">
				<Plus class="size-4" />新建模板
			</button>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table">
				<thead>
					<tr class="text-base-content/60">
						<th>模板名称</th>
						<th>描述</th>
						<th>变量</th>
						<th>创建时间</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					<tr v-if="loading"><td colspan="5" class="text-center py-8"><span class="loading loading-spinner loading-md text-primary" /></td></tr>
					<tr v-else-if="templates.length === 0"><td colspan="5" class="text-center py-8 text-base-content/40">暂无模板</td></tr>
					<tr v-for="t in templates" :key="t.id" class="hover">
						<td>
							<div class="flex items-center gap-2">
								<router-link :to="`/ci/templates/${t.id}`" class="link link-primary font-medium">{{ t.name }}</router-link>
								<span v-if="t.is_builtin" class="badge badge-xs badge-info">内置</span>
							</div>
						</td>
						<td class="cell-muted max-w-xs truncate">{{ t.description || '—' }}</td>
						<td class="cell-muted">{{ (t.variable_declarations ?? []).length }} 个</td>
						<td class="cell-muted">{{ formatTime(t.created_at) }}</td>
						<td>
							<div class="flex items-center gap-2">
								<router-link :to="`/ci/templates/${t.id}`" class="link link-primary">查看</router-link>
								<button v-if="!t.is_builtin" class="link link-error" @click="confirmDelete(t.id)">删除</button>
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
			<div class="modal-box w-full max-w-2xl">
				<h3 class="font-bold text-lg mb-4">新建模板</h3>
				<div class="flex flex-col gap-3">
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">模板名称</span></div>
						<input v-model="form.name" type="text" class="input input-bordered input-sm" :class="{ 'input-error': errors.name }" placeholder="例如: Python FastAPI 构建" />
						<div v-if="errors.name" class="label pt-1"><span class="label-text-alt text-error">{{ errors.name }}</span></div>
					</label>
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">描述（可选）</span></div>
						<textarea v-model="form.description" class="textarea textarea-bordered textarea-sm" rows="2" />
					</label>
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">Pipeline YAML</span></div>
						<textarea v-model="form.content" class="textarea textarea-bordered textarea-sm font-mono text-xs" rows="12" :class="{ 'textarea-error': errors.content }" placeholder="version: v1&#10;steps:&#10;  - name: checkout&#10;    uses: checkout" />
						<div v-if="errors.content" class="label pt-1"><span class="label-text-alt text-error">{{ errors.content }}</span></div>
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
				<h3 class="font-bold text-lg">删除模板</h3>
				<p class="py-4">确定删除此模板？</p>
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
import { Plus } from 'lucide-vue-next';
import { pipelineTemplateApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { formatTime } from '@/utils/time';
import type { PipelineTemplate } from '@/types/api';

const toast = useToast();
const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const templates = ref<PipelineTemplate[]>([]);
const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

const createModalRef = ref<HTMLDialogElement>();
const deleteModalRef = ref<HTMLDialogElement>();
const pendingDeleteId = ref('');

const form = reactive({ name: '', description: '', content: '' });
const errors = reactive({ name: '', content: '' });

async function fetchTemplates() {
	try {
		await execute(async () => {
			const res = await pipelineTemplateApi.list({ page: pagination.current, per_page: pagination.pageSize });
			templates.value = res.items; pagination.total = res.total;
		});
	} catch { toast.error('获取模板列表失败'); }
}

function goPage(p: number) { pagination.current = p; fetchTemplates(); }

function openCreateModal() {
	Object.assign(form, { name: '', description: '', content: '' });
	Object.assign(errors, { name: '', content: '' });
	createModalRef.value?.showModal();
}

async function handleCreateOk() {
	errors.name = form.name.trim() ? '' : '请输入模板名称';
	errors.content = form.content.trim() ? '' : '请输入 Pipeline YAML';
	if (errors.name || errors.content) return;
	try {
		await executeOp(async () => {
			await pipelineTemplateApi.create({ name: form.name, description: form.description || undefined, content: form.content });
			toast.success('创建成功'); createModalRef.value?.close(); fetchTemplates();
		});
	} catch (error) { toast.error(error instanceof Error ? error.message : '创建失败'); }
}

function confirmDelete(id: string) { pendingDeleteId.value = id; deleteModalRef.value?.showModal(); }

async function handleDelete() {
	try {
		await executeOp(async () => {
			await pipelineTemplateApi.delete(pendingDeleteId.value);
			toast.success('删除成功'); deleteModalRef.value?.close(); fetchTemplates();
		});
	} catch (error) { toast.error(error instanceof Error ? error.message : '删除失败'); }
}

onMounted(fetchTemplates);
</script>
