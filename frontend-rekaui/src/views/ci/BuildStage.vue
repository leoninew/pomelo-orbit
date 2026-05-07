<script setup lang="ts">
import { FilePen, Plus, X } from 'lucide-vue-next';
// import { CodeEditor } from 'monaco-editor-vue3';
import { computed, onMounted, reactive, ref, watch } from 'vue';
// import { VueDraggable } from 'vue-draggable-plus';
import { useRoute, useRouter } from 'vue-router';
import { buildStageApi } from '@/api/ci';
import AppDialog from '@/components/AppDialog.vue';
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
	{ value: 'file', label: '文件' },
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
				<button
					v-if="stage"
					class="h-9 rounded-md border border-input bg-background px-3 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="openEditModal"
				>
					编辑
				</button>
				<button
					v-if="stage"
					class="h-9 rounded-md border border-input bg-background px-3 text-sm font-medium text-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="duplicating"
					@click="handleDuplicate"
				>
					复制
				</button>
				<button
					v-if="stage"
					class="h-9 rounded-md border border-destructive/50 bg-background px-3 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="deleting"
					@click="openDeleteModal"
				>
					删除
				</button>
				<button
					class="h-9 rounded-md border border-input bg-background px-4 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="router.push('/ci/build-stage')"
				>
					返回
				</button>
			</div>
		</div>

		<!-- Loading State -->
		<div v-if="status === 'loading'" class="flex justify-center py-12">
			<span class="inline-block size-8 border-4 border-primary/20 border-t-primary rounded-full animate-spin" />
		</div>

		<!-- Content -->
		<template v-else-if="stage">
			<!-- Basic Info Card -->
			<div class="rounded-lg border border-border bg-card shadow-sm">
				<div class="border-b border-border px-5 py-4">
					<h2 class="font-semibold text-foreground">基本信息</h2>
				</div>
				<dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
					<div class="flex gap-2">
						<dt class="text-muted-foreground w-24 shrink-0">名称</dt>
						<dd class="text-foreground">{{ stage.name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-muted-foreground w-24 shrink-0">版本</dt>
						<dd class="text-foreground">v{{ stage.version }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-muted-foreground w-24 shrink-0">镜像</dt>
						<dd class="text-foreground">{{ stage.image }}</dd>
					</div>
					<div class="flex gap-2 sm:col-span-2">
						<dt class="text-muted-foreground w-24 shrink-0">描述</dt>
						<dd class="text-foreground">{{ stage.description || '—' }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-muted-foreground w-24 shrink-0">更新时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(stage.updated_at) }}</dd>
					</div>
				</dl>
			</div>

			<!-- Script Card -->
			<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
				<div class="flex items-center justify-between border-b border-border px-5 py-4">
					<h2 class="font-semibold text-foreground">执行脚本</h2>
					<button
						class="h-8 px-3 text-sm font-medium text-foreground bg-background border border-input rounded-md hover:bg-muted/50 transition-colors"
						@click="openScriptDrawer"
					>
						编辑
					</button>
				</div>
				<pre class="overflow-x-auto bg-muted/30 p-5 text-xs font-mono text-foreground">{{ stage.script }}</pre>
			</div>

			<!-- Artifacts Card -->
			<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
				<div class="flex items-center justify-between border-b border-border px-5 py-4">
					<h2 class="font-semibold text-foreground">制品配置</h2>
					<button
						class="h-8 px-3 text-sm font-medium bg-primary text-primary-foreground rounded-md hover:bg-primary/90 flex items-center gap-1.5 transition-colors"
						@click="openAddArtifactModal"
					>
						<Plus class="size-4" />
						添加制品
					</button>
				</div>
				<div v-if="sortableArtifacts.length === 0" class="px-5 py-10 text-center text-muted-foreground">
					<p class="text-sm">暂无制品配置</p>
				</div>
				<div v-else class="divide-y divide-border">
					<div
						v-for="(artifact, idx) in sortableArtifacts"
						:key="idx"
						class="flex items-center gap-3 px-5 py-4 transition-colors hover:bg-muted/30"
					>
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 mb-1">
								<span class="text-sm text-foreground">{{ artifact.name }}</span>
								<span class="px-2 py-0.5 text-xs bg-primary/10 text-primary rounded">
									{{ artifact.type }}
								</span>
							</div>
							<p class="text-sm text-muted-foreground truncate">{{ artifact.path }}</p>
						</div>
						<div class="flex items-center gap-2">
							<button
								class="p-1.5 hover:bg-muted rounded transition-colors"
								@click="openEditArtifactModal(idx)"
							>
								<FilePen class="size-4 text-muted-foreground" />
							</button>
							<button
								class="p-1.5 hover:bg-destructive/10 rounded transition-colors"
								@click="confirmRemoveArtifact(idx)"
							>
								<X class="size-4 text-destructive" />
							</button>
						</div>
					</div>
				</div>
			</div>
		</template>

		<AppDialog v-model:open="isEditDialogOpen" title="编辑构建">
			<div class="flex flex-col gap-3">
				<div class="flex flex-col gap-1.5">
					<label class="text-sm font-medium text-foreground">名称</label>
					<input
						v-model="form.name"
						type="text"
						class="px-3 py-2 text-sm bg-background border border-input rounded-md outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
					/>
				</div>
				<div class="flex flex-col gap-1.5">
					<label class="text-sm font-medium text-foreground">镜像</label>
					<input
						v-model="form.image"
						type="text"
						class="px-3 py-2 text-sm bg-background border border-input rounded-md outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
					/>
				</div>
				<div class="flex flex-col gap-1.5">
					<label class="text-sm font-medium text-foreground">描述</label>
					<input
						v-model="form.description"
						type="text"
						class="px-3 py-2 text-sm bg-background border border-input rounded-md outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
					/>
				</div>
			</div>
			<template #footer>
				<button
					class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="isEditDialogOpen = false"
				>
					取消
				</button>
				<button
					class="flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="saving"
					@click="handleSave"
				>
					<span v-if="saving" class="inline-block size-3.5 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin" />
					保存
				</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="showScriptDrawer"
			title="编辑脚本"
			width-class="w-[min(768px,calc(100vw-32px))]"
			body-class="px-6 py-4"
		>
			<textarea
				v-model="scriptTemp"
				class="w-full h-96 px-3 py-2 text-xs font-mono bg-background border border-input rounded-md outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
				placeholder="输入执行脚本..."
			/>
			<template #footer>
				<button
					class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="closeScriptDrawer"
				>
					取消
				</button>
				<button
					class="flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="saving"
					@click="confirmScript"
				>
					<span v-if="saving" class="inline-block size-3.5 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin" />
					保存
				</button>
			</template>
		</AppDialog>

		<AppDialog v-model:open="isArtifactDialogOpen" :title="artifactForm.isEdit ? '编辑制品' : '添加制品'">
			<div class="flex flex-col gap-3">
				<div class="flex flex-col gap-1.5">
					<label class="text-sm font-medium text-foreground">类型</label>
					<SelectControl
						v-model="artifactForm.type"
						:options="artifactTypeOptions"
						placeholder="选择制品类型"
					/>
				</div>
				<div class="flex flex-col gap-1.5">
					<label class="text-sm font-medium text-foreground">名称</label>
					<input
						v-model="artifactForm.name"
						type="text"
						class="px-3 py-2 text-sm bg-background border border-input rounded-md outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
						placeholder="例如: my-app"
					/>
				</div>
				<div class="flex flex-col gap-1.5">
					<label class="text-sm font-medium text-foreground">路径</label>
					<input
						v-model="artifactForm.path"
						type="text"
						class="px-3 py-2 text-sm bg-background border border-input rounded-md outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
						placeholder="例如: ./dist"
					/>
				</div>
			</div>
			<template #footer>
				<button
					class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="isArtifactDialogOpen = false"
				>
					取消
				</button>
				<button
					class="flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="saving"
					@click="handleSaveArtifact"
				>
					<span v-if="saving" class="inline-block size-3.5 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin" />
					保存
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
				<button
					class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="isDeleteArtifactDialogOpen = false"
				>
					取消
				</button>
				<button
					class="flex items-center gap-1.5 rounded-md bg-destructive px-4 py-2 text-sm font-medium text-destructive-foreground transition-colors hover:bg-destructive/90 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="saving"
					@click="removeArtifact"
				>
					<span v-if="saving" class="inline-block size-3.5 border-2 border-destructive-foreground/30 border-t-destructive-foreground rounded-full animate-spin" />
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
				<button
					class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="isDeleteDialogOpen = false"
				>
					取消
				</button>
				<button
					class="flex items-center gap-1.5 rounded-md bg-destructive px-4 py-2 text-sm font-medium text-destructive-foreground transition-colors hover:bg-destructive/90 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="deleting"
					@click="handleDelete"
				>
					<span v-if="deleting" class="inline-block size-3.5 border-2 border-destructive-foreground/30 border-t-destructive-foreground rounded-full animate-spin" />
					删除
				</button>
			</template>
		</AppDialog>
	</div>
</template>
