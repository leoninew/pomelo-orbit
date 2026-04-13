<template>
	<div class="flex flex-col gap-4">
		<!-- Page header -->
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">{{ stage?.name ?? 'Stage 详情' }}</h1>
			<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/ci/build-stage')">
				<ArrowLeft class="size-4" />
				返回
			</button>
		</div>

		<div v-if="status === 'loading'" class="flex justify-center py-16">
			<span class="loading loading-spinner loading-lg text-primary" />
		</div>

		<template v-else-if="stage">
			<!-- 基本信息 -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<div class="flex items-center justify-between mb-4">
						<h2 class="font-semibold">基本信息</h2>
						<div v-if="stage" class="flex items-center gap-2">
							<button
								class="btn btn-sm btn-ghost"
								:disabled="saving || deleting || duplicating"
								@click="openEditModal"
							>
								编辑
							</button>
							<button
								class="btn btn-sm btn-ghost"
								:disabled="saving || deleting || duplicating"
								@click="handleDuplicate"
							>
								<span v-if="duplicating" class="loading loading-spinner loading-xs" />
								复制
							</button>
							<button
								class="btn btn-sm btn-error btn-ghost"
								:disabled="saving || deleting || duplicating"
								@click="openDeleteModal"
							>
								删除
							</button>
						</div>
					</div>
					<dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">名称</dt>
							<dd class="font-medium">{{ stage.name }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">镜像</dt>
							<dd>
								<code class="text-xs bg-base-200 px-1.5 py-0.5 rounded">{{ stage.image }}</code>
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">版本</dt>
							<dd>
								<span class="badge badge-sm badge-ghost">v{{ stage.version }}</span>
							</dd>
						</div>
						<div v-if="stage.description" class="flex gap-2 sm:col-span-2">
							<dt class="text-base-content/70 w-24 shrink-0">描述</dt>
							<dd>{{ stage.description }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">创建时间</dt>
							<dd class="text-base-content/60">{{ formatTime(stage.created_at) }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">更新时间</dt>
							<dd class="text-base-content/60">{{ formatTime(stage.updated_at) }}</dd>
						</div>
					</dl>
				</div>
			</div>

			<!-- 脚本 -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<div class="flex items-center justify-between mb-4">
						<h2 class="font-semibold">脚本</h2>
						<button class="btn btn-sm btn-primary gap-1" @click="openScriptDrawer">
							<FilePen class="size-3.5" />
							{{ stage.script ? '编辑' : '添加' }}
						</button>
					</div>
					<pre
						v-if="stage.script"
						class="bg-base-200 rounded p-4 font-mono text-xs leading-relaxed whitespace-pre-wrap break-all overflow-x-auto max-h-64"
						>{{ stage.script }}</pre
					>
					<p v-else class="text-sm text-base-content/60 py-4 text-center">暂无数据</p>
				</div>
			</div>

			<!-- 制品 -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<div class="flex items-center justify-between mb-4">
						<h2 class="font-semibold">制品</h2>
						<button class="btn btn-sm btn-primary gap-1" @click="openAddArtifactModal">
							<Plus class="size-3.5" />
							添加制品
						</button>
					</div>
					<div
						v-if="!stage.artifacts || stage.artifacts.length === 0"
						class="text-sm text-base-content/60 py-8 text-center"
					>
						暂无数据
					</div>
					<table v-else class="table w-full">
						<thead>
							<tr class="text-base-content/60 text-xs">
								<th class="w-6 pr-0"></th>
								<th class="w-8">#</th>
								<th class="w-32">类型</th>
								<th>名称</th>
								<th>路径/镜像</th>
								<th class="w-40">操作</th>
							</tr>
						</thead>
						<VueDraggable
							v-model="sortableArtifacts"
							tag="tbody"
							handle=".drag-handle"
							:animation="150"
							ghost-class="opacity-30"
						>
							<tr v-for="(a, idx) in sortableArtifacts" :key="idx" class="hover">
								<td class="pr-0 w-6">
									<GripVertical
										class="drag-handle size-4 text-base-content/30 hover:text-base-content/60 cursor-grab active:cursor-grabbing transition-colors"
									/>
								</td>
								<td class="text-base-content/40 text-xs">{{ idx + 1 }}</td>
								<td>
									<span class="badge badge-sm badge-ghost">{{ a.type }}</span>
								</td>
								<td class="text-sm">{{ a.name }}</td>
								<td class="text-sm">{{ a.path }}</td>
								<td>
									<div class="flex items-center gap-3">
										<button class="link link-primary text-xs" @click="openEditArtifactModal(idx)">
											编辑
										</button>
										<button class="link link-error text-xs" @click="confirmRemoveArtifact(idx)">
											删除
										</button>
									</div>
								</td>
							</tr>
						</VueDraggable>
					</table>
				</div>
			</div>
		</template>

		<!-- Edit basic info modal -->
		<dialog ref="editModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">编辑基本信息</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">名称</legend>
						<input v-model="form.name" type="text" class="input w-full" />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">镜像</legend>
						<input v-model="form.image" type="text" class="input w-full text-sm" />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">描述（可选）</legend>
						<input v-model="form.description" type="text" class="input w-full" />
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="saving" @click="handleSave">
						<span v-if="saving" class="loading loading-spinner loading-xs" />
						保存
					</button>
					<button class="btn btn-ghost" @click="editModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Script drawer -->
		<Teleport to="body">
			<Transition
				enter-active-class="transition-transform duration-300 ease-out"
				enter-from-class="translate-x-full"
				enter-to-class="translate-x-0"
				leave-active-class="transition-transform duration-300 ease-in"
				leave-from-class="translate-x-0"
				leave-to-class="translate-x-full"
			>
				<div
					v-if="showScriptDrawer"
					class="fixed inset-y-0 right-0 z-50 w-[720px] max-w-full bg-base-100 shadow-2xl flex flex-col border-l border-base-200"
				>
					<div
						class="flex items-center justify-between px-5 py-4 border-b border-base-200 shrink-0"
					>
						<h3 class="font-semibold">编辑脚本</h3>
						<button class="btn btn-sm btn-ghost btn-circle" @click="closeScriptDrawer">
							<X class="size-4" />
						</button>
					</div>
					<div class="flex-1 overflow-hidden p-5 flex flex-col gap-4 bg-[#1a202c]">
						<div style="height: calc(100vh - 140px)">
							<CodeEditor
								v-model:value="scriptTemp"
								:style="{ height: '100%' }"
								theme="vs-dark"
								language="shell"
								:options="{
									minimap: { enabled: false },
									fontSize: 14,
									automaticLayout: true,
								}"
							/>
						</div>
					</div>
					<div class="flex items-center justify-end gap-2 px-5 py-4 border-t border-base-200">
						<button class="btn btn-sm btn-primary" @click="confirmScript">确定</button>
						<button class="btn btn-sm btn-ghost" @click="closeScriptDrawer">取消</button>
					</div>
				</div>
			</Transition>
			<Transition
				enter-active-class="transition-opacity duration-300"
				enter-from-class="opacity-0"
				enter-to-class="opacity-100"
				leave-active-class="transition-opacity duration-300"
				leave-from-class="opacity-100"
				leave-to-class="opacity-0"
			>
				<div
					v-if="showScriptDrawer"
					class="fixed inset-0 z-40 bg-black/30"
					@click="closeScriptDrawer"
				/>
			</Transition>
		</Teleport>

		<!-- Delete modal -->
		<dialog ref="deleteModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">删除 Stage</h3>
				<p class="py-4 text-sm">
					确定要删除 Stage「
					<strong>{{ stage?.name ?? '' }}</strong>
					」吗？此操作不可撤销。
				</p>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="deleting" @click="handleDelete">
						<span v-if="deleting" class="loading loading-spinner loading-xs" />
						删除
					</button>
					<button class="btn btn-ghost" @click="deleteModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- 添加/编辑制品 modal -->
		<dialog ref="artifactModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">{{ artifactForm.isEdit ? '编辑制品' : '添加制品' }}</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">类型</legend>
						<select v-model="artifactForm.type" class="select w-full">
							<option value="docker_image">Docker 镜像</option>
							<option value="binary">二进制文件</option>
						</select>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">名称</legend>
						<input
							v-model="artifactForm.name"
							type="text"
							class="input w-full"
							placeholder="制品名称"
						/>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">路径/镜像</legend>
						<input
							v-model="artifactForm.path"
							type="text"
							class="input w-full"
							:placeholder="artifactForm.type === 'docker_image' ? 'image:tag' : 'dist/app'"
						/>
					</fieldset>
				</div>
				<div class="modal-action">
					<button
						class="btn btn-primary"
						:disabled="!artifactForm.name.trim() || !artifactForm.path.trim() || saving"
						@click="handleSaveArtifact"
					>
						<span v-if="saving" class="loading loading-spinner loading-xs" />
						保存
					</button>
					<button class="btn btn-ghost" @click="artifactModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- 删除制品确认 modal -->
		<dialog ref="deleteArtifactModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">删除制品</h3>
				<p class="py-4 text-sm">
					确定要删除制品「
					<strong>{{ sortableArtifacts[artifactToDelete]?.name }}</strong>
					」吗？此操作不可撤销。
				</p>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="saving" @click="removeArtifact">
						<span v-if="saving" class="loading loading-spinner loading-xs" />
						删除
					</button>
					<button
						class="btn btn-ghost"
						@click="
							deleteArtifactModalRef?.close();
							artifactToDelete = -1;
						"
					>
						取消
					</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { ArrowLeft, FilePen, GripVertical, Plus, X } from 'lucide-vue-next';
import { CodeEditor } from 'monaco-editor-vue3';
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { VueDraggable } from 'vue-draggable-plus';
import { useRoute, useRouter } from 'vue-router';
import { buildStageApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { ArtifactConfig, ArtifactType, BuildStage } from '@/types/ci/template';
import { formatTime } from '@/utils/time';

const route = useRoute();
const router = useRouter();
const stageId = computed(() => route.params.id as string);
const toast = useToast();

const { status, execute } = useStatusAsync();
const { loading: saving, execute: executeSave } = useStatusAsync();
const { loading: deleting, execute: executeDelete } = useStatusAsync();
const { loading: duplicating, execute: executeDuplicate } = useStatusAsync();

const stage = ref<BuildStage>();
const deleteModalRef = ref<HTMLDialogElement>();
const editModalRef = ref<HTMLDialogElement>();
const artifactModalRef = ref<HTMLDialogElement>();
const deleteArtifactModalRef = ref<HTMLDialogElement>();
const showScriptDrawer = ref(false);
const scriptTemp = ref('');
const form = reactive({ name: '', image: '', description: '' });
const artifactForm = reactive({
	isEdit: false,
	order: -1,
	type: 'docker_image' as ArtifactType,
	name: '',
	path: '',
});
const sortableArtifacts = ref<ArtifactConfig[]>([]);
const artifactToDelete = ref(-1);

async function fetchStage() {
	try {
		await execute(async () => {
			stage.value = await buildStageApi.get(stageId.value);
			sortableArtifacts.value = stage.value.artifacts ? [...stage.value.artifacts] : [];
		});
	} catch {
		toast.error('获取 Stage 失败');
		router.push('/ci/build-stage');
	}
}

function openEditModal() {
	if (!stage.value) {
		return;
	}
	Object.assign(form, {
		name: stage.value.name,
		image: stage.value.image,
		description: stage.value.description,
	});
	editModalRef.value?.showModal();
}

function openScriptDrawer() {
	scriptTemp.value = stage.value?.script ?? '';
	showScriptDrawer.value = true;
}

function closeScriptDrawer() {
	showScriptDrawer.value = false;
}

async function confirmScript() {
	try {
		await executeSave(async () => {
			const updated = await buildStageApi.update(stageId.value, {
				script: scriptTemp.value,
			});
			stage.value = updated;
			showScriptDrawer.value = false;
			toast.success('脚本已保存');
		});
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '保存失败');
	}
}

async function handleSave() {
	try {
		await executeSave(async () => {
			const updated = await buildStageApi.update(stageId.value, {
				name: form.name,
				image: form.image,
				description: form.description,
			});
			stage.value = updated;
			toast.success('更新成功');
			editModalRef.value?.close();
		});
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '保存失败');
	}
}

function openAddArtifactModal() {
	Object.assign(artifactForm, {
		isEdit: false,
		order: -1,
		type: 'docker_image',
		name: '',
		path: '',
	});
	artifactModalRef.value?.showModal();
}

function openEditArtifactModal(idx: number) {
	const artifact = sortableArtifacts.value[idx];
	if (!artifact) {
		return;
	}
	Object.assign(artifactForm, {
		isEdit: true,
		order: idx,
		type: artifact.type,
		name: artifact.name,
		path: artifact.path,
	});
	artifactModalRef.value?.showModal();
}

function confirmRemoveArtifact(idx: number) {
	artifactToDelete.value = idx;
	deleteArtifactModalRef.value?.showModal();
}

async function removeArtifact() {
	const idx = artifactToDelete.value;
	if (idx === -1) {
		return;
	}

	sortableArtifacts.value.splice(idx, 1);

	try {
		await executeSave(async () => {
			stage.value = await buildStageApi.update(stageId.value, {
				artifacts: sortableArtifacts.value.length > 0 ? sortableArtifacts.value : [],
			});
			toast.success('删除成功');
			deleteArtifactModalRef.value?.close();
			artifactToDelete.value = -1;
		});
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '删除失败');
	}
}

async function handleSaveArtifact() {
	if (!artifactForm.name.trim() || !artifactForm.path.trim()) {
		toast.error('名称和路径不能为空');
		return;
	}

	if (artifactForm.isEdit) {
		// 编辑模式
		sortableArtifacts.value[artifactForm.order] = {
			type: artifactForm.type,
			name: artifactForm.name,
			path: artifactForm.path,
		};
	} else {
		// 添加模式 - 检查名称是否重复
		if (sortableArtifacts.value.some((a) => a.name === artifactForm.name)) {
			toast.error('制品名称已存在');
			return;
		}
		sortableArtifacts.value.push({
			type: artifactForm.type,
			name: artifactForm.name,
			path: artifactForm.path,
		});
	}

	try {
		await executeSave(async () => {
			stage.value = await buildStageApi.update(stageId.value, {
				artifacts: sortableArtifacts.value,
			});
			toast.success(artifactForm.isEdit ? '更新成功' : '添加成功');
			artifactModalRef.value?.close();
		});
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '保存失败');
	}
}

function openDeleteModal() {
	deleteModalRef.value?.showModal();
}

async function handleDuplicate() {
	try {
		await executeDuplicate(async () => {
			const newStage = await buildStageApi.duplicate(stageId.value);
			toast.success('复制成功');
			router.push(`/ci/build-stage/${newStage.id}`);
		});
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '复制失败');
	}
}

async function handleDelete() {
	try {
		await executeDelete(async () => {
			await buildStageApi.delete(stageId.value);
			toast.success('删除成功');
			router.push('/ci/build-stage');
		});
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '删除失败');
	}
}

watch(stageId, fetchStage);
onMounted(fetchStage);
</script>
