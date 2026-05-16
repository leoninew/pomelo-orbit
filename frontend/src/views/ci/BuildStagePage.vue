<template>
	<div class="space-y-6">
		<ToolbarRoot class="app-toolbar-simple" aria-label="构建阶段工具栏">
			<SearchControl
				v-model="searchText"
				placeholder="搜索名称/描述"
				:loading="status === 'loading'"
				@search="handleSearch"
			/>
			<button class="app-button-primary px-5" @click="openCreateModal">
				<Plus class="size-4" />
				新建构建
			</button>
		</ToolbarRoot>

		<div class="app-surface">
			<AppSpinner v-if="status === 'loading'" class="py-16" />
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error }}</p>
			</div>
			<div v-else-if="stages.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[920px]">
					<thead>
						<tr>
							<th>名称</th>
							<th>镜像</th>
							<th>版本</th>
							<th>描述</th>
							<th>创建时间</th>
							<th>操作</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="stage in stages" :key="stage.id">
							<td>
								<router-link :to="`/ci/build-stage/${stage.id}`" class="app-link whitespace-nowrap">
									{{ stage.name }}
								</router-link>
							</td>
							<td class="max-w-64 truncate text-foreground" :title="stage.image">
								{{ stage.image }}
							</td>
							<td>
								<AppBadge>v{{ stage.version }}</AppBadge>
							</td>
							<td
								class="max-w-xs truncate text-muted-foreground"
								:title="stage.description || undefined"
							>
								{{ stage.description || '—' }}
							</td>
							<td class="whitespace-nowrap text-muted-foreground">
								{{ formatTime(stage.created_at) }}
							</td>
							<td class="whitespace-nowrap">
								<router-link :to="`/ci/build-stage/${stage.id}`" class="app-link">查看</router-link>
								<button
									class="app-link ml-3"
									:disabled="duplicating"
									@click="handleDuplicate(stage.id)"
								>
									复制
								</button>
							</td>
						</tr>
					</tbody>
				</table>
			</div>

			<ListPagination
				:current="pagination.current"
				:page-size="pagination.pageSize"
				:total="pagination.total"
				:total-pages="totalPages"
				@change-page="goPage"
				@change-page-size="handlePageSizeChange"
			/>
		</div>
	</div>

	<AppDialog
		v-model:open="isModalOpen"
		title="新建构建"
		width-class="w-[min(600px,calc(100vw-32px))]"
	>
		<div class="space-y-4">
			<div class="space-y-1.5">
				<label class="app-field-label block">名称</label>
				<input
					v-model="form.name"
					type="text"
					class="app-input"
					:class="errors.name ? 'app-input-error' : ''"
					placeholder="例如: build"
				/>
				<p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">镜像</label>
				<input
					v-model="form.image"
					type="text"
					class="app-input"
					:class="errors.image ? 'app-input-error' : ''"
					placeholder="例如: alpine:latest"
				/>
				<p v-if="errors.image" class="app-field-error text-xs">{{ errors.image }}</p>
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">描述（可选）</label>
				<input v-model="form.description" type="text" class="app-input" placeholder="简短描述" />
			</div>
		</div>

		<template #footer>
			<button class="app-button" @click="isModalOpen = false">取消</button>
			<button class="app-button-primary" :disabled="operating" @click="handleModalOk">创建</button>
		</template>
	</AppDialog>
</template>
<script setup lang="ts">
	import { Plus } from 'lucide-vue-next';
	import { computed, onMounted, reactive, ref } from 'vue';
	import { useRouter } from 'vue-router';
	import { ToolbarRoot } from 'reka-ui';
	import { buildStageApi } from '@/api/ci';
	import AppDialog from '@/components/AppDialog.vue';
	import AppBadge from '@/components/AppBadge.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import ListPagination from '@/components/ListPagination.vue';
	import SearchControl from '@/components/SearchControl.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import { useProjectStore } from '@/stores/project';
	import type { BuildStage } from '@/types/ci/template';
	import { formatTime } from '@/utils/time';

	const router = useRouter();
	const toast = useToast();
	const projectStore = useProjectStore();
	const { status, error, execute } = useStatusAsync();
	const { loading: operating, execute: executeOp } = useStatusAsync();
	const { loading: duplicating, execute: executeDuplicate } = useStatusAsync();

	const stages = ref<BuildStage[]>([]);
	const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
	const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
	const searchText = ref('');
	const isModalOpen = ref(false);

	const form = reactive({ name: '', image: '', description: '' });
	const errors = reactive({ name: '', image: '' });

	function validate() {
		errors.name = form.name.trim() ? '' : '请输入名称';
		errors.image = form.image.trim() ? '' : '请输入镜像';
		return !errors.name && !errors.image;
	}

	async function fetchStages() {
		const projectId = projectStore.activeProjectId;
		if (!projectId) {
			toast.error('请先选择项目');
			return;
		}
		try {
			await execute(async () => {
				const res = await buildStageApi.list({
					page: pagination.current,
					per_page: pagination.pageSize,
					search: searchText.value || undefined,
					project_id: projectId,
				});
				stages.value = res.items;
				pagination.total = res.total;
			});
		} catch {
			toast.error('获取 Stage 列表失败');
		}
	}

	function handleSearch() {
		if (status.value === 'loading') {
			return;
		}
		pagination.current = 1;
		fetchStages();
	}

	function goPage(p: number) {
		pagination.current = p;
		fetchStages();
	}

	function handlePageSizeChange(pageSize: number) {
		pagination.pageSize = pageSize;
		pagination.current = 1;
		fetchStages();
	}

	function openCreateModal() {
		Object.assign(form, { name: '', image: '', description: '' });
		Object.assign(errors, { name: '', image: '' });
		isModalOpen.value = true;
	}

	async function handleModalOk() {
		if (!validate()) {
			return;
		}
		const projectId = projectStore.activeProjectId;
		if (!projectId) {
			toast.error('请先选择项目');
			return;
		}
		try {
			await executeOp(async () => {
				await buildStageApi.create(
					{
						name: form.name,
						image: form.image,
						script: '',
						description: form.description,
					},
					{ project_id: projectId }
				);
				toast.success('创建成功');
				isModalOpen.value = false;
				fetchStages();
			});
		} catch (err) {
			toast.error(err instanceof Error ? err.message : '操作失败');
		}
	}

	async function handleDuplicate(id: string) {
		const projectId = projectStore.activeProjectId;
		if (!projectId) {
			toast.error('请先选择项目');
			return;
		}
		try {
			await executeDuplicate(async () => {
				const newStage = await buildStageApi.duplicate(id);
				toast.success('复制成功');
				router.push(`/ci/build-stage/${newStage.id}`);
			});
		} catch (err) {
			toast.error(err instanceof Error ? err.message : '复制失败');
		}
	}

	onMounted(fetchStages);
</script>
