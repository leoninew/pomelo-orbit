<template>
	<div class="flex flex-col gap-4">
		<!-- Header -->
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold flex items-center gap-2">
				{{ template?.name ?? '模板详情' }}
				<span v-if="template?.latest_snapshot_version" class="badge badge-sm badge-ghost">
					v{{ template.latest_snapshot_version }}
				</span>
			</h1>
			<div class="flex items-center gap-2">
				<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/ci/templates')">
					<ArrowLeft class="size-4" />
					返回
				</button>
				<template v-if="template">
					<button class="btn btn-sm btn-ghost" @click="openEditInfoModal">编辑信息</button>
					<button
						class="btn btn-sm btn-primary"
						:disabled="saving"
						@click="handleSaveOrchestration"
					>
						<span v-if="saving" class="loading loading-spinner loading-xs" />
						保存编排
					</button>
				</template>
			</div>
		</div>

		<!-- Loading -->
		<div v-if="status === 'loading'" class="flex justify-center py-16">
			<span class="loading loading-spinner loading-lg text-primary" />
		</div>

		<template v-else-if="template">
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

					<StageListView
						v-if="viewMode === 'list'"
						:stages="template.stages"
						:orchestration="orchestration"
						@reorder="orchestration = $event"
						@remove-stage="removeOrch"
						@edit-stage="openEditOrchModal"
					/>

					<div v-else class="min-h-[300px]">
						<p
							v-if="orchestration.length === 0"
							class="text-sm text-base-content/60 py-4 text-center"
						>
							暂无 Stage
						</p>
						<StageDAGView
							v-else
							:stages="dagStages"
							:readonly="false"
							@update-dependencies="handleUpdateDependencies"
							@delete-dependency="handleDeleteDependency"
						/>
					</div>
				</div>
			</div>

			<!-- 变量声明 -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
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
					<button class="btn btn-ghost" @click="editInfoModalRef?.close()">取消</button>
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
									:value="orch.stage_key"
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
									:value="orch.stage_key"
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
import { ArrowLeft, Plus } from 'lucide-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';
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
import StageListView from './components/StageListView.vue';
import VariableDeclarationsTable from './components/VariableDeclarationsTable.vue';

const route = useRoute();
const router = useRouter();
const templateId = route.params.id as string;
const toast = useToast();

const { status, execute } = useStatusAsync();
const { loading: saving, execute: executeSave } = useStatusAsync();

const template = ref<PipelineTemplate>();
const orchestration = ref<StageOrchestration[]>([]);
const declarations = ref<VariableDeclaration[]>([]);
const allStages = ref<PipelineStage[]>([]);
const viewMode = ref<'list' | 'dag'>('list');

const editInfoModalRef = ref<HTMLDialogElement>();
const addOrchModalRef = ref<HTMLDialogElement>();
const editOrchModalRef = ref<HTMLDialogElement>();
const editForm = reactive({ name: '', description: '' });
const addOrchForm = reactive({ stageId: '', stageKey: '', dependsOn: [] as string[] });
const editOrchForm = reactive({ originalKey: '', stageKey: '', dependsOn: [] as string[] });

const stageKeyError = computed(() => {
	const key = addOrchForm.stageKey.trim();
	if (!key) return 'Stage Key 不能为空';
	if (orchestration.value.some((o) => o.stage_key === key)) return 'Stage Key 已存在';
	return '';
});

const editStageKeyError = computed(() => {
	const key = editOrchForm.stageKey.trim();
	if (!key) return 'Stage Key 不能为空';
	if (key !== editOrchForm.originalKey && orchestration.value.some((o) => o.stage_key === key))
		return 'Stage Key 已存在';
	return '';
});

const editableOrchOptions = computed(() =>
	orchestration.value.filter((o) => o.stage_key !== editOrchForm.originalKey)
);

const stageMap = computed(() =>
	Object.fromEntries((template.value?.stages ?? []).map((s) => [s.id, s]))
);

const availableStages = computed(() => {
	const inOrch = new Set(orchestration.value.map((o) => o.stage_id));
	return allStages.value.filter((s) => !inOrch.has(s.id));
});

const dagStages = computed(() =>
	orchestration.value.map((orch) => {
		const stage = stageMap.value[orch.stage_id];
		return {
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
			const [tmpl, stages] = await Promise.all([
				pipelineTemplateApi.get(templateId),
				pipelineStageApi.list(),
			]);
			template.value = tmpl;
			orchestration.value = [...tmpl.orchestration].sort((a, b) => a.sort_order - b.sort_order);
			declarations.value = [...tmpl.variable_declarations];
			allStages.value = stages;
			Object.assign(editForm, { name: tmpl.name, description: tmpl.description ?? '' });
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

async function handleSaveOrchestration() {
	const orchForCheck = orchestration.value.map((o) => ({
		name: o.stage_key,
		depends_on: o.depends_on,
	}));
	const cycle = detectCircularDependencies(orchForCheck);
	if (cycle) {
		toast.error(`检测到循环依赖: ${cycle.join(' → ')}`);
		return;
	}

	try {
		await executeSave(async () => {
			const data = await pipelineTemplateApi.update(templateId, {
				orchestration: orchestration.value,
				variable_declarations: declarations.value,
			});
			template.value = data;
			orchestration.value = [...data.orchestration].sort((a, b) => a.sort_order - b.sort_order);
			declarations.value = [...data.variable_declarations];
			toast.success(
				data.latest_snapshot_version != null
					? `保存成功，快照 v${data.latest_snapshot_version}`
					: '保存成功'
			);
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
			for (const m of (text ?? '').matchAll(re())) found.add(m[1]);
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

function openAddOrchModal() {
	addOrchForm.stageId = '';
	addOrchForm.stageKey = '';
	addOrchForm.dependsOn = [];
	addOrchModalRef.value?.showModal();
}

function onStageSelect() {
	const stage = allStages.value.find((s) => s.id === addOrchForm.stageId);
	if (stage) addOrchForm.stageKey = stage.name;
}

function confirmAddOrch() {
	if (!addOrchForm.stageId || stageKeyError.value) return;
	const maxOrder = orchestration.value.reduce((m, o) => Math.max(m, o.sort_order), 0);
	orchestration.value.push({
		stage_id: addOrchForm.stageId,
		stage_key: addOrchForm.stageKey.trim(),
		depends_on: [...addOrchForm.dependsOn],
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
		o.depends_on = o.depends_on.filter((key) => key !== removed.stage_key);
	}
	if (template.value) {
		template.value.stages = template.value.stages.filter((s) => s.id !== removed.stage_id);
		syncDeclarations(template.value.stages);
	}
}

function openEditOrchModal(idx: number) {
	const orch = orchestration.value[idx];
	if (!orch) return;
	editOrchForm.originalKey = orch.stage_key;
	editOrchForm.stageKey = orch.stage_key;
	editOrchForm.dependsOn = [...orch.depends_on];
	editOrchModalRef.value?.showModal();
}

function confirmEditOrch() {
	if (editStageKeyError.value) return;
	const newKey = editOrchForm.stageKey.trim();
	const oldKey = editOrchForm.originalKey;
	// 更新 stage_key 和 depends_on
	for (const o of orchestration.value) {
		if (o.stage_key === oldKey) {
			o.stage_key = newKey;
			o.depends_on = [...editOrchForm.dependsOn];
		} else {
			// 其他 stage 的依赖里如果引用了旧 key，同步更新
			o.depends_on = o.depends_on.map((k) => (k === oldKey ? newKey : k));
		}
	}
	editOrchForm.originalKey = '';
	editOrchForm.stageKey = '';
	editOrchForm.dependsOn = [];
	editOrchModalRef.value?.close();
}

function handleUpdateDependencies(stageKey: string, dependsOnKeys: string[]) {
	const orch = orchestration.value.find((o) => o.stage_key === stageKey);
	if (orch) orch.depends_on = dependsOnKeys;
}

function handleDeleteDependency(sourceKey: string, targetKey: string) {
	const orch = orchestration.value.find((o) => o.stage_key === targetKey);
	if (orch) orch.depends_on = orch.depends_on.filter((k) => k !== sourceKey);
}

onMounted(fetchTemplate);
</script>
