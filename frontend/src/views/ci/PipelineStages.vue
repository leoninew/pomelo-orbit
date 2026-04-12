<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">流水线阶段</h1>
			<button class="btn btn-sm btn-primary gap-1.5" @click="openCreateModal">
				<Plus class="size-4" />
				新建 Stage
			</button>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table min-h-48">
				<thead>
					<tr class="text-base-content/60">
						<th>名称</th>
						<th>镜像</th>
						<th>版本</th>
						<th>描述</th>
						<th>更新时间</th>
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
					<tr v-else-if="stages.length === 0">
						<td colspan="5" class="text-center py-8 text-base-content/60">暂无数据</td>
					</tr>
					<tr v-for="s in stages" :key="s.id" class="hover">
						<td>
							<router-link :to="`/ci/pipeline-stage/${s.id}`" class="link link-primary font-medium">
								{{ s.name }}
							</router-link>
						</td>
						<td class="font-mono text-base-content/70">{{ s.image }}</td>
						<td>
							<span class="badge badge-sm badge-ghost">v{{ s.version }}</span>
						</td>
						<td class="text-base-content/60 max-w-xs truncate">{{ s.description || '—' }}</td>
						<td class="text-base-content/60">{{ formatTime(s.updated_at) }}</td>
						<td>
							<router-link :to="`/ci/pipeline-stage/${s.id}`" class="link link-primary">
								查看
							</router-link>
							<button
								class="link link-primary ml-3"
								:disabled="duplicating"
								@click="handleDuplicate(s.id)"
							>
								复制
							</button>
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
		<dialog ref="modalRef" class="modal">
			<div class="modal-box w-full max-w-2xl">
				<h3 class="font-bold text-lg mb-4">新建 Stage</h3>
				<div class="flex flex-col gap-3">
					<div class="grid grid-cols-2 gap-3">
						<fieldset class="fieldset">
							<legend class="fieldset-legend">名称</legend>
							<input
								v-model="form.name"
								type="text"
								class="input w-full"
								:class="{ 'input-error': errors.name }"
								placeholder="例如: build"
							/>
							<p v-if="errors.name" class="fieldset-label text-error">{{ errors.name }}</p>
						</fieldset>
						<fieldset class="fieldset">
							<legend class="fieldset-legend">镜像</legend>
							<input
								v-model="form.image"
								type="text"
								class="input w-full"
								:class="{ 'input-error': errors.image }"
								placeholder="例如: alpine:latest"
							/>
							<p v-if="errors.image" class="fieldset-label text-error">{{ errors.image }}</p>
						</fieldset>
					</div>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">脚本</legend>
						<textarea
							v-model="form.script"
							class="textarea w-full font-mono text-xs"
							:class="{ 'textarea-error': errors.script }"
							rows="6"
							placeholder="echo hello&#10;echo world"
						/>
						<p v-if="errors.script" class="fieldset-label text-error">{{ errors.script }}</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">描述（可选）</legend>
						<input
							v-model="form.description"
							type="text"
							class="input w-full"
							placeholder="简短描述"
						/>
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
	</div>
</template>

<script setup lang="ts">
import { Plus } from 'lucide-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { pipelineStageApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { PipelineStage } from '@/types/ci/template';
import { formatTime } from '@/utils/time';

const router = useRouter();
const toast = useToast();
const { status, error, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const { loading: duplicating, execute: executeDuplicate } = useStatusAsync();

const stages = ref<PipelineStage[]>([]);
const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
const modalRef = ref<HTMLDialogElement>();

const form = reactive({ name: '', image: '', script: '', description: '' });
const errors = reactive({ name: '', image: '', script: '' });

function validate() {
	errors.name = form.name.trim() ? '' : '请输入名称';
	errors.image = form.image.trim() ? '' : '请输入镜像';
	errors.script = form.script.trim() ? '' : '请输入脚本';
	return !errors.name && !errors.image && !errors.script;
}

async function fetchStages() {
	try {
		await execute(async () => {
			const res = await pipelineStageApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
			});
			stages.value = res.items;
			pagination.total = res.total;
		});
	} catch {
		toast.error('获取 Stage 列表失败');
	}
}

function goPage(p: number) {
	pagination.current = p;
	fetchStages();
}

function openCreateModal() {
	Object.assign(form, { name: '', image: '', script: '', description: '' });
	Object.assign(errors, { name: '', image: '', script: '' });
	modalRef.value?.showModal();
}

async function handleModalOk() {
	if (!validate()) {
		return;
	}
	try {
		await executeOp(async () => {
			await pipelineStageApi.create({
				name: form.name,
				image: form.image,
				script: form.script,
				description: form.description,
			});
			toast.success('创建成功');
			modalRef.value?.close();
			fetchStages();
		});
	} catch (err) {
		toast.error(err instanceof Error ? err.message : '操作失败');
	}
}

async function handleDuplicate(id: string) {
	try {
		await executeDuplicate(async () => {
			const newStage = await pipelineStageApi.duplicate(id);
			toast.success('复制成功');
			router.push(`/ci/pipeline-stage/${newStage.id}`);
		});
	} catch (err) {
		toast.error(err instanceof Error ? err.message : '复制失败');
	}
}

onMounted(fetchStages);
</script>
