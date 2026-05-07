<script setup lang="ts">
import { Plus, X } from 'lucide-vue-next';
// import { CodeEditor } from 'monaco-editor-vue3';
import { reactive, ref, watch } from 'vue';
import { buildStageApi } from '@/api/ci';
import SelectControl from '@/components/SelectControl.vue';
import { useToast } from '@/composables/useToast';
import type { ArtifactConfig, BuildStage } from '@/types/ci/template';

const props = defineProps<{
	open: boolean
	editingStage?: BuildStage // undefined = 新建
}>();

const emit = defineEmits<{
	close: []
	saved: [stage: BuildStage]
}>();

const toast = useToast();
const saving = ref(false);
const scriptDrawerVisible = ref(false);
const scriptTemp = ref('');
const artifactTypeOptions = [
	{ value: 'docker_image', label: 'Docker 镜像' },
	{ value: 'binary', label: '二进制文件' },
];

const form = reactive({
	name: '',
	image: '',
	description: '',
	script: '',
	artifacts: [] as ArtifactConfig[],
});

watch(
	() => props.open,
	(val) => {
		if (!val) {
			return;
		}
		scriptDrawerVisible.value = false;
		if (props.editingStage) {
			const s = props.editingStage;
			Object.assign(form, {
				name: s.name,
				image: s.image,
				description: s.description,
				script: s.script,
				artifacts: s.artifacts ? JSON.parse(JSON.stringify(s.artifacts)) : [],
			});
		} else {
			Object.assign(form, {
				name: '',
				image: '',
				description: '',
				script: '',
				artifacts: [],
			});
		}
	}
);

function openScriptDrawer() {
	scriptTemp.value = form.script;
	scriptDrawerVisible.value = true;
}
function closeScriptDrawer() {
	scriptDrawerVisible.value = false;
}
function confirmScript() {
	form.script = scriptTemp.value;
	scriptDrawerVisible.value = false;
}

function handleClose() {
	emit('close');
}

async function handleSave() {
	if (!form.name.trim()) {
		toast.error('请输入 Stage 名称');
		return;
	}
	if (!form.image.trim()) {
		toast.error('请输入执行镜像');
		return;
	}
	if (!form.script.trim()) {
		toast.error('请输入脚本');
		return;
	}

	saving.value = true;
	try {
		const payload = {
			name: form.name.trim(),
			image: form.image.trim(),
			script: form.script.trim(),
			artifacts: form.artifacts.length > 0 ? form.artifacts : undefined,
			description: form.description,
		};
		const stage = props.editingStage
			? await buildStageApi.update(props.editingStage.id, payload)
			: await buildStageApi.create(payload);
		toast.success(props.editingStage ? '更新成功' : '创建成功');
		emit('saved', stage);
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '保存失败');
	} finally {
		saving.value = false;
	}
}

function addArtifact() {
	form.artifacts.push({ type: 'docker_image', path: '', name: '' });
}
function removeArtifact(idx: number) {
	form.artifacts.splice(idx, 1);
}
</script>

<template>
	<!-- Stage 编辑抽屉 -->
	<div
		v-if="open"
		class="fixed inset-0 z-50 flex items-center justify-end bg-black/50"
		@click.self="handleClose"
	>
		<div class="h-full w-full max-w-2xl overflow-y-auto rounded-l-lg border-l border-border bg-card shadow-xl">
			<!-- 头部 -->
			<div class="sticky top-0 z-10 flex items-center justify-between border-b border-border bg-card px-6 py-4">
				<h2 class="text-lg font-semibold text-foreground">
					{{ editingStage ? '编辑构建' : '新建构建' }}
				</h2>
				<div class="flex items-center gap-2">
					<button
						class="rounded-md border border-border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
						@click="handleClose"
					>
						取消
					</button>
					<button
						class="flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
						:disabled="saving"
						@click="handleSave"
					>
						<span v-if="saving" class="h-4 w-4 animate-spin rounded-full border-2 border-primary-foreground/20 border-t-primary-foreground"></span>
						保存
					</button>
				</div>
			</div>

			<!-- 表单内容 -->
			<div class="space-y-6 p-6">
				<!-- 基本信息 -->
				<div class="grid gap-4 md:grid-cols-2">
					<div>
						<label class="mb-2 block text-sm font-medium text-foreground">
							Stage 名称 <span class="text-destructive">*</span>
						</label>
						<input
							v-model="form.name"
							type="text"
							class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
							placeholder="例如: deploy, lint"
						/>
					</div>
					<div>
						<label class="mb-2 block text-sm font-medium text-foreground">
							执行镜像 <span class="text-destructive">*</span>
						</label>
						<input
							v-model="form.image"
							type="text"
							class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
							placeholder="例如: python:3.12-slim"
						/>
					</div>
				</div>

				<!-- 描述 -->
				<div>
					<label class="mb-2 block text-sm font-medium text-foreground">描述（可选）</label>
					<input
						v-model="form.description"
						type="text"
						class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
						placeholder="简短说明此 Stage 的用途"
					/>
				</div>

				<!-- 脚本 -->
				<div>
					<div class="mb-2 flex items-center justify-between">
						<label class="text-sm font-medium text-foreground">
							脚本 <span class="text-destructive">*</span>
						</label>
						<button
							class="rounded-md bg-primary px-3 py-1 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90"
							@click="openScriptDrawer"
						>
							{{ form.script ? '编辑' : '添加' }}
						</button>
					</div>
					<div
						v-if="form.script"
						class="max-h-32 cursor-pointer overflow-y-auto rounded-md border border-border bg-muted/30 p-3 font-mono text-xs leading-relaxed text-foreground transition-colors hover:bg-muted/50"
						@click="openScriptDrawer"
					>
						{{ form.script }}
					</div>
					<p v-else class="text-sm text-muted-foreground">暂无脚本</p>
				</div>

				<!-- 制品 -->
				<div>
					<div class="mb-2 flex items-center justify-between">
						<label class="text-sm font-medium text-foreground">制品（可选）</label>
						<button
							class="flex items-center gap-1 rounded-md border border-border bg-background px-3 py-1 text-xs font-medium text-foreground transition-colors hover:bg-muted/50"
							@click="addArtifact"
						>
							<Plus class="h-3 w-3" />
							添加
						</button>
					</div>
					<div v-if="form.artifacts.length > 0" class="space-y-2">
						<div
							v-for="(artifact, idx) in form.artifacts"
							:key="idx"
							class="flex items-center gap-2"
						>
							<SelectControl
								v-model="artifact.type"
								:options="artifactTypeOptions"
								width-class="w-36"
							/>
							<input
								v-model="artifact.name"
								type="text"
								class="w-28 rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
								placeholder="名称"
							/>
							<input
								v-model="artifact.path"
								type="text"
								class="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
								:placeholder="artifact.type === 'docker_image' ? 'myapp:latest' : 'dist/app'"
							/>
							<button
								class="shrink-0 rounded-md p-2 text-destructive transition-colors hover:bg-destructive/10"
								@click="removeArtifact(idx)"
							>
								<X class="h-4 w-4" />
							</button>
						</div>
					</div>
					<p v-else class="text-sm text-muted-foreground">暂无制品</p>
				</div>
			</div>
		</div>
	</div>

	<!-- 脚本编辑抽屉 -->
	<div
		v-if="scriptDrawerVisible"
		class="fixed inset-0 z-[60] flex items-center justify-end bg-black/50"
		@click.self="closeScriptDrawer"
	>
		<div class="flex h-full w-full max-w-3xl flex-col rounded-l-lg border-l border-border bg-card shadow-xl">
			<!-- 头部 -->
			<div class="flex items-center justify-between border-b border-border px-6 py-4">
				<h3 class="text-lg font-semibold text-foreground">编辑脚本</h3>
				<button
					class="rounded-md p-2 text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
					@click="closeScriptDrawer"
				>
					<X class="h-4 w-4" />
				</button>
			</div>

			<!-- 编辑器区域 -->
			<div class="flex-1 overflow-hidden p-4">
				<textarea
					v-model="scriptTemp"
					class="h-full w-full resize-none rounded-md border border-border bg-muted/30 p-4 font-mono text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
					placeholder="输入 Shell 脚本..."
				></textarea>
			</div>

			<!-- 底部按钮 -->
			<div class="flex items-center justify-end gap-2 border-t border-border px-6 py-4">
				<button
					class="rounded-md border border-border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="closeScriptDrawer"
				>
					取消
				</button>
				<button
					class="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
					@click="confirmScript"
				>
					确定
				</button>
			</div>
		</div>
	</div>
</template>
