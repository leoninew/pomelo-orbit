<template>
	<!-- Stage 编辑抽屉 -->
	<AppDrawer
		:open="open"
		:title="editingStage ? '编辑构建' : '新建构建'"
		width-class="w-[min(800px,100vw)]"
		@update:open="(val) => !val && handleClose()"
	>
		<!-- 表单内容 -->
		<div class="space-y-6">
			<!-- 基本信息 -->
			<div class="grid gap-4 md:grid-cols-2">
				<div class="space-y-1.5">
					<label class="app-field-label block">
						Stage 名称
						<span class="text-destructive">*</span>
					</label>
					<input
						v-model="form.name"
						type="text"
						class="app-input"
						placeholder="例如: deploy, lint"
					/>
				</div>
				<div class="space-y-1.5">
					<label class="app-field-label block">
						执行镜像
						<span class="text-destructive">*</span>
					</label>
					<input
						v-model="form.image"
						type="text"
						class="app-input"
						placeholder="例如: python:3.12-slim"
					/>
				</div>
			</div>

			<!-- 描述 -->
			<div class="space-y-1.5">
				<label class="app-field-label block">描述（可选）</label>
				<input
					v-model="form.description"
					type="text"
					class="app-input"
					placeholder="简短说明此 Stage 的用途"
				/>
			</div>

			<!-- 脚本 -->
			<div class="space-y-1.5">
				<div class="flex items-center justify-between">
					<label class="app-field-label block">
						脚本
						<span class="text-destructive">*</span>
					</label>
					<button class="app-button-primary px-3 py-1 text-xs" @click="openScriptDrawer">
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
			<div class="space-y-1.5">
				<div class="flex items-center justify-between">
					<label class="app-field-label block">制品（可选）</label>
					<button class="app-button flex items-center gap-1 px-3 py-1 text-xs" @click="addArtifact">
						<Plus class="h-3 w-3" />
						添加
					</button>
				</div>
				<div v-if="form.artifacts.length > 0" class="space-y-2">
					<div v-for="(artifact, idx) in form.artifacts" :key="idx" class="flex items-center gap-2">
						<SelectControl
							v-model="artifact.type"
							:options="artifactTypeOptions"
							width-class="w-36"
						/>
						<input v-model="artifact.name" type="text" class="app-input w-28" placeholder="名称" />
						<input
							v-model="artifact.path"
							type="text"
							class="app-input flex-1"
							:placeholder="artifact.type === 'docker_image' ? 'myapp:latest' : 'dist/app'"
						/>
						<button
							class="app-icon-button shrink-0 text-destructive hover:bg-destructive/10"
							@click="removeArtifact(idx)"
						>
							<X class="h-4 w-4" />
						</button>
					</div>
				</div>
				<p v-else class="text-sm text-muted-foreground">暂无制品</p>
			</div>
		</div>

		<template #footer>
			<button class="app-button" @click="handleClose">取消</button>
			<button class="app-button-primary" :disabled="saving" @click="handleSave">
				{{ editingStage ? '保存' : '创建' }}
			</button>
		</template>
	</AppDrawer>

	<!-- 脚本编辑抽屉 -->
	<AppDrawer
		:open="scriptDrawerVisible"
		title="编辑脚本"
		width-class="w-[min(960px,100vw)]"
		body-class="min-h-0 flex-1 overflow-hidden p-4"
		@update:open="(val) => !val && closeScriptDrawer()"
	>
		<textarea
			v-model="scriptTemp"
			class="app-textarea h-full resize-none font-mono"
			placeholder="输入 Shell 脚本..."
		></textarea>

		<template #footer>
			<button class="app-button" @click="closeScriptDrawer">取消</button>
			<button class="app-button-primary" @click="confirmScript">确定</button>
		</template>
	</AppDrawer>
</template>

<script setup lang="ts">
	import { Plus, X } from 'lucide-vue-next';
	// import { CodeEditor } from 'monaco-editor-vue3';
	import { reactive, ref, watch } from 'vue';
	import { buildStageApi } from '@/api/ci';
	import AppDrawer from '@/components/AppDrawer.vue';
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
			const fallbackMessage = props.editingStage ? '保存失败' : '创建失败';
			toast.error(e instanceof Error ? e.message : fallbackMessage);
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
