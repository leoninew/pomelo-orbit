<template>
	<div class="flex flex-col gap-4">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h1 class="text-xl font-semibold text-foreground">{{ template?.name || '模板详情' }}</h1>
			<div class="flex flex-wrap items-center gap-2">
				<button
					v-if="template && (isDirty || hasStageUpdates)"
					:disabled="saving"
					class="app-button-primary h-9 px-3"
					@click="handleSave"
				>
					{{ saving ? '保存中...' : hasStageUpdates && !isDirty ? '更新' : '保存' }}
				</button>
				<button v-if="template" class="app-button h-9 px-3" @click="openRunModal">运行</button>
				<button v-if="template" class="app-button h-9 px-3" @click="openEditInfoModal">编辑</button>
				<button
					v-if="template"
					:disabled="duplicating"
					class="app-button h-9 px-3"
					@click="handleDuplicate"
				>
					{{ duplicating ? '复制中...' : '复制' }}
				</button>
				<button v-if="template" class="app-button-danger h-9 px-3" @click="openDeleteModal">
					删除
				</button>
				<button class="app-button h-9 px-4" @click="router.push('/ci/template')">返回</button>
			</div>
		</div>

		<!-- 加载状态 -->
		<div v-if="status === 'loading'" class="flex items-center justify-center py-12">
			<div
				class="h-8 w-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary"
			></div>
		</div>

		<!-- 内容 -->
		<div v-else-if="template" class="flex flex-col gap-4">
			<!-- 基本信息卡片 -->
			<div class="app-surface">
				<div class="app-section-header">
					<h2 class="font-semibold text-foreground">基本信息</h2>
				</div>
				<dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">模板名称</dt>
						<dd class="text-foreground">{{ template.name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">版本</dt>
						<dd class="text-foreground">v{{ template.version }}</dd>
					</div>
					<div class="flex gap-2 sm:col-span-2">
						<dt class="w-24 shrink-0 text-muted-foreground">描述</dt>
						<dd class="text-foreground">{{ template.description || '无' }}</dd>
					</div>
				</dl>
			</div>

			<!-- Stage 编排 -->
			<div class="app-surface">
				<div class="app-section-header flex items-center justify-between">
					<div class="flex items-center gap-4">
						<h2 class="font-semibold text-foreground">阶段编排</h2>
						<div
							v-if="sortableOrch.length > 0"
							class="flex gap-1 rounded-md border border-border bg-background p-1"
						>
							<button
								class="rounded px-3 py-1 text-xs font-medium transition-colors"
								:class="
									viewMode === 'list'
										? 'bg-primary text-primary-foreground'
										: 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'
								"
								@click="viewMode = 'list'"
							>
								列表
							</button>
							<button
								class="rounded px-3 py-1 text-xs font-medium transition-colors"
								:class="
									viewMode === 'dag'
										? 'bg-primary text-primary-foreground'
										: 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'
								"
								@click="viewMode = 'dag'"
							>
								DAG
							</button>
						</div>
					</div>
					<button class="app-button-primary h-9 px-3" @click="openAddOrchModal">
						<Plus class="h-4 w-4" />
						添加阶段
					</button>
				</div>
				<!-- 列表视图 -->
				<div v-if="viewMode === 'list'" class="overflow-x-auto">
					<table class="app-table-detail min-w-[760px]">
						<thead>
							<tr>
								<th>#</th>
								<th>阶段</th>
								<th>版本</th>
								<th>依赖</th>
								<th>制品</th>
								<th>操作</th>
							</tr>
						</thead>
						<tbody>
							<tr v-if="sortableOrch.length === 0">
								<td colspan="6" class="text-center text-muted-foreground">
									暂无 Stage，点击上方按钮添加
								</td>
							</tr>
							<tr v-for="(orch, idx) in sortableOrch" :key="orch.stage_id">
								<td class="text-muted-foreground">{{ idx + 1 }}</td>
								<td>
									<div class="flex items-center gap-2">
										<router-link :to="`/ci/build-stage/${orch.stage_id}`" class="app-link">
											{{ stageCache[orch.stage_id]?.name ?? orch.stage_name }}
										</router-link>
										<span
											v-if="
												stageCache[orch.stage_id] &&
													stageCache[orch.stage_id].version > orch.stage_version
											"
											class="inline-block rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700"
										>
											有更新
										</span>
									</div>
								</td>
								<td class="text-foreground">v{{ orch.stage_version }}</td>
								<td>
									<div v-if="orch.depends_on.length > 0" class="flex flex-wrap gap-1">
										<span
											v-for="depId in orch.depends_on"
											:key="depId"
											class="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground"
										>
											{{ stageCache[depId]?.name ?? depId }}
										</span>
									</div>
									<span v-else class="text-muted-foreground">—</span>
								</td>
								<td class="text-foreground">
									{{ stageCache[orch.stage_id]?.artifacts?.length ?? '—' }}
								</td>
								<td>
									<div class="flex items-center gap-3">
										<button class="app-link" @click="openEditOrchModal(idx)">编辑</button>
										<button class="app-link-danger" @click="confirmRemoveOrch(idx)">移除</button>
									</div>
								</td>
							</tr>
						</tbody>
					</table>
				</div>

				<!-- DAG 视图 -->
				<div v-else-if="dagStages.length > 0" class="p-6">
					<div class="h-[500px]">
						<StageDAGView :stages="dagStages" />
					</div>
				</div>
			</div>

			<!-- 变量声明 -->
			<div class="app-surface">
				<div class="app-section-header flex items-center justify-between">
					<h2 class="font-semibold text-foreground">变量声明</h2>
					<button class="app-button-primary h-9 px-3" @click="openAddVarModal">
						<Plus class="h-4 w-4" />
						添加变量
					</button>
				</div>
				<VariableDeclarationsTable
					:declarations="declarations"
					:readonly="false"
					@edit="openEditVarModal"
					@delete="confirmDeleteVariable"
				/>
			</div>

			<!-- 制品声明 -->
			<div class="app-surface">
				<div class="app-section-header">
					<h2 class="font-semibold text-foreground">制品声明</h2>
				</div>
				<div
					v-if="artifactDeclarations.length === 0"
					class="px-5 py-12 text-center text-sm text-muted-foreground"
				>
					暂无制品
				</div>
				<div v-else class="overflow-x-auto">
					<table class="app-table-detail min-w-[720px]">
						<thead>
							<tr>
								<th>Stage</th>
								<th>类型</th>
								<th>名称</th>
								<th>路径/镜像</th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="(artifact, idx) in artifactDeclarations" :key="idx">
								<td class="text-foreground">{{ artifact.stageName }}</td>
								<td>
									<span
										class="inline-block rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground"
									>
										{{ artifact.type }}
									</span>
								</td>
								<td class="text-foreground">{{ artifact.name }}</td>
								<td class="text-muted-foreground">{{ artifact.path || '—' }}</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</div>

		<AppDialog
			v-model:open="isEditInfoDialogOpen"
			title="编辑基本信息"
			description="更新流水线模板名称和描述。"
		>
			<div class="space-y-1.5">
				<label class="app-field-label block">模板名称</label>
				<input v-model="editForm.name" type="text" class="app-input" />
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">描述</label>
				<textarea v-model="editForm.description" rows="3" class="app-textarea"></textarea>
			</div>
			<template #footer>
				<button class="app-button" @click="cancelEditInfo">取消</button>
				<button :disabled="saving" class="app-button-primary" @click="handleEditInfoOk">
					{{ saving ? '保存中...' : '保存' }}
				</button>
			</template>
		</AppDialog>

		<AppDialog v-model:open="isAddOrchDialogOpen" title="添加 Stage">
			<div class="space-y-1.5">
				<label class="app-field-label block">选择 Stage</label>
				<ComboboxSelect
					:model-value="addOrchForm.stageId"
					:options="stageSelectOptions"
					:portal="false"
					:open-on-focus="false"
					placeholder="搜索 Stage..."
					empty-text="未找到 Stage"
					@update:model-value="handleStageSelection"
				/>
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">依赖 Stage（可选）</label>
				<div class="space-y-2">
					<label v-for="orch in sortableOrch" :key="orch.stage_id" class="flex items-center gap-2">
						<input
							v-model="addOrchForm.dependsOn"
							:value="orch.stage_id"
							type="checkbox"
							class="app-checkbox"
						/>
						<span class="text-sm text-foreground">{{ orch.stage_name }}</span>
					</label>
				</div>
			</div>
			<template #footer>
				<button class="app-button" @click="isAddOrchDialogOpen = false">取消</button>
				<button :disabled="!addOrchForm.stageId" class="app-button-primary" @click="confirmAddOrch">
					添加
				</button>
			</template>
		</AppDialog>

		<AppDialog v-model:open="isEditOrchDialogOpen" title="编辑依赖">
			<div class="space-y-1.5">
				<label class="app-field-label block">依赖 Stage</label>
				<div class="space-y-2">
					<label
						v-for="orch in editableOrchOptions"
						:key="orch.stage_id"
						class="flex items-center gap-2"
					>
						<input
							v-model="editOrchForm.dependsOn"
							:value="orch.stage_id"
							type="checkbox"
							class="app-checkbox"
						/>
						<span class="text-sm text-foreground">{{ orch.stage_name }}</span>
					</label>
				</div>
			</div>
			<template #footer>
				<button class="app-button" @click="isEditOrchDialogOpen = false">取消</button>
				<button class="app-button-primary" @click="confirmEditOrch">保存</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isRunDialogOpen"
			title="运行流水线"
			description="选择仓库并指定触发分支。"
		>
			<div class="space-y-1.5">
				<label class="app-field-label block">选择项目</label>
				<ComboboxSelect
					:model-value="runForm.repositoryId"
					:options="repoSelectOptions"
					:portal="false"
					:open-on-focus="false"
					placeholder="搜索项目..."
					empty-text="未找到项目"
					@update:model-value="handleRepoSelection"
				/>
				<p
					v-if="selectedRepository && !selectedRepository.git_credential_id"
					class="mt-2 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive"
				>
					该项目未配置 Git 凭据，请先在仓库详情配置后再运行。
				</p>
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">触发分支</label>
				<input
					v-model="runForm.triggerRef"
					type="text"
					:placeholder="selectedRepository?.default_branch || 'main'"
					class="app-input"
				/>
			</div>
			<template #footer>
				<button class="app-button" @click="isRunDialogOpen = false">取消</button>
				<button
					:disabled="!runForm.repositoryId || !selectedRepository?.git_credential_id || running"
					class="app-button-primary"
					@click="handleRunOk"
				>
					{{ running ? '运行中...' : '运行' }}
				</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isAddVarDialogOpen"
			title="添加变量"
			description="添加模板自定义变量。"
		>
			<div class="space-y-1.5">
				<label class="app-field-label block">变量名</label>
				<input v-model="varForm.name" type="text" class="app-input" />
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">变量值</label>
				<input v-model="varForm.value" type="text" class="app-input" />
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">说明</label>
				<input v-model="varForm.description" type="text" class="app-input" />
			</div>
			<template #footer>
				<button class="app-button" @click="isAddVarDialogOpen = false">取消</button>
				<button class="app-button-primary" @click="handleAddVarOk">添加</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isEditVarDialogOpen"
			title="编辑变量"
			description="编辑模板自定义变量。"
		>
			<div class="space-y-1.5">
				<label class="app-field-label block">变量名</label>
				<input v-model="varForm.name" type="text" disabled class="app-input" />
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">变量值</label>
				<input v-model="varForm.value" type="text" class="app-input" />
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">说明</label>
				<input v-model="varForm.description" type="text" class="app-input" />
			</div>
			<template #footer>
				<button class="app-button" @click="isEditVarDialogOpen = false">取消</button>
				<button class="app-button-primary" @click="handleEditVarOk">保存</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isDeleteDialogOpen"
			title="确认删除"
			description="确定要删除此模板吗？此操作不可恢复。"
			width-class="w-[min(420px,calc(100vw-32px))]"
			body-class="hidden"
		>
			<template #footer>
				<button class="app-button" @click="isDeleteDialogOpen = false">取消</button>
				<button :disabled="deleting" class="app-button-destructive" @click="handleDeleteOk">
					{{ deleting ? '删除中...' : '确认删除' }}
				</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isDeleteOrchDialogOpen"
			title="确认移除"
			description="确定要移除此 Stage 吗？"
			width-class="w-[min(420px,calc(100vw-32px))]"
			body-class="hidden"
		>
			<template #footer>
				<button class="app-button" @click="isDeleteOrchDialogOpen = false">取消</button>
				<button class="app-button-destructive" @click="removeOrch">确认移除</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isDeleteVarDialogOpen"
			title="确认删除"
			:description="'确定要删除变量 ' + varToDelete + ' 吗？'"
			width-class="w-[min(420px,calc(100vw-32px))]"
			body-class="hidden"
		>
			<template #footer>
				<button class="app-button" @click="isDeleteVarDialogOpen = false">取消</button>
				<button class="app-button-destructive" @click="deleteVariable">确认删除</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { Plus } from 'lucide-vue-next';
	import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
	// import { VueDraggable } from 'vue-draggable-plus';
	import { useRoute, useRouter } from 'vue-router';
	import { buildStageApi, pipelineTemplateApi, repositoryApi } from '@/api/ci';
	import AppDialog from '@/components/AppDialog.vue';
	import ComboboxSelect, { type ComboboxOptionValue } from '@/components/ComboboxSelect.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import type {
		ArtifactDeclaration,
		BuildStage,
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
	const stageCache = reactive<Record<string, BuildStage>>({});
	const viewMode = ref<'list' | 'dag'>('list');

	const stageOptions = ref<BuildStage[]>([]);

	const stageSelectOptions = computed(() => {
		const inOrch = new Set(sortableOrch.value.map((o) => o.stage_id));
		return stageOptions.value
			.filter((stage) => !inOrch.has(stage.id))
			.map((stage) => ({
				value: stage.id,
				label: stage.name,
				description: `v${stage.version} · ${stage.image}`,
			}));
	});

	const repoOptions = ref<Repository[]>([]);
	const repoSelectOptions = computed(() =>
		repoOptions.value.map((repo) => ({
			value: repo.id,
			label: repo.name,
			description: repo.git_credential_id ? repo.repository_url : '未配置 Git 凭据',
		}))
	);

	// 已保存的快照，用于 dirty 检测
	const savedOrch = ref<string>('[]');
	const savedDeclarations = ref<string>('[]');

	const isDirty = computed(() => {
		const orchStr = JSON.stringify(sortableOrch.value.map((o, i) => ({ ...o, sort_order: i })));
		const declStr = JSON.stringify(declarations.value);
		return orchStr !== savedOrch.value || declStr !== savedDeclarations.value;
	});

	const hasStageUpdates = computed(() =>
		sortableOrch.value.some((orch) => {
			const stage = stageCache[orch.stage_id];
			return stage && stage.version > orch.stage_version;
		})
	);

	const artifactDeclarations = computed(() => {
		const result: ArtifactDeclaration[] = [];
		for (const orch of sortableOrch.value) {
			const stage = stageCache[orch.stage_id];
			for (const artifact of stage?.artifacts ?? []) {
				result.push({
					stageName: stage?.name ?? orch.stage_name,
					type: artifact.type,
					name: artifact.name,
					path: artifact.path,
				});
			}
		}
		return result;
	});

	const isEditInfoDialogOpen = ref(false);
	const isAddOrchDialogOpen = ref(false);
	const isEditOrchDialogOpen = ref(false);
	const isRunDialogOpen = ref(false);
	const isAddVarDialogOpen = ref(false);
	const isEditVarDialogOpen = ref(false);
	const isDeleteDialogOpen = ref(false);
	const isDeleteOrchDialogOpen = ref(false);
	const isDeleteVarDialogOpen = ref(false);

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
	const orchToDelete = ref(-1);
	const varToDelete = ref('');

	const editableOrchOptions = computed(() =>
		sortableOrch.value.filter((o) => o.stage_id !== editOrchForm.editingStageId)
	);

	const selectedRepository = computed(() =>
		repoOptions.value.find((r) => r.id === runForm.repositoryId)
	);

	const dagStages = computed(() =>
		sortableOrch.value.map((orch) => {
			const stage = stageCache[orch.stage_id];
			return {
				id: orch.stage_id,
				name: stage?.name ?? orch.stage_id,
				image: stage?.image ?? '',
				script: stage?.script ?? '',
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
				const repo = repoOptions.value.find((r) => r.id === newRepoId);
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
				applyTemplateState(tmpl);
			});
		} catch {
			toast.error('获取模板信息失败');
			router.push('/ci/template');
		}
	}

	async function searchStages() {
		try {
			const resp = await buildStageApi.list({
				per_page: 100,
			});
			stageOptions.value = resp.items;
		} catch (err: unknown) {
			toast.error(err instanceof Error ? err.message : '获取 Stage 列表失败');
		}
	}

	function handleStageSelection(value: ComboboxOptionValue) {
		addOrchForm.stageId = String(value || '');
		const stage = stageOptions.value.find((item) => item.id === addOrchForm.stageId);
		if (stage) {
			stageCache[stage.id] = stage;
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
		isEditInfoDialogOpen.value = true;
	}

	function cancelEditInfo() {
		Object.assign(editForm, {
			name: template.value?.name ?? '',
			description: template.value?.description ?? '',
		});
		isEditInfoDialogOpen.value = false;
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
			isEditInfoDialogOpen.value = false;
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
		isDeleteDialogOpen.value = true;
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

	async function openAddOrchModal() {
		addOrchForm.stageId = '';
		addOrchForm.dependsOn = [];
		isAddOrchDialogOpen.value = true;
		await searchStages();
	}

	async function confirmAddOrch() {
		if (!addOrchForm.stageId) {
			return;
		}
		const stage =
			stageOptions.value.find((s) => s.id === addOrchForm.stageId) ??
			stageCache[addOrchForm.stageId];
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
		isAddOrchDialogOpen.value = false;
	}

	function confirmRemoveOrch(idx: number) {
		orchToDelete.value = idx;
		isDeleteOrchDialogOpen.value = true;
	}

	async function removeOrch() {
		const idx = orchToDelete.value;
		if (idx === -1) {
			return;
		}
		const removed = sortableOrch.value[idx];
		sortableOrch.value.splice(idx, 1);
		for (const o of sortableOrch.value) {
			o.depends_on = o.depends_on.filter((depId) => depId !== removed.stage_id);
		}
		await syncDeclarations();
		isDeleteOrchDialogOpen.value = false;
		orchToDelete.value = -1;
	}

	function openEditOrchModal(idx: number) {
		const orch = sortableOrch.value[idx];
		if (!orch) {
			return;
		}
		editOrchForm.editingStageId = orch.stage_id;
		editOrchForm.dependsOn = [...orch.depends_on];
		isEditOrchDialogOpen.value = true;
	}

	function confirmEditOrch() {
		const orch = sortableOrch.value.find((o) => o.stage_id === editOrchForm.editingStageId);
		if (!orch) {
			return;
		}
		const orchForCheck = sortableOrch.value.map((o) => ({
			name: o.stage_id,
			depends_on:
				o.stage_id === editOrchForm.editingStageId ? editOrchForm.dependsOn : o.depends_on,
		}));
		const cycle = detectCircularDependencies(orchForCheck);
		if (cycle) {
			toast.error(`检测到循环依赖: ${cycle.join(' → ')}`);
			return;
		}
		orch.depends_on = editOrchForm.dependsOn;
		editOrchForm.editingStageId = '';
		editOrchForm.dependsOn = [];
		isEditOrchDialogOpen.value = false;
	}

	// ── 运行流水线 ──────────────────────────────────────────────────────────────────

	async function searchRepos() {
		try {
			const resp = await repositoryApi.list({
				per_page: 100,
			});
			repoOptions.value = resp.items;
		} catch (err: unknown) {
			toast.error(err instanceof Error ? err.message : '获取项目列表失败');
		}
	}

	function handleRepoSelection(value: ComboboxOptionValue) {
		runForm.repositoryId = String(value || '');
	}

	async function openRunModal() {
		if (isDirty.value) {
			toast.error('有未保存的变更，请先保存后再运行');
			return;
		}
		runForm.repositoryId = '';
		runForm.triggerRef = '';
		await searchRepos();
		isRunDialogOpen.value = true;
	}

	async function handleRunOk() {
		if (!runForm.repositoryId) {
			toast.error('请选择项目');
			return;
		}

		const repo = selectedRepository.value;
		if (!repo?.git_credential_id) {
			toast.error('该项目未配置 Git 凭据');
			return;
		}

		try {
			await executeRun(async () => {
				const triggerRef = runForm.triggerRef || repo.default_branch || 'main';
				const run = await repositoryApi.trigger(runForm.repositoryId, {
					template_id: templateId.value,
					trigger_ref: triggerRef,
					variables: {},
				});
				toast.success('触发成功');
				isRunDialogOpen.value = false;
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
		isAddVarDialogOpen.value = true;
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

		isAddVarDialogOpen.value = false;
	}

	function openEditVarModal(name: string) {
		const decl = declarations.value.find((d) => d.name === name);
		if (!decl) {
			return;
		}
		varForm.name = decl.name;
		varForm.value = String(decl.value ?? '');
		varForm.description = decl.description ?? '';
		isEditVarDialogOpen.value = true;
	}

	function handleEditVarOk() {
		const decl = declarations.value.find((d) => d.name === varForm.name);
		if (decl) {
			decl.value = varForm.value || undefined;
			decl.description = varForm.description || undefined;
		}
		isEditVarDialogOpen.value = false;
	}

	function confirmDeleteVariable(name: string) {
		varToDelete.value = name;
		isDeleteVarDialogOpen.value = true;
	}

	function deleteVariable() {
		const name = varToDelete.value;
		if (!name) {
			return;
		}
		const idx = declarations.value.findIndex((d) => d.name === name);
		if (idx !== -1) {
			declarations.value.splice(idx, 1);
		}
		isDeleteVarDialogOpen.value = false;
		varToDelete.value = '';
	}

	watch(templateId, fetchTemplate);
	onMounted(fetchTemplate);
	onUnmounted(() => {
		// Cleanup if needed
	});
</script>
