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
								<th class="w-36">Stage 名称</th>
								<th class="w-40">Stage Key</th>
								<th>依赖</th>
								<th class="w-16 text-center">制品</th>
								<th class="w-24">操作</th>
							</tr>
						</thead>
						<tbody>
							<tr v-if="sortableOrch.length === 0">
								<td colspan="7" class="text-center py-8 text-base-content/60">暂无数据</td>
							</tr>
							<tr v-for="(orch, idx) in sortableOrch" :key="orch.stage_key" class="hover">
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
										{{ stageMap[orch.stage_id]?.name ?? orch.stage_id }}
									</router-link>
								</td>
								<td class="text-xs text-base-content/70">{{ orch.stage_key }}</td>
								<td>
									<div v-if="orch.depends_on.length > 0" class="flex items-center gap-1 flex-wrap">
										<span
											v-for="depId in orch.depends_on"
											:key="depId"
											class="text-xs bg-base-200 rounded px-2 py-0.5 text-base-content/70"
										>
											{{ stageKeyMap[depId] ?? depId }}
										</span>
									</div>
									<span v-else class="text-base-content/40 text-xs">—</span>
								</td>
								<td class="text-center text-xs text-base-content/60">
									{{ stageMap[orch.stage_id]?.artifacts?.length ?? '—' }}
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
							v-if="orchestration.length === 0"
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
						<select v-model="addOrchForm.stageId" class="select w-full" @change="onStageSelect">
							<option value="">— 选择 —</option>
							<option v-for="s in availableStages" :key="s.id" :value="s.id">
								{{ s.name }} ({{ s.image }})
							</option>
						</select>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">Stage Key</legend>
						<input
							v-model="addOrchForm.stageKey"
							type="text"
							class="input w-full"
							:class="{ 'input-error': stageKeyError }"
							placeholder="模板内唯一标识"
						/>
						<p v-if="stageKeyError" class="fieldset-label text-error">{{ stageKeyError }}</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">依赖（depends_on）</legend>
						<div class="flex flex-wrap gap-2 pt-1">
							<label
								v-for="orch in orchestration"
								:key="orch.stage_key"
								class="flex items-center gap-2 cursor-pointer"
							>
								<input
									v-model="addOrchForm.dependsOn"
									type="checkbox"
									:value="orch.stage_id"
									class="checkbox checkbox-sm"
								/>
								{{ orch.stage_key }}
							</label>
							<span v-if="orchestration.length === 0" class="text-base-content/60">
								无其他 Stage
							</span>
						</div>
					</fieldset>
				</div>
				<div class="modal-action">
					<button
						class="btn btn-primary"
						:disabled="!addOrchForm.stageId || !!stageKeyError"
						@click="confirmAddOrch"
					>
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
				<h3 class="font-bold text-lg mb-4">编辑 Stage</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">Stage Key</legend>
						<input
							v-model="editOrchForm.stageKey"
							type="text"
							class="input w-full"
							:class="{ 'input-error': editStageKeyError }"
						/>
						<p v-if="editStageKeyError" class="fieldset-label text-error">
							{{ editStageKeyError }}
						</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">依赖（depends_on）</legend>
						<div class="flex flex-wrap gap-2 pt-1">
							<label
								v-for="orch in editableOrchOptions"
								:key="orch.stage_key"
								class="flex items-center gap-2 cursor-pointer"
							>
								<input
									v-model="editOrchForm.dependsOn"
									type="checkbox"
									:value="orch.stage_id"
									class="checkbox checkbox-sm"
								/>
								{{ orch.stage_key }}
							</label>
							<span v-if="editableOrchOptions.length === 0" class="text-base-content/60">
								无其他 Stage
							</span>
						</div>
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="!!editStageKeyError" @click="confirmEditOrch">
						确定
					</button>
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
const orchestration = ref<StageOrchestration[]>([]);
const sortableOrch = ref<StageOrchestration[]>([]);
const declarations = ref<VariableDeclaration[]>([]);
const allStages = ref<PipelineStage[]>([]);
const viewMode = ref<'list' | 'dag'>('list');

watch(
	orchestration,
	(val) => {
		sortableOrch.value = [...val];
	},
	{ immediate: true }
);

const editInfoModalRef = ref<HTMLDialogElement>();
const addOrchModalRef = ref<HTMLDialogElement>();
const editOrchModalRef = ref<HTMLDialogElement>();
const editForm = reactive({ name: '', description: '' });
const addOrchForm = reactive({
	stageId: '',
	stageKey: '',
	dependsOn: [] as string[],
});
const editOrchForm = reactive({
	originalKey: '',
	stageKey: '',
	dependsOn: [] as string[],
});

const stageKeyError = computed(() => {
	const key = addOrchForm.stageKey.trim();
	if (!key) {
		return 'Stage Key 不能为空';
	}
	if (orchestration.value.some((o) => o.stage_key === key)) {
		return 'Stage Key 已存在';
	}
	return '';
});

const editStageKeyError = computed(() => {
	const key = editOrchForm.stageKey.trim();
	if (!key) {
		return 'Stage Key 不能为空';
	}
	if (key !== editOrchForm.originalKey && orchestration.value.some((o) => o.stage_key === key)) {
		return 'Stage Key 已存在';
	}
	return '';
});

const editableOrchOptions = computed(() =>
	orchestration.value.filter((o) => o.stage_key !== editOrchForm.originalKey)
);

const stageMap = computed(() =>
	Object.fromEntries((template.value?.stages ?? []).map((s) => [s.id, s]))
);

const stageKeyMap = computed(() =>
	Object.fromEntries(orchestration.value.map((o) => [o.stage_id, o.stage_key]))
);

const availableStages = computed(() => {
	const inOrch = new Set(orchestration.value.map((o) => o.stage_id));
	return allStages.value.filter((s) => !inOrch.has(s.id));
});

const dagStages = computed(() =>
	orchestration.value.map((orch) => {
		const stage = stageMap.value[orch.stage_id];
		return {
			id: orch.stage_id,
			name: orch.stage_key,
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
			orchestration.value = [...tmpl.orchestration].sort((a, b) => a.sort_order - b.sort_order);
			declarations.value = [...tmpl.variable_declarations];
			allStages.value = []; // 初始为空，按需加载
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
	const orchForCheck = orchestration.value.map((o) => ({
		name: o.stage_key,
		depends_on: o.depends_on.map((depId) => {
			const dep = orchestration.value.find((orch) => orch.stage_id === depId);
			return dep?.stage_key ?? depId;
		}),
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
				orchestration: orchestration.value,
				variable_declarations: declarations.value,
			});
			template.value = data;
			orchestration.value = [...data.orchestration].sort((a, b) => a.sort_order - b.sort_order);
			declarations.value = [...data.variable_declarations];
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
	orchestration.value = sortableOrch.value.map((o, i) => ({
		...o,
		sort_order: i,
	}));
}

async function openAddOrchModal() {
	addOrchForm.stageId = '';
	addOrchForm.stageKey = '';
	addOrchForm.dependsOn = [];
	await fetchStages(); // 按需加载 stages
	addOrchModalRef.value?.showModal();
}

function onStageSelect() {
	const stage = allStages.value.find((s) => s.id === addOrchForm.stageId);
	if (stage) {
		addOrchForm.stageKey = stage.name;
	}
}

function confirmAddOrch() {
	if (!addOrchForm.stageId || stageKeyError.value) {
		return;
	}
	const maxOrder = orchestration.value.reduce((m, o) => Math.max(m, o.sort_order), 0);
	orchestration.value.push({
		stage_id: addOrchForm.stageId,
		stage_key: addOrchForm.stageKey.trim(),
		depends_on: addOrchForm.dependsOn,
		sort_order: maxOrder + 1,
	});
	const stage = allStages.value.find((s) => s.id === addOrchForm.stageId);
	if (stage && template.value && !template.value.stages.find((s) => s.id === stage.id)) {
		template.value.stages.push(stage);
	}
	syncDeclarations(template.value?.stages ?? []);
	addOrchModalRef.value?.close();
}

function removeOrch(idx: number) {
	const removed = orchestration.value[idx];
	orchestration.value.splice(idx, 1);
	for (const o of orchestration.value) {
		o.depends_on = o.depends_on.filter((depId) => depId !== removed.stage_id);
	}
	if (template.value) {
		template.value.stages = template.value.stages.filter((s) => s.id !== removed.stage_id);
		syncDeclarations(template.value.stages);
	}
}

function openEditOrchModal(idx: number) {
	const orch = orchestration.value[idx];
	if (!orch) {
		return;
	}
	editOrchForm.originalKey = orch.stage_key;
	editOrchForm.stageKey = orch.stage_key;
	editOrchForm.dependsOn = [...orch.depends_on];
	editOrchModalRef.value?.showModal();
}

function confirmEditOrch() {
	if (editStageKeyError.value) {
		return;
	}
	const orch = orchestration.value.find((o) => o.stage_key === editOrchForm.originalKey);
	if (!orch) {
		return;
	}
	orch.stage_key = editOrchForm.stageKey.trim();
	orch.depends_on = editOrchForm.dependsOn;
	editOrchForm.originalKey = '';
	editOrchForm.stageKey = '';
	editOrchForm.dependsOn = [];
	editOrchModalRef.value?.close();
}

onMounted(fetchTemplate);
</script>
