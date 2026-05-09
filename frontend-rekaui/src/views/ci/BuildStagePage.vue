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
				<table class="app-table-list min-w-[1040px]">
					<thead>
						<tr>
							<th>名称</th>
							<th>镜像</th>
							<th>版本</th>
							<th>制品</th>
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
								<span class="app-badge-sm">v{{ stage.version }}</span>
							</td>
							<td class="text-foreground">{{ stage.artifacts?.length ?? 0 }}</td>
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

	<AppDialog
		v-model:open="isModalOpen"
		title="新建构建"
		description="创建一个可被流水线模板复用的构建阶段。"
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
				<label class="app-field-label block">脚本</label>
				<textarea
					v-model="form.script"
					class="app-textarea min-h-40 font-mono"
					:class="errors.script ? 'app-input-error' : ''"
					placeholder="echo hello&#10;echo world"
				/>
				<p v-if="errors.script" class="app-field-error text-xs">{{ errors.script }}</p>
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
	import AppSpinner from '@/components/AppSpinner.vue';
	import ListPagination from '@/components/ListPagination.vue';
	import SearchControl from '@/components/SearchControl.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import type { BuildStage } from '@/types/ci/template';
	import { formatTime } from '@/utils/time';

	const router = useRouter();
	const toast = useToast();
	const { status, error, execute } = useStatusAsync();
	const { loading: operating, execute: executeOp } = useStatusAsync();
	const { loading: duplicating, execute: executeDuplicate } = useStatusAsync();

	const stages = ref<BuildStage[]>([]);
	const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
	const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
	const searchText = ref('');
	const isModalOpen = ref(false);

	const form = reactive({ name: '', image: '', script: '', description: '' });
	const errors = reactive({ name: '', image: '', script: '' });

	function validate() {
		errors.name = form.name.trim() ? '' : '请输入名称';
		errors.image = form.image.trim() ? '' : '请输入镜像';
		errors.script = form.script.trim() ? '' : '请输入脚本';
		return !errors.name && !errors.image && !errors.script;
	}

	async function fetchStages() {
		try {
			await execute(async () => {
				const res = await buildStageApi.list({
					page: pagination.current,
					per_page: pagination.pageSize,
					search: searchText.value || undefined,
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
		Object.assign(form, { name: '', image: '', script: '', description: '' });
		Object.assign(errors, { name: '', image: '', script: '' });
		isModalOpen.value = true;
	}

	async function handleModalOk() {
		if (!validate()) {
			return;
		}
		try {
			await executeOp(async () => {
				await buildStageApi.create({
					name: form.name,
					image: form.image,
					script: form.script,
					description: form.description,
				});
				toast.success('创建成功');
				isModalOpen.value = false;
				fetchStages();
			});
		} catch (err) {
			toast.error(err instanceof Error ? err.message : '操作失败');
		}
	}

	async function handleDuplicate(id: string) {
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
