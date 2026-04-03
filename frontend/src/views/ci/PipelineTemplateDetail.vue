<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold flex items-center gap-2">
				{{ template?.name ?? '模板详情' }}
				<span v-if="template?.is_builtin" class="badge badge-sm badge-outline badge-info">
					内置
				</span>
				<span v-if="template?.latest_snapshot_version" class="badge badge-sm badge-ghost">
					v{{ template.latest_snapshot_version }}
				</span>
			</h1>
			<div class="flex items-center gap-2">
				<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/ci/templates')">
					<ArrowLeft class="size-4" />
					返回
				</button>
				<template v-if="template && !template.is_builtin">
					<button class="btn btn-sm btn-ghost" @click="openEditInfoModal">编辑信息</button>
					<button class="btn btn-sm btn-primary" :disabled="saving" @click="handleSave">
						<span v-if="saving" class="loading loading-spinner loading-xs" />
						保存
					</button>
				</template>
			</div>
		</div>

		<!-- Loading -->
		<div v-if="status === 'loading'" class="flex justify-center py-16">
			<span class="loading loading-spinner loading-lg text-primary" />
		</div>

		<template v-else-if="template">
			<!-- Stages -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<div class="flex items-center justify-between mb-3">
						<h2 class="font-semibold">Stages</h2>
						<button
							v-if="!template.is_builtin"
							class="btn btn-xs btn-primary gap-1"
							@click="openAddStageModal"
						>
							<Plus class="size-3" />
							添加 Stage
						</button>
					</div>
					<div v-if="stages.length === 0" class="text-sm text-base-content/60 py-4 text-center">
						暂无 Stage，点击"添加 Stage"开始配置
					</div>
					<div v-else class="flex flex-col gap-2">
						<div
							v-for="(stage, idx) in stages"
							:key="stage.name"
							class="border border-base-200 rounded-lg p-3"
						>
							<div class="flex items-center justify-between gap-2">
								<div class="flex items-center gap-2 min-w-0">
									<span class="text-base-content/40 text-xs w-5 shrink-0">{{ idx + 1 }}</span>
									<span class="font-medium truncate">{{ stage.name }}</span>
									<span class="badge badge-xs badge-ghost shrink-0">{{ stage.type }}</span>
									<span
										v-if="stage.depends_on.length"
										class="text-xs text-base-content/50 shrink-0"
									>
										← {{ stage.depends_on.join(', ') }}
									</span>
								</div>
								<div v-if="!template.is_builtin" class="flex items-center gap-1 shrink-0">
									<button
										class="btn btn-xs btn-ghost"
										:disabled="idx === 0"
										@click="moveStage(idx, -1)"
									>
										↑
									</button>
									<button
										class="btn btn-xs btn-ghost"
										:disabled="idx === stages.length - 1"
										@click="moveStage(idx, 1)"
									>
										↓
									</button>
									<button class="btn btn-xs btn-ghost text-error" @click="removeStage(idx)">
										✕
									</button>
								</div>
							</div>
							<!-- Stage config summary -->
							<div class="mt-2 text-xs text-base-content/60 pl-7">
								<template v-if="stage.type === 'checkout'">
									ref: {{ (stage.config as any)?.ref || defaultBranchPlaceholder }}
								</template>
								<template v-else-if="stage.type === 'docker_build'">
									{{ (stage.config as any)?.image_name }} ·
									{{ (stage.config as any)?.dockerfile || 'Dockerfile' }}
								</template>
								<template v-else-if="stage.type === 'unit_test'">
									{{ (stage.config as any)?.image }} ·
									{{ (stage.config as any)?.commands?.length || 0 }} 条命令
								</template>
								<template v-else-if="stage.type === 'custom'">
									{{ stage.steps?.length || 0 }} 个 Step
								</template>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Variable declarations -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<div class="flex items-center justify-between mb-3">
						<h2 class="font-semibold">变量声明</h2>
						<span class="text-xs text-base-content/50">从 Stages 自动提取，可补充元数据</span>
					</div>
					<div
						v-if="declarations.length === 0"
						class="text-sm text-base-content/60 py-4 text-center"
					>
						无变量声明
					</div>
					<table v-else class="table table-sm">
						<thead>
							<tr class="text-base-content/60">
								<th>变量名</th>
								<th>必填</th>
								<th>默认值</th>
								<th>敏感</th>
								<th>锁定</th>
								<th>描述</th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="(decl, idx) in declarations" :key="decl.name" class="hover">
								<td>
									<code class="text-xs">{{ decl.name }}</code>
								</td>
								<td>
									<input
										v-model="declarations[idx].required"
										type="checkbox"
										class="checkbox checkbox-xs"
										:disabled="template.is_builtin"
									/>
								</td>
								<td>
									<input
										v-model="declarations[idx].default"
										type="text"
										class="input input-xs w-28"
										:disabled="template.is_builtin"
										placeholder="—"
									/>
								</td>
								<td>
									<input
										v-model="declarations[idx].secret"
										type="checkbox"
										class="checkbox checkbox-xs"
										:disabled="template.is_builtin"
									/>
								</td>
								<td>
									<input
										v-model="declarations[idx].locked"
										type="checkbox"
										class="checkbox checkbox-xs"
										:disabled="template.is_builtin"
									/>
								</td>
								<td>
									<input
										v-model="declarations[idx].description"
										type="text"
										class="input input-xs w-40"
										:disabled="template.is_builtin"
										placeholder="描述"
									/>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</template>

		<!-- Edit info modal -->
		<dialog ref="editInfoModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">编辑模板信息</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">模板名称</legend>
						<input v-model="editForm.name" type="text" class="input w-full" />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">描述</legend>
						<textarea v-model="editForm.description" class="textarea w-full" rows="2" />
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="saving" @click="handleEditInfoOk">
						<span v-if="saving" class="loading loading-spinner loading-xs" />
						保存
					</button>
					<button class="btn btn-ghost" @click="editInfoModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Add stage modal -->
		<dialog ref="addStageModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">添加 Stage</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">Stage 名称</legend>
						<input
							v-model="stageForm.name"
							type="text"
							class="input w-full"
							placeholder="例如: checkout, build, test"
						/>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">类型</legend>
						<select v-model="stageForm.type" class="select w-full">
							<option value="checkout">checkout — 拉取代码</option>
							<option value="docker_build">docker_build — 构建镜像</option>
							<option value="unit_test">unit_test — 单元测试</option>
							<option value="custom">custom — 自定义</option>
						</select>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">依赖（depends_on）</legend>
						<div class="flex flex-wrap gap-1">
							<label
								v-for="s in stages"
								:key="s.name"
								class="flex items-center gap-1 text-sm cursor-pointer"
							>
								<input
									v-model="stageForm.depends_on"
									type="checkbox"
									:value="s.name"
									class="checkbox checkbox-xs"
								/>
								{{ s.name }}
							</label>
						</div>
					</fieldset>

					<!-- checkout config -->
					<template v-if="stageForm.type === 'checkout'">
						<fieldset class="fieldset">
							<legend class="fieldset-legend">ref（分支/commit）</legend>
							<input
								v-model="stageForm.checkoutRef"
								type="text"
								class="input w-full"
								:placeholder="defaultBranchPlaceholder"
							/>
						</fieldset>
					</template>

					<!-- docker_build config -->
					<template v-else-if="stageForm.type === 'docker_build'">
						<fieldset class="fieldset">
							<legend class="fieldset-legend">镜像名称（image_name）</legend>
							<input
								v-model="stageForm.imageName"
								type="text"
								class="input w-full"
								:placeholder="imageNamePlaceholder"
							/>
						</fieldset>
						<fieldset class="fieldset">
							<legend class="fieldset-legend">构建上下文（context）</legend>
							<input v-model="stageForm.context" type="text" class="input w-full" placeholder="." />
						</fieldset>
						<fieldset class="fieldset">
							<legend class="fieldset-legend">Dockerfile 路径</legend>
							<input
								v-model="stageForm.dockerfile"
								type="text"
								class="input w-full"
								placeholder="Dockerfile"
							/>
						</fieldset>
					</template>

					<!-- unit_test config -->
					<template v-else-if="stageForm.type === 'unit_test'">
						<fieldset class="fieldset">
							<legend class="fieldset-legend">运行镜像（image）</legend>
							<input
								v-model="stageForm.image"
								type="text"
								class="input w-full"
								placeholder="python:3.12"
							/>
						</fieldset>
						<fieldset class="fieldset">
							<legend class="fieldset-legend">测试命令（每行一条）</legend>
							<textarea
								v-model="stageForm.commandsText"
								class="textarea w-full font-mono text-xs"
								rows="4"
								placeholder="uv run pytest tests/"
							/>
						</fieldset>
						<fieldset class="fieldset">
							<legend class="fieldset-legend">产物路径（可选，每行一条）</legend>
							<textarea
								v-model="stageForm.artifactPathsText"
								class="textarea w-full font-mono text-xs"
								rows="2"
								placeholder="coverage.xml"
							/>
						</fieldset>
					</template>

					<!-- custom: 提示用户后续在 YAML 中编辑 -->
					<template v-else-if="stageForm.type === 'custom'">
						<p class="text-sm text-base-content/60">
							custom stage 创建后可在 Stage 列表中查看，Steps 需通过 API 或直接编辑配置。
						</p>
					</template>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" @click="handleAddStageOk">添加</button>
					<button class="btn btn-ghost" @click="addStageModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ArrowLeft, Plus } from 'lucide-vue-next';
import { pipelineTemplateApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { PipelineTemplate, StageDefinition, VariableDeclaration } from '@/types/api';

const route = useRoute();
const router = useRouter();
const templateId = route.params.id as string;
const toast = useToast();

const defaultBranchPlaceholder = '{{ DEFAULT_BRANCH }}';
const imageNamePlaceholder = '{{ REGISTRY }}/{{ IMAGE_NAME }}:{{ trigger_ref }}';

const { status, execute } = useStatusAsync();
const { loading: saving, execute: executeSave } = useStatusAsync();

const template = ref<PipelineTemplate>();
const stages = ref<StageDefinition[]>([]);
const declarations = ref<VariableDeclaration[]>([]);

const editInfoModalRef = ref<HTMLDialogElement>();
const addStageModalRef = ref<HTMLDialogElement>();
const editForm = reactive({ name: '', description: '' });

const stageForm = reactive({
	name: '',
	type: 'checkout' as StageDefinition['type'],
	depends_on: [] as string[],
	// checkout
	checkoutRef: '',
	// docker_build
	imageName: '',
	context: '.',
	dockerfile: 'Dockerfile',
	// unit_test
	image: '',
	commandsText: '',
	artifactPathsText: '',
});

async function fetchTemplate() {
	try {
		await execute(async () => {
			const data = await pipelineTemplateApi.get(templateId);
			template.value = data;
			stages.value = data.stages ? [...data.stages] : [];
			declarations.value = data.variable_declarations
				? data.variable_declarations.map((d) => ({ ...d }))
				: [];
			Object.assign(editForm, { name: data.name, description: data.description ?? '' });
		});
	} catch {
		toast.error('获取模板信息失败');
		router.push('/ci/templates');
	}
}

function openEditInfoModal() {
	editInfoModalRef.value?.showModal();
}

async function handleEditInfoOk() {
	try {
		await executeSave(async () => {
			const data = await pipelineTemplateApi.update(templateId, {
				name: editForm.name,
				description: editForm.description || undefined,
			});
			template.value = data;
			toast.success('更新成功');
			editInfoModalRef.value?.close();
		});
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '更新失败');
	}
}

async function handleSave() {
	try {
		await executeSave(async () => {
			const data = await pipelineTemplateApi.update(templateId, {
				stages: stages.value,
				variable_declarations: declarations.value,
			});
			template.value = data;
			stages.value = data.stages ? [...data.stages] : [];
			declarations.value = data.variable_declarations
				? data.variable_declarations.map((d) => ({ ...d }))
				: [];
			toast.success(`保存成功，快照 v${data.latest_snapshot_version}`);
		});
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '保存失败');
	}
}

function moveStage(idx: number, dir: -1 | 1) {
	const arr = stages.value;
	const target = idx + dir;
	if (target < 0 || target >= arr.length) return;
	[arr[idx], arr[target]] = [arr[target], arr[idx]];
}

function removeStage(idx: number) {
	stages.value.splice(idx, 1);
}

function openAddStageModal() {
	Object.assign(stageForm, {
		name: '',
		type: 'checkout',
		depends_on: [],
		checkoutRef: '',
		imageName: '',
		context: '.',
		dockerfile: 'Dockerfile',
		image: '',
		commandsText: '',
		artifactPathsText: '',
	});
	addStageModalRef.value?.showModal();
}

function handleAddStageOk() {
	if (!stageForm.name.trim()) {
		toast.error('请输入 Stage 名称');
		return;
	}
	if (stages.value.some((s) => s.name === stageForm.name)) {
		toast.error('Stage 名称已存在');
		return;
	}

	let config: StageDefinition['config'];
	if (stageForm.type === 'checkout') {
		config = { ref: stageForm.checkoutRef || '{{ DEFAULT_BRANCH }}' };
	} else if (stageForm.type === 'docker_build') {
		config = {
			image_name: stageForm.imageName,
			context: stageForm.context || '.',
			dockerfile: stageForm.dockerfile || 'Dockerfile',
		};
	} else if (stageForm.type === 'unit_test') {
		config = {
			image: stageForm.image,
			commands: stageForm.commandsText
				.split('\n')
				.map((s) => s.trim())
				.filter(Boolean),
			artifact_paths: stageForm.artifactPathsText
				.split('\n')
				.map((s) => s.trim())
				.filter(Boolean),
		};
	}

	stages.value.push({
		name: stageForm.name,
		type: stageForm.type,
		depends_on: [...stageForm.depends_on],
		config,
		steps: stageForm.type === 'custom' ? [] : undefined,
	});

	addStageModalRef.value?.close();
}

onMounted(fetchTemplate);
</script>
