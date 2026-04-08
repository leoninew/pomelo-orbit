<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">流水线模板</h1>
			<div class="flex items-center gap-2">
				<div class="join">
					<button
						class="join-item btn btn-sm"
						:class="viewMode === 'card' ? 'btn-primary' : 'btn-ghost'"
						@click="viewMode = 'card'"
					>
						<LayoutGrid class="size-4" />
					</button>
					<button
						class="join-item btn btn-sm"
						:class="viewMode === 'table' ? 'btn-primary' : 'btn-ghost'"
						@click="viewMode = 'table'"
					>
						<List class="size-4" />
					</button>
				</div>
				<button class="btn btn-sm btn-primary gap-1.5" @click="openCreateModal">
					<Plus class="size-4" />
					新建模板
				</button>
			</div>
		</div>

		<div v-if="status === 'loading'" class="flex justify-center py-16">
			<span class="loading loading-spinner loading-lg text-primary" />
		</div>
		<div v-else-if="status === 'error'" class="flex justify-center py-16 text-error text-sm">
			{{ error }}
		</div>

		<template v-else-if="viewMode === 'card'">
			<div
				v-if="templates.length === 0"
				class="flex flex-col items-center gap-2 py-16 text-base-content/60"
			>
				<Inbox class="size-12" />
				<span>暂无模板</span>
			</div>
			<div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
				<div
					v-for="tpl in templates"
					:key="tpl.id"
					class="card bg-base-100 shadow-sm border border-base-200 cursor-pointer hover:shadow-md transition-shadow overflow-hidden group"
					@click="$router.push(`/ci/templates/${tpl.id}`)"
				>
					<div class="card-body p-4 gap-3">
						<div class="flex items-start justify-between gap-2">
							<span class="font-semibold truncate">{{ tpl.name }}</span>
							<div class="flex items-center gap-1 shrink-0">
								<span
									v-if="tpl.latest_snapshot_version"
									class="badge badge-sm badge-ghost"
								>
									v{{ tpl.latest_snapshot_version }}
								</span>
								<span class="badge badge-sm badge-ghost">自定义</span>
							</div>
						</div>
						<p class="text-xs text-base-content/70 line-clamp-2">
							{{ tpl.description || '暂无描述' }}
						</p>
						<div class="flex items-end justify-between mt-auto">
							<span class="text-xs text-base-content/50">{{ formatTime(tpl.created_at) }}</span>
							<div class="flex items-center gap-1" @click.stop>
								<button
									class="btn btn-xs btn-ghost text-error invisible group-hover:visible"
									@click="confirmDelete(tpl.id)"
								>
									删除
								</button>
							</div>
						</div>
					</div>
				</div>
			</div>

			<div v-if="totalPages > 0" class="flex justify-end mt-4">
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
		</template>

		<div v-else class="card bg-base-100 shadow-sm overflow-x-auto">
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
					<tr v-if="templates.length === 0">
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
import { computed, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { LayoutGrid, List, Plus, Inbox } from 'lucide-vue-next';
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
const viewMode = ref<'card' | 'table'>('card');
const pagination = reactive({ current: 1, pageSize: 12, total: 0 });
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
	if (errors.name) {
		return;
	}
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
