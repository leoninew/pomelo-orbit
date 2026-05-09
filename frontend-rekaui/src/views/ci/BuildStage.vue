<template>
	<div class="flex flex-col gap-4">
		<!-- Header -->
		<div class="flex flex-wrap items-center justify-between gap-3">
			<div class="flex items-center gap-3">
				<div>
					<h1 class="text-xl font-semibold text-foreground">{{ stage?.name ?? 'Stage 详情' }}</h1>
				</div>
			</div>
			<div class="flex flex-wrap items-center gap-2">
				<button v-if="stage" class="app-button-primary h-9 px-3" @click="openEditModal">
					<Pencil class="size-4" />
					编辑
				</button>
				<button
					v-if="stage"
					class="app-button h-9 px-3"
					:disabled="duplicating"
					@click="handleDuplicate"
				>
					<Copy class="size-4" />
					复制
				</button>
				<button
					v-if="stage"
					class="app-button-danger h-9 px-3"
					:disabled="deleting"
					@click="openDeleteModal"
				>
					<Trash2 class="size-4" />
					删除
				</button>
				<button class="app-button h-9 px-4" @click="router.push('/ci/build-stage')">
					<ArrowLeft class="size-4" />
					返回
				</button>
			</div>
		</div>

		<!-- Loading State -->
		<AppSpinner v-if="status === 'loading'" class="py-12" />

		<!-- Content -->
		<template v-else-if="stage">
			<!-- Basic Info Card -->
			<div class="app-surface">
				<div class="app-section-header">
					<h2 class="font-semibold text-foreground">基本信息</h2>
				</div>
				<dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">名称</dt>
						<dd class="text-foreground">{{ stage.name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">版本</dt>
						<dd class="text-foreground">v{{ stage.version }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">镜像</dt>
						<dd class="text-foreground">{{ stage.image }}</dd>
					</div>
					<div class="flex gap-2 sm:col-span-2">
						<dt class="w-24 shrink-0 text-muted-foreground">描述</dt>
						<dd class="text-foreground">{{ stage.description || '—' }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">创建时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(stage.created_at) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">更新时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(stage.updated_at) }}</dd>
					</div>
				</dl>
			</div>

			<!-- Script Card -->
			<div class="app-surface">
				<div class="app-section-header flex items-center justify-between">
					<h2 class="font-semibold text-foreground">执行脚本</h2>
					<button class="app-button h-8 px-3" @click="openScriptDrawer">编辑</button>
				</div>
				<pre
					v-if="stage.script"
					class="overflow-x-auto bg-muted/30 p-5 text-xs font-mono text-foreground"
				>{{ stage.script }}</pre
				>
				<div v-else class="px-5 py-10 text-center text-muted-foreground">
					<p class="text-sm">暂无脚本</p>
				</div>
			</div>

			<!-- Artifacts Card -->
			<div class="app-surface">
				<div class="app-section-header flex items-center justify-between">
					<h2 class="font-semibold text-foreground">制品配置</h2>
					<button class="app-button-primary h-8 px-3" @click="openAddArtifactModal">
						<Plus class="size-4" />
						添加制品
					</button>
				</div>
				<div class="overflow-x-auto">
					<table class="app-table-detail min-w-[720px]">
						<thead>
							<tr>
								<th>#</th>
								<th>类型</th>
								<th>名称</th>
								<th>路径/镜像</th>
								<th>操作</th>
							</tr>
						</thead>
						<tbody>
							<tr v-if="sortableArtifacts.length === 0">
								<td colspan="5" class="text-center text-muted-foreground">暂无制品配置</td>
							</tr>
							<tr v-for="(artifact, idx) in sortableArtifacts" :key="idx">
								<td class="text-muted-foreground">{{ idx + 1 }}</td>
								<td>
									<span class="app-badge">
										{{ artifact.type }}
									</span>
								</td>
								<td class="text-foreground">{{ artifact.name }}</td>
								<td class="text-muted-foreground">{{ artifact.path }}</td>
								<td>
									<div class="flex items-center gap-3">
										<button class="app-link" @click="openEditArtifactModal(idx)">编辑</button>
										<button class="app-link-danger" @click="confirmRemoveArtifact(idx)">
											删除
										</button>
									</div>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</template>

		<AppDialog
			v-model:open="isEditDialogOpen"
			title="编辑构建"
			description="更新构建阶段基本信息。"
		>
			<div class="space-y-4">
				<div class="space-y-1.5">
					<label class="app-field-label block">名称</label>
					<input v-model="form.name" type="text" class="app-input" placeholder="例如: build" />
				</div>
				<div class="space-y-1.5">
					<label class="app-field-label block">镜像</label>
					<input
						v-model="form.image"
						type="text"
						class="app-input"
						placeholder="例如: alpine:latest"
					/>
				</div>
				<div class="space-y-1.5">
					<label class="app-field-label block">描述（可选）</label>
					<input v-model="form.description" type="text" class="app-input" placeholder="简短描述" />
				</div>
			</div>
			<template #footer>
				<button class="app-button" @click="isEditDialogOpen = false">取消</button>
				<button class="app-button-primary" :disabled="saving" @click="handleSave">保存</button>
			</template>
		</AppDialog>

		<AppDrawer
			v-model:open="showScriptDrawer"
			title="编辑脚本"
			width-class="w-[min(960px,100vw)]"
			body-class="min-h-0 flex-1 overflow-hidden p-4"
		>
			<textarea
				v-model="scriptTemp"
				class="app-textarea h-full resize-none font-mono"
				placeholder="输入执行脚本..."
			/>
			<template #footer>
				<button class="app-button" @click="closeScriptDrawer">取消</button>
				<button class="app-button-primary" :disabled="saving" @click="confirmScript">保存</button>
			</template>
		</AppDrawer>

		<AppDialog
			v-model:open="isArtifactDialogOpen"
			:title="artifactForm.isEdit ? '编辑制品' : '添加制品'"
			description="配置构建阶段产出的制品。"
		>
			<div class="space-y-4">
				<div class="space-y-1.5">
					<label class="app-field-label block">类型</label>
					<SelectControl
						v-model="artifactForm.type"
						:options="artifactTypeOptions"
						placeholder="选择制品类型"
					/>
				</div>
				<div class="space-y-1.5">
					<label class="app-field-label block">名称</label>
					<input
						v-model="artifactForm.name"
						type="text"
						class="app-input"
						placeholder="例如: my-app"
					/>
				</div>
				<div class="space-y-1.5">
					<label class="app-field-label block">路径/镜像</label>
					<input
						v-model="artifactForm.path"
						type="text"
						class="app-input"
						placeholder="例如: image:tag 或 ./dist"
					/>
				</div>
			</div>
			<template #footer>
				<button class="app-button" @click="isArtifactDialogOpen = false">取消</button>
				<button class="app-button-primary" :disabled="saving" @click="handleSaveArtifact">
					{{ artifactForm.isEdit ? '保存' : '添加' }}
				</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isDeleteArtifactDialogOpen"
			title="删除制品"
			description="确定删除此制品配置？"
			width-class="w-[min(420px,calc(100vw-32px))]"
			body-class="hidden"
		>
			<template #footer>
				<button class="app-button" @click="isDeleteArtifactDialogOpen = false">取消</button>
				<button class="app-button-destructive" :disabled="saving" @click="removeArtifact">
					删除
				</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isDeleteDialogOpen"
			title="删除 Stage"
			description="确定删除此 Stage？此操作不可恢复。"
			width-class="w-[min(420px,calc(100vw-32px))]"
			body-class="hidden"
		>
			<template #footer>
				<button class="app-button" @click="isDeleteDialogOpen = false">取消</button>
				<button class="app-button-destructive" :disabled="deleting" @click="handleDelete">
					删除
				</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { ArrowLeft, Copy, Pencil, Plus, Trash2 } from 'lucide-vue-next';
	// import { CodeEditor } from 'monaco-editor-vue3';
	import { computed, onMounted, reactive, ref, watch } from 'vue';
	// import { VueDraggable } from 'vue-draggable-plus';
	import { useRoute, useRouter } from 'vue-router';
	import { buildStageApi } from '@/api/ci';
	import AppDialog from '@/components/AppDialog.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import AppDrawer from '@/components/AppDrawer.vue';
	import SelectControl from '@/components/SelectControl.vue';
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
	const isDeleteDialogOpen = ref(false);
	const isEditDialogOpen = ref(false);
	const isArtifactDialogOpen = ref(false);
	const isDeleteArtifactDialogOpen = ref(false);
	const showScriptDrawer = ref(false);
	const scriptTemp = ref('');
	const artifactTypeOptions = [
		{ value: 'docker_image', label: 'Docker 镜像' },
		{ value: 'binary', label: '二进制文件' },
	];
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
		isEditDialogOpen.value = true;
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
				isEditDialogOpen.value = false;
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
		isArtifactDialogOpen.value = true;
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
		isArtifactDialogOpen.value = true;
	}

	function confirmRemoveArtifact(idx: number) {
		artifactToDelete.value = idx;
		isDeleteArtifactDialogOpen.value = true;
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
				isDeleteArtifactDialogOpen.value = false;
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
				isArtifactDialogOpen.value = false;
			});
		} catch (e) {
			toast.error(e instanceof Error ? e.message : '保存失败');
		}
	}

	function openDeleteModal() {
		isDeleteDialogOpen.value = true;
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
