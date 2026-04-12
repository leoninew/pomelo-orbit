<template>
	<Teleport to="body">
		<!-- 背景遮罩 -->
		<Transition
			enter-active-class="transition-opacity duration-200"
			enter-from-class="opacity-0"
			enter-to-class="opacity-100"
			leave-active-class="transition-opacity duration-200"
			leave-from-class="opacity-100"
			leave-to-class="opacity-0"
		>
			<div v-if="open" class="fixed inset-0 z-40 bg-black/30" @click="handleClose" />
		</Transition>

		<!-- Stage 编辑抽屉 -->
		<Transition
			enter-active-class="transition-transform duration-300 ease-out"
			enter-from-class="translate-x-full"
			enter-to-class="translate-x-0"
			leave-active-class="transition-transform duration-300 ease-in"
			leave-from-class="translate-x-0"
			leave-to-class="translate-x-full"
		>
			<div
				v-if="open"
				class="fixed inset-y-0 right-0 z-50 w-full max-w-xl bg-base-100 shadow-xl flex flex-col"
			>
				<div class="flex items-center justify-between px-5 py-4 border-b border-base-200 shrink-0">
					<h3 class="font-bold text-lg">
						{{ editingStage ? '编辑 Stage' : '新建 Stage' }}
					</h3>
					<div class="flex items-center gap-2">
						<button class="btn btn-sm btn-ghost" @click="handleClose">取消</button>
						<button class="btn btn-sm btn-primary" :disabled="saving" @click="handleSave">
							<span v-if="saving" class="loading loading-spinner loading-xs" />
							保存
						</button>
					</div>
				</div>

				<div class="flex-1 overflow-y-auto px-5 py-4 flex flex-col gap-5">
					<!-- 基本信息 -->
					<div class="flex gap-3 flex-wrap">
						<fieldset class="fieldset flex-1 min-w-40">
							<legend class="fieldset-legend">Stage 名称</legend>
							<input
								v-model="form.name"
								type="text"
								class="input w-full"
								placeholder="例如: deploy, lint"
							/>
						</fieldset>
						<fieldset class="fieldset flex-1 min-w-40">
							<legend class="fieldset-legend">执行镜像</legend>
							<input
								v-model="form.image"
								type="text"
								class="input w-full font-mono text-sm"
								placeholder="例如: python:3.12-slim"
							/>
						</fieldset>
					</div>

					<fieldset class="fieldset">
						<legend class="fieldset-legend">描述（可选）</legend>
						<input
							v-model="form.description"
							type="text"
							class="input w-full"
							placeholder="简短说明此 Stage 的用途"
						/>
					</fieldset>

					<!-- 脚本 -->
					<fieldset class="fieldset">
						<legend class="fieldset-legend">
							<span>脚本</span>
							<button class="btn btn-xs btn-primary ml-2" @click="openScriptDrawer">
								{{ form.script ? '编辑' : '添加' }}
							</button>
						</legend>
						<div
							v-if="form.script"
							class="mt-1 font-mono text-xs bg-base-200 rounded p-3 whitespace-pre-wrap leading-relaxed max-h-32 overflow-y-auto cursor-pointer hover:bg-base-300 transition-colors"
							@click="openScriptDrawer"
						>
							{{ form.script }}
						</div>
						<p v-else class="text-sm text-base-content/60 pt-1">暂无</p>
					</fieldset>

					<!-- 环境变量 -->
					<fieldset class="fieldset">
						<legend class="fieldset-legend">
							<span>环境变量（可选）</span>
							<button class="btn btn-xs btn-ghost ml-2" @click="addEnvVar">
								<Plus class="size-3" />
							</button>
						</legend>
						<div v-if="envEntries.length > 0" class="flex flex-col gap-2 pt-1">
							<div v-for="(entry, idx) in envEntries" :key="idx" class="flex items-center gap-2">
								<input
									v-model="entry.key"
									type="text"
									class="input input-sm w-36 font-mono text-sm"
									placeholder="KEY"
									@change="syncEnv"
								/>
								<span class="text-base-content/40 shrink-0">=</span>
								<input
									v-model="entry.value"
									type="text"
									class="input input-sm flex-1 font-mono text-sm"
									placeholder="value"
									@input="syncEnv"
								/>
								<button class="btn btn-xs btn-ghost text-error shrink-0" @click="removeEnvVar(idx)">
									<X class="size-3" />
								</button>
							</div>
						</div>
						<p v-else class="text-sm text-base-content/60 pt-1">暂无</p>
					</fieldset>

					<!-- 制品 -->
					<fieldset class="fieldset">
						<legend class="fieldset-legend">
							<span>制品（可选）</span>
							<button class="btn btn-xs btn-ghost ml-2" @click="addArtifact">
								<Plus class="size-3" />
							</button>
						</legend>
						<div
							v-if="form.artifacts && form.artifacts.length > 0"
							class="flex flex-col gap-2 pt-1"
						>
							<div
								v-for="(artifact, idx) in form.artifacts"
								:key="idx"
								class="flex items-center gap-2"
							>
								<select v-model="artifact.type" class="select select-sm w-36">
									<option value="docker_image">Docker 镜像</option>
									<option value="binary">二进制文件</option>
								</select>
								<input
									v-model="artifact.name"
									type="text"
									class="input input-sm w-28"
									placeholder="名称"
								/>
								<input
									v-model="artifact.path"
									type="text"
									class="input input-sm flex-1 font-mono text-sm"
									:placeholder="artifact.type === 'docker_image' ? 'myapp:latest' : 'dist/app'"
								/>
								<button
									class="btn btn-xs btn-ghost text-error shrink-0"
									@click="removeArtifact(idx)"
								>
									<X class="size-3" />
								</button>
							</div>
						</div>
						<p v-else class="text-sm text-base-content/60 pt-1">暂无</p>
					</fieldset>
				</div>
			</div>
		</Transition>

		<!-- 脚本编辑器抽屉 -->
		<Transition
			enter-active-class="transition-opacity duration-200"
			enter-from-class="opacity-0"
			enter-to-class="opacity-100"
			leave-active-class="transition-opacity duration-200"
			leave-from-class="opacity-100"
			leave-to-class="opacity-0"
		>
			<div
				v-if="scriptDrawerVisible"
				class="fixed inset-0 z-[55] bg-black/30"
				@click="closeScriptDrawer"
			/>
		</Transition>
		<Transition
			enter-active-class="transition-transform duration-300 ease-out"
			enter-from-class="translate-x-full"
			enter-to-class="translate-x-0"
			leave-active-class="transition-transform duration-300 ease-in"
			leave-from-class="translate-x-0"
			leave-to-class="translate-x-full"
		>
			<div
				v-if="scriptDrawerVisible"
				class="fixed inset-y-0 right-0 z-[60] w-[720px] max-w-full bg-base-100 shadow-2xl flex flex-col border-l border-base-200"
			>
				<div class="flex items-center justify-between px-5 py-4 border-b border-base-200 shrink-0">
					<h3 class="font-semibold">编辑脚本</h3>
					<button class="btn btn-sm btn-ghost btn-circle" @click="closeScriptDrawer">
						<X class="size-4" />
					</button>
				</div>
				<div class="flex-1 overflow-hidden p-4 bg-[#1a202c]">
					<CodeEditor
						v-model:value="scriptTemp"
						style="height: 100%"
						theme="vs-dark"
						language="shell"
						:options="{
							minimap: { enabled: false },
							fontSize: 14,
							automaticLayout: true,
							scrollBeyondLastLine: false,
						}"
					/>
				</div>
				<div
					class="flex items-center justify-end gap-2 px-5 py-4 border-t border-base-200 shrink-0"
				>
					<button class="btn btn-sm btn-primary" @click="confirmScript">确定</button>
					<button class="btn btn-sm btn-ghost" @click="closeScriptDrawer">取消</button>
				</div>
			</div>
		</Transition>
	</Teleport>
</template>

<script setup lang="ts">
import { Plus, X } from 'lucide-vue-next';
import { CodeEditor } from 'monaco-editor-vue3';
import { reactive, ref, watch } from 'vue';
import { pipelineStageApi } from '@/api/ci';
import { useToast } from '@/composables/useToast';
import type { ArtifactConfig, PipelineStage } from '@/types/ci/template';

const props = defineProps<{
	open: boolean
	editingStage?: PipelineStage // undefined = 新建
}>();

const emit = defineEmits<{
	close: []
	saved: [stage: PipelineStage]
}>();

const toast = useToast();
const saving = ref(false);
const scriptDrawerVisible = ref(false);
const scriptTemp = ref('');

const form = reactive({
	name: '',
	image: '',
	description: '',
	script: '',
	env: {} as Record<string, string>,
	artifacts: [] as ArtifactConfig[],
});

const envEntries = ref<{ key: string; value: string }[]>([]);

function syncEnv() {
	form.env = Object.fromEntries(
		envEntries.value.filter((e) => e.key.trim()).map((e) => [e.key, e.value])
	);
}

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
				env: { ...s.env },
				artifacts: s.artifacts ? JSON.parse(JSON.stringify(s.artifacts)) : [],
			});
			envEntries.value = Object.entries(form.env).map(([key, value]) => ({
				key,
				value,
			}));
		} else {
			Object.assign(form, {
				name: '',
				image: '',
				description: '',
				script: '',
				env: {},
				artifacts: [],
			});
			envEntries.value = [];
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
	syncEnv();
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
			env: form.env,
			artifacts: form.artifacts.length > 0 ? form.artifacts : undefined,
			description: form.description,
		};
		const stage = props.editingStage
			? await pipelineStageApi.update(props.editingStage.id, payload)
			: await pipelineStageApi.create(payload);
		toast.success(props.editingStage ? '更新成功' : '创建成功');
		emit('saved', stage);
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '保存失败');
	} finally {
		saving.value = false;
	}
}

function addEnvVar() {
	envEntries.value.push({ key: '', value: '' });
}
function removeEnvVar(idx: number) {
	envEntries.value.splice(idx, 1);
	syncEnv();
}
function addArtifact() {
	form.artifacts.push({ type: 'docker_image', path: '', name: '' });
}
function removeArtifact(idx: number) {
	form.artifacts.splice(idx, 1);
}
</script>
