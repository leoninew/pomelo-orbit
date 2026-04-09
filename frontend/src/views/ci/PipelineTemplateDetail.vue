<template>
	<div class="flex flex-col gap-4">
		<!-- Header -->
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">{{ template?.name ?? '模板详情' }}</h1>
			<div class="flex items-center gap-2">
				<button
					v-if="template"
					class="btn btn-sm btn-primary"
					:disabled="saving"
					@click="handleSave"
				>
					<span v-if="saving" class="loading loading-spinner loading-xs" />
					保存
				</button>
				<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/ci/template')">
					<ArrowLeft class="size-4" />
					返回
				</button>
			</div>
		</div>

		<!-- Loading -->
		<div v-if="status === 'loading'" class="flex justify-center py-16">
			<span class="loading loading-spinner loading-lg text-primary" />
		</div>

		<template v-else-if="template">
			<!-- 基本信息 -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<div class="flex items-center justify-between mb-4">
						<h2 class="font-semibold">基本信息</h2>
						<button class="btn btn-sm btn-ghost" @click="openEditInfoModal">编辑</button>
					</div>
					<dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">模板名称</dt>
							<dd>{{ template.name }}</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">版本</dt>
							<dd>
								<span class="badge badge-sm badge-ghost">v{{ template.version }}</span>
							</dd>
						</div>
						<div class="flex gap-2 sm:col-span-2">
							<dt class="text-base-content/70 w-24 shrink-0">描述</dt>
							<dd class="text-base-content/60">{{ template.description || '—' }}</dd>
						</div>
					</dl>
				</div>
			</div>

			<!-- Stages 编排 -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<div class="flex items-center justify-between mb-4">
						<h2 class="font-semibold">Stages 编排</h2>
						<div class="flex items-center gap-2">
							<button class="btn btn-sm btn-ghost gap-1" @click="openAddOrchModal">
								<Plus class="size-3.5" />
								添加 Stage
							</button>
							<div class="join">
								<button
									class="btn btn-xs join-item"
									:class="viewMode === 'list' ? 'btn-active' : 'btn-ghost'"
									@click="viewMode = 'list'"
								>
									列表
								</button>
								<button
									class="btn btn-xs join-item"
									:class="viewMode === 'dag' ? 'btn-active' : 'btn-ghost'"
									@click="viewMode = 'dag'"
								>
									DAG
								</button>
							</div>
						</div>
					</div>

					<VueDraggable
						v-if="viewMode === 'list'"
						v-model="sortableOrch"
						tag="table"
						class="table w-full"
						handle=".drag-handle"
						:animation="150"
						ghost-class="opacity-30"
						@end="onDragEnd"
					>
						<thead>
							<tr class="text-base-content/60 text-xs">
								<th class="w-6 pr-0"></th>
								<th class="w-8">#</th>
								<th>Stage 名称</th>
								<th>依赖</th>
								<th class="w-24 text-center">制品</th>
								<th class="w-28">操作</th>
							</tr>
						</thead>
						<tbody>
							<tr v-if="sortableOrch.length === 0">
								<td colspan="6" class="text-center py-8 text-base-content/60">暂无数据</td>
							</tr>
							<tr v-for="(orch, idx) in sortableOrch" :key="orch.stage_id" class="hover">
								<td class="pr-0 w-6">
									<GripVertical
										class="drag-handle size-4 text-base-content/30 hover:text-base-content/60 cursor-grab active:cursor-grabbing transition-colors"
									/>
								</td>
								<td class="text-base-content/40 text-xs">{{ idx + 1 }}</td>
								<td>
									<router-link
										:to="`/ci/pipeline-stage/${orch.stage_id}`"
										class="link link-primary text-xs"
									>
										{{ stageCache[orch.stage_id]?.name ?? orch.stage_id }}
									</router-link>
								</td>
								<td>
									<div v-if="orch.depends_on.length > 0" class="flex items-center gap-1 flex-wrap">
										<span
											v-for="depId in orch.depends_on"
											:key="depId"
											class="text-xs bg-base-200 rounded px-2 py-0.5 text-base-content/70"
										>
											{{ stageCache[depId]?.name ?? depId }}
										</span>
									</div>
									<span v-else class="text-base-content/40 text-xs">—</span>
								</td>
								<td class="text-center text-xs text-base-content/60">
									{{ stageCache[orch.stage_id]?.artifacts?.length ?? '—' }}
								</td>
								<td>
									<div class="flex items-center gap-3">
										<button class="link link-primary text-xs" @click="openEditOrchModal(idx)">
											编辑
										</button>
										<button class="link link-error text-xs" @click="removeOrch(idx)">移除</button>
									</div>
								</td>
							</tr>
						</tbody>
					</VueDraggable>

					<div v-else class="min-h-[300px]">
						<p
							v-if="sortableOrch.length === 0"
							class="text-sm text-base-content/60 py-4 text-center"
						>
							暂无数据
						</p>
						<StageDAGView v-else :stages="dagStages" />
					</div>
				</div>
			</div>

			<!-- 变量声明 -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<h2 class="font-semibold mb-4">变量声明</h2>
					<VariableDeclarationsTable v-model:declarations="declarations" :readonly="false" />
				</div>
			</div>
		</template>

		<!-- 编辑模板信息 modal -->
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
					<button class="btn btn-ghost" @click="cancelEditInfo">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- 添加 Stage 到编排 modal -->
		<dialog ref="addOrchModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">添加 Stage 到编排</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">选择 Stage</legend>
						<select v-model="addOrchForm.stageId" class="select w-full">
							<option value="">— 选择 —</option>
							<option v-for="s in availableStages" :key="s.id" :value="s.id">
								{{ s.name }} ({{ s.image }})
							</option>
						</select>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">依赖（depends_on）</legend>
						<div class="flex flex-wrap gap-2 pt-1">
							<label
								v-for="orch in sortableOrch"
								:key="orch.stage_id"
								class="flex items-center gap-2 cursor-pointer"
							>
								<input
									v-model="addOrchForm.dependsOn"
									type="checkbox"
									:value="orch.stage_id"
									class="checkbox checkbox-sm"
								/>
								{{ stageCache[orch.stage_id]?.name ?? orch.stage_id }}
							</label>
							<span v-if="sortableOrch.length === 0" class="text-base-content/60">
								无其他 Stage
							</span>
						</div>
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="!addOrchForm.stageId" @click="confirmAddOrch">
						确定
					</button>
					<button class="btn btn-ghost" @click="addOrchModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
		<!-- 编辑 Stage 编排 modal -->
		<dialog ref="editOrchModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">编辑依赖</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">依赖（depends_on）</legend>
						<div class="flex flex-wrap gap-2 pt-1">
							<label
								v-for="orch in editableOrchOptions"
								:key="orch.stage_id"
								class="flex items-center gap-2 cursor-pointer"
							>
								<input
									v-model="editOrchForm.dependsOn"
									type="checkbox"
									:value="orch.stage_id"
									class="checkbox checkbox-sm"
								/>
								{{ stageCache[orch.stage_id]?.name ?? orch.stage_id }}
							</label>
							<span v-if="editableOrchOptions.length === 0" class="text-base-content/60">
								无其他 Stage
							</span>
						</div>
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" @click="confirmEditOrch">确定</button>
					<button class="btn btn-ghost" @click="editOrchModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { ArrowLeft, GripVertical, Plus } from 'lucide-vue-next';
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { VueDraggable } from 'vue-draggable-plus';
import { useRoute, useRouter } from 'vue-router';
import { pipelineStageApi, pipelineTemplateApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type {
	PipelineStage,
	PipelineTemplate,
	StageOrchestration,
	VariableDeclaration,
} from '@/types/ci/template';
import { detectCircularDependencies } from '@/utils/dag';
import StageDAGView from './components/StageDAGView.vue';
import VariableDeclarationsTable from './components/VariableDeclarationsTable.vue';

const route = useRoute();
const router = useRouter();
const templateId = route.params.id as string;
const toast = useToast();

const { status, execute } = useStatusAsync();
const { loading: saving, execute: executeSave } = useStatusAsync();

const template = ref<PipelineTemplate>();
const sortableOrch = ref<StageOrchestration[]>([]);
const declarations = ref<VariableDeclaration[]>([]);
const allStages = ref<PipelineStage[]>([]);
const stageCache = reactive<Record<string, PipelineStage>>({});
const viewMode = ref<'list' | 'dag'>('list');

const editInfoModalRef = ref<HTMLDialogElement>();
const addOrchModalRef = ref<HTMLDialogElement>();
const editOrchModalRef = ref<HTMLDialogElement>();
const editForm = reactive({ name: '', description: '' });
const addOrchForm = reactive({
	stageId: '',
	dependsOn: [] as string[],
});
const editOrchForm = reactive({
	editingStageId: '',
	dependsOn: [] as string[],
});

const editableOrchOptions = computed(() =>
	sortableOrch.value.filter((o) => o.stage_id !== editOrchForm.editingStageId)
);

const availableStages = computed(() => {
	const inOrch = new Set(sortableOrch.value.map((o) => o.stage_id));
	return allStages.value.filter((s) => !inOrch.has(s.id));
});

const dagStages = computed(() =>
	sortableOrch.value.map((orch) => {
		const stage = stageCache[orch.stage_id];
		return {
			id: orch.stage_id,
			name: stage?.name ?? orch.stage_id,
			image: stage?.image ?? '',
			script: stage?.script ?? '',
			env: stage?.env ?? {},
			depends_on: orch.depends_on,
		};
	})
);

async function fetchTemplate() {
	try {
		await execute(async () => {
			const tmpl = await pipelineTemplateApi.get(templateId);
			template.value = tmpl;
			sortableOrch.value = [...tmpl.orchestration].sort((a, b) => a.sort_order - b.sort_order);
			declarations.value = [...tmpl.variable_declarations];
			allStages.value = [];
			Object.keys(stageCache).forEach((k) => delete stageCache[k]);
			tmpl.stages.forEach((s) => (stageCache[s.id] = s));
			Object.assign(editForm, {
				name: tmpl.name,
				description: tmpl.description ?? '',
			});
		});
	} catch {
		toast.error('获取模板信息失败');
		router.push('/ci/template');
	}
}

async function fetchStages() {
	// 按需加载：仅在打开"添加 Stage"模态框时才加载
	if (allStages.value.length === 0) {
		try {
			const stages = await pipelineStageApi.list();
			allStages.value = stages;
		} catch {
			toast.error('获取 Stage 列表失败');
		}
	}
}

function openEditInfoModal() {
	Object.assign(editForm, {
		name: template.value?.name ?? '',
		description: template.value?.description ?? '',
	});
	editInfoModalRef.value?.showModal();
}

function cancelEditInfo() {
	Object.assign(editForm, {
		name: template.value?.name ?? '',
		description: template.value?.description ?? '',
	});
	editInfoModalRef.value?.close();
}

async function handleEditInfoOk() {
	if (!editForm.name.trim()) {
		toast.error('模板名称不能为空');
		return;
	}
	editInfoModalRef.value?.close();
	await handleSave();
}

async function handleSave() {
	const orchForCheck = sortableOrch.value.map((o) => ({
		name: stageCache[o.stage_id]?.name ?? o.stage_id,
		depends_on: o.depends_on.map((depId) => stageCache[depId]?.name ?? depId),
	}));
	const cycle = detectCircularDependencies(orchForCheck);
	if (cycle) {
		toast.error(`检测到循环依赖: ${cycle.join(' → ')}`);
		return;
	}

	try {
		await executeSave(async () => {
			const data = await pipelineTemplateApi.update(templateId, {
				name: editForm.name,
				description: editForm.description,
				orchestration: sortableOrch.value.map((o, i) => ({ ...o, sort_order: i })),
				variable_declarations: declarations.value,
			});
			template.value = data;
			sortableOrch.value = [...data.orchestration].sort((a, b) => a.sort_order - b.sort_order);
			declarations.value = [...data.variable_declarations];
			Object.keys(stageCache).forEach((k) => delete stageCache[k]);
			data.stages.forEach((s) => (stageCache[s.id] = s));
			Object.assign(editForm, {
				name: data.name,
				description: data.description ?? '',
			});
			toast.success(`保存成功，快照 v${data.version}`);
		});
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '保存失败');
	}
}

// ── 编排操作 ──────────────────────────────────────────────────────────────────

// 从当前编排的 stages 实时提取变量占位符，合并到 declarations
function extractVarNames(stages: PipelineStage[]): Set<string> {
	const found = new Set<string>();
	const re = () => /\{\{\s*([A-Za-z][A-Za-z0-9_]*)\s*\}\}/g;
	for (const s of stages) {
		for (const text of [
			s.script,
			...Object.values(s.env ?? {}),
			...(s.artifacts ?? []).flatMap((a) => [a.path, a.name]),
		]) {
			for (const m of (text ?? '').matchAll(re())) {
				found.add(m[1]);
			}
		}
	}
	return found;
}

function syncDeclarations(stages: PipelineStage[]) {
	const extracted = extractVarNames(stages);
	const existingMap = new Map(declarations.value.map((d) => [d.name, d]));
	const blank = (name: string) => ({
		name,
		description: '',
		required: false,
		default: null,
		secret: false,
		locked: false,
		builtin: false,
	});
	declarations.value = [...extracted].sort().map((name) => existingMap.get(name) ?? blank(name));
}

function onDragEnd() {
	// VueDraggable 已直接更新 sortableOrch，sort_order 在保存时统一写入
}

async function openAddOrchModal() {
	addOrchForm.stageId = '';
	addOrchForm.dependsOn = [];
	await fetchStages();
	addOrchModalRef.value?.showModal();
}

function confirmAddOrch() {
	if (!addOrchForm.stageId) {return;}
	const stage = allStages.value.find((s) => s.id === addOrchForm.stageId);
	if (!stage) {return;}
	sortableOrch.value.push({
		stage_id: addOrchForm.stageId,
		stage_name: stage.name,
		depends_on: addOrchForm.dependsOn,
		sort_order: sortableOrch.value.length,
	});
	stageCache[stage.id] = stage;
	syncDeclarations(
		sortableOrch.value.map((o) => stageCache[o.stage_id]).filter(Boolean) as PipelineStage[]
	);
	addOrchModalRef.value?.close();
}

function removeOrch(idx: number) {
	const removed = sortableOrch.value[idx];
	sortableOrch.value.splice(idx, 1);
	for (const o of sortableOrch.value) {
		o.depends_on = o.depends_on.filter((depId) => depId !== removed.stage_id);
	}
	syncDeclarations(
		sortableOrch.value.map((o) => stageCache[o.stage_id]).filter(Boolean) as PipelineStage[]
	);
}

function openEditOrchModal(idx: number) {
	const orch = sortableOrch.value[idx];
	if (!orch) {return;}
	editOrchForm.editingStageId = orch.stage_id;
	editOrchForm.dependsOn = [...orch.depends_on];
	editOrchModalRef.value?.showModal();
}

function confirmEditOrch() {
	const orch = sortableOrch.value.find((o) => o.stage_id === editOrchForm.editingStageId);
	if (!orch) {return;}
	orch.depends_on = editOrchForm.dependsOn;
	editOrchForm.editingStageId = '';
	editOrchForm.dependsOn = [];
	editOrchModalRef.value?.close();
}

onMounted(fetchTemplate);
</script>
