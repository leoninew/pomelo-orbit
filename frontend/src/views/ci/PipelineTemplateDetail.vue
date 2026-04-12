<template>
	<div class="flex flex-col gap-4">
		<!-- Header -->
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">{{ template?.name ?? '模板详情' }}</h1>
			<div class="flex items-center gap-2">
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
						<div class="flex items-center gap-2">
							<button class="btn btn-sm btn-primary gap-1" @click="openRunModal">
								<Play class="size-3.5" />
								运行
							</button>
							<button class="btn btn-sm btn-ghost" @click="openEditInfoModal">编辑</button>
							<button class="btn btn-sm btn-ghost" :disabled="duplicating" @click="handleDuplicate">
								<span v-if="duplicating" class="loading loading-spinner loading-xs" />
								复制
							</button>
							<button
								class="btn btn-sm btn-error btn-ghost"
								:disabled="saving"
								@click="openDeleteModal"
							>
								删除
							</button>
						</div>
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

			<!-- Stages 编排与变量声明 -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<div class="flex items-center justify-between mb-4">
						<div class="flex items-center gap-3">
							<h2 class="font-semibold">Stages 编排</h2>
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
						<div v-if="viewMode === 'list'" class="flex items-center gap-2">
							<button class="btn btn-sm btn-ghost gap-1" @click="openAddOrchModal">
								<Plus class="size-3.5" />
								添加 Stage
							</button>
							<button
								v-if="template"
								class="btn btn-sm"
								:class="isDirty || hasStageUpdates ? 'btn-warning animate-pulse' : 'btn-primary'"
								:disabled="saving"
								@click="handleSave"
							>
								<span v-if="saving" class="loading loading-spinner loading-xs" />
								更新
							</button>
						</div>
					</div>

					<!-- Stages 编排内容 -->
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
								<th class="w-20">版本</th>
								<th>依赖</th>
								<th class="w-24 text-center">制品</th>
								<th class="w-40">操作</th>
							</tr>
						</thead>
						<tbody>
							<tr v-if="sortableOrch.length === 0">
								<td colspan="7" class="text-center py-8 text-base-content/60">暂无数据</td>
							</tr>
							<tr v-for="(orch, idx) in sortableOrch" :key="orch.stage_id" class="hover">
								<td class="pr-0 w-6">
									<GripVertical
										class="drag-handle size-4 text-base-content/30 hover:text-base-content/60 cursor-grab active:cursor-grabbing transition-colors"
									/>
								</td>
								<td class="text-base-content/40 text-xs">{{ idx + 1 }}</td>
								<td>
									<div class="flex items-center gap-2">
										<router-link
											:to="`/ci/pipeline-stage/${orch.stage_id}`"
											class="link link-primary text-xs"
										>
											{{ stageCache[orch.stage_id]?.name ?? orch.stage_id }}
										</router-link>
										<span
											v-if="
												stageCache[orch.stage_id] &&
												stageCache[orch.stage_id].version > orch.stage_version
											"
											class="badge badge-xs badge-warning"
										>
											有更新
										</span>
									</div>
								</td>
								<td class="text-center">
									<span class="badge badge-sm badge-ghost">v{{ orch.stage_version }}</span>
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

					<!-- 变量声明内容 -->
					<div class="mt-6 pt-6 border-t border-base-300">
						<div class="flex items-center justify-between mb-4">
							<h3 class="font-semibold">变量声明</h3>
							<button class="btn btn-sm btn-ghost gap-1" @click="openAddVarModal">
								<Plus class="size-3.5" />
								添加变量
							</button>
						</div>
						<VariableDeclarationsTable
							:declarations="declarations"
							:readonly="false"
							@edit="openEditVarModal"
							@delete="deleteVariable"
						/>
					</div>
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

		<!-- 运行流水线 modal -->
		<dialog ref="runModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">运行流水线</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">选择项目</legend>
						<select v-model="runForm.repositoryId" class="select w-full">
							<option value="">— 选择项目 —</option>
							<option v-for="repo in repositories" :key="repo.id" :value="repo.id">
								{{ repo.name }} ({{ repo.code }})
							</option>
						</select>
						<p
							v-if="selectedRepository && !selectedRepository.git_credential_id"
							class="fieldset-label text-error"
						>
							该项目未配置 Git 凭据，请先在
							<router-link
								:to="`/ci/repository/${selectedRepository.id}`"
								class="link link-primary"
							>
								仓库详情
							</router-link>
							中配置
						</p>
					</fieldset>
					<fieldset v-if="selectedRepository" class="fieldset">
						<legend class="fieldset-legend">分支 / Ref</legend>
						<input
							v-model="runForm.triggerRef"
							type="text"
							class="input w-full"
							:placeholder="selectedRepository.default_branch || 'main'"
						/>
					</fieldset>
				</div>
				<div class="modal-action">
					<button
						class="btn btn-primary"
						:disabled="!runForm.repositoryId || !selectedRepository?.git_credential_id || running"
						@click="handleRunOk"
					>
						<span v-if="running" class="loading loading-spinner loading-xs" />
						运行
					</button>
					<button class="btn btn-ghost" @click="runModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- 添加变量 modal -->
		<dialog ref="addVarModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg mb-4">添加变量</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">变量名</legend>
						<input v-model="varForm.name" type="text" class="input w-full" placeholder="变量名" />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">变量值</legend>
						<input v-model="varForm.value" type="text" class="input w-full" placeholder="变量值" />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">说明（可选）</legend>
						<input
							v-model="varForm.description"
							type="text"
							class="input w-full"
							placeholder="变量说明"
						/>
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="!varForm.name" @click="handleAddVarOk">
						确定
					</button>
					<button class="btn btn-ghost" @click="addVarModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- 编辑变量 modal -->
		<dialog ref="editVarModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg mb-4">编辑变量</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">变量名</legend>
						<input :value="varForm.name" type="text" class="input w-full opacity-60" disabled />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">变量值</legend>
						<input v-model="varForm.value" type="text" class="input w-full" />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">说明（可选）</legend>
						<input
							v-model="varForm.description"
							type="text"
							class="input w-full"
							placeholder="变量说明"
						/>
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" @click="handleEditVarOk">保存</button>
					<button class="btn btn-ghost" @click="editVarModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Delete modal -->
		<dialog ref="deleteModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">删除模板</h3>
				<p class="py-4 text-sm">
					确定要删除模板「
					<strong>{{ template?.name }}</strong>
					」吗？此操作不可撤销。
				</p>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="deleting" @click="handleDeleteOk">
						<span v-if="deleting" class="loading loading-spinner loading-xs" />
						删除
					</button>
					<button class="btn btn-ghost" @click="deleteModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { ArrowLeft, GripVertical, Play, Plus } from 'lucide-vue-next';
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { VueDraggable } from 'vue-draggable-plus';
import { useRoute, useRouter } from 'vue-router';
import { pipelineStageApi, pipelineTemplateApi, repositoryApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type {
	PipelineStage,
	PipelineTemplate,
	StageOrchestration,
	VariableDeclaration,
} from '@/types/ci/template';
import type { Repository } from '@/types/ci/repository';
import { detectCircularDependencies } from '@/utils/dag';
import StageDAGView from './components/StageDAGView.vue';
import VariableDeclarationsTable from './components/VariableDeclarationsTable.vue';

const route = useRoute();
const router = useRouter();
const templateId = computed(() => route.params.id as string);
const toast = useToast();

const { status, execute } = useStatusAsync();
const { loading: saving, execute: executeSave } = useStatusAsync();
const { loading: deleting, execute: executeDelete } = useStatusAsync();
const { loading: running, execute: executeRun } = useStatusAsync();
const { loading: duplicating, execute: executeDuplicate } = useStatusAsync();

const template = ref<PipelineTemplate>();
const sortableOrch = ref<StageOrchestration[]>([]);
const declarations = ref<VariableDeclaration[]>([]);
const allStages = ref<PipelineStage[]>([]);
const stageCache = reactive<Record<string, PipelineStage>>({});
const viewMode = ref<'list' | 'dag'>('list');
const repositories = ref<Repository[]>([]);

// 已保存的快照，用于 dirty 检测
const savedOrch = ref<string>('[]');
const savedDeclarations = ref<string>('[]');

const isDirty = computed(() => {
	const orchStr = JSON.stringify(sortableOrch.value.map((o, i) => ({ ...o, sort_order: i })));
	const declStr = JSON.stringify(declarations.value);
	return orchStr !== savedOrch.value || declStr !== savedDeclarations.value;
});

const hasStageUpdates = computed(() =>
	sortableOrch.value.some((o) => {
		const stage = stageCache[o.stage_id];
		return stage && stage.version > o.stage_version;
	})
);

const editInfoModalRef = ref<HTMLDialogElement>();
const addOrchModalRef = ref<HTMLDialogElement>();
const editOrchModalRef = ref<HTMLDialogElement>();
const runModalRef = ref<HTMLDialogElement>();
const addVarModalRef = ref<HTMLDialogElement>();
const editVarModalRef = ref<HTMLDialogElement>();
const deleteModalRef = ref<HTMLDialogElement>();
const editForm = reactive({ name: '', description: '' });
const addOrchForm = reactive({
	stageId: '',
	dependsOn: [] as string[],
});
const runForm = reactive({
	repositoryId: '',
	triggerRef: '',
});
const editOrchForm = reactive({
	editingStageId: '',
	dependsOn: [] as string[],
});
const varForm = reactive({
	name: '',
	value: '',
	description: '',
});

const editableOrchOptions = computed(() =>
	sortableOrch.value.filter((o) => o.stage_id !== editOrchForm.editingStageId)
);

const availableStages = computed(() => {
	const inOrch = new Set(sortableOrch.value.map((o) => o.stage_id));
	return allStages.value.filter((s) => !inOrch.has(s.id));
});

const selectedRepository = computed(() =>
	repositories.value.find((r) => r.id === runForm.repositoryId)
);

const dagStages = computed(() =>
	sortableOrch.value.map((orch) => {
		const stage = stageCache[orch.stage_id];
		return {
			id: orch.stage_id,
			name: stage?.name ?? orch.stage_id,
			image: stage?.image ?? '',
			script: stage?.script ?? '',
			env: stage?.env ?? {},
			version: stage?.version ?? 1,
			depends_on: orch.depends_on,
		};
	})
);

// 监听项目选择，自动填充默认分支
watch(
	() => runForm.repositoryId,
	(newRepoId) => {
		if (newRepoId) {
			const repo = repositories.value.find((r) => r.id === newRepoId);
			if (repo?.default_branch) {
				runForm.triggerRef = repo.default_branch;
			}
		}
	}
);

function applyTemplateState(tmpl: PipelineTemplate) {
	template.value = tmpl;
	sortableOrch.value = [...tmpl.orchestration].sort((a, b) => a.sort_order - b.sort_order);
	declarations.value = [...tmpl.variable_declarations];
	// 更新已保存快照
	savedOrch.value = JSON.stringify(sortableOrch.value.map((o, i) => ({ ...o, sort_order: i })));
	savedDeclarations.value = JSON.stringify(declarations.value);
	Object.keys(stageCache).forEach((key) => delete stageCache[key]);
	tmpl.stages.forEach((stage) => (stageCache[stage.id] = stage));
	Object.assign(editForm, {
		name: tmpl.name,
		description: tmpl.description ?? '',
	});
}

async function fetchTemplate() {
	try {
		await execute(async () => {
			const tmpl = await pipelineTemplateApi.get(templateId.value);
			allStages.value = [];
			applyTemplateState(tmpl);
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
			const response = await pipelineStageApi.list({ page: 1, per_page: 100 });
			allStages.value = response.items;
		} catch {
			toast.error('获取 Stage 列表失败');
		}
	}
}

async function syncDeclarations() {
	try {
		declarations.value = await pipelineTemplateApi.resolveVariables({
			orchestration: sortableOrch.value.map((item, index) => ({
				...item,
				sort_order: index,
			})),
			variable_declarations: declarations.value,
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '同步变量失败');
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

	try {
		await executeSave(async () => {
			// 只保存基本信息，不传递 orchestration 和 variable_declarations
			const data = await pipelineTemplateApi.update(templateId.value, {
				name: editForm.name,
				description: editForm.description,
			});
			applyTemplateState(data);
			toast.success('保存成功');
		});
		editInfoModalRef.value?.close();
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '保存失败');
	}
}

async function handleSave() {
	const orchForCheck = sortableOrch.value.map((o) => ({
		name: o.stage_id,
		depends_on: o.depends_on,
	}));
	const cycle = detectCircularDependencies(orchForCheck);
	if (cycle) {
		toast.error(`检测到循环依赖: ${cycle.join(' → ')}`);
		return;
	}

	try {
		await executeSave(async () => {
			// 自动更新编排中的 stage_version
			for (const orch of sortableOrch.value) {
				const stage = stageCache[orch.stage_id];
				if (stage && stage.version > orch.stage_version) {
					orch.stage_version = stage.version;
				}
			}

			const data = await pipelineTemplateApi.update(templateId.value, {
				name: editForm.name,
				description: editForm.description,
				orchestration: sortableOrch.value.map((o, i) => ({
					...o,
					sort_order: i,
				})),
				variable_declarations: declarations.value,
			});
			applyTemplateState(data);
			toast.success(`保存成功，快照 v${data.version}`);
		});
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '保存失败');
	}
}

async function handleDuplicate() {
	try {
		await executeDuplicate(async () => {
			const newTemplate = await pipelineTemplateApi.duplicate(templateId.value);
			toast.success('复制成功');
			router.push(`/ci/template/${newTemplate.id}`);
		});
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '复制失败');
	}
}

function openDeleteModal() {
	deleteModalRef.value?.showModal();
}

async function handleDeleteOk() {
	try {
		await executeDelete(async () => {
			await pipelineTemplateApi.delete(templateId.value);
			toast.success('删除成功');
			router.push('/ci/template');
		});
	} catch (e) {
		toast.error(e instanceof Error ? e.message : '删除失败');
	}
}

// ── 编排操作 ──────────────────────────────────────────────────────────────────

function onDragEnd() {
	// VueDraggable 已直接更新 sortableOrch，sort_order 在保存时统一写入
}

async function openAddOrchModal() {
	addOrchForm.stageId = '';
	addOrchForm.dependsOn = [];
	await fetchStages();
	addOrchModalRef.value?.showModal();
}

async function confirmAddOrch() {
	if (!addOrchForm.stageId) {
		return;
	}
	const stage = allStages.value.find((s) => s.id === addOrchForm.stageId);
	if (!stage) {
		return;
	}
	const newOrch: StageOrchestration = {
		stage_id: addOrchForm.stageId,
		stage_name: stage.name,
		stage_version: stage.version,
		depends_on: addOrchForm.dependsOn,
		sort_order: sortableOrch.value.length,
	};
	const orchForCheck = [...sortableOrch.value, newOrch].map((o) => ({
		name: o.stage_id,
		depends_on: o.depends_on,
	}));
	const cycle = detectCircularDependencies(orchForCheck);
	if (cycle) {
		toast.error(`检测到循环依赖: ${cycle.join(' → ')}`);
		return;
	}
	sortableOrch.value.push(newOrch);
	stageCache[stage.id] = stage;
	await syncDeclarations();
	addOrchModalRef.value?.close();
}

async function removeOrch(idx: number) {
	const removed = sortableOrch.value[idx];
	sortableOrch.value.splice(idx, 1);
	for (const o of sortableOrch.value) {
		o.depends_on = o.depends_on.filter((depId) => depId !== removed.stage_id);
	}
	await syncDeclarations();
}

function openEditOrchModal(idx: number) {
	const orch = sortableOrch.value[idx];
	if (!orch) {
		return;
	}
	editOrchForm.editingStageId = orch.stage_id;
	editOrchForm.dependsOn = [...orch.depends_on];
	editOrchModalRef.value?.showModal();
}

function confirmEditOrch() {
	const orch = sortableOrch.value.find((o) => o.stage_id === editOrchForm.editingStageId);
	if (!orch) {
		return;
	}
	const orchForCheck = sortableOrch.value.map((o) => ({
		name: o.stage_id,
		depends_on: o.stage_id === editOrchForm.editingStageId ? editOrchForm.dependsOn : o.depends_on,
	}));
	const cycle = detectCircularDependencies(orchForCheck);
	if (cycle) {
		toast.error(`检测到循环依赖: ${cycle.join(' → ')}`);
		return;
	}
	orch.depends_on = editOrchForm.dependsOn;
	editOrchForm.editingStageId = '';
	editOrchForm.dependsOn = [];
	editOrchModalRef.value?.close();
}

// ── 运行流水线 ──────────────────────────────────────────────────────────────────

async function fetchRepositories() {
	try {
		const resp = await repositoryApi.list({ per_page: 100 });
		repositories.value = resp.items;
	} catch {
		toast.error('获取项目列表失败');
	}
}

async function openRunModal() {
	if (isDirty.value) {
		toast.error('有未保存的变更，请先保存后再运行');
		return;
	}
	runForm.repositoryId = '';
	runForm.triggerRef = '';
	await fetchRepositories();
	runModalRef.value?.showModal();
}

async function handleRunOk() {
	if (!runForm.repositoryId) {
		toast.error('请选择项目');
		return;
	}

	try {
		await executeRun(async () => {
			const repo = selectedRepository.value;
			const triggerRef = runForm.triggerRef || repo?.default_branch || 'main';
			const run = await repositoryApi.trigger(runForm.repositoryId, {
				template_id: templateId.value,
				trigger_ref: triggerRef,
				variables: {},
			});
			toast.success('触发成功');
			runModalRef.value?.close();
			router.push(`/ci/run/${run.id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '触发失败');
	}
}

// ── 变量管理 ──────────────────────────────────────────────────────────────────

function openAddVarModal() {
	varForm.name = '';
	varForm.value = '';
	varForm.description = '';
	addVarModalRef.value?.showModal();
}

function handleAddVarOk() {
	if (!varForm.name.trim()) {
		toast.error('变量名不能为空');
		return;
	}

	// 检查是否已存在
	if (declarations.value.some((d) => d.name === varForm.name)) {
		toast.error('变量名已存在');
		return;
	}

	declarations.value.push({
		name: varForm.name,
		value: varForm.value || undefined,
		description: varForm.description || undefined,
		source: 'template_custom',
		secret: false,
	});

	addVarModalRef.value?.close();
}

function openEditVarModal(name: string) {
	const decl = declarations.value.find((d) => d.name === name);
	if (!decl) {
		return;
	}
	varForm.name = decl.name;
	varForm.value = String(decl.value ?? '');
	varForm.description = decl.description ?? '';
	editVarModalRef.value?.showModal();
}

function handleEditVarOk() {
	const decl = declarations.value.find((d) => d.name === varForm.name);
	if (decl) {
		decl.value = varForm.value || undefined;
		decl.description = varForm.description || undefined;
	}
	editVarModalRef.value?.close();
}

function deleteVariable(name: string) {
	const idx = declarations.value.findIndex((d) => d.name === name);
	if (idx !== -1) {
		declarations.value.splice(idx, 1);
	}
}

watch(templateId, fetchTemplate);
onMounted(fetchTemplate);
</script>
